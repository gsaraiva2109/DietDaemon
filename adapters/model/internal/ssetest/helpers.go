// Package ssetest holds shared test scaffolding for the SSE-based chat
// adapters (anthropic, openai): draining a readStream implementation into a
// slice of events, a failing io.ReadCloser for exercising scanner-error
// branches, and an assertion helper for the common "sequence of Kind/Text
// events" check used by their characterization tests.
package ssetest

import (
	"context"
	"io"
	"testing"

	"github.com/gsaraiva2109/dietdaemon/core/ports"
)

// Drain feeds body into readStream on a buffered channel and collects every
// event until the channel closes.
func Drain(ctx context.Context, body io.ReadCloser, readStream func(context.Context, io.ReadCloser, chan<- ports.ChatEvent)) []ports.ChatEvent {
	ch := make(chan ports.ChatEvent, 20)
	readStream(ctx, body, ch)

	var events []ports.ChatEvent
	for e := range ch {
		events = append(events, e)
	}
	return events
}

// ErrReader is an io.ReadCloser that always fails, for exercising the
// scanner-error branch of readStream.
type ErrReader struct{ Err error }

func (r *ErrReader) Read([]byte) (int, error) { return 0, r.Err }
func (r *ErrReader) Close() error             { return nil }

// AssertEvents checks that got matches want by Kind and Text, in order.
func AssertEvents(t *testing.T, got, want []ports.ChatEvent) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("got %d events, want %d: %+v", len(got), len(want), got)
	}
	for i, w := range want {
		if got[i].Kind != w.Kind || got[i].Text != w.Text {
			t.Errorf("event[%d] = %+v, want %+v", i, got[i], w)
		}
	}
}
