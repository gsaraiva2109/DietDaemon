package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gsaraiva2109/dietdaemon/core/types"
	"github.com/gsaraiva2109/dietdaemon/internal/auth"
)

type passkeyTestStore struct {
	*fakeAuthStore
	createdCeremony struct {
		id, userID, session, expiresAt string
	}
}

func newPasskeyTestStore() *passkeyTestStore {
	return &passkeyTestStore{fakeAuthStore: newFakeAuthStore()}
}

func (s *passkeyTestStore) GetOrCreateWebAuthnHandle(_ context.Context, _ string) (string, error) {
	return auth.NewWebAuthnHandle(), nil
}

func (s *passkeyTestStore) CreateWebAuthnSession(_ context.Context, id, userID, session, expiresAt string) error {
	s.createdCeremony.id = id
	s.createdCeremony.userID = userID
	s.createdCeremony.session = session
	s.createdCeremony.expiresAt = expiresAt
	return nil
}

func (s *passkeyTestStore) ConsumeWebAuthnSession(_ context.Context, _ string) (string, string, error) {
	return "", "", types.ErrNotFound
}

func newPasskeyHandler(t *testing.T, authStore *passkeyTestStore) *Handler {
	t.Helper()
	wa, err := auth.NewWebAuthn(auth.WebAuthnConfig{
		RPID:          "example.com",
		RPDisplayName: "DietDaemon",
		RPOrigins:     []string{"https://example.com"},
	})
	if err != nil {
		t.Fatalf("NewWebAuthn: %v", err)
	}
	store := newFakeMealStore()
	store.user = types.User{ID: "user-1", AccountID: "account-1", Email: "user@example.com", DisplayName: "User"}
	return New(store, &fakeMealLogger{}, time.UTC, nil, nil,
		WithAuth(authStore, authStore, authStore, authStore, authStore, authStore, nil, "DietDaemon", AuthConfig{
			SessionCfg: auth.SessionConfig{IdleTTL: time.Hour, AbsoluteTTL: 24 * time.Hour, RememberTTL: 72 * time.Hour},
			LockoutCfg: auth.DefaultLockoutConfig(),
		}),
		WithWebAuthn(wa),
	)
}

func TestHandlePasskeyRegisterBeginCreatesCeremony(t *testing.T) {
	authStore := newPasskeyTestStore()
	h := newPasskeyHandler(t, authStore)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/passkeys/register/begin", nil)

	h.handlePasskeyRegisterBegin(rec, req, "user-1")

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200: %s", rec.Code, rec.Body.String())
	}
	if authStore.createdCeremony.id == "" || authStore.createdCeremony.userID != "user-1" || authStore.createdCeremony.session == "" {
		t.Fatalf("ceremony = %+v, want stored ceremony for user-1", authStore.createdCeremony)
	}
	if got := rec.Result().Cookies(); len(got) != 1 || got[0].Name != "dd_webauthn" || !got[0].HttpOnly || got[0].Path != "/" {
		t.Fatalf("cookies = %+v, want HttpOnly dd_webauthn ceremony cookie", got)
	}
}

func TestHandlePasskeyLoginBeginCreatesDiscoverableCeremony(t *testing.T) {
	authStore := newPasskeyTestStore()
	h := newPasskeyHandler(t, authStore)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/passkeys/login/begin", nil)

	h.handlePasskeyLoginBegin(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200: %s", rec.Code, rec.Body.String())
	}
	if authStore.createdCeremony.id == "" || authStore.createdCeremony.userID != "" || authStore.createdCeremony.session == "" {
		t.Fatalf("ceremony = %+v, want discoverable ceremony", authStore.createdCeremony)
	}
}

func TestHandlePasskeyLoginFinishRejectsMissingOrExpiredCeremony(t *testing.T) {
	h := newPasskeyHandler(t, newPasskeyTestStore())
	for name, req := range map[string]*http.Request{
		"missing": httptest.NewRequest(http.MethodPost, "/", nil),
		"expired": func() *http.Request {
			r := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{}`))
			r.AddCookie(&http.Cookie{Name: "dd_webauthn", Value: "expired"})
			return r
		}(),
	} {
		t.Run(name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			h.handlePasskeyLoginFinish(rec, req)
			want := http.StatusBadRequest
			if name == "expired" {
				want = http.StatusUnauthorized
			}
			if rec.Code != want {
				t.Fatalf("status = %d, want %d: %s", rec.Code, want, rec.Body.String())
			}
		})
	}
}
