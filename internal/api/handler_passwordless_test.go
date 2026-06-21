package api

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/gsaraiva2109/dietdaemon/core/types"
	"github.com/gsaraiva2109/dietdaemon/internal/auth"
	"github.com/gsaraiva2109/dietdaemon/internal/mailer"
)

// magicTestAuthStore extends emailTestAuthStore with magic code storage.
type magicTestAuthStore struct {
	*emailTestAuthStore
	magicCodes          map[string]magicCode // userID -> code
	deleteEmailTokensCb func(userID, purpose string)
}

type magicCode struct {
	codeHash  string
	expiresAt string
	attempts  int
}

func newMagicTestAuthStore() *magicTestAuthStore {
	s := &magicTestAuthStore{
		emailTestAuthStore: newEmailTestAuthStore(),
		magicCodes:         make(map[string]magicCode),
	}
	// Pre-register a test user with email and password.
	s.emailTestAuthStore.fakeAuthStore.userByEmail["test@example.com"] = types.User{
		ID:        "test-user",
		Email:     "test@example.com",
		AccountID: "acct-1",
		Status:    "active",
		CreatedAt: time.Now().UTC(),
	}
	s.emailTestAuthStore.fakeAuthStore.users["test-user"] = types.User{
		ID:        "test-user",
		Email:     "test@example.com",
		AccountID: "acct-1",
		Status:    "active",
		CreatedAt: time.Now().UTC(),
	}
	return s
}

func (s *magicTestAuthStore) UpsertMagicCode(_ context.Context, userID, codeHash, expiresAt string) error {
	s.magicCodes[userID] = magicCode{codeHash: codeHash, expiresAt: expiresAt, attempts: 0}
	return nil
}

func (s *magicTestAuthStore) GetMagicCode(_ context.Context, userID string) (string, string, int, error) {
	mc, ok := s.magicCodes[userID]
	if !ok {
		return "", "", 0, types.ErrNotFound
	}
	return mc.codeHash, mc.expiresAt, mc.attempts, nil
}

func (s *magicTestAuthStore) IncrementMagicCodeAttempts(_ context.Context, userID string) error {
	mc := s.magicCodes[userID]
	mc.attempts++
	s.magicCodes[userID] = mc
	return nil
}

func (s *magicTestAuthStore) DeleteMagicCode(_ context.Context, userID string) error {
	delete(s.magicCodes, userID)
	return nil
}

func (s *magicTestAuthStore) DeleteEmailTokensByUserAndPurpose(_ context.Context, userID, purpose string) error {
	if s.deleteEmailTokensCb != nil {
		s.deleteEmailTokensCb(userID, purpose)
	}
	return nil
}

func (s *magicTestAuthStore) HasConfirmedTOTP(_ context.Context, userID string) (bool, error) {
	return false, nil
}

func buildMagicHandler(authStore *magicTestAuthStore, m mailer.Mailer) *Handler {
	store := newFakeMealStore()
	store.user = types.User{ID: "test-user", Email: "test@example.com", AccountID: "acct-1", Status: "active", CreatedAt: time.Now().UTC()}
	return New(store, authStore, &fakeMealLogger{}, time.UTC, authStore, authStore, authStore, authStore, authStore, nil, "DietDaemon", nil, m, "none", "http://localhost:8080", AuthConfig{
		SessionCfg: auth.SessionConfig{
			IdleTTL:     1 * time.Hour,
			AbsoluteTTL: 24 * time.Hour,
			RememberTTL: 72 * time.Hour,
		},
		LockoutCfg:       auth.DefaultLockoutConfig(),
		RegistrationMode: types.RegistrationOpen,
		CookieSecure:     false,
	}, nil)
}

// --- Magic request tests ---

func TestMagicRequestGenericResponseUnknownEmail(t *testing.T) {
	authStore := newMagicTestAuthStore()
	fm := &fakeMailer{}
	h := buildMagicHandler(authStore, fm)

	rec := doRequest(h, "POST", "/api/v1/auth/magic/request", map[string]string{"email": "noone@example.com"}, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 for unknown email, got %d: %s", rec.Code, rec.Body.String())
	}

	// No email should have been sent.
	if len(fm.sent) != 0 {
		t.Error("no email should be sent for unknown address")
	}
}

