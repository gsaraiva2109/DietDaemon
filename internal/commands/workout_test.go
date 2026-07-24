package commands

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/gsaraiva2109/dietdaemon/core/types"
)

// fakeWorkoutStore is a minimal WorkoutStore stub for /workout tests.
type fakeWorkoutStore struct {
	workouts []types.Workout
	listErr  error
	logged   *types.Workout
	logErr   error
}

func (f *fakeWorkoutStore) LogWorkout(_ context.Context, w types.Workout) error {
	if f.logErr != nil {
		return f.logErr
	}
	f.logged = &w
	return nil
}

func (f *fakeWorkoutStore) ListWorkouts(_ context.Context, _ string, _ int) ([]types.Workout, error) {
	return f.workouts, f.listErr
}

func TestWorkoutCommand_UsageOnEmpty(t *testing.T) {
	cmd := NewWorkoutCommand(&fakeWorkoutStore{})
	reply, err := cmd.Handle(context.Background(), types.InboundMessage{UserID: "u1"}, "")
	if err != nil {
		t.Fatalf("Handle: %v", err)
	}
	if !strings.Contains(reply.Text, "Usage:") {
		t.Errorf("expected usage reply, got %q", reply.Text)
	}
}

func TestWorkoutCommand_ListEmpty(t *testing.T) {
	cmd := NewWorkoutCommand(&fakeWorkoutStore{})
	reply, err := cmd.Handle(context.Background(), types.InboundMessage{UserID: "u1"}, "list")
	if err != nil {
		t.Fatalf("Handle: %v", err)
	}
	if reply.Text != "No workouts logged yet." {
		t.Errorf("reply.Text = %q, want no-workouts message", reply.Text)
	}
}

func TestWorkoutCommand_ListWithEntries(t *testing.T) {
	cal := 300
	store := &fakeWorkoutStore{workouts: []types.Workout{
		{Name: "Bench Press", DurationMin: 45, Intensity: "heavy", CaloriesBurned: &cal},
		{Name: "Yoga", DurationMin: 30, Intensity: "light"},
	}}
	cmd := NewWorkoutCommand(store)

	reply, err := cmd.Handle(context.Background(), types.InboundMessage{UserID: "u1"}, "list")
	if err != nil {
		t.Fatalf("Handle: %v", err)
	}
	if !strings.Contains(reply.Text, "Bench Press — 45 min, heavy (~300 kcal)") {
		t.Errorf("expected first entry line, got %q", reply.Text)
	}
	if !strings.Contains(reply.Text, "Yoga — 30 min, light\n") {
		t.Errorf("expected second entry line without kcal, got %q", reply.Text)
	}
}

func TestWorkoutCommand_LogNoNumericToken(t *testing.T) {
	cmd := NewWorkoutCommand(&fakeWorkoutStore{})
	reply, err := cmd.Handle(context.Background(), types.InboundMessage{UserID: "u1"}, "Bench Press")
	if err != nil {
		t.Fatalf("Handle: %v", err)
	}
	if !strings.Contains(reply.Text, "Usage:") {
		t.Errorf("expected usage reply, got %q", reply.Text)
	}
}

func TestWorkoutCommand_LogNameOnlyDigitFirst(t *testing.T) {
	// A duration with nothing before it (no name) must be rejected.
	cmd := NewWorkoutCommand(&fakeWorkoutStore{})
	reply, err := cmd.Handle(context.Background(), types.InboundMessage{UserID: "u1"}, "45")
	if err != nil {
		t.Fatalf("Handle: %v", err)
	}
	if !strings.Contains(reply.Text, "Usage:") {
		t.Errorf("expected usage reply, got %q", reply.Text)
	}
}

func TestWorkoutCommand_LogInvalidDuration(t *testing.T) {
	cmd := NewWorkoutCommand(&fakeWorkoutStore{})
	reply, err := cmd.Handle(context.Background(), types.InboundMessage{UserID: "u1"}, "Bench Press 5000")
	if err != nil {
		t.Fatalf("Handle: %v", err)
	}
	if !strings.Contains(reply.Text, "Invalid duration") {
		t.Errorf("expected invalid-duration reply, got %q", reply.Text)
	}
}

