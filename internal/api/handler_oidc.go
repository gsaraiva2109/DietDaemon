package api

import (
	"context"
	"crypto/subtle"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/gsaraiva2109/dietdaemon/core/types"
	"github.com/gsaraiva2109/dietdaemon/internal/auth"
	"github.com/gsaraiva2109/dietdaemon/internal/oidc"
	"golang.org/x/oauth2"
)

// ---------------------------------------------------------------------------
// OIDC client login and account linking.
// ---------------------------------------------------------------------------

const oidcStateTTL = 10 * time.Minute

// --- GET /auth/oidc/{id}/start ---

func (h *Handler) handleOIDCStart(w http.ResponseWriter, r *http.Request) {
	provID := r.PathValue("id")
	prov := h.providers[provID]
	if prov == nil {
		h.redirectAuthCallback(w, r, "unknown_provider", "")
		return
	}

	link := r.URL.Query().Get("link") == "1"
	nxt := r.URL.Query().Get("next")

	// Link flow requires an authenticated session.
	var linkUserID string
	if link {
		uid, err := h.authenticate(r)
		if err != nil {
			h.redirectAuthCallback(w, r, "not_authenticated", "")
			return
		}
		linkUserID = uid
	}

	ctx := r.Context()

	// Generate state, nonce, and PKCE.
	state := auth.NewToken()
	nonce := auth.NewToken()
	pkceVer := oauth2.GenerateVerifier()

	stateID := auth.HashToken(state)
	expiresAt := time.Now().UTC().Add(oidcStateTTL).Format(time.RFC3339)

	if err := h.authStore.CreateOIDCState(ctx, stateID, nonce, pkceVer, linkUserID, nxt, expiresAt); err != nil {
		h.writeErr(w, err)
		return
	}

	// Short-lived HttpOnly state cookie for the callback to read.
	// #nosec G124 — Secure is config-driven; SameSite + HttpOnly are set.
	http.SetCookie(w, &http.Cookie{
		Name:     "dd_oidc_state",
		Value:    state,
		Path:     "/api/v1/auth/oidc/",
		MaxAge:   int(oidcStateTTL.Seconds()),
		Secure:   h.cookieSecure,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})

	authURL, err := prov.AuthCodeURL(ctx, state, nonce, pkceVer)
	if err != nil {
		// Lazy discovery failed; clean up the state row.
		_ = h.authStore.DeleteOIDCState(ctx, stateID)
		h.redirectAuthCallback(w, r, "provider_unavailable", "")
		return
	}

	http.Redirect(w, r, authURL, http.StatusFound)
}

// --- GET /auth/oidc/{id}/callback ---

func (h *Handler) handleOIDCCallback(w http.ResponseWriter, r *http.Request) {
	provID := r.PathValue("id")
	prov := h.providers[provID]
	if prov == nil {
		h.redirectAuthCallback(w, r, "unknown_provider", "")
		return
	}

	if h.oidcProviderReportedError(w, r, provID) {
		return
	}

	code, nonce, pkceVer, linkUserID, nxt, ok := h.oidcCallbackState(w, r)
	if !ok {
		return
	}

	callback := oidcCallbackContext{ctx: r.Context(), ip: h.clientIP(r), ua: r.UserAgent()}
	identity, ok := h.oidcIdentity(callback.ctx, prov, provID, code, pkceVer, nonce)
	if !ok {
		h.redirectAuthCallback(w, r, "provider_error", "")
		return
	}

	if linkUserID != "" {
		h.linkOIDCIdentity(w, r, callback, linkUserID, provID, identity)
		return
	}

	u, errCode := h.oidcUser(callback.ctx, provID, identity, callback.ip, callback.ua)
	if errCode != "" {
		h.redirectAuthCallback(w, r, errCode, "")
		return
	}
	h.finishOIDCLogin(w, r, callback, u, provID, nxt)
}

