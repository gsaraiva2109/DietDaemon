package suggest

import (
	"context"
	"testing"
	"time"

	"github.com/gsaraiva2109/dietdaemon/core/types"
)

type fakeStore struct {
	rollup     types.DailyRollup
	rollupErr  error
	targets    types.DailyTargets
	targetsErr error
	foods      []types.FoodDetail
	foodsErr   error
}

func (f *fakeStore) GetRollup(ctx context.Context, userID, localDate string) (types.DailyRollup, error) {
	return f.rollup, f.rollupErr
}

func (f *fakeStore) GetTargets(ctx context.Context, userID string) (types.DailyTargets, error) {
	return f.targets, f.targetsErr
}

func (f *fakeStore) FrequentFoods(ctx context.Context, userID string, limit int) ([]types.FoodDetail, error) {
	return f.foods, f.foodsErr
}

type fakeModel struct {
	completeResp string
	completeErr  error
}

func (f *fakeModel) Embed(ctx context.Context, text string) ([]float32, error) {
	return nil, nil
}

func (f *fakeModel) Complete(ctx context.Context, prompt string) (string, error) {
	return f.completeResp, f.completeErr
}

func testPool() []types.FoodDetail {
	return []types.FoodDetail{
		{FoodID: "egg", Name: "Egg", Per100g: types.Macros{Calories: 150, Protein: 13, Carbs: 1, Fat: 10}},
		{FoodID: "chicken", Name: "Chicken breast", Per100g: types.Macros{Calories: 165, Protein: 31, Carbs: 0, Fat: 4}},
	}
}

func TestSuggestFallsBackToTargetsWhenNoRollupYet(t *testing.T) {
	st := &fakeStore{
		rollupErr: types.ErrNotFound,
		targets:   types.DailyTargets{Targets: types.Macros{Calories: 2000, Protein: 150, Carbs: 200, Fat: 60}},
		foods:     testPool(),
	}
	m := &fakeModel{completeResp: `{"message":"Try eggs and chicken."}`}
	e := New(st, m, time.UTC)

	got, err := e.Suggest(t.Context(), "user1")
	if err != nil {
		t.Fatalf("Suggest: %v", err)
	}
	if got.Remaining != (types.Macros{Calories: 2000, Protein: 150, Carbs: 200, Fat: 60}) {
		t.Errorf("Remaining = %+v, want full targets (no consumption yet)", got.Remaining)
	}
	if got.Source != "llm" {
		t.Errorf("Source = %q, want llm", got.Source)
	}
}

func TestSuggestEmptyPoolShortCircuits(t *testing.T) {
	st := &fakeStore{
		rollup: types.DailyRollup{
			Targets:  types.Macros{Calories: 2000},
			Consumed: types.Macros{Calories: 500},
		},
		foods: nil,
	}
	called := false
	m := &fakeModel{completeResp: `{"message":"should not be used"}`}
	e := New(st, &recordingModel{fakeModel: m, called: &called}, time.UTC)

	got, err := e.Suggest(t.Context(), "user1")
	if err != nil {
		t.Fatalf("Suggest: %v", err)
	}
	if got.Source != "rules" {
		t.Errorf("Source = %q, want rules", got.Source)
	}
	if len(got.Candidates) != 0 {
		t.Errorf("Candidates = %v, want none", got.Candidates)
	}
	if called {
		t.Error("Complete was called on an empty pool, it should short-circuit before ranking")
	}
}

type recordingModel struct {
	*fakeModel
	called *bool
}

func (r *recordingModel) Complete(ctx context.Context, prompt string) (string, error) {
	*r.called = true
	return r.fakeModel.Complete(ctx, prompt)
}

func TestSuggestFallsBackToRulesOnModelError(t *testing.T) {
	st := &fakeStore{
		rollup: types.DailyRollup{
			Targets:  types.Macros{Calories: 500, Protein: 40, Carbs: 5, Fat: 10},
			Consumed: types.Macros{},
		},
		foods: testPool(),
	}
	m := &fakeModel{completeErr: context.DeadlineExceeded}
	e := New(st, m, time.UTC)

	got, err := e.Suggest(t.Context(), "user1")
	if err != nil {
		t.Fatalf("Suggest: %v", err)
	}
	if got.Source != "rules" {
		t.Errorf("Source = %q, want rules", got.Source)
	}
	if got.Message == "" {
		t.Error("expected a non-empty rule-based fallback message")
	}
	if len(got.Candidates) == 0 {
		t.Error("expected rule-based candidates even when the model errors")
	}
}

func TestSuggestHappyPath(t *testing.T) {
	st := &fakeStore{
		rollup: types.DailyRollup{
			Targets:  types.Macros{Calories: 500, Protein: 40, Carbs: 5, Fat: 10},
			Consumed: types.Macros{},
		},
		foods: testPool(),
	}
	m := &fakeModel{completeResp: `{"message":"Grab a chicken breast, it fits well."}`}
	e := New(st, m, time.UTC)

	got, err := e.Suggest(t.Context(), "user1")
	if err != nil {
		t.Fatalf("Suggest: %v", err)
	}
	if got.Source != "llm" {
		t.Errorf("Source = %q, want llm", got.Source)
	}
	if got.Message != "Grab a chicken breast, it fits well." {
		t.Errorf("Message = %q, want the LLM-phrased suggestion", got.Message)
	}
	if len(got.Candidates) == 0 {
		t.Error("expected candidates alongside the LLM message")
	}
}
