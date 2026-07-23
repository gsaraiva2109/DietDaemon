package openai

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gsaraiva2109/dietdaemon/core/ports"
)

// drainReadStream feeds sse into readStream on a real channel and collects
// every event until the channel closes.
func drainReadStream(ctx context.Context, sse string) []ports.ChatEvent {
	c := &ChatAdapter{}
	ch := make(chan ports.ChatEvent, 20)
	c.readStream(ctx, io.NopCloser(strings.NewReader(sse)), ch)

	var events []ports.ChatEvent
	for e := range ch {
		events = append(events, e)
	}
	return events
}

// errReader is an io.ReadCloser that always fails, for exercising the
// scanner-error branch of readStream.
type errReader struct{ err error }

func (r *errReader) Read([]byte) (int, error) { return 0, r.err }
func (r *errReader) Close() error             { return nil }

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
		_, _ = w.Write([]byte(`{"error":{"message":"model does not support tools"}}`))
	}))
	defer srv.Close()

	c := NewChatAdapter(srv.URL, "sk-test", "deepseek-chat", 5*time.Second)
	_, err := c.StreamChat(t.Context(), ports.ChatRequest{Messages: []ports.ChatMessage{{Role: "user", Content: "hi"}}})
	if err == nil {
		t.Fatal("expected error on 400, got nil")
	}
	if !strings.Contains(err.Error(), "model does not support tools") {
		t.Errorf("error = %q, want it to include the response body detail", err.Error())
	}
}

// TestReadStreamTextDeltaAccumulation covers several content deltas arriving
// in sequence, followed by the [DONE] sentinel.
func TestReadStreamTextDeltaAccumulation(t *testing.T) {
	sse := `data: {"choices":[{"index":0,"delta":{"content":"Hello "}}]}
data: {"choices":[{"index":0,"delta":{"content":"world"}}]}
data: [DONE]
`
	events := drainReadStream(t.Context(), sse)

	want := []ports.ChatEvent{
		{Kind: "text-delta", Text: "Hello "},
		{Kind: "text-delta", Text: "world"},
		{Kind: "done"},
	}
	if len(events) != len(want) {
		t.Fatalf("got %d events, want %d: %+v", len(events), len(want), events)
	}
	for i, w := range want {
		if events[i].Kind != w.Kind || events[i].Text != w.Text {
			t.Errorf("event[%d] = %+v, want %+v", i, events[i], w)
		}
	}
}

// TestReadStreamToolCallMultiChunkAccumulation covers OpenAI's incremental
// tool-call reconstruction: the first chunk carries id+name at an index,
// later chunks carry only Function.Arguments fragments at that same index,
// accumulated in the pending map until finish_reason flushes it.
func TestReadStreamToolCallMultiChunkAccumulation(t *testing.T) {
	sse := `data: {"choices":[{"index":0,"delta":{"tool_calls":[{"index":0,"id":"call_1","type":"function","function":{"name":"suggest_meal","arguments":""}}]}}]}
data: {"choices":[{"index":0,"delta":{"tool_calls":[{"index":0,"function":{"arguments":"{\"args\":\""}}]}}]}
data: {"choices":[{"index":0,"delta":{"tool_calls":[{"index":0,"function":{"arguments":"grilled chicken\"}"}}]}}]}
data: {"choices":[{"index":0,"finish_reason":"tool_calls"}]}
`
	events := drainReadStream(t.Context(), sse)

	if len(events) != 1 {
		t.Fatalf("got %d events, want 1 tool-call event: %+v", len(events), events)
	}
	if events[0].Kind != "tool-call" || events[0].ToolCall == nil {
		t.Fatalf("event[0] = %+v, want a tool-call event", events[0])
	}
	want := ports.ToolCallEvent{ID: "call_1", Name: "suggest_meal", Args: "grilled chicken"}
	if *events[0].ToolCall != want {
		t.Errorf("ToolCall = %+v, want %+v", *events[0].ToolCall, want)
	}
}

// TestReadStreamDoneFlushesPendingToolCalls covers the [DONE] sentinel
// flushing any still-pending tool call before emitting done.
func TestReadStreamDoneFlushesPendingToolCalls(t *testing.T) {
	sse := `data: {"choices":[{"index":0,"delta":{"tool_calls":[{"index":0,"id":"call_2","type":"function","function":{"name":"log_meal","arguments":"{\"args\":\"eggs\"}"}}]}}]}
data: [DONE]
`
	events := drainReadStream(t.Context(), sse)

	if len(events) != 2 {
		t.Fatalf("got %d events, want 2 (tool-call, done): %+v", len(events), events)
	}
	if events[0].Kind != "tool-call" || events[0].ToolCall == nil {
		t.Fatalf("event[0] = %+v, want a tool-call event", events[0])
	}
	want := ports.ToolCallEvent{ID: "call_2", Name: "log_meal", Args: "eggs"}
	if *events[0].ToolCall != want {
		t.Errorf("ToolCall = %+v, want %+v", *events[0].ToolCall, want)
	}
	if events[1].Kind != "done" {
		t.Errorf("event[1].Kind = %q, want done", events[1].Kind)
	}
}

