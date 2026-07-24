package api

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"strings"
	"testing"
	"time"

	"github.com/gsaraiva2109/dietdaemon/core/types"
	"github.com/gsaraiva2109/dietdaemon/internal/auth"
)

// ---------------------------------------------------------------------------
// decodeRegisterRequest / decodeLoginRequest edge cases
// ---------------------------------------------------------------------------

func TestHandleRegisterInvalidJSON(t *testing.T) {
	store := newAuthHandlerTestStore()
	h, _ := newAuthHandlerForTest(store, testAuthConfig())

	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", strings.NewReader("not json"))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer test-api-key")
	rec := httptest.NewRecorder()
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("invalid JSON register status = %d, want 400", rec.Code)
	}
}

func TestHandleRegisterMissingFields(t *testing.T) {
	store := newAuthHandlerTestStore()
	h, _ := newAuthHandlerForTest(store, testAuthConfig())

	cases := []map[string]string{
		{"email": "", "password": "correct horse battery staple"},
		{"email": "missing-password@example.com", "password": ""},
	}
	for _, body := range cases {
		rec := doRequest(h, http.MethodPost, "/api/v1/auth/register", body, nil)
		if rec.Code != http.StatusBadRequest {
			t.Errorf("register with %+v status = %d, want 400", body, rec.Code)
		}
	}
}

func TestHandleLoginInvalidJSON(t *testing.T) {
	store := newAuthHandlerTestStore()
	h, _ := newAuthHandlerForTest(store, testAuthConfig())

	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", strings.NewReader("not json"))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer test-api-key")
	rec := httptest.NewRecorder()
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("invalid JSON login status = %d, want 400", rec.Code)
	}
}

func TestHandleLoginMissingFields(t *testing.T) {
	store := newAuthHandlerTestStore()
	h, _ := newAuthHandlerForTest(store, testAuthConfig())

	cases := []map[string]string{
		{"email": "", "password": "whatever"},
		{"email": "someone@example.com", "password": ""},
	}
	for _, body := range cases {
		rec := doRequest(h, http.MethodPost, "/api/v1/auth/login", body, nil)
		if rec.Code != http.StatusBadRequest {
			t.Errorf("login with %+v status = %d, want 400", body, rec.Code)
		}
	}
}

// ---------------------------------------------------------------------------
// finishRegistrationEmail: mailer-configured branch
// ---------------------------------------------------------------------------

func TestHandleRegisterSendsVerificationEmailWhenMailerConfigured(t *testing.T) {
	authStore := newEmailTestAuthStore()
	fm := &fakeMailer{}
	meals := newFakeMealStore()
	h := New(meals, &fakeMealLogger{}, time.UTC, nil, nil,
		WithAuth(authStore, AuthRepos{Sessions: authStore, LoginAttempts: authStore, TOTP: authStore, MFAChallenges: authStore, RecoveryCodes: authStore}, nil, "DietDaemon", testAuthConfig()),
		WithMailer(fm, "smtp"),
		WithPublicBaseURL("http://localhost:8080"),
	)

	rec := doRequest(h, http.MethodPost, "/api/v1/auth/register", map[string]string{
		"email":    "new@example.com",
		"password": "correct horse battery staple",
	}, nil)
	if rec.Code != http.StatusCreated {
		t.Fatalf("register status = %d: %s", rec.Code, rec.Body.String())
	}

	got := decodeJSON[sessionResponse](t, rec)
	if got.User.EmailVerified {
		t.Error("user should not be auto-verified when a mailer is configured")
	}
	if len(fm.sent) != 1 || fm.sent[0].to != "new@example.com" {
		t.Errorf("expected verification email sent to new@example.com, got %#v", fm.sent)
	}
	found := false
	for _, ev := range authStore.auditEvents {
		if ev.Event == "email.verification_sent" {
			found = true
		}
	}
	if !found {
		t.Error("expected email.verification_sent audit event")
	}
}

