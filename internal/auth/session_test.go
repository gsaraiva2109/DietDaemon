package auth

import (
	"context"
	"testing"
	"time"
)

// fakeSessionRepo is an in-memory SessionRepo for tests.
type fakeSessionRepo struct {
	sessions map[string]Session
}

func newFakeSessionRepo() *fakeSessionRepo {
	return &fakeSessionRepo{sessions: make(map[string]Session)}
}

func (r *fakeSessionRepo) CreateSession(_ context.Context, s Session) error {
	r.sessions[s.ID] = s
	return nil
}

func (r *fakeSessionRepo) GetSession(_ context.Context, id string) (Session, error) {
	s, ok := r.sessions[id]
	if !ok {
		return Session{}, ErrInvalidCredentials // stand-in for "not found"
	}
	return s, nil
}

func (r *fakeSessionRepo) TouchSession(_ context.Context, id string, lastSeen, idleExpires time.Time) error {
	s, ok := r.sessions[id]
	if !ok {
		return ErrInvalidCredentials
	}
	s.LastSeenAt = lastSeen
	s.IdleExpiresAt = idleExpires
	r.sessions[id] = s
	return nil
}

func (r *fakeSessionRepo) DeleteSession(_ context.Context, id string) error {
	delete(r.sessions, id)
	return nil
}

func (r *fakeSessionRepo) DeleteUserSessions(_ context.Context, userID string) error {
	for id, s := range r.sessions {
		if s.UserID == userID {
			delete(r.sessions, id)
		}
	}
	return nil
}

func cfg() SessionConfig {
	return SessionConfig{
		IdleTTL:     1 * time.Hour,
		AbsoluteTTL: 24 * time.Hour,
		RememberTTL: 72 * time.Hour,
	}
}

func TestCreateSession(t *testing.T) {
	tok, csrf, s := CreateSession("user-1", false, "1.2.3.4", "GoTest", cfg())

	if tok == "" {
		t.Error("cookie token is empty")
	}
	if csrf == "" {
		t.Error("csrf token is empty")
	}
	if s.ID != HashToken(tok) {
		t.Error("session ID != HashToken(cookie)")
	}
	if s.UserID != "user-1" {
		t.Errorf("userID = %s, want user-1", s.UserID)
	}
	if s.CSRFToken != csrf {
		t.Error("CSRF token mismatch")
	}
	if s.IP != "1.2.3.4" {
		t.Errorf("IP = %s, want 1.2.3.4", s.IP)
	}
	if s.UserAgent != "GoTest" {
		t.Errorf("UserAgent = %s, want GoTest", s.UserAgent)
	}

	now := time.Now().UTC()
	if s.CreatedAt.After(now) {
		t.Error("created_at is in the future")
	}
	if s.AbsoluteExpiresAt.Sub(s.CreatedAt) != 24*time.Hour {
		t.Errorf("absolute TTL = %s, want 24h", s.AbsoluteExpiresAt.Sub(s.CreatedAt))
	}
	if s.Remember {
		t.Error("remember should be false")
	}
}

func TestCreateSessionRemember(t *testing.T) {
	_, _, s := CreateSession("user-1", true, "", "", cfg())
	if !s.Remember {
		t.Error("remember should be true")
	}
	if s.AbsoluteExpiresAt.Sub(s.CreatedAt) != 72*time.Hour {
		t.Errorf("remember absolute TTL = %s, want 72h", s.AbsoluteExpiresAt.Sub(s.CreatedAt))
	}
}

func TestValidateSessionOK(t *testing.T) {
	repo := newFakeSessionRepo()
	ctx := context.Background()
	c := cfg()

	tok, _, s := CreateSession("user-1", false, "", "", c)
	repo.sessions[s.ID] = s

	got, result, err := ValidateSession(ctx, repo, tok, c)
	if err != nil {
		t.Fatalf("ValidateSession: %v", err)
	}
	if result != ValidateOK {
		t.Errorf("result = %v, want ValidateOK", result)
	}
	if got.UserID != "user-1" {
		t.Errorf("userID = %s, want user-1", got.UserID)
	}
}

func TestValidateSessionExpiredAbsolute(t *testing.T) {
	repo := newFakeSessionRepo()
	ctx := context.Background()
	c := cfg()

	// Re-do: store normally, then expire.
	tok2, _, s2 := CreateSession("user-2", false, "", "", c)
	s2.AbsoluteExpiresAt = time.Now().UTC().Add(-1 * time.Hour)
	repo.sessions[s2.ID] = s2

	_, result, err := ValidateSession(ctx, repo, tok2, c)
	if err != nil {
		t.Fatalf("ValidateSession: %v", err)
	}
	if result != ValidateExpired {
		t.Errorf("result = %v, want ValidateExpired", result)
	}
	// Session should be deleted.
	if _, ok := repo.sessions[s2.ID]; ok {
		t.Error("expired session should be deleted")
	}
}

func TestValidateSessionExpiredIdle(t *testing.T) {
	repo := newFakeSessionRepo()
	ctx := context.Background()
	c := cfg()

	tok, _, s := CreateSession("user-1", false, "", "", c)
	s.IdleExpiresAt = time.Now().UTC().Add(-1 * time.Hour)
	repo.sessions[s.ID] = s

	_, result, err := ValidateSession(ctx, repo, tok, c)
	if err != nil {
		t.Fatalf("ValidateSession: %v", err)
	}
	if result != ValidateExpired {
		t.Errorf("result = %v, want ValidateExpired", result)
	}
	if _, ok := repo.sessions[s.ID]; ok {
		t.Error("expired session should be deleted")
	}
}

func TestValidateSessionNotFound(t *testing.T) {
	repo := newFakeSessionRepo()
	ctx := context.Background()
	c := cfg()

	tok := NewToken()
	_, result, err := ValidateSession(ctx, repo, tok, c)
	if err == nil {
		t.Error("expected error for missing session")
	}
	if result != ValidateNotFound {
		t.Errorf("result = %v, want ValidateNotFound", result)
	}
}

func TestRotateSession(t *testing.T) {
	repo := newFakeSessionRepo()
	ctx := context.Background()
	c := cfg()

	_, _, old := CreateSession("user-1", false, "", "", c)
	old.ID = HashToken("old-token")
	repo.sessions[old.ID] = old

	newTok, _, s := RotateSession(ctx, repo, old, false, "", "", c)

	if s.UserID != old.UserID {
		t.Errorf("rotated userID = %s, want %s", s.UserID, old.UserID)
	}
	if newTok == "" {
		t.Error("new token is empty")
	}
	if _, ok := repo.sessions[old.ID]; ok {
		t.Error("old session should be deleted after rotation")
	}
	// RotateSession does NOT persist the new session — the caller must do it.
	// Verify the returned session has a fresh ID.
	if s.ID == old.ID {
		t.Error("new session should have a different ID from old")
	}
}

func TestVerifyCSRF(t *testing.T) {
	if !VerifyCSRF("token", "token") {
		t.Error("matching tokens should verify")
	}
	if VerifyCSRF("token", "different") {
		t.Error("mismatched tokens should not verify")
	}
	if VerifyCSRF("", "token") {
		t.Error("empty header should not verify")
	}
	if VerifyCSRF("token", "") {
		t.Error("empty csrf should not verify")
	}
}
