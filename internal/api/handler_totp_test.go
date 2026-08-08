package api

import (
	"context"
	"encoding/base64"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gsaraiva2109/dietdaemon/core/types"
	"github.com/gsaraiva2109/dietdaemon/internal/auth"
	"github.com/pquerna/otp/totp"
)

// ---------------------------------------------------------------------------
// totpTestStore: a fakeAuthStore with real in-memory TOTP secret / recovery
// code / MFA challenge / session tracking, so handleTOTPEnroll,
// handleTOTPVerify, handleTOTPChallenge, handleTOTPDisable, and
// handleRegenerateRecovery can be driven through every branch instead of the
// package-wide fakeAuthStore's always-empty stubs.
// ---------------------------------------------------------------------------

type totpChallengeState struct {
	userID    string
	remember  bool
	expiresAt string
}

type totpTestStore struct {
	*fakeAuthStore

	secretByUser    map[string]string
	confirmedByUser map[string]bool
	recoveryHashes  map[string][]string
	challenges      map[string]totpChallengeState
	sessions        map[string]auth.Session

	upsertTOTPErr         error
	confirmTOTPErr        error
	getTOTPSecretErr      error
	deleteTOTPErr         error
	hasConfirmedErr       error
	replaceRecoveryErr    error
	consumeRecoveryErr    error
	createMFAChallengeErr error
	createSessionErr      error
}

func newTOTPTestStore() *totpTestStore {
	return &totpTestStore{
		fakeAuthStore:   newFakeAuthStore(),
		secretByUser:    map[string]string{},
		confirmedByUser: map[string]bool{},
		recoveryHashes:  map[string][]string{},
		challenges:      map[string]totpChallengeState{},
		sessions:        map[string]auth.Session{},
	}
}

func (s *totpTestStore) UpsertTOTPSecret(_ context.Context, userID, encSecret string) error {
	if s.upsertTOTPErr != nil {
		return s.upsertTOTPErr
	}
	s.secretByUser[userID] = encSecret
	s.confirmedByUser[userID] = false
	return nil
}

func (s *totpTestStore) ConfirmTOTP(_ context.Context, userID string) error {
	if s.confirmTOTPErr != nil {
		return s.confirmTOTPErr
	}
	s.confirmedByUser[userID] = true
	return nil
}

func (s *totpTestStore) GetTOTPSecret(_ context.Context, userID string) (string, bool, error) {
	if s.getTOTPSecretErr != nil {
		return "", false, s.getTOTPSecretErr
	}
	enc, ok := s.secretByUser[userID]
	if !ok {
		return "", false, types.ErrNotFound
	}
	return enc, s.confirmedByUser[userID], nil
}

func (s *totpTestStore) DeleteTOTP(_ context.Context, userID string) error {
	if s.deleteTOTPErr != nil {
		return s.deleteTOTPErr
	}
	delete(s.secretByUser, userID)
	delete(s.confirmedByUser, userID)
	return nil
}

func (s *totpTestStore) HasConfirmedTOTP(_ context.Context, userID string) (bool, error) {
	if s.hasConfirmedErr != nil {
		return false, s.hasConfirmedErr
	}
	return s.confirmedByUser[userID], nil
}

func (s *totpTestStore) ReplaceRecoveryCodes(_ context.Context, userID string, hashes []string) error {
	if s.replaceRecoveryErr != nil {
		return s.replaceRecoveryErr
	}
	s.recoveryHashes[userID] = hashes
	return nil
}

func (s *totpTestStore) ConsumeRecoveryCode(_ context.Context, userID, hash string) (bool, error) {
	if s.consumeRecoveryErr != nil {
		return false, s.consumeRecoveryErr
	}
	hashes := s.recoveryHashes[userID]
	for i, h := range hashes {
		if h == hash {
			s.recoveryHashes[userID] = append(hashes[:i], hashes[i+1:]...)
			return true, nil
		}
	}
	return false, nil
}

func (s *totpTestStore) CreateMFAChallenge(_ context.Context, id, userID string, remember bool, expiresAt string) error {
	if s.createMFAChallengeErr != nil {
		return s.createMFAChallengeErr
	}
	s.challenges[id] = totpChallengeState{userID: userID, remember: remember, expiresAt: expiresAt}
	return nil
}

func (s *totpTestStore) GetMFAChallenge(_ context.Context, id string) (string, bool, string, error) {
	c, ok := s.challenges[id]
	if !ok {
		return "", false, "", types.ErrNotFound
	}
	return c.userID, c.remember, c.expiresAt, nil
}

func (s *totpTestStore) DeleteMFAChallenge(_ context.Context, id string) error {
	delete(s.challenges, id)
	return nil
}

func (s *totpTestStore) CreateSession(_ context.Context, sess auth.Session) error {
	if s.createSessionErr != nil {
		return s.createSessionErr
	}
	s.sessions[sess.ID] = sess
	return nil
}

