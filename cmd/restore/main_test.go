package main

import (
	"bytes"
	"context"
	"fmt"
	"path/filepath"
	"testing"
	"time"

	"github.com/gsaraiva2109/dietdaemon/core/types"
	"github.com/gsaraiva2109/dietdaemon/internal/backup"
	"github.com/gsaraiva2109/dietdaemon/internal/backup/localdisk"
	"github.com/gsaraiva2109/dietdaemon/internal/store"
)

const testUserID = "user-1"

var wake = "2024-01-10T07:00:00Z"

// newTestStore opens a real temp-file SQLite store, matching the pattern
// used by cmd/import-mfp's tempStore helper.
func newTestStore(t *testing.T, path string) *store.Store {
	t.Helper()
	st, err := store.New("sqlite", path, store.SQLiteDialect(), nil)
	if err != nil {
		t.Fatalf("store.New: %v", err)
	}
	t.Cleanup(func() { _ = st.Close() })
	return st
}

// seedUser inserts the account/user rows every entity's foreign key requires.
func seedUser(t *testing.T, st *store.Store, userID string) {
	t.Helper()
	if _, err := st.DB().Exec(`INSERT INTO accounts (id, created_at) VALUES ('acct-1', datetime('now'))`); err != nil {
		t.Fatalf("insert test account: %v", err)
	}
	if _, err := st.DB().Exec(
		`INSERT INTO users (id, account_id, email, status, display_name, timezone, locale, created_at) VALUES (?, 'acct-1', ?, 'active', 'U', 'UTC', 'en', datetime('now'))`,
		userID, userID+"@example.com",
	); err != nil {
		t.Fatalf("insert test user: %v", err)
	}
}

