package matrix

import (
	"testing"

	"github.com/gsaraiva2109/dietdaemon/core/types"
)

func TestName(t *testing.T) {
	a := New("https://matrix.example.com", "@bot:example.com", "token")
	if a.Name() != "matrix" {
		t.Errorf("Name = %q, want %q", a.Name(), "matrix")
	}
}

func TestSendMissingRoom(t *testing.T) {
	a := New("https://matrix.example.com", "@bot:example.com", "token")
	err := a.Send(t.Context(), types.Reply{Text: "hello", ChannelMeta: nil})
	if err == nil {
		t.Error("expected error for missing room_id, got nil")
	}
}
