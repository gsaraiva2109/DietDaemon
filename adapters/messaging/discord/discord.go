// Package discord implements ports.MessagingAdapter for the Discord Bot API.
// Send uses the REST API; Receive connects to the gateway WebSocket to stream
// MESSAGE_CREATE events in real time.
package discord

import (
	"bufio"
	"context"
	"crypto/rand"
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/gsaraiva2109/dietdaemon/core/ports"
	"github.com/gsaraiva2109/dietdaemon/core/types"
)

// Compile-time interface check.
var _ ports.MessagingAdapter = (*Adapter)(nil)

// RESTBaseURL is the Discord REST API base.
const RESTBaseURL = "https://discord.com/api/v10"

// Adapter satisfies ports.MessagingAdapter for Discord.
type Adapter struct {
	token  string
	client *http.Client
}

// New returns a ready Adapter. token is the Discord bot token.
func New(token string) *Adapter {
	return &Adapter{
		token:  token,
		client: &http.Client{Timeout: 30 * time.Second},
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

	// Resolve gateway URL.
	gatewayURL, err := a.fetchGatewayURL(ctx)
	if err != nil {
		return
	}

	conn, err := dialWebSocket(ctx, gatewayURL)
	if err != nil {
		return
	}
	defer func() { _ = conn.Close() }()

	br := bufio.NewReader(conn)

	// Read HELLO.
	hello, err := readGatewayPayload(br)
	if err != nil {
		return
	}
	var hd helloData
	_ = json.Unmarshal(hello.D, &hd)

	// Start heartbeat goroutine.
	heartbeatCtx, cancelBeat := context.WithCancel(ctx)
	defer cancelBeat()
	go a.heartbeat(heartbeatCtx, conn, hd.HeartbeatInterval)

	// Send IDENTIFY.
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
	if err := writeGatewayFrame(conn, identify); err != nil {
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

		pl, err := readGatewayPayload(br)
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
				var msg messageCreateData
				if err := json.Unmarshal(pl.D, &msg); err != nil {
					continue
				}
				// Skip own messages.
				if msg.Author.Bot {
					continue
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
				case <-ctx.Done():
					return
				}

			case "INTERACTION_CREATE":
				var interaction interactionCreateData
				if err := json.Unmarshal(pl.D, &interaction); err != nil {
					continue
				}
				// Skip bot interactions.
				if interaction.Member != nil && interaction.Member.User.Bot {
					continue
				}
				if interaction.Data.CustomID == "" {
					continue
				}

				userID := ""
				if interaction.Member != nil {
					userID = interaction.Member.User.ID
				}
				if userID == "" {
					continue // No identifiable user.
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
				case <-ctx.Done():
					return
				}
			}

		case gatewayOpHeartbeat:
			// Server requests heartbeat — respond immediately.
			a.sendHeartbeat(conn, lastSeq)
		}
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

func (a *Adapter) heartbeat(ctx context.Context, conn *tls.Conn, intervalMs int) {
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

func (a *Adapter) sendHeartbeat(conn *tls.Conn, seq *int) {
	pl := gatewayPayload{Op: gatewayOpHeartbeat}
	if seq != nil {
		b, _ := json.Marshal(*seq)
		pl.D = b
	}
	_ = writeGatewayFrame(conn, pl)
}

// ---------------------------------------------------------------------------
// Minimal WebSocket helpers (stdlib only — no external dep)
// ---------------------------------------------------------------------------

func dialWebSocket(ctx context.Context, rawURL string) (*tls.Conn, error) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return nil, fmt.Errorf("discord: parse gateway url: %w", err)
	}
	host := u.Host
	if !strings.Contains(host, ":") {
		host += ":443"
	}

	dialer := &tls.Dialer{Config: &tls.Config{MinVersion: tls.VersionTLS12}}
	conn, err := dialer.DialContext(ctx, "tcp", host)
	if err != nil {
		return nil, fmt.Errorf("discord: dial: %w", err)
	}
	tlsConn := conn.(*tls.Conn)

	// WebSocket upgrade handshake.
	key := make([]byte, 16)
	rand.Read(key)
	// Sec-WebSocket-Key: 16 random bytes base64-encoded. Discord's gateway
	// does not enforce the SHA-1 accept hash for bot connections.
	wsKey := base64.StdEncoding.EncodeToString(key)

	req := fmt.Sprintf("GET %s HTTP/1.1\r\n", u.RequestURI())
	req += fmt.Sprintf("Host: %s\r\n", u.Hostname())
	req += "Upgrade: websocket\r\n"
	req += "Connection: Upgrade\r\n"
	req += "Sec-WebSocket-Version: 13\r\n"
	req += "Sec-WebSocket-Key: " + wsKey + "\r\n"
	req += "\r\n"

	if _, err := tlsConn.Write([]byte(req)); err != nil {
		_ = tlsConn.Close()
		return nil, fmt.Errorf("discord: ws handshake write: %w", err)
	}

	// Read HTTP 101 response.
	br := bufio.NewReader(tlsConn)
	resp, err := http.ReadResponse(br, nil)
	if err != nil {
		_ = tlsConn.Close()
		return nil, fmt.Errorf("discord: ws handshake read: %w", err)
	}
	if resp.StatusCode != 101 {
		_ = tlsConn.Close()
		return nil, fmt.Errorf("discord: ws upgrade got %d", resp.StatusCode)
	}

	return tlsConn, nil
}

