package api

import (
	"context"
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/gsaraiva2109/dietdaemon/core/types"
	"github.com/gsaraiva2109/dietdaemon/internal/auth"
	"github.com/gsaraiva2109/dietdaemon/internal/mailer"
)

type mfaEmailChallenge struct {
	userID    string
	remember  bool
	expiresAt string
}

type mfaEmailCode struct {
	hash      string
	expiresAt string
	attempts  int
}

type mfaEmailTestStore struct {
	*fakeAuthStore
	challenges map[string]mfaEmailChallenge
	codes      map[string]mfaEmailCode
	sessions   map[string]auth.Session
}

func newMFAEmailTestStore() *mfaEmailTestStore {
	return &mfaEmailTestStore{
		fakeAuthStore: newFakeAuthStore(),
		challenges:    make(map[string]mfaEmailChallenge),
		codes:         make(map[string]mfaEmailCode),
		sessions:      make(map[string]auth.Session),
	}
}

func (s *mfaEmailTestStore) GetMFAChallenge(_ context.Context, id string) (string, bool, string, error) {
	challenge, ok := s.challenges[id]
	if !ok {
		return "", false, "", types.ErrNotFound
	}
	return challenge.userID, challenge.remember, challenge.expiresAt, nil
}

func (s *mfaEmailTestStore) DeleteMFAChallenge(_ context.Context, id string) error {
	delete(s.challenges, id)
	return nil
}

func (s *mfaEmailTestStore) UpsertMFAEmailCode(_ context.Context, userID, hash, expiresAt string) error {
	s.codes[userID] = mfaEmailCode{hash: hash, expiresAt: expiresAt}
	return nil
}

func (s *mfaEmailTestStore) GetMFAEmailCode(_ context.Context, userID string) (string, string, int, error) {
	code, ok := s.codes[userID]
	if !ok {
		return "", "", 0, types.ErrNotFound
	}
	return code.hash, code.expiresAt, code.attempts, nil
}

func (s *mfaEmailTestStore) IncrementMFAEmailCodeAttempts(_ context.Context, userID string) error {
	code := s.codes[userID]
	code.attempts++
	s.codes[userID] = code
	return nil
}

func (s *mfaEmailTestStore) DeleteMFAEmailCode(_ context.Context, userID string) error {
	delete(s.codes, userID)
	return nil
}

func (s *mfaEmailTestStore) CreateSession(_ context.Context, session auth.Session) error {
	s.sessions[session.ID] = session
	return nil
}

func buildMFAEmailHandler(authStore *mfaEmailTestStore, m mailer.Mailer) *Handler {
	store := newFakeMealStore()
	verifiedAt := time.Now().UTC()
	store.user = types.User{
		ID:              "test-user",
		AccountID:       "acct-1",
		Email:           "test@example.com",
		EmailVerifiedAt: &verifiedAt,
		Status:          "active",
		CreatedAt:       verifiedAt,
	}
	return New(store, &fakeMealLogger{}, time.UTC, nil, nil,
		WithAuth(authStore, authStore, authStore, authStore, authStore, authStore, nil, "DietDaemon", AuthConfig{
			SessionCfg: auth.SessionConfig{
				IdleTTL:     time.Hour,
				AbsoluteTTL: 24 * time.Hour,
				RememberTTL: 72 * time.Hour,
			},
			LockoutCfg:       auth.DefaultLockoutConfig(),
			RegistrationMode: types.RegistrationOpen,
		}),
		WithMailer(m, "smtp"),
	)
}

func (s *mfaEmailTestStore) addChallenge(token string, expiresAt time.Time) {
	s.challenges[auth.HashToken(token)] = mfaEmailChallenge{
		userID:    "test-user",
		remember:  true,
		expiresAt: expiresAt.Format(time.RFC3339),
	}
}