// seedAllEntities writes a couple of rows per trackable entity using the real
// production store methods (not the Restore*/Import* idempotent variants),
// so the backup this test takes reflects genuine application writes.
func seedAllEntities(t *testing.T, st *store.Store, userID string) {
	t.Helper()
	ctx := context.Background()
	now := time.Now().UTC()

	meals := []types.Meal{
		{
			ID: "meal-1", UserID: userID, At: time.Date(2024, 1, 10, 8, 0, 0, 0, time.UTC),
			RawText: "oatmeal", Confidence: 1, ParserTier: types.TierDeterministic, CreatedAt: now,
			Items: []types.ResolvedItem{{Macros: types.Macros{Calories: 300, Protein: 10, Carbs: 50, Fat: 5, Fiber: 4}}},
		},
		{
			ID: "meal-2", UserID: userID, At: time.Date(2024, 1, 11, 12, 0, 0, 0, time.UTC),
			RawText: "chicken salad", Confidence: 1, ParserTier: types.TierDeterministic, CreatedAt: now,
			Items: []types.ResolvedItem{
				{Macros: types.Macros{Calories: 400, Protein: 30, Carbs: 20, Fat: 15, Fiber: 5}},
				{Macros: types.Macros{Calories: 100, Protein: 2, Carbs: 10, Fat: 3, Fiber: 1}},
			},
		},
	}
	for _, m := range meals {
		if err := st.SaveMeal(ctx, m); err != nil {
			t.Fatalf("SaveMeal: %v", err)
		}
	}

	rollups := []types.DailyRollup{
		{UserID: userID, Date: "2024-01-10", Consumed: types.Macros{Calories: 300, Protein: 10, Carbs: 50, Fat: 5, Fiber: 4}, Targets: types.Macros{Calories: 2000, Protein: 150, Carbs: 200, Fat: 60, Fiber: 30}},
		{UserID: userID, Date: "2024-01-11", Consumed: types.Macros{Calories: 500, Protein: 32, Carbs: 30, Fat: 18, Fiber: 6}, Targets: types.Macros{Calories: 2000, Protein: 150, Carbs: 200, Fat: 60, Fiber: 30}},
	}
	for _, r := range rollups {
		if err := st.UpsertRollup(ctx, r); err != nil {
			t.Fatalf("UpsertRollup: %v", err)
		}
	}

	for i, date := range []string{"2024-01-10", "2024-01-11"} {
		w := types.WeightEntry{ID: fmt.Sprintf("weight-%d", i+1), UserID: userID, Date: date, WeightKg: 70.5 + float64(i), Note: "n", CreatedAt: now}
		if _, err := st.LogWeight(ctx, w); err != nil {
			t.Fatalf("LogWeight: %v", err)
		}
	}

	for i, date := range []string{"2024-01-10", "2024-01-11"} {
		m := types.MeasurementEntry{
			ID: fmt.Sprintf("meas-%d", i+1), UserID: userID, Date: date,
			WaistCm: 80, HipsCm: 95, ChestCm: 100, LeftArmCm: 30, RightArmCm: 30, LeftThighCm: 55, RightThighCm: 55,
			Note: "n", CreatedAt: now,
		}
		if _, err := st.LogMeasurement(ctx, m); err != nil {
			t.Fatalf("LogMeasurement: %v", err)
		}
	}

	sleepLogs := []types.SleepLog{
		{ID: "sleep-1", UserID: userID, SleepAt: "2024-01-09T23:00:00Z", WakeAt: &wake, Quality: "good", Note: "n"},
		{ID: "sleep-2", UserID: userID, SleepAt: "2024-01-10T23:00:00Z", Quality: "fair"},
	}
	for _, sl := range sleepLogs {
		if err := st.LogSleep(ctx, sl); err != nil {
			t.Fatalf("LogSleep: %v", err)
		}
	}

	workouts := []types.Workout{
		{
			ID: "workout-1", UserID: userID, Name: "Leg day", DurationMin: 60, Intensity: "high", Note: "n",
			LoggedAt:  "2024-01-10T18:00:00Z",
			Exercises: []types.WorkoutExercise{{Name: "Squat", Sets: new(3), Reps: new(10), WeightKg: new(40.0)}},
		},
		{ID: "workout-2", UserID: userID, Name: "Run", DurationMin: 30, Intensity: "medium", LoggedAt: "2024-01-11T07:00:00Z"},
	}
	for _, w := range workouts {
		if err := st.LogWorkout(ctx, w); err != nil {
			t.Fatalf("LogWorkout: %v", err)
		}
	}

	waterLogs := []types.WaterLog{
		{ID: "water-1", UserID: userID, AmountML: 250, LoggedAt: "2024-01-10T09:00:00Z", Note: "n"},
		{ID: "water-2", UserID: userID, AmountML: 500, LoggedAt: "2024-01-10T15:00:00Z"},
	}
	for _, w := range waterLogs {
		if err := st.LogWater(ctx, w); err != nil {
			t.Fatalf("LogWater: %v", err)
		}
	}

	if err := st.StartFast(ctx, types.Fast{ID: "fast-1", UserID: userID, StartAt: time.Date(2024, 1, 12, 20, 0, 0, 0, time.UTC), TargetHours: 16, CreatedAt: now}); err != nil {
		t.Fatalf("StartFast (open): %v", err)
	}
	if err := st.StartFast(ctx, types.Fast{ID: "fast-2", UserID: userID, StartAt: time.Date(2024, 1, 8, 20, 0, 0, 0, time.UTC), TargetHours: 16, CreatedAt: now}); err != nil {
		t.Fatalf("StartFast (to close): %v", err)
	}
	if _, err := st.EndFast(ctx, userID, "fast-2", time.Date(2024, 1, 9, 12, 0, 0, 0, time.UTC), true); err != nil {
		t.Fatalf("EndFast: %v", err)
	}

	if err := st.UploadPhoto(ctx, types.ProgressPhoto{
		ID: "photo-1", UserID: userID, Date: "2024-01-10", View: "front", MimeType: "image/png",
		Data: bytes.Repeat([]byte{0x89, 0x50, 0x4e, 0x47}, 10), CreatedAt: now,
	}); err != nil {
		t.Fatalf("UploadPhoto: %v", err)
	}
}

func countRows(t *testing.T, st *store.Store, table, userID string) int {
	t.Helper()
	var n int
	if err := st.DB().QueryRow(fmt.Sprintf("SELECT COUNT(*) FROM %s WHERE user_id = ?", table), userID).Scan(&n); err != nil { // #nosec G201 -- table names are test-local constants
		t.Fatalf("count %s: %v", table, err)
	}
	return n
}

