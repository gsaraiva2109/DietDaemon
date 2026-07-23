package api

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/json"
	"fmt"
	"log/slog"
	"math/big"
	"net/http"
	"strings"
	"time"

	"github.com/gsaraiva2109/dietdaemon/core/types"
	"github.com/gsaraiva2109/dietdaemon/internal/auth"
	"github.com/gsaraiva2109/dietdaemon/internal/mailer"
)

// ---------------------------------------------------------------------------
// Passwordless email sign-in with magic codes and links.
// ---------------------------------------------------------------------------

const magicTTL = 15 * time.Minute

// ---------------------------------------------------------------------------
// POST /auth/magic/request  (public)
// ---------------------------------------------------------------------------

func (h *Handler) handleMagicRequest(w http.ResponseWriter, r *http.Request) {
	ip := h.clientIP(r)

	var body struct {
		Email string `json:"email"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		// Generic response always.
		_ = json.NewEncoder(w).Encode(map[string]string{"ok": "true"})
		return
	}

	email := strings.ToLower(strings.TrimSpace(body.Email))
	if email == "" {
		_ = json.NewEncoder(w).Encode(map[string]string{"ok": "true"})
		return
	}

	ctx := r.Context()

	// Per-email lockout (mirrors resend at handler_email.go:89).
	key := "magic:" + email
	locked, _, lockErr := auth.CheckLockout(ctx, h.loginAttempts, key, auth.LockoutConfig{
		MaxAttempts:  3,
		Window:       15 * time.Minute,
		LockDuration: 5 * time.Minute,
	})
	if lockErr != nil || locked {
		_ = json.NewEncoder(w).Encode(map[string]string{"ok": "true"})
		return
	}

	// Look up user — generic response if unknown.
	u, err := h.authStore.GetUserByEmail(ctx, email)
	if err != nil {
		_ = json.NewEncoder(w).Encode(map[string]string{"ok": "true"})
		return
	}

	// oidc-only mode → no-op.
	if h.registrationMode == types.RegistrationOIDCOnly {
		_ = json.NewEncoder(w).Encode(map[string]string{"ok": "true"})
		return
	}

	expiresAt := time.Now().UTC().Add(magicTTL).Format(time.RFC3339)

	// Generate and persist magic link token (reuses auth_verification_codes).
	linkToken := auth.NewToken()
	linkHash := auth.HashToken(linkToken)
	if err := h.authStore.CreateEmailToken(ctx, linkHash, u.ID, "magic_link", expiresAt); err != nil {
		h.writeErr(w, err)
		return
	}

	// Generate and persist 6-digit code.
	code := generateMagicCode()
	codeHash := auth.HashToken(code)
	if err := h.authStore.UpsertMagicCode(ctx, u.ID, codeHash, expiresAt); err != nil {
		h.writeErr(w, err)
		return
	}

	link := h.publicBaseURL + "/magic?token=" + linkToken
	msg := mailer.MagicSigninEmail(link, code)
	if err := h.mailer.Send(ctx, u.Email, msg); err != nil {
		slog.Error("send magic signin email failed", "err", err)
		h.writeAudit(ctx, u.AccountID, u.ID, "user.magic_request_failed", ip, r.UserAgent(), "delivery failed")
		_ = json.NewEncoder(w).Encode(map[string]string{"ok": "true"})
		return
	}

	_ = h.authStore.RecordLoginAttempt(ctx, key, false)
	h.writeAudit(ctx, u.AccountID, u.ID, "user.magic_requested", ip, r.UserAgent(), u.Email)

	_ = json.NewEncoder(w).Encode(map[string]string{"ok": "true"})
}

// ---------------------------------------------------------------------------
// POST /auth/magic/verify  (public)
// ---------------------------------------------------------------------------

func (h *Handler) handleMagicVerify(w http.ResponseWriter, r *http.Request) {
	ip := h.clientIP(r)

	var body struct {
		Email string `json:"email"`
		Code  string `json:"code"`
		Token string `json:"token"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeMagicVerifyUnauthorized(w)
		return
	}

	ctx := r.Context()
	u, userID, ok := h.verifyMagicCredentials(w, r, body.Email, body.Code, body.Token)
	if !ok {
		return
	}

	ua := r.UserAgent()

	// TOTP step-up: if user has confirmed TOTP, issue MFA challenge instead of session.
	if h.totp != nil {
		if confirmed, err := h.totp.HasConfirmedTOTP(ctx, userID); err == nil && confirmed {
			challengeTok := auth.NewToken()
			challengeID := auth.HashToken(challengeTok)
			expiresAt := time.Now().UTC().Add(5 * time.Minute).Format(time.RFC3339)
			if err := h.mfaChallenges.CreateMFAChallenge(ctx, challengeID, userID, false, expiresAt); err != nil {
				h.writeErr(w, err)
				return
			}
			h.writeAudit(ctx, u.AccountID, userID, "mfa.challenge_issued", ip, ua, "")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"mfa_required":    true,
				"challenge_token": challengeTok,
			})
			return
		}
	}

	// Issue session.
	cookieTok, csrfTok, sess := auth.CreateSession(userID, false, ip, ua, h.sessionCfg)
	h.setSessionCookies(w, cookieTok, csrfTok, false)
	if err := h.sessions.CreateSession(ctx, sess); err != nil {
		h.writeErr(w, err)
		return
	}

	h.writeAudit(ctx, u.AccountID, userID, "user.magic_login", ip, ua, "")

	_ = json.NewEncoder(w).Encode(sessionResponse{User: h.userToJSON(u)})
}

