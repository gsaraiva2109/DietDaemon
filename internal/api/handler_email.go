package api

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/mail"
	"time"

	"github.com/gsaraiva2109/dietdaemon/internal/auth"
	"github.com/gsaraiva2109/dietdaemon/internal/mailer"
)

// ---------------------------------------------------------------------------
// Email verification and password reset handlers.
// ---------------------------------------------------------------------------

const (
	verifyTokenTTL = 24 * time.Hour
	resetTokenTTL  = 1 * time.Hour

	errEmailInvalidJSONBody = "invalid JSON body"
)

// ---------------------------------------------------------------------------
// POST /auth/email/verify  (public)
// ---------------------------------------------------------------------------

func (h *Handler) handleEmailVerify(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Token string `json:"token"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": errEmailInvalidJSONBody})
		return
	}

	if body.Token == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "token is required"})
		return
	}

	ctx := r.Context()
	hashedID := auth.HashToken(body.Token)

	userID, err := h.authStore.ConsumeEmailToken(ctx, hashedID, "verify")
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid token"})
		return
	}

	if err := h.authStore.MarkEmailVerified(ctx, userID); err != nil {
		h.writeErr(w, err)
		return
	}

	ip := h.clientIP(r)
	u, _ := h.store.GetUser(ctx, userID)
	acctID := u.AccountID
	h.writeAudit(ctx, acctID, userID, "email.verified", ip, r.UserAgent(), "")

	w.WriteHeader(http.StatusNoContent)
}

// ---------------------------------------------------------------------------
// POST /auth/email/verify/resend  (session + CSRF)
// ---------------------------------------------------------------------------

func (h *Handler) handleResendVerify(w http.ResponseWriter, r *http.Request, userID string) {
	ctx := r.Context()
	u, err := h.store.GetUser(ctx, userID)
	if err != nil {
		h.writeErr(w, err)
		return
	}

	// No-op if already verified.
	if u.EmailVerifiedAt != nil {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	// Rate-limit resend per user (reuse lockout primitives).
	key := "resend:" + userID
	locked, retryAfter, err := auth.CheckLockout(ctx, h.loginAttempts, key, actionLockoutCfg)
	if err != nil {
		h.writeErr(w, err)
		return
	}
	if locked {
		_ = h.authStore.RecordLoginAttempt(ctx, key, false)
		w.Header().Set("Retry-After", fmt.Sprintf("%.0f", retryAfter.Seconds()))
		w.WriteHeader(http.StatusTooManyRequests)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "too many requests"})
		return
	}

	// Issue and send verification token.
	token := auth.NewToken()
	hashedID := auth.HashToken(token)
	expiresAt := time.Now().UTC().Add(verifyTokenTTL).Format(time.RFC3339)

	if err := h.authStore.CreateEmailToken(ctx, hashedID, userID, "verify", expiresAt); err != nil {
		h.writeErr(w, err)
		return
	}

	_ = h.authStore.RecordLoginAttempt(ctx, key, false)

	link := h.publicBaseURL + "/verify-email?token=" + token
	msg := mailer.VerificationEmail(link)
	// Best-effort — the token still exists even if the send fails, so this
	// never fails the request. auditedSend writes exactly one audit event
	// (never both "sent" and "send_failed" for the same call).
	_ = h.auditedSend(ctx, u.Email, msg, auditActor{AccountID: u.AccountID, UserID: userID, IP: h.clientIP(r), UA: r.UserAgent()}, "email.verification_sent", "email.verification_send_failed")

	w.WriteHeader(http.StatusNoContent)
}

// ---------------------------------------------------------------------------
// POST /auth/email/change  (session + CSRF)
// ---------------------------------------------------------------------------

func (h *Handler) handleEmailChange(w http.ResponseWriter, r *http.Request, userID string) {
	var body struct {
		Email           string `json:"email"`
		CurrentPassword string `json:"current_password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": errEmailInvalidJSONBody})
		return
	}

	newEmail := normalizeEmail(body.Email)
	parsedEmail, err := mail.ParseAddress(newEmail)
	if newEmail == "" || err != nil || parsedEmail.Address != newEmail {
		writeValidationError(w, "invalid email")
		return
	}

	if body.CurrentPassword == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "current_password is required"})
		return
	}

	ctx := r.Context()

	// Require re-authentication before allowing an email change (mirrors handleChangePassword).
	phc, err := h.authStore.GetPasswordHash(ctx, userID)
	if err != nil {
		h.writeErr(w, err)
		return
	}

	ok, err := auth.Verify(body.CurrentPassword, phc)
	if err != nil || !ok {
		w.WriteHeader(http.StatusUnauthorized)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "current password is incorrect"})
		return
	}

	// Check for conflict.
	if existing, err := h.authStore.GetUserByEmail(ctx, newEmail); err == nil {
		if existing.ID != userID {
			w.WriteHeader(http.StatusConflict)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "email in use"})
			return
		}
	}

	// Load current user for audit.
	u, err := h.store.GetUser(ctx, userID)
	if err != nil {
		h.writeErr(w, err)
		return
	}

	if err := h.authStore.UpdateUserEmail(ctx, userID, newEmail); err != nil {
		h.writeErr(w, err)
		return
	}

	// Issue verification token for the new address.
	token := auth.NewToken()
	hashedID := auth.HashToken(token)
	expiresAt := time.Now().UTC().Add(verifyTokenTTL).Format(time.RFC3339)

	if err := h.authStore.CreateEmailToken(ctx, hashedID, userID, "verify", expiresAt); err != nil {
		h.writeErr(w, err)
		return
	}

	link := h.publicBaseURL + "/verify-email?token=" + token
	msg := mailer.VerificationEmail(link)

	ip := h.clientIP(r)
	h.writeAudit(ctx, u.AccountID, userID, "email.changed", ip, r.UserAgent(), u.Email+" → "+newEmail)

	// The address has already changed; the user can't verify it without this
	// email, so a send failure must be surfaced rather than answered with a
	// false 204. auditedSend writes exactly one audit event either way.
	if err := h.auditedSend(ctx, newEmail, msg, auditActor{AccountID: u.AccountID, UserID: userID, IP: ip, UA: r.UserAgent()}, "email.verification_sent", "email.verification_send_failed"); err != nil {
		h.writeErr(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// ---------------------------------------------------------------------------
// POST /auth/password/forgot  (public)
// ---------------------------------------------------------------------------

func (h *Handler) handleForgotPassword(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Email string `json:"email"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		// Still return generic 200.
		_ = json.NewEncoder(w).Encode(map[string]string{"ok": "true"})
		return
	}

	email := normalizeEmail(body.Email)
	if email == "" {
		_ = json.NewEncoder(w).Encode(map[string]string{"ok": "true"})
		return
	}

	ctx := r.Context()

	// Generic response always — never reveal account existence.
	u, err := h.authStore.GetUserByEmail(ctx, email)
	if err != nil {
		_ = json.NewEncoder(w).Encode(map[string]string{"ok": "true"})
		return
	}

	// Only send reset if the account exists AND has a password (OIDC-only users have none).
	_, err = h.authStore.GetPasswordHash(ctx, u.ID)
	if err != nil {
		// No password — still generic.
		_ = json.NewEncoder(w).Encode(map[string]string{"ok": "true"})
		return
	}

	// Rate-limit per email too (reuse lockout primitives).
	key := "forgot:" + email
	locked, _, lockErr := auth.CheckLockout(ctx, h.loginAttempts, key, actionLockoutCfg)
	if lockErr != nil || locked {
		_ = json.NewEncoder(w).Encode(map[string]string{"ok": "true"})
		return
	}

	token := auth.NewToken()
	hashedID := auth.HashToken(token)
	expiresAt := time.Now().UTC().Add(resetTokenTTL).Format(time.RFC3339)

	if err := h.authStore.CreateEmailToken(ctx, hashedID, u.ID, "reset", expiresAt); err != nil {
		slog.Error("create password reset token failed", "err", err)
		_ = json.NewEncoder(w).Encode(map[string]string{"ok": "true"})
		return
	}

	_ = h.authStore.RecordLoginAttempt(ctx, key, false)

	// If EMAIL_PROVIDER=none, the links will be logged by the none mailer.
	link := h.publicBaseURL + "/reset-password?token=" + token
	msg := mailer.PasswordResetEmail(link)
	// A send failure here must not be reported as generic success (#268) --
	// the caller would otherwise believe a reset email is on the way when it
	// never left the server. This does narrow the anti-enumeration guarantee
	// above: while the mailer is broken, a request for a real account fails
	// visibly where a request for a nonexistent one still returns generic
	// "ok". That's an accepted trade-off (mailer outages are rare/operational,
	// not attacker-controlled) and is further bounded by the per-email
	// lockout already applied above.
	if err := h.auditedSend(ctx, email, msg, auditActor{AccountID: u.AccountID, UserID: u.ID, IP: h.clientIP(r), UA: r.UserAgent()}, "password.reset_email_sent", "password.reset_email_send_failed"); err != nil {
		h.writeErr(w, err)
		return
	}

	_ = json.NewEncoder(w).Encode(map[string]string{"ok": "true"})
}

// ---------------------------------------------------------------------------
// POST /auth/password/reset  (public)
// ---------------------------------------------------------------------------

func (h *Handler) handleResetPassword(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Token    string `json:"token"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": errEmailInvalidJSONBody})
		return
	}

	if body.Token == "" || body.Password == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "token and password are required"})
		return
	}

	// Validate password against existing policy.
	phc, err := auth.Hash(body.Password)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	ctx := r.Context()
	hashedID := auth.HashToken(body.Token)

	userID, err := h.authStore.ConsumeEmailToken(ctx, hashedID, "reset")
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid token"})
		return
	}

	// Revoke all sessions for this user — logout everywhere.
	if err := h.sessions.DeleteUserSessions(ctx, userID); err != nil {
		slog.Error("revoke sessions before password reset", "err", err)
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "internal server error"})
		return
	}

	if err := h.authStore.SetPasswordHash(ctx, userID, phc); err != nil {
		h.writeErr(w, err)
		return
	}

	ip := h.clientIP(r)
	u, _ := h.store.GetUser(ctx, userID)
	acctID := u.AccountID
	h.writeAudit(ctx, acctID, userID, "password.reset", ip, r.UserAgent(), "")

	w.WriteHeader(http.StatusNoContent)
}
