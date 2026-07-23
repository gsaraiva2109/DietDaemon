package backup

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"strings"
	"testing"
	"time"

	"github.com/gsaraiva2109/dietdaemon/core/types"
	"github.com/gsaraiva2109/dietdaemon/internal/exportfmt"
)

type fakeStore struct {
	users      []types.User
	configs    map[string]types.BackupConfig
	lastRuns   map[string]time.Time
	mealCounts map[string]int
	rollCounts map[string]int
}

func newFakeStore() *fakeStore {
	return &fakeStore{
		configs:    map[string]types.BackupConfig{},
		lastRuns:   map[string]time.Time{},
		mealCounts: map[string]int{},
		rollCounts: map[string]int{},
	}
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

func (f *fakeStore) SetBackupCounts(_ context.Context, userID string, mealsCount, rollupsCount int) error {
	f.mealCounts[userID] = mealsCount
	f.rollCounts[userID] = rollupsCount
	cfg := f.configs[userID]
	cfg.LastMealsCount = mealsCount
	cfg.LastRollupsCount = rollupsCount
	f.configs[userID] = cfg
	return nil
}

func (f *fakeStore) GetMealsInRange(context.Context, string, string, string) ([]types.Meal, error) {
	return nil, nil
}

func (f *fakeStore) GetRollups(context.Context, string, string, string) ([]types.DailyRollup, error) {
	return nil, nil
}

func (f *fakeStore) ListWeight(context.Context, string, int) ([]types.WeightEntry, error) {
	return nil, nil
}

func (f *fakeStore) ListMeasurements(context.Context, string, int) ([]types.MeasurementEntry, error) {
	return nil, nil
}

func (f *fakeStore) ListSleep(context.Context, string, int) ([]types.SleepLog, error) {
	return nil, nil
}

func (f *fakeStore) ListFasts(context.Context, string, int) ([]types.Fast, error) {
	return nil, nil
}

func (f *fakeStore) ListPhotoMetadata(context.Context, string) ([]types.ProgressPhoto, error) {
	return nil, nil
}

func (f *fakeStore) GetPhotoData(context.Context, string) (types.ProgressPhoto, error) {
	return types.ProgressPhoto{}, nil
}

func (f *fakeStore) GetWaterInRange(context.Context, string, string, string) ([]types.WaterLog, error) {
	return nil, nil
}

func (f *fakeStore) GetWorkoutsInRangeWithExercises(context.Context, string, string, string) ([]types.Workout, error) {
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

	if dst.writes != 9 { // meals, rollups, weight, measurements, sleep, workouts, water, fasts, photos csv (no photo blobs)
		t.Fatalf("expected 9 writes (9 empty CSVs), got %d", dst.writes)
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
	if dst.writes != 9 {
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

func TestRunFor_SetsBackupCounts(t *testing.T) {
	store := newFakeStore()
	store.configs["u1"] = types.BackupConfig{UserID: "u1", Enabled: true, Destination: "local"}
	dst := &fakeDest{}
	r := New(store, dst, nil, time.Hour)

	if err := r.RunOnce(context.Background(), "u1"); err != nil {
		t.Fatalf("RunOnce: %v", err)
	}
	// Both meals and rollups return empty slices, so counts should be 0.
	if store.mealCounts["u1"] != 0 {
		t.Fatalf("expected meal count 0, got %d", store.mealCounts["u1"])
	}
	if store.rollCounts["u1"] != 0 {
		t.Fatalf("expected rollup count 0, got %d", store.rollCounts["u1"])
	}
}

// allEntitiesFakeStore embeds fakeStore and overrides every list/get method
// to return exactly one row, so a run exercises every entity's export path.
type allEntitiesFakeStore struct {
	*fakeStore
}

func (f *allEntitiesFakeStore) GetMealsInRange(context.Context, string, string, string) ([]types.Meal, error) {
	return []types.Meal{{ID: "m1"}}, nil
}

func (f *allEntitiesFakeStore) GetRollups(context.Context, string, string, string) ([]types.DailyRollup, error) {
	return []types.DailyRollup{{Date: "2026-01-01"}}, nil
}

func (f *allEntitiesFakeStore) ListWeight(context.Context, string, int) ([]types.WeightEntry, error) {
	return []types.WeightEntry{{ID: "w1"}}, nil
}

func (f *allEntitiesFakeStore) ListMeasurements(context.Context, string, int) ([]types.MeasurementEntry, error) {
	return []types.MeasurementEntry{{ID: "meas1"}}, nil
}

func (f *allEntitiesFakeStore) ListSleep(context.Context, string, int) ([]types.SleepLog, error) {
	return []types.SleepLog{{ID: "s1"}}, nil
}

func (f *allEntitiesFakeStore) ListFasts(context.Context, string, int) ([]types.Fast, error) {
	return []types.Fast{{ID: "f1"}}, nil
}

func (f *allEntitiesFakeStore) ListPhotoMetadata(context.Context, string) ([]types.ProgressPhoto, error) {
	return []types.ProgressPhoto{{ID: "p1"}}, nil
}

func (f *allEntitiesFakeStore) GetPhotoData(_ context.Context, photoID string) (types.ProgressPhoto, error) {
	return types.ProgressPhoto{ID: photoID, Data: []byte("jpeg-bytes")}, nil
}

func (f *allEntitiesFakeStore) GetWaterInRange(context.Context, string, string, string) ([]types.WaterLog, error) {
	return []types.WaterLog{{ID: "wt1"}}, nil
}

func (f *allEntitiesFakeStore) GetWorkoutsInRangeWithExercises(context.Context, string, string, string) ([]types.Workout, error) {
	return []types.Workout{{ID: "wk1"}}, nil
}

func TestRunFor_ExportsAllEntities(t *testing.T) {
	store := &allEntitiesFakeStore{newFakeStore()}
	store.configs["u1"] = types.BackupConfig{UserID: "u1", Enabled: true, Destination: "local"}
	dst := &fakeDest{}
	r := New(store, dst, nil, time.Hour)

	if err := r.RunOnce(context.Background(), "u1"); err != nil {
		t.Fatalf("RunOnce: %v", err)
	}
	// 9 CSVs (meals, rollups, weight, measurements, sleep, workouts, water,
	// fasts, photos) + 1 photo blob = 10.
	if dst.writes != 10 {
		t.Fatalf("expected 10 writes (9 CSVs + 1 photo blob), got %d", dst.writes)
	}
}

// weightErrFakeStore fails ListWeight after meals/rollups have already
// loaded and written successfully, to exercise the abort-on-error path.
type weightErrFakeStore struct {
	*allEntitiesFakeStore
}

func (f *weightErrFakeStore) ListWeight(context.Context, string, int) ([]types.WeightEntry, error) {
	return nil, errors.New("db down")
}

// TestRunFor_LoadErrorAbortsRemainingEntities pins the asymmetry with
// restore.RunOnce: a load-or-write error on one entity aborts the rest of
// runFor immediately instead of collecting the error and continuing.
func TestRunFor_LoadErrorAbortsRemainingEntities(t *testing.T) {
	store := &weightErrFakeStore{&allEntitiesFakeStore{newFakeStore()}}
	store.configs["u1"] = types.BackupConfig{UserID: "u1", Enabled: true, Destination: "local"}
	dst := &fakeDest{}
	r := New(store, dst, nil, time.Hour)

	err := r.RunOnce(context.Background(), "u1")
	if err == nil {
		t.Fatal("expected error from weight load failure")
	}
	if !strings.Contains(err.Error(), "load weight") {
		t.Fatalf("expected error to mention 'load weight', got %v", err)
	}
	// Only meals.csv and rollups.csv should have been written before the
	// weight load failure aborted the run; measurements/sleep/etc never run.
	if dst.writes != 2 {
		t.Fatalf("expected exactly 2 writes (meals, rollups) before abort, got %d", dst.writes)
	}
	if !store.lastRuns["u1"].IsZero() {
		t.Fatalf("expected last_run_at NOT updated when the run aborts early")
	}
}

// orderedFakeDest records the filenames written, in order, so ordering
// guarantees between entities can be asserted.
type orderedFakeDest struct {
	filenames []string
}

func (d *orderedFakeDest) Write(_ context.Context, _ types.BackupConfig, filename string, _ []byte) error {
	d.filenames = append(d.filenames, filename)
	return nil
}

// TestRunFor_PhotoBlobWrittenBeforeIndex pins the ordering guarantee called
// out in runFor's comment: every photo blob is written before photos.csv, so
// a recovered index never references a missing blob.
func TestRunFor_PhotoBlobWrittenBeforeIndex(t *testing.T) {
	store := &allEntitiesFakeStore{newFakeStore()}
	store.configs["u1"] = types.BackupConfig{UserID: "u1", Enabled: true, Destination: "local"}
	dst := &orderedFakeDest{}
	r := New(store, dst, nil, time.Hour)

	if err := r.RunOnce(context.Background(), "u1"); err != nil {
		t.Fatalf("RunOnce: %v", err)
	}

	blobIdx, indexIdx := -1, -1
	wantBlob := exportfmt.PhotoFilename("p1")
	for i, f := range dst.filenames {
		if f == wantBlob {
			blobIdx = i
		}
		if f == "photos.csv" {
			indexIdx = i
		}
	}
	if blobIdx == -1 || indexIdx == -1 {
		t.Fatalf("expected both %q and photos.csv to be written, got %v", wantBlob, dst.filenames)
	}
	if blobIdx > indexIdx {
		t.Fatalf("photo blob written after photos.csv index: %v", dst.filenames)
	}
}

func TestRunFor_WarnsOnCountDrop(t *testing.T) {
	store := newFakeStore()
	// Previous run had 100 meals and 50 rollups.
	store.configs["u1"] = types.BackupConfig{
		UserID: "u1", Enabled: true, Destination: "local",
		LastMealsCount: 100, LastRollupsCount: 50,
	}
	dst := &fakeDest{}
	r := New(store, dst, nil, time.Hour)

	// Capture log output.
	var buf bytes.Buffer
	r.log = slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelWarn}))

	if err := r.RunOnce(context.Background(), "u1"); err != nil {
		t.Fatalf("RunOnce: %v", err)
	}

	// meals and rollups both return empty (0), which is >50% drop from 100 and 50.
	output := buf.String()
	if output == "" {
		t.Fatalf("expected warning logs for row count drops, got none")
	}
}
