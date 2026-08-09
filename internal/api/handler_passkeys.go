package api

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/go-webauthn/webauthn/protocol"
	gowa "github.com/go-webauthn/webauthn/webauthn"

	"github.com/gsaraiva2109/dietdaemon/core/types"
	"github.com/gsaraiva2109/dietdaemon/internal/auth"
)

const webauthnCeremonyTTL = 5 * time.Minute

const (
	errMissingCeremonyCookie  = "missing ceremony cookie"
	errInvalidExpiredCeremony = "invalid or expired ceremony"
	// #nosec G101 — error-message text, not a credential value.
	errInvalidCredentialPrefix = "invalid credential: "
)

// ---------------------------------------------------------------------------
// Passkey management (auth required — h.wrap)
// ---------------------------------------------------------------------------

// GET /auth/passkeys
func (h *Handler) handleListPasskeys(w http.ResponseWriter, r *http.Request, userID string) {
	pks, err := h.authStore.ListWebAuthnCredentials(r.Context(), userID)
	if err != nil {
		h.writeErr(w, err)
		return
	}
	if pks == nil {
		pks = []types.Passkey{}
	}
	_ = json.NewEncoder(w).Encode(pks)
}

// POST /auth/passkeys/register/begin
func (h *Handler) handlePasskeyRegisterBegin(w http.ResponseWriter, r *http.Request, userID string) {
	ctx := r.Context()

	u, err := h.store.GetUser(ctx, userID)
	if err != nil {
		h.writeErr(w, err)
		return
	}

	handle, err := h.authStore.GetOrCreateWebAuthnHandle(ctx, userID)
	if err != nil {
		h.writeErr(w, err)
		return
	}
	u.WebAuthnHandle = handle

	creds, err := h.authStore.GetWebAuthnCredentialsRaw(ctx, userID)
	if err != nil {
		h.writeErr(w, err)
		return
	}

	wUser := auth.WebAuthnUser{User: u, Credentials: creds}
	creation, session, err := h.webauthn.BeginRegistration(wUser)
	if err != nil {
		h.writeErr(w, fmt.Errorf("webauthn begin registration: %w", err))
		return
	}

	sessionJSON, err := auth.MarshalSessionData(session)
	if err != nil {
		h.writeErr(w, err)
		return
	}

	ceremonyID := auth.NewToken()
	expiresAt := time.Now().UTC().Add(webauthnCeremonyTTL).Format(time.RFC3339)
	if err := h.authStore.CreateWebAuthnSession(ctx, ceremonyID, userID, sessionJSON, expiresAt); err != nil {
		h.writeErr(w, err)
		return
	}

	h.setWebAuthnCookie(w, ceremonyID)
	_ = json.NewEncoder(w).Encode(creation)
}