func buildTOTPHandler(authStore *totpTestStore, meals *fakeMealStore, encKey []byte, lockoutCfg auth.LockoutConfig) *Handler {
	return New(meals, &fakeMealLogger{}, time.UTC, nil, nil,
		WithAuth(authStore, AuthRepos{Sessions: authStore, LoginAttempts: authStore, TOTP: authStore, MFAChallenges: authStore, RecoveryCodes: authStore}, encKey, "DietDaemon", AuthConfig{
			SessionCfg: auth.SessionConfig{
				IdleTTL:     time.Hour,
				AbsoluteTTL: 24 * time.Hour,
				RememberTTL: 72 * time.Hour,
			},
			LockoutCfg:       lockoutCfg,
			RegistrationMode: types.RegistrationOpen,
		}),
		WithMailer(&fakeMailer{}, "none"),
		WithPublicBaseURL("http://localhost:8080"),
	)
}

func defaultTOTPMeals() *fakeMealStore {
	meals := newFakeMealStore()
	meals.user = types.User{ID: "test-user", AccountID: "acct-1", Email: "totp@example.com", Status: "active", CreatedAt: time.Now().UTC()}
	return meals
}

// ---------------------------------------------------------------------------
// handleTOTPEnroll
// ---------------------------------------------------------------------------

func TestHandleTOTPEnrollNotConfigured(t *testing.T) {
	h := buildTOTPHandler(newTOTPTestStore(), defaultTOTPMeals(), nil, auth.DefaultLockoutConfig())

	rec := doRequest(h, http.MethodPost, "/api/v1/auth/totp/enroll", nil, nil)
	if rec.Code != http.StatusNotImplemented {
		t.Fatalf("enroll status = %d, want 501: %s", rec.Code, rec.Body.String())
	}
}

func TestHandleTOTPEnrollGetUserError(t *testing.T) {
	meals := defaultTOTPMeals()
	meals.getUserErr = errors.New("db down")
	encKey := make([]byte, 32)
	h := buildTOTPHandler(newTOTPTestStore(), meals, encKey, auth.DefaultLockoutConfig())

	rec := doRequest(h, http.MethodPost, "/api/v1/auth/totp/enroll", nil, nil)
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("enroll status = %d, want 500: %s", rec.Code, rec.Body.String())
	}
}

func TestHandleTOTPEnrollGenerateSecretError(t *testing.T) {
	meals := defaultTOTPMeals()
	meals.user.Email = "" // empty account name makes GenerateSecret fail
	encKey := make([]byte, 32)
	h := buildTOTPHandler(newTOTPTestStore(), meals, encKey, auth.DefaultLockoutConfig())

	rec := doRequest(h, http.MethodPost, "/api/v1/auth/totp/enroll", nil, nil)
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("enroll status = %d, want 500: %s", rec.Code, rec.Body.String())
	}
}

func TestHandleTOTPEnrollEncryptError(t *testing.T) {
	// A non-nil key of the wrong length passes totpReady but fails Encrypt.
	badKey := make([]byte, 16)
	h := buildTOTPHandler(newTOTPTestStore(), defaultTOTPMeals(), badKey, auth.DefaultLockoutConfig())

	rec := doRequest(h, http.MethodPost, "/api/v1/auth/totp/enroll", nil, nil)
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("enroll status = %d, want 500: %s", rec.Code, rec.Body.String())
	}
}

func TestHandleTOTPEnrollUpsertError(t *testing.T) {
	authStore := newTOTPTestStore()
	authStore.upsertTOTPErr = errors.New("store unavailable")
	encKey := make([]byte, 32)
	h := buildTOTPHandler(authStore, defaultTOTPMeals(), encKey, auth.DefaultLockoutConfig())

	rec := doRequest(h, http.MethodPost, "/api/v1/auth/totp/enroll", nil, nil)
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("enroll status = %d, want 500: %s", rec.Code, rec.Body.String())
	}
}

func TestHandleTOTPEnrollSuccess(t *testing.T) {
	authStore := newTOTPTestStore()
	encKey := make([]byte, 32)
	h := buildTOTPHandler(authStore, defaultTOTPMeals(), encKey, auth.DefaultLockoutConfig())

	rec := doRequest(h, http.MethodPost, "/api/v1/auth/totp/enroll", nil, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("enroll status = %d: %s", rec.Code, rec.Body.String())
	}
	body := decodeJSON[map[string]string](t, rec)
	if body["secret"] == "" || body["otpauth_url"] == "" {
		t.Errorf("expected secret and otpauth_url, got %#v", body)
	}
	if _, ok := authStore.secretByUser["test-user"]; !ok {
		t.Error("expected encrypted secret to be persisted")
	}
}

// ---------------------------------------------------------------------------
// handleTOTPVerify
// ---------------------------------------------------------------------------

// enrollTOTPSecret encrypts secret with encKey and stores it unconfirmed for
// userID, mirroring what handleTOTPEnroll does.
func enrollTOTPSecret(t *testing.T, authStore *totpTestStore, encKey []byte, userID, secret string) {
	t.Helper()
	ct, err := auth.Encrypt([]byte(secret), encKey)
	if err != nil {
		t.Fatalf("Encrypt: %v", err)
	}
	authStore.secretByUser[userID] = base64.RawStdEncoding.EncodeToString(ct)
	authStore.confirmedByUser[userID] = false
}

func TestHandleTOTPVerifyNotConfigured(t *testing.T) {
	h := buildTOTPHandler(newTOTPTestStore(), defaultTOTPMeals(), nil, auth.DefaultLockoutConfig())
	rec := doRequest(h, http.MethodPost, "/api/v1/auth/totp/verify", map[string]string{"code": "123456"}, nil)
	if rec.Code != http.StatusNotImplemented {
		t.Fatalf("verify status = %d, want 501: %s", rec.Code, rec.Body.String())
	}
}

