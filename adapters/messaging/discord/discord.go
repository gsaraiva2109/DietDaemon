// Package discord implements ports.MessagingAdapter for the Discord Bot API.
// Send uses the REST API; Receive connects to the gateway WebSocket to stream
// MESSAGE_CREATE events in real time.
package discord

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/gsaraiva2109/dietdaemon/core/ports"
	"github.com/gsaraiva2109/dietdaemon/core/types"
	"golang.org/x/net/websocket"
)

// Compile-time interface check.
var _ ports.MessagingAdapter = (*Adapter)(nil)

// RESTBaseURL is the Discord REST API base.
const RESTBaseURL = "https://discord.com/api/v10"

// Adapter satisfies ports.MessagingAdapter for Discord.
type Adapter struct {
	token         string
	client        *http.Client
	dialWebSocket func(context.Context, string) (*websocket.Conn, error)
}

// New returns a ready Adapter. token is the Discord bot token.
func New(token string) *Adapter {
	return &Adapter{
		token:         token,
		client:        &http.Client{Timeout: 30 * time.Second},
		dialWebSocket: dialWebSocket,
	}
}

// Name returns "discord".
func (a *Adapter) Name() string { return "discord" }

// ---------------------------------------------------------------------------
// Discord component types for inline keyboards
// ---------------------------------------------------------------------------

// actionRow is an Action Row (type 1) containing button components.
type actionRow struct {
	Type       int               `json:"type"` // 1 = ActionRow
	Components []buttonComponent `json:"components"`
}

// buttonComponent is a Button (type 2) component within an action row.
type buttonComponent struct {
	Type     int    `json:"type"`  // 2 = Button
	Style    int    `json:"style"` // 1 = PRIMARY (blurple)
	Label    string `json:"label"`
	CustomID string `json:"custom_id"`
}

// ---------------------------------------------------------------------------
// Send — POST /channels/{channel_id}/messages
// ---------------------------------------------------------------------------

type sendMessageRequest struct {
	Content    string      `json:"content"`
	Components []actionRow `json:"components,omitempty"`
}