// POST /auth/passkeys/register/finish
func (h *Handler) handlePasskeyRegisterFinish(w http.ResponseWriter, r *http.Request, userID string) {
	ctx := r.Context()

	ceremonyID := h.readWebAuthnCookie(r)
	if ceremonyID == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": errMissingCeremonyCookie})
		return
	}

	// Parse wrapper body: {label, credential}
	var body struct {
		Label      string          `json:"label"`
		Credential json.RawMessage `json:"credential"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": errInvalidJSONBody})
		return
	}

	storedUserID, sessionJSON, err := h.authStore.ConsumeWebAuthnSession(ctx, ceremonyID)
	if err != nil {
		h.clearWebAuthnCookie(w)
		w.WriteHeader(http.StatusUnauthorized)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": errInvalidExpiredCeremony})
		return
	}
	// Verify the authenticated user matches the ceremony user.
	if storedUserID != userID {
		h.clearWebAuthnCookie(w)
		w.WriteHeader(http.StatusUnauthorized)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "ceremony user mismatch"})
		return
	}

	session, err := auth.UnmarshalSessionData(sessionJSON)
	if err != nil {
		h.clearWebAuthnCookie(w)
		h.writeErr(w, err)
		return
	}

	u, err := h.store.GetUser(ctx, userID)
	if err != nil {
		h.clearWebAuthnCookie(w)
		h.writeErr(w, err)
		return
	}

	creds, err := h.authStore.GetWebAuthnCredentialsRaw(ctx, userID)
	if err != nil {
		h.clearWebAuthnCookie(w)
		h.writeErr(w, err)
		return
	}

	wUser := auth.WebAuthnUser{User: u, Credentials: creds}

	// Parse the inner credential via go-webauthn.
	parsed, err := auth.ParseCredentialCreationResponse(body.Credential)
	if err != nil {
		h.clearWebAuthnCookie(w)
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": errInvalidCredentialPrefix + err.Error()})
		return
	}

	cred, err := h.webauthn.CreateCredential(wUser, *session, parsed)
	if err != nil {
		h.clearWebAuthnCookie(w)
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "credential verification failed"})
		return
	}

	credJSON, err := auth.MarshalCredential(cred)
	if err != nil {
		h.clearWebAuthnCookie(w)
		h.writeErr(w, err)
		return
	}

	credID := base64.RawURLEncoding.EncodeToString(cred.ID)
	label := strings.TrimSpace(body.Label)
	if label == "" {
		label = "Passkey"
	}
	now := time.Now().UTC().Format(time.RFC3339)

	if err := h.authStore.CreateWebAuthnCredential(ctx, credID, userID, label, credJSON, int(cred.Authenticator.SignCount), now); err != nil {
		h.clearWebAuthnCookie(w)
		h.writeErr(w, err)
		return
	}

	h.clearWebAuthnCookie(w)

	ip := h.clientIP(r)
	h.writeAudit(ctx, "", userID, "passkey.registered", ip, r.UserAgent(), credID)

	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(types.Passkey{
		ID:         credID,
		Label:      label,
		CreatedAt:  now,
		LastUsedAt: "",
	})
}

// PATCH /auth/passkeys/{id}
func (h *Handler) handleRenamePasskey(w http.ResponseWriter, r *http.Request, userID string) {
	id := r.PathValue("id")
	if id == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "passkey id required"})
		return
	}

	var body struct {
		Label string `json:"label"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": errInvalidJSONBody})
		return
	}
	if strings.TrimSpace(body.Label) == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "label is required"})
		return
	}

	if err := h.authStore.RenameWebAuthnCredential(r.Context(), userID, id, strings.TrimSpace(body.Label)); err != nil {
		h.writeErr(w, err)
		return
	}

	ip := h.clientIP(r)
	h.writeAudit(r.Context(), "", userID, "passkey.renamed", ip, r.UserAgent(), id)

	// Return the updated passkey.
	pks, _ := h.authStore.ListWebAuthnCredentials(r.Context(), userID)
	for _, pk := range pks {
		if pk.ID == id {
			_ = json.NewEncoder(w).Encode(pk)
			return
		}
	}
	w.WriteHeader(http.StatusNoContent)
}

// DELETE /auth/passkeys/{id}
func (h *Handler) handleDeletePasskey(w http.ResponseWriter, r *http.Request, userID string) {
	id := r.PathValue("id")
	if id == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "passkey id required"})
		return
	}

	if err := h.authStore.DeleteWebAuthnCredential(r.Context(), userID, id); err != nil {
		h.writeErr(w, err)
		return
	}

	ip := h.clientIP(r)
	h.writeAudit(r.Context(), "", userID, "passkey.deleted", ip, r.UserAgent(), id)
	w.WriteHeader(http.StatusNoContent)
}

// ---------------------------------------------------------------------------
// Passwordless passkey login (public — h.wrapPublic)
// ---------------------------------------------------------------------------

