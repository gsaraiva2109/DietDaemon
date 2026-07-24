package api

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/descope/virtualwebauthn"

	"github.com/gsaraiva2109/dietdaemon/core/types"
	"github.com/gsaraiva2109/dietdaemon/internal/auth"
)

// --- shared helpers -------------------------------------------------------

// newPasskeyRP mirrors the WebAuthnConfig newPasskeyHandler wires up, so
// virtualwebauthn's signed responses validate against h.webauthn.
func newPasskeyRP() virtualwebauthn.RelyingParty {
	return virtualwebauthn.RelyingParty{ID: "example.com", Name: "DietDaemon", Origin: "https://example.com"}
}

// seedPasskeyUser makes userID/email resolvable via the auth store (needed
// for scoped login-by-email and discoverable login-by-handle), matching the
// fixed user newPasskeyHandler wires into the meal store.
func seedPasskeyUser(authStore *passkeyTestStore, u types.User) {
	authStore.users[u.ID] = u
	authStore.userByEmail[u.Email] = u
}

// seedScopedUser registers testUser1 in the auth store's email/ID lookup
// tables using the same WebAuthn handle newPasskeyHandler already assigned
// to the meal-store user. BeginLogin's scoped path reads the handle from the
// auth-store user (via GetUserByEmail); the later ValidateLogin call reads
// it via a fresh h.store.GetUser. They must agree or go-webauthn rejects
// with "ID mismatch for User and Session". Call after newPasskeyHandler.
func seedScopedUser(authStore *passkeyTestStore) {
	u := testUser1
	u.WebAuthnHandle = authStore.handleByUser["user-1"]
	seedPasskeyUser(authStore, u)
}

// registerPasskey drives a full register/begin + register/finish ceremony
// through a virtual (software) authenticator that produces a real, correctly
// signed attestation, so h.webauthn.CreateCredential verifies it end-to-end.
// It returns the authenticator (holding the credential) and the credential
// itself so callers can reuse them for a later login ceremony.
func registerPasskey(t *testing.T, h *Handler, userID string) (virtualwebauthn.Authenticator, virtualwebauthn.Credential) {
	t.Helper()

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/passkeys/register/begin", nil)
	h.handlePasskeyRegisterBegin(rec, req, userID)
	if rec.Code != http.StatusOK {
		t.Fatalf("register begin: status = %d: %s", rec.Code, rec.Body.String())
	}
	cookies := rec.Result().Cookies()
	if len(cookies) != 1 {
		t.Fatalf("register begin: cookies = %+v", cookies)
	}

	attestationOptions, err := virtualwebauthn.ParseAttestationOptions(rec.Body.String())
	if err != nil {
		t.Fatalf("ParseAttestationOptions: %v", err)
	}

	rp := newPasskeyRP()
	authenticator := virtualwebauthn.NewAuthenticator()
	credential := virtualwebauthn.NewCredential(virtualwebauthn.KeyTypeEC2)
	attestationResponse := virtualwebauthn.CreateAttestationResponse(rp, authenticator, credential, *attestationOptions)
	authenticator.AddCredential(credential)

	finishBody, _ := json.Marshal(map[string]any{
		"label":      "My Key",
		"credential": json.RawMessage(attestationResponse),
	})
	rec2 := httptest.NewRecorder()
	req2 := httptest.NewRequest(http.MethodPost, "/api/v1/auth/passkeys/register/finish", bytes.NewReader(finishBody))
	req2.AddCookie(cookies[0])
	h.handlePasskeyRegisterFinish(rec2, req2, userID)
	if rec2.Code != http.StatusCreated {
		t.Fatalf("register finish: status = %d: %s", rec2.Code, rec2.Body.String())
	}

	return authenticator, credential
}

func doPasskeyLoginFinish(t *testing.T, h *Handler, cookie *http.Cookie, credential json.RawMessage) *httptest.ResponseRecorder {
	t.Helper()
	body, _ := json.Marshal(map[string]any{"credential": credential})
	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(body))
	req.AddCookie(cookie)
	rec := httptest.NewRecorder()
	h.handlePasskeyLoginFinish(rec, req)
	return rec
}

var testUser1 = types.User{ID: "user-1", AccountID: "account-1", Email: "user@example.com", DisplayName: "User"}

// --- handleListPasskeys ----------------------------------------------------