func TestHandleRegisterCreateEmailTokenFailure(t *testing.T) {
	authStore := newEmailTestAuthStore()
	authStore.createEmailTokenErr = errors.New("store unavailable")
	fm := &fakeMailer{}
	meals := newFakeMealStore()
	h := New(meals, &fakeMealLogger{}, time.UTC, nil, nil,
		WithAuth(authStore, AuthRepos{Sessions: authStore, LoginAttempts: authStore, TOTP: authStore, MFAChallenges: authStore, RecoveryCodes: authStore}, nil, "DietDaemon", testAuthConfig()),
		WithMailer(fm, "smtp"),
		WithPublicBaseURL("http://localhost:8080"),
	)

	rec := doRequest(h, http.MethodPost, "/api/v1/auth/register", map[string]string{
		"email":    "new2@example.com",
		"password": "correct horse battery staple",
	}, nil)
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("register status = %d, want 500: %s", rec.Code, rec.Body.String())
	}
	if len(fm.sent) != 0 {
		t.Error("must not send verification email when token creation fails")
	}
}

func TestHandleRegisterHashPasswordTooShort(t *testing.T) {
	store := newAuthHandlerTestStore()
	h, _ := newAuthHandlerForTest(store, testAuthConfig())

	rec := doRequest(h, http.MethodPost, "/api/v1/auth/register", map[string]string{
		"email":    "short@example.com",
		"password": "short",
	}, nil)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("register with too-short password status = %d, want 400: %s", rec.Code, rec.Body.String())
	}
}

// ---------------------------------------------------------------------------
// checkLoginLockout
// ---------------------------------------------------------------------------

func TestCheckLoginLockoutStoreError(t *testing.T) {
	store := newAuthHandlerTestStore()
	store.recentFailedAttemptsErr = errors.New("store unavailable")
	h, _ := newAuthHandlerForTest(store, testAuthConfig())

	rec := doRequest(h, http.MethodPost, "/api/v1/auth/login", map[string]string{
		"email": "anyone@example.com", "password": "whatever-password",
	}, nil)
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("login status = %d, want 500: %s", rec.Code, rec.Body.String())
	}
}

func TestCheckLoginLockoutLocked(t *testing.T) {
	store := newAuthHandlerTestStore()
	cfg := testAuthConfig()
	h, _ := newAuthHandlerForTest(store, cfg)

	email := "locked-out@example.com"
	for range cfg.LockoutCfg.MaxAttempts {
		_ = store.RecordLoginAttempt(context.Background(), email, false)
	}

	rec := doRequest(h, http.MethodPost, "/api/v1/auth/login", map[string]string{
		"email": email, "password": "whatever-password",
	}, nil)
	if rec.Code != http.StatusTooManyRequests {
		t.Fatalf("locked-out login status = %d, want 429: %s", rec.Code, rec.Body.String())
	}
	if rec.Header().Get("Retry-After") == "" {
		t.Error("expected Retry-After header on lockout")
	}
}

// ---------------------------------------------------------------------------
// verifyLoginCredentials: user found by email but has no password hash
// (e.g. an OIDC-only account) must still fail like "wrong password".
// ---------------------------------------------------------------------------

func TestHandleLoginUserWithoutPasswordHash(t *testing.T) {
	store := newAuthHandlerTestStore()
	user := types.User{ID: "oidc-only", Email: "oidc-only@example.com", Status: "active", CreatedAt: time.Now().UTC()}
	store.users[user.ID] = user
	store.userByEmail[user.Email] = user
	// Deliberately no store.phcHash entry for this user.
	h, _ := newAuthHandlerForTest(store, testAuthConfig())

	rec := doRequest(h, http.MethodPost, "/api/v1/auth/login", map[string]string{
		"email": user.Email, "password": "whatever-password",
	}, nil)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("login status = %d, want 401: %s", rec.Code, rec.Body.String())
	}
}

// ---------------------------------------------------------------------------
// tryLoginMFAStepUp
// ---------------------------------------------------------------------------