func TestHandleTOTPVerifyInvalidJSON(t *testing.T) {
	encKey := make([]byte, 32)
	h := buildTOTPHandler(newTOTPTestStore(), defaultTOTPMeals(), encKey, auth.DefaultLockoutConfig())

	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/totp/verify", strings.NewReader("not json"))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer test-api-key")
	rec := httptest.NewRecorder()
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("verify status = %d, want 400", rec.Code)
	}
}

func TestHandleTOTPVerifyInvalidCodeFormat(t *testing.T) {
	encKey := make([]byte, 32)
	h := buildTOTPHandler(newTOTPTestStore(), defaultTOTPMeals(), encKey, auth.DefaultLockoutConfig())

	rec := doRequest(h, http.MethodPost, "/api/v1/auth/totp/verify", map[string]string{"code": "abc"}, nil)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("verify status = %d, want 400: %s", rec.Code, rec.Body.String())
	}
}

func TestHandleTOTPVerifyNoSecretEnrolled(t *testing.T) {
	encKey := make([]byte, 32)
	h := buildTOTPHandler(newTOTPTestStore(), defaultTOTPMeals(), encKey, auth.DefaultLockoutConfig())

	rec := doRequest(h, http.MethodPost, "/api/v1/auth/totp/verify", map[string]string{"code": "123456"}, nil)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("verify status = %d, want 404: %s", rec.Code, rec.Body.String())
	}
}

func TestHandleTOTPVerifyAlreadyConfirmed(t *testing.T) {
	authStore := newTOTPTestStore()
	encKey := make([]byte, 32)
	secret, _, err := auth.GenerateSecret("DietDaemon", "totp@example.com")
	if err != nil {
		t.Fatalf("GenerateSecret: %v", err)
	}
	enrollTOTPSecret(t, authStore, encKey, "test-user", secret)
	authStore.confirmedByUser["test-user"] = true
	h := buildTOTPHandler(authStore, defaultTOTPMeals(), encKey, auth.DefaultLockoutConfig())

	rec := doRequest(h, http.MethodPost, "/api/v1/auth/totp/verify", map[string]string{"code": "123456"}, nil)
	if rec.Code != http.StatusConflict {
		t.Fatalf("verify status = %d, want 409: %s", rec.Code, rec.Body.String())
	}
}

func TestHandleTOTPVerifyBadBase64(t *testing.T) {
	authStore := newTOTPTestStore()
	authStore.secretByUser["test-user"] = "not-valid-base64!!!"
	encKey := make([]byte, 32)
	h := buildTOTPHandler(authStore, defaultTOTPMeals(), encKey, auth.DefaultLockoutConfig())

	rec := doRequest(h, http.MethodPost, "/api/v1/auth/totp/verify", map[string]string{"code": "123456"}, nil)
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("verify status = %d, want 500: %s", rec.Code, rec.Body.String())
	}
}

func TestHandleTOTPVerifyDecryptError(t *testing.T) {
	authStore := newTOTPTestStore()
	encKeyA := make([]byte, 32)
	encKeyB := make([]byte, 32)
	encKeyB[0] = 1 // different key
	secret, _, err := auth.GenerateSecret("DietDaemon", "totp@example.com")
	if err != nil {
		t.Fatalf("GenerateSecret: %v", err)
	}
	enrollTOTPSecret(t, authStore, encKeyA, "test-user", secret)
	h := buildTOTPHandler(authStore, defaultTOTPMeals(), encKeyB, auth.DefaultLockoutConfig())

	rec := doRequest(h, http.MethodPost, "/api/v1/auth/totp/verify", map[string]string{"code": "123456"}, nil)
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("verify status = %d, want 500: %s", rec.Code, rec.Body.String())
	}
}

func TestHandleTOTPVerifyWrongCode(t *testing.T) {
	authStore := newTOTPTestStore()
	encKey := make([]byte, 32)
	secret, _, err := auth.GenerateSecret("DietDaemon", "totp@example.com")
	if err != nil {
		t.Fatalf("GenerateSecret: %v", err)
	}
	enrollTOTPSecret(t, authStore, encKey, "test-user", secret)
	h := buildTOTPHandler(authStore, defaultTOTPMeals(), encKey, auth.DefaultLockoutConfig())

	rec := doRequest(h, http.MethodPost, "/api/v1/auth/totp/verify", map[string]string{"code": "000000"}, nil)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("verify status = %d, want 400: %s", rec.Code, rec.Body.String())
	}
}

func TestHandleTOTPVerifyConfirmError(t *testing.T) {
	authStore := newTOTPTestStore()
	authStore.confirmTOTPErr = errors.New("store unavailable")
	encKey := make([]byte, 32)
	secret, _, err := auth.GenerateSecret("DietDaemon", "totp@example.com")
	if err != nil {
		t.Fatalf("GenerateSecret: %v", err)
	}
	enrollTOTPSecret(t, authStore, encKey, "test-user", secret)
	code, err := totp.GenerateCode(secret, time.Now())
	if err != nil {
		t.Fatalf("GenerateCode: %v", err)
	}
	h := buildTOTPHandler(authStore, defaultTOTPMeals(), encKey, auth.DefaultLockoutConfig())

	rec := doRequest(h, http.MethodPost, "/api/v1/auth/totp/verify", map[string]string{"code": code}, nil)
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("verify status = %d, want 500: %s", rec.Code, rec.Body.String())
	}
}