// Send delivers a reply to the channel identified by reply.ChannelMeta["channel_id"].
func (a *Adapter) Send(ctx context.Context, reply types.Reply) error {
	channelID := reply.ChannelMeta["channel_id"]
	if channelID == "" {
		return fmt.Errorf("discord: missing channel_id in ChannelMeta")
	}

	body := sendMessageRequest{Content: reply.Text}

	// Convert markup to Discord action rows.
	if reply.Markup != nil && len(reply.Markup.InlineKeyboard) > 0 {
		rows := make([]actionRow, len(reply.Markup.InlineKeyboard))
		for i, row := range reply.Markup.InlineKeyboard {
			buttons := make([]buttonComponent, len(row))
			for j, btn := range row {
				buttons[j] = buttonComponent{
					Type:     2, // Button
					Style:    1, // PRIMARY
					Label:    btn.Text,
					CustomID: btn.CallbackData,
				}
			}
			rows[i] = actionRow{Type: 1, Components: buttons}
		}
		body.Components = rows
		log.Printf("discord: sent action rows with %d row(s)", len(reply.Markup.InlineKeyboard))
	}

	payload, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("discord: marshal: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		RESTBaseURL+"/channels/"+channelID+"/messages",
		strings.NewReader(string(payload)),
	)
	if err != nil {
		return fmt.Errorf("discord: build request: %w", err)
	}
	req.Header.Set("Authorization", "Bot "+a.token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := a.client.Do(req)
	if err != nil {
		return fmt.Errorf("discord: send: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("discord: status %d: %s", resp.StatusCode, string(body))
	}
	return nil
}

// ---------------------------------------------------------------------------
// Receive — gateway WebSocket (opcode 0 DISPATCH, t MESSAGE_CREATE / INTERACTION_CREATE)
// ---------------------------------------------------------------------------

// gatewayPayload is the JSON structure for Discord gateway messages.
type gatewayPayload struct {
	Op int             `json:"op"`
	D  json.RawMessage `json:"d"`
	S  *int            `json:"s"`
	T  string          `json:"t"`
}

type helloData struct {
	HeartbeatInterval int `json:"heartbeat_interval"`
}

type identifyData struct {
	Token      string             `json:"token"`
	Properties identifyProperties `json:"properties"`
	Intents    int                `json:"intents"`
}

type identifyProperties struct {
	OS      string `json:"os"`
	Browser string `json:"browser"`
	Device  string `json:"device"`
}

type messageCreateData struct {
	ID        string `json:"id"`
	ChannelID string `json:"channel_id"`
	Author    struct {
		ID       string `json:"id"`
		Username string `json:"username"`
		Bot      bool   `json:"bot"`
	} `json:"author"`
	Content string `json:"content"`
}

// interactionCreateData is the payload for an INTERACTION_CREATE event,
// specifically for message component (button) interactions.
type interactionCreateData struct {
	ID        string `json:"id"`
	ChannelID string `json:"channel_id"`
	Type      int    `json:"type"`
	Data      struct {
		CustomID      string `json:"custom_id"`
		ComponentType int    `json:"component_type"`
	} `json:"data"`
	Member *struct {
		User struct {
			ID  string `json:"id"`
			Bot bool   `json:"bot"`
		} `json:"user"`
	} `json:"member,omitempty"`
	Token string `json:"token"`
}

const (
	gatewayOpDispatch           = 0
	gatewayOpHeartbeat          = 1
	gatewayOpIdentify           = 2
	gatewayIntentMessageContent = 1 << 15 // 32768
)

// Receive connects to the Discord gateway and streams MESSAGE_CREATE events
// as InboundMessage values. The channel closes when ctx is cancelled.
func (a *Adapter) Receive(ctx context.Context) (<-chan types.InboundMessage, error) {
	ch := make(chan types.InboundMessage)
	go a.gatewayLoop(ctx, ch)
	return ch, nil
}

func (a *Adapter) gatewayLoop(ctx context.Context, ch chan<- types.InboundMessage) {
	defer close(ch)

	conn, heartbeatInterval, err := a.connectGateway(ctx)
	if err != nil {
		return
	}
	defer func() { _ = conn.Close() }()

	// Start heartbeat goroutine.
	heartbeatCtx, cancelBeat := context.WithCancel(ctx)
	defer cancelBeat()
	go a.heartbeat(heartbeatCtx, conn, heartbeatInterval)

	if err := a.identifyGateway(conn); err != nil {
		return
	}

	// Read loop: filter DISPATCH MESSAGE_CREATE / INTERACTION_CREATE events.
	var lastSeq *int
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		pl, err := readGatewayPayload(conn)
		if err != nil {
			return
		}
		if pl.S != nil {
			lastSeq = pl.S
		}

		switch pl.Op {
		case gatewayOpDispatch:
			switch pl.T {
			case "MESSAGE_CREATE":
				if !a.handleMessageCreate(ctx, pl, ch) {
					return
				}
			case "INTERACTION_CREATE":
				if !a.handleInteractionCreate(ctx, pl, ch) {
					return
				}
			}

		case gatewayOpHeartbeat:
			// Server requests heartbeat — respond immediately.
			a.sendHeartbeat(conn, lastSeq)
		}
	}
}

// connectGateway resolves the gateway URL, dials the websocket, and reads
// the initial HELLO frame, returning the connection and heartbeat interval.
// On error, any partially opened connection is closed.
func (a *Adapter) connectGateway(ctx context.Context) (*websocket.Conn, int, error) {
	gatewayURL, err := a.fetchGatewayURL(ctx)
	if err != nil {
		return nil, 0, err
	}

	conn, err := a.dialWebSocket(ctx, gatewayURL)
	if err != nil {
		return nil, 0, err
	}

	hello, err := readGatewayPayload(conn)
	if err != nil {
		_ = conn.Close()
		return nil, 0, err
	}
	var hd helloData
	_ = json.Unmarshal(hello.D, &hd)

	return conn, hd.HeartbeatInterval, nil
}

// identifyGateway sends the IDENTIFY payload for this adapter's bot token.
func (a *Adapter) identifyGateway(conn *websocket.Conn) error {
	identify := gatewayPayload{
		Op: gatewayOpIdentify,
		D: mustMarshal(identifyData{
			Token: a.token,
			Properties: identifyProperties{
				OS:      "linux",
				Browser: "dietdaemon",
				Device:  "dietdaemon",
			},
			Intents: gatewayIntentMessageContent,
		}),
	}
	return writeGatewayFrame(conn, identify)
}

// handleMessageCreate processes a MESSAGE_CREATE dispatch payload. It
// returns false if ctx was cancelled while delivering the message, signaling
// that gatewayLoop should exit.
func (a *Adapter) handleMessageCreate(ctx context.Context, pl gatewayPayload, ch chan<- types.InboundMessage) bool {
	var msg messageCreateData
	if err := json.Unmarshal(pl.D, &msg); err != nil {
		return true
	}
	// Skip own messages.
	if msg.Author.Bot {
		return true
	}
	select {
	case ch <- types.InboundMessage{
		UserID: msg.Author.ID,
		At:     time.Now().UTC(),
		Kind:   types.MessageText,
		Text:   msg.Content,
		ChannelMeta: map[string]string{
			"channel_id": msg.ChannelID,
			"message_id": msg.ID,
		},
	}:
		return true
	case <-ctx.Done():
		return false
	}
}

// handleInteractionCreate processes an INTERACTION_CREATE dispatch payload.
// It returns false if ctx was cancelled while delivering the interaction,
// signaling that gatewayLoop should exit.
func (a *Adapter) handleInteractionCreate(ctx context.Context, pl gatewayPayload, ch chan<- types.InboundMessage) bool {
	var interaction interactionCreateData
	if err := json.Unmarshal(pl.D, &interaction); err != nil {
		return true
	}
	// Skip bot interactions.
	if interaction.Member != nil && interaction.Member.User.Bot {
		return true
	}
	if interaction.Data.CustomID == "" {
		return true
	}

	userID := ""
	if interaction.Member != nil {
		userID = interaction.Member.User.ID
	}
	if userID == "" {
		return true // No identifiable user.
	}

	log.Printf("discord: received interaction %s = %q", interaction.ID, interaction.Data.CustomID)

	select {
	case ch <- types.InboundMessage{
		UserID: userID,
		At:     time.Now().UTC(),
		Kind:   types.MessageText,
		Text:   interaction.Data.CustomID,
		ChannelMeta: map[string]string{
			"channel_id":        interaction.ChannelID,
			"is_callback":       "true",
			"interaction_id":    interaction.ID,
			"interaction_token": interaction.Token,
		},
	}:
		return true
	case <-ctx.Done():
		return false
	}
}

func (a *Adapter) fetchGatewayURL(ctx context.Context) (string, error) {
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet,
		RESTBaseURL+"/gateway", nil)
	req.Header.Set("Authorization", "Bot "+a.token)

	resp, err := a.client.Do(req)
	if err != nil {
		return "", err
	}
	defer func() { _ = resp.Body.Close() }()

	var result struct {
		URL string `json:"url"`
	}
	_ = json.NewDecoder(resp.Body).Decode(&result)
	return result.URL, nil
}

