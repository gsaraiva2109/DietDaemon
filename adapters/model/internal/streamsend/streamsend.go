// Package streamsend provides the send-to-a-possibly-full-or-abandoned-
// channel helper shared by the streaming chat adapters (anthropic, openai,
// ollama).
package streamsend

import (
	"context"

	"github.com/gsaraiva2109/dietdaemon/core/ports"
)

// Send delivers evt to ch, or bails if ctx is cancelled first — without
// this, a client that disconnects mid-stream while the channel's buffer is
// full leaks the reader goroutine (and its open upstream connection) forever.
func Send(ctx context.Context, ch chan<- ports.ChatEvent, evt ports.ChatEvent) bool {
	select {
	case ch <- evt:
		return true
	default:
	}
	select {
	case ch <- evt:
		return true
	case <-ctx.Done():
		return false
	}
}