func TestHandleTOTPVerifyReplaceRecoveryCodesError(t *testing.T) {
	authStore := newTOTPTestStore()
	authStore.replaceRecoveryErr = errors.New("store unavailable")
	encKey := make([]byte, 32)
	secret, _, err := auth.GenerateSecret("DietDaemon", "totp@example.com")
	if err != nil {
		t.Fatalf("GenerateSecret: %v", err)
	}
	enrollTOTPSecret(t, authStore, encKey, "test-user", secret)
	code, err := totp.GenerateCode(secret, time.Now())
	if err != nil {
		t.Fatalf("GenerateCode: %v", err)
	}
	h := buildTOTPHandler(authStore, defaultTOTPMeals(), encKey, auth.DefaultLockoutConfig())

	rec := doRequest(h, http.MethodPost, "/api/v1/auth/totp/verify", map[string]string{"code": code}, nil)
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("verify status = %d, want 500: %s", rec.Code, rec.Body.String())
	}
}

func TestHandleTOTPVerifySuccess(t *testing.T) {
	authStore := newTOTPTestStore()
	encKey := make([]byte, 32)
	secret, _, err := auth.GenerateSecret("DietDaemon", "totp@example.com")
	if err != nil {
		t.Fatalf("GenerateSecret: %v", err)
	}
	enrollTOTPSecret(t, authStore, encKey, "test-user", secret)
	code, err := totp.GenerateCode(secret, time.Now())
	if err != nil {
		t.Fatalf("GenerateCode: %v", err)
	}
	h := buildTOTPHandler(authStore, defaultTOTPMeals(), encKey, auth.DefaultLockoutConfig())

	rec := doRequest(h, http.MethodPost, "/api/v1/auth/totp/verify", map[string]string{"code": code}, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("verify status = %d: %s", rec.Code, rec.Body.String())
	}
	body := decodeJSON[map[string][]string](t, rec)
	if len(body["recovery_codes"]) != 10 {
		t.Errorf("expected 10 recovery codes, got %d", len(body["recovery_codes"]))
	}
	if !authStore.confirmedByUser["test-user"] {
		t.Error("expected totp to be confirmed")
	}
	if len(authStore.recoveryHashes["test-user"]) != 10 {
		t.Errorf("expected 10 persisted recovery hashes, got %d", len(authStore.recoveryHashes["test-user"]))
	}
	foundAudit := false
	for _, ev := range authStore.auditEvents {
		if ev.Event == "totp.enabled" {
			foundAudit = true
		}
	}
	if !foundAudit {
		t.Error("expected totp.enabled audit event")
	}
}

// ---------------------------------------------------------------------------
// handleTOTPDisable
// ---------------------------------------------------------------------------

func TestHandleTOTPDisableNotConfigured(t *testing.T) {
	h := buildTOTPHandler(newTOTPTestStore(), defaultTOTPMeals(), nil, auth.DefaultLockoutConfig())
	rec := doRequest(h, http.MethodDelete, "/api/v1/auth/totp", nil, nil)
	if rec.Code != http.StatusNotImplemented {
		t.Fatalf("disable status = %d, want 501: %s", rec.Code, rec.Body.String())
	}
}

func TestHandleTOTPDisableError(t *testing.T) {
	authStore := newTOTPTestStore()
	authStore.deleteTOTPErr = errors.New("store unavailable")
	encKey := make([]byte, 32)
	h := buildTOTPHandler(authStore, defaultTOTPMeals(), encKey, auth.DefaultLockoutConfig())

	rec := doRequest(h, http.MethodDelete, "/api/v1/auth/totp", nil, nil)
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("disable status = %d, want 500: %s", rec.Code, rec.Body.String())
	}
}

func TestHandleTOTPDisableSuccess(t *testing.T) {
	authStore := newTOTPTestStore()
	authStore.secretByUser["test-user"] = "whatever"
	authStore.confirmedByUser["test-user"] = true
	authStore.recoveryHashes["test-user"] = []string{"h1", "h2"}
	encKey := make([]byte, 32)
	h := buildTOTPHandler(authStore, defaultTOTPMeals(), encKey, auth.DefaultLockoutConfig())

	rec := doRequest(h, http.MethodDelete, "/api/v1/auth/totp", nil, nil)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("disable status = %d, want 204: %s", rec.Code, rec.Body.String())
	}
	if _, ok := authStore.secretByUser["test-user"]; ok {
		t.Error("expected secret to be deleted")
	}
	if authStore.recoveryHashes["test-user"] != nil {
		t.Error("expected recovery codes to be cleared")
	}
	foundAudit := false
	for _, ev := range authStore.auditEvents {
		if ev.Event == "totp.disabled" {
			foundAudit = true
		}
	}
	if !foundAudit {
		t.Error("expected totp.disabled audit event")
	}
}

// ---------------------------------------------------------------------------
// handleRegenerateRecovery
// ---------------------------------------------------------------------------

