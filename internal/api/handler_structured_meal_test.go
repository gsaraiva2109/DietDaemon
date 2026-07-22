package api

import (
	"net/http"
	"testing"

	"github.com/gsaraiva2109/dietdaemon/core/types"
)

func TestCreateStructuredMeal(t *testing.T) {
	store := newFakeMealStore()
	store.foodsByID = map[string]types.FoodMatch{
		"egg": {FoodID: "egg", Name: "Egg", Per100g: types.Macros{Calories: 155, Protein: 13}},
	}
	logger := &fakeMealLogger{}
	h := newHandler(store, logger)

	rec := doRequest(h, http.MethodPost, "/api/v1/meals", map[string]any{
		"items": []map[string]any{{"food_id": "egg", "grams": 200}},
	}, nil)
	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rec.Code, rec.Body.String())
	}
	meal := decodeJSON[types.Meal](t, rec)
	if meal.RawText != "Egg" || meal.Confidence != 1 || len(meal.Items) != 1 {
		t.Fatalf("unexpected meal: %+v", meal)
	}
	if meal.Items[0].Macros.Calories != 310 || logger.lastMeal.ID != meal.ID {
		t.Fatalf("expected synchronously logged 200g egg meal, got %+v", meal)
	}
}

// TestCreateStructuredMealUnitQuantityPassthrough covers #134/B5: unit and
// quantity are optional, display-only fields that ride alongside the still-
// authoritative grams value into Parsed.Unit/Parsed.Quantity.
func TestCreateStructuredMealUnitQuantityPassthrough(t *testing.T) {
	store := newFakeMealStore()
	store.foodsByID = map[string]types.FoodMatch{
		"egg": {FoodID: "egg", Name: "Egg", Per100g: types.Macros{Calories: 155, Protein: 13}},
	}
	logger := &fakeMealLogger{}
	h := newHandler(store, logger)

	rec := doRequest(h, http.MethodPost, "/api/v1/meals", map[string]any{
		"items": []map[string]any{{"food_id": "egg", "grams": 100, "unit": "1 large egg", "quantity": 2}},
	}, nil)
	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rec.Code, rec.Body.String())
	}
	meal := decodeJSON[types.Meal](t, rec)
	if len(meal.Items) != 1 {
		t.Fatalf("unexpected meal: %+v", meal)
	}
	got := meal.Items[0].Parsed
	if got.Unit != "1 large egg" || got.Quantity != 2 || got.NormalizedGrams != 100 {
		t.Fatalf("Parsed = %+v, want Unit=%q Quantity=2 NormalizedGrams=100", got, "1 large egg")
	}

	// Omitting unit/quantity still works — grams-only logging is unaffected.
	rec = doRequest(h, http.MethodPost, "/api/v1/meals", map[string]any{
		"items": []map[string]any{{"food_id": "egg", "grams": 50}},
	}, nil)
	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rec.Code, rec.Body.String())
	}
	meal = decodeJSON[types.Meal](t, rec)
	if meal.Items[0].Parsed.Unit != "" || meal.Items[0].Parsed.Quantity != 0 {
		t.Fatalf("expected empty Unit/Quantity for grams-only item, got %+v", meal.Items[0].Parsed)
	}
}

func TestCreateStructuredMealUnknownFood(t *testing.T) {
	rec := doRequest(newHandler(newFakeMealStore(), &fakeMealLogger{}), http.MethodPost, "/api/v1/meals", map[string]any{
		"items": []map[string]any{{"food_id": "missing", "grams": 100}},
	}, nil)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestCreateStructuredMealRequiresItems(t *testing.T) {
	rec := doRequest(newHandler(newFakeMealStore(), &fakeMealLogger{}), http.MethodPost, "/api/v1/meals", map[string]any{}, nil)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", rec.Code, rec.Body.String())
	}
}
