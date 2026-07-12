package telegram

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gsaraiva2109/dietdaemon/core/types"
)

// ---------------------------------------------------------------------------
// JSON marshal / unmarshal
// ---------------------------------------------------------------------------

func TestUpdateUnmarshal(t *testing.T) {
	body := `{
		"ok": true,
		"result": [
			{
				"update_id": 100,
				"message": {
					"message_id": 42,
					"text": "200g frango",
					"chat": {"id": 123456},
					"from": {"language_code": "pt-BR"}
				}
			}
		]
	}`

	var resp getUpdatesResponse
	if err := json.Unmarshal([]byte(body), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if !resp.OK {
		t.Error("expected ok=true")
	}
	if len(resp.Result) != 1 {
		t.Fatalf("expected 1 update, got %d", len(resp.Result))
	}
	u := resp.Result[0]
	if u.UpdateID != 100 || u.Message.MessageID != 42 || u.Message.Text != "200g frango" {
		t.Errorf("fields mismatch: %+v", u)
	}
	if u.Message.Chat.ID != 123456 {
		t.Errorf("chat_id = %d, want 123456", u.Message.Chat.ID)
	}
	if u.Message.From.LanguageCode != "pt-BR" {
		t.Errorf("language_code = %q, want pt-BR", u.Message.From.LanguageCode)
	}
}

func TestSendMessageMarshal(t *testing.T) {
	req := sendMessageRequest{ChatID: "123456", Text: "ok"}
	b, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var got sendMessageRequest
	if err := json.Unmarshal(b, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if got.ChatID != "123456" || got.Text != "ok" {
		t.Errorf("got %+v", got)
	}
}

// ---------------------------------------------------------------------------
// Receive — ChannelMeta mapping + message emission
// ---------------------------------------------------------------------------

func TestReceiveEmitsMessages(t *testing.T) {
	// Fake Telegram API that returns two updates then idles.
	callCount := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		var resp getUpdatesResponse
		switch callCount {
		case 1:
			resp = getUpdatesResponse{
				OK: true,
				Result: []update{
					{
						UpdateID: 1,
						Message: &tgMsg{
							MessageID: 10,
							Text:      "200g frango",
							Chat:      tgChat{ID: 111},
							From:      tgUser{LanguageCode: "pt-BR"},
						},
					},
				},
			}
		case 2:
			resp = getUpdatesResponse{
				OK: true,
				Result: []update{
					{
						UpdateID: 2,
						Message: &tgMsg{
							MessageID: 11,
							Text:      "2 ovos",
							Chat:      tgChat{ID: 222},
							From:      tgUser{LanguageCode: "en"},
						},
					},
				},
			}
		default:
			// Simulate long-poll timeout: return empty.
			resp = getUpdatesResponse{OK: true, Result: nil}
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	a := New("test-token")
	a.apiURL = srv.URL
	a.client.Timeout = 2 * time.Second

	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	ch, err := a.Receive(ctx)
	if err != nil {
		t.Fatalf("Receive: %v", err)
	}

	var msgs []types.InboundMessage
	for m := range ch {
		msgs = append(msgs, m)
	}

	if len(msgs) < 2 {
		t.Fatalf("expected at least 2 messages, got %d", len(msgs))
	}

	// Message 1.
	m1 := msgs[0]
	if m1.UserID != "111" || m1.Text != "200g frango" || m1.Locale != "pt-BR" {
		t.Errorf("msg1: %+v", m1)
	}
	if m1.ChannelMeta["chat_id"] != "111" || m1.ChannelMeta["message_id"] != "10" {
		t.Errorf("msg1 ChannelMeta: %v", m1.ChannelMeta)
	}
	if m1.Kind != types.MessageText {
		t.Errorf("msg1 Kind = %q, want text", m1.Kind)
	}

	// Message 2.
	m2 := msgs[1]
	if m2.UserID != "222" || m2.Text != "2 ovos" || m2.Locale != "en" {
		t.Errorf("msg2: %+v", m2)
	}
}

// ---------------------------------------------------------------------------
// Send
// ---------------------------------------------------------------------------

func TestSend(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %q, want POST", r.Method)
		}
		if r.URL.Path != "/bottest-token/sendMessage" {
			t.Errorf("path = %q", r.URL.Path)
		}

		body, _ := io.ReadAll(r.Body)
		var req sendMessageRequest
		if err := json.Unmarshal(body, &req); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		if req.ChatID != "111" || req.Text != "hello" {
			t.Errorf("request = %+v", req)
		}

		_ = json.NewEncoder(w).Encode(sendMessageResponse{OK: true})
	}))
	defer srv.Close()

	a := New("test-token")
	a.apiURL = srv.URL

	reply := types.Reply{
		UserID: "u1",
		Text:   "hello",
		ChannelMeta: map[string]string{
			"chat_id":    "111",
			"message_id": "10",
		},
	}

	if err := a.Send(context.Background(), reply); err != nil {
		t.Fatalf("Send: %v", err)
	}
}

func TestSendMissingChatID(t *testing.T) {
	a := New("t")
	reply := types.Reply{Text: "x", ChannelMeta: nil}
	if err := a.Send(context.Background(), reply); err == nil {
		t.Error("expected error for missing chat_id")
	}
}

func TestSendErrorResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(sendMessageResponse{OK: false})
	}))
	defer srv.Close()

	a := New("t")
	a.apiURL = srv.URL
	reply := types.Reply{ChannelMeta: map[string]string{"chat_id": "1"}, Text: "x"}
	if err := a.Send(context.Background(), reply); err == nil {
		t.Error("expected error on ok=false")
	}
}

// ---------------------------------------------------------------------------
// Name + interface guard
// ---------------------------------------------------------------------------

func TestName(t *testing.T) {
	a := New("t")
	if a.Name() != "telegram" {
		t.Errorf("Name() = %q, want telegram", a.Name())
	}
}