func TestHandleListPasskeys(t *testing.T) {
	authStore := newPasskeyTestStore()
	h := newPasskeyHandler(t, authStore)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/passkeys", nil)
	h.handleListPasskeys(rec, req, "user-1")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200: %s", rec.Code, rec.Body.String())
	}
	if got := strings.TrimSpace(rec.Body.String()); got != "[]" {
		t.Errorf("empty list body = %q, want []", got)
	}

	authStore.creds["cred-1"] = webauthnCred{userID: "user-1", label: "Phone", createdAt: "2026-01-01T00:00:00Z"}
	authStore.creds["cred-2"] = webauthnCred{userID: "other-user", label: "Not mine", createdAt: "2026-01-01T00:00:00Z"}

	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/api/v1/auth/passkeys", nil)
	h.handleListPasskeys(rec, req, "user-1")
	pks := decodeJSON[[]types.Passkey](t, rec)
	if len(pks) != 1 || pks[0].ID != "cred-1" || pks[0].Label != "Phone" {
		t.Errorf("passkeys = %+v, want one entry for cred-1", pks)
	}

	authStore.listCredsErr = errors.New("boom")
	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/api/v1/auth/passkeys", nil)
	h.handleListPasskeys(rec, req, "user-1")
	if rec.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want 500 on store error", rec.Code)
	}
}

// --- handleRenamePasskey ----------------------------------------------------

func TestHandleRenamePasskey(t *testing.T) {
	authStore := newPasskeyTestStore()
	h := newPasskeyHandler(t, authStore)
	authStore.creds["cred-1"] = webauthnCred{userID: "user-1", label: "Old label", createdAt: "2026-01-01T00:00:00Z"}

	t.Run("missing id", func(t *testing.T) { renamePasskeyMissingID(t, h) })
	t.Run("invalid JSON", func(t *testing.T) { renamePasskeyInvalidJSON(t, h) })
	t.Run("empty label", func(t *testing.T) { renamePasskeyEmptyLabel(t, h) })
	t.Run("store error", func(t *testing.T) { renamePasskeyStoreError(t, h, authStore) })
	t.Run("success", func(t *testing.T) { renamePasskeySuccess(t, h, authStore) })
}

func renamePasskeyMissingID(t *testing.T, h *Handler) {
	t.Helper()
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPatch, "/", nil)
	h.handleRenamePasskey(rec, req, "user-1")
	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", rec.Code)
	}
}

func renamePasskeyInvalidJSON(t *testing.T, h *Handler) {
	t.Helper()
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPatch, "/", strings.NewReader("{"))
	req.SetPathValue("id", "cred-1")
	h.handleRenamePasskey(rec, req, "user-1")
	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", rec.Code)
	}
}

func renamePasskeyEmptyLabel(t *testing.T, h *Handler) {
	t.Helper()
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPatch, "/", strings.NewReader(`{"label":"  "}`))
	req.SetPathValue("id", "cred-1")
	h.handleRenamePasskey(rec, req, "user-1")
	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", rec.Code)
	}
}

func renamePasskeyStoreError(t *testing.T, h *Handler, authStore *passkeyTestStore) {
	t.Helper()
	authStore.renameErr = errors.New("boom")
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPatch, "/", strings.NewReader(`{"label":"New"}`))
	req.SetPathValue("id", "cred-1")
	h.handleRenamePasskey(rec, req, "user-1")
	if rec.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want 500", rec.Code)
	}
	authStore.renameErr = nil
}

func renamePasskeySuccess(t *testing.T, h *Handler, authStore *passkeyTestStore) {
	t.Helper()
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPatch, "/", strings.NewReader(`{"label":"New label"}`))
	req.SetPathValue("id", "cred-1")
	h.handleRenamePasskey(rec, req, "user-1")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200: %s", rec.Code, rec.Body.String())
	}
	pk := decodeJSON[types.Passkey](t, rec)
	if pk.Label != "New label" {
		t.Errorf("label = %q, want %q", pk.Label, "New label")
	}
	found := false
	for _, ev := range authStore.auditEvents {
		if ev.Event == "passkey.renamed" {
			found = true
		}
	}
	if !found {
		t.Error("expected passkey.renamed audit event")
	}
}

// --- handleDeletePasskey ----------------------------------------------------