func readGatewayPayload(br *bufio.Reader) (gatewayPayload, error) {
	frame, err := readWSFrame(br)
	if err != nil {
		return gatewayPayload{}, err
	}
	var pl gatewayPayload
	if err := json.Unmarshal(frame, &pl); err != nil {
		return gatewayPayload{}, fmt.Errorf("discord: unmarshal gateway: %w", err)
	}
	return pl, nil
}

func writeGatewayFrame(conn *tls.Conn, pl gatewayPayload) error {
	data, _ := json.Marshal(pl)
	return writeWSFrame(conn, data)
}

// readWSFrame reads a single unmasked text frame from the server. Server frames
// are never masked per RFC 6455.
func readWSFrame(br *bufio.Reader) ([]byte, error) {
	// First two bytes: fin+opcode, mask+len.
	b0, err := br.ReadByte()
	if err != nil {
		return nil, err
	}
	b1, err := br.ReadByte()
	if err != nil {
		return nil, err
	}

	length := int64(b1 & 0x7f)
	switch length {
	case 126:
		var buf [2]byte
		if _, err := io.ReadFull(br, buf[:]); err != nil {
			return nil, err
		}
		length = int64(buf[0])<<8 | int64(buf[1])
	case 127:
		var buf [8]byte
		if _, err := io.ReadFull(br, buf[:]); err != nil {
			return nil, err
		}
		length = int64(buf[0])<<56 | int64(buf[1])<<48 | int64(buf[2])<<40 | int64(buf[3])<<32 |
			int64(buf[4])<<24 | int64(buf[5])<<16 | int64(buf[6])<<8 | int64(buf[7])
	}

	data := make([]byte, length)
	if _, err := io.ReadFull(br, data); err != nil {
		return nil, err
	}

	_ = b0 // opcode ignored; server always sends text frames
	return data, nil
}

// writeWSFrame writes a single masked text frame to the server.
func writeWSFrame(conn *tls.Conn, data []byte) error {
	var frame []byte
	length := len(data)

	frame = append(frame, 0x81) // FIN + text opcode
	if length < 126 {
		// #nosec G115
		frame = append(frame, byte(0x80|length)) // mask bit set
	} else if length < 65536 {
		frame = append(frame, 0xFE) // 126 with mask
		// #nosec G115
		frame = append(frame, byte(length>>8), byte(length))
	} else {
		frame = append(frame, 0xFF) // 127 with mask
		for i := 7; i >= 0; i-- {
			// #nosec G115
			frame = append(frame, byte(length>>(8*i)))
		}
	}

	// Masking key (4 random bytes).
	maskKey := make([]byte, 4)
	rand.Read(maskKey)
	frame = append(frame, maskKey...)

	// Masked payload.
	for i, b := range data {
		frame = append(frame, b^maskKey[i%4])
	}

	_, err := conn.Write(frame)
	return err
}

func mustMarshal(v any) json.RawMessage {
	b, _ := json.Marshal(v)
	return b
}
