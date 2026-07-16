package api

import (
	"context"
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
