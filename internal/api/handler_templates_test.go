package api

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gsaraiva2109/dietdaemon/core/types"
)

// --- handleListTemplates ---

func TestHandleListTemplates(t *testing.T) {
	store := newFakeMealStore()
	store.templates = []types.MealTemplate{
		{ID: "t1", UserID: "test-user", Name: "Breakfast"},
		{ID: "t2", UserID: "test-user", Name: "Lunch"},
	}
	h := newHandler(store, &fakeMealLogger{})

	rec := doRequest(h, "GET", "/api/v1/templates", nil, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	got := decodeJSON[[]types.MealTemplate](t, rec)
	if len(got) != 2 {
		t.Errorf("templates count = %d, want 2", len(got))
	}
}

func TestHandleListTemplatesEmptyNotNull(t *testing.T) {
	store := newFakeMealStore()
	h := newHandler(store, &fakeMealLogger{})

	rec := doRequest(h, "GET", "/api/v1/templates", nil, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	got := decodeJSON[[]types.MealTemplate](t, rec)
	if got == nil {
		t.Error("expected empty array, got null")
	}
}

func TestHandleListTemplatesStoreError(t *testing.T) {
	store := newFakeMealStore()
	store.templatesErr = errors.New("db down")
	h := newHandler(store, &fakeMealLogger{})

	rec := doRequest(h, "GET", "/api/v1/templates", nil, nil)
	if rec.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", rec.Code)
	}
}

func TestHandleListTemplatesUnauthorized(t *testing.T) {
	store := newFakeMealStore()
	h := newHandler(store, &fakeMealLogger{})

	rec := doRequest(h, "GET", "/api/v1/templates", nil, map[string]string{"Authorization": ""})
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rec.Code)
	}
}

// --- handleCreateTemplate ---

func TestHandleCreateTemplate(t *testing.T) {
	store := newFakeMealStore()
	h := newHandler(store, &fakeMealLogger{})

	body := map[string]any{
		"name": "Breakfast",
		"items": []types.ResolvedItem{
			{Parsed: types.ParsedItem{RawPhrase: "eggs", NormalizedGrams: 100}},
		},
	}
	rec := doRequest(h, "POST", "/api/v1/templates", body, nil)
	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rec.Code, rec.Body.String())
	}
	got := decodeJSON[types.MealTemplate](t, rec)
	if got.Name != "Breakfast" || got.UserID != "test-user" {
		t.Errorf("unexpected template: %+v", got)
	}
	if got.ID == "" {
		t.Error("expected generated ID")
	}
}

func TestHandleCreateTemplateMissingFields(t *testing.T) {
	store := newFakeMealStore()
	h := newHandler(store, &fakeMealLogger{})

	rec := doRequest(h, "POST", "/api/v1/templates", map[string]any{"name": ""}, nil)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("missing name/items expected 400, got %d", rec.Code)
	}
}

func TestHandleCreateTemplateInvalidJSON(t *testing.T) {
	store := newFakeMealStore()
	h := newHandler(store, &fakeMealLogger{})

	req := httptest.NewRequest("POST", "/api/v1/templates", strings.NewReader("not json"))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer test-api-key")
	rec := httptest.NewRecorder()
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("invalid JSON expected 400, got %d", rec.Code)
	}
}

func TestHandleCreateTemplateStoreError(t *testing.T) {
	store := newFakeMealStore()
	store.saveTemplateErr = errors.New("disk full")
	h := newHandler(store, &fakeMealLogger{})

	body := map[string]any{
		"name":  "Breakfast",
		"items": []types.ResolvedItem{{Parsed: types.ParsedItem{RawPhrase: "eggs"}}},
	}
	rec := doRequest(h, "POST", "/api/v1/templates", body, nil)
	if rec.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", rec.Code)
	}
}

func TestHandleCreateTemplateUnauthorized(t *testing.T) {
	store := newFakeMealStore()
	h := newHandler(store, &fakeMealLogger{})

	rec := doRequest(h, "POST", "/api/v1/templates", map[string]any{}, map[string]string{"Authorization": ""})
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rec.Code)
	}
}

// --- handleComposeTemplate ---

func composeFoodStore() *fakeMealStore {
	store := newFakeMealStore()
	store.foodsByID = map[string]types.FoodMatch{
		"chicken": {
			FoodID: "chicken", Name: "Chicken Breast", Source: "usda",
			Per100g: types.Macros{Calories: 165, Protein: 31},
		},
	}
	return store
}

