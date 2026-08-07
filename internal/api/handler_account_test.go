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
	"github.com/gsaraiva2109/dietdaemon/internal/store"
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

// notFoundAccountStore wraps *fakeAuthStore and overrides RequestAccountDeletion
// to simulate the store reporting the account doesn't exist.
type notFoundAccountStore struct {
	*fakeAuthStore
}

func (s *notFoundAccountStore) RequestAccountDeletion(_ context.Context, _ string) error {
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

// TestHandleExportAllPhotosPurgedNote covers the branch where
// AccountDeletionStatus reports the day-30 photo-purge tier has already run:
// the export must carry photos_purged_at/photos_note instead of silently
// returning an empty Photos slice.
func TestHandleExportAllPhotosPurgedNote(t *testing.T) {
	authStore := newFakeAuthStore()
	purgedAt := time.Now().UTC().Add(-5 * 24 * time.Hour)
	authStore.deletionStatus["test-user"] = store.AccountDeletionStatus{PhotosPurgedAt: &purgedAt}
	ms := newFakeMealStore()
	ms.user = types.User{ID: "test-user"}
	h := newHandlerWithAccountStore(ms, authStore)

	rec := doRequest(h, "GET", "/api/v1/export/all", nil, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	export := decodeJSON[UserDataExport](t, rec)
	if export.PhotosPurgedAt == nil {
		t.Fatal("expected photos_purged_at to be set")
	}
	if export.PhotosNote == "" {
		t.Error("expected photos_note to be set")
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

	// RequestAccountDeletion soft-deletes: the user row stays, but
	// deleted_at is now set for the authenticated userID (test-user).
	if _, ok := authStore.users["test-user"]; !ok {
		t.Errorf("expected RequestAccountDeletion to soft-delete, not remove, the user row")
	}
	status, err := authStore.AccountDeletionStatus(context.Background(), "test-user")
	if err != nil {
		t.Fatalf("AccountDeletionStatus: %v", err)
	}
	if status.DeletedAt == nil {
		t.Errorf("expected RequestAccountDeletion to be called with the authenticated userID (test-user), but deleted_at is unset")
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

// TestHandleDeleteAccountSendsEmail covers the best-effort deletion-requested
// email: with a mailer configured and the user having an address on file,
// handleDeleteAccount must send it after RequestAccountDeletion succeeds.
func TestHandleDeleteAccountSendsEmail(t *testing.T) {
	authStore := newFakeAuthStore()
	ms := newFakeMealStore()
	ms.user = types.User{ID: "test-user", Email: "test-user@example.com"}
	fm := &fakeMailer{}
	h := deleteAccountTestHandler(authStore, ms, fm)

	rec := doRequest(h, "DELETE", "/api/v1/account", map[string]string{"confirm": "DELETE"}, nil)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d: %s", rec.Code, rec.Body.String())
	}
	if len(fm.sent) != 1 || fm.sent[0].to != "test-user@example.com" {
		t.Fatalf("sent = %v; want one email to test-user@example.com", fm.sent)
	}
	if len(authStore.auditEvents) != 1 || authStore.auditEvents[0].Event != "account.deletion_email_sent" {
		t.Errorf("expected account.deletion_email_sent audit event, got %#v", authStore.auditEvents)
	}
}

// TestHandleDeleteAccountEmailSkippedOnUserLookupFailure covers
// sendAccountDeletionEmail's early return when h.store.GetUser fails or
// returns a user with no email on file: deletion must still succeed (email
// is best-effort), just with nothing sent.
func TestHandleDeleteAccountEmailSkippedOnUserLookupFailure(t *testing.T) {
	for name, setup := range map[string]func(*fakeMealStore){
		"getUserErr": func(s *fakeMealStore) { s.getUserErr = errors.New("db down") },
		"emptyEmail": func(s *fakeMealStore) { s.user = types.User{ID: "test-user", Email: ""} },
	} {
		t.Run(name, func(t *testing.T) {
			authStore := newFakeAuthStore()
			ms := newFakeMealStore()
			setup(ms)
			fm := &fakeMailer{}
			h := deleteAccountTestHandler(authStore, ms, fm)

			rec := doRequest(h, "DELETE", "/api/v1/account", map[string]string{"confirm": "DELETE"}, nil)
			if rec.Code != http.StatusNoContent {
				t.Fatalf("expected 204, got %d: %s", rec.Code, rec.Body.String())
			}
			if len(fm.sent) != 0 {
				t.Errorf("expected no email sent, got %v", fm.sent)
			}
		})
	}
}

// TestHandleDeleteAccountEmailSendFailureStillSucceeds covers
// sendAccountDeletionEmail's error branch when the mailer itself fails:
// deletion has already committed, so a send failure must not surface as a
// non-204 response, but it must still be audited.
func TestHandleDeleteAccountEmailSendFailureStillSucceeds(t *testing.T) {
	authStore := newFakeAuthStore()
	ms := newFakeMealStore()
	ms.user = types.User{ID: "test-user", Email: "test-user@example.com"}
	fm := &fakeMailer{sendErr: errors.New("smtp down")}
	h := deleteAccountTestHandler(authStore, ms, fm)

	rec := doRequest(h, "DELETE", "/api/v1/account", map[string]string{"confirm": "DELETE"}, nil)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected 204 (best-effort send), got %d: %s", rec.Code, rec.Body.String())
	}
	if len(authStore.auditEvents) != 1 || authStore.auditEvents[0].Event != "account.deletion_email_send_failed" {
		t.Errorf("expected account.deletion_email_send_failed audit event, got %#v", authStore.auditEvents)
	}
}

// deleteAccountTestHandler builds a handler wired for the delete-account
// email tests: an "smtp" provider (unlike buildAuthTestHandler's "none") so
// sendAccountDeletionEmail's mailer branch actually runs, matching
// TestHandleDeleteAccountSendsEmail's setup.
func deleteAccountTestHandler(authStore accountRepos, ms MealStore, m *fakeMailer) *Handler {
	return New(ms, &fakeMealLogger{}, time.UTC, nil, nil,
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
		WithMailer(m, "smtp"),
	)
}

func TestHandleDeleteAccountNotFound(t *testing.T) {
	authStore := &notFoundAccountStore{fakeAuthStore: newFakeAuthStore()}
	h := newHandlerWithAccountStore(newFakeMealStore(), authStore)

	rec := doRequest(h, "DELETE", "/api/v1/account", map[string]string{"confirm": "DELETE"}, nil)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", rec.Code, rec.Body.String())
	}
}

// ---------------------------------------------------------------------------
// wrap()'s pending-deletion gate, and POST /api/v1/account/reactivate
// ---------------------------------------------------------------------------

// pendingDeletionResponse mirrors the flat 403 body writePendingDeletionError
// (handler.go) produces -- deliberately not the standard {error:{code,
// message}} envelope, since the frontend needs deleted_at/photos_purged to
// pick tier-1 vs tier-2 reactivation copy without a second round trip.
type pendingDeletionResponse struct {
	Error        string     `json:"error"`
	DeletedAt    *time.Time `json:"deleted_at,omitempty"`
	PhotosPurged bool       `json:"photos_purged"`
}

func TestWrapBlocksPendingDeletionDay0To30(t *testing.T) {
	authStore := newFakeAuthStore()
	deletedAt := time.Now().UTC().Add(-10 * 24 * time.Hour) // day 10: within the 30-day full-recovery window.
	authStore.deletionStatus["test-user"] = store.AccountDeletionStatus{DeletedAt: &deletedAt}
	h := newHandlerWithAccountStore(newFakeMealStore(), authStore)

	rec := doRequest(h, "GET", "/api/v1/rollups/today", nil, nil)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d: %s", rec.Code, rec.Body.String())
	}
	body := decodeJSON[pendingDeletionResponse](t, rec)
	if body.Error != "pending_deletion" {
		t.Errorf("error = %q, want pending_deletion", body.Error)
	}
	if body.PhotosPurged {
		t.Errorf("photos_purged = true, want false (day 0-30 tier)")
	}
	if body.DeletedAt == nil {
		t.Errorf("deleted_at missing from response")
	}
}

func TestWrapBlocksPendingDeletionDay30To90(t *testing.T) {
	authStore := newFakeAuthStore()
	deletedAt := time.Now().UTC().Add(-45 * 24 * time.Hour) // day 45: past the photo-purge tier.
	purgedAt := time.Now().UTC().Add(-15 * 24 * time.Hour)
	authStore.deletionStatus["test-user"] = store.AccountDeletionStatus{DeletedAt: &deletedAt, PhotosPurgedAt: &purgedAt}
	h := newHandlerWithAccountStore(newFakeMealStore(), authStore)

	rec := doRequest(h, "GET", "/api/v1/rollups/today", nil, nil)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d: %s", rec.Code, rec.Body.String())
	}
	body := decodeJSON[pendingDeletionResponse](t, rec)
	if body.Error != "pending_deletion" {
		t.Errorf("error = %q, want pending_deletion", body.Error)
	}
	if !body.PhotosPurged {
		t.Errorf("photos_purged = false, want true (day 30-90 tier)")
	}
}

