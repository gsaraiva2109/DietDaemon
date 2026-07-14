package api

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gsaraiva2109/dietdaemon/core/types"
	"github.com/gsaraiva2109/dietdaemon/internal/auth"
	"github.com/gsaraiva2109/dietdaemon/internal/mailer"
)

// ---------------------------------------------------------------------------
// Auth endpoint handlers — register, login, logout, session, providers,
// change-password, and API-key CRUD.
// ---------------------------------------------------------------------------

// --- JSON shapes (frontend contract) ---

type sessionResponse struct {
	User userJSON `json:"user"`
}

type userJSON struct {
	ID            string `json:"id"`
	Email         string `json:"email"`
	DisplayName   string `json:"display_name"`
	EmailVerified bool   `json:"email_verified"`
	TOTPEnabled   bool   `json:"totp_enabled"`
	CreatedAt     string `json:"created_at"`
}

type providersResponse struct {
	RegistrationMode string         `json:"registration_mode"`
	Providers        []providerJSON `json:"providers"`
}

type providerJSON struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

func (h *Handler) userToJSON(u types.User) userJSON {
	ev := u.EmailVerifiedAt != nil
	var totp bool
	if h.totp != nil {
		totp, _ = h.totp.HasConfirmedTOTP(context.Background(), u.ID)
	}
	return userJSON{
		ID:            u.ID,
		Email:         u.Email,
		DisplayName:   u.DisplayName,
		EmailVerified: ev,
		TOTPEnabled:   totp,
		CreatedAt:     u.CreatedAt.Format(time.RFC3339),
	}
}

// ---------------------------------------------------------------------------
// POST /auth/register
// ---------------------------------------------------------------------------