func TestHandleLoginMFAStepUp(t *testing.T) {
	authStore := newTOTPTestStore()
	password := "correct horse battery staple"
	phc, err := auth.Hash(password)
	if err != nil {
		t.Fatalf("Hash: %v", err)
	}
	user := types.User{ID: "mfa-user", AccountID: "acct-1", Email: "mfa@example.com", Status: "active", CreatedAt: time.Now().UTC()}
	authStore.users[user.ID] = user
	authStore.userByEmail[user.Email] = user
	authStore.phcHash[user.ID] = phc
	authStore.confirmedByUser[user.ID] = true

	meals := newFakeMealStore()
	meals.user = user
	h := buildTOTPHandler(authStore, meals, nil, auth.DefaultLockoutConfig())

	rec := doRequest(h, http.MethodPost, "/api/v1/auth/login", map[string]string{
		"email": user.Email, "password": password,
	}, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("login status = %d: %s", rec.Code, rec.Body.String())
	}
	body := decodeJSON[map[string]any](t, rec)
	if mfaRequired, _ := body["mfa_required"].(bool); !mfaRequired {
		t.Errorf("expected mfa_required=true, got %#v", body)
	}
	if body["challenge_token"] == "" || body["challenge_token"] == nil {
		t.Error("expected a non-empty challenge_token")
	}
	if len(authStore.challenges) != 1 {
		t.Errorf("expected one MFA challenge stored, got %d", len(authStore.challenges))
	}
}

func TestHandleLoginMFAStepUpChallengeCreationFails(t *testing.T) {
	authStore := newTOTPTestStore()
	authStore.createMFAChallengeErr = errors.New("store unavailable")
	password := "correct horse battery staple"
	phc, err := auth.Hash(password)
	if err != nil {
		t.Fatalf("Hash: %v", err)
	}
	user := types.User{ID: "mfa-user2", Email: "mfa2@example.com", Status: "active", CreatedAt: time.Now().UTC()}
	authStore.users[user.ID] = user
	authStore.userByEmail[user.Email] = user
	authStore.phcHash[user.ID] = phc
	authStore.confirmedByUser[user.ID] = true

	meals := newFakeMealStore()
	meals.user = user
	h := buildTOTPHandler(authStore, meals, nil, auth.DefaultLockoutConfig())

	rec := doRequest(h, http.MethodPost, "/api/v1/auth/login", map[string]string{
		"email": user.Email, "password": password,
	}, nil)
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("login status = %d, want 500: %s", rec.Code, rec.Body.String())
	}
}

func TestHandleLoginHasConfirmedTOTPErrorFallsThroughToNormalLogin(t *testing.T) {
	authStore := newTOTPTestStore()
	authStore.hasConfirmedErr = errors.New("lookup failed")
	password := "correct horse battery staple"
	phc, err := auth.Hash(password)
	if err != nil {
		t.Fatalf("Hash: %v", err)
	}
	user := types.User{ID: "mfa-user3", Email: "mfa3@example.com", Status: "active", CreatedAt: time.Now().UTC()}
	authStore.users[user.ID] = user
	authStore.userByEmail[user.Email] = user
	authStore.phcHash[user.ID] = phc

	meals := newFakeMealStore()
	meals.user = user
	h := buildTOTPHandler(authStore, meals, nil, auth.DefaultLockoutConfig())

	rec := doRequest(h, http.MethodPost, "/api/v1/auth/login", map[string]string{
		"email": user.Email, "password": password,
	}, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("login status = %d, want 200 (normal login on lookup error): %s", rec.Code, rec.Body.String())
	}
	got := decodeJSON[sessionResponse](t, rec)
	if got.User.ID != user.ID {
		t.Errorf("expected normal session response for %q, got %+v", user.ID, got.User)
	}
}

// ---------------------------------------------------------------------------
// handleSession error branch
// ---------------------------------------------------------------------------