// POST /auth/passkeys/login/begin
func (h *Handler) handlePasskeyLoginBegin(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var body struct {
		Email string `json:"email"`
	}
	if err := decodeOptionalRequestJSON(r, &body); err != nil {
		writeValidationError(w, errInvalidJSONBody)
		return
	}
	email := normalizeEmail(body.Email)

	assertion, session, storeID := h.beginScopedPasskeyLogin(r, email)
	if assertion == nil {
		// Discoverable path (no email, unknown email, or scoped begin failed).
		var err error
		assertion, session, err = h.webauthn.BeginDiscoverableLogin()
		if err != nil {
			h.writeErr(w, fmt.Errorf("webauthn begin discoverable login: %w", err))
			return
		}
	}

	sessionJSON, err := auth.MarshalSessionData(session)
	if err != nil {
		h.writeErr(w, err)
		return
	}

	ceremonyID := auth.NewToken()
	expiresAt := time.Now().UTC().Add(webauthnCeremonyTTL).Format(time.RFC3339)
	if err := h.authStore.CreateWebAuthnSession(ctx, ceremonyID, storeID, sessionJSON, expiresAt); err != nil {
		h.writeErr(w, err)
		return
	}

	h.setWebAuthnCookie(w, ceremonyID)
	_ = json.NewEncoder(w).Encode(assertion)
}

// beginScopedPasskeyLogin starts a WebAuthn login ceremony scoped to a known
// user's registered credentials. It returns a nil assertion (and empty
// userID) when email is empty or the lookup/credential-fetch/BeginLogin step
// fails for any reason, so the caller can fall back to a discoverable login
// without distinguishing "unknown email" from "no passkeys" — that
// distinction would let an attacker enumerate registered emails.
func (h *Handler) beginScopedPasskeyLogin(r *http.Request, email string) (*protocol.CredentialAssertion, *gowa.SessionData, string) {
	if email == "" {
		return nil, nil, ""
	}
	ctx := r.Context()
	u, err := h.authStore.GetUserByEmail(ctx, email)
	if err != nil {
		return nil, nil, ""
	}
	creds, err := h.authStore.GetWebAuthnCredentialsRaw(ctx, u.ID)
	if err != nil || len(creds) == 0 {
		return nil, nil, ""
	}
	wUser := auth.WebAuthnUser{User: u, Credentials: creds}
	assertion, session, err := h.webauthn.BeginLogin(wUser)
	if err != nil {
		return nil, nil, ""
	}
	return assertion, session, u.ID
}

// POST /auth/passkeys/login/finish
func (h *Handler) handlePasskeyLoginFinish(w http.ResponseWriter, r *http.Request) {
	storedUserID, session, parsed, ok := h.resolvePasskeyLoginCeremony(w, r)
	if !ok {
		return
	}

	u, cred, ok := h.verifyPasskeyLogin(w, r, storedUserID, session, parsed)
	if !ok {
		return
	}

	h.completePasskeyLogin(w, r, u, cred)
}