func TestHandleRegenerateRecoveryNotConfigured(t *testing.T) {
	h := buildTOTPHandler(newTOTPTestStore(), defaultTOTPMeals(), nil, auth.DefaultLockoutConfig())
	rec := doRequest(h, http.MethodPost, "/api/v1/auth/totp/recovery-codes/regenerate", nil, nil)
	if rec.Code != http.StatusNotImplemented {
		t.Fatalf("regenerate status = %d, want 501: %s", rec.Code, rec.Body.String())
	}
}

func TestHandleRegenerateRecoveryHasConfirmedError(t *testing.T) {
	authStore := newTOTPTestStore()
	authStore.hasConfirmedErr = errors.New("store unavailable")
	encKey := make([]byte, 32)
	h := buildTOTPHandler(authStore, defaultTOTPMeals(), encKey, auth.DefaultLockoutConfig())

	rec := doRequest(h, http.MethodPost, "/api/v1/auth/totp/recovery-codes/regenerate", nil, nil)
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("regenerate status = %d, want 500: %s", rec.Code, rec.Body.String())
	}
}

func TestHandleRegenerateRecoveryNotEnabled(t *testing.T) {
	encKey := make([]byte, 32)
	h := buildTOTPHandler(newTOTPTestStore(), defaultTOTPMeals(), encKey, auth.DefaultLockoutConfig())

	rec := doRequest(h, http.MethodPost, "/api/v1/auth/totp/recovery-codes/regenerate", nil, nil)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("regenerate status = %d, want 400: %s", rec.Code, rec.Body.String())
	}
}

func TestHandleRegenerateRecoveryReplaceError(t *testing.T) {
	authStore := newTOTPTestStore()
	authStore.confirmedByUser["test-user"] = true
	authStore.replaceRecoveryErr = errors.New("store unavailable")
	encKey := make([]byte, 32)
	h := buildTOTPHandler(authStore, defaultTOTPMeals(), encKey, auth.DefaultLockoutConfig())

	rec := doRequest(h, http.MethodPost, "/api/v1/auth/totp/recovery-codes/regenerate", nil, nil)
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("regenerate status = %d, want 500: %s", rec.Code, rec.Body.String())
	}
}

func TestHandleRegenerateRecoverySuccess(t *testing.T) {
	authStore := newTOTPTestStore()
	authStore.confirmedByUser["test-user"] = true
	encKey := make([]byte, 32)
	h := buildTOTPHandler(authStore, defaultTOTPMeals(), encKey, auth.DefaultLockoutConfig())

	rec := doRequest(h, http.MethodPost, "/api/v1/auth/totp/recovery-codes/regenerate", nil, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("regenerate status = %d: %s", rec.Code, rec.Body.String())
	}
	body := decodeJSON[map[string][]string](t, rec)
	if len(body["recovery_codes"]) != 10 {
		t.Errorf("expected 10 recovery codes, got %d", len(body["recovery_codes"]))
	}
}

// ---------------------------------------------------------------------------
// handleTOTPChallenge and its decomposed helpers
// ---------------------------------------------------------------------------

func TestHandleTOTPChallengeNotConfigured(t *testing.T) {
	h := buildTOTPHandler(newTOTPTestStore(), defaultTOTPMeals(), nil, auth.DefaultLockoutConfig())
	rec := doRequest(h, http.MethodPost, "/api/v1/auth/totp/challenge", map[string]string{"challenge_token": "x", "code": "123456"}, nil)
	if rec.Code != http.StatusNotImplemented {
		t.Fatalf("challenge status = %d, want 501: %s", rec.Code, rec.Body.String())
	}
}

func TestHandleTOTPChallengeInvalidJSON(t *testing.T) {
	encKey := make([]byte, 32)
	h := buildTOTPHandler(newTOTPTestStore(), defaultTOTPMeals(), encKey, auth.DefaultLockoutConfig())

	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/totp/challenge", strings.NewReader("not json"))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer test-api-key")
	rec := httptest.NewRecorder()
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("challenge status = %d, want 400", rec.Code)
	}
}

func TestHandleTOTPChallengeMissingToken(t *testing.T) {
	encKey := make([]byte, 32)
	h := buildTOTPHandler(newTOTPTestStore(), defaultTOTPMeals(), encKey, auth.DefaultLockoutConfig())

	rec := doRequest(h, http.MethodPost, "/api/v1/auth/totp/challenge", map[string]string{"code": "123456"}, nil)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("challenge status = %d, want 400: %s", rec.Code, rec.Body.String())
	}
}

func TestHandleTOTPChallengeUnknownToken(t *testing.T) {
	encKey := make([]byte, 32)
	h := buildTOTPHandler(newTOTPTestStore(), defaultTOTPMeals(), encKey, auth.DefaultLockoutConfig())

	rec := doRequest(h, http.MethodPost, "/api/v1/auth/totp/challenge", map[string]string{"challenge_token": "bogus", "code": "123456"}, nil)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("challenge status = %d, want 401: %s", rec.Code, rec.Body.String())
	}
}

func TestHandleTOTPChallengeExpired(t *testing.T) {
	authStore := newTOTPTestStore()
	token := "expired-token"
	authStore.challenges[auth.HashToken(token)] = totpChallengeState{
		userID: "test-user", expiresAt: time.Now().UTC().Add(-time.Minute).Format(time.RFC3339),
	}
	encKey := make([]byte, 32)
	h := buildTOTPHandler(authStore, defaultTOTPMeals(), encKey, auth.DefaultLockoutConfig())

	rec := doRequest(h, http.MethodPost, "/api/v1/auth/totp/challenge", map[string]string{"challenge_token": token, "code": "123456"}, nil)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("challenge status = %d, want 401: %s", rec.Code, rec.Body.String())
	}
	if _, ok := authStore.challenges[auth.HashToken(token)]; ok {
		t.Error("expected expired challenge to be deleted")
	}
}