type oidcCallbackContext struct {
	ctx context.Context
	ip  string
	ua  string
}

type oidcIdentity struct {
	subject       string
	email         string
	emailVerified bool
	displayName   string
}

func (h *Handler) oidcProviderReportedError(w http.ResponseWriter, r *http.Request, provID string) bool {
	errParam := r.URL.Query().Get("error")
	if errParam == "" {
		return false
	}
	slog.Warn("oidc provider returned error", "provider", provID,
		"error", errParam, "description", r.URL.Query().Get("error_description"))
	h.redirectAuthCallback(w, r, "provider_error", "")
	return true
}

func (h *Handler) oidcCallbackState(w http.ResponseWriter, r *http.Request) (code, nonce, pkceVer, linkUserID, nxt string, ok bool) {
	code, queryState := r.URL.Query().Get("code"), r.URL.Query().Get("state")
	if code == "" || queryState == "" {
		h.redirectAuthCallback(w, r, "invalid_state", "")
		return "", "", "", "", "", false
	}

	cookie, err := r.Cookie("dd_oidc_state")
	if err != nil || cookie.Value == "" {
		h.redirectAuthCallback(w, r, "invalid_state", "")
		return "", "", "", "", "", false
	}

	// Clear the state cookie immediately.
	// #nosec G124 — Secure is config-driven.
	http.SetCookie(w, &http.Cookie{
		Name: "dd_oidc_state", Path: "/api/v1/auth/oidc/", MaxAge: -1,
		Secure: h.cookieSecure, HttpOnly: true, SameSite: http.SameSiteLaxMode,
	})
	if subtle.ConstantTimeCompare([]byte(queryState), []byte(cookie.Value)) != 1 {
		h.redirectAuthCallback(w, r, "invalid_state", "")
		return "", "", "", "", "", false
	}

	nonce, pkceVer, linkUserID, nxt, err = h.authStore.ConsumeOIDCState(r.Context(), auth.HashToken(cookie.Value))
	if err != nil {
		h.redirectAuthCallback(w, r, "invalid_state", "")
		return "", "", "", "", "", false
	}
	return code, nonce, pkceVer, linkUserID, nxt, true
}

func (h *Handler) oidcIdentity(ctx context.Context, prov *oidc.Provider, provID, code, pkceVer, nonce string) (oidcIdentity, bool) {
	tok, err := prov.Exchange(ctx, code, pkceVer)
	if err != nil {
		slog.Warn("oidc code exchange failed", "provider", provID, "err", err)
		return oidcIdentity{}, false
	}

	rawIDToken, ok := tok.Extra("id_token").(string)
	if !ok || rawIDToken == "" {
		slog.Warn("oidc token missing id_token", "provider", provID)
		return oidcIdentity{}, false
	}

	claims, err := prov.VerifyIDToken(ctx, rawIDToken, nonce)
	if err != nil {
		slog.Warn("oidc id_token verify failed", "provider", provID, "err", err)
		return oidcIdentity{}, false
	}
	if claims.Email == "" || !claims.EmailVerified || claims.Name == "" {
		h.backfillOIDCClaims(ctx, prov, provID, tok, &claims)
	}

	identity := oidcIdentity{
		subject:       claims.Subject,
		email:         strings.ToLower(strings.TrimSpace(claims.Email)),
		emailVerified: claims.EmailVerified,
		displayName:   strings.TrimSpace(claims.Name),
	}
	if !identity.emailVerified && identity.email != "" && prov.TrustEmail {
		identity.emailVerified = true
	}
	return identity, true
}

func (h *Handler) backfillOIDCClaims(ctx context.Context, prov *oidc.Provider, provID string, tok *oauth2.Token, claims *oidc.IDTokenClaims) {
	ui, err := prov.UserInfo(ctx, tok)
	if err != nil {
		slog.Warn("oidc userinfo backfill failed", "provider", provID, "err", err)
		return
	}
	if claims.Email == "" {
		claims.Email = ui.Email
	}
	if !claims.EmailVerified {
		claims.EmailVerified = ui.EmailVerified
	}
	if claims.Name == "" {
		claims.Name = ui.Name
	}
}

