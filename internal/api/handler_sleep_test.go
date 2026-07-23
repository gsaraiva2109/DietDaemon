package api

import (
	"net/http"
	"testing"
	"time"

	"github.com/gsaraiva2109/dietdaemon/core/types"
)

func TestHandleLogSleep(t *testing.T) {
	store := newFakeMealStore()
	h := newHandler(store, &fakeMealLogger{})

	sleepAt := time.Now().Add(-8 * time.Hour).UTC().Format(time.RFC3339)
	body := map[string]any{"sleep_at": sleepAt, "quality": "good"}
	rec := doRequest(h, "POST", "/api/v1/body/sleep", body, nil)
	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rec.Code, rec.Body.String())
	}
	entry := decodeJSON[types.SleepLog](t, rec)
	if entry.Quality != "good" {
		t.Errorf("quality = %q, want good", entry.Quality)
	}
	if entry.UserID != "test-user" {
		t.Errorf("user_id = %q, want test-user", entry.UserID)
	}
	if len(store.sleepLogs) != 1 {
		t.Errorf("expected 1 stored sleep log, got %d", len(store.sleepLogs))
	}
}

func TestHandleLogSleepDefaults(t *testing.T) {
	store := newFakeMealStore()
	h := newHandler(store, &fakeMealLogger{})

	rec := doRequest(h, "POST", "/api/v1/body/sleep", map[string]any{}, nil)
	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rec.Code, rec.Body.String())
	}
	entry := decodeJSON[types.SleepLog](t, rec)
	if entry.Quality != "ok" {
		t.Errorf("quality = %q, want default ok", entry.Quality)
	}
	if entry.SleepAt == "" {
		t.Error("expected sleep_at to default to now, got empty")
	}
}

func TestHandleLogSleepInvalidQuality(t *testing.T) {
	store := newFakeMealStore()
	h := newHandler(store, &fakeMealLogger{})

	rec := doRequest(h, "POST", "/api/v1/body/sleep", map[string]any{"quality": "amazing"}, nil)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("invalid quality expected 400, got %d", rec.Code)
	}
}

func TestHandleLogSleepUnauthorized(t *testing.T) {
	store := newFakeMealStore()
	h := newHandler(store, &fakeMealLogger{})

	rec := doRequest(h, "POST", "/api/v1/body/sleep", map[string]any{}, map[string]string{"Authorization": ""})
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rec.Code)
	}
}

func TestHandleListSleep(t *testing.T) {
	store := newFakeMealStore()
	wakeAt := time.Now().UTC().Format(time.RFC3339)
	store.sleepLogs = []types.SleepLog{
		{ID: "s1", UserID: "test-user", SleepAt: time.Now().Add(-8 * time.Hour).UTC().Format(time.RFC3339), WakeAt: &wakeAt, Quality: "good"},
	}
	h := newHandler(store, &fakeMealLogger{})

	rec := doRequest(h, "GET", "/api/v1/body/sleep?limit=5", nil, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	sleeps := decodeJSON[[]types.SleepLog](t, rec)
	if len(sleeps) != 1 {
		t.Fatalf("expected 1 sleep log, got %d", len(sleeps))
	}
	if sleeps[0].DurationHours <= 0 {
		t.Errorf("expected positive duration, got %v", sleeps[0].DurationHours)
	}
}

func TestHandleListSleepEmpty(t *testing.T) {
	store := newFakeMealStore()
	h := newHandler(store, &fakeMealLogger{})

	rec := doRequest(h, "GET", "/api/v1/body/sleep", nil, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	sleeps := decodeJSON[[]types.SleepLog](t, rec)
	if sleeps == nil {
		t.Error("expected empty array, got null")
	}
}

func TestHandleListSleepUnauthorized(t *testing.T) {
	store := newFakeMealStore()
	h := newHandler(store, &fakeMealLogger{})

	rec := doRequest(h, "GET", "/api/v1/body/sleep", nil, map[string]string{"Authorization": ""})
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rec.Code)
	}
}

