package anthropic

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gsaraiva2109/dietdaemon/adapters/model/internal/ssetest"
	"github.com/gsaraiva2109/dietdaemon/core/ports"
)

// drainReadStream feeds sse into readStream on a real channel and collects
// every event until the channel closes.
func drainReadStream(ctx context.Context, sse string) []ports.ChatEvent {
	c := &ChatAdapter{}
	return ssetest.Drain(ctx, io.NopCloser(strings.NewReader(sse)), c.readStream)
}

// TestExtractArgsEmptyValue guards the bug where a legitimately empty args
// value (no-arg commands like /help emit {"args":""}) was misread as a parse
// failure and the raw JSON blob leaked through as the command's argument.
func TestExtractArgsEmptyValue(t *testing.T) {
	if got := extractArgs(`{"args": ""}`); got != "" {
		t.Errorf("extractArgs(empty args) = %q, want empty string", got)
	}
	if got := extractArgs(`{"args": "grilled chicken"}`); got != "grilled chicken" {
		t.Errorf("extractArgs = %q, want %q", got, "grilled chicken")
	}
	if got := extractArgs(`not json`); got != "not json" {
		t.Errorf("extractArgs(invalid json) = %q, want raw fallback", got)
	}
}

func TestStreamChatHTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":{"message":"invalid model"}}`))
	}))
	defer srv.Close()

	c := &ChatAdapter{apiKey: "test-key", model: "claude-haiku-4-5-20251001", client: &http.Client{Timeout: 5 * time.Second}, baseURL: srv.URL}
	_, err := c.StreamChat(t.Context(), ports.ChatRequest{Messages: []ports.ChatMessage{{Role: "user", Content: "hi"}}})
	if err == nil {
		t.Fatal("expected error on 400, got nil")
	}
	if !strings.Contains(err.Error(), "invalid model") {
		t.Errorf("error = %q, want it to include the response body detail", err.Error())
	}
}

// TestToWireMessagesToolRoundTrip guards the bug where a "tool" role message
// was passed straight through to Anthropic (which rejects any role other than
// user/assistant) and the assistant's tool_use block was dropped from history
// entirely — Anthropic requires a tool_result's tool_use_id to match a
// tool_use block in the immediately preceding assistant turn.
func TestToWireMessagesToolRoundTrip(t *testing.T) {
	msgs := []ports.ChatMessage{
		{Role: "user", Content: "what should I eat"},
		{
			Role:      "assistant",
			Content:   "",
			ToolCalls: []ports.ToolCallEvent{{ID: "tc_1", Name: "suggest", Args: "breakfast"}},
		},
		{Role: "tool", Content: "eat oatmeal", ToolCallID: "tc_1"},
	}

	wire := toWireMessages(msgs)
	if len(wire) != 3 {
		t.Fatalf("got %d messages, want 3", len(wire))
	}

	for _, m := range wire {
		if m.Role != "user" && m.Role != "assistant" {
			t.Fatalf("message role %q is invalid for Anthropic (only user/assistant allowed)", m.Role)
		}
	}

	assistantMsg := wire[1]
	if assistantMsg.Role != "assistant" {
		t.Fatalf("wire[1].Role = %q, want assistant", assistantMsg.Role)
	}
	if len(assistantMsg.Content) != 1 || assistantMsg.Content[0].Type != "tool_use" {
		t.Fatalf("wire[1].Content = %+v, want single tool_use block", assistantMsg.Content)
	}
	if assistantMsg.Content[0].ID != "tc_1" || assistantMsg.Content[0].Name != "suggest" {
		t.Errorf("tool_use block = %+v, want ID tc_1 Name suggest", assistantMsg.Content[0])
	}
	var input struct {
		Args string `json:"args"`
	}
	if err := json.Unmarshal(assistantMsg.Content[0].Input, &input); err != nil || input.Args != "breakfast" {
		t.Errorf("tool_use input = %s, want {\"args\":\"breakfast\"}", assistantMsg.Content[0].Input)
	}

	toolResultMsg := wire[2]
	if toolResultMsg.Role != "user" {
		t.Fatalf("wire[2].Role = %q, want user (tool_result must be a user turn)", toolResultMsg.Role)
	}
	if len(toolResultMsg.Content) != 1 || toolResultMsg.Content[0].Type != "tool_result" {
		t.Fatalf("wire[2].Content = %+v, want single tool_result block", toolResultMsg.Content)
	}
	if toolResultMsg.Content[0].ToolUseID != "tc_1" {
		t.Errorf("tool_result.tool_use_id = %q, want tc_1 (must match the preceding tool_use block)", toolResultMsg.Content[0].ToolUseID)
	}
	if toolResultMsg.Content[0].Content != "eat oatmeal" {
		t.Errorf("tool_result.content = %q, want %q", toolResultMsg.Content[0].Content, "eat oatmeal")
	}
}

// TestReadStreamTextDeltaSingle covers the happy path: one text_delta then
// message_stop.
func TestReadStreamTextDeltaSingle(t *testing.T) {
	sse := `data: {"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":"Hello"}}
data: {"type":"message_stop"}
`
	events := drainReadStream(t.Context(), sse)

	ssetest.AssertEvents(t, events, []ports.ChatEvent{
		{Kind: "text-delta", Text: "Hello"},
		{Kind: "done"},
	})
}

// TestReadStreamTextDeltaMultiple covers several text_delta chunks arriving
// in sequence before message_stop.
func TestReadStreamTextDeltaMultiple(t *testing.T) {
	sse := `data: {"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":"Hello "}}
data: {"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":"world"}}
data: {"type":"message_stop"}
`
	events := drainReadStream(t.Context(), sse)

	ssetest.AssertEvents(t, events, []ports.ChatEvent{
		{Kind: "text-delta", Text: "Hello "},
		{Kind: "text-delta", Text: "world"},
		{Kind: "done"},
	})
}

