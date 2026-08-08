package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gsaraiva2109/dietdaemon/core/types"
)

// ---------------------------------------------------------------------------
// #274: API handlers must resolve "today"/date boundaries from the
// requesting user's own profile timezone, not the process-wide default
// (h.loc, set once at boot from cfg.Location). newHandler in handler_test.go
// always boots with h.loc = time.UTC, so every test below sets a distinct,
// non-UTC user profile timezone and proves the handler used it.
// ---------------------------------------------------------------------------

// testUserTZ is a fixed, far-from-UTC offset (UTC+14) used across these
// tests so a regression back to h.loc (UTC) is easy to notice in the
// captured date/range values.
const testUserTZ = "Pacific/Kiritimati"

// expectedTodayIn returns "today" formatted the same way handlers do,
// computed independently in the test using the given IANA zone. Comparing
// this against what a handler actually passed to the store proves the
// handler resolved the same location, without hard-coding a fixed clock.
func expectedTodayIn(t *testing.T, tz string) string {
	t.Helper()
	loc, err := time.LoadLocation(tz)
	if err != nil {
		t.Fatalf("load location %q: %v", tz, err)
	}
	return time.Now().In(loc).Format(dateLayout)
}

func TestHandlerUserLocResolution(t *testing.T) {
	store := newFakeMealStore()
	h := newHandler(store, &fakeMealLogger{})
	ctx := context.Background()

	t.Run("valid profile timezone wins over boot default", func(t *testing.T) {
		store.user = types.User{ID: "test-user", Timezone: testUserTZ}
		store.getUserErr = nil
		loc := h.userLoc(ctx, "test-user")
		want, _ := time.LoadLocation(testUserTZ)
		if loc.String() != want.String() {
			t.Errorf("loc = %v, want %v", loc, want)
		}
	})

	t.Run("empty timezone falls back to boot default", func(t *testing.T) {
		store.user = types.User{ID: "test-user", Timezone: ""}
		store.getUserErr = nil
		if loc := h.userLoc(ctx, "test-user"); loc != h.loc {
			t.Errorf("loc = %v, want boot default %v", loc, h.loc)
		}
	})

	t.Run("invalid timezone falls back to boot default", func(t *testing.T) {
		store.user = types.User{ID: "test-user", Timezone: "Not/A_Real_Zone"}
		store.getUserErr = nil
		if loc := h.userLoc(ctx, "test-user"); loc != h.loc {
			t.Errorf("loc = %v, want boot default %v", loc, h.loc)
		}
	})

	t.Run("GetUser error falls back to boot default", func(t *testing.T) {
		store.user = types.User{ID: "test-user", Timezone: testUserTZ}
		store.getUserErr = context.DeadlineExceeded
		defer func() { store.getUserErr = nil }()
		if loc := h.userLoc(ctx, "test-user"); loc != h.loc {
			t.Errorf("loc = %v, want boot default %v", loc, h.loc)
		}
	})
}

