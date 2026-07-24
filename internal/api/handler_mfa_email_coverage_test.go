package api

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gsaraiva2109/dietdaemon/core/types"
	"github.com/gsaraiva2109/dietdaemon/internal/auth"
)

// ---------------------------------------------------------------------------
// handleMFAEmailSend: remaining branches
// ---------------------------------------------------------------------------

func TestHandleMFAEmailSendMissingChallengeToken(t *testing.T) {
	h := buildMFAEmailHandler(newMFAEmailTestStore(), &fakeMailer{})

	rec := doRequest(h, http.MethodPost, "/api/v1/auth/mfa/email/send", map[string]string{"challenge_token": ""}, nil)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("send status = %d, want 400: %s", rec.Code, rec.Body.String())
	}
}

func TestHandleMFAEmailSendInvalidJSON(t *testing.T) {
	h := buildMFAEmailHandler(newMFAEmailTestStore(), &fakeMailer{})

	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/mfa/email/send", strings.NewReader("not json"))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("send status = %d, want 400", rec.Code)
	}
}

func TestHandleMFAEmailSendExpiredChallenge(t *testing.T) {
	authStore := newMFAEmailTestStore()
	h := buildMFAEmailHandler(authStore, &fakeMailer{})
	token := "expired-challenge"
	authStore.addChallenge(token, time.Now().UTC().Add(-time.Minute))

	rec := doRequest(h, http.MethodPost, "/api/v1/auth/mfa/email/send", map[string]string{"challenge_token": token}, nil)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("send status = %d, want 401: %s", rec.Code, rec.Body.String())
	}
	if _, ok := authStore.challenges[auth.HashToken(token)]; ok {
		t.Error("expected expired challenge to be deleted")
	}
}

func TestHandleMFAEmailSendGetUserError(t *testing.T) {
	authStore := newMFAEmailTestStore()
	store := newFakeMealStore()
	store.getUserErr = errors.New("db down")
	fm := &fakeMailer{}
	h := New(store, &fakeMealLogger{}, time.UTC, nil, nil,
		WithAuth(authStore, AuthRepos{Sessions: authStore, LoginAttempts: authStore, TOTP: authStore, MFAChallenges: authStore, RecoveryCodes: authStore}, nil, "DietDaemon", AuthConfig{
			SessionCfg:       auth.SessionConfig{IdleTTL: time.Hour, AbsoluteTTL: 24 * time.Hour, RememberTTL: 72 * time.Hour},
			LockoutCfg:       auth.DefaultLockoutConfig(),
			RegistrationMode: types.RegistrationOpen,
		}),
		WithMailer(fm, "smtp"),
	)
	token := "valid-challenge"
	authStore.addChallenge(token, time.Now().UTC().Add(time.Minute))

	rec := doRequest(h, http.MethodPost, "/api/v1/auth/mfa/email/send", map[string]string{"challenge_token": token}, nil)
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("send status = %d, want 500: %s", rec.Code, rec.Body.String())
	}
}

func TestHandleMFAEmailSendUnverifiedEmail(t *testing.T) {
	authStore := newMFAEmailTestStore()
	store := newFakeMealStore()
	store.user = types.User{ID: "test-user", AccountID: "acct-1", Email: "test@example.com", Status: "active", CreatedAt: time.Now().UTC()} // no EmailVerifiedAt
	fm := &fakeMailer{}
	h := New(store, &fakeMealLogger{}, time.UTC, nil, nil,
		WithAuth(authStore, AuthRepos{Sessions: authStore, LoginAttempts: authStore, TOTP: authStore, MFAChallenges: authStore, RecoveryCodes: authStore}, nil, "DietDaemon", AuthConfig{
			SessionCfg:       auth.SessionConfig{IdleTTL: time.Hour, AbsoluteTTL: 24 * time.Hour, RememberTTL: 72 * time.Hour},
			LockoutCfg:       auth.DefaultLockoutConfig(),
			RegistrationMode: types.RegistrationOpen,
		}),
		WithMailer(fm, "smtp"),
	)
	token := "valid-challenge"
	authStore.addChallenge(token, time.Now().UTC().Add(time.Minute))

	rec := doRequest(h, http.MethodPost, "/api/v1/auth/mfa/email/send", map[string]string{"challenge_token": token}, nil)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("send status = %d, want 400: %s", rec.Code, rec.Body.String())
	}
}