func TestMagicRequestGenericResponseKnownEmail(t *testing.T) {
	authStore := newMagicTestAuthStore()
	fm := &fakeMailer{}
	h := buildMagicHandler(authStore, fm)

	rec := doRequest(h, "POST", "/api/v1/auth/magic/request", map[string]string{"email": "test@example.com"}, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	// Email should have been sent.
	if len(fm.sent) != 1 {
		t.Fatalf("expected 1 email sent, got %d", len(fm.sent))
	}

	// A magic code should have been upserted.
	if _, _, _, err := authStore.GetMagicCode(t.Context(), "test-user"); err != nil {
		t.Error("magic code should exist after request")
	}
}

func TestMagicRequestOIDCOnlyNoOp(t *testing.T) {
	authStore := newMagicTestAuthStore()
	fm := &fakeMailer{}
	h := buildMagicHandler(authStore, fm)
	h.registrationMode = types.RegistrationOIDCOnly

	rec := doRequest(h, "POST", "/api/v1/auth/magic/request", map[string]string{"email": "test@example.com"}, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 in oidc-only mode, got %d", rec.Code)
	}

	if len(fm.sent) != 0 {
		t.Error("no email should be sent in oidc-only mode")
	}
}

func TestMagicRequestEmptyEmailGeneric(t *testing.T) {
	authStore := newMagicTestAuthStore()
	fm := &fakeMailer{}
	h := buildMagicHandler(authStore, fm)

	rec := doRequest(h, "POST", "/api/v1/auth/magic/request", map[string]string{"email": ""}, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 for empty email, got %d", rec.Code)
	}

	if len(fm.sent) != 0 {
		t.Error("no email should be sent for empty email")
	}
}

// --- Magic verify by code tests ---

func TestMagicVerifyCodeSuccess(t *testing.T) {
	authStore := newMagicTestAuthStore()
	fm := &fakeMailer{}
	h := buildMagicHandler(authStore, fm)

	code := "123456"
	codeHash := auth.HashToken(code)
	_ = authStore.UpsertMagicCode(t.Context(), "test-user", codeHash, time.Now().UTC().Add(magicTTL).Format(time.RFC3339))

	rec := doRequest(h, "POST", "/api/v1/auth/magic/verify", map[string]string{
		"email": "test@example.com",
		"code":  code,
	}, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	// Code should be consumed.
	if _, _, _, err := authStore.GetMagicCode(t.Context(), "test-user"); err == nil {
		t.Error("magic code should be consumed after successful verify")
	}

	// Session cookie should be set.
	cookies := rec.Result().Cookies()
	found := false
	for _, c := range cookies {
		if c.Name == "dd_session" && c.Value != "" {
			found = true
		}
	}
	if !found {
		t.Error("dd_session cookie should be set")
	}
}

func TestMagicVerifyCodeWrongCode(t *testing.T) {
	authStore := newMagicTestAuthStore()
	fm := &fakeMailer{}
	h := buildMagicHandler(authStore, fm)

	code := "123456"
	codeHash := auth.HashToken(code)
	_ = authStore.UpsertMagicCode(t.Context(), "test-user", codeHash, time.Now().UTC().Add(magicTTL).Format(time.RFC3339))

	rec := doRequest(h, "POST", "/api/v1/auth/magic/verify", map[string]string{
		"email": "test@example.com",
		"code":  "999999",
	}, nil)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 for wrong code, got %d: %s", rec.Code, rec.Body.String())
	}

	// Code should still exist (not deleted on wrong attempt).
	_, _, attempts, err := authStore.GetMagicCode(t.Context(), "test-user")
	if err != nil {
		t.Fatalf("code should still exist after wrong attempt: %v", err)
	}
	if attempts != 1 {
		t.Errorf("attempts should be 1, got %d", attempts)
	}
}

func TestMagicVerifyCodeExpired(t *testing.T) {
	authStore := newMagicTestAuthStore()
	fm := &fakeMailer{}
	h := buildMagicHandler(authStore, fm)

	code := "123456"
	codeHash := auth.HashToken(code)
	_ = authStore.UpsertMagicCode(t.Context(), "test-user", codeHash, time.Now().UTC().Add(-1*time.Minute).Format(time.RFC3339))

	rec := doRequest(h, "POST", "/api/v1/auth/magic/verify", map[string]string{
		"email": "test@example.com",
		"code":  code,
	}, nil)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 for expired code, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestMagicVerifyCodeAttemptCap(t *testing.T) {
	authStore := newMagicTestAuthStore()
	fm := &fakeMailer{}
	h := buildMagicHandler(authStore, fm)

	code := "123456"
	codeHash := auth.HashToken(code)
	_ = authStore.UpsertMagicCode(t.Context(), "test-user", codeHash, time.Now().UTC().Add(magicTTL).Format(time.RFC3339))
	// Pre-set attempts to 5 (cap).
	authStore.magicCodes["test-user"] = magicCode{codeHash: codeHash, expiresAt: time.Now().UTC().Add(magicTTL).Format(time.RFC3339), attempts: 5}

	rec := doRequest(h, "POST", "/api/v1/auth/magic/verify", map[string]string{
		"email": "test@example.com",
		"code":  code,
	}, nil)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 at attempt cap, got %d: %s", rec.Code, rec.Body.String())
	}

	// Code should be deleted when cap hit.
	if _, _, _, err := authStore.GetMagicCode(t.Context(), "test-user"); err == nil {
		t.Error("magic code should be deleted when attempt cap hit")
	}
}

func TestMagicVerifyCodeReuse(t *testing.T) {
	authStore := newMagicTestAuthStore()
	fm := &fakeMailer{}
	h := buildMagicHandler(authStore, fm)

	code := "123456"
	codeHash := auth.HashToken(code)
	_ = authStore.UpsertMagicCode(t.Context(), "test-user", codeHash, time.Now().UTC().Add(magicTTL).Format(time.RFC3339))

	// First use succeeds.
	rec := doRequest(h, "POST", "/api/v1/auth/magic/verify", map[string]string{
		"email": "test@example.com",
		"code":  code,
	}, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("first use expected 200, got %d", rec.Code)
	}

	// Second use — code consumed, should 401.
	rec = doRequest(h, "POST", "/api/v1/auth/magic/verify", map[string]string{
		"email": "test@example.com",
		"code":  code,
	}, nil)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("reuse expected 401, got %d: %s", rec.Code, rec.Body.String())
	}
}

