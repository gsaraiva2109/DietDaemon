package api

import (
	"context"
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/gsaraiva2109/dietdaemon/core/types"
	"github.com/gsaraiva2109/dietdaemon/internal/auth"
	"github.com/gsaraiva2109/dietdaemon/internal/mailer"
)

// fakeMailer records sent emails for inspection.
type fakeMailer struct {
	sent    []sentEmail
	sendErr error
}

type sentEmail struct {
	to      string
	subject string
}

func (m *fakeMailer) Send(_ context.Context, to string, msg mailer.Message) error {
	m.sent = append(m.sent, sentEmail{to: to, subject: msg.Subject})
	return m.sendErr
}

// emailTestAuthStore wraps fakeAuthStore with email token storage.
type emailTestAuthStore struct {
	*fakeAuthStore
	emailTokens           map[string]emailToken
	emailVerified         map[string]bool
	userEmails            map[string]string
	markVerifiedFail      bool
	updateEmailFail       bool
	deleteUserSessionsCb  func(userID string)
	createEmailTokenErr   error
	deleteUserSessionsErr error
}

type emailToken struct {
	userID    string
	purpose   string
	expiresAt string
}

func newEmailTestAuthStore() *emailTestAuthStore {
	return &emailTestAuthStore{
		fakeAuthStore: newFakeAuthStore(),
		emailTokens:   make(map[string]emailToken),
		emailVerified: make(map[string]bool),
		userEmails:    make(map[string]string),
	}
}

func (s *emailTestAuthStore) CreateEmailToken(_ context.Context, id, userID, purpose, expiresAt string) error {
	if s.createEmailTokenErr != nil {
		return s.createEmailTokenErr
	}
	s.emailTokens[id] = emailToken{userID: userID, purpose: purpose, expiresAt: expiresAt}
	return nil
}

func (s *emailTestAuthStore) ConsumeEmailToken(_ context.Context, id, purpose string) (string, error) {
	tok, ok := s.emailTokens[id]
	if !ok {
		return "", types.ErrNotFound
	}
	delete(s.emailTokens, id)

	if tok.purpose != purpose {
		return "", types.ErrNotFound
	}

	exp, err := time.Parse(time.RFC3339, tok.expiresAt)
	if err != nil || time.Now().UTC().After(exp) {
		return "", types.ErrNotFound
	}

	return tok.userID, nil
}

func (s *emailTestAuthStore) MarkEmailVerified(_ context.Context, userID string) error {
	if s.markVerifiedFail {
		return types.ErrNotFound
	}
	s.emailVerified[userID] = true
	return nil
}

func (s *emailTestAuthStore) UpdateUserEmail(_ context.Context, userID, email string) error {
	if s.updateEmailFail {
		return types.ErrNotFound
	}
	s.userEmails[userID] = email
	return nil
}

func (s *emailTestAuthStore) DeleteUserSessions(_ context.Context, userID string) error {
	if s.deleteUserSessionsCb != nil {
		s.deleteUserSessionsCb(userID)
	}
	return s.deleteUserSessionsErr
}

// Magic codes.
func (s *emailTestAuthStore) UpsertMagicCode(_ context.Context, userID, codeHash, expiresAt string) error {
	return nil
}
func (s *emailTestAuthStore) GetMagicCode(_ context.Context, userID string) (string, string, int, error) {
	return "", "", 0, types.ErrNotFound
}
func (s *emailTestAuthStore) IncrementMagicCodeAttempts(_ context.Context, userID string) error {
	return nil
}
func (s *emailTestAuthStore) DeleteMagicCode(_ context.Context, userID string) error { return nil }
func (s *emailTestAuthStore) DeleteEmailTokensByUserAndPurpose(_ context.Context, userID, purpose string) error {
	return nil
}

func buildEmailHandler(authStore *emailTestAuthStore, m mailer.Mailer) *Handler {
	store := newFakeMealStore()
	store.user = types.User{ID: "test-user", Email: "test@example.com", Status: "active", CreatedAt: time.Now().UTC()}
	return New(store, &fakeMealLogger{}, time.UTC, nil, nil,
		WithAuth(authStore, authStore, authStore, authStore, authStore, authStore, nil, "DietDaemon", AuthConfig{
			SessionCfg: auth.SessionConfig{
				IdleTTL:     1 * time.Hour,
				AbsoluteTTL: 24 * time.Hour,
				RememberTTL: 72 * time.Hour,
			},
			LockoutCfg:       auth.DefaultLockoutConfig(),
			RegistrationMode: types.RegistrationOpen,
			CookieSecure:     false,
		}),
		WithMailer(m, "none"),
		WithPublicBaseURL("http://localhost:8080"),
	)
}