func TestHandleSessionGetUserError(t *testing.T) {
	store := newAuthHandlerTestStore()
	h, meals := newAuthHandlerForTest(store, testAuthConfig())
	meals.getUserErr = types.ErrNotFound

	rec := doRequest(h, http.MethodGet, "/api/v1/auth/session", nil, nil)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("session status = %d, want 404: %s", rec.Code, rec.Body.String())
	}
}

// ---------------------------------------------------------------------------
// handleChangePassword edge cases
// ---------------------------------------------------------------------------

func TestHandleChangePasswordInvalidJSON(t *testing.T) {
	store := newAuthHandlerTestStore()
	h, _ := newAuthHandlerForTest(store, testAuthConfig())

	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/change-password", strings.NewReader("not json"))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer test-api-key")
	rec := httptest.NewRecorder()
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("invalid JSON change-password status = %d, want 400", rec.Code)
	}
}

func TestHandleChangePasswordMissingFields(t *testing.T) {
	store := newAuthHandlerTestStore()
	h, _ := newAuthHandlerForTest(store, testAuthConfig())

	cases := []map[string]string{
		{"current_password": "", "new_password": "newSecurePassword123!"},
		{"current_password": "correct horse battery staple", "new_password": ""},
	}
	for _, body := range cases {
		rec := doRequest(h, http.MethodPost, "/api/v1/auth/change-password", body, nil)
		if rec.Code != http.StatusBadRequest {
			t.Errorf("change-password with %+v status = %d, want 400", body, rec.Code)
		}
	}
}

func TestHandleChangePasswordNoExistingHash(t *testing.T) {
	store := newAuthHandlerTestStore()
	h, _ := newAuthHandlerForTest(store, testAuthConfig())
	// No store.phcHash["test-user"] entry — GetPasswordHash returns ErrNotFound.

	rec := doRequest(h, http.MethodPost, "/api/v1/auth/change-password", map[string]string{
		"current_password": "whatever", "new_password": "newSecurePassword123!",
	}, nil)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("change-password status = %d, want 404: %s", rec.Code, rec.Body.String())
	}
}

func TestHandleChangePasswordWrongCurrentPassword(t *testing.T) {
	store := newAuthHandlerTestStore()
	oldHash, err := auth.Hash("correct horse battery staple")
	if err != nil {
		t.Fatal(err)
	}
	store.phcHash["test-user"] = oldHash
	h, _ := newAuthHandlerForTest(store, testAuthConfig())

	rec := doRequest(h, http.MethodPost, "/api/v1/auth/change-password", map[string]string{
		"current_password": "wrong password", "new_password": "newSecurePassword123!",
	}, nil)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("change-password status = %d, want 401: %s", rec.Code, rec.Body.String())
	}
}

func TestHandleChangePasswordNewPasswordTooShort(t *testing.T) {
	store := newAuthHandlerTestStore()
	oldHash, err := auth.Hash("correct horse battery staple")
	if err != nil {
		t.Fatal(err)
	}
	store.phcHash["test-user"] = oldHash
	h, _ := newAuthHandlerForTest(store, testAuthConfig())

	rec := doRequest(h, http.MethodPost, "/api/v1/auth/change-password", map[string]string{
		"current_password": "correct horse battery staple", "new_password": "short",
	}, nil)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("change-password status = %d, want 400: %s", rec.Code, rec.Body.String())
	}
}

func TestHandleChangePasswordSuccess(t *testing.T) {
	store := newAuthHandlerTestStore()
	oldHash, err := auth.Hash("correct horse battery staple")
	if err != nil {
		t.Fatal(err)
	}
	store.phcHash["test-user"] = oldHash
	h, _ := newAuthHandlerForTest(store, testAuthConfig())

	rec := doRequest(h, http.MethodPost, "/api/v1/auth/change-password", map[string]string{
		"current_password": "correct horse battery staple", "new_password": "newSecurePassword123!",
	}, nil)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("change-password status = %d, want 204: %s", rec.Code, rec.Body.String())
	}
	if store.phcHash["test-user"] == oldHash {
		t.Error("password hash should have changed")
	}
	if len(store.sessions) != 1 {
		t.Errorf("expected a fresh session after password change, got %d", len(store.sessions))
	}
	foundAudit := false
	for _, ev := range store.auditEvents {
		if ev.Event == "user.password_changed" {
			foundAudit = true
		}
	}
	if !foundAudit {
		t.Error("expected user.password_changed audit event")
	}
}

