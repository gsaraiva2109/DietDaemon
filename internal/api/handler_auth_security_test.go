package api

import (
	"context"
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"testing"
	"time"

	"github.com/gsaraiva2109/dietdaemon/core/types"
	"github.com/gsaraiva2109/dietdaemon/internal/auth"
)

// --- clientIP trusted-proxy tests (fix: XFF/X-Real-IP spoofing) ---

func TestClientIPUntrustedPeerIgnoresHeaders(t *testing.T) {
	h := &Handler{trustedProxies: []netip.Prefix{netip.MustParsePrefix("127.0.0.0/8")}}

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "203.0.113.9:5555" // not in trustedProxies
	req.Header.Set("X-Forwarded-For", "9.9.9.9")
	req.Header.Set("X-Real-IP", "8.8.8.8")

	if got := h.clientIP(req); got != "203.0.113.9" {
		t.Fatalf("clientIP = %q, want the untrusted peer's own address (headers must be ignored)", got)
	}
}

func TestClientIPTrustedPeerHonorsForwardedFor(t *testing.T) {
	h := &Handler{trustedProxies: []netip.Prefix{netip.MustParsePrefix("127.0.0.0/8")}}

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "127.0.0.1:5555" // trusted proxy
	req.Header.Set("X-Forwarded-For", "9.9.9.9, 127.0.0.1")

	if got := h.clientIP(req); got != "9.9.9.9" {
		t.Fatalf("clientIP = %q, want leftmost X-Forwarded-For entry from a trusted peer", got)
	}
}

func TestClientIPTrustedPeerFallsBackToRealIP(t *testing.T) {
	h := &Handler{trustedProxies: []netip.Prefix{netip.MustParsePrefix("127.0.0.0/8")}}

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "127.0.0.1:5555"
	req.Header.Set("X-Real-IP", "8.8.8.8")

	if got := h.clientIP(req); got != "8.8.8.8" {
		t.Fatalf("clientIP = %q, want X-Real-IP from a trusted peer", got)
	}
}

func TestClientIPNoTrustedProxiesConfigured(t *testing.T) {
	h := &Handler{} // trustedProxies is nil — trust nothing

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "127.0.0.1:5555"
	req.Header.Set("X-Forwarded-For", "9.9.9.9")

	if got := h.clientIP(req); got != "127.0.0.1" {
		t.Fatalf("clientIP = %q, want the raw peer address when no trusted proxies are configured", got)
	}
}

// --- Register: uniqueness-before-hash (fix: CPU-wasting/timing-leaking order) ---

func buildAuthSecurityHandler(authStore *fakeAuthStore) *Handler {
	store := newFakeMealStore()
	return New(store, authStore, &fakeMealLogger{}, time.UTC, authStore, authStore, authStore, authStore, authStore, nil, "DietDaemon", nil, &fakeMailer{}, "none", "http://localhost:8080", AuthConfig{
		SessionCfg: auth.SessionConfig{
			IdleTTL:     1 * time.Hour,
			AbsoluteTTL: 24 * time.Hour,
			RememberTTL: 72 * time.Hour,
		},
		LockoutCfg:       auth.DefaultLockoutConfig(),
		RegistrationMode: types.RegistrationOpen,
		CookieSecure:     false,
	}, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
}

// argon2id (memory=64MiB) takes tens of ms on ordinary hardware — comfortably
// above this bound, so it distinguishes "a hash ran" from "no hash ran"
// without being sensitive to machine speed.
const hashCostFloor = 8 * time.Millisecond

func TestHandleRegisterDuplicateEmailSkipsHash(t *testing.T) {
	authStore := newFakeAuthStore()
	authStore.userByEmail["taken@example.com"] = types.User{ID: "existing-user", Email: "taken@example.com", Status: "active", CreatedAt: time.Now().UTC()}
	h := buildAuthSecurityHandler(authStore)

	start := time.Now()
	rec := doRequest(h, "POST", "/api/v1/auth/register", map[string]string{
		"email":    "taken@example.com",
		"password": "correcthorsebatterystaple",
	}, nil)
	elapsed := time.Since(start)

	if rec.Code != http.StatusConflict {
		t.Fatalf("expected 409 for duplicate email, got %d: %s", rec.Code, rec.Body.String())
	}
	if elapsed >= hashCostFloor {
		t.Errorf("duplicate-email register took %v, expected well under %v (password should never be hashed once the email is known to be taken)", elapsed, hashCostFloor)
	}
}

// --- Login: dummy-hash fallback for nonexistent users (fix: timing side-channel) ---

func TestHandleLoginUnknownEmailStillHashes(t *testing.T) {
	authStore := newFakeAuthStore()
	h := buildAuthSecurityHandler(authStore)

	start := time.Now()
	rec := doRequest(h, "POST", "/api/v1/auth/login", map[string]string{
		"email":    "nobody@example.com",
		"password": "whatever-password",
	}, nil)
	elapsed := time.Since(start)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 for unknown email, got %d: %s", rec.Code, rec.Body.String())
	}
	if elapsed < hashCostFloor {
		t.Errorf("unknown-email login took %v, expected at least %v (must run a dummy hash so timing doesn't reveal the account doesn't exist)", elapsed, hashCostFloor)
	}
}