func TestHandleGetActiveSleep(t *testing.T) {
	store := newFakeMealStore()
	store.activeSleep = &types.SleepLog{
		ID: "s1", UserID: "test-user",
		SleepAt: time.Now().Add(-2 * time.Hour).UTC().Format(time.RFC3339),
		Quality: "ok",
	}
	h := newHandler(store, &fakeMealLogger{})

	rec := doRequest(h, "GET", "/api/v1/body/sleep/active", nil, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	sleep := decodeJSON[types.SleepLog](t, rec)
	if sleep.ID != "s1" {
		t.Errorf("id = %q, want s1", sleep.ID)
	}
	if sleep.DurationHours <= 0 {
		t.Errorf("expected positive duration, got %v", sleep.DurationHours)
	}
}

func TestHandleGetActiveSleepNotFound(t *testing.T) {
	store := newFakeMealStore()
	h := newHandler(store, &fakeMealLogger{})

	rec := doRequest(h, "GET", "/api/v1/body/sleep/active", nil, nil)
	if rec.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", rec.Code)
	}
}

func TestHandleGetActiveSleepUnauthorized(t *testing.T) {
	store := newFakeMealStore()
	h := newHandler(store, &fakeMealLogger{})

	rec := doRequest(h, "GET", "/api/v1/body/sleep/active", nil, map[string]string{"Authorization": ""})
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rec.Code)
	}
}

func TestHandleEndSleep(t *testing.T) {
	store := newFakeMealStore()
	h := newHandler(store, &fakeMealLogger{})

	body := map[string]any{"quality": "great"}
	rec := doRequest(h, "PATCH", "/api/v1/body/sleep/s1/end", body, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestHandleEndSleepInvalidQuality(t *testing.T) {
	store := newFakeMealStore()
	h := newHandler(store, &fakeMealLogger{})

	rec := doRequest(h, "PATCH", "/api/v1/body/sleep/s1/end", map[string]any{"quality": "meh"}, nil)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("invalid quality expected 400, got %d", rec.Code)
	}
}

func TestHandleEndSleepNotFound(t *testing.T) {
	store := newFakeMealStore()
	store.endSleepErr = types.ErrNotFound
	h := newHandler(store, &fakeMealLogger{})

	rec := doRequest(h, "PATCH", "/api/v1/body/sleep/missing/end", map[string]any{}, nil)
	if rec.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", rec.Code)
	}
}

func TestHandleEndSleepUnauthorized(t *testing.T) {
	store := newFakeMealStore()
	h := newHandler(store, &fakeMealLogger{})

	rec := doRequest(h, "PATCH", "/api/v1/body/sleep/s1/end", map[string]any{}, map[string]string{"Authorization": ""})
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rec.Code)
	}
}

func TestHandleDeleteSleep(t *testing.T) {
	store := newFakeMealStore()
	h := newHandler(store, &fakeMealLogger{})

	rec := doRequest(h, "DELETE", "/api/v1/body/sleep/s1", nil, nil)
	if rec.Code != http.StatusNoContent {
		t.Errorf("expected 204, got %d", rec.Code)
	}
}

func TestHandleDeleteSleepNotFound(t *testing.T) {
	store := newFakeMealStore()
	store.deleteSleepErr = types.ErrNotFound
	h := newHandler(store, &fakeMealLogger{})

	rec := doRequest(h, "DELETE", "/api/v1/body/sleep/missing", nil, nil)
	if rec.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", rec.Code)
	}
}

func TestHandleDeleteSleepUnauthorized(t *testing.T) {
	store := newFakeMealStore()
	h := newHandler(store, &fakeMealLogger{})

	rec := doRequest(h, "DELETE", "/api/v1/body/sleep/s1", nil, map[string]string{"Authorization": ""})
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rec.Code)
	}
}
