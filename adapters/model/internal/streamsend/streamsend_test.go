package streamsend

import (
	"context"
	"testing"
	"time"

	"github.com/gsaraiva2109/dietdaemon/core/ports"
)

func TestSendDeliversImmediately(t *testing.T) {
	ch := make(chan ports.ChatEvent, 1)
	if !Send(context.Background(), ch, ports.ChatEvent{Kind: "done"}) {
		t.Fatal("expected Send to succeed on a buffered channel with room")
	}
	if got := <-ch; got.Kind != "done" {
		t.Errorf("got %+v, want Kind=done", got)
	}
}

func TestSendBlocksThenDeliversWhenReaderCatchesUp(t *testing.T) {
	ch := make(chan ports.ChatEvent) // unbuffered: first select's default fires
	done := make(chan bool, 1)
	go func() { done <- Send(context.Background(), ch, ports.ChatEvent{Kind: "text-delta"}) }()

	select {
	case got := <-ch:
		if got.Kind != "text-delta" {
			t.Errorf("got %+v, want Kind=text-delta", got)
		}
	case <-time.After(time.Second):
		t.Fatal("Send never delivered on the second (blocking) select")
	}
	if ok := <-done; !ok {
		t.Error("expected Send to report success")
	}
}

func TestSendBailsOnCancelledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	ch := make(chan ports.ChatEvent) // unbuffered, no reader: both selects miss
	if Send(ctx, ch, ports.ChatEvent{Kind: "done"}) {
		t.Error("expected Send to bail when ctx is already cancelled and no reader is ready")
	}
}
