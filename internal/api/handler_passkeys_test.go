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

// webauthnCeremony is a stored (pending) ceremony session, keyed by ceremony
// cookie value.
type webauthnCeremony struct {
	userID, session, expiresAt string
}

// webauthnCred is a stored passkey credential, keyed by credential ID.
type webauthnCred struct {
	userID, label, credentialJSON, createdAt, lastUsedAt string
	signCount                                            int
}

type mfaChallengeRec struct {
	userID, expiresAt string
	remember          bool
}

// passkeyTestStore is a stateful WebAuthnStore double: ceremony sessions,
// credentials, handles, and MFA challenges are actually persisted (in-memory)
// so multi-step ceremonies (begin -> finish) round-trip like the real store,
// instead of the earlier always-fails stub.
type passkeyTestStore struct {
	*fakeAuthStore
	createdCeremony struct {
		id, userID, session, expiresAt string
	}

	ceremonies map[string]webauthnCeremony

	handleByUser   map[string]string
	userIDByHandle map[string]string

	creds map[string]webauthnCred

	mfaChallenges map[string]mfaChallengeRec

	sessions map[string]auth.Session

	totpConfirmed bool

	listCredsErr, renameErr, deleteErr                          error
	getOrCreateHandleErr, getCredsErr, createWebAuthnSessionErr error
}

func newPasskeyTestStore() *passkeyTestStore {
	return &passkeyTestStore{
		fakeAuthStore:  newFakeAuthStore(),
		ceremonies:     make(map[string]webauthnCeremony),
		handleByUser:   make(map[string]string),
		userIDByHandle: make(map[string]string),
		creds:          make(map[string]webauthnCred),
		mfaChallenges:  make(map[string]mfaChallengeRec),
		sessions:       make(map[string]auth.Session),
	}
}

func (s *passkeyTestStore) GetOrCreateWebAuthnHandle(_ context.Context, userID string) (string, error) {
	if s.getOrCreateHandleErr != nil {
		return "", s.getOrCreateHandleErr
	}
	if h, ok := s.handleByUser[userID]; ok {
		return h, nil
	}
	h := auth.NewWebAuthnHandle()
	s.handleByUser[userID] = h
	s.userIDByHandle[h] = userID
	return h, nil
}

func (s *passkeyTestStore) GetUserByWebAuthnHandle(_ context.Context, handle string) (types.User, error) {
	userID, ok := s.userIDByHandle[handle]
	if !ok {
		return types.User{}, types.ErrNotFound
	}
	if u, ok := s.users[userID]; ok {
		return u, nil
	}
	return types.User{}, types.ErrNotFound
}

func (s *passkeyTestStore) CreateWebAuthnCredential(_ context.Context, id, userID, label, credentialJSON string, signCount int, createdAt string) error {
	s.creds[id] = webauthnCred{userID: userID, label: label, credentialJSON: credentialJSON, signCount: signCount, createdAt: createdAt}
	return nil
}

func (s *passkeyTestStore) ListWebAuthnCredentials(_ context.Context, userID string) ([]types.Passkey, error) {
	if s.listCredsErr != nil {
		return nil, s.listCredsErr
	}
	var out []types.Passkey
	for id, c := range s.creds {
		if c.userID != userID {
			continue
		}
		out = append(out, types.Passkey{ID: id, Label: c.label, CreatedAt: c.createdAt, LastUsedAt: c.lastUsedAt})
	}
	return out, nil
}

func (s *passkeyTestStore) GetWebAuthnCredentialsRaw(_ context.Context, userID string) ([]types.WebAuthnCredential, error) {
	if s.getCredsErr != nil {
		return nil, s.getCredsErr
	}
	var out []types.WebAuthnCredential
	for id, c := range s.creds {
		if c.userID != userID {
			continue
		}
		out = append(out, types.WebAuthnCredential{ID: id, CredentialJSON: c.credentialJSON})
	}
	return out, nil
}

