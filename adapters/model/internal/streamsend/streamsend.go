// Package streamsend provides the send-to-a-possibly-full-or-abandoned-
// channel helper shared by the streaming chat adapters (anthropic, openai,
// ollama).
package streamsend

import (
	"context"
	"encoding/json"

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

// ExtractArgs returns the args field from raw tool-call JSON, or raw when it
// is incomplete or invalid.
func ExtractArgs(raw string) string {
	var obj struct {
		Args string `json:"args"`
	}
	if err := json.Unmarshal([]byte(raw), &obj); err == nil {
		return obj.Args
	}
	return raw
}
