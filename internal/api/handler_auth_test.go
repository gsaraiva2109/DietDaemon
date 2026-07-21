package api

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gsaraiva2109/dietdaemon/core/types"
	"github.com/gsaraiva2109/dietdaemon/internal/auth"
)

// authHandlerTestStore adds observable session writes to the shared auth fake.
type authHandlerTestStore struct {
	*fakeAuthStore
	sessions map[string]auth.Session
	deleted  []string
}

func newAuthHandlerTestStore() *authHandlerTestStore {
	return &authHandlerTestStore{
		fakeAuthStore: newFakeAuthStore(),
		sessions:      make(map[string]auth.Session),
	}
}

func (s *authHandlerTestStore) CreateSession(_ context.Context, sess auth.Session) error {
	s.sessions[sess.ID] = sess
	return nil
}

func (s *authHandlerTestStore) GetSession(_ context.Context, id string) (auth.Session, error) {
	sess, ok := s.sessions[id]
	if !ok {
		return auth.Session{}, types.ErrNotFound
	}
	return sess, nil
}

func (s *authHandlerTestStore) DeleteSession(_ context.Context, id string) error {
	delete(s.sessions, id)
	s.deleted = append(s.deleted, id)
	return nil
}

func newAuthHandlerForTest(store *authHandlerTestStore, cfg AuthConfig) (*Handler, *fakeMealStore) {
	meals := newFakeMealStore()
	meals.user = types.User{
		ID:          "test-user",
		AccountID:   "acct-1",
		Email:       "test@example.com",
		DisplayName: "Test User",
		Status:      "active",
		CreatedAt:   time.Now().UTC(),
	}
	h := New(meals, &fakeMealLogger{}, time.UTC, nil, nil,
		WithAuth(store, store, store, store, store, store, nil, "DietDaemon", cfg),
		WithMailer(&fakeMailer{}, "none"),
		WithPublicBaseURL("http://localhost:8080"),
	)
	return h, meals
}

func testAuthConfig() AuthConfig {
	return AuthConfig{
		SessionCfg: auth.SessionConfig{
			IdleTTL:     time.Hour,
			AbsoluteTTL: 24 * time.Hour,
			RememberTTL: 72 * time.Hour,
		},
		LockoutCfg:       auth.DefaultLockoutConfig(),
		RegistrationMode: types.RegistrationOpen,
	}
}

func cookieNamed(t *testing.T, rec *httptest.ResponseRecorder, name string) *http.Cookie {
	t.Helper()
	for _, cookie := range rec.Result().Cookies() {
		if cookie.Name == name {
			return cookie
		}
	}
	t.Fatalf("response did not set %s cookie", name)
	return nil
}

func TestHandleRegisterCreatesSession(t *testing.T) {
	store := newAuthHandlerTestStore()
	h, _ := newAuthHandlerForTest(store, testAuthConfig())

	rec := doRequest(h, http.MethodPost, "/api/v1/auth/register", map[string]string{
		"email":        "new@example.com",
		"password":     "correct horse battery staple",
		"display_name": "New User",
	}, nil)
	if rec.Code != http.StatusCreated {
		t.Fatalf("register status = %d: %s", rec.Code, rec.Body.String())
	}

	got := decodeJSON[sessionResponse](t, rec)
	if got.User.Email != "new@example.com" || got.User.DisplayName != "New User" {
		t.Errorf("registered user = %+v", got.User)
	}
	if len(store.sessions) != 1 {
		t.Fatalf("created sessions = %d, want 1", len(store.sessions))
	}
	if cookieNamed(t, rec, "dd_session").HttpOnly != true {
		t.Error("dd_session must be HttpOnly")
	}
	if cookieNamed(t, rec, "dd_csrf").HttpOnly {
		t.Error("dd_csrf must be readable by JavaScript")
	}
}