func (h *Handler) linkOIDCIdentity(w http.ResponseWriter, r *http.Request, callback oidcCallbackContext, userID, provID string, identity oidcIdentity) {
	err := h.authStore.LinkOIDCIdentity(callback.ctx, newHandlerID(), userID, provID, identity.subject, identity.email)
	if errors.Is(err, types.ErrIdentityLinked) {
		h.redirectAuthCallback(w, r, "already_linked", "")
		return
	}
	if err != nil {
		h.redirectAuthCallback(w, r, "internal_error", "")
		return
	}

	u, _ := h.store.GetUser(callback.ctx, userID)
	h.writeAudit(callback.ctx, u.AccountID, userID, "oidc.linked", callback.ip, callback.ua, provID+":"+identity.subject)
	h.redirectAuthCallback(w, r, "", "link=1")
}

func (h *Handler) oidcUser(ctx context.Context, provID string, identity oidcIdentity, ip, ua string) (types.User, string) {
	u, err := h.authStore.GetUserByOIDCIdentity(ctx, provID, identity.subject)
	if err == nil {
		return u, ""
	}
	if !errors.Is(err, types.ErrNotFound) {
		return types.User{}, "internal_error"
	}

	u, errCode := h.autoLinkOIDCIdentity(ctx, u, provID, identity, ip, ua)
	if errCode != "" || u.ID != "" {
		return u, errCode
	}
	return h.registerOIDCUser(ctx, provID, identity, ip, ua)
}

func (h *Handler) autoLinkOIDCIdentity(ctx context.Context, current types.User, provID string, identity oidcIdentity, ip, ua string) (types.User, string) {
	if !identity.emailVerified || identity.email == "" {
		return current, ""
	}

	u, err := h.authStore.GetUserByEmail(ctx, identity.email)
	if err != nil {
		return current, ""
	}
	if err := h.authStore.LinkOIDCIdentity(ctx, newHandlerID(), u.ID, provID, identity.subject, identity.email); err != nil && !errors.Is(err, types.ErrIdentityLinked) {
		return types.User{}, "internal_error"
	}
	h.writeAudit(ctx, u.AccountID, u.ID, "oidc.linked", ip, ua, provID+":"+identity.subject)
	return u, ""
}

func (h *Handler) registerOIDCUser(ctx context.Context, provID string, identity oidcIdentity, ip, ua string) (types.User, string) {
	allowed, err := h.registrationAllowed(ctx, true)
	if err != nil || !allowed {
		return types.User{}, "registration_closed"
	}
	if !identity.emailVerified || identity.email == "" {
		slog.Warn("oidc account creation blocked: email unverified/missing",
			"provider", provID, "has_email", identity.email != "", "email_verified", identity.emailVerified)
		return types.User{}, "email_unverified"
	}

	displayName := identity.displayName
	if displayName == "" {
		displayName = identity.email
	}
	accountID := newHandlerID()
	u, err := h.authStore.CreateUserWithOIDC(ctx, accountID, newHandlerID(), identity.email, displayName, newHandlerID(), provID, identity.subject)
	if err != nil {
		return types.User{}, "internal_error"
	}
	h.writeAudit(ctx, accountID, u.ID, "user.registered", ip, ua, "oidc:"+provID)
	return u, ""
}