// TestReadStreamToolCallRoundTrip covers content_block_start(tool_use) ->
// accumulating input_json_delta chunks -> content_block_stop, verifying the
// emitted tool-call event has args parsed via extractArgs.
func TestReadStreamToolCallRoundTrip(t *testing.T) {
	sse := `data: {"type":"content_block_start","index":0,"content_block":{"type":"tool_use","id":"toolu_1","name":"suggest_meal"}}
data: {"type":"content_block_delta","index":0,"delta":{"type":"input_json_delta","partial_json":"{\"args\":\""}}
data: {"type":"content_block_delta","index":0,"delta":{"type":"input_json_delta","partial_json":"grilled chicken\"}"}}
data: {"type":"content_block_stop","index":0}
data: {"type":"message_stop"}
`
	events := drainReadStream(t.Context(), sse)

	if len(events) != 2 {
		t.Fatalf("got %d events, want 2 (tool-call, done): %+v", len(events), events)
	}
	tc := events[0]
	if tc.Kind != "tool-call" {
		t.Fatalf("event[0].Kind = %q, want tool-call", tc.Kind)
	}
	if tc.ToolCall == nil {
		t.Fatal("event[0].ToolCall = nil")
	}
	want := ports.ToolCallEvent{ID: "toolu_1", Name: "suggest_meal", Args: "grilled chicken"}
	if *tc.ToolCall != want {
		t.Errorf("ToolCall = %+v, want %+v", *tc.ToolCall, want)
	}
	if events[1].Kind != "done" {
		t.Errorf("event[1].Kind = %q, want done", events[1].Kind)
	}
}

// TestReadStreamErrorEvent covers an "error" SSE event: it emits a formatted
// error event and returns without processing anything after it.
func TestReadStreamErrorEvent(t *testing.T) {
	sse := `data: {"type":"error","error":{"type":"overloaded_error","message":"Overloaded"}}
data: {"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":"should not appear"}}
`
	events := drainReadStream(t.Context(), sse)

	if len(events) != 1 {
		t.Fatalf("got %d events, want 1 (error only, stream must stop): %+v", len(events), events)
	}
	if events[0].Kind != "error" {
		t.Fatalf("event.Kind = %q, want error", events[0].Kind)
	}
	if events[0].Err == nil || events[0].Err.Error() != "anthropic: overloaded_error: Overloaded" {
		t.Errorf("event.Err = %v, want %q", events[0].Err, "anthropic: overloaded_error: Overloaded")
	}
}

// TestReadStreamMalformedJSONSkipped covers a malformed data line being
// silently skipped, with subsequent valid lines still processed.
func TestReadStreamMalformedJSONSkipped(t *testing.T) {
	sse := `data: {not valid json
data: {"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":"ok"}}
data: {"type":"message_stop"}
`
	events := drainReadStream(t.Context(), sse)

	ssetest.AssertEvents(t, events, []ports.ChatEvent{
		{Kind: "text-delta", Text: "ok"},
		{Kind: "done"},
	})
}

// TestReadStreamNonDataLinesSkipped covers SSE "event: ..." lines and blank
// lines being skipped since they don't have the "data: " prefix.
func TestReadStreamNonDataLinesSkipped(t *testing.T) {
	sse := "event: content_block_delta\n" +
		`data: {"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":"hi"}}` + "\n" +
		"\n" +
		`data: {"type":"message_stop"}` + "\n"

	events := drainReadStream(t.Context(), sse)

	ssetest.AssertEvents(t, events, []ports.ChatEvent{
		{Kind: "text-delta", Text: "hi"},
		{Kind: "done"},
	})
}

// TestReadStreamScannerReadError covers the scanner erroring mid-read: it
// must emit a single error event carrying the underlying error.
func TestReadStreamScannerReadError(t *testing.T) {
	c := &ChatAdapter{}
	events := ssetest.Drain(t.Context(), &ssetest.ErrReader{Err: errors.New("boom")}, c.readStream)

	if len(events) != 1 {
		t.Fatalf("got %d events, want 1 error event: %+v", len(events), events)
	}
	if events[0].Kind != "error" {
		t.Fatalf("event.Kind = %q, want error", events[0].Kind)
	}
	if events[0].Err == nil || !strings.Contains(events[0].Err.Error(), "boom") {
		t.Errorf("event.Err = %v, want it to include the underlying read error", events[0].Err)
	}
}

// TestReadStreamContextCancelledMidStream covers sendEvent's non-blocking
// path: a full buffered channel plus an already-cancelled context must make
// readStream return cleanly instead of hanging or panicking.
func TestReadStreamContextCancelledMidStream(t *testing.T) {
	ch := make(chan ports.ChatEvent, 1)
	ch <- ports.ChatEvent{Kind: "filler"} // fills the buffer so sendEvent can't send immediately

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	sse := `data: {"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":"hello"}}
data: {"type":"message_stop"}
`
	c := &ChatAdapter{}
	body := io.NopCloser(strings.NewReader(sse))

	done := make(chan struct{})
	go func() {
		c.readStream(ctx, body, ch)
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("readStream did not return after context cancellation; sendEvent may be blocking")
	}

	first := <-ch
	if first.Kind != "filler" {
		t.Fatalf("first buffered event = %+v, want filler", first)
	}
	if _, open := <-ch; open {
		t.Fatal("expected channel closed with no further events after cancelled sendEvent")
	}
}
