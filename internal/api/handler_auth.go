package api

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"net/netip"
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

// registrationAllowed reports whether a new account may be created.
// viaOIDC distinguishes OIDC auto-provisioning (where RegistrationOIDCOnly
// permits account creation) from password registration (where it never does).
func (h *Handler) registrationAllowed(ctx context.Context, viaOIDC bool) (bool, error) {
	if h.registrationMode == types.RegistrationOIDCOnly && !viaOIDC {
		return false, nil
	}
	if !h.multiUser {
		count, err := h.authStore.CountUsers(ctx)
		if err != nil {
			return false, err
		}
		return count == 0, nil
	}
	if h.registrationMode == types.RegistrationInvite {
		count, err := h.authStore.CountUsers(ctx)
		if err != nil {
			return false, err
		}
		return count == 0, nil
	}
	return true, nil
}

// ---------------------------------------------------------------------------
// POST /auth/register
// ---------------------------------------------------------------------------

func (h *Handler) handleRegister(w http.ResponseWriter, r *http.Request) {
	ip := h.clientIP(r)

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

	// Registration-mode / multi-user gate.
	allowed, err := h.registrationAllowed(ctx, false)
	if err != nil {
		h.writeErr(w, err)
		return
	}
	if !allowed {
		w.WriteHeader(http.StatusForbidden)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": auth.ErrRegistrationClosed.Error()})
		return
	}

	// Reject duplicate email before hashing — hashing is the expensive step
	// (argon2id, tuned to take tens of ms), so checking uniqueness first
	// avoids wasting CPU (a cheap DoS amplifier) and avoids leaking
	// account-existence timing through hash-then-reject.
	if _, err := h.authStore.GetUserByEmail(ctx, email); err == nil {
		w.WriteHeader(http.StatusConflict)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": auth.ErrEmailTaken.Error()})
		return
	} else if !errors.Is(err, types.ErrNotFound) {
		h.writeErr(w, err)
		return
	}

	// Hash password (with length guards).
	phc, err := auth.Hash(password)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
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
		if err := h.mailer.Send(ctx, u.Email, msg); err != nil {
			slog.Error("send verification email failed", "err", err)
		}
		h.writeAudit(ctx, accountID, u.ID, "email.verification_sent", ip, ua, u.Email)
	}

	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(sessionResponse{User: h.userToJSON(u)})
}

// ---------------------------------------------------------------------------
// POST /auth/login
// ---------------------------------------------------------------------------