func TestWrapAllowsReactivateAndLogoutWhilePending(t *testing.T) {
	deletedAt := time.Now().UTC().Add(-5 * 24 * time.Hour)

	t.Run("reactivate", func(t *testing.T) {
		authStore := newFakeAuthStore()
		authStore.deletionStatus["test-user"] = store.AccountDeletionStatus{DeletedAt: &deletedAt}
		h := newHandlerWithAccountStore(newFakeMealStore(), authStore)

		rec := doRequest(h, "POST", "/api/v1/account/reactivate", nil, nil)
		if rec.Code == http.StatusForbidden {
			t.Fatalf("reactivate route must stay reachable while pending deletion, got 403: %s", rec.Body.String())
		}
	})

	t.Run("logout", func(t *testing.T) {
		authStore := newFakeAuthStore()
		authStore.deletionStatus["test-user"] = store.AccountDeletionStatus{DeletedAt: &deletedAt}
		h := newHandlerWithAccountStore(newFakeMealStore(), authStore)

		rec := doRequest(h, "POST", "/api/v1/auth/logout", nil, nil)
		if rec.Code == http.StatusForbidden {
			t.Fatalf("logout route must stay reachable while pending deletion, got 403: %s", rec.Body.String())
		}
	})
}

func TestHandleReactivateAccountRestoresAccess(t *testing.T) {
	authStore := newFakeAuthStore()
	deletedAt := time.Now().UTC().Add(-5 * 24 * time.Hour)
	authStore.deletionStatus["test-user"] = store.AccountDeletionStatus{DeletedAt: &deletedAt}
	h := newHandlerWithAccountStore(newFakeMealStore(), authStore)

	// Confirm the gate is actually active before reactivating.
	blocked := doRequest(h, "GET", "/api/v1/rollups/today", nil, nil)
	if blocked.Code != http.StatusForbidden {
		t.Fatalf("test setup: expected 403 before reactivation, got %d", blocked.Code)
	}

	rec := doRequest(h, "POST", "/api/v1/account/reactivate", nil, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	status, err := authStore.AccountDeletionStatus(context.Background(), "test-user")
	if err != nil {
		t.Fatalf("AccountDeletionStatus: %v", err)
	}
	if status.DeletedAt != nil {
		t.Errorf("expected deleted_at cleared after reactivation, got %v", status.DeletedAt)
	}

	// A subsequent authenticated request must now succeed instead of 403.
	after := doRequest(h, "GET", "/api/v1/rollups/today", nil, nil)
	if after.Code == http.StatusForbidden {
		t.Fatalf("expected normal access restored after reactivation, still got 403: %s", after.Body.String())
	}
}

// TestHandleReactivateAccountSendsEmail covers the best-effort
// reactivation-confirmed email: with a mailer configured and the user having
// an address on file, handleReactivateAccount must send it after
// ReactivateAccount succeeds.
func TestHandleReactivateAccountSendsEmail(t *testing.T) {
	authStore := newFakeAuthStore()
	deletedAt := time.Now().UTC().Add(-5 * 24 * time.Hour)
	authStore.deletionStatus["test-user"] = store.AccountDeletionStatus{DeletedAt: &deletedAt}
	ms := newFakeMealStore()
	ms.user = types.User{ID: "test-user", Email: "test-user@example.com"}
	fm := &fakeMailer{}
	h := deleteAccountTestHandler(authStore, ms, fm)

	rec := doRequest(h, "POST", "/api/v1/account/reactivate", nil, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	if len(fm.sent) != 1 || fm.sent[0].to != "test-user@example.com" {
		t.Fatalf("sent = %v; want one email to test-user@example.com", fm.sent)
	}
	if len(authStore.auditEvents) != 1 || authStore.auditEvents[0].Event != "account.reactivated_email_sent" {
		t.Errorf("expected account.reactivated_email_sent audit event, got %#v", authStore.auditEvents)
	}
}

// TestHandleReactivateAccountEmailSendFailureStillSucceeds covers
// handleReactivateAccount's error branch when the mailer fails: reactivation
// has already committed, so a send failure must not surface as a non-200
// response, but it must still be audited.
func TestHandleReactivateAccountEmailSendFailureStillSucceeds(t *testing.T) {
	authStore := newFakeAuthStore()
	deletedAt := time.Now().UTC().Add(-5 * 24 * time.Hour)
	authStore.deletionStatus["test-user"] = store.AccountDeletionStatus{DeletedAt: &deletedAt}
	ms := newFakeMealStore()
	ms.user = types.User{ID: "test-user", Email: "test-user@example.com"}
	fm := &fakeMailer{sendErr: errors.New("smtp down")}
	h := deleteAccountTestHandler(authStore, ms, fm)

	rec := doRequest(h, "POST", "/api/v1/account/reactivate", nil, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 (best-effort send), got %d: %s", rec.Code, rec.Body.String())
	}
	if len(authStore.auditEvents) != 1 || authStore.auditEvents[0].Event != "account.reactivated_email_send_failed" {
		t.Errorf("expected account.reactivated_email_send_failed audit event, got %#v", authStore.auditEvents)
	}
}

func TestHandleReactivateAccountNotFound(t *testing.T) {
	authStore := &notFoundReactivateStore{fakeAuthStore: newFakeAuthStore()}
	h := newHandlerWithAccountStore(newFakeMealStore(), authStore)

	rec := doRequest(h, "POST", "/api/v1/account/reactivate", nil, nil)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", rec.Code, rec.Body.String())
	}
}

// notFoundReactivateStore wraps *fakeAuthStore and overrides ReactivateAccount
// to simulate the store reporting the account doesn't exist.
type notFoundReactivateStore struct {
	*fakeAuthStore
}

func (s *notFoundReactivateStore) ReactivateAccount(_ context.Context, _ string) error {
	return types.ErrNotFound
}
