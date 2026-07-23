package api

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gsaraiva2109/dietdaemon/core/types"
)

// doRequestRawBody sends a raw (possibly malformed) request body, since
// doRequest only knows how to JSON-marshal well-formed values.
func doRequestRawBody(h *Handler, method, path, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer test-api-key")
	rec := httptest.NewRecorder()
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)
	mux.ServeHTTP(rec, req)
	return rec
}

func TestFastingRoutesRequireAuth(t *testing.T) {
	store := newFakeMealStore()
	h := newHandler(store, &fakeMealLogger{})

	rec := doRequest(h, "GET", "/api/v1/fasting/active", nil, map[string]string{"Authorization": ""})
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rec.Code)
	}
}

func TestStartFastInvalidJSON(t *testing.T) {
	store := newFakeMealStore()
	h := newHandler(store, &fakeMealLogger{})

	req := doRequestRawBody(h, "POST", "/api/v1/fasting/start", "not json")
	if req.Code != http.StatusBadRequest {
		t.Errorf("invalid JSON expected 400, got %d", req.Code)
	}
}

func TestStartFastActiveFastCheckError(t *testing.T) {
	store := newFakeMealStore()
	store.activeFastErr = errors.New("db unavailable")
	h := newHandler(store, &fakeMealLogger{})

	rec := doRequest(h, "POST", "/api/v1/fasting/start", nil, nil)
	if rec.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", rec.Code)
	}
}

func TestStartFastStoreError(t *testing.T) {
	store := newFakeMealStore()
	store.startFastErr = errors.New("db unavailable")
	h := newHandler(store, &fakeMealLogger{})

	rec := doRequest(h, "POST", "/api/v1/fasting/start", nil, nil)
	if rec.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", rec.Code)
	}
}

func TestEndFastStoreError(t *testing.T) {
	store := newFakeMealStore()
	h := newHandler(store, &fakeMealLogger{})

	if rec := doRequest(h, "POST", "/api/v1/fasting/start", nil, nil); rec.Code != http.StatusCreated {
		t.Fatalf("start: expected 201, got %d", rec.Code)
	}
	store.endFastErr = errors.New("db unavailable")
	rec := doRequest(h, "POST", "/api/v1/fasting/end", nil, nil)
	if rec.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", rec.Code)
	}
}

func TestGetActiveFastStoreError(t *testing.T) {
	store := newFakeMealStore()
	store.activeFastErr = errors.New("db unavailable")
	h := newHandler(store, &fakeMealLogger{})

	rec := doRequest(h, "GET", "/api/v1/fasting/active", nil, nil)
	if rec.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", rec.Code)
	}
}

func TestListFastsWithData(t *testing.T) {
	store := newFakeMealStore()
	store.fasts = []types.Fast{
		{ID: "f1", UserID: "test-user", TargetHours: 16, Completed: true},
		{ID: "f2", UserID: "test-user", TargetHours: 18},
	}
	h := newHandler(store, &fakeMealLogger{})

	rec := doRequest(h, "GET", "/api/v1/fasting/history?limit=5", nil, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	fasts := decodeJSON[[]types.Fast](t, rec)
	if len(fasts) != 2 {
		t.Errorf("expected 2 fasts, got %d", len(fasts))
	}
}

func TestListFastsInvalidLimit(t *testing.T) {
	store := newFakeMealStore()
	h := newHandler(store, &fakeMealLogger{})

	rec := doRequest(h, "GET", "/api/v1/fasting/history?limit=0", nil, nil)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("limit=0 expected 400, got %d", rec.Code)
	}
}

func TestListFastsStoreError(t *testing.T) {
	store := newFakeMealStore()
	store.listFastsErr = errors.New("db unavailable")
	h := newHandler(store, &fakeMealLogger{})

	rec := doRequest(h, "GET", "/api/v1/fasting/history", nil, nil)
	if rec.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", rec.Code)
	}
}