func (h *Handler) verifyMagicCredentials(w http.ResponseWriter, r *http.Request, email, code, token string) (types.User, string, bool) {
	if token != "" {
		return h.verifyMagicToken(w, r, token)
	}
	return h.verifyMagicCode(w, r, email, code)
}

func (h *Handler) verifyMagicToken(w http.ResponseWriter, r *http.Request, token string) (types.User, string, bool) {
	ctx := r.Context()
	userID, err := h.authStore.ConsumeEmailToken(ctx, auth.HashToken(token), "magic_link")
	if err != nil {
		writeMagicVerifyUnauthorized(w)
		return types.User{}, "", false
	}
	u, err := h.store.GetUser(ctx, userID)
	if err != nil {
		h.writeErr(w, err)
		return types.User{}, "", false
	}
	_ = h.authStore.DeleteMagicCode(ctx, userID)
	return u, userID, true
}

func (h *Handler) verifyMagicCode(w http.ResponseWriter, r *http.Request, email, code string) (types.User, string, bool) {
	ctx := r.Context()
	email = strings.ToLower(strings.TrimSpace(email))
	if email == "" || code == "" {
		writeMagicVerifyUnauthorized(w)
		return types.User{}, "", false
	}

	key := "magicverify:" + email
	locked, retryAfter, lockErr := auth.CheckLockout(ctx, h.loginAttempts, key, h.lockoutCfg)
	if lockErr == nil && locked {
		_ = h.authStore.RecordLoginAttempt(ctx, key, false)
		w.Header().Set("Retry-After", fmt.Sprintf("%.0f", retryAfter.Seconds()))
		w.WriteHeader(http.StatusTooManyRequests)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "too many requests"})
		return types.User{}, "", false
	}

	u, err := h.authStore.GetUserByEmail(ctx, email)
	if err != nil {
		writeMagicVerifyUnauthorized(w)
		return types.User{}, "", false
	}

	codeHash, expiresAt, attempts, err := h.authStore.GetMagicCode(ctx, u.ID)
	if err != nil {
		_ = h.authStore.RecordLoginAttempt(ctx, key, false)
		writeMagicVerifyUnauthorized(w)
		return types.User{}, "", false
	}

	expires, err := time.Parse(time.RFC3339, expiresAt)
	if err != nil || time.Now().UTC().After(expires) || attempts >= 5 {
		_ = h.authStore.DeleteMagicCode(ctx, u.ID)
		_ = h.authStore.RecordLoginAttempt(ctx, key, false)
		writeMagicVerifyUnauthorized(w)
		return types.User{}, "", false
	}

	if subtle.ConstantTimeCompare([]byte(auth.HashToken(code)), []byte(codeHash)) != 1 {
		_ = h.authStore.IncrementMagicCodeAttempts(ctx, u.ID)
		_ = h.authStore.RecordLoginAttempt(ctx, key, false)
		writeMagicVerifyUnauthorized(w)
		return types.User{}, "", false
	}

	_ = h.authStore.DeleteMagicCode(ctx, u.ID)
	_ = h.authStore.DeleteEmailTokensByUserAndPurpose(ctx, u.ID, "magic_link")
	return u, u.ID, true
}

func writeMagicVerifyUnauthorized(w http.ResponseWriter) {
	w.WriteHeader(http.StatusUnauthorized)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid or expired code"})
}

// generateMagicCode returns a 6-digit code using crypto/rand.
func generateMagicCode() string {
	n, err := rand.Int(rand.Reader, big.NewInt(1_000_000))
	if err != nil {
		panic(fmt.Sprintf("api: crypto/rand.Int failed: %v", err))
	}
	return fmt.Sprintf("%06d", n.Int64())
}
