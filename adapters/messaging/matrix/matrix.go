// Package matrix implements ports.MessagingAdapter for the Matrix protocol.
// Send uses the Client-Server REST API; Receive long-polls the /sync endpoint.
package matrix

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/gsaraiva2109/dietdaemon/core/ports"
	"github.com/gsaraiva2109/dietdaemon/core/types"
)

// Compile-time interface check.
var _ ports.MessagingAdapter = (*Adapter)(nil)

// Adapter satisfies ports.MessagingAdapter for Matrix.
type Adapter struct {
	homeserverURL string // e.g. "https://matrix-client.matrix.org"
	userID        string // e.g. "@dietdaemon:example.com"
	token         string // access token
	client        *http.Client
}

// New returns a ready Adapter. homeserverURL is the Matrix homeserver base,
// userID is the bot's full Matrix ID, and token is the access token from login.
func New(homeserverURL, userID, token string) *Adapter {
	return &Adapter{
		homeserverURL: strings.TrimRight(homeserverURL, "/"),
		userID:        userID,
		token:         token,
		client:        &http.Client{Timeout: 60 * time.Second},
	}
}

// Name returns "matrix".
func (a *Adapter) Name() string { return "matrix" }

// ---------------------------------------------------------------------------
// Send — PUT /_matrix/client/v3/rooms/{roomId}/send/m.room.message/{txnId}
// ---------------------------------------------------------------------------

type matrixMessageContent struct {
	MsgType string `json:"msgtype"`
	Body    string `json:"body"`
}

// Send delivers a text reply to the room identified by reply.ChannelMeta["room_id"].
func (a *Adapter) Send(ctx context.Context, reply types.Reply) error {
	roomID := reply.ChannelMeta["room_id"]
	if roomID == "" {
		return fmt.Errorf("matrix: missing room_id in ChannelMeta")
	}

	txnID := fmt.Sprintf("dietdaemon-%d", time.Now().UnixNano())

	body := matrixMessageContent{MsgType: "m.text", Body: reply.Text}
	payload, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("matrix: marshal: %w", err)
	}

	u := fmt.Sprintf("%s/_matrix/client/v3/rooms/%s/send/m.room.message/%s",
		a.homeserverURL, roomID, txnID)

	req, err := http.NewRequestWithContext(ctx, http.MethodPut, u,
		strings.NewReader(string(payload)))
	if err != nil {
		return fmt.Errorf("matrix: build request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+a.token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := a.client.Do(req)
	if err != nil {
		return fmt.Errorf("matrix: send: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("matrix: status %d: %s", resp.StatusCode, string(body))
	}
	return nil
}

// ---------------------------------------------------------------------------
// Receive — long-poll GET /_matrix/client/v3/sync
// ---------------------------------------------------------------------------

type syncResponse struct {
	NextBatch string `json:"next_batch"`
	Rooms     struct {
		Join map[string]joinedRoom `json:"join"`
	} `json:"rooms"`
}

type joinedRoom struct {
	Timeline struct {
		Events []timelineEvent `json:"events"`
	} `json:"timeline"`
}

type timelineEvent struct {
	Type    string `json:"type"`
	Sender  string `json:"sender"`
	EventID string `json:"event_id"`
	Content struct {
		Body    string `json:"body"`
		MsgType string `json:"msgtype"`
	} `json:"content"`
}

// Receive starts a long-poll loop against /sync and emits InboundMessage values
// for m.room.message events. The channel closes when ctx is cancelled.
func (a *Adapter) Receive(ctx context.Context) (<-chan types.InboundMessage, error) {
	ch := make(chan types.InboundMessage)
	go a.syncLoop(ctx, ch)
	return ch, nil
}

func (a *Adapter) syncLoop(ctx context.Context, ch chan<- types.InboundMessage) {
	defer close(ch)

	baseURL := a.homeserverURL + "/_matrix/client/v3/sync"
	since := ""

	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		u := baseURL + "?timeout=30000"
		if since != "" {
			u += "&since=" + since
		}

		req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
		if err != nil {
			return
		}
		req.Header.Set("Authorization", "Bearer "+a.token)

		resp, err := a.client.Do(req)
		if err != nil {
			// Back off briefly, then retry.
			select {
			case <-ctx.Done():
				return
			case <-time.After(2 * time.Second):
			}
			continue
		}

		var sr syncResponse
		if err := json.NewDecoder(resp.Body).Decode(&sr); err != nil {
			_ = resp.Body.Close()
			continue
		}
		_ = resp.Body.Close()

		since = sr.NextBatch

		for roomID, room := range sr.Rooms.Join {
			for _, ev := range room.Timeline.Events {
				if ev.Type != "m.room.message" {
					continue
				}
				if ev.Sender == a.userID {
					continue // skip own messages
				}
				if ev.Content.MsgType != "m.text" {
					continue
				}
				select {
				case ch <- types.InboundMessage{
					UserID: ev.Sender,
					At:     time.Now().UTC(),
					Kind:   types.MessageText,
					Text:   ev.Content.Body,
					ChannelMeta: map[string]string{
						"room_id":  roomID,
						"event_id": ev.EventID,
					},
				}:
				case <-ctx.Done():
					return
				}
			}
		}
	}
}
