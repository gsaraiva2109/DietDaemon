package store

import (
	"fmt"
	"testing"
	"time"

	"github.com/gsaraiva2109/dietdaemon/core/types"
)

func TestGetChatMessagesCapsHistoryAndPreservesOrder(t *testing.T) {
	s, cleanup := tempDB(t)
	defer cleanup()

	mustUser(t, s, types.User{ID: "u-history"})
	if err := s.CreateChatSession(ctx(), "sess-history", "u-history", "history"); err != nil {
		t.Fatalf("CreateChatSession: %v", err)
	}
	for i := 0; i < chatHistoryLimit+2; i++ {
		id := fmt.Sprintf("msg-%03d", i)
		if err := s.AppendChatMessage(ctx(), id, "u-history", "sess-history", "user", id, ""); err != nil {
			t.Fatalf("AppendChatMessage %d: %v", i, err)
		}
	}

	msgs, err := s.GetChatMessages(ctx(), "u-history", "sess-history")
	if err != nil {
		t.Fatalf("GetChatMessages: %v", err)
	}
	if len(msgs) != chatHistoryLimit {
		t.Fatalf("message count = %d, want %d", len(msgs), chatHistoryLimit)
	}
	if msgs[0].ID != "msg-002" || msgs[len(msgs)-1].ID != fmt.Sprintf("msg-%03d", chatHistoryLimit+1) {
		t.Fatalf("history range = %q..%q, want msg-002..msg-%03d", msgs[0].ID, msgs[len(msgs)-1].ID, chatHistoryLimit+1)
	}
}

func TestSoftDeleteChatSession(t *testing.T) {
	s, cleanup := tempDB(t)
	defer cleanup()

	mustUser(t, s, types.User{ID: "u-softdel"})
	if err := s.CreateChatSession(ctx(), "sess-sd", "u-softdel", "my session"); err != nil {
		t.Fatalf("CreateChatSession: %v", err)
	}

	// Soft-delete the session.
	if err := s.SoftDeleteChatSession(ctx(), "u-softdel", "sess-sd"); err != nil {
		t.Fatalf("SoftDeleteChatSession: %v", err)
	}

	// Should no longer appear in active list.
	active, err := s.ListChatSessions(ctx(), "u-softdel")
	if err != nil {
		t.Fatalf("ListChatSessions: %v", err)
	}
	if len(active) != 0 {
		t.Fatalf("expected 0 active sessions, got %d", len(active))
	}

	// Should appear in deleted list.
	deleted, err := s.ListDeletedChatSessions(ctx(), "u-softdel")
	if err != nil {
		t.Fatalf("ListDeletedChatSessions: %v", err)
	}
	if len(deleted) != 1 {
		t.Fatalf("expected 1 deleted session, got %d", len(deleted))
	}
	if deleted[0].ID != "sess-sd" {
		t.Errorf("deleted session ID = %q, want sess-sd", deleted[0].ID)
	}
}

func TestSoftDeleteChatSessionAlreadyDeleted(t *testing.T) {
	s, cleanup := tempDB(t)
	defer cleanup()

	mustUser(t, s, types.User{ID: "u-sd-twice"})
	if err := s.CreateChatSession(ctx(), "sess-twice", "u-sd-twice", "twice"); err != nil {
		t.Fatalf("CreateChatSession: %v", err)
	}

	// First delete succeeds.
	if err := s.SoftDeleteChatSession(ctx(), "u-sd-twice", "sess-twice"); err != nil {
		t.Fatalf("first SoftDeleteChatSession: %v", err)
	}

	// Second delete on same session: already deleted, should return ErrNotFound.
	if err := s.SoftDeleteChatSession(ctx(), "u-sd-twice", "sess-twice"); err != types.ErrNotFound {
		t.Fatalf("second SoftDeleteChatSession: got %v, want ErrNotFound", err)
	}
}

func TestSoftDeleteChatSessionForeignUser(t *testing.T) {
	s, cleanup := tempDB(t)
	defer cleanup()

	mustUser(t, s, types.User{ID: "u-owner"})
	mustUser(t, s, types.User{ID: "u-foreign"})
	if err := s.CreateChatSession(ctx(), "sess-f", "u-owner", "owner session"); err != nil {
		t.Fatalf("CreateChatSession: %v", err)
	}

	// Foreign user tries to delete owner's session.
	err := s.SoftDeleteChatSession(ctx(), "u-foreign", "sess-f")
	if err != types.ErrNotFound {
		t.Fatalf("SoftDeleteChatSession by foreign user: got %v, want ErrNotFound", err)
	}

	// Owner's session should still be active.
	active, err := s.ListChatSessions(ctx(), "u-owner")
	if err != nil {
		t.Fatalf("ListChatSessions: %v", err)
	}
	if len(active) != 1 {
		t.Fatalf("expected 1 active session, got %d", len(active))
	}
}