func TestHandleTOTPChallengeLockoutStoreError(t *testing.T) {
	authStore := newTOTPTestStore()
	authStore.recentFailedAttemptsErr = errors.New("store unavailable")
	token := "valid-token"
	authStore.challenges[auth.HashToken(token)] = totpChallengeState{
		userID: "test-user", expiresAt: time.Now().UTC().Add(5 * time.Minute).Format(time.RFC3339),
	}
	encKey := make([]byte, 32)
	h := buildTOTPHandler(authStore, defaultTOTPMeals(), encKey, auth.DefaultLockoutConfig())

	rec := doRequest(h, http.MethodPost, "/api/v1/auth/totp/challenge", map[string]string{"challenge_token": token, "code": "123456"}, nil)
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("challenge status = %d, want 500: %s", rec.Code, rec.Body.String())
	}
}

func TestHandleTOTPChallengeBadCodeFormat(t *testing.T) {
	authStore := newTOTPTestStore()
	token := "valid-token"
	authStore.challenges[auth.HashToken(token)] = totpChallengeState{
		userID: "test-user", expiresAt: time.Now().UTC().Add(5 * time.Minute).Format(time.RFC3339),
	}
	encKey := make([]byte, 32)
	h := buildTOTPHandler(authStore, defaultTOTPMeals(), encKey, auth.DefaultLockoutConfig())

	rec := doRequest(h, http.MethodPost, "/api/v1/auth/totp/challenge", map[string]string{"challenge_token": token, "code": "12a456"}, nil)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("challenge status = %d, want 400: %s", rec.Code, rec.Body.String())
	}
	foundAudit := false
	for _, ev := range authStore.auditEvents {
		if ev.Event == auditMFAFail && ev.Meta == "bad code format" {
			foundAudit = true
		}
	}
	if !foundAudit {
		t.Error("expected mfa.fail audit event with 'bad code format'")
	}
}

func TestHandleTOTPChallengeNoSecretEnrolled(t *testing.T) {
	authStore := newTOTPTestStore()
	token := "valid-token"
	authStore.challenges[auth.HashToken(token)] = totpChallengeState{
		userID: "test-user", expiresAt: time.Now().UTC().Add(5 * time.Minute).Format(time.RFC3339),
	}
	encKey := make([]byte, 32)
	h := buildTOTPHandler(authStore, defaultTOTPMeals(), encKey, auth.DefaultLockoutConfig())

	rec := doRequest(h, http.MethodPost, "/api/v1/auth/totp/challenge", map[string]string{"challenge_token": token, "code": "123456"}, nil)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("challenge status = %d, want 404: %s", rec.Code, rec.Body.String())
	}
}

func TestHandleTOTPChallengeBadBase64Secret(t *testing.T) {
	authStore := newTOTPTestStore()
	authStore.secretByUser["test-user"] = "not-valid-base64!!!"
	token := "valid-token"
	authStore.challenges[auth.HashToken(token)] = totpChallengeState{
		userID: "test-user", expiresAt: time.Now().UTC().Add(5 * time.Minute).Format(time.RFC3339),
	}
	encKey := make([]byte, 32)
	h := buildTOTPHandler(authStore, defaultTOTPMeals(), encKey, auth.DefaultLockoutConfig())

	rec := doRequest(h, http.MethodPost, "/api/v1/auth/totp/challenge", map[string]string{"challenge_token": token, "code": "123456"}, nil)
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("challenge status = %d, want 500: %s", rec.Code, rec.Body.String())
	}
}

func TestHandleTOTPChallengeDecryptError(t *testing.T) {
	authStore := newTOTPTestStore()
	encKeyA := make([]byte, 32)
	encKeyB := make([]byte, 32)
	encKeyB[0] = 1
	secret, _, err := auth.GenerateSecret("DietDaemon", "totp@example.com")
	if err != nil {
		t.Fatalf("GenerateSecret: %v", err)
	}
	enrollTOTPSecret(t, authStore, encKeyA, "test-user", secret)
	authStore.confirmedByUser["test-user"] = true
	token := "valid-token"
	authStore.challenges[auth.HashToken(token)] = totpChallengeState{
		userID: "test-user", expiresAt: time.Now().UTC().Add(5 * time.Minute).Format(time.RFC3339),
	}
	h := buildTOTPHandler(authStore, defaultTOTPMeals(), encKeyB, auth.DefaultLockoutConfig())

	rec := doRequest(h, http.MethodPost, "/api/v1/auth/totp/challenge", map[string]string{"challenge_token": token, "code": "123456"}, nil)
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("challenge status = %d, want 500: %s", rec.Code, rec.Body.String())
	}
}