func (h *Handler) handleLogin(w http.ResponseWriter, r *http.Request) {
	ip := h.clientIP(r)

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

	// Look up user by email. When the user doesn't exist (or has no password
	// hash, e.g. an OIDC-only account), fall through to verifying against a
	// dummy hash instead of returning early — an early return here would be
	// much faster than the hash-and-compare path below, letting an attacker
	// distinguish "no such account" from "wrong password" by timing.
	u, err := h.authStore.GetUserByEmail(ctx, email)
	phc := auth.DummyPHC
	userFound := err == nil
	if userFound {
		if realPHC, phcErr := h.authStore.GetPasswordHash(ctx, u.ID); phcErr == nil {
			phc = realPHC
		} else {
			userFound = false
		}
	}

	// Verify (always runs, even for a nonexistent user — see above).
	ok, err := auth.Verify(body.Password, phc)
	if !userFound || err != nil || !ok {
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

	ip := h.clientIP(r)
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
// API-key / share-token CRUD  (auth required)
//
// The two credential types are structurally identical trios (list, create,
// revoke) that differ only in secret generation, store method, audit event
// name, and response JSON shape. Each trio below plugs those differences
// into one shared handler body per verb.
// ---------------------------------------------------------------------------

// writeJSONList encodes items as JSON, substituting an empty slice for nil
// so the response is always `[]`, never `null`.
func writeJSONList[T any](w http.ResponseWriter, items []T) {
	if items == nil {
		items = []T{}
	}
	_ = json.NewEncoder(w).Encode(items)
}

// GET /auth/api-keys
func (h *Handler) handleListAPIKeys(w http.ResponseWriter, r *http.Request, userID string) {
	keys, err := h.authStore.ListAPIKeys(r.Context(), userID)
	if err != nil {
		h.writeErr(w, err)
		return
	}
	writeJSONList(w, keys)
}

// GET /auth/share-tokens
func (h *Handler) handleListShareTokens(w http.ResponseWriter, r *http.Request, userID string) {
	tokens, err := h.authStore.ListShareTokens(r.Context(), userID)
	if err != nil {
		h.writeErr(w, err)
		return
	}
	writeJSONList(w, tokens)
}

// credCreateConfig bundles the per-type pieces of a "create credential"
// request so handleCreateCred can serve both API keys and share tokens.
type credCreateConfig struct {
	genSecret  func() (raw, hashed string)
	store      func(ctx context.Context, id, userID, hashed, label string) error
	auditEvent string
	// response builds the created-credential JSON body from the new id,
	// label, creation time, and one-time raw secret.
	response func(id, userID, label string, createdAt time.Time, raw string) any
}

func (h *Handler) handleCreateCred(w http.ResponseWriter, r *http.Request, userID string, cfg credCreateConfig) {
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

	raw, hashed := cfg.genSecret()
	id := newHandlerID()

	if err := cfg.store(r.Context(), id, userID, hashed, body.Label); err != nil {
		h.writeErr(w, err)
		return
	}

	ip := h.clientIP(r)
	h.writeAudit(r.Context(), "", userID, cfg.auditEvent, ip, r.UserAgent(), id)

	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(cfg.response(id, userID, body.Label, time.Now().UTC(), raw))
}

// POST /auth/api-keys
func (h *Handler) handleCreateAPIKey(w http.ResponseWriter, r *http.Request, userID string) {
	h.handleCreateCred(w, r, userID, credCreateConfig{
		genSecret:  auth.NewAPIKey,
		store:      h.authStore.CreateAPIKey,
		auditEvent: "api_key.created",
		response: func(id, userID, label string, createdAt time.Time, raw string) any {
			return types.NewAPIKeyResponse{
				APIKey: types.APIKey{ID: id, UserID: userID, Label: label, CreatedAt: createdAt},
				Key:    raw,
			}
		},
	})
}

// POST /auth/share-tokens
func (h *Handler) handleCreateShareToken(w http.ResponseWriter, r *http.Request, userID string) {
	h.handleCreateCred(w, r, userID, credCreateConfig{
		genSecret: func() (string, string) {
			raw := auth.NewToken()
			return raw, auth.HashToken(raw)
		},
		store:      h.authStore.CreateShareToken,
		auditEvent: "share_token.created",
		response: func(id, userID, label string, createdAt time.Time, raw string) any {
			return types.NewShareTokenResponse{
				ShareToken: types.ShareToken{ID: id, UserID: userID, Label: label, CreatedAt: createdAt},
				Token:      raw,
			}
		},
	})
}

// credRevokeConfig bundles the per-type pieces of a "revoke credential"
// request. auditEvent == "" skips the audit write, preserving the existing
// (pre-refactor) asymmetry where API-key revocation isn't audited but
// share-token revocation is.
type credRevokeConfig struct {
	revoke     func(ctx context.Context, userID, id string) error
	auditEvent string
}

func (h *Handler) handleRevokeCred(w http.ResponseWriter, r *http.Request, userID string, cfg credRevokeConfig) {
	id := r.PathValue("id")
	if err := cfg.revoke(r.Context(), userID, id); err != nil {
		h.writeErr(w, err)
		return
	}
	if cfg.auditEvent != "" {
		ip := h.clientIP(r)
		h.writeAudit(r.Context(), "", userID, cfg.auditEvent, ip, r.UserAgent(), id)
	}
	w.WriteHeader(http.StatusNoContent)
}

// DELETE /auth/api-keys/{id}
func (h *Handler) handleRevokeAPIKey(w http.ResponseWriter, r *http.Request, userID string) {
	h.handleRevokeCred(w, r, userID, credRevokeConfig{revoke: h.authStore.RevokeAPIKey})
}

// DELETE /auth/share-tokens/{id}
func (h *Handler) handleRevokeShareToken(w http.ResponseWriter, r *http.Request, userID string) {
	h.handleRevokeCred(w, r, userID, credRevokeConfig{
		revoke:     h.authStore.RevokeShareToken,
		auditEvent: "share_token.revoked",
	})
}

// ---------------------------------------------------------------------------
// Cookie helpers
// ---------------------------------------------------------------------------

func (h *Handler) setSessionCookies(w http.ResponseWriter, token, csrf string, remember bool) {
	secure := h.cookieSecure
	sameSite := http.SameSiteLaxMode
	path := "/"
	// Empty Domain is a no-op for net/http (Cookie.String only emits the
	// Domain attribute when non-empty), so it's safe to always set this.
	domain := h.cookieDomain

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
		Domain:   domain,
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
		Domain:   domain,
		MaxAge:   maxAge,
		Secure:   secure,
		HttpOnly: false,
		SameSite: sameSite,
	})
}