func (s *passkeyTestStore) UpdateWebAuthnCredentialOnAuth(_ context.Context, id, credentialJSON string, signCount int, lastUsedAt string) error {
	c, ok := s.creds[id]
	if !ok {
		return types.ErrNotFound
	}
	c.credentialJSON = credentialJSON
	c.signCount = signCount
	c.lastUsedAt = lastUsedAt
	s.creds[id] = c
	return nil
}

func (s *passkeyTestStore) RenameWebAuthnCredential(_ context.Context, userID, id, label string) error {
	if s.renameErr != nil {
		return s.renameErr
	}
	c, ok := s.creds[id]
	if !ok || c.userID != userID {
		return types.ErrNotFound
	}
	c.label = label
	s.creds[id] = c
	return nil
}

func (s *passkeyTestStore) DeleteWebAuthnCredential(_ context.Context, userID, id string) error {
	if s.deleteErr != nil {
		return s.deleteErr
	}
	c, ok := s.creds[id]
	if !ok || c.userID != userID {
		return types.ErrNotFound
	}
	delete(s.creds, id)
	return nil
}

func (s *passkeyTestStore) CreateWebAuthnSession(_ context.Context, id, userID, session, expiresAt string) error {
	if s.createWebAuthnSessionErr != nil {
		return s.createWebAuthnSessionErr
	}
	s.createdCeremony.id = id
	s.createdCeremony.userID = userID
	s.createdCeremony.session = session
	s.createdCeremony.expiresAt = expiresAt
	s.ceremonies[id] = webauthnCeremony{userID: userID, session: session, expiresAt: expiresAt}
	return nil
}

func (s *passkeyTestStore) ConsumeWebAuthnSession(_ context.Context, id string) (string, string, error) {
	c, ok := s.ceremonies[id]
	if !ok {
		return "", "", types.ErrNotFound
	}
	delete(s.ceremonies, id)
	return c.userID, c.session, nil
}

func (s *passkeyTestStore) HasConfirmedTOTP(_ context.Context, _ string) (bool, error) {
	return s.totpConfirmed, nil
}

func (s *passkeyTestStore) CreateMFAChallenge(_ context.Context, id, userID string, remember bool, expiresAt string) error {
	s.mfaChallenges[id] = mfaChallengeRec{userID: userID, remember: remember, expiresAt: expiresAt}
	return nil
}

func (s *passkeyTestStore) GetMFAChallenge(_ context.Context, id string) (string, bool, string, error) {
	c, ok := s.mfaChallenges[id]
	if !ok {
		return "", false, "", types.ErrNotFound
	}
	return c.userID, c.remember, c.expiresAt, nil
}

func (s *passkeyTestStore) DeleteMFAChallenge(_ context.Context, id string) error {
	delete(s.mfaChallenges, id)
	return nil
}

func (s *passkeyTestStore) CreateSession(_ context.Context, sess auth.Session) error {
	s.sessions[sess.ID] = sess
	return nil
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
	// The real store persists GetOrCreateWebAuthnHandle's result onto the
	// user row, so a later GetUser sees it too. Our meal-store and
	// auth-store doubles are separate fakes, so pre-agree on the handle here
	// to keep WebAuthnID() consistent between BeginRegistration (which reads
	// it via authStore) and CreateCredential (which reads it via a fresh
	// h.store.GetUser) — otherwise go-webauthn rejects with "ID mismatch for
	// User and Session".
	handle := auth.NewWebAuthnHandle()
	store.user = types.User{ID: "user-1", AccountID: "account-1", Email: "user@example.com", DisplayName: "User", WebAuthnHandle: handle}
	authStore.handleByUser["user-1"] = handle
	authStore.userIDByHandle[handle] = "user-1"
	return New(store, &fakeMealLogger{}, time.UTC, nil, nil,
		WithAuth(authStore, AuthRepos{Sessions: authStore, LoginAttempts: authStore, TOTP: authStore, MFAChallenges: authStore, RecoveryCodes: authStore}, nil, "DietDaemon", AuthConfig{
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