func TestRestoreCLI_RoundTrip(t *testing.T) {
	dir := t.TempDir()
	srcDBPath := filepath.Join(dir, "src.db")
	targetDBPath := filepath.Join(dir, "target.db")
	backupDir := filepath.Join(dir, "backup")
	ctx := context.Background()

	// 1. Seed a source store with real production writes across all 9 entities.
	srcStore := newTestStore(t, srcDBPath)
	seedUser(t, srcStore, testUserID)
	seedAllEntities(t, srcStore, testUserID)

	if err := srcStore.SetBackupConfig(ctx, types.BackupConfig{UserID: testUserID, Enabled: true, Destination: "local", IntervalHrs: 24}); err != nil {
		t.Fatalf("SetBackupConfig: %v", err)
	}

	// 2. Back it up to a local disk dir via internal/backup.
	localDst, err := localdisk.New(backupDir)
	if err != nil {
		t.Fatalf("localdisk.New: %v", err)
	}
	backupRunner := backup.New(srcStore, localDst, nil, time.Hour)
	if err := backupRunner.RunOnce(ctx, testUserID); err != nil {
		t.Fatalf("backup RunOnce: %v", err)
	}

	// 3. Restore into a second, empty target store via the CLI's run().
	targetStore := newTestStore(t, targetDBPath)
	seedUser(t, targetStore, testUserID)
	if err := targetStore.Close(); err != nil {
		t.Fatalf("close target seed store: %v", err)
	}

	if err := run(ctx, testUserID, targetDBPath, "local", backupDir, "", "", "", "", "", false); err != nil {
		t.Fatalf("restore run: %v", err)
	}

	// 4. Assert restored data matches what was written to the source.
	assertStore := newTestStore(t, targetDBPath)

	meals, err := assertStore.GetMealsInRange(ctx, testUserID, "1970-01-01", "2100-01-01")
	if err != nil {
		t.Fatalf("GetMealsInRange: %v", err)
	}
	if len(meals) != 2 {
		t.Fatalf("restored meals = %d, want 2", len(meals))
	}
	var meal2 types.Meal
	for _, m := range meals {
		if m.ID == "meal-2" {
			meal2 = m
		}
	}
	if total := meal2.Total(); total.Calories != 500 || total.Protein != 32 {
		t.Errorf("meal-2 restored totals = %+v, want kcal=500 protein=32", total)
	}

	rollups, err := assertStore.GetRollups(ctx, testUserID, "1970-01-01", "2100-01-01")
	if err != nil {
		t.Fatalf("GetRollups: %v", err)
	}
	if len(rollups) != 2 {
		t.Fatalf("restored rollups = %d, want 2", len(rollups))
	}

	weight, err := assertStore.ListWeight(ctx, testUserID, 100000)
	if err != nil {
		t.Fatalf("ListWeight: %v", err)
	}
	if len(weight) != 2 {
		t.Fatalf("restored weight = %d, want 2", len(weight))
	}

	measurements, err := assertStore.ListMeasurements(ctx, testUserID, 100000)
	if err != nil {
		t.Fatalf("ListMeasurements: %v", err)
	}
	if len(measurements) != 2 {
		t.Fatalf("restored measurements = %d, want 2", len(measurements))
	}

	sleep, err := assertStore.ListSleep(ctx, testUserID, 100)
	if err != nil {
		t.Fatalf("ListSleep: %v", err)
	}
	if len(sleep) != 2 {
		t.Fatalf("restored sleep = %d, want 2", len(sleep))
	}
	var sleep1 types.SleepLog
	for _, sl := range sleep {
		if sl.ID == "sleep-1" {
			sleep1 = sl
		}
	}
	if sleep1.WakeAt == nil || *sleep1.WakeAt != wake {
		t.Errorf("sleep-1 restored WakeAt = %v, want %s", sleep1.WakeAt, wake)
	}

	workouts, err := assertStore.GetWorkoutsInRangeWithExercises(ctx, testUserID, "1970-01-01", "2100-01-01")
	if err != nil {
		t.Fatalf("GetWorkoutsInRangeWithExercises: %v", err)
	}
	if len(workouts) != 2 {
		t.Fatalf("restored workouts = %d, want 2", len(workouts))
	}
	var workout1 types.Workout
	for _, w := range workouts {
		if w.ID == "workout-1" {
			workout1 = w
		}
	}
	if len(workout1.Exercises) != 1 || workout1.Exercises[0].Name != "Squat" {
		t.Fatalf("workout-1 restored exercises = %+v, want 1 exercise named Squat", workout1.Exercises)
	}
	if workout1.Exercises[0].Sets == nil || *workout1.Exercises[0].Sets != 3 {
		t.Errorf("workout-1 restored sets = %v, want 3", workout1.Exercises[0].Sets)
	}

	water, err := assertStore.GetWaterInRange(ctx, testUserID, "1970-01-01", "2100-01-01")
	if err != nil {
		t.Fatalf("GetWaterInRange: %v", err)
	}
	if len(water) != 2 {
		t.Fatalf("restored water = %d, want 2", len(water))
	}

	fasts, err := assertStore.ListFasts(ctx, testUserID, 100)
	if err != nil {
		t.Fatalf("ListFasts: %v", err)
	}
	if len(fasts) != 2 {
		t.Fatalf("restored fasts = %d, want 2", len(fasts))
	}
	var fast2 types.Fast
	for _, f := range fasts {
		if f.ID == "fast-2" {
			fast2 = f
		}
	}
	if fast2.EndAt == nil || !fast2.Completed {
		t.Errorf("fast-2 restored EndAt/Completed = %v/%v, want set/true", fast2.EndAt, fast2.Completed)
	}

	photos, err := assertStore.ListPhotoMetadata(ctx, testUserID)
	if err != nil {
		t.Fatalf("ListPhotoMetadata: %v", err)
	}
	if len(photos) != 1 {
		t.Fatalf("restored photos = %d, want 1", len(photos))
	}
	full, err := assertStore.GetPhotoData(ctx, photos[0].ID)
	if err != nil {
		t.Fatalf("GetPhotoData: %v", err)
	}
	if !bytes.Equal(full.Data, bytes.Repeat([]byte{0x89, 0x50, 0x4e, 0x47}, 10)) {
		t.Errorf("restored photo data mismatch")
	}

	if err := assertStore.Close(); err != nil {
		t.Fatalf("close assert store: %v", err)
	}

	// 5. Re-running restore against the same target must be a no-op: no
	// error, and no duplicate rows for any entity.
	if err := run(ctx, testUserID, targetDBPath, "local", backupDir, "", "", "", "", "", false); err != nil {
		t.Fatalf("second restore run: %v", err)
	}

	idempotentStore := newTestStore(t, targetDBPath)
	checks := map[string]int{
		"meals": 2, "daily_rollups": 2, "weight_log": 2, "measurement_log": 2,
		"sleep_logs": 2, "workouts": 2, "water_logs": 2, "fasts": 2, "progress_photos": 1,
	}
	for table, want := range checks {
		if got := countRows(t, idempotentStore, table, testUserID); got != want {
			t.Errorf("after second restore, %s count = %d, want %d (unchanged, no duplicates)", table, got, want)
		}
	}
}