func TestHandleDeletePasskey(t *testing.T) {
	authStore := newPasskeyTestStore()
	h := newPasskeyHandler(t, authStore)
	authStore.creds["cred-1"] = webauthnCred{userID: "user-1", label: "Phone"}

	t.Run("missing id", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodDelete, "/", nil)
		h.handleDeletePasskey(rec, req, "user-1")
		if rec.Code != http.StatusBadRequest {
			t.Errorf("status = %d, want 400", rec.Code)
		}
	})

	t.Run("store error", func(t *testing.T) {
		authStore.deleteErr = errors.New("boom")
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodDelete, "/", nil)
		req.SetPathValue("id", "cred-1")
		h.handleDeletePasskey(rec, req, "user-1")
		if rec.Code != http.StatusInternalServerError {
			t.Errorf("status = %d, want 500", rec.Code)
		}
		authStore.deleteErr = nil
	})

	t.Run("success", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodDelete, "/", nil)
		req.SetPathValue("id", "cred-1")
		h.handleDeletePasskey(rec, req, "user-1")
		if rec.Code != http.StatusNoContent {
			t.Fatalf("status = %d, want 204: %s", rec.Code, rec.Body.String())
		}
		if _, ok := authStore.creds["cred-1"]; ok {
			t.Error("expected credential to be deleted")
		}
		found := false
		for _, ev := range authStore.auditEvents {
			if ev.Event == "passkey.deleted" {
				found = true
			}
		}
		if !found {
			t.Error("expected passkey.deleted audit event")
		}
	})
}

// --- handlePasskeyRegisterFinish --------------------------------------------

func TestHandlePasskeyRegisterFinishErrors(t *testing.T) {
	t.Run("missing cookie", func(t *testing.T) {
		h := newPasskeyHandler(t, newPasskeyTestStore())
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/", nil)
		h.handlePasskeyRegisterFinish(rec, req, "user-1")
		if rec.Code != http.StatusBadRequest {
			t.Errorf("status = %d, want 400", rec.Code)
		}
	})

	t.Run("invalid JSON body", func(t *testing.T) {
		h := newPasskeyHandler(t, newPasskeyTestStore())
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader("{"))
		req.AddCookie(&http.Cookie{Name: "dd_webauthn", Value: "any"})
		h.handlePasskeyRegisterFinish(rec, req, "user-1")
		if rec.Code != http.StatusBadRequest {
			t.Errorf("status = %d, want 400", rec.Code)
		}
	})

	t.Run("expired ceremony", func(t *testing.T) {
		h := newPasskeyHandler(t, newPasskeyTestStore())
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{}`))
		req.AddCookie(&http.Cookie{Name: "dd_webauthn", Value: "unknown"})
		h.handlePasskeyRegisterFinish(rec, req, "user-1")
		if rec.Code != http.StatusUnauthorized {
			t.Errorf("status = %d, want 401", rec.Code)
		}
	})

	t.Run("ceremony user mismatch", func(t *testing.T) {
		authStore := newPasskeyTestStore()
		h := newPasskeyHandler(t, authStore)
		authStore.ceremonies["cer-1"] = webauthnCeremony{userID: "someone-else", session: `{}`, expiresAt: "later"}
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{}`))
		req.AddCookie(&http.Cookie{Name: "dd_webauthn", Value: "cer-1"})
		h.handlePasskeyRegisterFinish(rec, req, "user-1")
		if rec.Code != http.StatusUnauthorized {
			t.Errorf("status = %d, want 401", rec.Code)
		}
		body := decodeJSON[map[string]string](t, rec)
		if body["error"] != "ceremony user mismatch" {
			t.Errorf("error = %q", body["error"])
		}
	})

	t.Run("malformed credential JSON", func(t *testing.T) {
		authStore := newPasskeyTestStore()
		h := newPasskeyHandler(t, authStore)

		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/passkeys/register/begin", nil)
		h.handlePasskeyRegisterBegin(rec, req, "user-1")
		cookie := rec.Result().Cookies()[0]

		body, _ := json.Marshal(map[string]any{"label": "x", "credential": json.RawMessage(`{"not":"valid"}`)})
		rec2 := httptest.NewRecorder()
		req2 := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(body))
		req2.AddCookie(cookie)
		h.handlePasskeyRegisterFinish(rec2, req2, "user-1")
		if rec2.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want 400: %s", rec2.Code, rec2.Body.String())
		}
	})
}

func TestHandlePasskeyRegisterFinishSuccess(t *testing.T) {
	authStore := newPasskeyTestStore()
	h := newPasskeyHandler(t, authStore)

	_, credential := registerPasskey(t, h, "user-1")

	if len(authStore.creds) != 1 {
		t.Fatalf("creds = %+v, want one stored credential", authStore.creds)
	}
	var stored webauthnCred
	for _, c := range authStore.creds {
		stored = c
	}
	if stored.userID != "user-1" || stored.label != "My Key" {
		t.Errorf("stored cred = %+v", stored)
	}
	if len(credential.ID) == 0 {
		t.Fatal("expected credential ID")
	}
	found := false
	for _, ev := range authStore.auditEvents {
		if ev.Event == "passkey.registered" {
			found = true
		}
	}
	if !found {
		t.Error("expected passkey.registered audit event")
	}
}