func TestHandleTOTPChallengeWrongCodeAudited(t *testing.T) {
	authStore := newTOTPTestStore()
	encKey := make([]byte, 32)
	secret, _, err := auth.GenerateSecret("DietDaemon", "totp@example.com")
	if err != nil {
		t.Fatalf("GenerateSecret: %v", err)
	}
	enrollTOTPSecret(t, authStore, encKey, "test-user", secret)
	authStore.confirmedByUser["test-user"] = true
	token := "valid-token"
	authStore.challenges[auth.HashToken(token)] = totpChallengeState{
		userID: "test-user", expiresAt: time.Now().UTC().Add(5 * time.Minute).Format(time.RFC3339),
	}
	h := buildTOTPHandler(authStore, defaultTOTPMeals(), encKey, auth.DefaultLockoutConfig())

	rec := doRequest(h, http.MethodPost, "/api/v1/auth/totp/challenge", map[string]string{"challenge_token": token, "code": "000000"}, nil)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("challenge status = %d, want 401: %s", rec.Code, rec.Body.String())
	}
	foundAudit := false
	for _, ev := range authStore.auditEvents {
		if ev.Event == auditMFAFail && ev.Meta == "bad totp code" {
			foundAudit = true
		}
	}
	if !foundAudit {
		t.Error("expected mfa.fail audit event with 'bad totp code'")
	}
}

func TestHandleTOTPChallengeCodeSuccess(t *testing.T) {
	authStore := newTOTPTestStore()
	encKey := make([]byte, 32)
	secret, _, err := auth.GenerateSecret("DietDaemon", "totp@example.com")
	if err != nil {
		t.Fatalf("GenerateSecret: %v", err)
	}
	enrollTOTPSecret(t, authStore, encKey, "test-user", secret)
	authStore.confirmedByUser["test-user"] = true
	token := "valid-token"
	authStore.challenges[auth.HashToken(token)] = totpChallengeState{
		userID: "test-user", remember: true, expiresAt: time.Now().UTC().Add(5 * time.Minute).Format(time.RFC3339),
	}
	code, err := totp.GenerateCode(secret, time.Now())
	if err != nil {
		t.Fatalf("GenerateCode: %v", err)
	}
	h := buildTOTPHandler(authStore, defaultTOTPMeals(), encKey, auth.DefaultLockoutConfig())

	rec := doRequest(h, http.MethodPost, "/api/v1/auth/totp/challenge", map[string]string{"challenge_token": token, "code": code}, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("challenge status = %d: %s", rec.Code, rec.Body.String())
	}
	got := decodeJSON[sessionResponse](t, rec)
	if got.User.ID != "test-user" {
		t.Errorf("session user = %+v", got.User)
	}
	if len(authStore.sessions) != 1 {
		t.Errorf("expected one session created, got %d", len(authStore.sessions))
	}
	for _, sess := range authStore.sessions {
		if !sess.Remember {
			t.Error("expected remembered session")
		}
	}
	if _, ok := authStore.challenges[auth.HashToken(token)]; ok {
		t.Error("expected challenge to be consumed")
	}
	foundAudit := false
	for _, ev := range authStore.auditEvents {
		if ev.Event == "mfa.success" {
			foundAudit = true
		}
	}
	if !foundAudit {
		t.Error("expected mfa.success audit event")
	}
}

func TestHandleTOTPChallengeRejectsPendingDeletionAccount(t *testing.T) {
	authStore := newTOTPTestStore()
	encKey := make([]byte, 32)
	secret, _, err := auth.GenerateSecret("DietDaemon", "totp@example.com")
	if err != nil {
		t.Fatalf("GenerateSecret: %v", err)
	}
	enrollTOTPSecret(t, authStore, encKey, "test-user", secret)
	authStore.confirmedByUser["test-user"] = true
	if err := authStore.RequestAccountDeletion(context.Background(), "test-user"); err != nil {
		t.Fatalf("RequestAccountDeletion: %v", err)
	}
	token := "valid-token"
	authStore.challenges[auth.HashToken(token)] = totpChallengeState{
		userID: "test-user", expiresAt: time.Now().UTC().Add(5 * time.Minute).Format(time.RFC3339),
	}
	code, err := totp.GenerateCode(secret, time.Now())
	if err != nil {
		t.Fatalf("GenerateCode: %v", err)
	}
	h := buildTOTPHandler(authStore, defaultTOTPMeals(), encKey, auth.DefaultLockoutConfig())

	rec := doRequest(h, http.MethodPost, "/api/v1/auth/totp/challenge", map[string]string{"challenge_token": token, "code": code}, nil)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("challenge status = %d, want 403: %s", rec.Code, rec.Body.String())
	}
	if body := decodeJSON[map[string]any](t, rec); body["error"] != "pending_deletion" {
		t.Errorf("error body = %#v, want pending_deletion", body)
	}
	if len(authStore.sessions) != 0 {
		t.Errorf("created sessions = %d, want 0: pending-deletion account must not get a new session via TOTP challenge", len(authStore.sessions))
	}
}