// ---------------------------------------------------------------------------
// handleProviders
// ---------------------------------------------------------------------------

func TestHandleProvidersEmpty(t *testing.T) {
	store := newAuthHandlerTestStore()
	h, _ := newAuthHandlerForTest(store, testAuthConfig())

	rec := doRequest(h, http.MethodGet, "/api/v1/auth/providers", nil, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("providers status = %d: %s", rec.Code, rec.Body.String())
	}
	got := decodeJSON[providersResponse](t, rec)
	if got.Providers == nil || len(got.Providers) != 0 {
		t.Errorf("expected empty providers slice, got %#v", got.Providers)
	}
	if got.RegistrationMode != string(types.RegistrationOpen) {
		t.Errorf("registration mode = %q, want %q", got.RegistrationMode, types.RegistrationOpen)
	}
}

// ---------------------------------------------------------------------------
// Credential CRUD: API keys and share tokens
// ---------------------------------------------------------------------------

// credAuthStore adds error-injection for the API-key / share-token CRUD
// paths that fakeAuthStore always succeeds at.
type credAuthStore struct {
	*fakeAuthStore
	listAPIKeysErr     error
	createAPIKeyErr    error
	listShareTokensErr error
}

func (s *credAuthStore) ListAPIKeys(ctx context.Context, userID string) ([]types.APIKey, error) {
	if s.listAPIKeysErr != nil {
		return nil, s.listAPIKeysErr
	}
	return s.fakeAuthStore.ListAPIKeys(ctx, userID)
}

func (s *credAuthStore) CreateAPIKey(ctx context.Context, id, userID, hashedKey, label string) error {
	if s.createAPIKeyErr != nil {
		return s.createAPIKeyErr
	}
	return s.fakeAuthStore.CreateAPIKey(ctx, id, userID, hashedKey, label)
}

func (s *credAuthStore) ListShareTokens(ctx context.Context, userID string) ([]types.ShareToken, error) {
	if s.listShareTokensErr != nil {
		return nil, s.listShareTokensErr
	}
	return s.fakeAuthStore.ListShareTokens(ctx, userID)
}

func buildCredHandler(authStore *credAuthStore) *Handler {
	meals := newFakeMealStore()
	meals.user = types.User{ID: "test-user", Email: "test@example.com", Status: "active", CreatedAt: time.Now().UTC()}
	return New(meals, &fakeMealLogger{}, time.UTC, nil, nil,
		WithAuth(authStore, AuthRepos{Sessions: authStore, LoginAttempts: authStore, TOTP: authStore, MFAChallenges: authStore, RecoveryCodes: authStore}, nil, "DietDaemon", testAuthConfig()),
		WithMailer(&fakeMailer{}, "none"),
		WithPublicBaseURL("http://localhost:8080"),
	)
}