// TestReadStreamFinishReasonFlushesPending covers every terminal
// finish_reason value triggering a flush of pending tool calls.
func TestReadStreamFinishReasonFlushesPending(t *testing.T) {
	for _, reason := range []string{"stop", "tool_calls", "length", "content_filter"} {
		t.Run(reason, func(t *testing.T) {
			sse := `data: {"choices":[{"index":0,"delta":{"tool_calls":[{"index":0,"id":"call_x","type":"function","function":{"name":"fn","arguments":"{\"args\":\"v\"}"}}]}}]}
data: {"choices":[{"index":0,"finish_reason":"` + reason + `"}]}
`
			events := drainReadStream(t.Context(), sse)

			if len(events) != 1 {
				t.Fatalf("got %d events, want 1 flushed tool-call event: %+v", len(events), events)
			}
			want := ports.ToolCallEvent{ID: "call_x", Name: "fn", Args: "v"}
			if events[0].Kind != "tool-call" || events[0].ToolCall == nil || *events[0].ToolCall != want {
				t.Errorf("event[0] = %+v, want tool-call %+v", events[0], want)
			}
		})
	}
}

// TestReadStreamFinishReasonResetsPendingBetweenRounds covers the pending
// map being reset after a finish_reason flush, so a second round of tool
// calls at the same index doesn't inherit the first round's accumulated args.
func TestReadStreamFinishReasonResetsPendingBetweenRounds(t *testing.T) {
	sse := `data: {"choices":[{"index":0,"delta":{"tool_calls":[{"index":0,"id":"call_a","type":"function","function":{"name":"a_fn","arguments":"{\"args\":\"A"}}]}}]}
data: {"choices":[{"index":0,"delta":{"tool_calls":[{"index":0,"function":{"arguments":"1\"}"}}]}}]}
data: {"choices":[{"index":0,"finish_reason":"tool_calls"}]}
data: {"choices":[{"index":0,"delta":{"tool_calls":[{"index":0,"id":"call_b","type":"function","function":{"name":"b_fn","arguments":"{\"args\":\"B1\"}"}}]}}]}
data: {"choices":[{"index":0,"finish_reason":"stop"}]}
`
	events := drainReadStream(t.Context(), sse)

	if len(events) != 2 {
		t.Fatalf("got %d events, want 2 tool-call events: %+v", len(events), events)
	}
	wantA := ports.ToolCallEvent{ID: "call_a", Name: "a_fn", Args: "A1"}
	wantB := ports.ToolCallEvent{ID: "call_b", Name: "b_fn", Args: "B1"}
	if events[0].ToolCall == nil || *events[0].ToolCall != wantA {
		t.Errorf("event[0].ToolCall = %+v, want %+v", events[0].ToolCall, wantA)
	}
	if events[1].ToolCall == nil || *events[1].ToolCall != wantB {
		t.Errorf("event[1].ToolCall = %+v, want %+v (must not bleed round 1's args)", events[1].ToolCall, wantB)
	}
}

// TestReadStreamMalformedAndEmptyChoicesSkipped covers malformed JSON and
// empty-choices lines being skipped, with subsequent valid lines still
// processed.
func TestReadStreamMalformedAndEmptyChoicesSkipped(t *testing.T) {
	sse := `data: not valid json at all
data: {"choices":[]}
data: {"choices":[{"index":0,"delta":{"content":"ok"}}]}
data: [DONE]
`
	events := drainReadStream(t.Context(), sse)

	want := []ports.ChatEvent{
		{Kind: "text-delta", Text: "ok"},
		{Kind: "done"},
	}
	if len(events) != len(want) {
		t.Fatalf("got %d events, want %d: %+v", len(events), len(want), events)
	}
	for i, w := range want {
		if events[i].Kind != w.Kind || events[i].Text != w.Text {
			t.Errorf("event[%d] = %+v, want %+v", i, events[i], w)
		}
	}
}

// TestReadStreamScannerReadError covers the scanner erroring mid-read: it
// must emit a single error event carrying the underlying error.
func TestReadStreamScannerReadError(t *testing.T) {
	c := &ChatAdapter{}
	ch := make(chan ports.ChatEvent, 5)
	c.readStream(t.Context(), &errReader{err: errors.New("boom")}, ch)

	var events []ports.ChatEvent
	for e := range ch {
		events = append(events, e)
	}

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
