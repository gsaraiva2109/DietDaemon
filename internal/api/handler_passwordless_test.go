package api

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
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
	totpConfirmed       bool
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
	s.userByEmail["test@example.com"] = types.User{
		ID:        "test-user",
		Email:     "test@example.com",
		AccountID: "acct-1",
		Status:    "active",
		CreatedAt: time.Now().UTC(),
	}
	s.users["test-user"] = types.User{
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
	return s.totpConfirmed, nil
}

func buildMagicHandler(authStore *magicTestAuthStore, m mailer.Mailer) *Handler {
	user := types.User{ID: "test-user", Email: "test@example.com", AccountID: "acct-1", Status: "active", CreatedAt: time.Now().UTC()}
	return buildAuthTestHandler(authStore, user, m)
}

// upsertTestMagicCode hashes code and stores it for "test-user" with the
// given TTL, returning the hash for tests that need to inspect/override it.
func upsertTestMagicCode(t *testing.T, authStore *magicTestAuthStore, code string, ttl time.Duration) string {
	t.Helper()
	codeHash := auth.HashToken(code)
	_ = authStore.UpsertMagicCode(t.Context(), "test-user", codeHash, time.Now().UTC().Add(ttl).Format(time.RFC3339))
	return codeHash
}

// verifyMagicCode POSTs an email+code pair to the magic verify endpoint.
func verifyMagicCode(h *Handler, code string) *httptest.ResponseRecorder {
	return doRequest(h, "POST", "/api/v1/auth/magic/verify", map[string]string{
		"email": "test@example.com",
		"code":  code,
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

func TestMagicRequestLockoutFailuresAreGenericNoOps(t *testing.T) {
	for _, name := range []string{"locked", "store error"} {
		t.Run(name, func(t *testing.T) {
			authStore := newMagicTestAuthStore()
			if name == "locked" {
				for range 3 {
					_ = authStore.RecordLoginAttempt(t.Context(), "magic:test@example.com", false)
				}
			} else {
				authStore.recentFailedAttemptsErr = errors.New("store unavailable")
			}
			fm := &fakeMailer{}

			rec := doRequest(buildMagicHandler(authStore, fm), http.MethodPost, "/api/v1/auth/magic/request", map[string]string{"email": "test@example.com"}, nil)
			if rec.Code != http.StatusOK {
				t.Fatalf("expected generic 200, got %d", rec.Code)
			}
			if len(authStore.emailTokens) != 0 || len(authStore.magicCodes) != 0 || len(fm.sent) != 0 {
				t.Error("lockout failure must not issue credentials or send email")
			}
		})
	}
}

func TestMagicRequestDeliveryFailureDoesNotConsumeLockoutAttempt(t *testing.T) {
	authStore := newMagicTestAuthStore()
	fm := &fakeMailer{sendErr: errors.New("mail unavailable")}

	rec := doRequest(buildMagicHandler(authStore, fm), http.MethodPost, "/api/v1/auth/magic/request", map[string]string{"email": "test@example.com"}, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected generic 200, got %d", rec.Code)
	}
	if len(authStore.loginAttempts) != 0 {
		t.Errorf("delivery failure recorded lockout attempts: %#v", authStore.loginAttempts)
	}
	if len(authStore.auditEvents) != 1 || authStore.auditEvents[0].Event != "user.magic_request_failed" {
		t.Errorf("expected failed-delivery audit event, got %#v", authStore.auditEvents)
	}
}

// --- Magic verify by code tests ---

func TestMagicVerifyCodeSuccess(t *testing.T) {
	authStore := newMagicTestAuthStore()
	fm := &fakeMailer{}
	h := buildMagicHandler(authStore, fm)

	code := "123456"
	upsertTestMagicCode(t, authStore, code, magicTTL)

	rec := verifyMagicCode(h, code)
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
	upsertTestMagicCode(t, authStore, code, magicTTL)

	rec := verifyMagicCode(h, "999999")
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
	upsertTestMagicCode(t, authStore, code, -1*time.Minute)

	rec := verifyMagicCode(h, code)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 for expired code, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestMagicVerifyCodeAttemptCap(t *testing.T) {
	authStore := newMagicTestAuthStore()
	fm := &fakeMailer{}
	h := buildMagicHandler(authStore, fm)

	code := "123456"
	codeHash := upsertTestMagicCode(t, authStore, code, magicTTL)
	// Pre-set attempts to 5 (cap).
	authStore.magicCodes["test-user"] = magicCode{codeHash: codeHash, expiresAt: time.Now().UTC().Add(magicTTL).Format(time.RFC3339), attempts: 5}

	rec := verifyMagicCode(h, code)
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
	upsertTestMagicCode(t, authStore, code, magicTTL)

	// First use succeeds.
	rec := verifyMagicCode(h, code)
	if rec.Code != http.StatusOK {
		t.Fatalf("first use expected 200, got %d", rec.Code)
	}

	// Second use — code consumed, should 401.
	rec = verifyMagicCode(h, code)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("reuse expected 401, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestMagicVerifyCodeTOTPStepUpIssuesChallenge(t *testing.T) {
	authStore := newMagicTestAuthStore()
	authStore.totpConfirmed = true
	fm := &fakeMailer{}
	h := buildMagicHandler(authStore, fm)

	code := "123456"
	upsertTestMagicCode(t, authStore, code, magicTTL)

	rec := verifyMagicCode(h, code)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var body map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("bad JSON: %v", err)
	}
	if mfaRequired, _ := body["mfa_required"].(bool); !mfaRequired {
		t.Errorf("expected mfa_required=true, got %#v", body)
	}
	if tok, _ := body["challenge_token"].(string); tok == "" {
		t.Error("expected a non-empty challenge_token")
	}

	// No session cookie should be issued — the caller must complete the
	// TOTP challenge before a session exists.
	for _, c := range rec.Result().Cookies() {
		if c.Name == "dd_session" && c.Value != "" {
			t.Error("dd_session cookie should not be set until the TOTP challenge is completed")
		}
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
	upsertTestMagicCode(t, authStore, code, magicTTL)

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

	rec := verifyMagicCode(h, code)
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

// Stubs — not exercised by existing tests.

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