// --- TOTP challenge: per-user lockout (fix: unlimited code-guessing) ---

// totpChallengeAuthStore fakes a live MFA challenge and TOTP secret so
// handleTOTPChallenge can be driven with wrong codes to exercise lockout.
type totpChallengeAuthStore struct {
	*fakeAuthStore
	userID    string
	encSecret string
}

func (s *totpChallengeAuthStore) GetMFAChallenge(_ context.Context, _ string) (string, bool, string, error) {
	return s.userID, false, time.Now().UTC().Add(5 * time.Minute).Format(time.RFC3339), nil
}

func (s *totpChallengeAuthStore) GetTOTPSecret(_ context.Context, _ string) (string, bool, error) {
	return s.encSecret, true, nil
}

func TestHandleTOTPChallengeLockout(t *testing.T) {
	encKey := make([]byte, 32)
	secret, _, err := auth.GenerateSecret("DietDaemon", "test@example.com")
	if err != nil {
		t.Fatalf("GenerateSecret: %v", err)
	}
	ct, err := auth.Encrypt([]byte(secret), encKey)
	if err != nil {
		t.Fatalf("Encrypt: %v", err)
	}

	authStore := &totpChallengeAuthStore{
		fakeAuthStore: newFakeAuthStore(),
		userID:        "totp-user",
		encSecret:     base64.RawStdEncoding.EncodeToString(ct),
	}

	store := newFakeMealStore()
	lockoutCfg := auth.LockoutConfig{MaxAttempts: 3, Window: time.Hour, LockDuration: time.Hour}
	h := New(store, authStore, &fakeMealLogger{}, time.UTC, authStore, authStore, authStore, authStore, authStore, encKey, "DietDaemon", nil, &fakeMailer{}, "none", "http://localhost:8080", AuthConfig{
		SessionCfg: auth.SessionConfig{
			IdleTTL:     1 * time.Hour,
			AbsoluteTTL: 24 * time.Hour,
			RememberTTL: 72 * time.Hour,
		},
		LockoutCfg:       lockoutCfg,
		RegistrationMode: types.RegistrationOpen,
		CookieSecure:     false,
	}, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)

	body := map[string]string{"challenge_token": "any-token", "code": "000000"}

	var lastCode int
	for i := 0; i < lockoutCfg.MaxAttempts; i++ {
		rec := doRequest(h, "POST", "/api/v1/auth/totp/challenge", body, nil)
		lastCode = rec.Code
		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("attempt %d: expected 401 for wrong code, got %d: %s", i, rec.Code, rec.Body.String())
		}
	}

	// One more attempt beyond MaxAttempts must be locked out, regardless of
	// whether the code guessed happens to be correct.
	rec := doRequest(h, "POST", "/api/v1/auth/totp/challenge", body, nil)
	if rec.Code != http.StatusTooManyRequests {
		t.Fatalf("after %d failed attempts (last=%d), expected 429 lockout, got %d: %s", lockoutCfg.MaxAttempts, lastCode, rec.Code, rec.Body.String())
	}
}
