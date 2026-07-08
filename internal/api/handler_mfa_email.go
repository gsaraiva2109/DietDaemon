package api

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"
	"time"

	"github.com/gsaraiva2109/dietdaemon/internal/auth"
	"github.com/gsaraiva2109/dietdaemon/internal/mailer"
)

const mfaEmailTTL = 10 * time.Minute
const mfaEmailMaxAttempts = 5

// ---------------------------------------------------------------------------
// Email OTP fallback for MFA step-up.
// ---------------------------------------------------------------------------

// POST /auth/mfa/email/send
func (h *Handler) handleMFAEmailSend(w http.ResponseWriter, r *http.Request) {
	ip := clientIP(r)
	ctx := r.Context()

	var body struct {
		ChallengeToken string `json:"challenge_token"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.ChallengeToken == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "challenge_token is required"})
		return
	}

	challengeID := auth.HashToken(body.ChallengeToken)
	chUserID, _, expiresAt, err := h.mfaChallenges.GetMFAChallenge(ctx, challengeID)
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid challenge"})
		return
	}

	exp, parseErr := time.Parse(time.RFC3339, expiresAt)
	if parseErr != nil || time.Now().UTC().After(exp) {
		_ = h.mfaChallenges.DeleteMFAChallenge(ctx, challengeID)
		w.WriteHeader(http.StatusUnauthorized)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid challenge"})
		return
	}

	u, err := h.store.GetUser(ctx, chUserID)
	if err != nil {
		h.writeErr(w, err)
		return
	}

	// Require verified email.
	if u.EmailVerifiedAt == nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "no verified email on file"})
		return
	}

	// Generate 6-digit code.
	code := generateMFAEmailCode()
	codeHash := auth.HashToken(code)
	codeExpiresAt := time.Now().UTC().Add(mfaEmailTTL).Format(time.RFC3339)

	if err := h.authStore.UpsertMFAEmailCode(ctx, chUserID, codeHash, codeExpiresAt); err != nil {
		h.writeErr(w, err)
		return
	}

	// Best-effort send.
	if h.mailer != nil && h.emailProvider != "none" {
		_ = h.mailer.Send(ctx, u.Email, mailer.MFAEmailCodeEmail(code))
	} else {
		// Log the code for dev/homelab (mirrors forgot-password pattern).
		h.logCode(ctx, u.Email, code)
	}

	h.writeAudit(ctx, u.AccountID, chUserID, "mfa.email_code_sent", ip, r.UserAgent(), "")
	_ = json.NewEncoder(w).Encode(map[string]string{"ok": "true"})
}

// POST /auth/mfa/email/verify
func (h *Handler) handleMFAEmailVerify(w http.ResponseWriter, r *http.Request) {
	ip := clientIP(r)
	ctx := r.Context()

	var body struct {
		ChallengeToken string `json:"challenge_token"`
		Code           string `json:"code"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid JSON body"})
		return
	}
	if body.ChallengeToken == "" || body.Code == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "challenge_token and code are required"})
		return
	}
	if !isSixDigit(body.Code) {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid code"})
		return
	}

	challengeID := auth.HashToken(body.ChallengeToken)
	chUserID, remember, chExpiresAt, err := h.mfaChallenges.GetMFAChallenge(ctx, challengeID)
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid challenge"})
		return
	}

	chExp, parseErr := time.Parse(time.RFC3339, chExpiresAt)
	if parseErr != nil || time.Now().UTC().After(chExp) {
		_ = h.mfaChallenges.DeleteMFAChallenge(ctx, challengeID)
		_ = h.authStore.DeleteMFAEmailCode(ctx, chUserID)
		w.WriteHeader(http.StatusUnauthorized)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid challenge"})
		return
	}

	storedHash, codeExpiresAt, attempts, err := h.authStore.GetMFAEmailCode(ctx, chUserID)
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "no code requested"})
		return
	}

	codeExp, parseErr := time.Parse(time.RFC3339, codeExpiresAt)
	if parseErr != nil || time.Now().UTC().After(codeExp) || attempts >= mfaEmailMaxAttempts {
		_ = h.authStore.DeleteMFAEmailCode(ctx, chUserID)
		_ = h.mfaChallenges.DeleteMFAChallenge(ctx, challengeID)
		w.WriteHeader(http.StatusUnauthorized)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid code"})
		return
	}

	// Constant-time compare.
	inputHash := auth.HashToken(body.Code)
	if subtle.ConstantTimeCompare([]byte(inputHash), []byte(storedHash)) != 1 {
		_ = h.authStore.IncrementMFAEmailCodeAttempts(ctx, chUserID)
		ua := r.UserAgent()
		h.writeAudit(ctx, "", chUserID, "mfa.fail", ip, ua, "bad email code")
		w.WriteHeader(http.StatusUnauthorized)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid code"})
		return
	}

	// Success — consume code and challenge.
	_ = h.authStore.DeleteMFAEmailCode(ctx, chUserID)
	_ = h.mfaChallenges.DeleteMFAChallenge(ctx, challengeID)

	cookieTok, csrfTok, sess := auth.CreateSession(chUserID, remember, ip, r.UserAgent(), h.sessionCfg)
	h.setSessionCookies(w, cookieTok, csrfTok, remember)
	if err := h.sessions.CreateSession(ctx, sess); err != nil {
		h.writeErr(w, err)
		return
	}

	u, err := h.store.GetUser(ctx, chUserID)
	if err != nil {
		h.writeErr(w, err)
		return
	}

	h.writeAudit(ctx, u.AccountID, chUserID, "mfa.success", ip, r.UserAgent(), "email")
	_ = json.NewEncoder(w).Encode(sessionResponse{User: h.userToJSON(u)})
}

// generateMFAEmailCode returns a 6-digit string from crypto/rand.
func generateMFAEmailCode() string {
	n, err := rand.Int(rand.Reader, big.NewInt(1_000_000))
	if err != nil {
		panic(fmt.Sprintf("api: crypto/rand.Int failed: %v", err))
	}
	return fmt.Sprintf("%06d", n.Int64())
}

// logCode writes the code to the server log. Only used when emailProvider == "none".
func (h *Handler) logCode(ctx context.Context, email, code string) {
	// Structured log so the operator can grep for it.
	fmt.Printf("[MFA-EMAIL-CODE] to=%s code=%s expires=%s\n", email, code, time.Now().UTC().Add(mfaEmailTTL).Format(time.RFC3339))
}
