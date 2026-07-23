package api

import (
	"errors"
	"net/http"
	"strings"
	"testing"

	"github.com/gsaraiva2109/dietdaemon/core/types"
)

// --- handleExportMeals ---

func TestHandleExportMealsJSON(t *testing.T) {
	store := newFakeMealStore()
	store.mealsInRange = []types.Meal{
		{ID: "m1", UserID: "test-user", RawText: "200g chicken"},
	}
	h := newHandler(store, &fakeMealLogger{})

	rec := doRequest(h, "GET", "/api/v1/export/meals?start=2026-06-01&end=2026-06-30", nil, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	if got := rec.Header().Get("Content-Disposition"); !strings.Contains(got, "meals.json") {
		t.Errorf("Content-Disposition = %q, want meals.json", got)
	}
	meals := decodeJSON[[]types.Meal](t, rec)
	if len(meals) != 1 {
		t.Errorf("meals count = %d, want 1", len(meals))
	}
}

func TestHandleExportMealsJSONEmptyNotNull(t *testing.T) {
	store := newFakeMealStore()
	h := newHandler(store, &fakeMealLogger{})

	rec := doRequest(h, "GET", "/api/v1/export/meals?start=2026-06-01&end=2026-06-30", nil, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	meals := decodeJSON[[]types.Meal](t, rec)
	if meals == nil {
		t.Error("expected empty array, got null")
	}
}

func TestHandleExportMealsCSV(t *testing.T) {
	store := newFakeMealStore()
	store.mealsInRange = []types.Meal{
		{ID: "m1", UserID: "test-user", RawText: "200g chicken"},
	}
	h := newHandler(store, &fakeMealLogger{})

	rec := doRequest(h, "GET", "/api/v1/export/meals?start=2026-06-01&end=2026-06-30&format=csv", nil, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	if ct := rec.Header().Get("Content-Type"); ct != "text/csv" {
		t.Errorf("Content-Type = %q, want text/csv", ct)
	}
	if got := rec.Header().Get("Content-Disposition"); !strings.Contains(got, "meals.csv") {
		t.Errorf("Content-Disposition = %q, want meals.csv", got)
	}
	if !strings.HasPrefix(rec.Body.String(), "id,date,raw_text,kcal,protein,carbs,fat,fiber") {
		t.Errorf("unexpected CSV header: %s", rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "m1") {
		t.Errorf("expected meal row in CSV, got: %s", rec.Body.String())
	}
}

func TestHandleExportMealsMissingParams(t *testing.T) {
	store := newFakeMealStore()
	h := newHandler(store, &fakeMealLogger{})

	rec := doRequest(h, "GET", "/api/v1/export/meals", nil, nil)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("missing start/end expected 400, got %d", rec.Code)
	}
}

func TestHandleExportMealsStoreError(t *testing.T) {
	store := newFakeMealStore()
	store.mealsInRangeErr = errors.New("db down")
	h := newHandler(store, &fakeMealLogger{})

	rec := doRequest(h, "GET", "/api/v1/export/meals?start=2026-06-01&end=2026-06-30", nil, nil)
	if rec.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", rec.Code)
	}
}

func TestHandleExportMealsUnauthorized(t *testing.T) {
	store := newFakeMealStore()
	h := newHandler(store, &fakeMealLogger{})

	rec := doRequest(h, "GET", "/api/v1/export/meals?start=2026-06-01&end=2026-06-30", nil, map[string]string{"Authorization": ""})
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rec.Code)
	}
}

// --- handleExportRollups ---

func TestHandleExportRollupsJSON(t *testing.T) {
	store := newFakeMealStore()
	store.rollups = []types.DailyRollup{
		{UserID: "test-user", Date: "2026-06-15", Consumed: types.Macros{Calories: 2000}},
	}
	h := newHandler(store, &fakeMealLogger{})

	rec := doRequest(h, "GET", "/api/v1/export/rollups?start=2026-06-01&end=2026-06-30", nil, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	if got := rec.Header().Get("Content-Disposition"); !strings.Contains(got, "rollups.json") {
		t.Errorf("Content-Disposition = %q, want rollups.json", got)
	}
	rollups := decodeJSON[[]types.DailyRollup](t, rec)
	if len(rollups) != 1 {
		t.Errorf("rollups count = %d, want 1", len(rollups))
	}
}

func TestHandleExportRollupsJSONEmptyNotNull(t *testing.T) {
	store := newFakeMealStore()
	h := newHandler(store, &fakeMealLogger{})

	rec := doRequest(h, "GET", "/api/v1/export/rollups?start=2026-06-01&end=2026-06-30", nil, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	rollups := decodeJSON[[]types.DailyRollup](t, rec)
	if rollups == nil {
		t.Error("expected empty array, got null")
	}
}

func TestHandleExportRollupsCSV(t *testing.T) {
	store := newFakeMealStore()
	store.rollups = []types.DailyRollup{
		{UserID: "test-user", Date: "2026-06-15", Consumed: types.Macros{Calories: 2000}},
	}
	h := newHandler(store, &fakeMealLogger{})

	rec := doRequest(h, "GET", "/api/v1/export/rollups?start=2026-06-01&end=2026-06-30&format=csv", nil, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	if ct := rec.Header().Get("Content-Type"); ct != "text/csv" {
		t.Errorf("Content-Type = %q, want text/csv", ct)
	}
	if got := rec.Header().Get("Content-Disposition"); !strings.Contains(got, "rollups.csv") {
		t.Errorf("Content-Disposition = %q, want rollups.csv", got)
	}
	if !strings.HasPrefix(rec.Body.String(), "date,consumed_kcal") {
		t.Errorf("unexpected CSV header: %s", rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "2026-06-15") {
		t.Errorf("expected rollup row in CSV, got: %s", rec.Body.String())
	}
}

func TestHandleExportRollupsMissingParams(t *testing.T) {
	store := newFakeMealStore()
	h := newHandler(store, &fakeMealLogger{})

	rec := doRequest(h, "GET", "/api/v1/export/rollups", nil, nil)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("missing start/end expected 400, got %d", rec.Code)
	}
}

func TestHandleExportRollupsStoreError(t *testing.T) {
	store := newFakeMealStore()
	store.rollupsErr = errors.New("db down")
	h := newHandler(store, &fakeMealLogger{})

	rec := doRequest(h, "GET", "/api/v1/export/rollups?start=2026-06-01&end=2026-06-30", nil, nil)
	if rec.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", rec.Code)
	}
}

func TestHandleExportRollupsUnauthorized(t *testing.T) {
	store := newFakeMealStore()
	h := newHandler(store, &fakeMealLogger{})

	rec := doRequest(h, "GET", "/api/v1/export/rollups?start=2026-06-01&end=2026-06-30", nil, map[string]string{"Authorization": ""})
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rec.Code)
	}
}
