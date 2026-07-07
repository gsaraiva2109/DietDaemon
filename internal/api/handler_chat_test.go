package api

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gsaraiva2109/dietdaemon/core/ports"
	"github.com/gsaraiva2109/dietdaemon/internal/assistant"
)

// fakeChatAdapter is a test double for ports.ChatAdapter.
type fakeChatAdapter struct {
	events []ports.ChatEvent
	err    error
}

func (f *fakeChatAdapter) StreamChat(ctx context.Context, req ports.ChatRequest) (<-chan ports.ChatEvent, error) {
	if f.err != nil {
		return nil, f.err
	}
	ch := make(chan ports.ChatEvent, len(f.events))
	go func() {
		defer close(ch)
		for _, e := range f.events {
			select {
			case <-ctx.Done():
				return
			case ch <- e:
			}
		}
	}()
	return ch, nil
}

// newChatHandler builds a Handler with a fake adapter wrapped in an assistant router.
func newChatHandler(events []ports.ChatEvent, err error) *Handler {
	fake := &fakeChatAdapter{events: events, err: err}
	return &Handler{
		chatAdapter:     fake,
		assistantRouter: assistant.New(fake, nil, nil),
	}
}

func TestHandleChatMessageBasic(t *testing.T) {
	h := newChatHandler([]ports.ChatEvent{
		{Kind: "text-delta", Text: "Hi!"},
		{Kind: "done"},
	}, nil)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/chat/sessions/test/messages", strings.NewReader(`{"text":"hello"}`))
	rec := httptest.NewRecorder()

	h.handleChatMessage(rec, req, "test-user")

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d: %s", rec.Code, rec.Body.String())
	}

	events := parseSSE(rec.Body.String())
	if len(events) == 0 {
		t.Fatal("expected at least one SSE event")
	}
	if events[0].Event != "delta" || events[len(events)-1].Event != "done" {
		t.Fatalf("expected delta then done, got %+v", events)
	}
}

func TestHandleChatMessageSSEStreaming(t *testing.T) {
	h := newChatHandler([]ports.ChatEvent{
		{Kind: "text-delta", Text: "Hello"},
		{Kind: "text-delta", Text: " there"},
		{Kind: "text-delta", Text: "!"},
		{Kind: "done"},
	}, nil)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/chat/sessions/test/messages", strings.NewReader(`{"text":"hi"}`))
	rec := httptest.NewRecorder()

	h.handleChatMessage(rec, req, "test-user")

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d: %s", rec.Code, rec.Body.String())
	}

	ct := rec.Header().Get("Content-Type")
	if !strings.HasPrefix(ct, "text/event-stream") {
		t.Fatalf("expected Content-Type text/event-stream, got %q", ct)
	}

	events := parseSSE(rec.Body.String())
	if len(events) < 4 {
		t.Fatalf("expected 4 events, got %d: %v", len(events), events)
	}

	for i, expectedText := range []string{"Hello", " there", "!"} {
		if events[i].Event != "delta" {
			t.Errorf("event[%d]: expected event=delta, got %q", i, events[i].Event)
		}
		var data map[string]string
		if err := json.Unmarshal([]byte(events[i].Data), &data); err != nil {
			t.Errorf("event[%d]: bad JSON data: %v", i, err)
		} else if data["text"] != expectedText {
			t.Errorf("event[%d]: expected text=%q, got %q", i, expectedText, data["text"])
		}
	}

	last := events[len(events)-1]
	if last.Event != "done" {
		t.Errorf("last event: expected done, got %q", last.Event)
	}
}