func TestWorkoutCommand_LogNameAndDurationOnly(t *testing.T) {
	store := &fakeWorkoutStore{}
	cmd := NewWorkoutCommand(store)

	reply, err := cmd.Handle(context.Background(), types.InboundMessage{UserID: "u1"}, "Bench Press 45")
	if err != nil {
		t.Fatalf("Handle: %v", err)
	}
	if store.logged == nil {
		t.Fatal("expected LogWorkout to be called")
	}
	if store.logged.Name != "Bench Press" {
		t.Errorf("Name = %q, want %q", store.logged.Name, "Bench Press")
	}
	if store.logged.DurationMin != 45 {
		t.Errorf("DurationMin = %d, want 45", store.logged.DurationMin)
	}
	if store.logged.Intensity != "moderate" {
		t.Errorf("Intensity = %q, want moderate (default)", store.logged.Intensity)
	}
	if store.logged.Note != "" {
		t.Errorf("Note = %q, want empty", store.logged.Note)
	}
	if !strings.Contains(reply.Text, "Bench Press — 45 min, moderate") {
		t.Errorf("expected confirmation, got %q", reply.Text)
	}
}

func TestWorkoutCommand_LogWithIntensityAndNote(t *testing.T) {
	store := &fakeWorkoutStore{}
	cmd := NewWorkoutCommand(store)

	reply, err := cmd.Handle(context.Background(), types.InboundMessage{UserID: "u1"}, "Bench Press 45 heavy new PR today")
	if err != nil {
		t.Fatalf("Handle: %v", err)
	}
	if store.logged.Intensity != "heavy" {
		t.Errorf("Intensity = %q, want heavy", store.logged.Intensity)
	}
	if store.logged.Note != "new PR today" {
		t.Errorf("Note = %q, want %q", store.logged.Note, "new PR today")
	}
	if !strings.Contains(reply.Text, "heavy") {
		t.Errorf("expected intensity in reply, got %q", reply.Text)
	}
}

func TestWorkoutCommand_LogUnrecognizedIntensityBecomesNote(t *testing.T) {
	// A token after duration that isn't light/moderate/heavy is not an
	// intensity: it's swallowed into the note, and intensity stays default.
	store := &fakeWorkoutStore{}
	cmd := NewWorkoutCommand(store)

	_, err := cmd.Handle(context.Background(), types.InboundMessage{UserID: "u1"}, "Run 30 outside")
	if err != nil {
		t.Fatalf("Handle: %v", err)
	}
	if store.logged.Intensity != "moderate" {
		t.Errorf("Intensity = %q, want moderate (default)", store.logged.Intensity)
	}
	if store.logged.Note != "outside" {
		t.Errorf("Note = %q, want %q", store.logged.Note, "outside")
	}
}

func TestWorkoutCommand_MultiWordName(t *testing.T) {
	store := &fakeWorkoutStore{}
	cmd := NewWorkoutCommand(store)

	_, err := cmd.Handle(context.Background(), types.InboundMessage{UserID: "u1"}, "Evening Trail Run 60 light")
	if err != nil {
		t.Fatalf("Handle: %v", err)
	}
	if store.logged.Name != "Evening Trail Run" {
		t.Errorf("Name = %q, want %q", store.logged.Name, "Evening Trail Run")
	}
}

func TestWorkoutCommand_LogStoreError(t *testing.T) {
	store := &fakeWorkoutStore{logErr: errors.New("db down")}
	cmd := NewWorkoutCommand(store)

	_, err := cmd.Handle(context.Background(), types.InboundMessage{UserID: "u1"}, "Bench Press 45")
	if err == nil {
		t.Fatal("expected error when LogWorkout fails")
	}
}

func TestWorkoutCommand_Metadata(t *testing.T) {
	cmd := NewWorkoutCommand(&fakeWorkoutStore{})
	if cmd.Name() != "/workout" {
		t.Errorf("Name() = %q, want /workout", cmd.Name())
	}
	if cmd.Help() != types.I18nKey("cmd.workout.usage") {
		t.Errorf("Help() = %q, want cmd.workout.usage", cmd.Help())
	}
}