func (a *Adapter) heartbeat(ctx context.Context, conn *websocket.Conn, intervalMs int) {
	if intervalMs <= 0 {
		return
	}
	// Discord recommends jitter; we just use the given interval.
	ticker := time.NewTicker(time.Duration(intervalMs) * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			a.sendHeartbeat(conn, nil)
		}
	}
}

func (a *Adapter) sendHeartbeat(conn *websocket.Conn, seq *int) {
	pl := gatewayPayload{Op: gatewayOpHeartbeat}
	if seq != nil {
		b, _ := json.Marshal(*seq)
		pl.D = b
	}
	_ = writeGatewayFrame(conn, pl)
}

func dialWebSocket(ctx context.Context, rawURL string) (*websocket.Conn, error) {
	return dialWebSocketWithTLSConfig(ctx, rawURL, &tls.Config{MinVersion: tls.VersionTLS12})
}

func dialWebSocketWithTLSConfig(ctx context.Context, rawURL string, tlsConfig *tls.Config) (*websocket.Conn, error) {
	config, err := websocket.NewConfig(rawURL, "https://discord.com")
	if err != nil {
		return nil, fmt.Errorf("discord: configure gateway: %w", err)
	}
	config.TlsConfig = tlsConfig
	conn, err := config.DialContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("discord: dial: %w", err)
	}
	return conn, nil
}

func readGatewayPayload(conn *websocket.Conn) (gatewayPayload, error) {
	var payload gatewayPayload
	if err := websocket.JSON.Receive(conn, &payload); err != nil {
		return gatewayPayload{}, fmt.Errorf("discord: read gateway: %w", err)
	}
	return payload, nil
}

func writeGatewayFrame(conn *websocket.Conn, payload gatewayPayload) error {
	if err := websocket.JSON.Send(conn, payload); err != nil {
		return fmt.Errorf("discord: write gateway: %w", err)
	}
	return nil
}

func mustMarshal(v any) json.RawMessage {
	b, _ := json.Marshal(v)
	return b
}