func TestHandlePasskeyRegisterBeginStoreErrors(t *testing.T) {
	cases := []struct {
		name string
		set  func(*passkeyTestStore)
	}{
		{"handle lookup error", func(s *passkeyTestStore) { s.getOrCreateHandleErr = errors.New("boom") }},
		{"creds lookup error", func(s *passkeyTestStore) { s.getCredsErr = errors.New("boom") }},
		{"session store error", func(s *passkeyTestStore) { s.createWebAuthnSessionErr = errors.New("boom") }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			authStore := newPasskeyTestStore()
			h := newPasskeyHandler(t, authStore)
			tc.set(authStore)
			rec := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodPost, "/", nil)
			h.handlePasskeyRegisterBegin(rec, req, "user-1")
			if rec.Code != http.StatusInternalServerError {
				t.Errorf("status = %d, want 500: %s", rec.Code, rec.Body.String())
			}
		})
	}
}

// --- handlePasskeyLoginBegin (scoped path) ----------------------------------

func TestHandlePasskeyLoginBeginScoped(t *testing.T) {
	authStore := newPasskeyTestStore()
	h := newPasskeyHandler(t, authStore)
	seedScopedUser(authStore)
	registerPasskey(t, h, "user-1")

	rec := httptest.NewRecorder()
	body, _ := json.Marshal(map[string]string{"email": "user@example.com"})
	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(body))
	h.handlePasskeyLoginBegin(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200: %s", rec.Code, rec.Body.String())
	}
	if authStore.createdCeremony.userID != "user-1" {
		t.Errorf("ceremony userID = %q, want user-1 (scoped)", authStore.createdCeremony.userID)
	}
}

// --- handlePasskeyLoginFinish ------------------------------------------------

func TestHandlePasskeyLoginFinishMalformedCredential(t *testing.T) {
	h := newPasskeyHandler(t, newPasskeyTestStore())
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/", nil)
	h.handlePasskeyLoginBegin(rec, req)
	cookie := rec.Result().Cookies()[0]

	finRec := doPasskeyLoginFinish(t, h, cookie, json.RawMessage(`{"not":"valid"}`))
	if finRec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400: %s", finRec.Code, finRec.Body.String())
	}
}

func TestHandlePasskeyLoginFinishValidationFailure(t *testing.T) {
	authStore := newPasskeyTestStore()
	h := newPasskeyHandler(t, authStore)
	seedScopedUser(authStore)
	registerPasskey(t, h, "user-1")

	rec := httptest.NewRecorder()
	beginBody, _ := json.Marshal(map[string]string{"email": "user@example.com"})
	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(beginBody))
	h.handlePasskeyLoginBegin(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("login begin: status = %d: %s", rec.Code, rec.Body.String())
	}
	cookie := rec.Result().Cookies()[0]

	assertionOptions, err := virtualwebauthn.ParseAssertionOptions(rec.Body.String())
	if err != nil {
		t.Fatalf("ParseAssertionOptions: %v", err)
	}
	rp := newPasskeyRP()
	// An unregistered authenticator/credential: the signature won't match
	// anything the server has stored for this user.
	rogueAuthenticator := virtualwebauthn.NewAuthenticator()
	rogueCredential := virtualwebauthn.NewCredential(virtualwebauthn.KeyTypeEC2)
	assertionResponse := virtualwebauthn.CreateAssertionResponse(rp, rogueAuthenticator, rogueCredential, *assertionOptions)

	finRec := doPasskeyLoginFinish(t, h, cookie, json.RawMessage(assertionResponse))
	if finRec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401: %s", finRec.Code, finRec.Body.String())
	}
	body := decodeJSON[map[string]string](t, finRec)
	if body["error"] != "passkey sign-in failed" {
		t.Errorf("error = %q", body["error"])
	}
}

func TestHandlePasskeyLoginFinishScopedSuccess(t *testing.T) {
	authStore := newPasskeyTestStore()
	h := newPasskeyHandler(t, authStore)
	seedScopedUser(authStore)
	authenticator, credential := registerPasskey(t, h, "user-1")

	rec := httptest.NewRecorder()
	beginBody, _ := json.Marshal(map[string]string{"email": "user@example.com"})
	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(beginBody))
	h.handlePasskeyLoginBegin(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("login begin: status = %d: %s", rec.Code, rec.Body.String())
	}
	cookie := rec.Result().Cookies()[0]

	assertionOptions, err := virtualwebauthn.ParseAssertionOptions(rec.Body.String())
	if err != nil {
		t.Fatalf("ParseAssertionOptions: %v", err)
	}
	rp := newPasskeyRP()
	assertionResponse := virtualwebauthn.CreateAssertionResponse(rp, authenticator, credential, *assertionOptions)

	finRec := doPasskeyLoginFinish(t, h, cookie, json.RawMessage(assertionResponse))
	if finRec.Code != http.StatusOK {
		t.Fatalf("login finish: status = %d: %s", finRec.Code, finRec.Body.String())
	}
	resp := decodeJSON[sessionResponse](t, finRec)
	if resp.User.ID != "user-1" {
		t.Errorf("session user = %+v, want user-1", resp.User)
	}
	if len(authStore.sessions) != 1 {
		t.Errorf("sessions = %d, want 1", len(authStore.sessions))
	}
	foundLogin := false
	for _, ev := range authStore.auditEvents {
		if ev.Event == "passkey.login" {
			foundLogin = true
		}
	}
	if !foundLogin {
		t.Error("expected passkey.login audit event")
	}
}