func TestHandleListAPIKeysEmptyAndPopulated(t *testing.T) {
	authStore := &credAuthStore{fakeAuthStore: newFakeAuthStore()}
	h := buildCredHandler(authStore)

	rec := doRequest(h, http.MethodGet, "/api/v1/auth/api-keys", nil, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("list api-keys status = %d: %s", rec.Code, rec.Body.String())
	}
	keys := decodeJSON[[]types.APIKey](t, rec)
	if keys == nil || len(keys) != 0 {
		t.Errorf("expected empty array, got %#v", keys)
	}

	createRec := doRequest(h, http.MethodPost, "/api/v1/auth/api-keys", map[string]string{"label": "my key"}, nil)
	if createRec.Code != http.StatusCreated {
		t.Fatalf("create api-key status = %d: %s", createRec.Code, createRec.Body.String())
	}
	created := decodeJSON[types.NewAPIKeyResponse](t, createRec)
	if created.Key == "" || created.Label != "my key" || created.UserID != "test-user" {
		t.Errorf("unexpected created api key: %+v", created)
	}
	foundAudit := false
	for _, ev := range authStore.auditEvents {
		if ev.Event == "api_key.created" {
			foundAudit = true
		}
	}
	if !foundAudit {
		t.Error("expected api_key.created audit event")
	}

	rec2 := doRequest(h, http.MethodGet, "/api/v1/auth/api-keys", nil, nil)
	keys2 := decodeJSON[[]types.APIKey](t, rec2)
	if len(keys2) != 1 {
		t.Fatalf("expected one api key after creation, got %d", len(keys2))
	}

	revokeRec := doRequest(h, http.MethodDelete, "/api/v1/auth/api-keys/"+keys2[0].ID, nil, nil)
	if revokeRec.Code != http.StatusNoContent {
		t.Fatalf("revoke api-key status = %d, want 204: %s", revokeRec.Code, revokeRec.Body.String())
	}
}

func TestHandleListAPIKeysStoreError(t *testing.T) {
	authStore := &credAuthStore{fakeAuthStore: newFakeAuthStore(), listAPIKeysErr: errors.New("store unavailable")}
	h := buildCredHandler(authStore)

	rec := doRequest(h, http.MethodGet, "/api/v1/auth/api-keys", nil, nil)
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("list api-keys status = %d, want 500: %s", rec.Code, rec.Body.String())
	}
}

func TestHandleCreateAPIKeyInvalidJSON(t *testing.T) {
	authStore := &credAuthStore{fakeAuthStore: newFakeAuthStore()}
	h := buildCredHandler(authStore)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/api-keys", strings.NewReader("not json"))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer test-api-key")
	rec := httptest.NewRecorder()
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("invalid JSON create api-key status = %d, want 400", rec.Code)
	}
}

func TestHandleCreateAPIKeyDefaultLabel(t *testing.T) {
	authStore := &credAuthStore{fakeAuthStore: newFakeAuthStore()}
	h := buildCredHandler(authStore)

	rec := doRequest(h, http.MethodPost, "/api/v1/auth/api-keys", map[string]string{"label": "   "}, nil)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create api-key status = %d: %s", rec.Code, rec.Body.String())
	}
	created := decodeJSON[types.NewAPIKeyResponse](t, rec)
	if created.Label != "default" {
		t.Errorf("label = %q, want default", created.Label)
	}
}

func TestHandleCreateAPIKeyStoreError(t *testing.T) {
	authStore := &credAuthStore{fakeAuthStore: newFakeAuthStore(), createAPIKeyErr: errors.New("store unavailable")}
	h := buildCredHandler(authStore)

	rec := doRequest(h, http.MethodPost, "/api/v1/auth/api-keys", map[string]string{"label": "k"}, nil)
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("create api-key status = %d, want 500: %s", rec.Code, rec.Body.String())
	}
}

func TestHandleRevokeAPIKeyNotFound(t *testing.T) {
	// fakeAuthStore.RevokeAPIKey always succeeds regardless of id, so this
	// exercises the success path with a non-existent id (204 either way —
	// revocation is idempotent at this layer).
	authStore := &credAuthStore{fakeAuthStore: newFakeAuthStore()}
	h := buildCredHandler(authStore)

	rec := doRequest(h, http.MethodDelete, "/api/v1/auth/api-keys/does-not-exist", nil, nil)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("revoke api-key status = %d, want 204: %s", rec.Code, rec.Body.String())
	}
}

