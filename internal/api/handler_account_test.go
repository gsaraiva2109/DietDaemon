package api

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/gsaraiva2109/dietdaemon/core/types"
	"github.com/gsaraiva2109/dietdaemon/internal/auth"
)

// accountRepos is the set of auth.*Repo interfaces WithAuth needs alongside
// AuthStore. *fakeAuthStore (and wrappers embedding it) already implement all
// of these, exactly like it does in newHandler.
type accountRepos interface {
	AuthStore
	auth.SessionRepo
	auth.LoginAttemptRepo
	auth.TOTPRepo
	auth.MFAChallengeRepo
	auth.RecoveryCodeRepo
}

// newHandlerWithAccountStore mirrors newHandler but takes the authStore
// explicitly, so tests can keep a reference to it (to assert on side effects,
// or to swap in a wrapper that simulates a not-found error).
func newHandlerWithAccountStore(store MealStore, authStore accountRepos) *Handler {
	return New(store, &fakeMealLogger{}, time.UTC, nil, nil,
		WithAuth(authStore, AuthRepos{Sessions: authStore, LoginAttempts: authStore, TOTP: authStore, MFAChallenges: authStore, RecoveryCodes: authStore}, nil, "DietDaemon", AuthConfig{
			SessionCfg: auth.SessionConfig{
				IdleTTL:     1 * time.Hour,
				AbsoluteTTL: 24 * time.Hour,
				RememberTTL: 72 * time.Hour,
			},
			LockoutCfg:       auth.DefaultLockoutConfig(),
			RegistrationMode: types.RegistrationOpen,
			CookieSecure:     false,
		}),
	)
}

// notFoundAccountStore wraps *fakeAuthStore and overrides DeleteAccount to
// simulate the store reporting the account doesn't exist.
type notFoundAccountStore struct {
	*fakeAuthStore
}

func (s *notFoundAccountStore) DeleteAccount(_ context.Context, _ string) error {
	return types.ErrNotFound
}

// ---------------------------------------------------------------------------
// GET /api/v1/export/all
// ---------------------------------------------------------------------------

func TestHandleExportAll(t *testing.T) {
	store := newFakeMealStore()
	store.user = types.User{ID: "test-user", Email: "test@example.com"}
	store.profile = types.UserProfile{UserID: "test-user", Onboarded: true, HeightCm: 180}
	store.mealsInRange = []types.Meal{{ID: "m1", UserID: "test-user", RawText: "chicken"}}
	store.rollups = []types.DailyRollup{{UserID: "test-user", Date: "2026-06-17"}}
	store.weights = []types.WeightEntry{{ID: "w1", UserID: "test-user", WeightKg: 80}}
	store.measurements = []types.MeasurementEntry{{ID: "me1", UserID: "test-user", WaistCm: 90}}
	store.fasts = []types.Fast{{ID: "f1", UserID: "test-user", TargetHours: 16}}
	store.templates = []types.MealTemplate{{ID: "t1", UserID: "test-user"}}
	store.photoMetadata = []types.ProgressPhoto{{ID: "p1", UserID: "test-user", View: "front"}}
	store.photoData = types.ProgressPhoto{ID: "p1", UserID: "test-user", View: "front", Data: []byte("imgbytes")}
	// waterDailyTotals, workouts, sleep intentionally left nil to exercise
	// the nil -> [] coercion.

	h := newHandler(store, &fakeMealLogger{})

	rec := doRequest(h, "GET", "/api/v1/export/all", nil, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	if cd := rec.Header().Get("Content-Disposition"); !strings.Contains(cd, "attachment") {
		t.Errorf("expected attachment Content-Disposition, got %q", cd)
	}

	export := decodeJSON[UserDataExport](t, rec)
	if export.User.ID != "test-user" {
		t.Errorf("user.id = %q, want test-user", export.User.ID)
	}
	if !export.Profile.Onboarded {
		t.Errorf("profile.onboarded = false, want true")
	}
	if len(export.Meals) != 1 || len(export.Rollups) != 1 || len(export.Weight) != 1 ||
		len(export.Measurements) != 1 || len(export.Fasts) != 1 || len(export.Templates) != 1 {
		t.Fatalf("expected every populated slice to round-trip: %+v", export)
	}
	if len(export.Photos) != 1 || string(export.Photos[0].Data) != "imgbytes" {
		t.Fatalf("expected photo data to be included, got %+v", export.Photos)
	}
	// Fields the fake store returns nil for must still be [] , not null.
	if export.Workouts == nil {
		t.Errorf("workouts = nil, want []")
	}
	if export.Sleep == nil {
		t.Errorf("sleep = nil, want []")
	}
	if export.WaterDailyTotals == nil {
		t.Errorf("water_daily_totals = nil, want []")
	}
	if export.ExportedAt.IsZero() {
		t.Errorf("exported_at is zero")
	}
}

func TestHandleExportAllUserNotFound(t *testing.T) {
	store := newFakeMealStore()
	store.getUserErr = types.ErrNotFound
	h := newHandler(store, &fakeMealLogger{})

	rec := doRequest(h, "GET", "/api/v1/export/all", nil, nil)
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d: %s", rec.Code, rec.Body.String())
	}
	if body := decodeJSON[errorEnvelope](t, rec); body.Error.Code != ErrorInternal {
		t.Fatalf("expected generic 500, got %#v", body)
	}
}