func TestRestoreChatSession(t *testing.T) {
	s, cleanup := tempDB(t)
	defer cleanup()

	mustUser(t, s, types.User{ID: "u-restore"})
	if err := s.CreateChatSession(ctx(), "sess-r", "u-restore", "restore me"); err != nil {
		t.Fatalf("CreateChatSession: %v", err)
	}

	// Soft-delete.
	if err := s.SoftDeleteChatSession(ctx(), "u-restore", "sess-r"); err != nil {
		t.Fatalf("SoftDeleteChatSession: %v", err)
	}

	// Restore.
	if err := s.RestoreChatSession(ctx(), "u-restore", "sess-r"); err != nil {
		t.Fatalf("RestoreChatSession: %v", err)
	}

	// Should now appear in active list.
	active, err := s.ListChatSessions(ctx(), "u-restore")
	if err != nil {
		t.Fatalf("ListChatSessions: %v", err)
	}
	if len(active) != 1 {
		t.Fatalf("expected 1 active session after restore, got %d", len(active))
	}
	if active[0].ID != "sess-r" {
		t.Errorf("restored session ID = %q, want sess-r", active[0].ID)
	}

	// Should no longer appear in deleted list.
	deleted, err := s.ListDeletedChatSessions(ctx(), "u-restore")
	if err != nil {
		t.Fatalf("ListDeletedChatSessions: %v", err)
	}
	if len(deleted) != 0 {
		t.Fatalf("expected 0 deleted sessions, got %d", len(deleted))
	}
}

func TestRestoreChatSessionNotDeleted(t *testing.T) {
	s, cleanup := tempDB(t)
	defer cleanup()

	mustUser(t, s, types.User{ID: "u-nd"})
	if err := s.CreateChatSession(ctx(), "sess-nd", "u-nd", "not deleted"); err != nil {
		t.Fatalf("CreateChatSession: %v", err)
	}

	err := s.RestoreChatSession(ctx(), "u-nd", "sess-nd")
	if err != types.ErrNotFound {
		t.Fatalf("RestoreChatSession on non-deleted session: got %v, want ErrNotFound", err)
	}
}

func TestPurgeDeletedChatSessions(t *testing.T) {
	s, cleanup := tempDB(t)
	defer cleanup()

	mustUser(t, s, types.User{ID: "u-purge"})
	if err := s.CreateChatSession(ctx(), "sess-p1", "u-purge", "old"); err != nil {
		t.Fatalf("CreateChatSession sess-p1: %v", err)
	}
	if err := s.CreateChatSession(ctx(), "sess-p2", "u-purge", "recent"); err != nil {
		t.Fatalf("CreateChatSession sess-p2: %v", err)
	}

	// Add messages to both sessions (to verify cascade).
	if err := s.AppendChatMessage(ctx(), "m-p1", "u-purge", "sess-p1", "user", "hi", ""); err != nil {
		t.Fatalf("AppendChatMessage sess-p1: %v", err)
	}
	if err := s.AppendChatMessage(ctx(), "m-p2", "u-purge", "sess-p2", "user", "hey", ""); err != nil {
		t.Fatalf("AppendChatMessage sess-p2: %v", err)
	}

	// Soft-delete both sessions.
	if err := s.SoftDeleteChatSession(ctx(), "u-purge", "sess-p1"); err != nil {
		t.Fatalf("SoftDeleteChatSession sess-p1: %v", err)
	}
	if err := s.SoftDeleteChatSession(ctx(), "u-purge", "sess-p2"); err != nil {
		t.Fatalf("SoftDeleteChatSession sess-p2: %v", err)
	}

	// Backdate sess-p1's deleted_at to 31 days ago (should be purged).
	_, err := s.db.Exec(`UPDATE chat_sessions SET deleted_at = ? WHERE id = ?`,
		time.Now().AddDate(0, 0, -31).UTC().Format("2006-01-02 15:04:05"), "sess-p1")
	if err != nil {
		t.Fatalf("backdate sess-p1: %v", err)
	}

	// sess-p2 deleted now (should NOT be purged).

	// Purge sessions older than 30 days.
	n, err := s.PurgeDeletedChatSessions(ctx(), time.Now().AddDate(0, 0, -30))
	if err != nil {
		t.Fatalf("PurgeDeletedChatSessions: %v", err)
	}
	if n != 1 {
		t.Fatalf("expected 1 purged session, got %d", n)
	}

	// sess-p1 should be gone from deleted list.
	deleted, err := s.ListDeletedChatSessions(ctx(), "u-purge")
	if err != nil {
		t.Fatalf("ListDeletedChatSessions: %v", err)
	}
	if len(deleted) != 1 {
		t.Fatalf("expected 1 deleted session remaining, got %d", len(deleted))
	}
	if deleted[0].ID != "sess-p2" {
		t.Errorf("remaining deleted session = %q, want sess-p2", deleted[0].ID)
	}

	// Messages for sess-p1 should cascade-delete, sess-p2 messages survive.
	msgs, err := s.GetChatMessages(ctx(), "u-purge", "sess-p2")
	if err != nil {
		t.Fatalf("GetChatMessages sess-p2: %v", err)
	}
	if len(msgs) != 1 {
		t.Fatalf("expected 1 message for sess-p2, got %d", len(msgs))
	}
}
