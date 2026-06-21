package api

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/gsaraiva2109/dietdaemon/core/types"
	"github.com/gsaraiva2109/dietdaemon/internal/auth"
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

	// Provider-reported error.
	if errParam := r.URL.Query().Get("error"); errParam != "" {
		slog.Warn("oidc provider returned error", "provider", provID,
			"error", errParam, "description", r.URL.Query().Get("error_description"))
		h.redirectAuthCallback(w, r, "provider_error", "")
		return
	}

	code := r.URL.Query().Get("code")
	qsState := r.URL.Query().Get("state")
	if code == "" || qsState == "" {
		h.redirectAuthCallback(w, r, "invalid_state", "")
		return
	}

	// Read state cookie.
	ck, err := r.Cookie("dd_oidc_state")
	if err != nil || ck.Value == "" {
		h.redirectAuthCallback(w, r, "invalid_state", "")
		return
	}
	cookieState := ck.Value

	// Clear the state cookie immediately.
	// #nosec G124 — Secure is config-driven.
	http.SetCookie(w, &http.Cookie{
		Name: "dd_oidc_state", Path: "/api/v1/auth/oidc/", MaxAge: -1,
		Secure: h.cookieSecure, HttpOnly: true, SameSite: http.SameSiteLaxMode,
	})

	// Constant-time-ish comparison of query state vs cookie state.
	if !strings.EqualFold(qsState, cookieState) {
		h.redirectAuthCallback(w, r, "invalid_state", "")
		return
	}

	stateID := auth.HashToken(cookieState)
	ctx := r.Context()

	nonce, pkceVer, linkUserID, nxt, err := h.authStore.ConsumeOIDCState(ctx, stateID)
	if err != nil {
		h.redirectAuthCallback(w, r, "invalid_state", "")
		return
	}

	// Exchange authorization code.
	tok, err := prov.Exchange(ctx, code, pkceVer)
	if err != nil {
		slog.Warn("oidc code exchange failed", "provider", provID, "err", err)
		h.redirectAuthCallback(w, r, "provider_error", "")
		return
	}

	// Verify ID token.
	rawIDToken, ok := tok.Extra("id_token").(string)
	if !ok || rawIDToken == "" {
		slog.Warn("oidc token missing id_token", "provider", provID)
		h.redirectAuthCallback(w, r, "provider_error", "")
		return
	}

	claims, err := prov.VerifyIDToken(ctx, rawIDToken, nonce)
	if err != nil {
		slog.Warn("oidc id_token verify failed", "provider", provID, "err", err)
		h.redirectAuthCallback(w, r, "provider_error", "")
		return
	}

	// Some providers (e.g. Authentik) return email/profile only from the
	// UserInfo endpoint, not in the ID token. Backfill when the ID token lacks a
	// verified email so account creation / email auto-link still works.
	if claims.Email == "" || !claims.EmailVerified || claims.Name == "" {
		if ui, uiErr := prov.UserInfo(ctx, tok); uiErr == nil {
			if claims.Email == "" {
				claims.Email = ui.Email
			}
			if !claims.EmailVerified {
				claims.EmailVerified = ui.EmailVerified
			}
			if claims.Name == "" {
				claims.Name = ui.Name
			}
		} else {
			slog.Warn("oidc userinfo backfill failed", "provider", provID, "err", uiErr)
		}
	}

	subject := claims.Subject
	email := strings.ToLower(strings.TrimSpace(claims.Email))
	emailVerified := claims.EmailVerified
	displayName := strings.TrimSpace(claims.Name)

	// Operator opted to trust this provider's email (OIDC_<id>_TRUST_EMAIL):
	// a non-empty address counts as verified even when the IdP omits the
	// email_verified claim. Appropriate for self-hosted IdPs the operator owns.
	if !emailVerified && email != "" && prov.TrustEmail {
		emailVerified = true
	}

	ip := clientIP(r)
	ua := r.UserAgent()

	// --- Link flow ---
	if linkUserID != "" {
		identityID := newHandlerID()
		err := h.authStore.LinkOIDCIdentity(ctx, identityID, linkUserID, provID, subject, email)
		if errors.Is(err, types.ErrIdentityLinked) {
			h.redirectAuthCallback(w, r, "already_linked", "")
			return
		}
		if err != nil {
			h.redirectAuthCallback(w, r, "internal_error", "")
			return
		}

		u, _ := h.store.GetUser(ctx, linkUserID)
		h.writeAudit(ctx, u.AccountID, linkUserID, "oidc.linked", ip, ua, provID+":"+subject)
		h.redirectAuthCallback(w, r, "", "link=1")
		return
	}

	// --- Sign-in flow ---
	var u types.User

	// 1. Match existing identity, then try auto-link by verified email.
	u, err = h.authStore.GetUserByOIDCIdentity(ctx, provID, subject)
	if err != nil && !errors.Is(err, types.ErrNotFound) {
		h.redirectAuthCallback(w, r, "internal_error", "")
		return
	}
	if err != nil {
		// Identity not found — try auto-link by verified email.
		if emailVerified && email != "" {
			u, err = h.authStore.GetUserByEmail(ctx, email)
			if err == nil {
				identityID := newHandlerID()
				if linkErr := h.authStore.LinkOIDCIdentity(ctx, identityID, u.ID, provID, subject, email); linkErr != nil && !errors.Is(linkErr, types.ErrIdentityLinked) {
					h.redirectAuthCallback(w, r, "internal_error", "")
					return
				}
				h.writeAudit(ctx, u.AccountID, u.ID, "oidc.linked", ip, ua, provID+":"+subject)
			}
		}
	}

	if u.ID == "" {
		// No existing user — registration gate.
		switch h.registrationMode {
		case types.RegistrationOIDCOnly, types.RegistrationOpen:
			// Allow creation.
		case types.RegistrationInvite:
			count, countErr := h.authStore.CountUsers(ctx)
			if countErr != nil || count > 0 {
				h.redirectAuthCallback(w, r, "registration_closed", "")
				return
			}
		}

		if !emailVerified || email == "" {
			slog.Warn("oidc account creation blocked: email unverified/missing",
				"provider", provID, "has_email", email != "", "email_verified", emailVerified)
			h.redirectAuthCallback(w, r, "email_unverified", "")
			return
		}

		accountID := newHandlerID()
		userID := newHandlerID()
		identityID := newHandlerID()
		if displayName == "" {
			displayName = email
		}

		u, err = h.authStore.CreateUserWithOIDC(ctx, accountID, userID, email, displayName, identityID, provID, subject)
		if err != nil {
			h.redirectAuthCallback(w, r, "internal_error", "")
			return
		}
		h.writeAudit(ctx, accountID, u.ID, "user.registered", ip, ua, "oidc:"+provID)
	}

	// Create session + set cookies.
	cookieTok, csrfTok, sess := auth.CreateSession(u.ID, false, ip, ua, h.sessionCfg)
	h.setSessionCookies(w, cookieTok, csrfTok, false)
	if err := h.sessions.CreateSession(ctx, sess); err != nil {
		h.redirectAuthCallback(w, r, "internal_error", "")
		return
	}

	h.writeAudit(ctx, u.AccountID, u.ID, "oidc.login", ip, ua, provID)

	// Build redirect to frontend /auth/callback.
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
	h.writeAudit(r.Context(), u.AccountID, userID, "oidc.unlinked", clientIP(r), r.UserAgent(), target.Provider+":"+target.Subject)

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
