package commands

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/gsaraiva2109/dietdaemon/core/types"
)

// fakeNudgeStore is a minimal stub for /nudge undo tests.
type fakeNudgeStore struct {
	nudges map[string]types.SentNudge

	updateCalled    bool
	updateGotID     string
	updateGotStatus string
}

func (f *fakeNudgeStore) GetSentNudge(_ context.Context, id string) (types.SentNudge, error) {
	if n, ok := f.nudges[id]; ok {
		return n, nil
	}
	return types.SentNudge{}, types.ErrNotFound
}

func (f *fakeNudgeStore) UpdateSentNudgeStatus(_ context.Context, id, status string) error {
	f.updateCalled = true
	f.updateGotID = id
	f.updateGotStatus = status
	return nil
}

func TestNudgeUndo_OwnNudge(t *testing.T) {
	store := &fakeNudgeStore{
		nudges: map[string]types.SentNudge{
			"n1": {
				ID:     "n1",
				UserID: "u1",
				RuleID: "protein-evening",
				SentAt: time.Now(),
				Body:   "Protein behind: 100/180 g",
				Status: "sent",
			},
		},
	}
	cmd := NewNudgeCommand(store)

	reply, err := cmd.Handle(context.Background(), types.InboundMessage{UserID: "u1"}, "undo n1")
	if err != nil {
		t.Fatalf("Handle: %v", err)
	}
	if !store.updateCalled {
		t.Fatal("expected UpdateSentNudgeStatus to be called")
	}
	if store.updateGotID != "n1" {
		t.Errorf("update id = %q, want %q", store.updateGotID, "n1")
	}
	if store.updateGotStatus != "dismissed" {
		t.Errorf("update status = %q, want %q", store.updateGotStatus, "dismissed")
	}
	if !strings.Contains(strings.ToLower(reply.Text), "undo") || !strings.Contains(strings.ToLower(reply.Text), "n1") {
		t.Errorf("expected confirmation reply mentioning undo and id, got %q", reply.Text)
	}
}

func TestNudgeUndo_WrongUser(t *testing.T) {
	store := &fakeNudgeStore{
		nudges: map[string]types.SentNudge{
			"n1": {
				ID:     "n1",
				UserID: "u2", // belongs to someone else
				RuleID: "protein-evening",
				SentAt: time.Now(),
				Body:   "...",
				Status: "sent",
			},
		},
	}
	cmd := NewNudgeCommand(store)

	reply, err := cmd.Handle(context.Background(), types.InboundMessage{UserID: "u1"}, "undo n1")
	if err != nil {
		t.Fatalf("Handle: %v", err)
	}
	if store.updateCalled {
		t.Fatal("expected UpdateSentNudgeStatus NOT to be called for someone else's nudge")
	}
	if !strings.Contains(strings.ToLower(reply.Text), "not yours") || !strings.Contains(strings.ToLower(reply.Text), "n1") {
		t.Errorf("expected rejection reply, got %q", reply.Text)
	}
}

func TestNudgeUndo_AlreadyHandled(t *testing.T) {
	store := &fakeNudgeStore{
		nudges: map[string]types.SentNudge{
			"n1": {
				ID:     "n1",
				UserID: "u1",
				RuleID: "protein-evening",
				SentAt: time.Now(),
				Body:   "Protein behind: 100/180 g",
				Status: "dismissed", // already handled
			},
		},
	}
	cmd := NewNudgeCommand(store)

	reply, err := cmd.Handle(context.Background(), types.InboundMessage{UserID: "u1"}, "undo n1")
	if err != nil {
		t.Fatalf("Handle: %v", err)
	}
	if store.updateCalled {
		t.Fatal("expected UpdateSentNudgeStatus NOT to be called on already-handled nudge")
	}
	if !strings.Contains(strings.ToLower(reply.Text), "already handled") {
		t.Errorf("expected 'already handled' reply, got %q", reply.Text)
	}
}

func TestNudgeUndo_NotFound(t *testing.T) {
	store := &fakeNudgeStore{nudges: map[string]types.SentNudge{}}
	cmd := NewNudgeCommand(store)

	reply, err := cmd.Handle(context.Background(), types.InboundMessage{UserID: "u1"}, "undo nope")
	if err != nil {
		t.Fatalf("Handle: %v", err)
	}
	if store.updateCalled {
		t.Fatal("expected UpdateSentNudgeStatus NOT to be called for missing nudge")
	}
	if !strings.Contains(strings.ToLower(reply.Text), "not found") {
		t.Errorf("expected 'not found' reply, got %q", reply.Text)
	}
}
