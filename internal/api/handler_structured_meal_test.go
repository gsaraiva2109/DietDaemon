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
	if meal.RawText != "structured entry" || meal.Confidence != 1 || len(meal.Items) != 1 {
		t.Fatalf("unexpected meal: %+v", meal)
	}
	if meal.Items[0].Macros.Calories != 310 || logger.lastMeal.ID != meal.ID {
		t.Fatalf("expected synchronously logged 200g egg meal, got %+v", meal)
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
