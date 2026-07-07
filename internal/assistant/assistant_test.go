package assistant

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"

	"github.com/gsaraiva2109/dietdaemon/core/ports"
	"github.com/gsaraiva2109/dietdaemon/core/types"
)

// ---------------------------------------------------------------------------
// Test doubles
// ---------------------------------------------------------------------------

// fakeChatAdapter returns pre-programmed event sequences per round, and
// records the ChatRequest it was called with each round so tests can assert
// on what history the router seeded.
type fakeChatAdapter struct {
	mu     sync.Mutex
	rounds [][]ports.ChatEvent
	index  int
	reqs   []ports.ChatRequest
}

func (f *fakeChatAdapter) StreamChat(_ context.Context, req ports.ChatRequest) (<-chan ports.ChatEvent, error) {
	f.mu.Lock()
	f.reqs = append(f.reqs, req)
	if f.index >= len(f.rounds) {
		f.mu.Unlock()
		ch := make(chan ports.ChatEvent)
		close(ch)
		return ch, nil
	}
	events := f.rounds[f.index]
	f.index++
	f.mu.Unlock()

	ch := make(chan ports.ChatEvent, len(events))
	for _, e := range events {
		ch <- e
	}
	close(ch)
	return ch, nil
}

// errChatAdapter returns a fixed error from StreamChat.
type errChatAdapter struct{ err error }

func (e *errChatAdapter) StreamChat(_ context.Context, _ ports.ChatRequest) (<-chan ports.ChatEvent, error) {
	return nil, e.err
}

// blockingChatAdapter sends its programmed events then blocks until ctx is
// cancelled. If ready is non-nil it is closed just before blocking so the test
// can synchronise on "events sent, adapter now blocking".
type blockingChatAdapter struct {
	events []ports.ChatEvent
	ready  chan struct{}
}

func (b *blockingChatAdapter) StreamChat(ctx context.Context, _ ports.ChatRequest) (<-chan ports.ChatEvent, error) {
	ch := make(chan ports.ChatEvent)
	go func() {
		defer close(ch)
		for _, e := range b.events {
			select {
			case ch <- e:
			case <-ctx.Done():
				return
			}
		}
		if b.ready != nil {
			close(b.ready)
		}
		<-ctx.Done()
	}()
	return ch, nil
}

// fakeCommand is a minimal Command stub for tests.
type fakeCommand struct {
	name   string
	result string
}

