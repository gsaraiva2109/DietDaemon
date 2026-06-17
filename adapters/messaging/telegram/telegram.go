// Package telegram implements ports.MessagingAdapter for the Telegram Bot API.
// It long-polls getUpdates to receive text messages and calls sendMessage to
// deliver replies.
package telegram

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/gsaraiva2109/dietdaemon/core/ports"
	"github.com/gsaraiva2109/dietdaemon/core/types"
)

// Compile-time interface check.
var _ ports.MessagingAdapter = (*Adapter)(nil)

// Adapter satisfies ports.MessagingAdapter via the Telegram Bot API.
type Adapter struct {
	token  string
	client *http.Client
	apiURL string
}

// New returns a ready Adapter. token is the Bot API token from @BotFather.
func New(token string) *Adapter {
	return &Adapter{
		token:  token,
		client: &http.Client{Timeout: 60 * time.Second},
		apiURL: "https://api.telegram.org",
	}
}

// Name returns "telegram".
func (a *Adapter) Name() string { return "telegram" }

// ---------------------------------------------------------------------------
// Receive — long-poll getUpdates
// ---------------------------------------------------------------------------

// update is a minimal Telegram Update payload. Only fields used are decoded.
type update struct {
	UpdateID int     `json:"update_id"`
	Message  *tgMsg  `json:"message"`
}

type tgMsg struct {
	MessageID int    `json:"message_id"`
	Text      string `json:"text"`
	Chat      tgChat `json:"chat"`
	From      tgUser `json:"from"`
}

type tgChat struct {
	ID int64 `json:"id"`
}

type tgUser struct {
	LanguageCode string `json:"language_code"`
}

type getUpdatesResponse struct {
	OK     bool     `json:"ok"`
	Result []update `json:"result"`
}

// Receive starts a long-poll loop against getUpdates and emits an
// InboundMessage for each text message received. The channel is closed when
// ctx is cancelled.
func (a *Adapter) Receive(ctx context.Context) (<-chan types.InboundMessage, error) {
	ch := make(chan types.InboundMessage)
	go a.poll(ctx, ch)
	return ch, nil
}

func (a *Adapter) poll(ctx context.Context, ch chan<- types.InboundMessage) {
	defer close(ch)

	var offset int
	pollURL := a.apiURL + "/bot" + a.token + "/getUpdates"

	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		updates, newOffset, err := a.fetchUpdates(ctx, pollURL, offset)
		if err != nil {
			// Back off briefly on error, then retry.
			select {
			case <-ctx.Done():
				return
			case <-time.After(2 * time.Second):
			}
			continue
		}
		if newOffset > offset {
			offset = newOffset
		}

		for _, u := range updates {
			if u.Message == nil || u.Message.Text == "" {
				continue
			}
			msg := types.InboundMessage{
				UserID: strconv.FormatInt(u.Message.Chat.ID, 10),
				At:     time.Now().UTC(),
				Kind:   types.MessageText,
				Text:   u.Message.Text,
				Locale: u.Message.From.LanguageCode,
				ChannelMeta: map[string]string{
					"chat_id":    strconv.FormatInt(u.Message.Chat.ID, 10),
					"message_id": strconv.Itoa(u.Message.MessageID),
				},
			}
			select {
			case ch <- msg:
			case <-ctx.Done():
				return
			}
		}
	}
}

func (a *Adapter) fetchUpdates(ctx context.Context, pollURL string, offset int) ([]update, int, error) {
	u, err := url.Parse(pollURL)
	if err != nil {
		return nil, offset, fmt.Errorf("telegram: parse url: %w", err)
	}

	q := u.Query()
	if offset > 0 {
		q.Set("offset", strconv.Itoa(offset))
	}
	q.Set("timeout", "30")
	q.Set("allowed_updates", `["message"]`)
	u.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, offset, fmt.Errorf("telegram: build request: %w", err)
	}

	resp, err := a.client.Do(req)
	if err != nil {
		return nil, offset, fmt.Errorf("telegram: getUpdates: %w", err)
	}
	defer resp.Body.Close()

	var body getUpdatesResponse
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return nil, offset, fmt.Errorf("telegram: decode: %w", err)
	}
	if !body.OK {
		return nil, offset, fmt.Errorf("telegram: getUpdates not ok")
	}

	newOffset := offset
	for _, u := range body.Result {
		if u.UpdateID >= newOffset {
			newOffset = u.UpdateID + 1
		}
	}
	return body.Result, newOffset, nil
}

// ---------------------------------------------------------------------------
// Send — call sendMessage
// ---------------------------------------------------------------------------

type sendMessageRequest struct {
	ChatID string `json:"chat_id"`
	Text   string `json:"text"`
}

type sendMessageResponse struct {
	OK bool `json:"ok"`
}

// Send delivers a reply to the chat identified by reply.ChannelMeta["chat_id"].
func (a *Adapter) Send(ctx context.Context, reply types.Reply) error {
	chatID := reply.ChannelMeta["chat_id"]
	if chatID == "" {
		return fmt.Errorf("telegram: missing chat_id in ChannelMeta")
	}

	body := sendMessageRequest{ChatID: chatID, Text: reply.Text}
	payload, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("telegram: marshal: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		a.apiURL+"/bot"+a.token+"/sendMessage",
		bytes.NewReader(payload),
	)
	if err != nil {
		return fmt.Errorf("telegram: build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := a.client.Do(req)
	if err != nil {
		return fmt.Errorf("telegram: sendMessage: %w", err)
	}
	defer resp.Body.Close()

	var r sendMessageResponse
	if err := json.NewDecoder(resp.Body).Decode(&r); err != nil {
		return fmt.Errorf("telegram: decode sendMessage: %w", err)
	}
	if !r.OK {
		return fmt.Errorf("telegram: sendMessage not ok")
	}
	return nil
}