func TestHandleRollupsTodayUsesPerUserTimezone(t *testing.T) {
	store := newFakeMealStore()
	store.user = types.User{ID: "test-user", Timezone: testUserTZ}
	store.rollup = types.DailyRollup{UserID: "test-user"}
	h := newHandler(store, &fakeMealLogger{})

	want := expectedTodayIn(t, testUserTZ)
	rec := doRequest(h, "GET", "/api/v1/rollups/today", nil, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	if store.lastRollupDate != want {
		t.Errorf("GetRollup date = %q, want %q (per-user tz), boot default was %v", store.lastRollupDate, want, h.loc)
	}
}

func TestHandleGetSharedDayTypeUsesPerUserTimezone(t *testing.T) {
	store := newFakeMealStore()
	store.user = types.User{ID: "test-user", Timezone: testUserTZ}
	h := newHandler(store, &fakeMealLogger{})

	createRec := doRequest(h, "POST", "/api/v1/auth/share-tokens", map[string]string{"label": "test"}, nil)
	if createRec.Code != http.StatusCreated {
		t.Fatalf("create share token: expected 201, got %d: %s", createRec.Code, createRec.Body.String())
	}
	created := decodeJSON[types.NewShareTokenResponse](t, createRec)

	want := expectedTodayIn(t, testUserTZ)
	readRec := doRequest(h, "GET", "/api/v1/shared/"+created.Token+"/day-type", nil, map[string]string{"Authorization": ""})
	if readRec.Code != http.StatusOK {
		t.Fatalf("shared day-type: expected 200, got %d: %s", readRec.Code, readRec.Body.String())
	}
	if store.lastTargetsForDate != want {
		t.Errorf("TargetsFor date = %q, want %q (per-user tz)", store.lastTargetsForDate, want)
	}
}

func TestHandleGoalSuggestionsUsesPerUserTimezone(t *testing.T) {
	store := newFakeMealStore()
	store.user = types.User{ID: "test-user", Timezone: testUserTZ}
	store.profile = types.UserProfile{UserID: "test-user", HeightCm: 180}
	h := newHandler(store, &fakeMealLogger{})

	wantEnd := expectedTodayIn(t, testUserTZ)
	rec := doRequest(h, "GET", "/api/v1/goals/suggestions", nil, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	if store.lastRollupsEnd != wantEnd {
		t.Errorf("GetRollups end date = %q, want %q (per-user tz)", store.lastRollupsEnd, wantEnd)
	}
}

func TestHandleGetActivePlanUsesPerUserTimezone(t *testing.T) {
	store := newFakeMealStore()
	store.user = types.User{ID: "test-user", Timezone: testUserTZ}
	h := newHandler(store, &fakeMealLogger{})

	want := expectedTodayIn(t, testUserTZ)
	// No active plan configured: handler still resolves "today" before the
	// not-found response, so the captured date is enough to prove the fix.
	_ = doRequest(h, "GET", "/api/v1/plans/active", nil, nil)
	if store.lastActivePlanDate != want {
		t.Errorf("GetActivePlan date = %q, want %q (per-user tz)", store.lastActivePlanDate, want)
	}
}

func TestHandleGetWaterTodayUsesPerUserTimezone(t *testing.T) {
	store := newFakeMealStore()
	store.user = types.User{ID: "test-user", Timezone: testUserTZ}
	h := newHandler(store, &fakeMealLogger{})

	want := expectedTodayIn(t, testUserTZ)
	rec := doRequest(h, "GET", "/api/v1/body/water", nil, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	if store.lastWaterTodayDate != want {
		t.Errorf("GetWaterToday date = %q, want %q (per-user tz)", store.lastWaterTodayDate, want)
	}
}

func TestHandleUploadPhotoDefaultsToPerUserTimezone(t *testing.T) {
	store := newFakeMealStore()
	store.user = types.User{ID: "test-user", Timezone: testUserTZ}
	h := newHandler(store, &fakeMealLogger{})

	want := expectedTodayIn(t, testUserTZ)
	req := multipartUploadRequest(nil, true, pngBytes) // no "date" field: handler must default it
	rec := httptest.NewRecorder()
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rec.Code, rec.Body.String())
	}
	photo := decodeJSON[types.ProgressPhoto](t, rec)
	if photo.Date != want {
		t.Errorf("default photo date = %q, want %q (per-user tz)", photo.Date, want)
	}
	if store.lastUploadedPhotoAt != want {
		t.Errorf("UploadPhoto date = %q, want %q (per-user tz)", store.lastUploadedPhotoAt, want)
	}
}

func TestHandleStreakUsesPerUserTimezone(t *testing.T) {
	store := newFakeMealStore()
	store.user = types.User{ID: "test-user", Timezone: testUserTZ}
	h := newHandler(store, &fakeMealLogger{})

	wantEnd := time.Now().In(mustLoadLocation(t, testUserTZ)).AddDate(0, 0, -1).Format("2006-01-02")
	rec := doRequest(h, "GET", "/api/v1/streak", nil, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	if store.lastRollupsEnd != wantEnd {
		t.Errorf("GetRollups end date = %q, want %q (per-user tz), not the server-local time.Now() this handler used to call directly", store.lastRollupsEnd, wantEnd)
	}
}

func TestHandleStreakStoreError(t *testing.T) {
	store := newFakeMealStore()
	store.rollupsErr = context.DeadlineExceeded
	h := newHandler(store, &fakeMealLogger{})

	rec := doRequest(h, "GET", "/api/v1/streak", nil, nil)
	if rec.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d: %s", rec.Code, rec.Body.String())
	}
}

func mustLoadLocation(t *testing.T, tz string) *time.Location {
	t.Helper()
	loc, err := time.LoadLocation(tz)
	if err != nil {
		t.Fatalf("load location %q: %v", tz, err)
	}
	return loc
}

// TestHandleLogMeasurementsResolvesPerUserTimezone proves handleLogMeasurements
// consults the requesting user's own profile (via h.userLoc, backed by
// h.store.GetUser) rather than the process-wide boot default when validating
// body.Date -- the minimal, non-flaky check for a code path whose date-math
// output otherwise depends on the wall clock at test-run time.
func TestHandleLogMeasurementsResolvesPerUserTimezone(t *testing.T) {
	store := newFakeMealStore()
	store.user = types.User{ID: "test-user", Timezone: testUserTZ}
	h := newHandler(store, &fakeMealLogger{})

	body := map[string]any{"date": "2020-01-01", "waist_cm": 80.0}
	rec := doRequest(h, "POST", "/api/v1/body/measurements", body, nil)
	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rec.Code, rec.Body.String())
	}
	if !containsStr(store.getUserCalls, "test-user") {
		t.Errorf("GetUser calls = %v, want a lookup for %q (per-user tz resolution)", store.getUserCalls, "test-user")
	}
}

func TestHandleLogWeightResolvesPerUserTimezone(t *testing.T) {
	store := newFakeMealStore()
	store.user = types.User{ID: "test-user", Timezone: testUserTZ}
	h := newHandler(store, &fakeMealLogger{})

	body := map[string]any{"date": "2020-01-01", "weight_kg": 80.0}
	rec := doRequest(h, "POST", "/api/v1/body/weight", body, nil)
	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rec.Code, rec.Body.String())
	}
	if !containsStr(store.getUserCalls, "test-user") {
		t.Errorf("GetUser calls = %v, want a lookup for %q (per-user tz resolution)", store.getUserCalls, "test-user")
	}
}