func (h *Handler) clearSessionCookies(w http.ResponseWriter) {
	past := -1
	domain := h.cookieDomain
	// #nosec G124 — Secure is config-driven; SameSite + HttpOnly set as needed.
	http.SetCookie(w, &http.Cookie{Name: "dd_session", Path: "/", Domain: domain, MaxAge: past, Secure: h.cookieSecure, HttpOnly: true, SameSite: http.SameSiteLaxMode})
	// #nosec G124 — HttpOnly=false intentional; CSRF cookie must be JS-readable to clear properly.
	http.SetCookie(w, &http.Cookie{Name: "dd_csrf", Path: "/", Domain: domain, MaxAge: past, Secure: h.cookieSecure, SameSite: http.SameSiteLaxMode})
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

	ip := h.clientIP(r)
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

	ip := h.clientIP(r)
	ua := r.UserAgent()

	// Per-user lockout on TOTP/recovery-code guesses. The challenge endpoint
	// is only IP-rate-limited (wrapPublicLimited), which a distributed or
	// IP-rotating attacker can bypass; brute-forcing a 6-digit code (or a
	// recovery code) needs a cap keyed on the account being attacked too.
	lockKey := "totp:" + chUserID
	locked, retryAfter, err := auth.CheckLockout(ctx, h.loginAttempts, lockKey, h.lockoutCfg)
	if err != nil {
		h.writeErr(w, err)
		return
	}
	if locked {
		_ = h.loginAttempts.RecordLoginAttempt(ctx, lockKey, false)
		w.Header().Set("Retry-After", fmt.Sprintf("%.0f", retryAfter.Seconds()))
		w.WriteHeader(http.StatusTooManyRequests)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": auth.ErrLocked.Error()})
		return
	}

	// Try recovery code first if provided.
	if body.RecoveryCode != "" {
		codeHash := auth.HashToken(body.RecoveryCode)
		consumed, err := h.recoveryCodes.ConsumeRecoveryCode(ctx, chUserID, codeHash)
		if err != nil {
			h.writeErr(w, err)
			return
		}
		if !consumed {
			_ = h.loginAttempts.RecordLoginAttempt(ctx, lockKey, false)
			h.writeAudit(ctx, "", chUserID, "mfa.fail", ip, ua, "bad recovery code")
			w.WriteHeader(http.StatusUnauthorized)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid code"})
			return
		}
	} else {
		// Validate TOTP code.
		if !isSixDigit(body.Code) {
			_ = h.loginAttempts.RecordLoginAttempt(ctx, lockKey, false)
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
			_ = h.loginAttempts.RecordLoginAttempt(ctx, lockKey, false)
			h.writeAudit(ctx, "", chUserID, "mfa.fail", ip, ua, "bad totp code")
			w.WriteHeader(http.StatusUnauthorized)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid code"})
			return
		}
	}

	// Success — delete challenge, create session.
	_ = h.loginAttempts.RecordLoginAttempt(ctx, lockKey, true)
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

	ip := h.clientIP(r)
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

// clientIP resolves the request's client IP for rate limiting, lockout, and
// audit logging. X-Forwarded-For / X-Real-IP are only honored when the
// immediate peer (r.RemoteAddr) is a configured trusted proxy — otherwise
// any client could set those headers itself to spoof an arbitrary IP and
// dodge IP-based lockout/rate limiting entirely.
func (h *Handler) clientIP(r *http.Request) string {
	remoteHost := hostOnly(r.RemoteAddr)

	if h.isTrustedProxy(remoteHost) {
		if fwd := r.Header.Get("X-Forwarded-For"); fwd != "" {
			first := fwd
			if idx := strings.IndexByte(fwd, ','); idx > 0 {
				first = fwd[:idx]
			}
			if ip := strings.TrimSpace(first); ip != "" {
				return ip
			}
		}
		if real := strings.TrimSpace(r.Header.Get("X-Real-IP")); real != "" {
			return real
		}
	}

	return remoteHost
}

// isTrustedProxy reports whether addr (no port) is in the Handler's
// configured trusted-proxy allowlist (empty by default — see
// config.TrustedProxies).
func (h *Handler) isTrustedProxy(addr string) bool {
	if len(h.trustedProxies) == 0 {
		return false
	}
	ip, err := netip.ParseAddr(addr)
	if err != nil {
		return false
	}
	for _, p := range h.trustedProxies {
		if p.Contains(ip) {
			return true
		}
	}
	return false
}

// hostOnly strips the port from an address of the form "host:port" (as used
// by http.Request.RemoteAddr), including bracketed IPv6 addresses. Returns
// addr unchanged if it isn't a valid host:port pair.
func hostOnly(addr string) string {
	if host, _, err := net.SplitHostPort(addr); err == nil {
		return host
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
