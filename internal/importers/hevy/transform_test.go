package hevy

import (
	"encoding/json"
	"testing"
	"time"
)

func TestToWorkout(t *testing.T) {
	hw := HevyWorkout{
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
					{Index: 0, Type: "warmup", WeightKg: floatPtr(40), Reps: intPtr(10)},
					{Index: 1, Type: "normal", WeightKg: floatPtr(80), Reps: intPtr(8)},
					{Index: 2, Type: "normal", WeightKg: floatPtr(90), Reps: intPtr(5)},
				},
			},
		},
	}

	w, err := ToWorkout("user-1", hw)
	if err != nil {
		t.Fatalf("ToWorkout: %v", err)
	}

	if w.Name != "Leg Day" {
		t.Errorf("Name = %q, want %q", w.Name, "Leg Day")
	}
	if w.DurationMin != 75 {
		t.Errorf("DurationMin = %d, want 75", w.DurationMin)
	}
	if w.ExternalID == nil || *w.ExternalID != "hw-123" {
		t.Errorf("ExternalID = %v, want hw-123", w.ExternalID)
	}
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

	// Note round-trips to original set data.
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