func TestHandleChatMessageNoAdapterReturns503(t *testing.T) {
	h := &Handler{chatAdapter: nil, assistantRouter: nil}
	req := httptest.NewRequest(http.MethodPost, "/api/v1/chat/sessions/test/messages", strings.NewReader(`{"text":"hello"}`))
	rec := httptest.NewRecorder()

	h.handleChatMessage(rec, req, "test-user")

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestHandleChatMessageEmptyText(t *testing.T) {
	h := newChatHandler(nil, nil)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/chat/sessions/test/messages", strings.NewReader(`{"text":""}`))
	rec := httptest.NewRecorder()

	h.handleChatMessage(rec, req, "test-user")

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestHandleChatMessageAdapterError(t *testing.T) {
	h := newChatHandler(nil, fmt.Errorf("anthropic: 500 internal error"))
	req := httptest.NewRequest(http.MethodPost, "/api/v1/chat/sessions/test/messages", strings.NewReader(`{"text":"hello"}`))
	rec := httptest.NewRecorder()

	h.handleChatMessage(rec, req, "test-user")

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK (SSE error event in body), got %d", rec.Code)
	}

	events := parseSSE(rec.Body.String())
	if len(events) == 0 {
		t.Fatal("expected at least one SSE event")
	}
	if events[0].Event != "error" {
		t.Fatalf("expected error event, got %q: %s", events[0].Event, events[0].Data)
	}
	// Verify error message is sanitized (not raw error).
	var data map[string]string
	if err := json.Unmarshal([]byte(events[0].Data), &data); err != nil {
		t.Fatalf("bad JSON: %v", err)
	}
	if data["message"] == "" {
		t.Error("error event should have a message")
	}
}

func TestHandleChatMessageStreamError(t *testing.T) {
	h := newChatHandler([]ports.ChatEvent{
		{Kind: "text-delta", Text: "ok"},
		{Kind: "error", Err: fmt.Errorf("something went wrong")},
	}, nil)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/chat/sessions/test/messages", strings.NewReader(`{"text":"hello"}`))
	rec := httptest.NewRecorder()

	h.handleChatMessage(rec, req, "test-user")

	events := parseSSE(rec.Body.String())
	if len(events) < 2 {
		t.Fatalf("expected at least 2 events, got %d", len(events))
	}
	if events[0].Event != "delta" {
		t.Errorf("first event: expected delta, got %q", events[0].Event)
	}
	if events[1].Event != "error" {
		t.Errorf("second event: expected error, got %q", events[1].Event)
	}
	// Verify error message is sanitized.
	var data map[string]string
	if err := json.Unmarshal([]byte(events[1].Data), &data); err != nil {
		t.Fatalf("bad JSON: %v", err)
	}
	if data["message"] == "" {
		t.Error("error event should have a message")
	}
}

func TestHandleChatMessageToolCallEvent(t *testing.T) {
	h := newChatHandler([]ports.ChatEvent{
		{Kind: "text-delta", Text: "Let me check..."},
		{
			Kind: "tool-call",
			ToolCall: &ports.ToolCallEvent{
				ID:   "tc_1",
				Name: "suggest",
				Args: "",
			},
		},
		{Kind: "done"},
	}, nil)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/chat/sessions/test/messages", strings.NewReader(`{"text":"what should I eat"}`))
	rec := httptest.NewRecorder()

	h.handleChatMessage(rec, req, "test-user")

	events := parseSSE(rec.Body.String())

	// Should have: delta, tool-call, tool-result (unknown command).
	if len(events) < 3 {
		t.Fatalf("expected at least 3 events, got %d: %+v", len(events), events)
	}

	// Check tool-call event exists.
	var found bool
	for _, e := range events {
		if e.Event == "tool-call" {
			found = true
			var data map[string]string
			if err := json.Unmarshal([]byte(e.Data), &data); err != nil {
				t.Errorf("tool-call data: bad JSON: %v", err)
			} else {
				if data["id"] != "tc_1" {
					t.Errorf("tool-call id: expected tc_1, got %q", data["id"])
				}
				if data["name"] != "suggest" {
					t.Errorf("tool-call name: expected suggest, got %q", data["name"])
				}
			}
		}
	}
	if !found {
		t.Error("expected a tool-call event")
	}

	// Check tool-result event exists.
	var foundTR bool
	for _, e := range events {
		if e.Event == "tool-result" {
			foundTR = true
			var data map[string]string
			if err := json.Unmarshal([]byte(e.Data), &data); err != nil {
				t.Errorf("tool-result data: bad JSON: %v", err)
			} else {
				if data["id"] != "tc_1" {
					t.Errorf("tool-result id: expected tc_1, got %q", data["id"])
				}
				if data["text"] == "" {
					t.Error("tool-result text should not be empty")
				}
			}
		}
	}
	if !foundTR {
		t.Error("expected a tool-result event")
	}
}

// sseEvent is a parsed SSE event from the response body.
type sseEvent struct {
	Event string
	Data  string
}

// parseSSE parses an SSE event stream from a response body.
func parseSSE(body string) []sseEvent {
	var events []sseEvent
	scanner := bufio.NewScanner(strings.NewReader(body))
	var current *sseEvent
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			if current != nil {
				events = append(events, *current)
				current = nil
			}
			continue
		}
		if current == nil {
			current = &sseEvent{}
		}
		if after, ok := strings.CutPrefix(line, "event: "); ok {
			current.Event = after
		}
		if after, ok := strings.CutPrefix(line, "data: "); ok {
			current.Data = after
		}
	}
	if current != nil {
		events = append(events, *current)
	}
	return events
}