func TestRegistrationAllowed(t *testing.T) {
	cases := []struct {
		name      string
		mode      types.RegistrationMode
		multiUser bool
		userCount int
		viaOIDC   bool
		want      bool
	}{
		{"open/single-user/no-existing/password", types.RegistrationOpen, false, 0, false, true},
		{"open/single-user/existing/password", types.RegistrationOpen, false, 1, false, false},
		{"open/multi-user/existing/password", types.RegistrationOpen, true, 5, false, true},
		{"invite/single-user/no-existing/password", types.RegistrationInvite, false, 0, false, true},
		{"invite/single-user/existing/password", types.RegistrationInvite, false, 1, false, false},
		{"invite/multi-user/no-existing/oidc", types.RegistrationInvite, true, 0, true, true},
		{"invite/multi-user/existing/oidc", types.RegistrationInvite, true, 1, true, false},
		{"oidc-only/multi-user/password-blocked", types.RegistrationOIDCOnly, true, 0, false, false},
		{"oidc-only/multi-user/oidc-allowed", types.RegistrationOIDCOnly, true, 0, true, true},
		{"oidc-only/single-user/existing/oidc-capped", types.RegistrationOIDCOnly, false, 1, true, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			store := newFakeAuthStore()
			store.userCount = tc.userCount
			h := &Handler{authStore: store, registrationMode: tc.mode, multiUser: tc.multiUser}
			got, err := h.registrationAllowed(context.Background(), tc.viaOIDC)
			if err != nil {
				t.Fatalf("registrationAllowed: %v", err)
			}
			if got != tc.want {
				t.Errorf("registrationAllowed(viaOIDC=%v) = %v, want %v", tc.viaOIDC, got, tc.want)
			}
		})
	}
}

// erroringCountAuthStore forces CountUsers to fail, to exercise
// registrationAllowed's error propagation path.
type erroringCountAuthStore struct {
	*fakeAuthStore
}

func (s *erroringCountAuthStore) CountUsers(_ context.Context) (int, error) {
	return 0, errors.New("boom")
}

func TestRegistrationAllowedCountUsersError(t *testing.T) {
	store := &erroringCountAuthStore{fakeAuthStore: newFakeAuthStore()}
	h := &Handler{authStore: store, registrationMode: types.RegistrationOpen, multiUser: false}

	allowed, err := h.registrationAllowed(context.Background(), false)
	if err == nil {
		t.Fatal("registrationAllowed: expected error from CountUsers")
	}
	if allowed {
		t.Error("registrationAllowed: allowed must be false on error")
	}
}

func TestHandleRegisterBlockedWhenMultiUserFalseAndOneUserExists(t *testing.T) {
	store := newAuthHandlerTestStore()
	store.userCount = 1
	h, _ := newAuthHandlerForTest(store, testAuthConfig())

	rec := doRequest(h, http.MethodPost, "/api/v1/auth/register", map[string]string{
		"email":    "second@example.com",
		"password": "correct horse battery staple",
	}, nil)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("register status = %d: %s", rec.Code, rec.Body.String())
	}
	if len(store.sessions) != 0 {
		t.Errorf("created sessions = %d, want 0", len(store.sessions))
	}
}

func TestHandleRegisterAllowedWhenMultiUserTrueUnderOpenMode(t *testing.T) {
	store := newAuthHandlerTestStore()
	store.userCount = 1
	cfg := testAuthConfig()
	cfg.MultiUser = true
	h, _ := newAuthHandlerForTest(store, cfg)

	rec := doRequest(h, http.MethodPost, "/api/v1/auth/register", map[string]string{
		"email":    "second@example.com",
		"password": "correct horse battery staple",
	}, nil)
	if rec.Code != http.StatusCreated {
		t.Fatalf("register status = %d: %s", rec.Code, rec.Body.String())
	}
}

func TestHandleRegisterDuplicateEmail(t *testing.T) {
	store := newAuthHandlerTestStore()
	store.userByEmail["taken@example.com"] = types.User{ID: "taken", Email: "taken@example.com"}
	h, _ := newAuthHandlerForTest(store, testAuthConfig())

	rec := doRequest(h, http.MethodPost, "/api/v1/auth/register", map[string]string{
		"email":    "taken@example.com",
		"password": "correct horse battery staple",
	}, nil)
	if rec.Code != http.StatusConflict {
		t.Fatalf("duplicate register status = %d: %s", rec.Code, rec.Body.String())
	}
	if len(store.sessions) != 0 {
		t.Errorf("created sessions = %d, want 0", len(store.sessions))
	}
}

func TestHandleLoginSuccessAndWrongPassword(t *testing.T) {
	store := newAuthHandlerTestStore()
	password := "correct horse battery staple"
	phc, err := auth.Hash(password)
	if err != nil {
		t.Fatalf("Hash: %v", err)
	}
	user := types.User{ID: "login-user", AccountID: "acct-1", Email: "login@example.com", DisplayName: "Login User", Status: "active", CreatedAt: time.Now().UTC()}
	store.users[user.ID] = user
	store.userByEmail[user.Email] = user
	store.phcHash[user.ID] = phc
	h, _ := newAuthHandlerForTest(store, testAuthConfig())

	rec := doRequest(h, http.MethodPost, "/api/v1/auth/login", map[string]string{"email": user.Email, "password": password}, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("login status = %d: %s", rec.Code, rec.Body.String())
	}
	if got := decodeJSON[sessionResponse](t, rec); got.User.ID != user.ID {
		t.Errorf("login user ID = %q, want %q", got.User.ID, user.ID)
	}
	if len(store.sessions) != 1 {
		t.Fatalf("created sessions = %d, want 1", len(store.sessions))
	}

	rec = doRequest(h, http.MethodPost, "/api/v1/auth/login", map[string]string{"email": user.Email, "password": "wrong password"}, nil)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("wrong-password status = %d: %s", rec.Code, rec.Body.String())
	}
	if len(store.sessions) != 1 {
		t.Errorf("wrong password created a session")
	}
}

