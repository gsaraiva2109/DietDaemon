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

	// details is keyed by food_id; a missing key means "not in this user's
	// library" (types.ErrNotFound), forcing the GetFood catalog fallback.
	details map[string]types.FoodDetail
	// catalog is keyed by food_id; a missing key means types.ErrNoMatch.
	catalog map[string]types.FoodMatch
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

func (f *fakeStore) GetFoodDetail(ctx context.Context, userID, foodID string) (types.FoodDetail, error) {
	if d, ok := f.details[foodID]; ok {
		return d, nil
	}
	return types.FoodDetail{}, types.ErrNotFound
}

func (f *fakeStore) GetFoodForUser(ctx context.Context, _ string, foodID string) (types.FoodMatch, error) {
	if m, ok := f.catalog[foodID]; ok {
		return m, nil
	}
	return types.FoodMatch{}, types.ErrNoMatch
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

func TestSuggestFromIngredientsEmptyPoolShortCircuits(t *testing.T) {
	st := &fakeStore{
		rollup: types.DailyRollup{
			Targets:  types.Macros{Calories: 2000},
			Consumed: types.Macros{Calories: 500},
		},
		// Neither ID resolves via GetFoodDetail or GetFood.
	}
	called := false
	m := &fakeModel{completeResp: `{"message":"should not be used"}`}
	e := New(st, &recordingModel{fakeModel: m, called: &called}, time.UTC)

	got, err := e.SuggestFromIngredients(t.Context(), "user1", []string{"nope1", "nope2"})
	if err != nil {
		t.Fatalf("SuggestFromIngredients: %v", err)
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

func TestSuggestFromIngredientsFallsBackToCatalog(t *testing.T) {
	// "chicken" is in this user's logged library; "quinoa" never was but
	// exists in the global catalog. Both should end up in the pool.
	st := &fakeStore{
		rollup: types.DailyRollup{
			Targets:  types.Macros{Calories: 500, Protein: 40, Carbs: 20, Fat: 10},
			Consumed: types.Macros{},
		},
		details: map[string]types.FoodDetail{
			"chicken": {FoodID: "chicken", Name: "Chicken breast", Per100g: types.Macros{Calories: 165, Protein: 31, Carbs: 0, Fat: 4}},
		},
		catalog: map[string]types.FoodMatch{
			"quinoa": {FoodID: "quinoa", Name: "Quinoa", Per100g: types.Macros{Calories: 120, Protein: 4, Carbs: 21, Fat: 2}},
		},
	}
	m := &fakeModel{completeErr: context.DeadlineExceeded}
	e := New(st, m, time.UTC)

	got, err := e.SuggestFromIngredients(t.Context(), "user1", []string{"chicken", "quinoa", "unknown-id"})
	if err != nil {
		t.Fatalf("SuggestFromIngredients: %v", err)
	}
	if got.Source != "rules" {
		t.Errorf("Source = %q, want rules (model errored)", got.Source)
	}
	if len(got.Candidates) == 0 {
		t.Fatal("expected candidates built from the resolved chicken+quinoa pool")
	}

	var sawChicken, sawQuinoa, sawUnknown bool
	for _, c := range got.Candidates {
		for _, it := range c.Items {
			switch it.FoodID {
			case "chicken":
				sawChicken = true
			case "quinoa":
				sawQuinoa = true
			case "unknown-id":
				sawUnknown = true
			}
		}
	}
	if !sawChicken {
		t.Error("expected the logged food (chicken, via GetFoodDetail) to appear in candidates")
	}
	if !sawQuinoa {
		t.Error("expected the catalog-only food (quinoa, via GetFood fallback) to appear in candidates")
	}
	if sawUnknown {
		t.Error("unresolvable food ID should have been skipped, not surfaced in candidates")
	}
}

func TestSuggestFromIngredientsLLMRanks(t *testing.T) {
	st := &fakeStore{
		rollup: types.DailyRollup{
			Targets:  types.Macros{Calories: 500, Protein: 40, Carbs: 5, Fat: 10},
			Consumed: types.Macros{},
		},
		details: map[string]types.FoodDetail{
			"egg":     {FoodID: "egg", Name: "Egg", Per100g: types.Macros{Calories: 150, Protein: 13, Carbs: 1, Fat: 10}},
			"chicken": {FoodID: "chicken", Name: "Chicken breast", Per100g: types.Macros{Calories: 165, Protein: 31, Carbs: 0, Fat: 4}},
		},
	}
	m := &fakeModel{completeResp: `{"message":"Use what you've got: eggs and chicken."}`}
	e := New(st, m, time.UTC)

	got, err := e.SuggestFromIngredients(t.Context(), "user1", []string{"egg", "chicken"})
	if err != nil {
		t.Fatalf("SuggestFromIngredients: %v", err)
	}
	if got.Source != "llm" {
		t.Errorf("Source = %q, want llm", got.Source)
	}
	if got.Message != "Use what you've got: eggs and chicken." {
		t.Errorf("Message = %q, want the LLM-phrased suggestion", got.Message)
	}
}
