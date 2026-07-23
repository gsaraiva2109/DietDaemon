package api

import (
	"errors"
	"net/http"
	"testing"

	"github.com/gsaraiva2109/dietdaemon/core/types"
)

func TestMeasurementsRoutesRequireAuth(t *testing.T) {
	store := newFakeMealStore()
	h := newHandler(store, &fakeMealLogger{})

	rec := doRequest(h, "GET", "/api/v1/body/measurements", nil, map[string]string{"Authorization": ""})
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rec.Code)
	}
}

func TestListMeasurementsStoreError(t *testing.T) {
	store := newFakeMealStore()
	store.measurementsErr = errors.New("db unavailable")
	h := newHandler(store, &fakeMealLogger{})

	rec := doRequest(h, "GET", "/api/v1/body/measurements", nil, nil)
	if rec.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", rec.Code)
	}
}

func TestLogMeasurementsNoValuesProvided(t *testing.T) {
	store := newFakeMealStore()
	h := newHandler(store, &fakeMealLogger{})

	rec := doRequest(h, "POST", "/api/v1/body/measurements", map[string]any{"date": "2026-06-17"}, nil)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("no measurements provided expected 400, got %d", rec.Code)
	}
}

func TestLogMeasurementsInvalidDate(t *testing.T) {
	store := newFakeMealStore()
	h := newHandler(store, &fakeMealLogger{})

	rec := doRequest(h, "POST", "/api/v1/body/measurements", map[string]any{
		"date": "9999-12-31", "waist_cm": 90,
	}, nil)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("future date expected 400, got %d", rec.Code)
	}
}

func TestLogMeasurementsNegativeValue(t *testing.T) {
	store := newFakeMealStore()
	h := newHandler(store, &fakeMealLogger{})

	rec := doRequest(h, "POST", "/api/v1/body/measurements", map[string]any{
		"date": "2026-06-17", "waist_cm": -10,
	}, nil)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("negative measurement expected 400, got %d", rec.Code)
	}
}

func TestLogMeasurementsStoreError(t *testing.T) {
	store := newFakeMealStore()
	store.logMeasurementErr = errors.New("db unavailable")
	h := newHandler(store, &fakeMealLogger{})

	rec := doRequest(h, "POST", "/api/v1/body/measurements", map[string]any{
		"date": "2026-06-17", "waist_cm": 90,
	}, nil)
	if rec.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", rec.Code)
	}
}

func TestDeleteMeasurementNotFound(t *testing.T) {
	store := newFakeMealStore()
	store.deleteMeasurementErr = types.ErrNotFound
	h := newHandler(store, &fakeMealLogger{})

	rec := doRequest(h, "DELETE", "/api/v1/body/measurements/missing", nil, nil)
	if rec.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", rec.Code)
	}
}