func TestHandleComposeTemplate(t *testing.T) {
	store := composeFoodStore()
	h := newHandler(store, &fakeMealLogger{})

	body := map[string]any{
		"name": "Chicken Meal",
		"items": []map[string]any{
			{"food_id": "chicken", "grams": 200},
		},
	}
	rec := doRequest(h, "POST", "/api/v1/templates/compose", body, nil)
	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rec.Code, rec.Body.String())
	}
	got := decodeJSON[types.MealTemplate](t, rec)
	if len(got.Items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(got.Items))
	}
	if got.Items[0].Macros.Calories != 330 {
		t.Errorf("calories = %v, want 330 (165 * 2)", got.Items[0].Macros.Calories)
	}
}

func TestHandleComposeTemplateMissingFields(t *testing.T) {
	store := composeFoodStore()
	h := newHandler(store, &fakeMealLogger{})

	rec := doRequest(h, "POST", "/api/v1/templates/compose", map[string]any{"name": "x"}, nil)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("missing items expected 400, got %d", rec.Code)
	}
}

func TestHandleComposeTemplateInvalidGrams(t *testing.T) {
	store := composeFoodStore()
	h := newHandler(store, &fakeMealLogger{})

	body := map[string]any{
		"name":  "Chicken Meal",
		"items": []map[string]any{{"food_id": "chicken", "grams": 0}},
	}
	rec := doRequest(h, "POST", "/api/v1/templates/compose", body, nil)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("non-positive grams expected 400, got %d", rec.Code)
	}
}

func TestHandleComposeTemplateUnknownFood(t *testing.T) {
	store := composeFoodStore()
	h := newHandler(store, &fakeMealLogger{})

	body := map[string]any{
		"name":  "Chicken Meal",
		"items": []map[string]any{{"food_id": "nonexistent", "grams": 100}},
	}
	rec := doRequest(h, "POST", "/api/v1/templates/compose", body, nil)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("unknown food_id expected 400, got %d", rec.Code)
	}
}

func TestHandleComposeTemplateUnauthorized(t *testing.T) {
	store := composeFoodStore()
	h := newHandler(store, &fakeMealLogger{})

	rec := doRequest(h, "POST", "/api/v1/templates/compose", map[string]any{}, map[string]string{"Authorization": ""})
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rec.Code)
	}
}

// --- handleGetTemplate ---

func TestHandleGetTemplate(t *testing.T) {
	store := newFakeMealStore()
	store.template = types.MealTemplate{ID: "t1", UserID: "test-user", Name: "Breakfast"}
	h := newHandler(store, &fakeMealLogger{})

	rec := doRequest(h, "GET", "/api/v1/templates/t1", nil, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	got := decodeJSON[types.MealTemplate](t, rec)
	if got.ID != "t1" {
		t.Errorf("id = %q, want t1", got.ID)
	}
}

func TestHandleGetTemplateNotFound(t *testing.T) {
	store := newFakeMealStore()
	store.templateErr = types.ErrNotFound
	h := newHandler(store, &fakeMealLogger{})

	rec := doRequest(h, "GET", "/api/v1/templates/missing", nil, nil)
	if rec.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", rec.Code)
	}
}

func TestHandleGetTemplateWrongUser(t *testing.T) {
	store := newFakeMealStore()
	store.template = types.MealTemplate{ID: "t1", UserID: "other-user"}
	h := newHandler(store, &fakeMealLogger{})

	rec := doRequest(h, "GET", "/api/v1/templates/t1", nil, nil)
	if rec.Code != http.StatusNotFound {
		t.Errorf("cross-user template access expected 404, got %d", rec.Code)
	}
}

func TestHandleGetTemplateUnauthorized(t *testing.T) {
	store := newFakeMealStore()
	h := newHandler(store, &fakeMealLogger{})

	rec := doRequest(h, "GET", "/api/v1/templates/t1", nil, map[string]string{"Authorization": ""})
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rec.Code)
	}
}

// --- handleDeleteTemplate ---

func TestHandleDeleteTemplate(t *testing.T) {
	store := newFakeMealStore()
	h := newHandler(store, &fakeMealLogger{})

	rec := doRequest(h, "DELETE", "/api/v1/templates/t1", nil, nil)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestHandleDeleteTemplateNotFound(t *testing.T) {
	store := newFakeMealStore()
	store.deleteTemplateErr = types.ErrNotFound
	h := newHandler(store, &fakeMealLogger{})

	rec := doRequest(h, "DELETE", "/api/v1/templates/missing", nil, nil)
	if rec.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", rec.Code)
	}
}

func TestHandleDeleteTemplateUnauthorized(t *testing.T) {
	store := newFakeMealStore()
	h := newHandler(store, &fakeMealLogger{})

	rec := doRequest(h, "DELETE", "/api/v1/templates/t1", nil, map[string]string{"Authorization": ""})
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rec.Code)
	}
}

