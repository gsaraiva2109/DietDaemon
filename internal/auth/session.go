package auth

import (
	"context"
	"crypto/subtle"
	"time"
)

// Session is a server-side session row. ID is HashToken(cookieValue) so the
// raw token never hits the database.
type Session struct {
	ID                string
	UserID            string
	CSRFToken         string
	CreatedAt         time.Time
	LastSeenAt        time.Time
	IdleExpiresAt     time.Time
	AbsoluteExpiresAt time.Time
	Remember          bool
	IP                string
	UserAgent         string
}

// SessionRepo is the persistence boundary for sessions. Implemented by the
// store; the auth package only calls this interface.
type SessionRepo interface {
	CreateSession(ctx context.Context, s Session) error
	GetSession(ctx context.Context, id string) (Session, error)
	TouchSession(ctx context.Context, id string, lastSeen, idleExpires time.Time) error
	DeleteSession(ctx context.Context, id string) error
	DeleteUserSessions(ctx context.Context, userID string) error
}

// SessionConfig holds the TTL and cookie settings injected from config.
type SessionConfig struct {
	IdleTTL     time.Duration
	AbsoluteTTL time.Duration
	RememberTTL time.Duration
}

// CreateSession generates a new session for userID. It returns the raw cookie
// token (sent to the browser), the CSRF token, and the Session row to persist.
func CreateSession(userID string, remember bool, ip, userAgent string, cfg SessionConfig) (cookieToken, csrfToken string, s Session) {
	cookieToken = NewToken()
	s.ID = HashToken(cookieToken)
	s.UserID = userID
	csrfToken = NewToken()
	s.CSRFToken = csrfToken
	now := time.Now().UTC()
	s.CreatedAt = now
	s.LastSeenAt = now
	s.IdleExpiresAt = now.Add(cfg.IdleTTL)
	if remember {
		s.AbsoluteExpiresAt = now.Add(cfg.RememberTTL)
	} else {
		s.AbsoluteExpiresAt = now.Add(cfg.AbsoluteTTL)
	}
	s.Remember = remember
	s.IP = ip
	s.UserAgent = userAgent
	return cookieToken, csrfToken, s
}

// ValidateResult is the outcome of ValidateSession.
type ValidateResult int

const (
	ValidateOK       ValidateResult = iota // session valid
	ValidateExpired                        // idle or absolute expiry reached
	ValidateNotFound                       // no such session
)

// ValidateSession checks whether the session identified by cookieValue is
// still alive. It returns the Session on ValidateOK. On idle or absolute
// expiry, the session row is deleted. cfg.IdleTTL is used to slide idle
// expiry forward on each validation.
func ValidateSession(ctx context.Context, repo SessionRepo, cookieValue string, cfg SessionConfig) (Session, ValidateResult, error) {
	id := HashToken(cookieValue)
	s, err := repo.GetSession(ctx, id)
	if err != nil {
		return Session{}, ValidateNotFound, err
	}

	now := time.Now().UTC()

	// Absolute expiry: session lived its full lifetime.
	if now.After(s.AbsoluteExpiresAt) {
		_ = repo.DeleteSession(ctx, id)
		return Session{}, ValidateExpired, nil
	}

	// Idle expiry: user was inactive too long.
	if now.After(s.IdleExpiresAt) {
		_ = repo.DeleteSession(ctx, id)
		return Session{}, ValidateExpired, nil
	}

	// Slide idle expiry forward, capped at absolute expiry.
	newIdle := now.Add(cfg.IdleTTL)
	if newIdle.After(s.AbsoluteExpiresAt) {
		newIdle = s.AbsoluteExpiresAt
	}
	_ = repo.TouchSession(ctx, id, now, newIdle)

	return s, ValidateOK, nil
}

// RotateSession creates a new session, copies user/reference data from old,
// and deletes old. Returns the new cookie token (+ row). Called on login and
// privilege-change to prevent session-fixation.
func RotateSession(ctx context.Context, repo SessionRepo, oldSession Session, remember bool, ip, userAgent string, cfg SessionConfig) (cookieToken, csrfToken string, s Session) {
	_ = repo.DeleteSession(ctx, oldSession.ID)
	return CreateSession(oldSession.UserID, remember, ip, userAgent, cfg)
}

// VerifyCSRF compares a header value against the session's CSRF token in
// constant time.
func VerifyCSRF(headerVal, sessionCSRF string) bool {
	return subtle.ConstantTimeCompare([]byte(headerVal), []byte(sessionCSRF)) == 1
}