func TestHandleMFAEmailSendUpsertCodeError(t *testing.T) {
	authStore := newMFAEmailTestStore()
	authStore.upsertMFAEmailCodeErr = errors.New("store unavailable")
	h := buildMFAEmailHandler(authStore, &fakeMailer{})
	token := "valid-challenge"
	authStore.addChallenge(token, time.Now().UTC().Add(time.Minute))

	rec := doRequest(h, http.MethodPost, "/api/v1/auth/mfa/email/send", map[string]string{"challenge_token": token}, nil)
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("send status = %d, want 500: %s", rec.Code, rec.Body.String())
	}
}

// ---------------------------------------------------------------------------
// handleMFAEmailVerify: remaining branches
// ---------------------------------------------------------------------------

func TestHandleMFAEmailVerifyInvalidJSON(t *testing.T) {
	h := buildMFAEmailHandler(newMFAEmailTestStore(), &fakeMailer{})

	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/mfa/email/verify", strings.NewReader("not json"))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("verify status = %d, want 400", rec.Code)
	}
}

func TestHandleMFAEmailVerifyMissingFields(t *testing.T) {
	h := buildMFAEmailHandler(newMFAEmailTestStore(), &fakeMailer{})

	cases := []map[string]string{
		{"challenge_token": "", "code": "123456"},
		{"challenge_token": "sometoken", "code": ""},
	}
	for _, body := range cases {
		rec := doRequest(h, http.MethodPost, "/api/v1/auth/mfa/email/verify", body, nil)
		if rec.Code != http.StatusBadRequest {
			t.Errorf("verify with %+v status = %d, want 400", body, rec.Code)
		}
	}
}

func TestHandleMFAEmailVerifyBadCodeFormat(t *testing.T) {
	h := buildMFAEmailHandler(newMFAEmailTestStore(), &fakeMailer{})

	rec := doRequest(h, http.MethodPost, "/api/v1/auth/mfa/email/verify", map[string]string{
		"challenge_token": "sometoken", "code": "12ab56",
	}, nil)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("verify status = %d, want 400: %s", rec.Code, rec.Body.String())
	}
}

func TestHandleMFAEmailVerifyUnknownChallenge(t *testing.T) {
	h := buildMFAEmailHandler(newMFAEmailTestStore(), &fakeMailer{})

	rec := doRequest(h, http.MethodPost, "/api/v1/auth/mfa/email/verify", map[string]string{
		"challenge_token": "bogus", "code": "123456",
	}, nil)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("verify status = %d, want 401: %s", rec.Code, rec.Body.String())
	}
}

func TestHandleMFAEmailVerifyNoCodeRequested(t *testing.T) {
	authStore := newMFAEmailTestStore()
	h := buildMFAEmailHandler(authStore, &fakeMailer{})
	token := "valid-challenge"
	authStore.addChallenge(token, time.Now().UTC().Add(time.Minute))
	// No authStore.codes["test-user"] entry.

	rec := doRequest(h, http.MethodPost, "/api/v1/auth/mfa/email/verify", map[string]string{
		"challenge_token": token, "code": "123456",
	}, nil)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("verify status = %d, want 401: %s", rec.Code, rec.Body.String())
	}
}

