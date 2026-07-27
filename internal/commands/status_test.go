package commands

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/gsaraiva2109/dietdaemon/core/types"
)

// fakeStatusStore is a minimal stub implementing StatusStore for /status tests.
type fakeStatusStore struct {
	user          types.User
	getUserErr    error
	targets       types.DailyTargets
	targetsErr    error
	rollup        types.DailyRollup
	rollupErr     error
	meals         []types.Meal
	recentMealErr error
}

func (f *fakeStatusStore) TargetsFor(_ context.Context, _, _ string) (types.DailyTargets, error) {
	if f.targetsErr != nil {
		return types.DailyTargets{}, f.targetsErr
	}
	return f.targets, nil
}

func (f *fakeStatusStore) GetRollup(_ context.Context, _, _ string) (types.DailyRollup, error) {
	if f.rollupErr != nil {
		return types.DailyRollup{}, f.rollupErr
	}
	return f.rollup, nil
}

func (f *fakeStatusStore) RecentMeals(_ context.Context, _ string, _ int) ([]types.Meal, error) {
	if f.recentMealErr != nil {
		return nil, f.recentMealErr
	}
	return f.meals, nil
}

func (f *fakeStatusStore) GetUser(_ context.Context, _ string) (types.User, error) {
	if f.getUserErr != nil {
		return types.User{}, f.getUserErr
	}
	return f.user, nil
}

func TestStatusCommand_NoTargets(t *testing.T) {
	store := &fakeStatusStore{targetsErr: types.ErrNotFound, getUserErr: types.ErrNotFound}
	cmd := NewStatusCommand(store, time.UTC)

	reply, err := cmd.Handle(context.Background(), types.InboundMessage{UserID: "u1"}, "")
	if err != nil {
		t.Fatalf("Handle: %v", err)
	}
	if !strings.Contains(reply.Text, "No targets set") {
		t.Errorf("expected no-targets reply, got %q", reply.Text)
	}
}

func TestStatusCommand_WithMealsAndUserTimezone(t *testing.T) {
	store := &fakeStatusStore{
		user:    types.User{ID: "u1", Timezone: "America/Sao_Paulo"},
		targets: types.DailyTargets{UserID: "u1", Targets: types.Macros{Calories: 2000, Protein: 150, Carbs: 200, Fat: 60}},
		rollup:  types.DailyRollup{Consumed: types.Macros{Calories: 500, Protein: 40, Carbs: 50, Fat: 15}},
		meals: []types.Meal{
			{RawText: "frango grelhado", Items: []types.ResolvedItem{{Macros: types.Macros{Calories: 300}}}},
		},
	}
	cmd := NewStatusCommand(store, time.UTC)

	reply, err := cmd.Handle(context.Background(), types.InboundMessage{UserID: "u1"}, "")
	if err != nil {
		t.Fatalf("Handle: %v", err)
	}
	if !strings.Contains(reply.Text, "Today's Summary") {
		t.Errorf("expected summary header, got %q", reply.Text)
	}
	if !strings.Contains(reply.Text, "Recent meals:") || !strings.Contains(reply.Text, "frango grelhado") {
		t.Errorf("expected recent meals section, got %q", reply.Text)
	}
}

func TestStatusCommand_NoRollupFallsBackToZeroConsumed(t *testing.T) {
	store := &fakeStatusStore{
		targets:   types.DailyTargets{UserID: "u1", Targets: types.Macros{Calories: 2000}},
		rollupErr: types.ErrNotFound,
	}
	cmd := NewStatusCommand(store, time.UTC)

	reply, err := cmd.Handle(context.Background(), types.InboundMessage{UserID: "u1"}, "")
	if err != nil {
		t.Fatalf("Handle: %v", err)
	}
	if !strings.Contains(reply.Text, "0 / 2000 kcal") {
		t.Errorf("expected zeroed consumed, got %q", reply.Text)
	}
	if !strings.Contains(reply.Text, "No meals logged today") {
		t.Errorf("expected no-meals nudge when calPct < 1, got %q", reply.Text)
	}
}

func TestStatusCommand_NameAliasesHelp(t *testing.T) {
	cmd := NewStatusCommand(&fakeStatusStore{}, time.UTC)
	if cmd.Name() != "/status" {
		t.Errorf("Name() = %q, want /status", cmd.Name())
	}
	if len(cmd.Aliases()) != 1 || cmd.Aliases()[0] != "/summary" {
		t.Errorf("Aliases() = %v, want [/summary]", cmd.Aliases())
	}
	if cmd.Help() != "cmd.status.title" {
		t.Errorf("Help() = %q, want cmd.status.title", cmd.Help())
	}
}