func (h *Handler) handleRegister(w http.ResponseWriter, r *http.Request) {
	ip := clientIP(r)

	var body struct {
		Email       string `json:"email"`
		Password    string `json:"password"`
		DisplayName string `json:"display_name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid JSON body"})
		return
	}

	email := strings.ToLower(strings.TrimSpace(body.Email))
	password := body.Password
	if email == "" || password == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "email and password are required"})
		return
	}

	ctx := r.Context()

	// Registration-mode gate.
	switch h.registrationMode {
	case types.RegistrationOIDCOnly:
		w.WriteHeader(http.StatusForbidden)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": auth.ErrRegistrationClosed.Error()})
		return
	case types.RegistrationInvite:
		count, err := h.authStore.CountUsers(ctx)
		if err != nil {
			h.writeErr(w, err)
			return
		}
		if count > 0 {
			w.WriteHeader(http.StatusForbidden)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": auth.ErrRegistrationClosed.Error()})
			return
		}
	}

	// Hash password (with length guards).
	phc, err := auth.Hash(password)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	// Reject duplicate email.
	if _, err := h.authStore.GetUserByEmail(ctx, email); err == nil {
		w.WriteHeader(http.StatusConflict)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": auth.ErrEmailTaken.Error()})
		return
	} else if !errors.Is(err, types.ErrNotFound) {
		h.writeErr(w, err)
		return
	}

	accountID := newHandlerID()
	userID := newHandlerID()
	displayName := strings.TrimSpace(body.DisplayName)
	if displayName == "" {
		displayName = email
	}

	u, err := h.authStore.CreateUserWithPassword(ctx, accountID, userID, email, displayName, phc)
	if err != nil {
		h.writeErr(w, err)
		return
	}

	// Create session + set cookies.
	ua := r.UserAgent()
	cookieTok, csrfTok, sess := auth.CreateSession(u.ID, false, ip, ua, h.sessionCfg)
	h.setSessionCookies(w, cookieTok, csrfTok, false)
	if err := h.sessions.CreateSession(ctx, sess); err != nil {
		h.writeErr(w, err)
		return
	}

	h.writeAudit(ctx, accountID, u.ID, "user.registered", ip, ua, email)

	// Email verification: auto-verify when EMAIL_PROVIDER=none,
	// otherwise send verification email.
	if h.emailProvider == "" || h.emailProvider == "none" {
		// No mailer configured or explicitly "none" — auto-verify.
		_ = h.authStore.MarkEmailVerified(ctx, u.ID)
		u.EmailVerifiedAt = new(time.Now().UTC())
	} else {
		token := auth.NewToken()
		hashedID := auth.HashToken(token)
		expiresAt := time.Now().UTC().Add(24 * time.Hour).Format(time.RFC3339)
		if err := h.authStore.CreateEmailToken(ctx, hashedID, u.ID, "verify", expiresAt); err != nil {
			h.writeErr(w, err)
			return
		}
		link := h.publicBaseURL + "/verify-email?token=" + token
		msg := mailer.VerificationEmail(link)
		_ = h.mailer.Send(ctx, u.Email, msg)
		h.writeAudit(ctx, accountID, u.ID, "email.verification_sent", ip, ua, u.Email)
	}

	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(sessionResponse{User: h.userToJSON(u)})
}

// ---------------------------------------------------------------------------
// POST /auth/login
// ---------------------------------------------------------------------------

func (h *Handler) handleLogin(w http.ResponseWriter, r *http.Request) {
	ip := clientIP(r)

	var body struct {
		Email    string `json:"email"`
		Password string `json:"password"`
		Remember bool   `json:"remember"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid JSON body"})
		return
	}

	email := strings.ToLower(strings.TrimSpace(body.Email))
	if email == "" || body.Password == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "email and password are required"})
		return
	}

	ctx := r.Context()

	// Brute-force lockout on the email identifier.
	locked, retryAfter, err := auth.CheckLockout(ctx, h.loginAttempts, email, h.lockoutCfg)
	if err != nil {
		h.writeErr(w, err)
		return
	}
	if locked {
		_ = h.authStore.RecordLoginAttempt(ctx, email, false)
		w.Header().Set("Retry-After", fmt.Sprintf("%.0f", retryAfter.Seconds()))
		w.WriteHeader(http.StatusTooManyRequests)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": auth.ErrLocked.Error()})
		return
	}

	// Look up user by email.
	u, err := h.authStore.GetUserByEmail(ctx, email)
	if err != nil {
		_ = h.authStore.RecordLoginAttempt(ctx, email, false)
		h.writeAuthError(w, auth.ErrInvalidCredentials)
		return
	}

	// Get stored password hash.
	phc, err := h.authStore.GetPasswordHash(ctx, u.ID)
	if err != nil {
		_ = h.authStore.RecordLoginAttempt(ctx, email, false)
		h.writeAuthError(w, auth.ErrInvalidCredentials)
		return
	}

	// Verify.
	ok, err := auth.Verify(body.Password, phc)
	if err != nil || !ok {
		_ = h.authStore.RecordLoginAttempt(ctx, email, false)
		h.writeAuthError(w, auth.ErrInvalidCredentials)
		return
	}

	// Success.
	_ = h.authStore.RecordLoginAttempt(ctx, email, true)
	ua := r.UserAgent()

	// MFA step-up when TOTP is confirmed.
	if h.totp != nil {
		if confirmed, err := h.totp.HasConfirmedTOTP(ctx, u.ID); err == nil && confirmed {
			challengeTok := auth.NewToken()
			challengeID := auth.HashToken(challengeTok)
			expiresAt := time.Now().UTC().Add(5 * time.Minute)
			if err := h.mfaChallenges.CreateMFAChallenge(ctx, challengeID, u.ID, body.Remember, expiresAt.Format(time.RFC3339)); err != nil {
				h.writeErr(w, err)
				return
			}
			h.writeAudit(ctx, u.AccountID, u.ID, "mfa.challenge_issued", ip, ua, "")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"mfa_required":    true,
				"challenge_token": challengeTok,
			})
			return
		}
	}

	cookieTok, csrfTok, sess := auth.CreateSession(u.ID, body.Remember, ip, ua, h.sessionCfg)
	h.setSessionCookies(w, cookieTok, csrfTok, body.Remember)
	if err := h.sessions.CreateSession(ctx, sess); err != nil {
		h.writeErr(w, err)
		return
	}

	h.writeAudit(ctx, u.AccountID, u.ID, "user.login", ip, ua, "")

	_ = json.NewEncoder(w).Encode(sessionResponse{User: h.userToJSON(u)})
}

