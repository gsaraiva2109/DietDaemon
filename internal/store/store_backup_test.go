package store

import (
	"testing"

	"github.com/gsaraiva2109/dietdaemon/core/types"
)

func TestBackupConfigCountsRoundTrip(t *testing.T) {
	s, cleanup := tempDB(t)
	defer cleanup()

	uid := "u-cnt"

	// Create a user first (FK constraint).
	mustUser(t, s, types.User{ID: uid})

	// No config yet — GetBackupConfig returns ErrNotFound.
	_, err := s.GetBackupConfig(ctx(), uid)
	if err != types.ErrNotFound {
		t.Fatalf("expected ErrNotFound before insert, got %v", err)
	}

	// Insert a config so the UPDATE has a row to hit.
	if err := s.SetBackupConfig(ctx(), types.BackupConfig{
		UserID: uid, Enabled: true, Destination: "local", IntervalHrs: 24,
	}); err != nil {
		t.Fatalf("SetBackupConfig: %v", err)
	}

	// Set counts.
	if err := s.SetBackupCounts(ctx(), uid, 42, 7); err != nil {
		t.Fatalf("SetBackupCounts: %v", err)
	}

	// Read back.
	cfg, err := s.GetBackupConfig(ctx(), uid)
	if err != nil {
		t.Fatalf("GetBackupConfig: %v", err)
	}
	if cfg.LastMealsCount != 42 {
		t.Fatalf("expected LastMealsCount=42, got %d", cfg.LastMealsCount)
	}
	if cfg.LastRollupsCount != 7 {
		t.Fatalf("expected LastRollupsCount=7, got %d", cfg.LastRollupsCount)
	}
}
