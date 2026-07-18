package exportfmt

import (
	"bytes"
	"testing"
	"time"

	"github.com/gsaraiva2109/dietdaemon/core/types"
)

func TestMealsCSVRoundTrip(t *testing.T) {
	meals := []types.Meal{
		{
			ID:      "m1",
			At:      time.Date(2026, 7, 15, 0, 0, 0, 0, time.UTC),
			RawText: `chicken, "extra" rice`,
			Items: []types.ResolvedItem{
				{Macros: types.Macros{Calories: 500, Protein: 40, Carbs: 50, Fat: 10, Fiber: 5}},
			},
		},
	}
	var buf bytes.Buffer
	if err := WriteMealsCSV(&buf, meals); err != nil {
		t.Fatalf("WriteMealsCSV: %v", err)
	}
	got, err := ReadMealsCSV(&buf)
	if err != nil {
		t.Fatalf("ReadMealsCSV: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("got %d meals, want 1", len(got))
	}
	if got[0].ID != "m1" || !got[0].At.Equal(meals[0].At) || got[0].RawText != meals[0].RawText {
		t.Errorf("got %+v, want id/at/rawtext matching %+v", got[0], meals[0])
	}
	if got[0].Total() != meals[0].Total() {
		t.Errorf("Total() = %+v, want %+v", got[0].Total(), meals[0].Total())
	}
}

func TestReadMealsCSV_LossyReconstruction(t *testing.T) {
	meals := []types.Meal{
		{
			ID:      "m1",
			At:      time.Date(2026, 7, 15, 0, 0, 0, 0, time.UTC),
			RawText: "chicken and rice",
			Items: []types.ResolvedItem{
				{Macros: types.Macros{Calories: 300, Protein: 30, Carbs: 20, Fat: 5, Fiber: 2}},
				{Macros: types.Macros{Calories: 200, Protein: 10, Carbs: 30, Fat: 5, Fiber: 3}},
			},
		},
		{
			ID:      "m2",
			At:      time.Date(2026, 7, 16, 0, 0, 0, 0, time.UTC),
			RawText: "eggs and toast",
			Items: []types.ResolvedItem{
				{Macros: types.Macros{Calories: 150, Protein: 12, Carbs: 5, Fat: 10, Fiber: 1}},
				{Macros: types.Macros{Calories: 120, Protein: 4, Carbs: 22, Fat: 2, Fiber: 1}},
				{Macros: types.Macros{Calories: 80, Protein: 2, Carbs: 15, Fat: 1, Fiber: 1}},
			},
		},
	}
	var buf bytes.Buffer
	if err := WriteMealsCSV(&buf, meals); err != nil {
		t.Fatalf("WriteMealsCSV: %v", err)
	}
	got, err := ReadMealsCSV(&buf)
	if err != nil {
		t.Fatalf("ReadMealsCSV: %v", err)
	}
	if len(got) != len(meals) {
		t.Fatalf("got %d meals, want %d", len(got), len(meals))
	}
	for i, m := range got {
		if len(m.Items) != 1 {
			t.Errorf("meal %d: got %d items, want exactly 1", i, len(m.Items))
		}
		if m.Total() != meals[i].Total() {
			t.Errorf("meal %d: Total() = %+v, want %+v", i, m.Total(), meals[i].Total())
		}
	}
}

func TestRollupsCSVRoundTrip(t *testing.T) {
	rollups := []types.DailyRollup{
		{
			Date:     "2026-07-15",
			Consumed: types.Macros{Calories: 2200, Protein: 150.5, Carbs: 200.1, Fat: 60.2, Fiber: 25.3},
			Targets:  types.Macros{Calories: 2400, Protein: 160, Carbs: 220, Fat: 70, Fiber: 30},
		},
	}
	var buf bytes.Buffer
	if err := WriteRollupsCSV(&buf, rollups); err != nil {
		t.Fatalf("WriteRollupsCSV: %v", err)
	}
	got, err := ReadRollupsCSV(&buf)
	if err != nil {
		t.Fatalf("ReadRollupsCSV: %v", err)
	}
	if len(got) != 1 || got[0].Date != rollups[0].Date {
		t.Fatalf("got %+v, want date %s", got, rollups[0].Date)
	}
	if got[0].Consumed != rollups[0].Consumed || got[0].Targets != rollups[0].Targets {
		t.Errorf("got %+v, want %+v", got[0], rollups[0])
	}
}

func TestWeightCSVRoundTrip(t *testing.T) {
	entries := []types.WeightEntry{
		{ID: "w1", Date: "2026-07-15", WeightKg: 82.35, Note: `feeling "great" today, up a bit`},
	}
	var buf bytes.Buffer
	if err := WriteWeightCSV(&buf, entries); err != nil {
		t.Fatalf("WriteWeightCSV: %v", err)
	}
	got, err := ReadWeightCSV(&buf)
	if err != nil {
		t.Fatalf("ReadWeightCSV: %v", err)
	}
	if len(got) != 1 || got[0] != entries[0] {
		t.Errorf("got %+v, want %+v", got, entries)
	}
}

func TestMeasurementsCSVRoundTrip(t *testing.T) {
	entries := []types.MeasurementEntry{
		{
			ID: "meas1", Date: "2026-07-15",
			WaistCm: 80.5, HipsCm: 95.25, ChestCm: 100.1,
			LeftArmCm: 30.2, RightArmCm: 30.4,
			LeftThighCm: 55.6, RightThighCm: 55.8,
			Note: "post-workout",
		},
	}
	var buf bytes.Buffer
	if err := WriteMeasurementsCSV(&buf, entries); err != nil {
		t.Fatalf("WriteMeasurementsCSV: %v", err)
	}
	got, err := ReadMeasurementsCSV(&buf)
	if err != nil {
		t.Fatalf("ReadMeasurementsCSV: %v", err)
	}
	if len(got) != 1 || got[0] != entries[0] {
		t.Errorf("got %+v, want %+v", got, entries)
	}
}

func TestSleepCSVRoundTrip(t *testing.T) {
	wake := "2026-07-15 07:00:00"
	entries := []types.SleepLog{
		{ID: "s1", SleepAt: "2026-07-14 23:00:00", WakeAt: &wake, Quality: "good", Note: "slept well"},
		{ID: "s2", SleepAt: "2026-07-15 23:00:00", WakeAt: nil, Quality: "", Note: ""},
	}
	var buf bytes.Buffer
	if err := WriteSleepCSV(&buf, entries); err != nil {
		t.Fatalf("WriteSleepCSV: %v", err)
	}
	got, err := ReadSleepCSV(&buf)
	if err != nil {
		t.Fatalf("ReadSleepCSV: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("got %d entries, want 2", len(got))
	}
	if got[0].ID != entries[0].ID || got[0].WakeAt == nil || *got[0].WakeAt != wake || got[0].Quality != "good" || got[0].Note != "slept well" {
		t.Errorf("row 0: got %+v", got[0])
	}
	if got[1].WakeAt != nil {
		t.Errorf("row 1: WakeAt = %v, want nil", *got[1].WakeAt)
	}
}

func TestWorkoutsCSVRoundTrip(t *testing.T) {
	sets, reps := 3, 10
	weight := 60.5
	calories := 350
	extID := "hevy-123"
	workouts := []types.Workout{
		{
			ID: "wo1", Name: "Leg Day, heavy", DurationMin: 45, Intensity: "high",
			CaloriesBurned: &calories, Note: `felt "strong"`, LoggedAt: "2026-07-15 18:00:00",
			ExternalID: &extID,
			Exercises: []types.WorkoutExercise{
				{ID: "ex1", WorkoutID: "wo1", Name: "Squat", Sets: &sets, Reps: &reps, WeightKg: &weight, Note: "PR"},
			},
		},
		{
			ID: "wo2", Name: "Rest day walk", DurationMin: 20, Intensity: "low",
			LoggedAt: "2026-07-16 08:00:00",
		},
	}
	var buf bytes.Buffer
	if err := WriteWorkoutsCSV(&buf, workouts); err != nil {
		t.Fatalf("WriteWorkoutsCSV: %v", err)
	}
	got, err := ReadWorkoutsCSV(&buf)
	if err != nil {
		t.Fatalf("ReadWorkoutsCSV: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("got %d workouts, want 2", len(got))
	}
	if got[0].Name != workouts[0].Name || got[0].Note != workouts[0].Note ||
		got[0].CaloriesBurned == nil || *got[0].CaloriesBurned != calories ||
		got[0].ExternalID == nil || *got[0].ExternalID != extID {
		t.Errorf("workout 0: got %+v", got[0])
	}
	if len(got[0].Exercises) != 1 || got[0].Exercises[0].Name != "Squat" ||
		*got[0].Exercises[0].Sets != sets || *got[0].Exercises[0].Reps != reps || *got[0].Exercises[0].WeightKg != weight {
		t.Errorf("workout 0 exercises: got %+v", got[0].Exercises)
	}
	if got[1].CaloriesBurned != nil || got[1].ExternalID != nil || len(got[1].Exercises) != 0 {
		t.Errorf("workout 1: got %+v, want nil-able fields nil/empty", got[1])
	}
}

func TestWaterCSVRoundTrip(t *testing.T) {
	logs := []types.WaterLog{
		{ID: "wa1", AmountML: 250, LoggedAt: "2026-07-15 10:00:00", Note: "post-run"},
	}
	var buf bytes.Buffer
	if err := WriteWaterCSV(&buf, logs); err != nil {
		t.Fatalf("WriteWaterCSV: %v", err)
	}
	got, err := ReadWaterCSV(&buf)
	if err != nil {
		t.Fatalf("ReadWaterCSV: %v", err)
	}
	if len(got) != 1 || got[0] != logs[0] {
		t.Errorf("got %+v, want %+v", got, logs)
	}
}

func TestFastsCSVRoundTrip(t *testing.T) {
	start := time.Date(2026, 7, 14, 20, 0, 0, 0, time.UTC)
	end := time.Date(2026, 7, 15, 12, 0, 0, 0, time.UTC)
	fasts := []types.Fast{
		{ID: "f1", StartAt: start, EndAt: &end, TargetHours: 16, Completed: true},
		{ID: "f2", StartAt: start, EndAt: nil, TargetHours: 18.5, Completed: false},
	}
	var buf bytes.Buffer
	if err := WriteFastsCSV(&buf, fasts); err != nil {
		t.Fatalf("WriteFastsCSV: %v", err)
	}
	got, err := ReadFastsCSV(&buf)
	if err != nil {
		t.Fatalf("ReadFastsCSV: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("got %d fasts, want 2", len(got))
	}
	if !got[0].StartAt.Equal(start) || got[0].EndAt == nil || !got[0].EndAt.Equal(end) || got[0].TargetHours != 16 || !got[0].Completed {
		t.Errorf("fast 0: got %+v", got[0])
	}
	if !got[1].StartAt.Equal(start) || got[1].EndAt != nil || got[1].TargetHours != 18.5 || got[1].Completed {
		t.Errorf("fast 1: got %+v", got[1])
	}
}

func TestPhotosCSVRoundTrip(t *testing.T) {
	photos := []types.ProgressPhoto{
		{ID: "p1", Date: "2026-07-15", View: "front", MimeType: "image/jpeg", Data: []byte("blob-bytes-not-written")},
	}
	var buf bytes.Buffer
	if err := WritePhotosCSV(&buf, photos); err != nil {
		t.Fatalf("WritePhotosCSV: %v", err)
	}
	got, err := ReadPhotosCSV(&buf)
	if err != nil {
		t.Fatalf("ReadPhotosCSV: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("got %d photos, want 1", len(got))
	}
	p := got[0].Photo
	if p.ID != "p1" || p.Date != "2026-07-15" || p.View != "front" || p.MimeType != "image/jpeg" ||
		len(p.Data) != 0 || got[0].Filename != PhotoFilename("p1") {
		t.Errorf("got %+v, want id=p1 date=2026-07-15 view=front mime=image/jpeg data=empty filename=%s", got[0], PhotoFilename("p1"))
	}
}
