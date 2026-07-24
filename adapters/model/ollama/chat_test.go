package ollama

import (
	"context"
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

const kindTextDelta = "text-delta"

// drainReadStream feeds ndjson into readStream on a real channel and collects
// every event until the channel closes.
func drainReadStream(ctx context.Context, ndjson string) []ports.ChatEvent {
	c := &ChatAdapter{}
	return ssetest.Drain(ctx, io.NopCloser(strings.NewReader(ndjson)), c.readStream)
}

func TestExtractArgsOllama(t *testing.T) {
	// Unlike openai/anthropic's extractArgs, an empty "args" value falls
	// back to the raw JSON blob here (see the ponytail comment on
	// extractArgsOllama) — pinning existing behavior, not asserting it's
	// correct.
	if got := extractArgsOllama([]byte(`{"args": ""}`)); got != `{"args": ""}` {
		t.Errorf("extractArgsOllama(empty args) = %q, want raw fallback", got)
	}
	if got := extractArgsOllama([]byte(`{"args": "grilled chicken"}`)); got != "grilled chicken" {
		t.Errorf("extractArgsOllama = %q, want %q", got, "grilled chicken")
	}
	if got := extractArgsOllama([]byte(`not json`)); got != "not json" {
		t.Errorf("extractArgsOllama(invalid json) = %q, want raw fallback", got)
	}
}

func TestStreamChatHTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
	}))
	defer srv.Close()

	c := NewChatAdapter(srv.URL, "llama3.1", 5*time.Second)
	_, err := c.StreamChat(t.Context(), ports.ChatRequest{Messages: []ports.ChatMessage{{Role: "user", Content: "hi"}}})
	if err == nil {
		t.Fatal("expected error on 502, got nil")
	}
	if !strings.Contains(err.Error(), "502") {
		t.Errorf("error = %q, want it to mention the status code", err.Error())
	}
}

// TestReadStreamTextDeltaAccumulation covers several content deltas arriving
// in sequence, followed by the done chunk.
func TestReadStreamTextDeltaAccumulation(t *testing.T) {
	ndjson := `{"message":{"role":"assistant","content":"Hello "}}
{"message":{"role":"assistant","content":"world"}}
{"message":{"role":"assistant","content":""},"done":true}
`
	events := drainReadStream(t.Context(), ndjson)

	ssetest.AssertEvents(t, events, []ports.ChatEvent{
		{Kind: kindTextDelta, Text: "Hello "},
		{Kind: kindTextDelta, Text: "world"},
		{Kind: "done"},
	})
}

// TestReadStreamToolCall covers Ollama delivering a tool call complete in a
// single chunk (no incremental accumulation, unlike OpenAI).
func TestReadStreamToolCall(t *testing.T) {
	ndjson := `{"message":{"role":"assistant","tool_calls":[{"function":{"name":"log_meal","arguments":{"args":"200g frango"}}}]},"done":true}
`
	events := drainReadStream(t.Context(), ndjson)

	if len(events) != 2 {
		t.Fatalf("got %d events, want 2 (tool-call, done): %+v", len(events), events)
	}
	want := ports.ToolCallEvent{ID: "log_meal", Name: "log_meal", Args: "200g frango"}
	if events[0].Kind != "tool-call" || events[0].ToolCall == nil || *events[0].ToolCall != want {
		t.Errorf("event[0] = %+v, want tool-call %+v", events[0], want)
	}
	if events[1].Kind != "done" {
		t.Errorf("event[1].Kind = %q, want done", events[1].Kind)
	}
}

// TestReadStreamTextAndToolCallSameChunk covers a single chunk carrying both
// content and tool_calls — the two are not mutually exclusive per the
// emitChunk doc comment.
func TestReadStreamTextAndToolCallSameChunk(t *testing.T) {
	ndjson := `{"message":{"role":"assistant","content":"noted","tool_calls":[{"function":{"name":"log_meal","arguments":{"args":"2 ovos"}}}]},"done":true}
`
	events := drainReadStream(t.Context(), ndjson)

	if len(events) != 3 {
		t.Fatalf("got %d events, want 3 (text-delta, tool-call, done): %+v", len(events), events)
	}
	if events[0].Kind != kindTextDelta || events[0].Text != "noted" {
		t.Errorf("event[0] = %+v, want text-delta %q", events[0], "noted")
	}
	if events[1].Kind != "tool-call" {
		t.Errorf("event[1].Kind = %q, want tool-call", events[1].Kind)
	}
	if events[2].Kind != "done" {
		t.Errorf("event[2].Kind = %q, want done", events[2].Kind)
	}
}

// TestReadStreamEmptyAndMalformedLinesSkipped covers blank lines and invalid
// JSON being skipped, with subsequent valid lines still processed.
func TestReadStreamEmptyAndMalformedLinesSkipped(t *testing.T) {
	ndjson := "\n" + `not valid json at all
{"message":{"role":"assistant","content":"ok"},"done":true}
`
	events := drainReadStream(t.Context(), ndjson)

	ssetest.AssertEvents(t, events, []ports.ChatEvent{
		{Kind: kindTextDelta, Text: "ok"},
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
