package api

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gsaraiva2109/dietdaemon/core/types"
)

func TestHandleLogWater(t *testing.T) {
	store := newFakeMealStore()
	h := newHandler(store, &fakeMealLogger{})

	body := map[string]any{"amount_ml": 250, "note": "glass"}
	rec := doRequest(h, "POST", "/api/v1/body/water", body, nil)
	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rec.Code, rec.Body.String())
	}
	entry := decodeJSON[types.WaterLog](t, rec)
	if entry.AmountML != 250 {
		t.Errorf("amount_ml = %d, want 250", entry.AmountML)
	}
	if entry.UserID != "test-user" {
		t.Errorf("user_id = %q, want test-user", entry.UserID)
	}
	if len(store.waterLogs) != 1 {
		t.Errorf("expected 1 stored water log, got %d", len(store.waterLogs))
	}
}

func TestHandleLogWaterInvalidAmount(t *testing.T) {
	store := newFakeMealStore()
	h := newHandler(store, &fakeMealLogger{})

	rec := doRequest(h, "POST", "/api/v1/body/water", map[string]any{"amount_ml": 0}, nil)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("zero amount_ml expected 400, got %d", rec.Code)
	}
}

func TestHandleLogWaterNegativeAmount(t *testing.T) {
	store := newFakeMealStore()
	h := newHandler(store, &fakeMealLogger{})

	rec := doRequest(h, "POST", "/api/v1/body/water", map[string]any{"amount_ml": -100}, nil)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("negative amount_ml expected 400, got %d", rec.Code)
	}
}

func TestHandleLogWaterInvalidJSON(t *testing.T) {
	store := newFakeMealStore()
	h := newHandler(store, &fakeMealLogger{})

	req := httptest.NewRequest("POST", "/api/v1/body/water", strings.NewReader("not json"))
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

func TestHandleLogWaterStoreError(t *testing.T) {
	store := newFakeMealStore()
	store.logWaterErr = errors.New("db unavailable")
	h := newHandler(store, &fakeMealLogger{})

	rec := doRequest(h, "POST", "/api/v1/body/water", map[string]any{"amount_ml": 250}, nil)
	if rec.Code != http.StatusInternalServerError {
		t.Errorf("store error expected 500, got %d", rec.Code)
	}
}

func TestHandleLogWaterUnauthorized(t *testing.T) {
	store := newFakeMealStore()
	h := newHandler(store, &fakeMealLogger{})

	rec := doRequest(h, "POST", "/api/v1/body/water", map[string]any{"amount_ml": 250}, map[string]string{"Authorization": ""})
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rec.Code)
	}
}

func TestHandleGetWaterToday(t *testing.T) {
	store := newFakeMealStore()
	store.waterLogs = []types.WaterLog{
		{ID: "w1", UserID: "test-user", AmountML: 250},
		{ID: "w2", UserID: "test-user", AmountML: 500},
	}
	store.waterTotal = 750
	h := newHandler(store, &fakeMealLogger{})

	rec := doRequest(h, "GET", "/api/v1/body/water", nil, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	got := decodeJSON[map[string]any](t, rec)
	if got["today_ml"].(float64) != 750 {
		t.Errorf("today_ml = %v, want 750", got["today_ml"])
	}
	logs, ok := got["logs"].([]any)
	if !ok || len(logs) != 2 {
		t.Errorf("expected 2 logs, got %v", got["logs"])
	}
	if got["goal_ml"].(float64) != defaultWaterGoalMl {
		t.Errorf("goal_ml = %v, want default %d", got["goal_ml"], defaultWaterGoalMl)
	}
}

func TestHandleGetWaterTodayCustomGoal(t *testing.T) {
	store := newFakeMealStore()
	store.targets = types.DailyTargets{WaterGoalMl: 3000}
	h := newHandler(store, &fakeMealLogger{})

	rec := doRequest(h, "GET", "/api/v1/body/water", nil, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	got := decodeJSON[map[string]any](t, rec)
	if got["goal_ml"].(float64) != 3000 {
		t.Errorf("goal_ml = %v, want 3000", got["goal_ml"])
	}
}

func TestHandleGetWaterTodayStoreError(t *testing.T) {
	store := newFakeMealStore()
	store.getWaterTodayErr = errors.New("db unavailable")
	h := newHandler(store, &fakeMealLogger{})

	rec := doRequest(h, "GET", "/api/v1/body/water", nil, nil)
	if rec.Code != http.StatusInternalServerError {
		t.Errorf("store error expected 500, got %d", rec.Code)
	}
}

func TestHandleGetWaterTodayTargetsError(t *testing.T) {
	store := newFakeMealStore()
	store.targetsErr = errors.New("db unavailable")
	h := newHandler(store, &fakeMealLogger{})

	rec := doRequest(h, "GET", "/api/v1/body/water", nil, nil)
	if rec.Code != http.StatusInternalServerError {
		t.Errorf("targets error expected 500, got %d", rec.Code)
	}
}

func TestHandleGetWaterTodayUnauthorized(t *testing.T) {
	store := newFakeMealStore()
	h := newHandler(store, &fakeMealLogger{})

	rec := doRequest(h, "GET", "/api/v1/body/water", nil, map[string]string{"Authorization": ""})
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rec.Code)
	}
}

func TestHandleDeleteWater(t *testing.T) {
	store := newFakeMealStore()
	h := newHandler(store, &fakeMealLogger{})

	rec := doRequest(h, "DELETE", "/api/v1/body/water/w1", nil, nil)
	if rec.Code != http.StatusNoContent {
		t.Errorf("expected 204, got %d", rec.Code)
	}
}

func TestHandleDeleteWaterNotFound(t *testing.T) {
	store := newFakeMealStore()
	store.deleteWaterErr = types.ErrNotFound
	h := newHandler(store, &fakeMealLogger{})

	rec := doRequest(h, "DELETE", "/api/v1/body/water/missing", nil, nil)
	if rec.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", rec.Code)
	}
}

func TestHandleDeleteWaterUnauthorized(t *testing.T) {
	store := newFakeMealStore()
	h := newHandler(store, &fakeMealLogger{})

	rec := doRequest(h, "DELETE", "/api/v1/body/water/w1", nil, map[string]string{"Authorization": ""})
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rec.Code)
	}
}