func TestRestoreCLI_DryRun(t *testing.T) {
	dir := t.TempDir()
	srcDBPath := filepath.Join(dir, "src.db")
	backupDir := filepath.Join(dir, "backup")
	ctx := context.Background()

	srcStore := newTestStore(t, srcDBPath)
	seedUser(t, srcStore, testUserID)
	seedAllEntities(t, srcStore, testUserID)
	if err := srcStore.SetBackupConfig(ctx, types.BackupConfig{UserID: testUserID, Enabled: true, Destination: "local", IntervalHrs: 24}); err != nil {
		t.Fatalf("SetBackupConfig: %v", err)
	}
	localDst, err := localdisk.New(backupDir)
	if err != nil {
		t.Fatalf("localdisk.New: %v", err)
	}
	if err := backup.New(srcStore, localDst, nil, time.Hour).RunOnce(ctx, testUserID); err != nil {
		t.Fatalf("backup RunOnce: %v", err)
	}

	// A non-existent -db path proves dry-run never opens the store: if it
	// did, store.New would fail trying to migrate a file in a directory
	// that doesn't exist.
	missingDBPath := filepath.Join(dir, "does", "not", "exist", "target.db")
	if err := run(ctx, testUserID, missingDBPath, "local", backupDir, "", "", "", "", "", true); err != nil {
		t.Fatalf("dry-run should not error even with an unreachable -db path: %v", err)
	}
}
