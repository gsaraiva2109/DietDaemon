package backup

import (
	"context"
	"testing"
	"time"

	"github.com/gsaraiva2109/dietdaemon/core/types"
)

type fakeStore struct {
	users    []types.User
	configs  map[string]types.BackupConfig
	lastRuns map[string]time.Time
}

func newFakeStore() *fakeStore {
	return &fakeStore{configs: map[string]types.BackupConfig{}, lastRuns: map[string]time.Time{}}
}

func (f *fakeStore) ListUsers(context.Context) ([]types.User, error) { return f.users, nil }

func (f *fakeStore) GetBackupConfig(_ context.Context, userID string) (types.BackupConfig, error) {
	cfg, ok := f.configs[userID]
	if !ok {
		return types.BackupConfig{}, types.ErrNotFound
	}
	return cfg, nil
}

func (f *fakeStore) SetBackupLastRun(_ context.Context, userID string, t time.Time) error {
	f.lastRuns[userID] = t
	cfg := f.configs[userID]
	cfg.LastRunAt = t
	f.configs[userID] = cfg
	return nil
}

func (f *fakeStore) GetMealsInRange(context.Context, string, string, string) ([]types.Meal, error) {
	return nil, nil
}

func (f *fakeStore) GetRollups(context.Context, string, string, string) ([]types.DailyRollup, error) {
	return nil, nil
}

type fakeDest struct {
	writes int
}

func (d *fakeDest) Write(context.Context, types.BackupConfig, string, []byte) error {
	d.writes++
	return nil
}

func TestTick_RunsWhenIntervalElapsed(t *testing.T) {
	store := newFakeStore()
	store.users = []types.User{{ID: "u1"}}
	store.configs["u1"] = types.BackupConfig{
		UserID: "u1", Enabled: true, Destination: "local",
		IntervalHrs: 24,
		LastRunAt:   time.Now().Add(-25 * time.Hour),
	}
	dst := &fakeDest{}
	r := New(store, dst, nil, time.Hour)

	r.tick(context.Background())

	if dst.writes != 2 { // meals.csv + rollups.csv
		t.Fatalf("expected 2 writes (meals+rollups), got %d", dst.writes)
	}
	if store.lastRuns["u1"].IsZero() {
		t.Fatalf("expected last_run_at to be updated")
	}
}

func TestTick_SkipsWhenNotYetDue(t *testing.T) {
	store := newFakeStore()
	store.users = []types.User{{ID: "u1"}}
	store.configs["u1"] = types.BackupConfig{
		UserID: "u1", Enabled: true, Destination: "local",
		IntervalHrs: 24,
		LastRunAt:   time.Now().Add(-1 * time.Hour), // only 1h ago, interval is 24h
	}
	dst := &fakeDest{}
	r := New(store, dst, nil, time.Hour)

	r.tick(context.Background())

	if dst.writes != 0 {
		t.Fatalf("expected no writes before interval elapses, got %d", dst.writes)
	}
	if !store.lastRuns["u1"].IsZero() {
		t.Fatalf("expected last_run_at untouched")
	}
}

func TestTick_SkipsDisabledOrUnconfigured(t *testing.T) {
	store := newFakeStore()
	store.users = []types.User{{ID: "u1"}, {ID: "u2"}}
	store.configs["u1"] = types.BackupConfig{UserID: "u1", Enabled: false, Destination: "local"}
	// u2 has no config at all -> types.ErrNotFound -> treated as disabled.
	dst := &fakeDest{}
	r := New(store, dst, nil, time.Hour)

	r.tick(context.Background())

	if dst.writes != 0 {
		t.Fatalf("expected no writes for disabled/unconfigured users, got %d", dst.writes)
	}
}

func TestRunOnce_IgnoresIntervalGate(t *testing.T) {
	store := newFakeStore()
	store.configs["u1"] = types.BackupConfig{
		UserID: "u1", Enabled: true, Destination: "local",
		IntervalHrs: 24,
		LastRunAt:   time.Now(), // just ran
	}
	dst := &fakeDest{}
	r := New(store, dst, nil, time.Hour)

	if err := r.RunOnce(context.Background(), "u1"); err != nil {
		t.Fatalf("RunOnce: %v", err)
	}
	if dst.writes != 2 {
		t.Fatalf("expected RunOnce to write regardless of interval, got %d writes", dst.writes)
	}
}

func TestRunFor_MissingDestinationErrors(t *testing.T) {
	store := newFakeStore()
	store.configs["u1"] = types.BackupConfig{UserID: "u1", Enabled: true, Destination: "s3"}
	r := New(store, nil, nil, time.Hour) // no s3 destination configured

	if err := r.RunOnce(context.Background(), "u1"); err == nil {
		t.Fatalf("expected error when s3 destination is nil")
	}
}