func (f *fakeCommand) Name() string        { return f.name }
func (f *fakeCommand) Aliases() []string   { return nil }
func (f *fakeCommand) Help() types.I18nKey { return "" }
func (f *fakeCommand) Handle(_ context.Context, _ types.InboundMessage, _ string) (types.Reply, error) {
	return types.Reply{Text: f.result}, nil
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func collectEvents(ch <-chan ports.ChatEvent) []ports.ChatEvent {
	var out []ports.ChatEvent
	for e := range ch {
		out = append(out, e)
	}
	return out
}

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

func TestRouterTextOnly(t *testing.T) {
	adapter := &fakeChatAdapter{
		rounds: [][]ports.ChatEvent{
			{
				{Kind: "text-delta", Text: "Hello!"},
				{Kind: "done"},
			},
		},
	}
	r := New(adapter, nil, nil)
	ch := r.Run(context.Background(), "u1", "system", nil, "hi")

	events := collectEvents(ch)
	if len(events) != 2 {
		t.Fatalf("got %d events, want 2", len(events))
	}
	if events[0].Kind != "text-delta" || events[0].Text != "Hello!" {
		t.Errorf("events[0] = %+v, want text-delta 'Hello!'", events[0])
	}
	if events[1].Kind != "done" {
		t.Errorf("events[1].Kind = %q, want done", events[1].Kind)
	}
	// ponytail: one check covers all — no tool events should appear.
	for _, e := range events {
		if e.Kind == "tool-call" || e.Kind == "tool-result" {
			t.Errorf("unexpected tool event: %+v", e)
		}
	}
}

func TestRouterTextOnly_doneForwarded(t *testing.T) {
	// Separate test to explicitly verify done is forwarded in the no-tool path.
	adapter := &fakeChatAdapter{
		rounds: [][]ports.ChatEvent{
			{
				{Kind: "text-delta", Text: "first"},
				{Kind: "text-delta", Text: "second"},
				{Kind: "done"},
			},
		},
	}
	r := New(adapter, nil, nil)
	ch := r.Run(context.Background(), "u1", "system", nil, "hello")

	events := collectEvents(ch)
	if len(events) != 3 {
		t.Fatalf("got %d events, want 3", len(events))
	}
	last := events[len(events)-1]
	if last.Kind != "done" {
		t.Errorf("last event.Kind = %q, want done", last.Kind)
	}
}

// TestRouterSeedsHistory guards the bug where handleChatMessage never loaded
// prior session turns, so every message after the first in a session was
// answered with zero memory of the conversation so far. Run must place the
// given history ahead of the new user message in the first request sent to
// the adapter.
func TestRouterSeedsHistory(t *testing.T) {
	adapter := &fakeChatAdapter{
		rounds: [][]ports.ChatEvent{
			{
				{Kind: "text-delta", Text: "sure, oatmeal again"},
				{Kind: "done"},
			},
		},
	}
	history := []ports.ChatMessage{
		{Role: "user", Content: "what should I eat for breakfast"},
		{Role: "assistant", Content: "how about oatmeal?"},
	}
	r := New(adapter, nil, nil)
	collectEvents(r.Run(context.Background(), "u1", "system", history, "same as yesterday"))

	if len(adapter.reqs) != 1 {
		t.Fatalf("got %d adapter calls, want 1", len(adapter.reqs))
	}
	got := adapter.reqs[0].Messages
	want := []ports.ChatMessage{
		{Role: "user", Content: "what should I eat for breakfast"},
		{Role: "assistant", Content: "how about oatmeal?"},
		{Role: "user", Content: "same as yesterday"},
	}
	if len(got) != len(want) {
		t.Fatalf("got %d messages, want %d: %+v", len(got), len(want), got)
	}
	for i := range want {
		if got[i].Role != want[i].Role || got[i].Content != want[i].Content {
			t.Errorf("messages[%d] = %+v, want %+v", i, got[i], want[i])
		}
	}
}

func TestRouterToolCallSingle(t *testing.T) {
	adapter := &fakeChatAdapter{
		rounds: [][]ports.ChatEvent{
			{
				{Kind: "text-delta", Text: "Checking..."},
				{Kind: "tool-call", ToolCall: &ports.ToolCallEvent{ID: "c1", Name: "/search", Args: "diet"}},
				{Kind: "done"},
			},
			{
				{Kind: "text-delta", Text: "Here's the answer"},
				{Kind: "done"},
			},
		},
	}
	cmd := &fakeCommand{name: "/search", result: "found it"}
	r := New(adapter, []ports.Command{cmd}, map[string]string{"/search": "Search tool"})
	ch := r.Run(context.Background(), "u1", "system", nil, "search something")

	events := collectEvents(ch)
	if len(events) != 5 {
		t.Fatalf("got %d events, want 5", len(events))
	}

	// (0) first text-delta
	e := events[0]
	if e.Kind != "text-delta" || e.Text != "Checking..." {
		t.Errorf("events[0] = %+v, want text-delta 'Checking...'", e)
	}

	// (1) tool-call forwarded
	e = events[1]
	if e.Kind != "tool-call" {
		t.Fatalf("events[1].Kind = %q, want tool-call", e.Kind)
	}
	if e.ToolCall == nil || e.ToolCall.ID != "c1" || e.ToolCall.Name != "/search" || e.ToolCall.Args != "diet" {
		t.Errorf("events[1].ToolCall = %+v, want {c1 /search diet}", e.ToolCall)
	}

	// (2) tool-result emitted
	e = events[2]
	if e.Kind != "tool-result" {
		t.Fatalf("events[2].Kind = %q, want tool-result", e.Kind)
	}
	if e.ToolCall == nil || e.ToolCall.ID != "c1" || e.ToolCall.Name != "/search" {
		t.Errorf("events[2].ToolCall = %+v, want {c1 /search ...}", e.ToolCall)
	}
	if e.ToolCall.Args != "found it" {
		t.Errorf("events[2].ToolCall.Args = %q, want 'found it'", e.ToolCall.Args)
	}

	// (3) second text-delta
	e = events[3]
	if e.Kind != "text-delta" || e.Text != "Here's the answer" {
		t.Errorf("events[3] = %+v, want text-delta 'Here's the answer'", e)
	}

	// (4) done
	e = events[4]
	if e.Kind != "done" {
		t.Errorf("events[4].Kind = %q, want done", e.Kind)
	}
}

func TestRouterToolCallMaxRounds(t *testing.T) {
	rounds := make([][]ports.ChatEvent, 6)
	for i := range rounds {
		rounds[i] = []ports.ChatEvent{
			{Kind: "tool-call", ToolCall: &ports.ToolCallEvent{ID: "c", Name: "/search", Args: "x"}},
			{Kind: "done"},
		}
	}
	adapter := &fakeChatAdapter{rounds: rounds}
	cmd := &fakeCommand{name: "/search", result: "ok"}
	r := New(adapter, []ports.Command{cmd}, map[string]string{"/search": "Search"})
	ch := r.Run(context.Background(), "u1", "system", nil, "loop")

	events := collectEvents(ch)
	// 6 rounds * (tool-call + tool-result) + 1 error = 13
	if len(events) != 13 {
		t.Fatalf("got %d events, want 13", len(events))
	}

	// First 12 events alternate tool-call / tool-result.
	for i := 0; i < 12; i += 2 {
		if events[i].Kind != "tool-call" {
			t.Errorf("events[%d].Kind = %q, want tool-call", i, events[i].Kind)
		}
		if events[i+1].Kind != "tool-result" {
			t.Errorf("events[%d].Kind = %q, want tool-result", i+1, events[i+1].Kind)
		}
	}

	// Last event is the "over max rounds" error.
	last := events[12]
	if last.Kind != "error" {
		t.Fatalf("events[12].Kind = %q, want error", last.Kind)
	}
	if last.Err == nil || last.Err.Error() != suggestFallback {
		t.Errorf("events[12].Err = %v, want '%s'", last.Err, suggestFallback)
	}
}

func TestRouterErrorPropagation(t *testing.T) {
	r := New(&errChatAdapter{err: errors.New("connection failed")}, nil, nil)
	ch := r.Run(context.Background(), "u1", "system", nil, "hi")

	events := collectEvents(ch)
	if len(events) != 1 {
		t.Fatalf("got %d events, want 1", len(events))
	}
	e := events[0]
	if e.Kind != "error" {
		t.Fatalf("events[0].Kind = %q, want error", e.Kind)
	}
	if e.Err == nil {
		t.Fatal("events[0].Err is nil")
	}
	if !strings.Contains(e.Err.Error(), "assistant: stream: connection failed") {
		t.Errorf("events[0].Err = %v, want wrapping 'assistant: stream: connection failed'", e.Err)
	}
}

func TestRouterMidStreamError(t *testing.T) {
	adapter := &fakeChatAdapter{
		rounds: [][]ports.ChatEvent{
			{
				{Kind: "text-delta", Text: "first"},
				{Kind: "error", Err: errors.New("timeout")},
			},
		},
	}
	r := New(adapter, nil, nil)
	ch := r.Run(context.Background(), "u1", "system", nil, "hi")

	events := collectEvents(ch)
	if len(events) != 2 {
		t.Fatalf("got %d events, want 2", len(events))
	}
	if events[0].Kind != "text-delta" || events[0].Text != "first" {
		t.Errorf("events[0] = %+v, want text-delta 'first'", events[0])
	}
	e := events[1]
	if e.Kind != "error" {
		t.Fatalf("events[1].Kind = %q, want error", e.Kind)
	}
	if e.Err == nil || e.Err.Error() != "timeout" {
		t.Errorf("events[1].Err = %v, want 'timeout'", e.Err)
	}
}

func TestRouterUnknownCommand(t *testing.T) {
	adapter := &fakeChatAdapter{
		rounds: [][]ports.ChatEvent{
			{
				{Kind: "tool-call", ToolCall: &ports.ToolCallEvent{ID: "t1", Name: "nonexistent", Args: "blah"}},
				{Kind: "done"},
			},
			{
				{Kind: "done"},
			},
		},
	}
	// No commands registered — unknown command path triggers.
	r := New(adapter, nil, nil)
	ch := r.Run(context.Background(), "u1", "system", nil, "do something")

	events := collectEvents(ch)
	if len(events) != 3 {
		t.Fatalf("got %d events, want 3", len(events))
	}
	// tool-call
	if events[0].Kind != "tool-call" {
		t.Errorf("events[0].Kind = %q, want tool-call", events[0].Kind)
	}
	// tool-result with "unknown command" message
	if events[1].Kind != "tool-result" {
		t.Fatalf("events[1].Kind = %q, want tool-result", events[1].Kind)
	}
	if events[1].ToolCall == nil {
		t.Fatal("events[1].ToolCall is nil")
	}
	if events[1].ToolCall.Name != "nonexistent" {
		t.Errorf("events[1].ToolCall.Name = %q, want 'nonexistent'", events[1].ToolCall.Name)
	}
	if !strings.Contains(events[1].ToolCall.Args, "unknown command: nonexistent") {
		t.Errorf("events[1].ToolCall.Args = %q, want containing 'unknown command: nonexistent'", events[1].ToolCall.Args)
	}
	// done
	if events[2].Kind != "done" {
		t.Errorf("events[2].Kind = %q, want done", events[2].Kind)
	}
}

func TestRouterContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	ready := make(chan struct{})
	adapter := &blockingChatAdapter{
		events: []ports.ChatEvent{
			{Kind: "text-delta", Text: "partial"},
		},
		ready: ready,
	}
	r := New(adapter, nil, nil)
	ch := r.Run(ctx, "u1", "system", nil, "hello")

	// Wait for the adapter to have sent its event and block on ctx.Done().
	<-ready
	cancel()

	events := collectEvents(ch)
	if len(events) != 1 {
		t.Fatalf("got %d events, want 1 — expected one text-delta before cancellation", len(events))
	}
	if events[0].Kind != "text-delta" || events[0].Text != "partial" {
		t.Errorf("events[0] = %+v, want text-delta 'partial'", events[0])
	}
}
