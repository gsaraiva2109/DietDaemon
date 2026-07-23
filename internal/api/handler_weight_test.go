package api

import (
	"errors"
	"net/http"
	"testing"

	"github.com/gsaraiva2109/dietdaemon/core/types"
)

// TestWeightRoutesRequireAuth is a terse check that the weight sub-routes are
// wired through h.wrap (identical unauthenticated behavior across handlers,
// so one representative endpoint is enough).
func TestWeightRoutesRequireAuth(t *testing.T) {
	store := newFakeMealStore()
	h := newHandler(store, &fakeMealLogger{})

	rec := doRequest(h, "GET", "/api/v1/body/weight", nil, map[string]string{"Authorization": ""})
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rec.Code)
	}
}

func TestListWeightInvalidDaysParam(t *testing.T) {
	store := newFakeMealStore()
	h := newHandler(store, &fakeMealLogger{})

	rec := doRequest(h, "GET", "/api/v1/body/weight?days=0", nil, nil)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("days=0 expected 400, got %d", rec.Code)
	}
}

func TestListWeightStoreError(t *testing.T) {
	store := newFakeMealStore()
	store.weightsErr = errors.New("db unavailable")
	h := newHandler(store, &fakeMealLogger{})

	rec := doRequest(h, "GET", "/api/v1/body/weight", nil, nil)
	if rec.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", rec.Code)
	}
}

func TestLogWeightInvalidDate(t *testing.T) {
	store := newFakeMealStore()
	h := newHandler(store, &fakeMealLogger{})

	rec := doRequest(h, "POST", "/api/v1/body/weight", map[string]any{
		"weight_kg": 80.5, "date": "9999-12-31",
	}, nil)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("future date expected 400, got %d", rec.Code)
	}
}

func TestLogWeightOutOfRange(t *testing.T) {
	store := newFakeMealStore()
	h := newHandler(store, &fakeMealLogger{})

	rec := doRequest(h, "POST", "/api/v1/body/weight", map[string]any{
		"weight_kg": 600, "date": "2026-06-17",
	}, nil)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("out-of-range weight expected 400, got %d", rec.Code)
	}
}

func TestLogWeightStoreError(t *testing.T) {
	store := newFakeMealStore()
	store.logWeightErr = errors.New("db unavailable")
	h := newHandler(store, &fakeMealLogger{})

	rec := doRequest(h, "POST", "/api/v1/body/weight", map[string]any{
		"weight_kg": 80.5, "date": "2026-06-17",
	}, nil)
	if rec.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", rec.Code)
	}
}

func TestWeightTrendStoreError(t *testing.T) {
	store := newFakeMealStore()
	store.weightTrendErr = errors.New("db unavailable")
	h := newHandler(store, &fakeMealLogger{})

	rec := doRequest(h, "GET", "/api/v1/body/weight/trend", nil, nil)
	if rec.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", rec.Code)
	}
}

func TestDeleteWeightNotFound(t *testing.T) {
	store := newFakeMealStore()
	store.deleteWeightErr = types.ErrNotFound
	h := newHandler(store, &fakeMealLogger{})

	rec := doRequest(h, "DELETE", "/api/v1/body/weight/missing", nil, nil)
	if rec.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", rec.Code)
	}
}

func TestBodySummaryWithData(t *testing.T) {
	store := newFakeMealStore()
	store.weights = []types.WeightEntry{
		{ID: "w1", WeightKg: 85, Date: "2026-06-01"},
		{ID: "w2", WeightKg: 80, Date: "2026-06-17"},
	}
	store.weightTrend = []types.WeightTrend{
		{Date: "2026-06-15", WeightKg: 81, RollingAvg: 82},
		{Date: "2026-06-16", WeightKg: 80, RollingAvg: 80},
	}
	h := newHandler(store, &fakeMealLogger{})

	rec := doRequest(h, "GET", "/api/v1/body/summary", nil, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	summary := decodeJSON[types.BodyCompositionSummary](t, rec)
	if summary.CurrentWeightKg != 80 || summary.StartWeightKg != 85 {
		t.Errorf("unexpected summary weights: %+v", summary)
	}
	if summary.ChangeKg != -5 {
		t.Errorf("expected change -5, got %v", summary.ChangeKg)
	}
	if summary.TrendDirection != "down" {
		t.Errorf("expected trend direction down, got %q", summary.TrendDirection)
	}
}

func TestBodySummaryStoreError(t *testing.T) {
	store := newFakeMealStore()
	store.weightsErr = errors.New("db unavailable")
	h := newHandler(store, &fakeMealLogger{})

	rec := doRequest(h, "GET", "/api/v1/body/summary", nil, nil)
	if rec.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", rec.Code)
	}
}