func TestEmailVerifySuccess(t *testing.T) {
	authStore := newEmailTestAuthStore()
	fm := &fakeMailer{}
	h := buildEmailHandler(authStore, fm)

	tok := auth.NewToken()
	hashed := auth.HashToken(tok)
	_ = authStore.CreateEmailToken(t.Context(), hashed, "test-user", "verify", time.Now().UTC().Add(24*time.Hour).Format(time.RFC3339))

	rec := doRequest(h, "POST", "/api/v1/auth/email/verify", map[string]string{"token": tok}, nil)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d: %s", rec.Code, rec.Body.String())
	}

	// Token should be consumed (single-use).
	if _, err := authStore.ConsumeEmailToken(t.Context(), hashed, "verify"); err == nil {
		t.Error("token should have been consumed (single-use)")
	}

	// User should be verified.
	if !authStore.emailVerified["test-user"] {
		t.Error("user should be marked verified")
	}
}

func TestEmailVerifyInvalidToken(t *testing.T) {
	authStore := newEmailTestAuthStore()
	fm := &fakeMailer{}
	h := buildEmailHandler(authStore, fm)

	rec := doRequest(h, "POST", "/api/v1/auth/email/verify", map[string]string{"token": "bogus"}, nil)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestEmailVerifyPurposeMismatch(t *testing.T) {
	authStore := newEmailTestAuthStore()
	fm := &fakeMailer{}
	h := buildEmailHandler(authStore, fm)

	tok := auth.NewToken()
	hashed := auth.HashToken(tok)
	_ = authStore.CreateEmailToken(t.Context(), hashed, "test-user", "reset", time.Now().UTC().Add(1*time.Hour).Format(time.RFC3339))

	rec := doRequest(h, "POST", "/api/v1/auth/email/verify", map[string]string{"token": tok}, nil)
	if rec.Code == http.StatusNoContent {
		t.Error("reset token should not work for verify (purpose mismatch)")
	}
}

func TestEmailVerifyExpiredToken(t *testing.T) {
	authStore := newEmailTestAuthStore()
	fm := &fakeMailer{}
	h := buildEmailHandler(authStore, fm)

	tok := auth.NewToken()
	hashed := auth.HashToken(tok)
	_ = authStore.CreateEmailToken(t.Context(), hashed, "test-user", "verify", time.Now().UTC().Add(-1*time.Hour).Format(time.RFC3339))

	rec := doRequest(h, "POST", "/api/v1/auth/email/verify", map[string]string{"token": tok}, nil)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for expired token, got %d", rec.Code)
	}
}

func TestForgotPasswordGenericResponse(t *testing.T) {
	authStore := newEmailTestAuthStore()
	fm := &fakeMailer{}
	h := buildEmailHandler(authStore, fm)

	// Non-existent email — still 200 {ok:true}.
	rec := doRequest(h, "POST", "/api/v1/auth/password/forgot", map[string]string{"email": "noone@example.com"}, nil)
	if rec.Code != http.StatusOK {
		t.Errorf("forgot should return 200 for unknown email, got %d", rec.Code)
	}

	// Known email but no password (OIDC-only style) — still 200.
	rec = doRequest(h, "POST", "/api/v1/auth/password/forgot", map[string]string{"email": "test@example.com"}, nil)
	if rec.Code != http.StatusOK {
		t.Errorf("forgot should return 200 for user without password, got %d", rec.Code)
	}
}

func TestForgotPasswordLockoutFailuresAreGenericNoOps(t *testing.T) {
	for _, name := range []string{"locked", "store error"} {
		t.Run(name, func(t *testing.T) {
			authStore := newEmailTestAuthStore()
			authStore.userByEmail["test@example.com"] = types.User{ID: "test-user", Email: "test@example.com"}
			authStore.phcHash["test-user"] = "password hash"
			if name == "locked" {
				for range 3 {
					_ = authStore.RecordLoginAttempt(t.Context(), "forgot:test@example.com", false)
				}
			} else {
				authStore.recentFailedAttemptsErr = errors.New("store unavailable")
			}

			fm := &fakeMailer{}
			rec := doRequest(buildEmailHandler(authStore, fm), http.MethodPost, "/api/v1/auth/password/forgot", map[string]string{"email": "test@example.com"}, nil)
			if rec.Code != http.StatusOK {
				t.Fatalf("expected generic 200, got %d", rec.Code)
			}
			if len(authStore.emailTokens) != 0 || len(fm.sent) != 0 {
				t.Error("lockout failure must not issue a reset token or send email")
			}
		})
	}
}

