package api

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
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
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid JSON body"})
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
	acctID := ""
	if u.AccountID != "" {
		acctID = u.AccountID
	}
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
	locked, retryAfter, err := auth.CheckLockout(ctx, h.loginAttempts, key, auth.LockoutConfig{
		MaxAttempts:  3,
		Window:       15 * time.Minute,
		LockDuration: 5 * time.Minute,
	})
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
	if err := h.mailer.Send(ctx, u.Email, msg); err != nil {
		// Log but don't fail — the token still exists.
		h.writeAudit(ctx, u.AccountID, userID, "email.verification_send_failed", h.clientIP(r), r.UserAgent(), u.Email)
	}

	h.writeAudit(ctx, u.AccountID, userID, "email.verification_sent", h.clientIP(r), r.UserAgent(), u.Email)

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
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid JSON body"})
		return
	}

	newEmail := strings.ToLower(strings.TrimSpace(body.Email))
	if newEmail == "" || !strings.ContainsRune(newEmail, '@') {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid email"})
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
	if err := h.mailer.Send(ctx, newEmail, msg); err != nil {
		slog.Error("send verification email failed", "err", err)
	}

	ip := h.clientIP(r)
	h.writeAudit(ctx, u.AccountID, userID, "email.changed", ip, r.UserAgent(), u.Email+" → "+newEmail)
	h.writeAudit(ctx, u.AccountID, userID, "email.verification_sent", ip, r.UserAgent(), newEmail)

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

	email := strings.ToLower(strings.TrimSpace(body.Email))
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
	locked, _, lockErr := auth.CheckLockout(ctx, h.loginAttempts, key, auth.LockoutConfig{
		MaxAttempts:  3,
		Window:       15 * time.Minute,
		LockDuration: 5 * time.Minute,
	})
	if lockErr == nil && locked {
		_ = json.NewEncoder(w).Encode(map[string]string{"ok": "true"})
		return
	}

	token := auth.NewToken()
	hashedID := auth.HashToken(token)
	expiresAt := time.Now().UTC().Add(resetTokenTTL).Format(time.RFC3339)

	if err := h.authStore.CreateEmailToken(ctx, hashedID, u.ID, "reset", expiresAt); err != nil {
		h.writeErr(w, err)
		return
	}

	_ = h.authStore.RecordLoginAttempt(ctx, key, false)

	// If EMAIL_PROVIDER=none, the links will be logged by the none mailer.
	link := h.publicBaseURL + "/reset-password?token=" + token
	msg := mailer.PasswordResetEmail(link)
	if err := h.mailer.Send(ctx, email, msg); err != nil {
		slog.Error("send password reset email failed", "err", err)
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
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid JSON body"})
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

	if err := h.authStore.SetPasswordHash(ctx, userID, phc); err != nil {
		h.writeErr(w, err)
		return
	}

	// Revoke all sessions for this user — logout everywhere.
	_ = h.sessions.DeleteUserSessions(ctx, userID)

	ip := h.clientIP(r)
	u, _ := h.store.GetUser(ctx, userID)
	acctID := ""
	if u.AccountID != "" {
		acctID = u.AccountID
	}
	h.writeAudit(ctx, acctID, userID, "password.reset", ip, r.UserAgent(), "")

	w.WriteHeader(http.StatusNoContent)
}
