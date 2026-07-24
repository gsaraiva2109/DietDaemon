package commands

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/gsaraiva2109/dietdaemon/core/types"
)

// fakeSleepStore is a minimal SleepStore stub for /sleep tests.
type fakeSleepStore struct {
	active    *types.SleepLog
	activeErr error
	logs      []types.SleepLog
	listErr   error
	logged    *types.SleepLog
	logErr    error
}

func (f *fakeSleepStore) LogSleep(_ context.Context, sl types.SleepLog) error {
	if f.logErr != nil {
		return f.logErr
	}
	f.logged = &sl
	return nil
}

func (f *fakeSleepStore) GetActiveSleep(_ context.Context, _ string) (*types.SleepLog, error) {
	return f.active, f.activeErr
}

func (f *fakeSleepStore) ListSleep(_ context.Context, _ string, _ int) ([]types.SleepLog, error) {
	return f.logs, f.listErr
}

func TestSleepCommand_UsageOnEmpty(t *testing.T) {
	cmd := NewSleepCommand(&fakeSleepStore{})
	reply, err := cmd.Handle(context.Background(), types.InboundMessage{UserID: "u1"}, "")
	if err != nil {
		t.Fatalf("Handle: %v", err)
	}
	if !strings.Contains(reply.Text, "Usage:") {
		t.Errorf("expected usage reply, got %q", reply.Text)
	}
}

func TestSleepCommand_StatusNoActiveSession(t *testing.T) {
	cmd := NewSleepCommand(&fakeSleepStore{activeErr: errors.New("none")})
	reply, err := cmd.Handle(context.Background(), types.InboundMessage{UserID: "u1"}, "status")
	if err != nil {
		t.Fatalf("Handle: %v", err)
	}
	if !strings.Contains(reply.Text, "No active sleep session") {
		t.Errorf("expected no-active-session reply, got %q", reply.Text)
	}
}

func TestSleepCommand_StatusActiveSession(t *testing.T) {
	// sleep_at one hour before now guarantees a stable, predictable elapsed time.
	past := time.Now().Add(-1 * time.Hour).Format("15:04")
	store := &fakeSleepStore{active: &types.SleepLog{SleepAt: past}}
	cmd := NewSleepCommand(store)

	reply, err := cmd.Handle(context.Background(), types.InboundMessage{UserID: "u1"}, "status")
	if err != nil {
		t.Fatalf("Handle: %v", err)
	}
	if !strings.Contains(reply.Text, "Sleeping since "+past) {
		t.Errorf("expected active session reply, got %q", reply.Text)
	}
	if !strings.Contains(reply.Text, "elapsed") {
		t.Errorf("expected elapsed duration, got %q", reply.Text)
	}
}

func TestSleepCommand_ListEmpty(t *testing.T) {
	cmd := NewSleepCommand(&fakeSleepStore{})
	reply, err := cmd.Handle(context.Background(), types.InboundMessage{UserID: "u1"}, "list")
	if err != nil {
		t.Fatalf("Handle: %v", err)
	}
	if reply.Text != "No sleep logs yet." {
		t.Errorf("reply.Text = %q, want no-logs message", reply.Text)
	}
}

func TestSleepCommand_ListWithEntries(t *testing.T) {
	wake := "07:00"
	store := &fakeSleepStore{logs: []types.SleepLog{
		{SleepAt: "23:00", WakeAt: &wake, Quality: "good", Note: "felt rested"},
		{SleepAt: "01:00", WakeAt: nil, Quality: "ok"},
	}}
	cmd := NewSleepCommand(store)

	reply, err := cmd.Handle(context.Background(), types.InboundMessage{UserID: "u1"}, "list")
	if err != nil {
		t.Fatalf("Handle: %v", err)
	}
	if !strings.Contains(reply.Text, "23:00 to 07:00 (8.0h) — good") {
		t.Errorf("expected first entry line, got %q", reply.Text)
	}
	if !strings.Contains(reply.Text, "felt rested") {
		t.Errorf("expected note line, got %q", reply.Text)
	}
	if !strings.Contains(reply.Text, "01:00 to active") {
		t.Errorf("expected active entry, got %q", reply.Text)
	}
}

func TestSleepCommand_LogTooFewArgs(t *testing.T) {
	cmd := NewSleepCommand(&fakeSleepStore{})
	reply, err := cmd.Handle(context.Background(), types.InboundMessage{UserID: "u1"}, "23:00")
	if err != nil {
		t.Fatalf("Handle: %v", err)
	}
	if !strings.Contains(reply.Text, "Usage:") {
		t.Errorf("expected usage reply, got %q", reply.Text)
	}
}

func TestSleepCommand_LogInvalidSleepTime(t *testing.T) {
	cmd := NewSleepCommand(&fakeSleepStore{})
	reply, err := cmd.Handle(context.Background(), types.InboundMessage{UserID: "u1"}, "25:99 07:00")
	if err != nil {
		t.Fatalf("Handle: %v", err)
	}
	if !strings.Contains(reply.Text, "Invalid time format: 25:99") {
		t.Errorf("expected invalid-sleep-time reply, got %q", reply.Text)
	}
}