// resolvePasskeyLoginCeremony reads the ceremony cookie, decodes the request
// body, consumes the pending WebAuthn session, and parses the credential
// assertion. On any failure it writes the appropriate error response itself
// and returns ok=false.
func (h *Handler) resolvePasskeyLoginCeremony(w http.ResponseWriter, r *http.Request) (storedUserID string, session *gowa.SessionData, parsed *protocol.ParsedCredentialAssertionData, ok bool) {
	ctx := r.Context()

	ceremonyID := h.readWebAuthnCookie(r)
	if ceremonyID == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": errMissingCeremonyCookie})
		return "", nil, nil, false
	}

	// Parse wrapper body: {credential}
	var body struct {
		Credential json.RawMessage `json:"credential"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": errInvalidJSONBody})
		return "", nil, nil, false
	}

	storedUserID, sessionJSON, err := h.authStore.ConsumeWebAuthnSession(ctx, ceremonyID)
	if err != nil {
		h.clearWebAuthnCookie(w)
		w.WriteHeader(http.StatusUnauthorized)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": errInvalidExpiredCeremony})
		return "", nil, nil, false
	}

	session, err = auth.UnmarshalSessionData(sessionJSON)
	if err != nil {
		h.clearWebAuthnCookie(w)
		h.writeErr(w, err)
		return "", nil, nil, false
	}

	parsed, err = auth.ParseCredentialRequestResponse(body.Credential)
	if err != nil {
		h.clearWebAuthnCookie(w)
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": errInvalidCredentialPrefix + err.Error()})
		return "", nil, nil, false
	}

	return storedUserID, session, parsed, true
}

// verifyPasskeyLogin validates the parsed assertion against either the known
// user's credentials (storedUserID set — scoped login) or, for discoverable
// login, resolves the user from the authenticator response. On failure it
// writes the error response (auditing a sign-count regression when
// detected) and returns ok=false.
func (h *Handler) verifyPasskeyLogin(w http.ResponseWriter, r *http.Request, storedUserID string, session *gowa.SessionData, parsed *protocol.ParsedCredentialAssertionData) (types.User, *gowa.Credential, bool) {
	ctx := r.Context()

	var cred *gowa.Credential
	var u types.User
	var err error

	if storedUserID != "" {
		// Scoped login — user known.
		u, err = h.store.GetUser(ctx, storedUserID)
		if err != nil {
			h.clearWebAuthnCookie(w)
			h.writeErr(w, err)
			return types.User{}, nil, false
		}
		creds, _ := h.authStore.GetWebAuthnCredentialsRaw(ctx, u.ID)
		wUser := auth.WebAuthnUser{User: u, Credentials: creds}
		cred, err = h.webauthn.ValidateLogin(wUser, *session, parsed)
	} else {
		// Discoverable login — resolve user from authenticator response.
		cred, err = h.webauthn.ValidateDiscoverableLogin(
			func(rawID, userHandle []byte) (gowa.User, error) {
				handle := base64.RawStdEncoding.EncodeToString(userHandle)
				resolved, resolveErr := h.authStore.GetUserByWebAuthnHandle(ctx, handle)
				if resolveErr != nil {
					return nil, resolveErr
				}
				u = resolved // capture for later use
				creds, _ := h.authStore.GetWebAuthnCredentialsRaw(ctx, u.ID)
				return auth.WebAuthnUser{User: u, Credentials: creds}, nil
			},
			*session, parsed,
		)
	}

	if err != nil {
		h.clearWebAuthnCookie(w)
		// Check for sign-count regression.
		if strings.Contains(err.Error(), "sign count") || strings.Contains(err.Error(), "counter") {
			h.writeAudit(ctx, "", storedUserID, "passkey.signcount_anomaly", h.clientIP(r), r.UserAgent(), "")
		}
		w.WriteHeader(http.StatusUnauthorized)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "passkey sign-in failed"})
		return types.User{}, nil, false
	}

	return u, cred, true
}

// completePasskeyLogin finalizes a verified passkey login: it clears the
// ceremony cookie, persists the updated credential, applies TOTP step-up if
// the user has it confirmed, and otherwise establishes the session.
func (h *Handler) completePasskeyLogin(w http.ResponseWriter, r *http.Request, u types.User, cred *gowa.Credential) {
	ctx := r.Context()
	ip := h.clientIP(r)
	ua := r.UserAgent()

	h.clearWebAuthnCookie(w)

	// Update credential after successful assertion.
	credJSON, _ := auth.MarshalCredential(cred)
	_ = h.authStore.UpdateWebAuthnCredentialOnAuth(ctx,
		base64.RawURLEncoding.EncodeToString(cred.ID),
		credJSON, int(cred.Authenticator.SignCount),
		time.Now().UTC().Format(time.RFC3339),
	)

	// TOTP step-up check: passkey proves possession, but TOTP policy still applies.
	if h.totp != nil {
		if confirmed, checkErr := h.totp.HasConfirmedTOTP(ctx, u.ID); checkErr == nil && confirmed {
			h.issuePasskeyMFAChallenge(w, ctx, u, ip, ua)
			return
		}
	}

	cookieTok, csrfTok, sess := auth.CreateSession(u.ID, false, ip, ua, h.sessionCfg)
	h.setSessionCookies(w, cookieTok, csrfTok, false)
	if err := h.sessions.CreateSession(ctx, sess); err != nil {
		h.writeErr(w, err)
		return
	}

	h.writeAudit(ctx, u.AccountID, u.ID, "passkey.login", ip, ua, "")
	_ = json.NewEncoder(w).Encode(sessionResponse{User: h.userToJSON(u)})
}

// issuePasskeyMFAChallenge creates and returns the TOTP step-up challenge
// that a passkey login must still pass when the user has TOTP confirmed.
func (h *Handler) issuePasskeyMFAChallenge(w http.ResponseWriter, ctx context.Context, u types.User, ip, ua string) {
	challengeTok := auth.NewToken()
	challengeID := auth.HashToken(challengeTok)
	expiresAt := time.Now().UTC().Add(5 * time.Minute)
	if err := h.mfaChallenges.CreateMFAChallenge(ctx, challengeID, u.ID, false, expiresAt.Format(time.RFC3339)); err != nil {
		h.writeErr(w, err)
		return
	}
	h.writeAudit(ctx, u.AccountID, u.ID, "mfa.challenge_issued", ip, ua, "passkey")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"mfa_required":    true,
		"challenge_token": challengeTok,
	})
}

// ---------------------------------------------------------------------------
// Passkey-as-2FA step-up (public — challenge-scoped)
// ---------------------------------------------------------------------------

// POST /auth/mfa/passkey/begin
func (h *Handler) handleMFAPasskeyBegin(w http.ResponseWriter, r *http.Request) {
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
	chUserID, _, ok := h.resolveMFAChallenge(w, ctx, challengeID, errInvalidChallenge, errInvalidChallenge, nil, nil)
	if !ok {
		return
	}

	u, err := h.store.GetUser(ctx, chUserID)
	if err != nil {
		h.writeErr(w, err)
		return
	}

	creds, err := h.authStore.GetWebAuthnCredentialsRaw(ctx, chUserID)
	if err != nil || len(creds) == 0 {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "no passkeys registered"})
		return
	}

	wUser := auth.WebAuthnUser{User: u, Credentials: creds}
	assertion, session, err := h.webauthn.BeginLogin(wUser)
	if err != nil {
		h.writeErr(w, fmt.Errorf("webauthn begin mfa login: %w", err))
		return
	}

	sessionJSON, err := auth.MarshalSessionData(session)
	if err != nil {
		h.writeErr(w, err)
		return
	}

	ceremonyID := auth.NewToken()
	ceremonyExpiresAt := time.Now().UTC().Add(webauthnCeremonyTTL).Format(time.RFC3339)
	if err := h.authStore.CreateWebAuthnSession(ctx, ceremonyID, chUserID, sessionJSON, ceremonyExpiresAt); err != nil {
		h.writeErr(w, err)
		return
	}

	h.setWebAuthnCookie(w, ceremonyID)
	_ = json.NewEncoder(w).Encode(assertion)
}

// POST /auth/mfa/passkey/finish
func (h *Handler) handleMFAPasskeyFinish(w http.ResponseWriter, r *http.Request) {
	ip := h.clientIP(r)
	ctx := r.Context()

	ceremonyID := h.readWebAuthnCookie(r)
	if ceremonyID == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": errMissingCeremonyCookie})
		return
	}

	var body struct {
		ChallengeToken string          `json:"challenge_token"`
		Credential     json.RawMessage `json:"credential"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": errInvalidJSONBody})
		return
	}
	if body.ChallengeToken == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "challenge_token is required"})
		return
	}

	mfaChallengeID := auth.HashToken(body.ChallengeToken)
	chUserID, remember, ok := h.resolveMFAChallenge(w, ctx, mfaChallengeID, errInvalidChallenge, errInvalidChallenge, func() { h.clearWebAuthnCookie(w) }, nil)
	if !ok {
		return
	}

	storedUserID, sessionJSON, err := h.authStore.ConsumeWebAuthnSession(ctx, ceremonyID)
	if err != nil {
		h.clearWebAuthnCookie(w)
		w.WriteHeader(http.StatusUnauthorized)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": errInvalidExpiredCeremony})
		return
	}
	if storedUserID != chUserID {
		h.clearWebAuthnCookie(w)
		w.WriteHeader(http.StatusUnauthorized)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "ceremony user mismatch"})
		return
	}

	session, err := auth.UnmarshalSessionData(sessionJSON)
	if err != nil {
		h.clearWebAuthnCookie(w)
		h.writeErr(w, err)
		return
	}

	u, err := h.store.GetUser(ctx, chUserID)
	if err != nil {
		h.clearWebAuthnCookie(w)
		h.writeErr(w, err)
		return
	}

	creds, _ := h.authStore.GetWebAuthnCredentialsRaw(ctx, chUserID)
	wUser := auth.WebAuthnUser{User: u, Credentials: creds}

	parsed, err := auth.ParseCredentialRequestResponse(body.Credential)
	if err != nil {
		h.clearWebAuthnCookie(w)
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": errInvalidCredentialPrefix + err.Error()})
		return
	}

	cred, err := h.webauthn.ValidateLogin(wUser, *session, parsed)
	if err != nil {
		h.clearWebAuthnCookie(w)
		w.WriteHeader(http.StatusUnauthorized)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "passkey verification failed"})
		return
	}

	h.clearWebAuthnCookie(w)

	// Verify the asserting credential belongs to the challenge's user.
	// (Already guaranteed by ValidateLogin + ceremony userID match above.)

	// Update credential after successful assertion.
	credJSON, _ := auth.MarshalCredential(cred)
	_ = h.authStore.UpdateWebAuthnCredentialOnAuth(ctx,
		base64.RawURLEncoding.EncodeToString(cred.ID),
		credJSON, int(cred.Authenticator.SignCount),
		time.Now().UTC().Format(time.RFC3339),
	)

	// Complete the MFA challenge — same as TOTP success.
	_ = h.mfaChallenges.DeleteMFAChallenge(ctx, mfaChallengeID)

	cookieTok, csrfTok, sess := auth.CreateSession(chUserID, remember, ip, r.UserAgent(), h.sessionCfg)
	h.setSessionCookies(w, cookieTok, csrfTok, remember)
	if err := h.sessions.CreateSession(ctx, sess); err != nil {
		h.writeErr(w, err)
		return
	}

	h.writeAudit(ctx, u.AccountID, chUserID, "mfa.success", ip, r.UserAgent(), "passkey")
	_ = json.NewEncoder(w).Encode(sessionResponse{User: h.userToJSON(u)})
}

// ---------------------------------------------------------------------------
// WebAuthn cookie helpers
// ---------------------------------------------------------------------------

func (h *Handler) setWebAuthnCookie(w http.ResponseWriter, value string) {
	secure := h.cookieSecure
	// #nosec G124 — Secure is config-driven; SameSite + HttpOnly are set.
	http.SetCookie(w, &http.Cookie{
		Name:     "dd_webauthn",
		Value:    value,
		Path:     "/",
		MaxAge:   int(webauthnCeremonyTTL.Seconds()),
		Secure:   secure,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})
}

func (h *Handler) clearWebAuthnCookie(w http.ResponseWriter) {
	// #nosec G124 — Secure is config-driven.
	http.SetCookie(w, &http.Cookie{
		Name:     "dd_webauthn",
		Path:     "/",
		MaxAge:   -1,
		Secure:   h.cookieSecure,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})
}

func (h *Handler) readWebAuthnCookie(r *http.Request) string {
	c, err := r.Cookie("dd_webauthn")
	if err != nil || c.Value == "" {
		return ""
	}
	return c.Value
}