// --- Magic verify by token tests ---

func TestMagicVerifyTokenSuccess(t *testing.T) {
	authStore := newMagicTestAuthStore()
	fm := &fakeMailer{}
	h := buildMagicHandler(authStore, fm)

	tok := auth.NewToken()
	hashed := auth.HashToken(tok)
	_ = authStore.CreateEmailToken(t.Context(), hashed, "test-user", "magic_link", time.Now().UTC().Add(magicTTL).Format(time.RFC3339))

	rec := doRequest(h, "POST", "/api/v1/auth/magic/verify", map[string]string{"token": tok}, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	// Token should be consumed (single-use).
	if _, err := authStore.ConsumeEmailToken(t.Context(), hashed, "magic_link"); err == nil {
		t.Error("link token should have been consumed")
	}
}

func TestMagicVerifyTokenReused(t *testing.T) {
	authStore := newMagicTestAuthStore()
	fm := &fakeMailer{}
	h := buildMagicHandler(authStore, fm)

	tok := auth.NewToken()
	hashed := auth.HashToken(tok)
	_ = authStore.CreateEmailToken(t.Context(), hashed, "test-user", "magic_link", time.Now().UTC().Add(magicTTL).Format(time.RFC3339))

	// First use succeeds.
	rec := doRequest(h, "POST", "/api/v1/auth/magic/verify", map[string]string{"token": tok}, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("first use expected 200, got %d", rec.Code)
	}

	// Second use — 401.
	rec = doRequest(h, "POST", "/api/v1/auth/magic/verify", map[string]string{"token": tok}, nil)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("reuse expected 401, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestMagicVerifyTokenInvalid(t *testing.T) {
	authStore := newMagicTestAuthStore()
	fm := &fakeMailer{}
	h := buildMagicHandler(authStore, fm)

	rec := doRequest(h, "POST", "/api/v1/auth/magic/verify", map[string]string{"token": "bogus"}, nil)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 for invalid token, got %d", rec.Code)
	}
}

// --- Sibling credential cleanup tests ---

func TestMagicVerifyCodeCleansUpSiblingToken(t *testing.T) {
	authStore := newMagicTestAuthStore()
	fm := &fakeMailer{}
	h := buildMagicHandler(authStore, fm)

	code := "123456"
	codeHash := auth.HashToken(code)
	_ = authStore.UpsertMagicCode(t.Context(), "test-user", codeHash, time.Now().UTC().Add(magicTTL).Format(time.RFC3339))

	// Also create a sibling link token.
	tok := auth.NewToken()
	hashed := auth.HashToken(tok)
	_ = authStore.CreateEmailToken(t.Context(), hashed, "test-user", "magic_link", time.Now().UTC().Add(magicTTL).Format(time.RFC3339))

	deleteCalled := false
	authStore.deleteEmailTokensCb = func(userID, purpose string) {
		if userID == "test-user" && purpose == "magic_link" {
			deleteCalled = true
		}
	}

	rec := doRequest(h, "POST", "/api/v1/auth/magic/verify", map[string]string{
		"email": "test@example.com",
		"code":  code,
	}, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	if !deleteCalled {
		t.Error("sibling magic_link tokens should be deleted on code success")
	}
}

func TestMagicVerifyTokenCleansUpSiblingCode(t *testing.T) {
	authStore := newMagicTestAuthStore()
	fm := &fakeMailer{}
	h := buildMagicHandler(authStore, fm)

	// Create a magic code.
	codeHash := auth.HashToken("123456")
	_ = authStore.UpsertMagicCode(t.Context(), "test-user", codeHash, time.Now().UTC().Add(magicTTL).Format(time.RFC3339))

	// Create a link token.
	tok := auth.NewToken()
	hashed := auth.HashToken(tok)
	_ = authStore.CreateEmailToken(t.Context(), hashed, "test-user", "magic_link", time.Now().UTC().Add(magicTTL).Format(time.RFC3339))

	rec := doRequest(h, "POST", "/api/v1/auth/magic/verify", map[string]string{"token": tok}, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	// Sibling code should be deleted.
	if _, _, _, err := authStore.GetMagicCode(t.Context(), "test-user"); err == nil {
		t.Error("sibling magic code should be deleted on token success")
	}
}

// Phase 6 stubs — not exercised by existing tests.

func (s *magicTestAuthStore) GetOrCreateWebAuthnHandle(_ context.Context, _ string) (string, error) {
	return "", nil
}
func (s *magicTestAuthStore) GetUserByWebAuthnHandle(_ context.Context, _ string) (types.User, error) {
	return types.User{}, types.ErrNotFound
}
func (s *magicTestAuthStore) CreateWebAuthnCredential(_ context.Context, _, _, _, _ string, _ int, _ string) error {
	return nil
}
func (s *magicTestAuthStore) ListWebAuthnCredentials(_ context.Context, _ string) ([]types.Passkey, error) {
	return nil, nil
}
func (s *magicTestAuthStore) GetWebAuthnCredentialsRaw(_ context.Context, _ string) ([]types.WebAuthnCredential, error) {
	return nil, nil
}
func (s *magicTestAuthStore) UpdateWebAuthnCredentialOnAuth(_ context.Context, _, _ string, _ int, _ string) error {
	return nil
}
func (s *magicTestAuthStore) RenameWebAuthnCredential(_ context.Context, _, _, _ string) error {
	return nil
}
func (s *magicTestAuthStore) DeleteWebAuthnCredential(_ context.Context, _, _ string) error {
	return nil
}
func (s *magicTestAuthStore) CreateWebAuthnSession(_ context.Context, _, _, _, _ string) error {
	return nil
}
func (s *magicTestAuthStore) ConsumeWebAuthnSession(_ context.Context, _ string) (string, string, error) {
	return "", "", nil
}
func (s *magicTestAuthStore) UpsertMFAEmailCode(_ context.Context, _, _, _ string) error { return nil }
func (s *magicTestAuthStore) GetMFAEmailCode(_ context.Context, _ string) (string, string, int, error) {
	return "", "", 0, nil
}
func (s *magicTestAuthStore) IncrementMFAEmailCodeAttempts(_ context.Context, _ string) error {
	return nil
}
func (s *magicTestAuthStore) DeleteMFAEmailCode(_ context.Context, _ string) error { return nil }
