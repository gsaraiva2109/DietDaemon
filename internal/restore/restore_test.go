package restore

import (
	"bytes"
	"context"
	"fmt"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/gsaraiva2109/dietdaemon/core/types"
	"github.com/gsaraiva2109/dietdaemon/internal/exportfmt"
)

// fakeSource is a hand-rolled Source: files holds what Read returns per
// filename, list is what List reports as present, and readErr forces Read to
// fail for specific filenames (simulating a missing/corrupt blob).
type fakeSource struct {
	files   map[string][]byte
	list    []string
	readErr map[string]error
}

func (f *fakeSource) List(context.Context, types.BackupConfig) ([]string, error) {
	return f.list, nil
}

func (f *fakeSource) Read(_ context.Context, _ types.BackupConfig, filename string) ([]byte, error) {
	if err, ok := f.readErr[filename]; ok {
		return nil, err
	}
	data, ok := f.files[filename]
	if !ok {
		return nil, fmt.Errorf("fakeSource: no such file %q", filename)
	}
	return data, nil
}

type fakeStore struct {
	meals        []types.Meal
	rollups      []types.DailyRollup
	weight       []types.WeightEntry
	measurements []types.MeasurementEntry
	sleep        []types.SleepLog
	workouts     []types.Workout
	water        []types.WaterLog
	fasts        []types.Fast
	photos       []types.ProgressPhoto
}

func (f *fakeStore) SaveMeal(_ context.Context, m types.Meal) error {
	f.meals = append(f.meals, m)
	return nil
}

func (f *fakeStore) UpsertRollup(_ context.Context, r types.DailyRollup) error {
	f.rollups = append(f.rollups, r)
	return nil
}

func (f *fakeStore) LogWeight(_ context.Context, w types.WeightEntry) (string, error) {
	f.weight = append(f.weight, w)
	return w.ID, nil
}

func (f *fakeStore) LogMeasurement(_ context.Context, m types.MeasurementEntry) (string, error) {
	f.measurements = append(f.measurements, m)
	return m.ID, nil
}

func (f *fakeStore) RestoreSleep(_ context.Context, sl types.SleepLog) error {
	f.sleep = append(f.sleep, sl)
	return nil
}

func (f *fakeStore) ImportWorkout(_ context.Context, w types.Workout) error {
	f.workouts = append(f.workouts, w)
	return nil
}

func (f *fakeStore) RestorePhoto(_ context.Context, p types.ProgressPhoto) error {
	f.photos = append(f.photos, p)
	return nil
}

func (f *fakeStore) RestoreWater(_ context.Context, w types.WaterLog) error {
	f.water = append(f.water, w)
	return nil
}

func (f *fakeStore) RestoreFast(_ context.Context, fs types.Fast) error {
	f.fasts = append(f.fasts, fs)
	return nil
}

