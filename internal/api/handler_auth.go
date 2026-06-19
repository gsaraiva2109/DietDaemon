package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gsaraiva2109/dietdaemon/core/types"
	"github.com/gsaraiva2109/dietdaemon/internal/auth"
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

func userToJSON(u types.User) userJSON {
	ev := false
	if u.EmailVerifiedAt != nil {
		ev = true
	}
	return userJSON{
		ID:            u.ID,
		Email:         u.Email,
		DisplayName:   u.DisplayName,
		EmailVerified: ev,
		CreatedAt:     u.CreatedAt.Format(time.RFC3339),
	}
}

// ---------------------------------------------------------------------------
// POST /auth/register
// ---------------------------------------------------------------------------

func (h *Handler) handleRegister(w http.ResponseWriter, r *http.Request) {
	ip := clientIP(r)
	if !h.ipLimiter.Allow(ip) {
		w.Header().Set("Retry-After", "30")
		w.WriteHeader(http.StatusTooManyRequests)
		json.NewEncoder(w).Encode(map[string]string{"error": "too many requests"})
		return
	}

	var body struct {
		Email       string `json:"email"`
		Password    string `json:"password"`
		DisplayName string `json:"display_name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "invalid JSON body"})
		return
	}

	email := strings.ToLower(strings.TrimSpace(body.Email))
	password := body.Password
	if email == "" || password == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "email and password are required"})
		return
	}

	ctx := r.Context()

	// Registration-mode gate.
	switch h.registrationMode {
	case types.RegistrationOIDCOnly:
		w.WriteHeader(http.StatusForbidden)
		json.NewEncoder(w).Encode(map[string]string{"error": auth.ErrRegistrationClosed.Error()})
		return
	case types.RegistrationInvite:
		count, err := h.authStore.CountUsers(ctx)
		if err != nil {
			h.writeErr(w, err)
			return
		}
		if count > 0 {
			w.WriteHeader(http.StatusForbidden)
			json.NewEncoder(w).Encode(map[string]string{"error": auth.ErrRegistrationClosed.Error()})
			return
		}
	}

	// Hash password (with length guards).
	phc, err := auth.Hash(password)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	// Reject duplicate email.
	if _, err := h.authStore.GetUserByEmail(ctx, email); err == nil {
		w.WriteHeader(http.StatusConflict)
		json.NewEncoder(w).Encode(map[string]string{"error": auth.ErrEmailTaken.Error()})
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

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(sessionResponse{User: userToJSON(u)})
}

// ---------------------------------------------------------------------------
// POST /auth/login
// ---------------------------------------------------------------------------

func (h *Handler) handleLogin(w http.ResponseWriter, r *http.Request) {
	ip := clientIP(r)

	// Per-IP rate limit.
	if !h.ipLimiter.Allow(ip) {
		w.Header().Set("Retry-After", "30")
		w.WriteHeader(http.StatusTooManyRequests)
		json.NewEncoder(w).Encode(map[string]string{"error": "too many requests"})
		return
	}

	var body struct {
		Email    string `json:"email"`
		Password string `json:"password"`
		Remember bool   `json:"remember"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "invalid JSON body"})
		return
	}

	email := strings.ToLower(strings.TrimSpace(body.Email))
	if email == "" || body.Password == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "email and password are required"})
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
		json.NewEncoder(w).Encode(map[string]string{"error": auth.ErrLocked.Error()})
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
	cookieTok, csrfTok, sess := auth.CreateSession(u.ID, body.Remember, ip, ua, h.sessionCfg)
	h.setSessionCookies(w, cookieTok, csrfTok, body.Remember)
	if err := h.sessions.CreateSession(ctx, sess); err != nil {
		h.writeErr(w, err)
		return
	}

	h.writeAudit(ctx, u.AccountID, u.ID, "user.login", ip, ua, "")

	json.NewEncoder(w).Encode(sessionResponse{User: userToJSON(u)})
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
	json.NewEncoder(w).Encode(sessionResponse{User: userToJSON(u)})
}

// ---------------------------------------------------------------------------
// GET /auth/providers  (public)
// ---------------------------------------------------------------------------

func (h *Handler) handleProviders(w http.ResponseWriter, r *http.Request) {
	json.NewEncoder(w).Encode(providersResponse{
		RegistrationMode: string(h.registrationMode),
		Providers:        []providerJSON{},
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
		json.NewEncoder(w).Encode(map[string]string{"error": "invalid JSON body"})
		return
	}

	if body.CurrentPassword == "" || body.NewPassword == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "current_password and new_password are required"})
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
		json.NewEncoder(w).Encode(map[string]string{"error": "current password is incorrect"})
		return
	}

	newPhc, err := auth.Hash(body.NewPassword)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
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
	json.NewEncoder(w).Encode(keys)
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
		json.NewEncoder(w).Encode(map[string]string{"error": "invalid JSON body"})
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
	json.NewEncoder(w).Encode(types.NewAPIKeyResponse{
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
	json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
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
