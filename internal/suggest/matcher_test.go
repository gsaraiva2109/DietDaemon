package suggest

import (
	"math"
	"testing"

	"github.com/gsaraiva2109/dietdaemon/core/types"
)

func approxEqual(a, b float64) bool {
	return math.Abs(a-b) < 0.01
}

func TestFindCombos_SingleBestKnownByConstruction(t *testing.T) {
	remaining := types.Macros{Calories: 300, Protein: 30, Carbs: 20, Fat: 10}

	best := types.FoodDetail{
		FoodID:  "best",
		Name:    "Perfect Fit",
		Per100g: types.Macros{Calories: 300, Protein: 30, Carbs: 20, Fat: 10},
	}
	decoy1 := types.FoodDetail{
		FoodID:  "decoy1",
		Name:    "Way Off",
		Per100g: types.Macros{Calories: 900, Protein: 2, Carbs: 100, Fat: 80},
	}
	decoy2 := types.FoodDetail{
		FoodID:  "decoy2",
		Name:    "Also Off",
		Per100g: types.Macros{Calories: 50, Protein: 1, Carbs: 5, Fat: 1},
	}

	pool := []types.FoodDetail{decoy1, best, decoy2}

	results := FindCombos(pool, remaining, 3)
	if len(results) == 0 {
		t.Fatal("expected at least one result")
	}

	top := results[0]
	if top.Score < 0.99 {
		t.Errorf("expected near-perfect score for best match, got %f", top.Score)
	}
	if len(top.Items) != 1 || top.Items[0].Food.FoodID != "best" {
		t.Fatalf("expected top result to be single item 'best', got %+v", top.Items)
	}
	if !approxEqual(top.Items[0].Grams, 100) {
		t.Errorf("expected multiplier 1.0 (100g), got %f", top.Items[0].Grams)
	}

	for _, other := range results[1:] {
		if other.Score >= top.Score {
			t.Errorf("expected top score %f to beat other score %f", top.Score, other.Score)
		}
	}
}

func TestFindCombos_TwoItemComboWins(t *testing.T) {
	remaining := types.Macros{Calories: 400, Protein: 40, Carbs: 30, Fat: 15}

	foodA := types.FoodDetail{
		FoodID:  "a",
		Name:    "Half A",
		Per100g: types.Macros{Calories: 250, Protein: 30, Carbs: 10, Fat: 5},
	}
	foodB := types.FoodDetail{
		FoodID:  "b",
		Name:    "Half B",
		Per100g: types.Macros{Calories: 150, Protein: 10, Carbs: 20, Fat: 10},
	}
	// A poor single-food decoy that should not beat the A+B combo.
	decoy := types.FoodDetail{
		FoodID:  "decoy",
		Name:    "Irrelevant",
		Per100g: types.Macros{Calories: 800, Protein: 5, Carbs: 90, Fat: 60},
	}

	pool := []types.FoodDetail{foodA, foodB, decoy}

	results := FindCombos(pool, remaining, 5)
	if len(results) == 0 {
		t.Fatal("expected at least one result")
	}

	top := results[0]
	if len(top.Items) != 2 {
		t.Fatalf("expected top result to have 2 items, got %d: %+v", len(top.Items), top.Items)
	}
	if !approxEqual(top.Macros.Calories, remaining.Calories) ||
		!approxEqual(top.Macros.Protein, remaining.Protein) ||
		!approxEqual(top.Macros.Carbs, remaining.Carbs) ||
		!approxEqual(top.Macros.Fat, remaining.Fat) {
		t.Errorf("expected top combo macros to approximately equal remaining, got %+v vs %+v", top.Macros, remaining)
	}
}

func TestFindCombos_EmptyPool(t *testing.T) {
	remaining := types.Macros{Calories: 300, Protein: 30, Carbs: 20, Fat: 10}
	results := FindCombos(nil, remaining, 5)
	if len(results) != 0 {
		t.Errorf("expected empty result for empty pool, got %d", len(results))
	}
}

func TestFindCombos_TopNSmallerThanAvailable(t *testing.T) {
	remaining := types.Macros{Calories: 300, Protein: 30, Carbs: 20, Fat: 10}
	pool := []types.FoodDetail{
		{FoodID: "1", Per100g: types.Macros{Calories: 100, Protein: 10, Carbs: 5, Fat: 2}},
		{FoodID: "2", Per100g: types.Macros{Calories: 200, Protein: 20, Carbs: 10, Fat: 5}},
		{FoodID: "3", Per100g: types.Macros{Calories: 300, Protein: 30, Carbs: 15, Fat: 8}},
	}

	results := FindCombos(pool, remaining, 3)
	if len(results) != 3 {
		t.Errorf("expected topN=3 results, got %d", len(results))
	}
}

func TestFindCombos_TopNLargerThanAvailable(t *testing.T) {
	remaining := types.Macros{Calories: 300, Protein: 30, Carbs: 20, Fat: 10}
	pool := []types.FoodDetail{
		{FoodID: "only", Per100g: types.Macros{Calories: 100, Protein: 10, Carbs: 5, Fat: 2}},
	}

	// 1 food x 4 multipliers = 4 possible combos total.
	results := FindCombos(pool, remaining, 50)
	if len(results) != 4 {
		t.Errorf("expected 4 combos (1 food x 4 multipliers), got %d", len(results))
	}
}