func TestHandleMFAEmailVerifyCodeExpired(t *testing.T) {
	authStore := newMFAEmailTestStore()
	h := buildMFAEmailHandler(authStore, &fakeMailer{})
	token := "valid-challenge"
	authStore.addChallenge(token, time.Now().UTC().Add(time.Minute))
	authStore.codes["test-user"] = mfaEmailCode{
		hash: auth.HashToken("123456"), expiresAt: time.Now().UTC().Add(-time.Minute).Format(time.RFC3339),
	}

	rec := doRequest(h, http.MethodPost, "/api/v1/auth/mfa/email/verify", map[string]string{
		"challenge_token": token, "code": "123456",
	}, nil)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("verify status = %d, want 401: %s", rec.Code, rec.Body.String())
	}
	if _, ok := authStore.codes["test-user"]; ok {
		t.Error("expected expired code to be cleaned up")
	}
	if _, ok := authStore.challenges[auth.HashToken(token)]; ok {
		t.Error("expected challenge to be cleaned up when code has expired")
	}
}

func TestHandleMFAEmailVerifyTooManyAttempts(t *testing.T) {
	authStore := newMFAEmailTestStore()
	h := buildMFAEmailHandler(authStore, &fakeMailer{})
	token := "valid-challenge"
	authStore.addChallenge(token, time.Now().UTC().Add(time.Minute))
	authStore.codes["test-user"] = mfaEmailCode{
		hash: auth.HashToken("123456"), expiresAt: time.Now().UTC().Add(time.Minute).Format(time.RFC3339), attempts: 5,
	}

	rec := doRequest(h, http.MethodPost, "/api/v1/auth/mfa/email/verify", map[string]string{
		"challenge_token": token, "code": "123456",
	}, nil)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("verify status = %d, want 401: %s", rec.Code, rec.Body.String())
	}
}

func TestHandleMFAEmailVerifyCreateSessionError(t *testing.T) {
	authStore := newMFAEmailTestStore()
	authStore.createSessionErr = errors.New("store unavailable")
	h := buildMFAEmailHandler(authStore, &fakeMailer{})
	token := "valid-challenge"
	authStore.addChallenge(token, time.Now().UTC().Add(time.Minute))
	authStore.codes["test-user"] = mfaEmailCode{
		hash: auth.HashToken("123456"), expiresAt: time.Now().UTC().Add(time.Minute).Format(time.RFC3339),
	}

	rec := doRequest(h, http.MethodPost, "/api/v1/auth/mfa/email/verify", map[string]string{
		"challenge_token": token, "code": "123456",
	}, nil)
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("verify status = %d, want 500: %s", rec.Code, rec.Body.String())
	}
}

func TestHandleMFAEmailVerifyGetUserError(t *testing.T) {
	authStore := newMFAEmailTestStore()
	store := newFakeMealStore()
	store.getUserErr = errors.New("db down")
	fm := &fakeMailer{}
	h := New(store, &fakeMealLogger{}, time.UTC, nil, nil,
		WithAuth(authStore, AuthRepos{Sessions: authStore, LoginAttempts: authStore, TOTP: authStore, MFAChallenges: authStore, RecoveryCodes: authStore}, nil, "DietDaemon", AuthConfig{
			SessionCfg:       auth.SessionConfig{IdleTTL: time.Hour, AbsoluteTTL: 24 * time.Hour, RememberTTL: 72 * time.Hour},
			LockoutCfg:       auth.DefaultLockoutConfig(),
			RegistrationMode: types.RegistrationOpen,
		}),
		WithMailer(fm, "smtp"),
	)
	token := "valid-challenge"
	authStore.addChallenge(token, time.Now().UTC().Add(time.Minute))
	authStore.codes["test-user"] = mfaEmailCode{
		hash: auth.HashToken("123456"), expiresAt: time.Now().UTC().Add(time.Minute).Format(time.RFC3339),
	}

	rec := doRequest(h, http.MethodPost, "/api/v1/auth/mfa/email/verify", map[string]string{
		"challenge_token": token, "code": "123456",
	}, nil)
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("verify status = %d, want 500: %s", rec.Code, rec.Body.String())
	}
	// The session was still created before GetUser failed.
	if len(authStore.sessions) != 1 {
		t.Errorf("expected session to be created despite GetUser failure, got %d", len(authStore.sessions))
	}
}