func TestChangePasswordRevocationFailureLeavesPasswordUnchanged(t *testing.T) {
	store := newAuthHandlerTestStore()
	oldHash, err := auth.Hash("correct horse battery staple")
	if err != nil {
		t.Fatal(err)
	}
	store.phcHash["test-user"] = oldHash
	store.deleteUserSessionsErr = errors.New("store unavailable")
	h, _ := newAuthHandlerForTest(store, testAuthConfig())

	rec := doRequest(h, http.MethodPost, "/api/v1/auth/change-password", map[string]string{
		"current_password": "correct horse battery staple",
		"new_password":     "newSecurePassword123!",
	}, nil)
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", rec.Code)
	}
	if store.phcHash["test-user"] != oldHash {
		t.Error("password changed despite session revocation failure")
	}
}

func TestHandleLoginCookieAttributes(t *testing.T) {
	store := newAuthHandlerTestStore()
	password := "correct horse battery staple"
	phc, err := auth.Hash(password)
	if err != nil {
		t.Fatalf("Hash: %v", err)
	}
	user := types.User{ID: "cookie-user", Email: "cookie@example.com", Status: "active", CreatedAt: time.Now().UTC()}
	store.userByEmail[user.Email] = user
	store.phcHash[user.ID] = phc

	cfg := testAuthConfig()
	cfg.CookieSecure = true
	cfg.CookieDomain = "example.com"
	h, _ := newAuthHandlerForTest(store, cfg)
	rec := doRequest(h, http.MethodPost, "/api/v1/auth/login", map[string]any{"email": user.Email, "password": password, "remember": true}, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("login status = %d: %s", rec.Code, rec.Body.String())
	}

	for _, name := range []string{"dd_session", "dd_csrf"} {
		cookie := cookieNamed(t, rec, name)
		if cookie.Path != "/" || cookie.Domain != "example.com" || !cookie.Secure || cookie.SameSite != http.SameSiteLaxMode {
			t.Errorf("%s attributes = %+v", name, cookie)
		}
		if cookie.MaxAge != int(cfg.SessionCfg.RememberTTL.Seconds()) {
			t.Errorf("%s MaxAge = %d, want %d", name, cookie.MaxAge, int(cfg.SessionCfg.RememberTTL.Seconds()))
		}
	}
}

func TestHandleLogoutInvalidatesSession(t *testing.T) {
	store := newAuthHandlerTestStore()
	h, _ := newAuthHandlerForTest(store, testAuthConfig())
	token := "session-token"
	id := auth.HashToken(token)

	rec := doRequest(h, http.MethodPost, "/api/v1/auth/logout", nil, map[string]string{
		"Authorization": "Bearer test-api-key",
		"Cookie":        "dd_session=" + token,
	})
	if rec.Code != http.StatusNoContent {
		t.Fatalf("logout status = %d: %s", rec.Code, rec.Body.String())
	}
	if len(store.deleted) != 1 || store.deleted[0] != id {
		t.Errorf("deleted sessions = %v, want [%s]", store.deleted, id)
	}
	if cookieNamed(t, rec, "dd_session").MaxAge >= 0 || cookieNamed(t, rec, "dd_csrf").MaxAge >= 0 {
		t.Error("logout must expire both session cookies")
	}
}

func TestHandleSessionResponse(t *testing.T) {
	store := newAuthHandlerTestStore()
	h, meals := newAuthHandlerForTest(store, testAuthConfig())
	verifiedAt := time.Now().UTC()
	meals.user.EmailVerifiedAt = &verifiedAt

	rec := doRequest(h, http.MethodGet, "/api/v1/auth/session", nil, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("session status = %d: %s", rec.Code, rec.Body.String())
	}
	got := decodeJSON[sessionResponse](t, rec)
	if got.User.ID != meals.user.ID || got.User.Email != meals.user.Email || !got.User.EmailVerified {
		t.Errorf("session user = %+v", got.User)
	}
}