// buildBackupFiles renders a small (1-2 row) valid backup using the real
// exportfmt writers, so fixtures stay in lockstep with the actual format.
func buildBackupFiles() map[string][]byte {
	files := map[string][]byte{}

	var mealsBuf bytes.Buffer
	_ = exportfmt.WriteMealsCSV(&mealsBuf, []types.Meal{
		{ID: "meal1", At: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC), RawText: "oatmeal",
			Items: []types.ResolvedItem{{Macros: types.Macros{Calories: 300, Protein: 10, Carbs: 50, Fat: 5, Fiber: 4}}}},
	})
	files["meals.csv"] = mealsBuf.Bytes()

	var rollupsBuf bytes.Buffer
	_ = exportfmt.WriteRollupsCSV(&rollupsBuf, []types.DailyRollup{
		{Date: "2026-01-01", Consumed: types.Macros{Calories: 300}, Targets: types.Macros{Calories: 2000}},
	})
	files["rollups.csv"] = rollupsBuf.Bytes()

	var weightBuf bytes.Buffer
	_ = exportfmt.WriteWeightCSV(&weightBuf, []types.WeightEntry{
		{ID: "w1", Date: "2026-01-01", WeightKg: 80.5, Note: "morning"},
	})
	files["weight.csv"] = weightBuf.Bytes()

	var measurementsBuf bytes.Buffer
	_ = exportfmt.WriteMeasurementsCSV(&measurementsBuf, []types.MeasurementEntry{
		{ID: "me1", Date: "2026-01-01", WaistCm: 80, HipsCm: 90},
	})
	files["measurements.csv"] = measurementsBuf.Bytes()

	var sleepBuf bytes.Buffer
	_ = exportfmt.WriteSleepCSV(&sleepBuf, []types.SleepLog{
		{ID: "s1", SleepAt: "2026-01-01T00:00:00Z", WakeAt: new("2026-01-01T07:00:00Z"), Quality: "good"},
	})
	files["sleep.csv"] = sleepBuf.Bytes()

	var workoutsBuf bytes.Buffer
	_ = exportfmt.WriteWorkoutsCSV(&workoutsBuf, []types.Workout{
		{ID: "wk1", Name: "run", DurationMin: 30, Intensity: "medium", LoggedAt: "2026-01-01T00:00:00Z"},
	})
	files["workouts.csv"] = workoutsBuf.Bytes()

	var waterBuf bytes.Buffer
	_ = exportfmt.WriteWaterCSV(&waterBuf, []types.WaterLog{
		{ID: "wa1", AmountML: 250, LoggedAt: "2026-01-01T00:00:00Z"},
	})
	files["water.csv"] = waterBuf.Bytes()

	var fastsBuf bytes.Buffer
	_ = exportfmt.WriteFastsCSV(&fastsBuf, []types.Fast{
		{ID: "f1", StartAt: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC), TargetHours: 16},
	})
	files["fasts.csv"] = fastsBuf.Bytes()

	var photosBuf bytes.Buffer
	_ = exportfmt.WritePhotosCSV(&photosBuf, []types.ProgressPhoto{
		{ID: "p1", Date: "2026-01-01", View: "front", MimeType: "image/jpeg"},
	})
	files["photos.csv"] = photosBuf.Bytes()
	files[exportfmt.PhotoFilename("p1")] = []byte("fake-jpeg-bytes")

	return files
}

func allFilenames() []string {
	return []string{
		"meals.csv", "rollups.csv", "weight.csv", "measurements.csv",
		"sleep.csv", "workouts.csv", "water.csv", "fasts.csv", "photos.csv",
	}
}

func TestRunOnce_FullRestore(t *testing.T) {
	src := &fakeSource{files: buildBackupFiles(), list: allFilenames()}
	store := &fakeStore{}
	r := New(store, src)

	sum, err := r.RunOnce(context.Background(), "u1", types.BackupConfig{})
	if err != nil {
		t.Fatalf("RunOnce: %v", err)
	}
	want := Summary{Meals: 1, Rollups: 1, Weight: 1, Measurements: 1, Sleep: 1, Workouts: 1, Water: 1, Fasts: 1, Photos: 1}
	if !reflect.DeepEqual(sum, want) {
		t.Fatalf("summary = %+v, want %+v", sum, want)
	}
	if len(sum.Skipped) != 0 {
		t.Fatalf("expected no skipped files, got %v", sum.Skipped)
	}
	if store.meals[0].UserID != "u1" || store.photos[0].UserID != "u1" {
		t.Fatalf("expected UserID stamped on restored rows")
	}
	// Every entity's UserID must be stamped to the restoring user, not just
	// the two spot-checked above.
	if store.rollups[0].UserID != "u1" || store.weight[0].UserID != "u1" ||
		store.measurements[0].UserID != "u1" || store.sleep[0].UserID != "u1" ||
		store.workouts[0].UserID != "u1" || store.water[0].UserID != "u1" ||
		store.fasts[0].UserID != "u1" {
		t.Fatalf("expected UserID stamped on every restored entity, got: rollups=%q weight=%q measurements=%q sleep=%q workouts=%q water=%q fasts=%q",
			store.rollups[0].UserID, store.weight[0].UserID, store.measurements[0].UserID,
			store.sleep[0].UserID, store.workouts[0].UserID, store.water[0].UserID, store.fasts[0].UserID)
	}
	if string(store.photos[0].Data) != "fake-jpeg-bytes" {
		t.Fatalf("expected photo blob restored, got %q", store.photos[0].Data)
	}
}

