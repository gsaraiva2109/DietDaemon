package discord

import (
	"testing"

	"github.com/gsaraiva2109/dietdaemon/core/types"
)

func TestName(t *testing.T) {
	a := New("token")
	if a.Name() != "discord" {
		t.Errorf("Name = %q, want %q", a.Name(), "discord")
	}
}

func TestSendMissingChannel(t *testing.T) {
	a := New("token")
	err := a.Send(t.Context(), types.Reply{Text: "hello", ChannelMeta: nil})
	if err == nil {
		t.Error("expected error for missing channel_id, got nil")
	}
}