func TestMFAEmailSendAndVerify(t *testing.T) {
	authStore := newMFAEmailTestStore()
	mailer := &fakeMailer{}
	h := buildMFAEmailHandler(authStore, mailer)
	challengeToken := "valid-challenge"
	authStore.addChallenge(challengeToken, time.Now().UTC().Add(time.Minute))

	rec := doRequest(h, http.MethodPost, "/api/v1/auth/mfa/email/send", map[string]string{"challenge_token": challengeToken}, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	if body := decodeJSON[map[string]string](t, rec); body["ok"] != "true" {
		t.Errorf("expected ok response, got %#v", body)
	}
	if len(mailer.sent) != 1 || mailer.sent[0].to != "test@example.com" {
		t.Errorf("expected MFA email to test user, got %#v", mailer.sent)
	}
	if _, ok := authStore.codes["test-user"]; !ok {
		t.Fatal("expected send to store an MFA code")
	}

	authStore.codes["test-user"] = mfaEmailCode{
		hash:      auth.HashToken("123456"),
		expiresAt: time.Now().UTC().Add(time.Minute).Format(time.RFC3339),
	}
	rec = doRequest(h, http.MethodPost, "/api/v1/auth/mfa/email/verify", map[string]string{
		"challenge_token": challengeToken,
		"code":            "123456",
	}, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	if body := decodeJSON[sessionResponse](t, rec); body.User.ID != "test-user" {
		t.Errorf("expected test-user session response, got %#v", body)
	}
	if len(authStore.sessions) != 1 {
		t.Errorf("expected one session, got %d", len(authStore.sessions))
	}
	for _, session := range authStore.sessions {
		if !session.Remember {
			t.Error("expected remembered MFA session")
		}
	}
	if _, ok := authStore.codes["test-user"]; ok {
		t.Error("expected code to be consumed")
	}
	if _, ok := authStore.challenges[auth.HashToken(challengeToken)]; ok {
		t.Error("expected challenge to be consumed")
	}
}

func TestMFAEmailSendInvalidChallenge(t *testing.T) {
	h := buildMFAEmailHandler(newMFAEmailTestStore(), &fakeMailer{})

	rec := doRequest(h, http.MethodPost, "/api/v1/auth/mfa/email/send", map[string]string{"challenge_token": "invalid"}, nil)
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rec.Code)
	}
	if body := decodeJSON[errorEnvelope](t, rec); body.Error.Code != ErrorUnauthorized || body.Error.Message != "invalid challenge" {
		t.Errorf("expected invalid challenge response, got %#v", body)
	}
}

func TestMFAEmailSendDeliveryFailureIsInternalAndNotSentAudit(t *testing.T) {
	authStore := newMFAEmailTestStore()
	h := buildMFAEmailHandler(authStore, &fakeMailer{sendErr: errors.New("mail unavailable")})
	challengeToken := "valid-challenge"
	authStore.addChallenge(challengeToken, time.Now().UTC().Add(time.Minute))

	rec := doRequest(h, http.MethodPost, "/api/v1/auth/mfa/email/send", map[string]string{"challenge_token": challengeToken}, nil)
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", rec.Code)
	}
	for _, event := range authStore.auditEvents {
		if event.Event == "mfa.email_code_sent" {
			t.Error("delivery failure must not be audited as sent")
		}
	}
}

func TestMFAEmailVerifyExpiredChallenge(t *testing.T) {
	authStore := newMFAEmailTestStore()
	h := buildMFAEmailHandler(authStore, &fakeMailer{})
	challengeToken := "expired-challenge"
	authStore.addChallenge(challengeToken, time.Now().UTC().Add(-time.Minute))
	authStore.codes["test-user"] = mfaEmailCode{hash: auth.HashToken("123456"), expiresAt: time.Now().UTC().Add(time.Minute).Format(time.RFC3339)}

	rec := doRequest(h, http.MethodPost, "/api/v1/auth/mfa/email/verify", map[string]string{
		"challenge_token": challengeToken,
		"code":            "123456",
	}, nil)
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rec.Code)
	}
	if body := decodeJSON[errorEnvelope](t, rec); body.Error.Code != ErrorUnauthorized || body.Error.Message != "invalid challenge" {
		t.Errorf("expected invalid challenge response, got %#v", body)
	}
	if len(authStore.challenges) != 0 || len(authStore.codes) != 0 {
		t.Errorf("expected expired challenge and code cleanup, got challenges=%d codes=%d", len(authStore.challenges), len(authStore.codes))
	}
}

func TestMFAEmailVerifyWrongCode(t *testing.T) {
	authStore := newMFAEmailTestStore()
	h := buildMFAEmailHandler(authStore, &fakeMailer{})
	challengeToken := "valid-challenge"
	authStore.addChallenge(challengeToken, time.Now().UTC().Add(time.Minute))
	authStore.codes["test-user"] = mfaEmailCode{hash: auth.HashToken("123456"), expiresAt: time.Now().UTC().Add(time.Minute).Format(time.RFC3339)}

	rec := doRequest(h, http.MethodPost, "/api/v1/auth/mfa/email/verify", map[string]string{
		"challenge_token": challengeToken,
		"code":            "654321",
	}, nil)
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rec.Code)
	}
	if body := decodeJSON[errorEnvelope](t, rec); body.Error.Code != ErrorUnauthorized || body.Error.Message != "invalid code" {
		t.Errorf("expected invalid code response, got %#v", body)
	}
	if got := authStore.codes["test-user"].attempts; got != 1 {
		t.Errorf("expected one failed attempt, got %d", got)
	}
}