func TestHandleTOTPChallengeCreateSessionError(t *testing.T) {
	authStore := newTOTPTestStore()
	authStore.createSessionErr = errors.New("store unavailable")
	encKey := make([]byte, 32)
	secret, _, err := auth.GenerateSecret("DietDaemon", "totp@example.com")
	if err != nil {
		t.Fatalf("GenerateSecret: %v", err)
	}
	enrollTOTPSecret(t, authStore, encKey, "test-user", secret)
	authStore.confirmedByUser["test-user"] = true
	token := "valid-token"
	authStore.challenges[auth.HashToken(token)] = totpChallengeState{
		userID: "test-user", expiresAt: time.Now().UTC().Add(5 * time.Minute).Format(time.RFC3339),
	}
	code, err := totp.GenerateCode(secret, time.Now())
	if err != nil {
		t.Fatalf("GenerateCode: %v", err)
	}
	h := buildTOTPHandler(authStore, defaultTOTPMeals(), encKey, auth.DefaultLockoutConfig())

	rec := doRequest(h, http.MethodPost, "/api/v1/auth/totp/challenge", map[string]string{"challenge_token": token, "code": code}, nil)
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("challenge status = %d, want 500: %s", rec.Code, rec.Body.String())
	}
}

func TestHandleTOTPChallengeFinishGetUserError(t *testing.T) {
	authStore := newTOTPTestStore()
	encKey := make([]byte, 32)
	secret, _, err := auth.GenerateSecret("DietDaemon", "totp@example.com")
	if err != nil {
		t.Fatalf("GenerateSecret: %v", err)
	}
	enrollTOTPSecret(t, authStore, encKey, "test-user", secret)
	authStore.confirmedByUser["test-user"] = true
	token := "valid-token"
	authStore.challenges[auth.HashToken(token)] = totpChallengeState{
		userID: "test-user", expiresAt: time.Now().UTC().Add(5 * time.Minute).Format(time.RFC3339),
	}
	code, err := totp.GenerateCode(secret, time.Now())
	if err != nil {
		t.Fatalf("GenerateCode: %v", err)
	}
	meals := defaultTOTPMeals()
	meals.getUserErr = errors.New("db down")
	h := buildTOTPHandler(authStore, meals, encKey, auth.DefaultLockoutConfig())

	rec := doRequest(h, http.MethodPost, "/api/v1/auth/totp/challenge", map[string]string{"challenge_token": token, "code": code}, nil)
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("challenge status = %d, want 500: %s", rec.Code, rec.Body.String())
	}
	// The session was still created before the GetUser failure.
	if len(authStore.sessions) != 1 {
		t.Errorf("expected session to be created despite GetUser failure, got %d", len(authStore.sessions))
	}
}

// --- Recovery-code branch of verifyTOTPChallengeCode ---

func TestHandleTOTPChallengeRecoveryCodeConsumeError(t *testing.T) {
	authStore := newTOTPTestStore()
	authStore.consumeRecoveryErr = errors.New("store unavailable")
	token := "valid-token"
	authStore.challenges[auth.HashToken(token)] = totpChallengeState{
		userID: "test-user", expiresAt: time.Now().UTC().Add(5 * time.Minute).Format(time.RFC3339),
	}
	encKey := make([]byte, 32)
	h := buildTOTPHandler(authStore, defaultTOTPMeals(), encKey, auth.DefaultLockoutConfig())

	rec := doRequest(h, http.MethodPost, "/api/v1/auth/totp/challenge", map[string]string{"challenge_token": token, "recovery_code": "whatever"}, nil)
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("challenge status = %d, want 500: %s", rec.Code, rec.Body.String())
	}
}

func TestHandleTOTPChallengeRecoveryCodeWrong(t *testing.T) {
	authStore := newTOTPTestStore()
	token := "valid-token"
	authStore.challenges[auth.HashToken(token)] = totpChallengeState{
		userID: "test-user", expiresAt: time.Now().UTC().Add(5 * time.Minute).Format(time.RFC3339),
	}
	encKey := make([]byte, 32)
	h := buildTOTPHandler(authStore, defaultTOTPMeals(), encKey, auth.DefaultLockoutConfig())

	rec := doRequest(h, http.MethodPost, "/api/v1/auth/totp/challenge", map[string]string{"challenge_token": token, "recovery_code": "nonexistent"}, nil)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("challenge status = %d, want 401: %s", rec.Code, rec.Body.String())
	}
	foundAudit := false
	for _, ev := range authStore.auditEvents {
		if ev.Event == auditMFAFail && ev.Meta == "bad recovery code" {
			foundAudit = true
		}
	}
	if !foundAudit {
		t.Error("expected mfa.fail audit event with 'bad recovery code'")
	}
}

func TestHandleTOTPChallengeRecoveryCodeSuccess(t *testing.T) {
	authStore := newTOTPTestStore()
	recoveryCode := "abcd-efgh-ijkl"
	authStore.recoveryHashes["test-user"] = []string{auth.HashToken(recoveryCode)}
	token := "valid-token"
	authStore.challenges[auth.HashToken(token)] = totpChallengeState{
		userID: "test-user", expiresAt: time.Now().UTC().Add(5 * time.Minute).Format(time.RFC3339),
	}
	encKey := make([]byte, 32)
	h := buildTOTPHandler(authStore, defaultTOTPMeals(), encKey, auth.DefaultLockoutConfig())

	rec := doRequest(h, http.MethodPost, "/api/v1/auth/totp/challenge", map[string]string{"challenge_token": token, "recovery_code": recoveryCode}, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("challenge status = %d: %s", rec.Code, rec.Body.String())
	}
	got := decodeJSON[sessionResponse](t, rec)
	if got.User.ID != "test-user" {
		t.Errorf("session user = %+v", got.User)
	}
	if len(authStore.recoveryHashes["test-user"]) != 0 {
		t.Error("expected recovery code to be consumed")
	}
}