// ---------------------------------------------------------------------------
// POST /auth/logout  (auth required)
// ---------------------------------------------------------------------------

func (h *Handler) handleLogout(w http.ResponseWriter, r *http.Request, userID string) {
	if c, err := r.Cookie("dd_session"); err == nil && c.Value != "" {
		_ = h.sessions.DeleteSession(r.Context(), auth.HashToken(c.Value))
	}
	h.clearSessionCookies(w)
	w.WriteHeader(http.StatusNoContent)
}

// ---------------------------------------------------------------------------
// GET /auth/session  (auth required)
// ---------------------------------------------------------------------------

func (h *Handler) handleSession(w http.ResponseWriter, r *http.Request, userID string) {
	u, err := h.store.GetUser(r.Context(), userID)
	if err != nil {
		h.writeErr(w, err)
		return
	}
	_ = json.NewEncoder(w).Encode(sessionResponse{User: h.userToJSON(u)})
}

// ---------------------------------------------------------------------------
// GET /auth/providers  (public)
// ---------------------------------------------------------------------------

func (h *Handler) handleProviders(w http.ResponseWriter, r *http.Request) {
	var provs []providerJSON
	for _, p := range h.providers {
		provs = append(provs, providerJSON{ID: p.ID, Name: p.Name})
	}
	if provs == nil {
		provs = []providerJSON{}
	}
	_ = json.NewEncoder(w).Encode(providersResponse{
		RegistrationMode: string(h.registrationMode),
		Providers:        provs,
	})
}

// ---------------------------------------------------------------------------
// POST /auth/change-password  (auth required)
// ---------------------------------------------------------------------------

func (h *Handler) handleChangePassword(w http.ResponseWriter, r *http.Request, userID string) {
	var body struct {
		CurrentPassword string `json:"current_password"`
		NewPassword     string `json:"new_password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid JSON body"})
		return
	}

	if body.CurrentPassword == "" || body.NewPassword == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "current_password and new_password are required"})
		return
	}

	ctx := r.Context()

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

	newPhc, err := auth.Hash(body.NewPassword)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	if err := h.authStore.SetPasswordHash(ctx, userID, newPhc); err != nil {
		h.writeErr(w, err)
		return
	}

	// Invalidate all existing sessions (force re-login on other devices),
	// then create a fresh one for this device.
	_ = h.sessions.DeleteUserSessions(ctx, userID)

	ip := clientIP(r)
	ua := r.UserAgent()
	cookieTok, csrfTok, sess := auth.CreateSession(userID, false, ip, ua, h.sessionCfg)
	h.setSessionCookies(w, cookieTok, csrfTok, false)
	if err := h.sessions.CreateSession(ctx, sess); err != nil {
		h.writeErr(w, err)
		return
	}

	h.writeAudit(ctx, "", userID, "user.password_changed", ip, ua, "")

	w.WriteHeader(http.StatusNoContent)
}

// ---------------------------------------------------------------------------
// GET /auth/api-keys  (auth required)
// ---------------------------------------------------------------------------

func (h *Handler) handleListAPIKeys(w http.ResponseWriter, r *http.Request, userID string) {
	keys, err := h.authStore.ListAPIKeys(r.Context(), userID)
	if err != nil {
		h.writeErr(w, err)
		return
	}
	if keys == nil {
		keys = []types.APIKey{}
	}
	_ = json.NewEncoder(w).Encode(keys)
}

// ---------------------------------------------------------------------------
// POST /auth/api-keys  (auth required)
// ---------------------------------------------------------------------------

func (h *Handler) handleCreateAPIKey(w http.ResponseWriter, r *http.Request, userID string) {
	var body struct {
		Label string `json:"label"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid JSON body"})
		return
	}

	if strings.TrimSpace(body.Label) == "" {
		body.Label = "default"
	}

	raw, hashed := auth.NewAPIKey()
	keyID := newHandlerID()

	if err := h.authStore.CreateAPIKey(r.Context(), keyID, userID, hashed, body.Label); err != nil {
		h.writeErr(w, err)
		return
	}

	ip := clientIP(r)
	h.writeAudit(r.Context(), "", userID, "api_key.created", ip, r.UserAgent(), keyID)

	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(types.NewAPIKeyResponse{
		APIKey: types.APIKey{
			ID:        keyID,
			UserID:    userID,
			Label:     body.Label,
			CreatedAt: time.Now().UTC(),
		},
		Key: raw,
	})
}