// --- handleLogTemplate ---

func TestHandleLogTemplate(t *testing.T) {
	store := newFakeMealStore()
	store.template = types.MealTemplate{ID: "t1", UserID: "test-user", Name: "Breakfast"}
	logger := &fakeMealLogger{}
	h := newHandler(store, logger)

	rec := doRequest(h, "POST", "/api/v1/templates/t1/log", nil, nil)
	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rec.Code, rec.Body.String())
	}
	if logger.lastMeal.RawText != "Breakfast" {
		t.Errorf("logged meal raw text = %q, want Breakfast", logger.lastMeal.RawText)
	}
	if logger.lastMeal.UserID != "test-user" {
		t.Errorf("logged meal userID = %q, want test-user", logger.lastMeal.UserID)
	}
}

func TestHandleLogTemplateNotFound(t *testing.T) {
	store := newFakeMealStore()
	store.templateErr = types.ErrNotFound
	h := newHandler(store, &fakeMealLogger{})

	rec := doRequest(h, "POST", "/api/v1/templates/missing/log", nil, nil)
	if rec.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", rec.Code)
	}
}

func TestHandleLogTemplateWrongUser(t *testing.T) {
	store := newFakeMealStore()
	store.template = types.MealTemplate{ID: "t1", UserID: "other-user"}
	h := newHandler(store, &fakeMealLogger{})

	rec := doRequest(h, "POST", "/api/v1/templates/t1/log", nil, nil)
	if rec.Code != http.StatusNotFound {
		t.Errorf("cross-user template log expected 404, got %d", rec.Code)
	}
}

func TestHandleLogTemplateLoggerError(t *testing.T) {
	store := newFakeMealStore()
	store.template = types.MealTemplate{ID: "t1", UserID: "test-user", Name: "Breakfast"}
	logger := &fakeMealLogger{err: errors.New("pipeline busy")}
	h := newHandler(store, logger)

	rec := doRequest(h, "POST", "/api/v1/templates/t1/log", nil, nil)
	if rec.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", rec.Code)
	}
}

func TestHandleLogTemplateUnauthorized(t *testing.T) {
	store := newFakeMealStore()
	h := newHandler(store, &fakeMealLogger{})

	rec := doRequest(h, "POST", "/api/v1/templates/t1/log", nil, map[string]string{"Authorization": ""})
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rec.Code)
	}
}

// --- handleDuplicateMeal ---

func TestHandleDuplicateMeal(t *testing.T) {
	store := newFakeMealStore()
	store.meals["m1"] = types.Meal{ID: "m1", UserID: "test-user", RawText: "200g chicken"}
	logger := &fakeMealLogger{}
	h := newHandler(store, logger)

	rec := doRequest(h, "POST", "/api/v1/meals/m1/duplicate", nil, nil)
	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rec.Code, rec.Body.String())
	}
	if logger.lastMeal.RawText != "200g chicken" {
		t.Errorf("duplicated meal raw text = %q, want %q", logger.lastMeal.RawText, "200g chicken")
	}
	if logger.lastMeal.ID == "m1" {
		t.Error("expected a new meal ID, got the original")
	}
}

func TestHandleDuplicateMealNotFound(t *testing.T) {
	store := newFakeMealStore()
	h := newHandler(store, &fakeMealLogger{})

	rec := doRequest(h, "POST", "/api/v1/meals/missing/duplicate", nil, nil)
	if rec.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", rec.Code)
	}
}

func TestHandleDuplicateMealWrongUser(t *testing.T) {
	store := newFakeMealStore()
	store.meals["m1"] = types.Meal{ID: "m1", UserID: "other-user"}
	h := newHandler(store, &fakeMealLogger{})

	rec := doRequest(h, "POST", "/api/v1/meals/m1/duplicate", nil, nil)
	if rec.Code != http.StatusNotFound {
		t.Errorf("cross-user meal duplicate expected 404, got %d", rec.Code)
	}
}

func TestHandleDuplicateMealLoggerError(t *testing.T) {
	store := newFakeMealStore()
	store.meals["m1"] = types.Meal{ID: "m1", UserID: "test-user"}
	logger := &fakeMealLogger{err: errors.New("pipeline busy")}
	h := newHandler(store, logger)

	rec := doRequest(h, "POST", "/api/v1/meals/m1/duplicate", nil, nil)
	if rec.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", rec.Code)
	}
}

func TestHandleDuplicateMealUnauthorized(t *testing.T) {
	store := newFakeMealStore()
	h := newHandler(store, &fakeMealLogger{})

	rec := doRequest(h, "POST", "/api/v1/meals/m1/duplicate", nil, map[string]string{"Authorization": ""})
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rec.Code)
	}
}