func TestForgotPasswordTokenPersistenceFailureIsGeneric(t *testing.T) {
	authStore := newEmailTestAuthStore()
	authStore.userByEmail["test@example.com"] = types.User{ID: "test-user", Email: "test@example.com"}
	authStore.phcHash["test-user"] = "password hash"
	authStore.createEmailTokenErr = errors.New("store unavailable")
	fm := &fakeMailer{}

	rec := doRequest(buildEmailHandler(authStore, fm), http.MethodPost, "/api/v1/auth/password/forgot", map[string]string{"email": "test@example.com"}, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected generic 200, got %d", rec.Code)
	}
	if len(fm.sent) != 0 {
		t.Error("must not send when token persistence fails")
	}
}

func TestPasswordResetRevokesSessions(t *testing.T) {
	authStore := newEmailTestAuthStore()
	// Give the test user a password so forgot→reset works.
	authStore.phcHash["test-user"] = "$argon2id$v=19$m=65536,t=3,p=4$c29tZXNhbHQ$c29tZWhhc2g"

	sessionsDeleted := false
	authStore.deleteUserSessionsCb = func(userID string) {
		if userID == "test-user" {
			sessionsDeleted = true
		}
	}

	fm := &fakeMailer{}
	h := buildEmailHandler(authStore, fm)

	tok := auth.NewToken()
	hashed := auth.HashToken(tok)
	_ = authStore.CreateEmailToken(t.Context(), hashed, "test-user", "reset", time.Now().UTC().Add(1*time.Hour).Format(time.RFC3339))

	rec := doRequest(h, "POST", "/api/v1/auth/password/reset", map[string]string{
		"token":    tok,
		"password": "newSecurePassword123!",
	}, nil)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d: %s", rec.Code, rec.Body.String())
	}

	if !sessionsDeleted {
		t.Error("password reset should delete all user sessions")
	}
}

func TestPasswordResetRevocationFailureLeavesPasswordUnchanged(t *testing.T) {
	authStore := newEmailTestAuthStore()
	oldHash := "old password hash"
	authStore.phcHash["test-user"] = oldHash
	authStore.deleteUserSessionsErr = errors.New("store unavailable")
	tok := auth.NewToken()
	_ = authStore.CreateEmailToken(t.Context(), auth.HashToken(tok), "test-user", "reset", time.Now().UTC().Add(time.Hour).Format(time.RFC3339))

	rec := doRequest(buildEmailHandler(authStore, &fakeMailer{}), http.MethodPost, "/api/v1/auth/password/reset", map[string]string{
		"token": tok, "password": "newSecurePassword123!",
	}, nil)
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", rec.Code)
	}
	if authStore.phcHash["test-user"] != oldHash {
		t.Error("password changed despite session revocation failure")
	}
}

// Stubs — not exercised by existing tests.

func (s *emailTestAuthStore) GetOrCreateWebAuthnHandle(_ context.Context, _ string) (string, error) {
	return "", nil
}
func (s *emailTestAuthStore) GetUserByWebAuthnHandle(_ context.Context, _ string) (types.User, error) {
	return types.User{}, types.ErrNotFound
}
func (s *emailTestAuthStore) CreateWebAuthnCredential(_ context.Context, _, _, _, _ string, _ int, _ string) error {
	return nil
}
func (s *emailTestAuthStore) ListWebAuthnCredentials(_ context.Context, _ string) ([]types.Passkey, error) {
	return nil, nil
}
func (s *emailTestAuthStore) GetWebAuthnCredentialsRaw(_ context.Context, _ string) ([]types.WebAuthnCredential, error) {
	return nil, nil
}
func (s *emailTestAuthStore) UpdateWebAuthnCredentialOnAuth(_ context.Context, _, _ string, _ int, _ string) error {
	return nil
}
func (s *emailTestAuthStore) RenameWebAuthnCredential(_ context.Context, _, _, _ string) error {
	return nil
}
func (s *emailTestAuthStore) DeleteWebAuthnCredential(_ context.Context, _, _ string) error {
	return nil
}
func (s *emailTestAuthStore) CreateWebAuthnSession(_ context.Context, _, _, _, _ string) error {
	return nil
}
func (s *emailTestAuthStore) ConsumeWebAuthnSession(_ context.Context, _ string) (string, string, error) {
	return "", "", nil
}
func (s *emailTestAuthStore) UpsertMFAEmailCode(_ context.Context, _, _, _ string) error { return nil }
func (s *emailTestAuthStore) GetMFAEmailCode(_ context.Context, _ string) (string, string, int, error) {
	return "", "", 0, nil
}
func (s *emailTestAuthStore) IncrementMFAEmailCodeAttempts(_ context.Context, _ string) error {
	return nil
}
func (s *emailTestAuthStore) DeleteMFAEmailCode(_ context.Context, _ string) error { return nil }