// ---------------------------------------------------------------------------
// DELETE /auth/api-keys/{id}  (auth required)
// ---------------------------------------------------------------------------

func (h *Handler) handleRevokeAPIKey(w http.ResponseWriter, r *http.Request, userID string) {
	keyID := r.PathValue("id")
	if err := h.authStore.RevokeAPIKey(r.Context(), userID, keyID); err != nil {
		h.writeErr(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// ---------------------------------------------------------------------------
// GET /auth/share-tokens  (auth required)
// ---------------------------------------------------------------------------

func (h *Handler) handleListShareTokens(w http.ResponseWriter, r *http.Request, userID string) {
	tokens, err := h.authStore.ListShareTokens(r.Context(), userID)
	if err != nil {
		h.writeErr(w, err)
		return
	}
	if tokens == nil {
		tokens = []types.ShareToken{}
	}
	_ = json.NewEncoder(w).Encode(tokens)
}

// ---------------------------------------------------------------------------
// POST /auth/share-tokens  (auth required)
// ---------------------------------------------------------------------------

func (h *Handler) handleCreateShareToken(w http.ResponseWriter, r *http.Request, userID string) {
	var body struct {
		Label string `json:"label"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid JSON body"})
		return
	}

	if strings.TrimSpace(body.Label) == "" {
		body.Label = "default"
	}

	raw := auth.NewToken()
	hashed := auth.HashToken(raw)
	tokenID := newHandlerID()

	if err := h.authStore.CreateShareToken(r.Context(), tokenID, userID, hashed, body.Label); err != nil {
		h.writeErr(w, err)
		return
	}

	ip := clientIP(r)
	h.writeAudit(r.Context(), "", userID, "share_token.created", ip, r.UserAgent(), tokenID)

	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(types.NewShareTokenResponse{
		ShareToken: types.ShareToken{
			ID:        tokenID,
			UserID:    userID,
			Label:     body.Label,
			CreatedAt: time.Now().UTC(),
		},
		Token: raw,
	})
}

// ---------------------------------------------------------------------------
// DELETE /auth/share-tokens/{id}  (auth required)
// ---------------------------------------------------------------------------

func (h *Handler) handleRevokeShareToken(w http.ResponseWriter, r *http.Request, userID string) {
	tokenID := r.PathValue("id")
	if err := h.authStore.RevokeShareToken(r.Context(), userID, tokenID); err != nil {
		h.writeErr(w, err)
		return
	}
	ip := clientIP(r)
	h.writeAudit(r.Context(), "", userID, "share_token.revoked", ip, r.UserAgent(), tokenID)
	w.WriteHeader(http.StatusNoContent)
}

// ---------------------------------------------------------------------------
// Cookie helpers
// ---------------------------------------------------------------------------

func (h *Handler) setSessionCookies(w http.ResponseWriter, token, csrf string, remember bool) {
	secure := h.cookieSecure
	sameSite := http.SameSiteLaxMode
	path := "/"

	// dd_session — HttpOnly, readable only by the server.
	maxAge := 0 // session cookie
	if remember {
		maxAge = int(h.sessionCfg.RememberTTL.Seconds())
	}
	// #nosec G124 — Secure is config-driven; SameSite + HttpOnly are set.
	http.SetCookie(w, &http.Cookie{
		Name:     "dd_session",
		Value:    token,
		Path:     path,
		MaxAge:   maxAge,
		Secure:   secure,
		HttpOnly: true,
		SameSite: sameSite,
	})

	// dd_csrf — readable by JS, echoed in X-CSRF-Token on mutations.
	// HttpOnly=false is intentional (JS must read this cookie).
	// #nosec G124 — Secure is config-driven; HttpOnly=false is by design for CSRF.
	http.SetCookie(w, &http.Cookie{
		Name:     "dd_csrf",
		Value:    csrf,
		Path:     path,
		MaxAge:   maxAge,
		Secure:   secure,
		HttpOnly: false,
		SameSite: sameSite,
	})
}

func (h *Handler) clearSessionCookies(w http.ResponseWriter) {
	past := -1
	// #nosec G124 — Secure is config-driven; SameSite + HttpOnly set as needed.
	http.SetCookie(w, &http.Cookie{Name: "dd_session", Path: "/", MaxAge: past, Secure: h.cookieSecure, HttpOnly: true, SameSite: http.SameSiteLaxMode})
	// #nosec G124 — HttpOnly=false intentional; CSRF cookie must be JS-readable to clear properly.
	http.SetCookie(w, &http.Cookie{Name: "dd_csrf", Path: "/", MaxAge: past, Secure: h.cookieSecure, SameSite: http.SameSiteLaxMode})
}

func readSessionCookie(r *http.Request) string {
	c, err := r.Cookie("dd_session")
	if err != nil || c.Value == "" {
		return ""
	}
	return c.Value
}

// ---------------------------------------------------------------------------
// TOTP two-factor authentication handlers
// ---------------------------------------------------------------------------

// POST /auth/totp/enroll — begin enrollment, returns otpauth URL + secret.
func (h *Handler) handleTOTPEnroll(w http.ResponseWriter, r *http.Request, userID string) {
	if !h.totpReady(w) {
		return
	}

	ctx := r.Context()
	u, err := h.store.GetUser(ctx, userID)
	if err != nil {
		h.writeErr(w, err)
		return
	}

	secret, otpauthURL, err := auth.GenerateSecret(h.totpIssuer, u.Email)
	if err != nil {
		h.writeErr(w, err)
		return
	}

	// Encrypt the secret at rest.
	encSecret, err := auth.Encrypt([]byte(secret), h.totpEncKey)
	if err != nil {
		h.writeErr(w, err)
		return
	}

	if err := h.totp.UpsertTOTPSecret(ctx, userID, base64.RawStdEncoding.EncodeToString(encSecret)); err != nil {
		h.writeErr(w, err)
		return
	}

	_ = json.NewEncoder(w).Encode(map[string]string{
		"otpauth_url": otpauthURL,
		"secret":      secret,
	})
}

// POST /auth/totp/verify — confirm enrollment with a TOTP code, return recovery codes.
func (h *Handler) handleTOTPVerify(w http.ResponseWriter, r *http.Request, userID string) {
	if !h.totpReady(w) {
		return
	}

	var body struct {
		Code string `json:"code"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid JSON body"})
		return
	}

	if !isSixDigit(body.Code) {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid code"})
		return
	}

	ctx := r.Context()
	encSecret, confirmed, err := h.totp.GetTOTPSecret(ctx, userID)
	if err != nil {
		h.writeErr(w, err)
		return
	}
	if confirmed {
		w.WriteHeader(http.StatusConflict)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "totp already enabled"})
		return
	}

	ct, err := base64.RawStdEncoding.DecodeString(encSecret)
	if err != nil {
		h.writeErr(w, err)
		return
	}

	plain, err := auth.Decrypt(ct, h.totpEncKey)
	if err != nil {
		h.writeErr(w, err)
		return
	}

	if !auth.ValidateCode(string(plain), body.Code) {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid code"})
		return
	}

	if err := h.totp.ConfirmTOTP(ctx, userID); err != nil {
		h.writeErr(w, err)
		return
	}

	// Generate and persist recovery codes.
	codes, err := auth.GenerateRecoveryCodes(10)
	if err != nil {
		h.writeErr(w, err)
		return
	}

	hashes := make([]string, len(codes))
	for i, c := range codes {
		hashes[i] = auth.HashToken(c)
	}

	if err := h.recoveryCodes.ReplaceRecoveryCodes(ctx, userID, hashes); err != nil {
		h.writeErr(w, err)
		return
	}

	ip := clientIP(r)
	h.writeAudit(ctx, "", userID, "totp.enabled", ip, r.UserAgent(), "")

	_ = json.NewEncoder(w).Encode(map[string]any{
		"recovery_codes": codes,
	})
}

// POST /auth/totp/challenge — second login step. Accepts TOTP code or recovery code.
func (h *Handler) handleTOTPChallenge(w http.ResponseWriter, r *http.Request) {
	if !h.totpReady(w) {
		return
	}

	var body struct {
		ChallengeToken string `json:"challenge_token"`
		Code           string `json:"code"`
		RecoveryCode   string `json:"recovery_code"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid JSON body"})
		return
	}

	if body.ChallengeToken == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "challenge_token is required"})
		return
	}

	ctx := r.Context()
	challengeID := auth.HashToken(body.ChallengeToken)

	chUserID, remember, expiresAt, err := h.mfaChallenges.GetMFAChallenge(ctx, challengeID)
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid challenge"})
		return
	}

	// Check expiry.
	exp, err := time.Parse(time.RFC3339, expiresAt)
	if err != nil || time.Now().UTC().After(exp) {
		_ = h.mfaChallenges.DeleteMFAChallenge(ctx, challengeID)
		w.WriteHeader(http.StatusUnauthorized)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "challenge expired"})
		return
	}

	ip := clientIP(r)
	ua := r.UserAgent()

	// Try recovery code first if provided.
	if body.RecoveryCode != "" {
		codeHash := auth.HashToken(body.RecoveryCode)
		consumed, err := h.recoveryCodes.ConsumeRecoveryCode(ctx, chUserID, codeHash)
		if err != nil {
			h.writeErr(w, err)
			return
		}
		if !consumed {
			h.writeAudit(ctx, "", chUserID, "mfa.fail", ip, ua, "bad recovery code")
			w.WriteHeader(http.StatusUnauthorized)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid code"})
			return
		}
	} else {
		// Validate TOTP code.
		if !isSixDigit(body.Code) {
			h.writeAudit(ctx, "", chUserID, "mfa.fail", ip, ua, "bad code format")
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid code"})
			return
		}

		encSecret, _, err := h.totp.GetTOTPSecret(ctx, chUserID)
		if err != nil {
			h.writeErr(w, err)
			return
		}

		ct, err := base64.RawStdEncoding.DecodeString(encSecret)
		if err != nil {
			h.writeErr(w, err)
			return
		}

		plain, err := auth.Decrypt(ct, h.totpEncKey)
		if err != nil {
			h.writeErr(w, err)
			return
		}

		if !auth.ValidateCode(string(plain), body.Code) {
			h.writeAudit(ctx, "", chUserID, "mfa.fail", ip, ua, "bad totp code")
			w.WriteHeader(http.StatusUnauthorized)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid code"})
			return
		}
	}

	// Success — delete challenge, create session.
	_ = h.mfaChallenges.DeleteMFAChallenge(ctx, challengeID)

	cookieTok, csrfTok, sess := auth.CreateSession(chUserID, remember, ip, ua, h.sessionCfg)
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

	h.writeAudit(ctx, u.AccountID, chUserID, "mfa.success", ip, ua, "")

	_ = json.NewEncoder(w).Encode(sessionResponse{User: h.userToJSON(u)})
}

// DELETE /auth/totp — disable TOTP factor for the authenticated user.
func (h *Handler) handleTOTPDisable(w http.ResponseWriter, r *http.Request, userID string) {
	if !h.totpReady(w) {
		return
	}

	ctx := r.Context()
	if err := h.totp.DeleteTOTP(ctx, userID); err != nil {
		h.writeErr(w, err)
		return
	}

	// Also clean up recovery codes.
	_ = h.recoveryCodes.ReplaceRecoveryCodes(ctx, userID, nil)

	ip := clientIP(r)
	h.writeAudit(ctx, "", userID, "totp.disabled", ip, r.UserAgent(), "")

	w.WriteHeader(http.StatusNoContent)
}

// POST /auth/totp/recovery-codes/regenerate — replace all recovery codes.
func (h *Handler) handleRegenerateRecovery(w http.ResponseWriter, r *http.Request, userID string) {
	if !h.totpReady(w) {
		return
	}

	ctx := r.Context()
	confirmed, err := h.totp.HasConfirmedTOTP(ctx, userID)
	if err != nil {
		h.writeErr(w, err)
		return
	}
	if !confirmed {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "totp not enabled"})
		return
	}

	codes, err := auth.GenerateRecoveryCodes(10)
	if err != nil {
		h.writeErr(w, err)
		return
	}

	hashes := make([]string, len(codes))
	for i, c := range codes {
		hashes[i] = auth.HashToken(c)
	}

	if err := h.recoveryCodes.ReplaceRecoveryCodes(ctx, userID, hashes); err != nil {
		h.writeErr(w, err)
		return
	}

	_ = json.NewEncoder(w).Encode(map[string]any{
		"recovery_codes": codes,
	})
}

// totpReady checks that TOTP is configured, writing 501 if not.
func (h *Handler) totpReady(w http.ResponseWriter) bool {
	if h.totpEncKey == nil || h.totp == nil {
		w.WriteHeader(http.StatusNotImplemented)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "totp not configured"})
		return false
	}
	return true
}

func isSixDigit(s string) bool {
	if len(s) != 6 {
		return false
	}
	for _, c := range s {
		if c < '0' || c > '9' {
			return false
		}
	}
	return true
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func clientIP(r *http.Request) string {
	// Respect X-Forwarded-For when behind a trusted reverse proxy.
	if fwd := r.Header.Get("X-Forwarded-For"); fwd != "" {
		idx := strings.IndexByte(fwd, ',')
		if idx > 0 {
			return strings.TrimSpace(fwd[:idx])
		}
		return strings.TrimSpace(fwd)
	}
	// Strip port from RemoteAddr.
	addr := r.RemoteAddr
	if idx := strings.LastIndexByte(addr, ':'); idx > 0 {
		return addr[:idx]
	}
	return addr
}

func (h *Handler) writeAuthError(w http.ResponseWriter, err error) {
	w.WriteHeader(http.StatusUnauthorized)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
}

func (h *Handler) writeAudit(ctx context.Context, accountID, userID, event, ip, ua, meta string) {
	ev := types.AuditEvent{
		ID:        newHandlerID(),
		AccountID: accountID,
		UserID:    userID,
		Event:     event,
		IP:        ip,
		UserAgent: ua,
		Meta:      meta,
		CreatedAt: time.Now().UTC(),
	}
	// Best-effort; never fail a request over audit logging.
	_ = h.authStore.WriteAuditEvent(ctx, ev)
}