func TestHandlePasskeyLoginFinishDiscoverableSuccess(t *testing.T) {
	authStore := newPasskeyTestStore()
	h := newPasskeyHandler(t, authStore)
	seedScopedUser(authStore)
	authenticator, credential := registerPasskey(t, h, "user-1")

	handle := authStore.handleByUser["user-1"]
	rawHandle, err := base64.RawStdEncoding.DecodeString(handle)
	if err != nil {
		t.Fatalf("decode handle: %v", err)
	}
	authenticator.Options.UserHandle = rawHandle

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/", nil)
	h.handlePasskeyLoginBegin(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("login begin: status = %d: %s", rec.Code, rec.Body.String())
	}
	cookie := rec.Result().Cookies()[0]

	assertionOptions, err := virtualwebauthn.ParseAssertionOptions(rec.Body.String())
	if err != nil {
		t.Fatalf("ParseAssertionOptions: %v", err)
	}
	rp := newPasskeyRP()
	assertionResponse := virtualwebauthn.CreateAssertionResponse(rp, authenticator, credential, *assertionOptions)

	finRec := doPasskeyLoginFinish(t, h, cookie, json.RawMessage(assertionResponse))
	if finRec.Code != http.StatusOK {
		t.Fatalf("login finish: status = %d: %s", finRec.Code, finRec.Body.String())
	}
	resp := decodeJSON[sessionResponse](t, finRec)
	if resp.User.ID != "user-1" {
		t.Errorf("session user = %+v, want user-1", resp.User)
	}
}

func TestHandlePasskeyLoginFinishTOTPStepUp(t *testing.T) {
	authStore := newPasskeyTestStore()
	authStore.totpConfirmed = true
	h := newPasskeyHandler(t, authStore)
	seedScopedUser(authStore)
	authenticator, credential := registerPasskey(t, h, "user-1")

	rec := httptest.NewRecorder()
	beginBody, _ := json.Marshal(map[string]string{"email": "user@example.com"})
	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(beginBody))
	h.handlePasskeyLoginBegin(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("login begin: status = %d: %s", rec.Code, rec.Body.String())
	}
	cookie := rec.Result().Cookies()[0]

	assertionOptions, err := virtualwebauthn.ParseAssertionOptions(rec.Body.String())
	if err != nil {
		t.Fatalf("ParseAssertionOptions: %v", err)
	}
	rp := newPasskeyRP()
	assertionResponse := virtualwebauthn.CreateAssertionResponse(rp, authenticator, credential, *assertionOptions)

	finRec := doPasskeyLoginFinish(t, h, cookie, json.RawMessage(assertionResponse))
	if finRec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200: %s", finRec.Code, finRec.Body.String())
	}
	resp := decodeJSON[map[string]any](t, finRec)
	if mfaRequired, _ := resp["mfa_required"].(bool); !mfaRequired {
		t.Errorf("expected mfa_required=true, got %#v", resp)
	}
	if resp["challenge_token"] == "" || resp["challenge_token"] == nil {
		t.Error("expected a challenge_token")
	}
	if len(authStore.sessions) != 0 {
		t.Error("expected no session to be created when TOTP step-up is required")
	}
	if len(authStore.mfaChallenges) != 1 {
		t.Errorf("mfaChallenges = %d, want 1", len(authStore.mfaChallenges))
	}
}

// --- handleMFAPasskeyBegin ---------------------------------------------------

func TestHandleMFAPasskeyBegin(t *testing.T) {
	t.Run("missing challenge token", mfaPasskeyBeginMissingToken)
	t.Run("invalid JSON", mfaPasskeyBeginInvalidJSON)
	t.Run("unknown challenge", mfaPasskeyBeginUnknownChallenge)
	t.Run("expired challenge", mfaPasskeyBeginExpiredChallenge)
	t.Run("no passkeys registered", mfaPasskeyBeginNoPasskeysRegistered)
	t.Run("success", mfaPasskeyBeginSuccess)
}