func TestHandleListShareTokensEmptyAndPopulated(t *testing.T) {
	authStore := &credAuthStore{fakeAuthStore: newFakeAuthStore()}
	h := buildCredHandler(authStore)

	rec := doRequest(h, http.MethodGet, "/api/v1/auth/share-tokens", nil, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("list share-tokens status = %d: %s", rec.Code, rec.Body.String())
	}
	toks := decodeJSON[[]types.ShareToken](t, rec)
	if toks == nil || len(toks) != 0 {
		t.Errorf("expected empty array, got %#v", toks)
	}

	createRec := doRequest(h, http.MethodPost, "/api/v1/auth/share-tokens", map[string]string{"label": "dashboard"}, nil)
	if createRec.Code != http.StatusCreated {
		t.Fatalf("create share-token status = %d: %s", createRec.Code, createRec.Body.String())
	}
	created := decodeJSON[types.NewShareTokenResponse](t, createRec)
	if created.Token == "" || created.Label != "dashboard" {
		t.Errorf("unexpected created share token: %+v", created)
	}

	rec2 := doRequest(h, http.MethodGet, "/api/v1/auth/share-tokens", nil, nil)
	toks2 := decodeJSON[[]types.ShareToken](t, rec2)
	if len(toks2) != 1 {
		t.Fatalf("expected one share token after creation, got %d", len(toks2))
	}

	revokeRec := doRequest(h, http.MethodDelete, "/api/v1/auth/share-tokens/"+toks2[0].ID, nil, nil)
	if revokeRec.Code != http.StatusNoContent {
		t.Fatalf("revoke share-token status = %d, want 204: %s", revokeRec.Code, revokeRec.Body.String())
	}
	foundAudit := false
	for _, ev := range authStore.auditEvents {
		if ev.Event == "share_token.revoked" {
			foundAudit = true
		}
	}
	if !foundAudit {
		t.Error("expected share_token.revoked audit event")
	}
}

func TestHandleListShareTokensStoreError(t *testing.T) {
	authStore := &credAuthStore{fakeAuthStore: newFakeAuthStore(), listShareTokensErr: errors.New("store unavailable")}
	h := buildCredHandler(authStore)

	rec := doRequest(h, http.MethodGet, "/api/v1/auth/share-tokens", nil, nil)
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("list share-tokens status = %d, want 500: %s", rec.Code, rec.Body.String())
	}
}

func TestHandleRevokeShareTokenNotFound(t *testing.T) {
	authStore := &credAuthStore{fakeAuthStore: newFakeAuthStore()}
	h := buildCredHandler(authStore)

	rec := doRequest(h, http.MethodDelete, "/api/v1/auth/share-tokens/does-not-exist", nil, nil)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("revoke share-token status = %d, want 404: %s", rec.Code, rec.Body.String())
	}
}

// ---------------------------------------------------------------------------
// Pure helper functions: isSixDigit, hostOnly, isTrustedProxy
// ---------------------------------------------------------------------------

func TestIsSixDigit(t *testing.T) {
	cases := []struct {
		in   string
		want bool
	}{
		{"123456", true},
		{"12345", false},   // too short
		{"1234567", false}, // too long
		{"12a456", false},  // non-digit
		{"", false},
	}
	for _, tc := range cases {
		if got := isSixDigit(tc.in); got != tc.want {
			t.Errorf("isSixDigit(%q) = %v, want %v", tc.in, got, tc.want)
		}
	}
}

func TestHostOnly(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{"127.0.0.1:8080", "127.0.0.1"},
		{"[::1]:8080", "::1"},
		{"not-a-host-port", "not-a-host-port"},
	}
	for _, tc := range cases {
		if got := hostOnly(tc.in); got != tc.want {
			t.Errorf("hostOnly(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestIsTrustedProxyInvalidAddr(t *testing.T) {
	h := &Handler{trustedProxies: []netip.Prefix{netip.MustParsePrefix("127.0.0.0/8")}}
	if h.isTrustedProxy("not-an-ip") {
		t.Error("isTrustedProxy should return false for an unparsable address")
	}
}

func TestIsTrustedProxyNoConfig(t *testing.T) {
	h := &Handler{}
	if h.isTrustedProxy("127.0.0.1") {
		t.Error("isTrustedProxy should return false when no proxies are configured")
	}
}