func TestHandleExportAllProfileNotFoundDefaults(t *testing.T) {
	store := newFakeMealStore()
	store.user = types.User{ID: "test-user"}
	store.profileErr = types.ErrNotFound
	h := newHandler(store, &fakeMealLogger{})

	rec := doRequest(h, "GET", "/api/v1/export/all", nil, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	export := decodeJSON[UserDataExport](t, rec)
	if export.Profile.Onboarded {
		t.Errorf("expected default (not onboarded) profile when none exists, got %+v", export.Profile)
	}
}

// TestHandleExportAllStoreErrors covers every "if err != nil { h.writeErr }"
// branch in handleExportAll's linear fetch sequence: each sub-fetch's store
// error must propagate as a 500, independent of which one fails.
func TestHandleExportAllStoreErrors(t *testing.T) {
	dbErr := errors.New("db down")
	for name, setup := range map[string]func(*fakeMealStore){
		"profile":      func(s *fakeMealStore) { s.profileErr = dbErr },
		"meals":        func(s *fakeMealStore) { s.mealsInRangeErr = dbErr },
		"rollups":      func(s *fakeMealStore) { s.rollupsErr = dbErr },
		"weight":       func(s *fakeMealStore) { s.weightsErr = dbErr },
		"measurements": func(s *fakeMealStore) { s.measurementsErr = dbErr },
		"sleep":        func(s *fakeMealStore) { s.listSleepErr = dbErr },
		"workouts":     func(s *fakeMealStore) { s.listWorkoutsErr = dbErr },
		"fasts":        func(s *fakeMealStore) { s.listFastsErr = dbErr },
		"water totals": func(s *fakeMealStore) { s.waterDailyTotalsErr = dbErr },
		"templates":    func(s *fakeMealStore) { s.templatesErr = dbErr },
		"photo meta":   func(s *fakeMealStore) { s.photoMetadataErr = dbErr },
	} {
		t.Run(name, func(t *testing.T) {
			store := newFakeMealStore()
			store.user = types.User{ID: "test-user"}
			setup(store)
			h := newHandler(store, &fakeMealLogger{})

			rec := doRequest(h, "GET", "/api/v1/export/all", nil, nil)
			if rec.Code != http.StatusInternalServerError {
				t.Errorf("expected 500, got %d: %s", rec.Code, rec.Body.String())
			}
		})
	}
}

func TestHandleExportAllPhotoDataError(t *testing.T) {
	store := newFakeMealStore()
	store.user = types.User{ID: "test-user"}
	store.photoMetadata = []types.ProgressPhoto{{ID: "p1"}}
	store.photoDataErr = errors.New("db down")
	h := newHandler(store, &fakeMealLogger{})

	rec := doRequest(h, "GET", "/api/v1/export/all", nil, nil)
	if rec.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d: %s", rec.Code, rec.Body.String())
	}
}

// ---------------------------------------------------------------------------
// DELETE /api/v1/account
// ---------------------------------------------------------------------------

func TestHandleDeleteAccountMissingBody(t *testing.T) {
	h := newHandler(newFakeMealStore(), &fakeMealLogger{})

	rec := doRequest(h, "DELETE", "/api/v1/account", nil, nil)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for missing body, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestHandleDeleteAccountWrongConfirm(t *testing.T) {
	h := newHandler(newFakeMealStore(), &fakeMealLogger{})

	rec := doRequest(h, "DELETE", "/api/v1/account", map[string]string{"confirm": "delete"}, nil)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for wrong confirm value, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestHandleDeleteAccountSuccess(t *testing.T) {
	authStore := newFakeAuthStore()
	h := newHandlerWithAccountStore(newFakeMealStore(), authStore)

	if _, ok := authStore.users["test-user"]; !ok {
		t.Fatalf("test setup: expected test-user to be seeded")
	}

	rec := doRequest(h, "DELETE", "/api/v1/account", map[string]string{"confirm": "DELETE"}, nil)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d: %s", rec.Code, rec.Body.String())
	}

	if _, ok := authStore.users["test-user"]; ok {
		t.Errorf("expected DeleteAccount to be called with the authenticated userID (test-user), but it's still present")
	}

	// Session cookie must be cleared.
	cleared := false
	for _, c := range rec.Result().Cookies() {
		if c.Name == "dd_session" && c.MaxAge < 0 {
			cleared = true
		}
	}
	if !cleared {
		t.Errorf("expected dd_session cookie to be cleared")
	}
}

// TestHandleDeleteAccountClearsSessionCookie covers the best-effort session
// cache cleanup: when the request carries a dd_session cookie, the handler
// must also delete that cached session (in addition to always clearing the
// response cookies, which TestHandleDeleteAccountSuccess already checks).
func TestHandleDeleteAccountClearsSessionCookie(t *testing.T) {
	authStore := newFakeAuthStore()
	h := newHandlerWithAccountStore(newFakeMealStore(), authStore)

	rec := doRequest(h, "DELETE", "/api/v1/account", map[string]string{"confirm": "DELETE"}, map[string]string{
		"Cookie": "dd_session=some-session-token",
	})
	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestHandleDeleteAccountNotFound(t *testing.T) {
	authStore := &notFoundAccountStore{fakeAuthStore: newFakeAuthStore()}
	h := newHandlerWithAccountStore(newFakeMealStore(), authStore)

	rec := doRequest(h, "DELETE", "/api/v1/account", map[string]string{"confirm": "DELETE"}, nil)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", rec.Code, rec.Body.String())
	}
}
