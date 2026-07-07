package store

import (
	"testing"

	"github.com/gsaraiva2109/dietdaemon/core/types"
)

// TestChatMessageOwnershipScoping guards the IDOR where a session ID alone
// (a predictable timestamp-based ID, see newHandlerID in internal/api) let
// any authenticated user read or write another user's chat history.
// AppendChatMessage/GetChatMessages must scope by user_id like every other
// by-ID store method in this codebase (see RevokeAPIKey for the pattern).
func TestChatMessageOwnershipScoping(t *testing.T) {
	s, cleanup := tempDB(t)
	defer cleanup()

	owner, intruder := "u-owner", "u-intruder"
	mustUser(t, s, types.User{ID: owner})
	mustUser(t, s, types.User{ID: intruder})

	if err := s.CreateChatSession(ctx(), "sess-1", owner, "my session"); err != nil {
		t.Fatalf("CreateChatSession: %v", err)
	}

	// Intruder tries to write into the owner's session.
	err := s.AppendChatMessage(ctx(), "m-1", intruder, "sess-1", "user", "not my session", "")
	if err != types.ErrNotFound {
		t.Fatalf("AppendChatMessage by intruder: got %v, want ErrNotFound", err)
	}

	// Owner writes fine.
	if err := s.AppendChatMessage(ctx(), "m-2", owner, "sess-1", "user", "hi", ""); err != nil {
		t.Fatalf("AppendChatMessage by owner: %v", err)
	}

	// Intruder reads the owner's session: empty, not the owner's messages.
	msgs, err := s.GetChatMessages(ctx(), intruder, "sess-1")
	if err != nil {
		t.Fatalf("GetChatMessages by intruder: %v", err)
	}
	if len(msgs) != 0 {
		t.Fatalf("GetChatMessages by intruder returned %d messages, want 0", len(msgs))
	}

	// Owner reads and sees their own message.
	msgs, err = s.GetChatMessages(ctx(), owner, "sess-1")
	if err != nil {
		t.Fatalf("GetChatMessages by owner: %v", err)
	}
	if len(msgs) != 1 || msgs[0].Content != "hi" {
		t.Fatalf("GetChatMessages by owner = %+v, want one message 'hi'", msgs)
	}
}