func TestRunOnce_MissingFilesSkipped(t *testing.T) {
	src := &fakeSource{files: buildBackupFiles(), list: []string{"meals.csv", "rollups.csv"}}
	store := &fakeStore{}
	r := New(store, src)

	sum, err := r.RunOnce(context.Background(), "u1", types.BackupConfig{})
	if err != nil {
		t.Fatalf("RunOnce: %v", err)
	}
	if sum.Meals != 1 || sum.Rollups != 1 {
		t.Fatalf("expected meals/rollups restored, got %+v", sum)
	}
	wantSkipped := []string{"weight.csv", "measurements.csv", "sleep.csv", "workouts.csv", "water.csv", "fasts.csv", "photos.csv"}
	for _, f := range wantSkipped {
		found := false
		for _, s := range sum.Skipped {
			if s == f {
				found = true
			}
		}
		if !found {
			t.Fatalf("expected %s in Skipped, got %v", f, sum.Skipped)
		}
	}
}

func TestRunOnce_CorruptFileDoesNotBlockOthers(t *testing.T) {
	files := buildBackupFiles()
	files["weight.csv"] = []byte("bad,header\n1,2\n") // wrong header -> parse failure
	src := &fakeSource{files: files, list: allFilenames()}
	store := &fakeStore{}
	r := New(store, src)

	sum, err := r.RunOnce(context.Background(), "u1", types.BackupConfig{})
	if err == nil {
		t.Fatalf("expected error mentioning weight.csv")
	}
	if !strings.Contains(err.Error(), "weight.csv") {
		t.Fatalf("expected error to mention weight.csv, got %v", err)
	}
	if sum.Weight != 0 {
		t.Fatalf("expected weight restore count 0, got %d", sum.Weight)
	}
	// Every other entity still restored despite weight.csv failing to parse.
	if sum.Meals != 1 || sum.Rollups != 1 || sum.Measurements != 1 || sum.Sleep != 1 ||
		sum.Workouts != 1 || sum.Water != 1 || sum.Fasts != 1 || sum.Photos != 1 {
		t.Fatalf("expected other entities unaffected, got %+v", sum)
	}
}

func TestRunOnce_IdempotentRerun(t *testing.T) {
	src := &fakeSource{files: buildBackupFiles(), list: allFilenames()}
	store := &fakeStore{}
	r := New(store, src)

	sum1, err := r.RunOnce(context.Background(), "u1", types.BackupConfig{})
	if err != nil {
		t.Fatalf("first RunOnce: %v", err)
	}
	sum2, err := r.RunOnce(context.Background(), "u1", types.BackupConfig{})
	if err != nil {
		t.Fatalf("second RunOnce: %v", err)
	}
	if !reflect.DeepEqual(sum1, sum2) {
		t.Fatalf("expected same summary on rerun, got %+v then %+v", sum1, sum2)
	}
}

func TestRunOnce_MissingPhotoBlobIsNonFatal(t *testing.T) {
	files := buildBackupFiles()
	src := &fakeSource{
		files:   files,
		list:    allFilenames(),
		readErr: map[string]error{exportfmt.PhotoFilename("p1"): fmt.Errorf("blob not found")},
	}
	store := &fakeStore{}
	r := New(store, src)

	sum, err := r.RunOnce(context.Background(), "u1", types.BackupConfig{})
	if err == nil {
		t.Fatalf("expected non-nil error mentioning the missing photo blob")
	}
	if sum.Photos != 0 {
		t.Fatalf("expected photo restore count 0 for missing blob, got %d", sum.Photos)
	}
	// Every other entity still fully restored.
	if sum.Meals != 1 || sum.Rollups != 1 || sum.Weight != 1 || sum.Measurements != 1 ||
		sum.Sleep != 1 || sum.Workouts != 1 || sum.Water != 1 || sum.Fasts != 1 {
		t.Fatalf("expected other entities unaffected, got %+v", sum)
	}
}
