package hevy

import (
	"encoding/json"
	"math"
	"testing"
	"time"

	"github.com/gsaraiva2109/dietdaemon/core/types"
)

// legDayWorkout is the fixture shared by TestToWorkout's assertion helpers:
// one exercise, three sets, weight ascending and reps descending so both the
// max-reps and max-weight aggregation branches are exercised.
func legDayWorkout() HevyWorkout {
	return HevyWorkout{
		ID:        "hw-123",
		Title:     "Leg Day",
		StartTime: time.Date(2025, 6, 15, 9, 0, 0, 0, time.UTC),
		EndTime:   time.Date(2025, 6, 15, 10, 15, 0, 0, time.UTC),
		Exercises: []HevyExercise{
			{
				Index:              0,
				Title:              "Squat",
				ExerciseTemplateID: "tmpl-1",
				Sets: []HevySet{
					{Index: 0, Type: "warmup", WeightKg: new(float64(40)), Reps: new(10)},
					{Index: 1, Type: "normal", WeightKg: new(float64(80)), Reps: new(8)},
					{Index: 2, Type: "normal", WeightKg: new(float64(90)), Reps: new(5)},
				},
			},
		},
	}
}

func TestToWorkout(t *testing.T) {
	w, err := ToWorkout("user-1", legDayWorkout())
	if err != nil {
		t.Fatalf("ToWorkout: %v", err)
	}

	assertWorkoutFields(t, w)
	ex := assertExerciseAggregation(t, w)
	assertNoteRoundTrips(t, ex)
}

func assertWorkoutFields(t *testing.T, w types.Workout) {
	t.Helper()
	if w.Name != "Leg Day" {
		t.Errorf("Name = %q, want %q", w.Name, "Leg Day")
	}
	if w.DurationMin != 75 {
		t.Errorf("DurationMin = %d, want 75", w.DurationMin)
	}
	if w.ExternalID == nil || *w.ExternalID != "hw-123" {
		t.Errorf("ExternalID = %v, want hw-123", w.ExternalID)
	}
}

func assertExerciseAggregation(t *testing.T, w types.Workout) types.WorkoutExercise {
	t.Helper()
	if len(w.Exercises) != 1 {
		t.Fatalf("len(Exercises) = %d, want 1", len(w.Exercises))
	}

	ex := w.Exercises[0]
	if ex.Name != "Squat" {
		t.Errorf("exercise name = %q, want Squat", ex.Name)
	}
	if ex.Sets == nil || *ex.Sets != 3 {
		t.Errorf("sets = %v, want 3", ex.Sets)
	}
	if ex.Reps == nil || *ex.Reps != 10 {
		t.Errorf("reps = %v, want 10", ex.Reps)
	}
	if ex.WeightKg == nil || *ex.WeightKg != 90 {
		t.Errorf("weight_kg = %v, want 90", ex.WeightKg)
	}
	return ex
}

// assertNoteRoundTrips checks that Note round-trips to the original set data.
func assertNoteRoundTrips(t *testing.T, ex types.WorkoutExercise) {
	t.Helper()
	var roundTripped []HevySet
	if err := json.Unmarshal([]byte(ex.Note), &roundTripped); err != nil {
		t.Fatalf("unmarshal note: %v", err)
	}
	if len(roundTripped) != 3 {
		t.Fatalf("round-tripped sets len = %d, want 3", len(roundTripped))
	}
	if roundTripped[1].Type != "normal" || *roundTripped[1].WeightKg != 80 {
		t.Errorf("round-tripped set[1] = %+v, want type=normal weight=80", roundTripped[1])
	}
}

func TestToWorkoutNilSafety(t *testing.T) {
	// All-nil set data — shouldn't panic.
	hw := HevyWorkout{
		ID:        "hw-nil",
		Title:     "Empty",
		StartTime: time.Date(2025, 6, 15, 12, 0, 0, 0, time.UTC),
		EndTime:   time.Date(2025, 6, 15, 12, 1, 0, 0, time.UTC),
		Exercises: []HevyExercise{
			{
				Index: 0,
				Title: "Test",
				Sets: []HevySet{
					{Index: 0, Type: "normal", WeightKg: nil, Reps: nil},
				},
			},
		},
	}
	w, err := ToWorkout("user-1", hw)
	if err != nil {
		t.Fatalf("ToWorkout: %v", err)
	}
	ex := w.Exercises[0]
	if ex.Reps != nil {
		t.Errorf("reps should be nil, got %v", ex.Reps)
	}
	if ex.WeightKg != nil {
		t.Errorf("weight should be nil, got %v", ex.WeightKg)
	}
}

func TestToWorkoutNegativeDuration(t *testing.T) {
	// EndTime before StartTime (clock skew / bad export data) clamps to 0
	// rather than going negative.
	hw := HevyWorkout{
		ID:        "hw-clock-skew",
		Title:     "Odd Export",
		StartTime: time.Date(2025, 6, 15, 12, 0, 0, 0, time.UTC),
		EndTime:   time.Date(2025, 6, 15, 11, 0, 0, 0, time.UTC),
	}
	w, err := ToWorkout("user-1", hw)
	if err != nil {
		t.Fatalf("ToWorkout: %v", err)
	}
	if w.DurationMin != 0 {
		t.Errorf("DurationMin = %d, want 0", w.DurationMin)
	}
}

func TestToWorkoutMarshalError(t *testing.T) {
	// json.Marshal rejects NaN/Inf floats, giving a real (if rare) way to
	// exercise ToWorkout's error path end to end.
	nan := math.NaN()
	hw := HevyWorkout{
		ID:        "hw-bad-weight",
		Title:     "Bad Data",
		StartTime: time.Date(2025, 6, 15, 9, 0, 0, 0, time.UTC),
		EndTime:   time.Date(2025, 6, 15, 9, 30, 0, 0, time.UTC),
		Exercises: []HevyExercise{
			{
				Index: 0,
				Title: "Bench Press",
				Sets: []HevySet{
					{Index: 0, Type: "normal", WeightKg: &nan, Reps: new(5)},
				},
			},
		},
	}
	if _, err := ToWorkout("user-1", hw); err == nil {
		t.Fatal("expected error for unmarshalable set data, got nil")
	}
}