func mfaPasskeyBeginMissingToken(t *testing.T) {
	t.Helper()
	h := newPasskeyHandler(t, newPasskeyTestStore())
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{}`))
	h.handleMFAPasskeyBegin(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", rec.Code)
	}
}

func mfaPasskeyBeginInvalidJSON(t *testing.T) {
	t.Helper()
	h := newPasskeyHandler(t, newPasskeyTestStore())
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{`))
	h.handleMFAPasskeyBegin(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", rec.Code)
	}
}

func mfaPasskeyBeginUnknownChallenge(t *testing.T) {
	t.Helper()
	h := newPasskeyHandler(t, newPasskeyTestStore())
	rec := httptest.NewRecorder()
	body, _ := json.Marshal(map[string]string{"challenge_token": "nope"})
	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(body))
	h.handleMFAPasskeyBegin(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", rec.Code)
	}
}

func mfaPasskeyBeginExpiredChallenge(t *testing.T) {
	t.Helper()
	authStore := newPasskeyTestStore()
	h := newPasskeyHandler(t, authStore)
	authStore.mfaChallenges[auth.HashToken("tok")] = mfaChallengeRec{
		userID:    "user-1",
		expiresAt: time.Now().UTC().Add(-time.Minute).Format(time.RFC3339),
	}
	rec := httptest.NewRecorder()
	body, _ := json.Marshal(map[string]string{"challenge_token": "tok"})
	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(body))
	h.handleMFAPasskeyBegin(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", rec.Code)
	}
	if _, ok := authStore.mfaChallenges[auth.HashToken("tok")]; ok {
		t.Error("expected expired challenge to be deleted")
	}
}

func mfaPasskeyBeginNoPasskeysRegistered(t *testing.T) {
	t.Helper()
	authStore := newPasskeyTestStore()
	h := newPasskeyHandler(t, authStore)
	authStore.mfaChallenges[auth.HashToken("tok")] = mfaChallengeRec{
		userID:    "user-1",
		expiresAt: time.Now().UTC().Add(time.Minute).Format(time.RFC3339),
	}
	rec := httptest.NewRecorder()
	body, _ := json.Marshal(map[string]string{"challenge_token": "tok"})
	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(body))
	h.handleMFAPasskeyBegin(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", rec.Code)
	}
}

func mfaPasskeyBeginSuccess(t *testing.T) {
	t.Helper()
	authStore := newPasskeyTestStore()
	h := newPasskeyHandler(t, authStore)
	registerPasskey(t, h, "user-1")
	authStore.mfaChallenges[auth.HashToken("tok")] = mfaChallengeRec{
		userID:    "user-1",
		expiresAt: time.Now().UTC().Add(time.Minute).Format(time.RFC3339),
	}
	rec := httptest.NewRecorder()
	body, _ := json.Marshal(map[string]string{"challenge_token": "tok"})
	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(body))
	h.handleMFAPasskeyBegin(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200: %s", rec.Code, rec.Body.String())
	}
	if authStore.createdCeremony.userID != "user-1" {
		t.Errorf("ceremony userID = %q", authStore.createdCeremony.userID)
	}
}

// --- handleMFAPasskeyFinish ---------------------------------------------------

func TestHandleMFAPasskeyFinishErrors(t *testing.T) {
	t.Run("missing cookie", mfaPasskeyFinishMissingCookie)
	t.Run("invalid JSON body", mfaPasskeyFinishInvalidJSONBody)
	t.Run("missing challenge token", mfaPasskeyFinishMissingChallengeToken)
	t.Run("unknown challenge", mfaPasskeyFinishUnknownChallenge)
	t.Run("expired challenge", mfaPasskeyFinishExpiredChallenge)
	t.Run("ceremony consume fails", mfaPasskeyFinishCeremonyConsumeFails)
	t.Run("ceremony user mismatch", mfaPasskeyFinishCeremonyUserMismatch)
	t.Run("malformed credential", mfaPasskeyFinishMalformedCredential)
	t.Run("validation failure", mfaPasskeyFinishValidationFailure)
}

func mfaPasskeyFinishMissingCookie(t *testing.T) {
	t.Helper()
	h := newPasskeyHandler(t, newPasskeyTestStore())
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/", nil)
	h.handleMFAPasskeyFinish(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", rec.Code)
	}
}

func mfaPasskeyFinishInvalidJSONBody(t *testing.T) {
	t.Helper()
	h := newPasskeyHandler(t, newPasskeyTestStore())
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader("{"))
	req.AddCookie(&http.Cookie{Name: "dd_webauthn", Value: "any"})
	h.handleMFAPasskeyFinish(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", rec.Code)
	}
}