func TestSleepCommand_LogInvalidWakeTime(t *testing.T) {
	cmd := NewSleepCommand(&fakeSleepStore{})
	reply, err := cmd.Handle(context.Background(), types.InboundMessage{UserID: "u1"}, "23:00 notatime")
	if err != nil {
		t.Fatalf("Handle: %v", err)
	}
	if !strings.Contains(reply.Text, "Invalid time format: notatime") {
		t.Errorf("expected invalid-wake-time reply, got %q", reply.Text)
	}
}

func TestSleepCommand_LogValidNoQuality(t *testing.T) {
	store := &fakeSleepStore{}
	cmd := NewSleepCommand(store)

	reply, err := cmd.Handle(context.Background(), types.InboundMessage{UserID: "u1"}, "23:00 07:00")
	if err != nil {
		t.Fatalf("Handle: %v", err)
	}
	if store.logged == nil {
		t.Fatal("expected LogSleep to be called")
	}
	if store.logged.Quality != "ok" {
		t.Errorf("Quality = %q, want ok (default)", store.logged.Quality)
	}
	if !strings.Contains(reply.Text, "8.0h from 23:00 to 07:00 (ok)") {
		t.Errorf("expected logged confirmation, got %q", reply.Text)
	}
}

func TestSleepCommand_LogValidWithQualityAndNote(t *testing.T) {
	store := &fakeSleepStore{}
	cmd := NewSleepCommand(store)

	reply, err := cmd.Handle(context.Background(), types.InboundMessage{UserID: "u1"}, "23:00 07:00 good felt rested well")
	if err != nil {
		t.Fatalf("Handle: %v", err)
	}
	if store.logged.Quality != "good" {
		t.Errorf("Quality = %q, want good", store.logged.Quality)
	}
	if store.logged.Note != "felt rested well" {
		t.Errorf("Note = %q, want %q", store.logged.Note, "felt rested well")
	}
	if !strings.Contains(reply.Text, "(good)") {
		t.Errorf("expected quality in reply, got %q", reply.Text)
	}
}

// TestSleepCommand_LogInvalidQualityRejected pins the S3923 fix: an
// unrecognized third token must be rejected with a clear error instead of
// being silently accepted as the quality (which used to also eat the next
// word as a "note").
func TestSleepCommand_LogInvalidQualityRejected(t *testing.T) {
	store := &fakeSleepStore{}
	cmd := NewSleepCommand(store)

	reply, err := cmd.Handle(context.Background(), types.InboundMessage{UserID: "u1"}, "23:00 07:00 godo")
	if err != nil {
		t.Fatalf("Handle: %v", err)
	}
	if store.logged != nil {
		t.Fatal("expected LogSleep NOT to be called for an invalid quality")
	}
	if !strings.Contains(reply.Text, "Invalid quality: godo") {
		t.Errorf("expected invalid-quality reply, got %q", reply.Text)
	}
	if !strings.Contains(reply.Text, "poor, fair, good, great") {
		t.Errorf("expected reply to list valid qualities, got %q", reply.Text)
	}
}

func TestSleepCommand_LogQualityCaseInsensitive(t *testing.T) {
	store := &fakeSleepStore{}
	cmd := NewSleepCommand(store)

	_, err := cmd.Handle(context.Background(), types.InboundMessage{UserID: "u1"}, "23:00 07:00 GREAT")
	if err != nil {
		t.Fatalf("Handle: %v", err)
	}
	if store.logged == nil || store.logged.Quality != "great" {
		t.Fatalf("expected Quality=great, got %+v", store.logged)
	}
}

func TestSleepCommand_LogStoreError(t *testing.T) {
	store := &fakeSleepStore{logErr: errors.New("db down")}
	cmd := NewSleepCommand(store)

	_, err := cmd.Handle(context.Background(), types.InboundMessage{UserID: "u1"}, "23:00 07:00 good")
	if err == nil {
		t.Fatal("expected error when LogSleep fails")
	}
}

func TestSleepCommand_Metadata(t *testing.T) {
	cmd := NewSleepCommand(&fakeSleepStore{})
	if cmd.Name() != "/sleep" {
		t.Errorf("Name() = %q, want /sleep", cmd.Name())
	}
	if cmd.Help() != types.I18nKey("cmd.sleep.usage") {
		t.Errorf("Help() = %q, want cmd.sleep.usage", cmd.Help())
	}
}

func TestCalcSleepHours_Overnight(t *testing.T) {
	wake := "07:00"
	got := calcSleepHours("23:00", &wake)
	if got != 8.0 {
		t.Errorf("calcSleepHours = %v, want 8.0", got)
	}
}

func TestCalcSleepHours_NilWake(t *testing.T) {
	if got := calcSleepHours("23:00", nil); got != 0 {
		t.Errorf("calcSleepHours = %v, want 0", got)
	}
}
