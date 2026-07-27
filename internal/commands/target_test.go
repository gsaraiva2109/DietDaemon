package commands

import (
	"context"
	"strings"
	"testing"

	"github.com/gsaraiva2109/dietdaemon/core/types"
)

// fakeMealStore is a minimal stub implementing MealStore for /target tests.
type fakeMealStore struct {
	existingTargets types.DailyTargets
	getTargetsErr   error
	setTargetsErr   error
	savedTargets    types.DailyTargets
}

func (f *fakeMealStore) UpsertUser(_ context.Context, _ types.User) error { return nil }
func (f *fakeMealStore) GetUser(_ context.Context, _ string) (types.User, error) {
	return types.User{}, nil
}
func (f *fakeMealStore) SaveMeal(_ context.Context, _ types.Meal) error { return nil }
func (f *fakeMealStore) GetTargets(_ context.Context, _ string) (types.DailyTargets, error) {
	if f.getTargetsErr != nil {
		return types.DailyTargets{}, f.getTargetsErr
	}
	return f.existingTargets, nil
}
func (f *fakeMealStore) SetTargets(_ context.Context, t types.DailyTargets) error {
	if f.setTargetsErr != nil {
		return f.setTargetsErr
	}
	f.savedTargets = t
	return nil
}
func (f *fakeMealStore) GetRollup(_ context.Context, _, _ string) (types.DailyRollup, error) {
	return types.DailyRollup{}, nil
}
func (f *fakeMealStore) UpsertRollup(_ context.Context, _ types.DailyRollup) error { return nil }
func (f *fakeMealStore) GetUserIDByChannel(_ context.Context, _, _ string) (string, error) {
	return "", nil
}
func (f *fakeMealStore) MapChannelUser(_ context.Context, _, _, _ string) error { return nil }

func TestTargetCommand_EmptyArgsShowsUsage(t *testing.T) {
	cmd := NewTargetCommand(&fakeMealStore{})
	reply, err := cmd.Handle(context.Background(), types.InboundMessage{UserID: "u1"}, "")
	if err != nil {
		t.Fatalf("Handle: %v", err)
	}
	if !strings.Contains(reply.Text, "Usage: /target") {
		t.Errorf("expected usage text, got %q", reply.Text)
	}
}

func TestTargetCommand_UnrecognizedArgsShowsUsage(t *testing.T) {
	cmd := NewTargetCommand(&fakeMealStore{})
	reply, err := cmd.Handle(context.Background(), types.InboundMessage{UserID: "u1"}, "banana=yes")
	if err != nil {
		t.Fatalf("Handle: %v", err)
	}
	if !strings.Contains(reply.Text, "Usage: /target") {
		t.Errorf("expected usage text for unrecognized key, got %q", reply.Text)
	}
}

func TestTargetCommand_SetsTargetsPreservingWaterGoal(t *testing.T) {
	store := &fakeMealStore{existingTargets: types.DailyTargets{WaterGoalMl: 2000}}
	cmd := NewTargetCommand(store)

	reply, err := cmd.Handle(context.Background(), types.InboundMessage{UserID: "u1"}, "kcal=2200 protein=180 carbs=220 fat=70")
	if err != nil {
		t.Fatalf("Handle: %v", err)
	}
	if !strings.Contains(reply.Text, "Targets set: 2200 kcal") {
		t.Errorf("expected confirmation text, got %q", reply.Text)
	}
	if store.savedTargets.WaterGoalMl != 2000 {
		t.Errorf("expected existing water goal preserved, got %d", store.savedTargets.WaterGoalMl)
	}
	if store.savedTargets.Targets.Protein != 180 {
		t.Errorf("expected protein 180, got %f", store.savedTargets.Targets.Protein)
	}
}

func TestTargetCommand_SetTargetsError(t *testing.T) {
	store := &fakeMealStore{setTargetsErr: types.ErrNotFound}
	cmd := NewTargetCommand(store)

	_, err := cmd.Handle(context.Background(), types.InboundMessage{UserID: "u1"}, "kcal=2000")
	if err == nil {
		t.Fatal("expected error when SetTargets fails")
	}
}

func TestTargetCommand_NameHelp(t *testing.T) {
	cmd := NewTargetCommand(&fakeMealStore{})
	if cmd.Name() != "/target" {
		t.Errorf("Name() = %q, want /target", cmd.Name())
	}
	if cmd.Aliases() != nil {
		t.Errorf("Aliases() = %v, want nil", cmd.Aliases())
	}
	if cmd.Help() != "cmd.target.usage" {
		t.Errorf("Help() = %q, want cmd.target.usage", cmd.Help())
	}
}

func TestParseTargetArgs(t *testing.T) {
	cases := []struct {
		name    string
		args    string
		wantOK  bool
		wantVal float64
		field   func(types.Macros) float64
	}{
		{"kcal alias", "cal=1500", true, 1500, func(m types.Macros) float64 { return m.Calories }},
		{"protein short", "p=120", true, 120, func(m types.Macros) float64 { return m.Protein }},
		{"carbs short", "c=100", true, 100, func(m types.Macros) float64 { return m.Carbs }},
		{"fat short", "f=50", true, 50, func(m types.Macros) float64 { return m.Fat }},
		{"fiber", "fiber=30", true, 30, func(m types.Macros) float64 { return m.Fiber }},
		{"invalid number ignored", "kcal=notanumber", false, 0, func(m types.Macros) float64 { return m.Calories }},
		{"no equals ignored", "kcal", false, 0, func(m types.Macros) float64 { return m.Calories }},
		{"unknown key ignored", "banana=5", false, 0, func(m types.Macros) float64 { return m.Calories }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			m, ok := parseTargetArgs(tc.args)
			if ok != tc.wantOK {
				t.Fatalf("ok = %v, want %v", ok, tc.wantOK)
			}
			if ok && tc.field(m) != tc.wantVal {
				t.Fatalf("value = %f, want %f", tc.field(m), tc.wantVal)
			}
		})
	}
}