func mfaPasskeyFinishMissingChallengeToken(t *testing.T) {
	t.Helper()
	h := newPasskeyHandler(t, newPasskeyTestStore())
	rec := httptest.NewRecorder()
	body, _ := json.Marshal(map[string]any{"credential": json.RawMessage(`{}`)})
	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(body))
	req.AddCookie(&http.Cookie{Name: "dd_webauthn", Value: "any"})
	h.handleMFAPasskeyFinish(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", rec.Code)
	}
}

func mfaPasskeyFinishUnknownChallenge(t *testing.T) {
	t.Helper()
	h := newPasskeyHandler(t, newPasskeyTestStore())
	rec := httptest.NewRecorder()
	body, _ := json.Marshal(map[string]any{"challenge_token": "nope", "credential": json.RawMessage(`{}`)})
	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(body))
	req.AddCookie(&http.Cookie{Name: "dd_webauthn", Value: "any"})
	h.handleMFAPasskeyFinish(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", rec.Code)
	}
}

func mfaPasskeyFinishExpiredChallenge(t *testing.T) {
	t.Helper()
	authStore := newPasskeyTestStore()
	h := newPasskeyHandler(t, authStore)
	authStore.mfaChallenges[auth.HashToken("tok")] = mfaChallengeRec{
		userID:    "user-1",
		expiresAt: time.Now().UTC().Add(-time.Minute).Format(time.RFC3339),
	}
	rec := httptest.NewRecorder()
	body, _ := json.Marshal(map[string]any{"challenge_token": "tok", "credential": json.RawMessage(`{}`)})
	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(body))
	req.AddCookie(&http.Cookie{Name: "dd_webauthn", Value: "any"})
	h.handleMFAPasskeyFinish(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", rec.Code)
	}
}

func mfaPasskeyFinishCeremonyConsumeFails(t *testing.T) {
	t.Helper()
	authStore := newPasskeyTestStore()
	h := newPasskeyHandler(t, authStore)
	authStore.mfaChallenges[auth.HashToken("tok")] = mfaChallengeRec{
		userID:    "user-1",
		expiresAt: time.Now().UTC().Add(time.Minute).Format(time.RFC3339),
	}
	rec := httptest.NewRecorder()
	body, _ := json.Marshal(map[string]any{"challenge_token": "tok", "credential": json.RawMessage(`{}`)})
	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(body))
	req.AddCookie(&http.Cookie{Name: "dd_webauthn", Value: "no-such-ceremony"})
	h.handleMFAPasskeyFinish(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", rec.Code)
	}
}

func mfaPasskeyFinishCeremonyUserMismatch(t *testing.T) {
	t.Helper()
	authStore := newPasskeyTestStore()
	h := newPasskeyHandler(t, authStore)
	authStore.mfaChallenges[auth.HashToken("tok")] = mfaChallengeRec{
		userID:    "user-1",
		expiresAt: time.Now().UTC().Add(time.Minute).Format(time.RFC3339),
	}
	authStore.ceremonies["cer-1"] = webauthnCeremony{userID: "someone-else", session: `{}`, expiresAt: "later"}
	rec := httptest.NewRecorder()
	body, _ := json.Marshal(map[string]any{"challenge_token": "tok", "credential": json.RawMessage(`{}`)})
	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(body))
	req.AddCookie(&http.Cookie{Name: "dd_webauthn", Value: "cer-1"})
	h.handleMFAPasskeyFinish(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", rec.Code)
	}
	body2 := decodeJSON[map[string]string](t, rec)
	if body2["error"] != "ceremony user mismatch" {
		t.Errorf("error = %q", body2["error"])
	}
}

func mfaPasskeyFinishMalformedCredential(t *testing.T) {
	t.Helper()
	authStore := newPasskeyTestStore()
	h := newPasskeyHandler(t, authStore)
	registerPasskey(t, h, "user-1")
	authStore.mfaChallenges[auth.HashToken("tok")] = mfaChallengeRec{
		userID:    "user-1",
		expiresAt: time.Now().UTC().Add(time.Minute).Format(time.RFC3339),
	}

	beginRec := httptest.NewRecorder()
	beginBody, _ := json.Marshal(map[string]string{"challenge_token": "tok"})
	beginReq := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(beginBody))
	h.handleMFAPasskeyBegin(beginRec, beginReq)
	if beginRec.Code != http.StatusOK {
		t.Fatalf("mfa begin: status = %d: %s", beginRec.Code, beginRec.Body.String())
	}
	cookie := beginRec.Result().Cookies()[0]

	finBody, _ := json.Marshal(map[string]any{"challenge_token": "tok", "credential": json.RawMessage(`{"not":"valid"}`)})
	finReq := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(finBody))
	finReq.AddCookie(cookie)
	finRec := httptest.NewRecorder()
	h.handleMFAPasskeyFinish(finRec, finReq)
	if finRec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400: %s", finRec.Code, finRec.Body.String())
	}
}

