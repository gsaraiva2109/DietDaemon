package auth

import (
	"context"
	"sync"
	"time"
)

// LoginAttemptRepo is the persistence boundary for login-attempt tracking.
// Implemented by the store.
type LoginAttemptRepo interface {
	RecordLoginAttempt(ctx context.Context, identifier string, succeeded bool) error
	RecentFailedAttempts(ctx context.Context, identifier string, since time.Time) (int, error)
}

// LockoutConfig holds the brute-force lockout policy knobs.
type LockoutConfig struct {
	MaxAttempts  int           // failures before lockout
	Window       time.Duration // sliding window for counting failures
	LockDuration time.Duration // how long the lockout lasts
}

// DefaultLockoutConfig returns the standard policy: 5 fails in 15 min → locked 15 min.
func DefaultLockoutConfig() LockoutConfig {
	return LockoutConfig{
		MaxAttempts:  5,
		Window:       15 * time.Minute,
		LockDuration: 15 * time.Minute,
	}
}

// CheckLockout queries recent failed attempts and decides whether the
// identifier (email or "ip:<addr>") is locked. Returns true + retryAfter
// when locked.
func CheckLockout(ctx context.Context, repo LoginAttemptRepo, identifier string, cfg LockoutConfig) (locked bool, retryAfter time.Duration, err error) {
	since := time.Now().UTC().Add(-cfg.Window)
	count, err := repo.RecentFailedAttempts(ctx, identifier, since)
	if err != nil {
		return false, 0, err
	}

	if count >= cfg.MaxAttempts {
		// Locked — tell the caller how long to wait.
		// We don't know exactly when the Nth attempt was, so assume now.
		return true, cfg.LockDuration, nil
	}

	return false, 0, nil
}

// IPRateLimiter is a lightweight in-memory fixed-window limiter. Callers use
// client IPs for public endpoints and user IDs for authenticated endpoints.
type IPRateLimiter struct {
	mu       sync.Mutex
	buckets  map[string]*ipBucket
	maxReqs  int
	interval time.Duration
}

type ipBucket struct {
	tokens   int
	lastSeen time.Time
}

// NewIPRateLimiter creates a limiter allowing maxReqs per interval per IP.
func NewIPRateLimiter(maxReqs int, interval time.Duration) *IPRateLimiter {
	return &IPRateLimiter{
		buckets:  make(map[string]*ipBucket),
		maxReqs:  maxReqs,
		interval: interval,
	}
}

// Allow reports whether this key may proceed. True = allowed, false = throttled.
func (l *IPRateLimiter) Allow(key string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := time.Now()
	b, ok := l.buckets[key]
	if !ok || now.Sub(b.lastSeen) > l.interval {
		l.buckets[key] = &ipBucket{tokens: l.maxReqs - 1, lastSeen: now}
		return true
	}

	b.lastSeen = now
	if b.tokens > 0 {
		b.tokens--
		return true
	}
	return false
}

// Cleanup removes keys idle for at least one limiter interval.
func (l *IPRateLimiter) Cleanup() {
	l.mu.Lock()
	defer l.mu.Unlock()
	now := time.Now()
	for key, bucket := range l.buckets {
		if now.Sub(bucket.lastSeen) >= l.interval {
			delete(l.buckets, key)
		}
	}
}