func (h *Handler) finishOIDCLogin(w http.ResponseWriter, r *http.Request, callback oidcCallbackContext, u types.User, provID, nxt string) {
	cookieTok, csrfTok, sess := auth.CreateSession(u.ID, false, callback.ip, callback.ua, h.sessionCfg)
	h.setSessionCookies(w, cookieTok, csrfTok, false)
	if err := h.sessions.CreateSession(callback.ctx, sess); err != nil {
		h.redirectAuthCallback(w, r, "internal_error", "")
		return
	}
	h.writeAudit(callback.ctx, u.AccountID, u.ID, "oidc.login", callback.ip, callback.ua, provID)

	params := url.Values{}
	if nxt != "" {
		params.Set("next", nxt)
	}
	redir := "/auth/callback"
	if enc := params.Encode(); enc != "" {
		redir += "?" + enc
	}
	http.Redirect(w, r, redir, http.StatusFound)
}

// --- GET /auth/identities (auth required) ---

func (h *Handler) handleListIdentities(w http.ResponseWriter, r *http.Request, userID string) {
	identities, err := h.authStore.ListOIDCIdentities(r.Context(), userID)
	if err != nil {
		h.writeErr(w, err)
		return
	}
	if identities == nil {
		identities = []types.OIDCIdentity{}
	}
	_ = json.NewEncoder(w).Encode(identities)
}

// --- DELETE /auth/identities/{id} (auth + CSRF) ---

func (h *Handler) handleUnlinkIdentity(w http.ResponseWriter, r *http.Request, userID string) {
	identityID := r.PathValue("id")

	// Check if the identity exists before trying to delete.
	identities, err := h.authStore.ListOIDCIdentities(r.Context(), userID)
	if err != nil {
		h.writeErr(w, err)
		return
	}

	// Find the target identity.
	var target *types.OIDCIdentity
	for i := range identities {
		if identities[i].ID == identityID {
			target = &identities[i]
			break
		}
	}
	if target == nil {
		h.writeErr(w, types.ErrNotFound)
		return
	}

	// Guard against lockout: refuse if the user has no password AND this is
	// their only OIDC identity.
	_, pwdErr := h.authStore.GetPasswordHash(r.Context(), userID)
	noPassword := errors.Is(pwdErr, types.ErrNotFound)
	if noPassword && len(identities) == 1 {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "last_credential"})
		return
	}

	if err := h.authStore.DeleteOIDCIdentity(r.Context(), userID, identityID); err != nil {
		h.writeErr(w, err)
		return
	}

	u, _ := h.store.GetUser(r.Context(), userID)
	h.writeAudit(r.Context(), u.AccountID, userID, "oidc.unlinked", h.clientIP(r), r.UserAgent(), target.Provider+":"+target.Subject)

	w.WriteHeader(http.StatusNoContent)
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// redirectAuthCallback 302s the browser to the frontend /auth/callback route
// with optional error and extra params.
func (h *Handler) redirectAuthCallback(w http.ResponseWriter, r *http.Request, errCode, extra string) {
	params := url.Values{}
	if errCode != "" {
		// OIDC failures are otherwise silent server-side; log the reason so the
		// operator can see why a sign-in was rejected.
		slog.Warn("oidc auth callback rejected", "code", errCode, "path", r.URL.Path)
		params.Set("error", errCode)
	}
	if extra != "" {
		// extra is already encoded, e.g. "link=1" or "next=/dashboard"
		for _, part := range strings.Split(extra, "&") {
			k, v, ok := strings.Cut(part, "=")
			if ok {
				params.Set(k, v)
			}
		}
	}
	redir := "/auth/callback"
	if enc := params.Encode(); enc != "" {
		redir += "?" + enc
	}
	http.Redirect(w, r, redir, http.StatusFound)
}

// handleProviders (edit handler_auth.go:299) — populated from the OIDC registry
// instead of returning an empty array.
// Moved implementation below; the original placeholder in handler_auth.go is
// replaced by wiring that calls this method.

// ---------------------------------------------------------------------------
// PKCE helpers (re-exported from oauth2 for clarity)
// ---------------------------------------------------------------------------

// Verify that the oauth2 package provides what we expect at compile time.
var _ = oauth2.GenerateVerifier
var _ = oauth2.S256ChallengeFromVerifier