func mfaPasskeyFinishValidationFailure(t *testing.T) {
	t.Helper()
	authStore := newPasskeyTestStore()
	h := newPasskeyHandler(t, authStore)
	registerPasskey(t, h, "user-1")
	authStore.mfaChallenges[auth.HashToken("tok")] = mfaChallengeRec{
		userID:    "user-1",
		expiresAt: time.Now().UTC().Add(time.Minute).Format(time.RFC3339),
	}

	beginRec := httptest.NewRecorder()
	beginBody, _ := json.Marshal(map[string]string{"challenge_token": "tok"})
	beginReq := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(beginBody))
	h.handleMFAPasskeyBegin(beginRec, beginReq)
	if beginRec.Code != http.StatusOK {
		t.Fatalf("mfa begin: status = %d: %s", beginRec.Code, beginRec.Body.String())
	}
	cookie := beginRec.Result().Cookies()[0]

	assertionOptions, err := virtualwebauthn.ParseAssertionOptions(beginRec.Body.String())
	if err != nil {
		t.Fatalf("ParseAssertionOptions: %v", err)
	}
	rp := newPasskeyRP()
	rogueAuthenticator := virtualwebauthn.NewAuthenticator()
	rogueCredential := virtualwebauthn.NewCredential(virtualwebauthn.KeyTypeEC2)
	assertionResponse := virtualwebauthn.CreateAssertionResponse(rp, rogueAuthenticator, rogueCredential, *assertionOptions)

	finBody, _ := json.Marshal(map[string]any{"challenge_token": "tok", "credential": json.RawMessage(assertionResponse)})
	finReq := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(finBody))
	finReq.AddCookie(cookie)
	finRec := httptest.NewRecorder()
	h.handleMFAPasskeyFinish(finRec, finReq)
	if finRec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401: %s", finRec.Code, finRec.Body.String())
	}
}

func TestHandleMFAPasskeyFinishSuccess(t *testing.T) {
	authStore := newPasskeyTestStore()
	h := newPasskeyHandler(t, authStore)
	authenticator, credential := registerPasskey(t, h, "user-1")
	authStore.mfaChallenges[auth.HashToken("tok")] = mfaChallengeRec{
		userID:    "user-1",
		remember:  true,
		expiresAt: time.Now().UTC().Add(time.Minute).Format(time.RFC3339),
	}

	beginRec := httptest.NewRecorder()
	beginBody, _ := json.Marshal(map[string]string{"challenge_token": "tok"})
	beginReq := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(beginBody))
	h.handleMFAPasskeyBegin(beginRec, beginReq)
	if beginRec.Code != http.StatusOK {
		t.Fatalf("mfa begin: status = %d: %s", beginRec.Code, beginRec.Body.String())
	}
	cookie := beginRec.Result().Cookies()[0]

	assertionOptions, err := virtualwebauthn.ParseAssertionOptions(beginRec.Body.String())
	if err != nil {
		t.Fatalf("ParseAssertionOptions: %v", err)
	}
	rp := newPasskeyRP()
	assertionResponse := virtualwebauthn.CreateAssertionResponse(rp, authenticator, credential, *assertionOptions)

	finBody, _ := json.Marshal(map[string]any{"challenge_token": "tok", "credential": json.RawMessage(assertionResponse)})
	finReq := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(finBody))
	finReq.AddCookie(cookie)
	finRec := httptest.NewRecorder()
	h.handleMFAPasskeyFinish(finRec, finReq)
	if finRec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200: %s", finRec.Code, finRec.Body.String())
	}
	resp := decodeJSON[sessionResponse](t, finRec)
	if resp.User.ID != "user-1" {
		t.Errorf("session user = %+v, want user-1", resp.User)
	}
	if len(authStore.sessions) != 1 {
		t.Errorf("sessions = %d, want 1", len(authStore.sessions))
	}
	for _, sess := range authStore.sessions {
		if !sess.Remember {
			t.Error("expected remembered session")
		}
	}
	if _, ok := authStore.mfaChallenges[auth.HashToken("tok")]; ok {
		t.Error("expected challenge to be consumed")
	}
	found := false
	for _, ev := range authStore.auditEvents {
		if ev.Event == "mfa.success" {
			found = true
		}
	}
	if !found {
		t.Error("expected mfa.success audit event")
	}
}
