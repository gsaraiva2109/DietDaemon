package api

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

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
