// Package matrix implements ports.MessagingAdapter for the Matrix protocol.
// Send uses the Client-Server REST API; Receive long-polls the /sync endpoint.
package matrix

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gsaraiva2109/dietdaemon/core/ports"
	"github.com/gsaraiva2109/dietdaemon/core/types"
)

// Compile-time interface check.
var _ ports.MessagingAdapter = (*Adapter)(nil)

// Adapter satisfies ports.MessagingAdapter for Matrix.
type Adapter struct {
	homeserverURL  string // e.g. "https://matrix-client.matrix.org"
	userID         string // e.g. "@dietdaemon:example.com"
	token          string // access token
	client         *http.Client
	pendingMarkups *pendingMarkupStore // stores recent markups for number-to-callback resolution
}

// New returns a ready Adapter. homeserverURL is the Matrix homeserver base,
// userID is the bot's full Matrix ID, and token is the access token from login.
func New(homeserverURL, userID, token string) *Adapter {
	return &Adapter{
		homeserverURL:  strings.TrimRight(homeserverURL, "/"),
		userID:         userID,
		token:          token,
		client:         &http.Client{Timeout: 60 * time.Second},
		pendingMarkups: newPendingMarkupStore(),
	}
}

// Name returns "matrix".
func (a *Adapter) Name() string { return "matrix" }

// ---------------------------------------------------------------------------
// Pending markup store — maps room IDs to the last sent markup so a numbered
// reply can be resolved to a callback data value.
// ---------------------------------------------------------------------------

type pendingMarkupStore struct {
	mu      sync.Mutex
	markups map[string]types.ReplyMarkup // roomID -> markup
}

func newPendingMarkupStore() *pendingMarkupStore {
	return &pendingMarkupStore{markups: make(map[string]types.ReplyMarkup)}
}

func (s *pendingMarkupStore) store(roomID string, markup types.ReplyMarkup) {
	s.mu.Lock()
	s.markups[roomID] = markup
	s.mu.Unlock()
}

func (s *pendingMarkupStore) get(roomID string) (types.ReplyMarkup, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	m, ok := s.markups[roomID]
	return m, ok
}

// callbackDataByIndex returns the CallbackData of the button at the given
// flat index (0-based), or false if the index is out of range.
func callbackDataByIndex(markup types.ReplyMarkup, idx int) (string, bool) {
	cur := 0
	for _, row := range markup.InlineKeyboard {
		for _, btn := range row {
			if cur == idx {
				return btn.CallbackData, true
			}
			cur++
		}
	}
	return "", false
}

// ---------------------------------------------------------------------------
// Send — PUT /_matrix/client/v3/rooms/{roomId}/send/m.room.message/{txnId}
// ---------------------------------------------------------------------------

type matrixMessageContent struct {
	MsgType string `json:"msgtype"`
	Body    string `json:"body"`
}

// Send delivers a text reply to the room identified by reply.ChannelMeta["room_id"].
// When the reply has a Markup, numbered options are appended as text fallback
// and the markup is stored so the next user reply can resolve numbers to
// callback data.
func (a *Adapter) Send(ctx context.Context, reply types.Reply) error {
	roomID := reply.ChannelMeta["room_id"]
	if roomID == "" {
		return fmt.Errorf("matrix: missing room_id in ChannelMeta")
	}

	txnID := fmt.Sprintf("dietdaemon-%d", time.Now().UnixNano())

	body := matrixMessageContent{MsgType: "m.text", Body: reply.Text}

	// Convert markup to numbered text fallback.
	if reply.Markup != nil && len(reply.Markup.InlineKeyboard) > 0 {
		var b strings.Builder
		b.WriteString(reply.Text)
		b.WriteString("\n\nReply with:")
		idx := 1
		for _, row := range reply.Markup.InlineKeyboard {
			for _, btn := range row {
				fmt.Fprintf(&b, "\n%d — %s", idx, btn.Text)
				idx++
			}
		}
		body.Body = b.String()
		log.Printf("matrix: appended %d option(s) to message body", idx-1)

		// Store the markup so numbered replies can be resolved.
		a.pendingMarkups.store(roomID, *reply.Markup)
	}

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
	defer func() { _ = resp.Body.Close() }()

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
		sr, retry, ok := a.pollSync(ctx, baseURL, since)
		if !ok {
			return
		}
		if retry {
			continue
		}

		since = sr.NextBatch
		if !a.emitEvents(ctx, ch, sr.Rooms.Join) {
			return
		}
	}
}

// pollSync returns retry when the current token remains valid and the next
// request should reuse it. ok is false when the loop must stop.
func (a *Adapter) pollSync(ctx context.Context, baseURL, since string) (sr syncResponse, retry, ok bool) {
	select {
	case <-ctx.Done():
		return syncResponse{}, false, false
	default:
	}

	u := baseURL + "?timeout=30000"
	if since != "" {
		u += "&since=" + since
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return syncResponse{}, false, false
	}
	req.Header.Set("Authorization", "Bearer "+a.token)

	resp, err := a.client.Do(req)
	if err != nil {
		select {
		case <-ctx.Done():
			return syncResponse{}, false, false
		case <-time.After(2 * time.Second):
			return syncResponse{}, true, true
		}
	}
	defer func() { _ = resp.Body.Close() }()

	if err := json.NewDecoder(resp.Body).Decode(&sr); err != nil {
		return syncResponse{}, true, true
	}
	return sr, false, true
}

func (a *Adapter) emitEvents(ctx context.Context, ch chan<- types.InboundMessage, rooms map[string]joinedRoom) bool {
	for roomID, room := range rooms {
		for _, ev := range room.Timeline.Events {
			msg, ok := a.messageFromEvent(roomID, ev)
			if !ok {
				continue
			}
			select {
			case ch <- msg:
			case <-ctx.Done():
				return false
			}
		}
	}
	return true
}

func (a *Adapter) messageFromEvent(roomID string, ev timelineEvent) (types.InboundMessage, bool) {
	if ev.Type != "m.room.message" || ev.Sender == a.userID || ev.Content.MsgType != "m.text" {
		return types.InboundMessage{}, false
	}

	text := ev.Content.Body
	if markup, ok := a.pendingMarkups.get(roomID); ok {
		if num, err := strconv.Atoi(strings.TrimSpace(text)); err == nil {
			if callback, found := callbackDataByIndex(markup, num-1); found {
				log.Printf("matrix: resolved number %d to callback data %q in room %s", num, callback, roomID)
				text = callback
			}
		}
	}

	return types.InboundMessage{
		UserID: ev.Sender,
		At:     time.Now().UTC(),
		Kind:   types.MessageText,
		Text:   text,
		ChannelMeta: map[string]string{
			"room_id":  roomID,
			"event_id": ev.EventID,
		},
	}, true
}
