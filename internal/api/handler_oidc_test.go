package api

import (
	"crypto/rand"
	"crypto/rsa"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	goidc "github.com/coreos/go-oidc/v3/oidc"
	"github.com/coreos/go-oidc/v3/oidc/oidctest"
	"github.com/gsaraiva2109/dietdaemon/core/types"
	"github.com/gsaraiva2109/dietdaemon/internal/oidc"
)

func TestHandleOIDCStartRejectsUnknownProvider(t *testing.T) {
	h := buildAuthSecurityHandler(newFakeAuthStore())
	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/oidc/unknown/start", nil)
	req.SetPathValue("id", "unknown")
	rec := httptest.NewRecorder()

	h.handleOIDCStart(rec, req)

	if rec.Code != http.StatusFound {
		t.Fatalf("status = %d, want 302", rec.Code)
	}
	if got := rec.Header().Get("Location"); got != "/auth/callback?error=unknown_provider" {
		t.Fatalf("Location = %q, want unknown_provider callback", got)
	}
}

func TestHandleOIDCCallbackRejectsUnknownProviderAndInvalidState(t *testing.T) {
	h := buildAuthSecurityHandler(newFakeAuthStore())
	h.providers["known"] = &oidc.Provider{ID: "known"}
	for name, tc := range map[string]struct {
		path string
		want string
	}{
		"unknown provider": {"/api/v1/auth/oidc/unknown/callback", "unknown_provider"},
		"missing state":    {"/api/v1/auth/oidc/known/callback?code=code", "invalid_state"},
	} {
		t.Run(name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tc.path, nil)
			if name == "unknown provider" {
				req.SetPathValue("id", "unknown")
			} else {
				req.SetPathValue("id", "known")
			}
			rec := httptest.NewRecorder()
			h.handleOIDCCallback(rec, req)
			if rec.Code != http.StatusFound || !strings.Contains(rec.Header().Get("Location"), "error="+tc.want) {
				t.Fatalf("status/location = %d/%q, want callback error %q", rec.Code, rec.Header().Get("Location"), tc.want)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// handleOIDCCallback — full characterization suite.
//
// internal/oidc.Provider (see internal/oidc/provider.go) is a thin wrapper
// around go-oidc's discovery + verification against a real HTTP endpoint —
// there's no seam in this codebase to fake Exchange/VerifyIDToken/UserInfo
// directly. So every branch past the CSRF/state check is exercised against a
// real *oidc.Provider pointed at a local httptest server that serves
// discovery, JWKS (via go-oidc's own oidctest test-IdP helper), and canned
// /token + /userinfo responses per test case.
// ---------------------------------------------------------------------------

// testIdP is a minimal fake OIDC identity provider: discovery document, JWKS,
// token endpoint, and userinfo endpoint, all backed by one RSA keypair.
type testIdP struct {
	srv      *httptest.Server
	priv     *rsa.PrivateKey
	keyID    string
	clientID string

	tokenStatus int    // 0 => 200
	tokenBody   string // raw JSON body; empty => default success body wrapping idToken
	idToken     string // signed JWT returned as "id_token" in the default token body

	userInfoStatus int    // 0 => 200
	userInfoBody   string // raw JSON body; empty => "{}"
}

func newTestIdP(t *testing.T, clientID string) *testIdP {
	t.Helper()
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate RSA key: %v", err)
	}
	idp := &testIdP{priv: priv, keyID: "test-key", clientID: clientID}

	keys := &oidctest.Server{PublicKeys: []oidctest.PublicKey{
		{PublicKey: priv.Public(), KeyID: idp.keyID, Algorithm: goidc.RS256},
	}}

	mux := http.NewServeMux()
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	idp.srv = srv
	keys.SetIssuer(srv.URL)

	mux.HandleFunc("/.well-known/openid-configuration", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"issuer":%q,"authorization_endpoint":%q,"token_endpoint":%q,"userinfo_endpoint":%q,"jwks_uri":%q,"id_token_signing_alg_values_supported":["RS256"]}`,
			srv.URL, srv.URL+"/auth", srv.URL+"/token", srv.URL+"/userinfo", srv.URL+"/keys")
	})
	mux.Handle("/keys", keys)
	mux.HandleFunc("/token", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		status := idp.tokenStatus
		if status == 0 {
			status = http.StatusOK
		}
		w.WriteHeader(status)
		if idp.tokenBody != "" {
			_, _ = w.Write([]byte(idp.tokenBody))
			return
		}
		fmt.Fprintf(w, `{"access_token":"test-access-token","token_type":"Bearer","expires_in":3600,"id_token":%q}`, idp.idToken)
	})
	mux.HandleFunc("/userinfo", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		status := idp.userInfoStatus
		if status == 0 {
			status = http.StatusOK
		}
		w.WriteHeader(status)
		if idp.userInfoBody != "" {
			_, _ = w.Write([]byte(idp.userInfoBody))
			return
		}
		_, _ = w.Write([]byte(`{}`))
	})

	return idp
}

// signIDToken signs a minimal ID token for this IdP. nonce must match what
// the fake auth store's ConsumeOIDCState returns, or VerifyIDToken's nonce
// check rejects it.
func (idp *testIdP) signIDToken(subject, nonce, email string, emailVerified bool, name string) string {
	claims := fmt.Sprintf(
		`{"iss":%q,"aud":%q,"sub":%q,"exp":%d,"nonce":%q,"email":%q,"email_verified":%t,"name":%q}`,
		idp.srv.URL, idp.clientID, subject, time.Now().Add(time.Hour).Unix(), nonce, email, emailVerified, name,
	)
	return oidctest.SignIDToken(idp.priv, idp.keyID, goidc.RS256, claims)
}

// provider builds the *oidc.Provider the handler looks up in h.providers,
// wired to this fake IdP.
func (idp *testIdP) provider(trustEmail bool) *oidc.Provider {
	return &oidc.Provider{
		ID:           "known",
		Issuer:       idp.srv.URL,
		ClientID:     idp.clientID,
		ClientSecret: "test-secret",
		RedirectURL:  idp.srv.URL + "/cb",
		Scopes:       []string{"openid"},
		TrustEmail:   trustEmail,
	}
}

// oidcCallbackRequest builds a GET .../callback request carrying a matching
// state cookie + query param plus the fixed "code" query param the handler
// requires to proceed past the initial validation.
func oidcCallbackRequest(providerID, state string) *http.Request {
	req := httptest.NewRequest(http.MethodGet,
		"/api/v1/auth/oidc/"+providerID+"/callback?code=test-code&state="+state, nil)
	req.SetPathValue("id", providerID)
	req.AddCookie(&http.Cookie{Name: "dd_oidc_state", Value: state})
	return req
}

// locationParams parses the query string off a redirect Location header.
func locationParams(t *testing.T, rec *httptest.ResponseRecorder) url.Values {
	t.Helper()
	loc := rec.Header().Get("Location")
	u, err := url.Parse(loc)
	if err != nil {
		t.Fatalf("parse Location %q: %v", loc, err)
	}
	return u.Query()
}

// --- CSRF / state handling ---

func TestHandleOIDCCallbackCSRFMismatch(t *testing.T) {
	h := buildAuthSecurityHandler(newFakeAuthStore())
	h.providers["known"] = &oidc.Provider{ID: "known"}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/oidc/known/callback?code=test-code&state=query-state", nil)
	req.SetPathValue("id", "known")
	req.AddCookie(&http.Cookie{Name: "dd_oidc_state", Value: "cookie-state"})
	rec := httptest.NewRecorder()

	h.handleOIDCCallback(rec, req)

	if rec.Code != http.StatusFound || locationParams(t, rec).Get("error") != "invalid_state" {
		t.Fatalf("status/location = %d/%q, want 302 with error=invalid_state", rec.Code, rec.Header().Get("Location"))
	}
}

func TestHandleOIDCCallbackConsumeStateFails(t *testing.T) {
	authStore := newFakeAuthStore()
	authStore.consumeOIDCStateErr = errors.New("state expired")
	h := buildAuthSecurityHandler(authStore)
	h.providers["known"] = &oidc.Provider{ID: "known"}

	req := oidcCallbackRequest("known", "matching-state")
	rec := httptest.NewRecorder()
	h.handleOIDCCallback(rec, req)

	if rec.Code != http.StatusFound || locationParams(t, rec).Get("error") != "invalid_state" {
		t.Fatalf("status/location = %d/%q, want 302 with error=invalid_state", rec.Code, rec.Header().Get("Location"))
	}
}

// --- code exchange / id_token validation ---

func TestHandleOIDCCallbackExchangeFailure(t *testing.T) {
	idp := newTestIdP(t, "test-client")
	idp.tokenStatus = http.StatusBadRequest
	idp.tokenBody = `{"error":"invalid_grant","error_description":"bad code"}`

	authStore := newFakeAuthStore()
	authStore.consumeOIDCStateErr = nil
	h := buildAuthSecurityHandler(authStore)
	h.providers["known"] = idp.provider(false)

	req := oidcCallbackRequest("known", "matching-state")
	rec := httptest.NewRecorder()
	h.handleOIDCCallback(rec, req)

	if rec.Code != http.StatusFound || locationParams(t, rec).Get("error") != "provider_error" {
		t.Fatalf("status/location = %d/%q, want 302 with error=provider_error", rec.Code, rec.Header().Get("Location"))
	}
}

func TestHandleOIDCCallbackMissingIDToken(t *testing.T) {
	idp := newTestIdP(t, "test-client")
	idp.tokenBody = `{"access_token":"test-access-token","token_type":"Bearer","expires_in":3600}`

	authStore := newFakeAuthStore()
	authStore.consumeOIDCStateErr = nil
	h := buildAuthSecurityHandler(authStore)
	h.providers["known"] = idp.provider(false)

	req := oidcCallbackRequest("known", "matching-state")
	rec := httptest.NewRecorder()
	h.handleOIDCCallback(rec, req)

	if rec.Code != http.StatusFound || locationParams(t, rec).Get("error") != "provider_error" {
		t.Fatalf("status/location = %d/%q, want 302 with error=provider_error", rec.Code, rec.Header().Get("Location"))
	}
}

// --- UserInfo backfill / TrustEmail ---

func TestHandleOIDCCallbackUserInfoBackfill(t *testing.T) {
	idp := newTestIdP(t, "test-client")
	// ID token carries no email/name — everything must come from UserInfo.
	idp.idToken = idp.signIDToken("backfill-sub", "test-nonce", "", false, "")
	idp.userInfoBody = `{"sub":"backfill-sub","email":"backfill@example.com","email_verified":true,"name":"Backfill User"}`

	authStore := newFakeAuthStore()
	authStore.consumeOIDCStateErr = nil
	authStore.oidcStateNonce = "test-nonce"
	h := buildAuthSecurityHandler(authStore)
	h.providers["known"] = idp.provider(false)

	req := oidcCallbackRequest("known", "matching-state")
	rec := httptest.NewRecorder()
	h.handleOIDCCallback(rec, req)

	if rec.Code != http.StatusFound || locationParams(t, rec).Get("error") != "" {
		t.Fatalf("status/location = %d/%q, want success redirect", rec.Code, rec.Header().Get("Location"))
	}
	if authStore.lastOIDCCreateEmail != "backfill@example.com" {
		t.Errorf("created user email = %q, want backfilled backfill@example.com", authStore.lastOIDCCreateEmail)
	}
	if authStore.lastOIDCCreateDisplay != "Backfill User" {
		t.Errorf("created user display name = %q, want backfilled Backfill User", authStore.lastOIDCCreateDisplay)
	}
}

func TestHandleOIDCCallbackTrustEmailOverride(t *testing.T) {
	idp := newTestIdP(t, "test-client")
	// email_verified=false in the ID token; only TrustEmail should let this
	// through to account creation.
	idp.idToken = idp.signIDToken("trust-sub", "test-nonce", "trust@example.com", false, "Trust User")

	authStore := newFakeAuthStore()
	authStore.consumeOIDCStateErr = nil
	authStore.oidcStateNonce = "test-nonce"
	h := buildAuthSecurityHandler(authStore)
	h.providers["known"] = idp.provider(true) // TrustEmail

	req := oidcCallbackRequest("known", "matching-state")
	rec := httptest.NewRecorder()
	h.handleOIDCCallback(rec, req)

	if rec.Code != http.StatusFound || locationParams(t, rec).Get("error") != "" {
		t.Fatalf("status/location = %d/%q, want success redirect (TrustEmail should allow an unverified email)", rec.Code, rec.Header().Get("Location"))
	}
	if authStore.lastOIDCCreateEmail != "trust@example.com" {
		t.Errorf("created user email = %q, want trust@example.com", authStore.lastOIDCCreateEmail)
	}
}

// --- Link flow ---

func TestHandleOIDCCallbackLinkSuccess(t *testing.T) {
	idp := newTestIdP(t, "test-client")
	idp.idToken = idp.signIDToken("link-sub", "test-nonce", "link@example.com", true, "Link User")

	authStore := newFakeAuthStore()
	authStore.consumeOIDCStateErr = nil
	authStore.oidcStateNonce = "test-nonce"
	authStore.oidcStateLinkUserID = "linked-user-id"
	h := buildAuthSecurityHandler(authStore)
	h.providers["known"] = idp.provider(false)
	h.store.(*fakeMealStore).user = types.User{ID: "linked-user-id", AccountID: "acct-1"}

	req := oidcCallbackRequest("known", "matching-state")
	rec := httptest.NewRecorder()
	h.handleOIDCCallback(rec, req)

	if rec.Code != http.StatusFound {
		t.Fatalf("status = %d, want 302", rec.Code)
	}
	if got := rec.Header().Get("Location"); got != "/auth/callback?link=1" {
		t.Fatalf("Location = %q, want /auth/callback?link=1", got)
	}
	if _, ok := authStore.oidcIdentities["known|link-sub"]; !ok {
		t.Error("expected the identity to be recorded as linked")
	}
}

func TestHandleOIDCCallbackLinkIdentityConflict(t *testing.T) {
	idp := newTestIdP(t, "test-client")
	idp.idToken = idp.signIDToken("link-sub", "test-nonce", "link@example.com", true, "Link User")

	authStore := newFakeAuthStore()
	authStore.consumeOIDCStateErr = nil
	authStore.oidcStateNonce = "test-nonce"
	authStore.oidcStateLinkUserID = "linked-user-id"
	authStore.linkOIDCIdentityErr = types.ErrIdentityLinked
	h := buildAuthSecurityHandler(authStore)
	h.providers["known"] = idp.provider(false)

	req := oidcCallbackRequest("known", "matching-state")
	rec := httptest.NewRecorder()
	h.handleOIDCCallback(rec, req)

	if rec.Code != http.StatusFound || locationParams(t, rec).Get("error") != "already_linked" {
		t.Fatalf("status/location = %d/%q, want 302 with error=already_linked", rec.Code, rec.Header().Get("Location"))
	}
}

func TestHandleOIDCCallbackLinkInternalError(t *testing.T) {
	idp := newTestIdP(t, "test-client")
	idp.idToken = idp.signIDToken("link-sub", "test-nonce", "link@example.com", true, "Link User")

	authStore := newFakeAuthStore()
	authStore.consumeOIDCStateErr = nil
	authStore.oidcStateNonce = "test-nonce"
	authStore.oidcStateLinkUserID = "linked-user-id"
	authStore.linkOIDCIdentityErr = errors.New("db unavailable")
	h := buildAuthSecurityHandler(authStore)
	h.providers["known"] = idp.provider(false)

	req := oidcCallbackRequest("known", "matching-state")
	rec := httptest.NewRecorder()
	h.handleOIDCCallback(rec, req)

	if rec.Code != http.StatusFound || locationParams(t, rec).Get("error") != "internal_error" {
		t.Fatalf("status/location = %d/%q, want 302 with error=internal_error", rec.Code, rec.Header().Get("Location"))
	}
}

// --- Sign-in flow ---

func TestHandleOIDCCallbackSignInExistingIdentityMatch(t *testing.T) {
	idp := newTestIdP(t, "test-client")
	idp.idToken = idp.signIDToken("existing-sub", "test-nonce", "existing@example.com", true, "Existing User")

	authStore := newFakeAuthStore()
	authStore.consumeOIDCStateErr = nil
	authStore.oidcStateNonce = "test-nonce"
	authStore.oidcIdentities = map[string]types.User{
		"known|existing-sub": {ID: "existing-user-id", AccountID: "acct-2", Email: "existing@example.com"},
	}
	h := buildAuthSecurityHandler(authStore)
	h.providers["known"] = idp.provider(false)

	req := oidcCallbackRequest("known", "matching-state")
	rec := httptest.NewRecorder()
	h.handleOIDCCallback(rec, req)

	if rec.Code != http.StatusFound {
		t.Fatalf("status = %d, want 302", rec.Code)
	}
	if got := rec.Header().Get("Location"); got != "/auth/callback" {
		t.Fatalf("Location = %q, want /auth/callback (no error, no new account)", got)
	}
	if authStore.lastOIDCCreateEmail != "" {
		t.Error("expected no new user creation when an existing identity matched")
	}
}

func TestHandleOIDCCallbackSignInAutoLinkByVerifiedEmail(t *testing.T) {
	idp := newTestIdP(t, "test-client")
	idp.idToken = idp.signIDToken("auto-link-sub", "test-nonce", "autolink@example.com", true, "Auto Link")

	authStore := newFakeAuthStore()
	authStore.consumeOIDCStateErr = nil
	authStore.oidcStateNonce = "test-nonce"
	authStore.userByEmail["autolink@example.com"] = types.User{ID: "user-x", AccountID: "acct-3", Email: "autolink@example.com"}
	h := buildAuthSecurityHandler(authStore)
	h.providers["known"] = idp.provider(false)

	req := oidcCallbackRequest("known", "matching-state")
	rec := httptest.NewRecorder()
	h.handleOIDCCallback(rec, req)

	if rec.Code != http.StatusFound || locationParams(t, rec).Get("error") != "" {
		t.Fatalf("status/location = %d/%q, want success redirect", rec.Code, rec.Header().Get("Location"))
	}
	if _, ok := authStore.oidcIdentities["known|auto-link-sub"]; !ok {
		t.Error("expected the identity to be auto-linked by verified email")
	}
	if authStore.lastOIDCCreateEmail != "" {
		t.Error("expected no new user creation on auto-link by email")
	}
}

func TestHandleOIDCCallbackSignInNewUserRegistrationOpen(t *testing.T) {
	idp := newTestIdP(t, "test-client")
	idp.idToken = idp.signIDToken("new-sub", "test-nonce", "newuser@example.com", true, "New User")

	authStore := newFakeAuthStore()
	authStore.consumeOIDCStateErr = nil
	authStore.oidcStateNonce = "test-nonce"
	h := buildAuthSecurityHandler(authStore) // userCount=0, single-user, RegistrationOpen => allowed

	h.providers["known"] = idp.provider(false)

	req := oidcCallbackRequest("known", "matching-state")
	rec := httptest.NewRecorder()
	h.handleOIDCCallback(rec, req)

	if rec.Code != http.StatusFound || locationParams(t, rec).Get("error") != "" {
		t.Fatalf("status/location = %d/%q, want success redirect", rec.Code, rec.Header().Get("Location"))
	}
	if authStore.lastOIDCCreateEmail != "newuser@example.com" {
		t.Errorf("created user email = %q, want newuser@example.com", authStore.lastOIDCCreateEmail)
	}
}

func TestHandleOIDCCallbackSignInRejectedRegistrationClosed(t *testing.T) {
	idp := newTestIdP(t, "test-client")
	idp.idToken = idp.signIDToken("closed-sub", "test-nonce", "closed@example.com", true, "Closed User")

	authStore := newFakeAuthStore()
	authStore.consumeOIDCStateErr = nil
	authStore.oidcStateNonce = "test-nonce"
	authStore.userCount = 1 // single-user mode already has a user => registration closed
	h := buildAuthSecurityHandler(authStore)
	h.providers["known"] = idp.provider(false)

	req := oidcCallbackRequest("known", "matching-state")
	rec := httptest.NewRecorder()
	h.handleOIDCCallback(rec, req)

	if rec.Code != http.StatusFound || locationParams(t, rec).Get("error") != "registration_closed" {
		t.Fatalf("status/location = %d/%q, want 302 with error=registration_closed", rec.Code, rec.Header().Get("Location"))
	}
}

func TestHandleOIDCCallbackSignInRejectedEmailUnverified(t *testing.T) {
	idp := newTestIdP(t, "test-client")
	idp.idToken = idp.signIDToken("unverified-sub", "test-nonce", "unverified@example.com", false, "Unverified User")

	authStore := newFakeAuthStore()
	authStore.consumeOIDCStateErr = nil
	authStore.oidcStateNonce = "test-nonce"
	h := buildAuthSecurityHandler(authStore)
	h.providers["known"] = idp.provider(false) // no TrustEmail

	req := oidcCallbackRequest("known", "matching-state")
	rec := httptest.NewRecorder()
	h.handleOIDCCallback(rec, req)

	if rec.Code != http.StatusFound || locationParams(t, rec).Get("error") != "email_unverified" {
		t.Fatalf("status/location = %d/%q, want 302 with error=email_unverified", rec.Code, rec.Header().Get("Location"))
	}
}

// --- Session creation / redirect ---

func TestHandleOIDCCallbackSessionCreationFailure(t *testing.T) {
	idp := newTestIdP(t, "test-client")
	idp.idToken = idp.signIDToken("existing-sub", "test-nonce", "existing@example.com", true, "Existing User")

	authStore := newFakeAuthStore()
	authStore.consumeOIDCStateErr = nil
	authStore.oidcStateNonce = "test-nonce"
	authStore.oidcIdentities = map[string]types.User{
		"known|existing-sub": {ID: "existing-user-id", AccountID: "acct-2", Email: "existing@example.com"},
	}
	authStore.createSessionErr = errors.New("db write failed")
	h := buildAuthSecurityHandler(authStore)
	h.providers["known"] = idp.provider(false)

	req := oidcCallbackRequest("known", "matching-state")
	rec := httptest.NewRecorder()
	h.handleOIDCCallback(rec, req)

	if rec.Code != http.StatusFound || locationParams(t, rec).Get("error") != "internal_error" {
		t.Fatalf("status/location = %d/%q, want 302 with error=internal_error", rec.Code, rec.Header().Get("Location"))
	}
}

func TestHandleOIDCCallbackNextRedirectPassthrough(t *testing.T) {
	idp := newTestIdP(t, "test-client")
	idp.idToken = idp.signIDToken("existing-sub", "test-nonce", "existing@example.com", true, "Existing User")

	authStore := newFakeAuthStore()
	authStore.consumeOIDCStateErr = nil
	authStore.oidcStateNonce = "test-nonce"
	authStore.oidcStateNext = "/dashboard"
	authStore.oidcIdentities = map[string]types.User{
		"known|existing-sub": {ID: "existing-user-id", AccountID: "acct-2", Email: "existing@example.com"},
	}
	h := buildAuthSecurityHandler(authStore)
	h.providers["known"] = idp.provider(false)

	req := oidcCallbackRequest("known", "matching-state")
	rec := httptest.NewRecorder()
	h.handleOIDCCallback(rec, req)

	if rec.Code != http.StatusFound {
		t.Fatalf("status = %d, want 302", rec.Code)
	}
	params := locationParams(t, rec)
	if params.Get("error") != "" {
		t.Fatalf("unexpected error param: %q", params.Get("error"))
	}
	if got := params.Get("next"); got != "/dashboard" {
		t.Errorf("next param = %q, want /dashboard", got)
	}
}
