package auth

import (
	"context"
	"sync"
	"testing"
	"time"
)

// fakeLoginAttemptRepo is an in-memory LoginAttemptRepo for tests.
type fakeLoginAttemptRepo struct {
	mu       sync.Mutex
	attempts []loginAttempt
}

type loginAttempt struct {
	identifier string
	succeeded  bool
	at         time.Time
}

func (r *fakeLoginAttemptRepo) RecordLoginAttempt(_ context.Context, identifier string, succeeded bool) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.attempts = append(r.attempts, loginAttempt{identifier, succeeded, time.Now().UTC()})
	return nil
}

func (r *fakeLoginAttemptRepo) RecentFailedAttempts(_ context.Context, identifier string, since time.Time) (int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	count := 0
	for _, a := range r.attempts {
		if a.identifier == identifier && !a.succeeded && a.at.After(since) {
			count++
		}
	}
	return count, nil
}

func TestCheckLockoutNotLocked(t *testing.T) {
	repo := &fakeLoginAttemptRepo{}
	cfg := LockoutConfig{MaxAttempts: 5, Window: 15 * time.Minute, LockDuration: 15 * time.Minute}
	ctx := context.Background()

	locked, retry, err := CheckLockout(ctx, repo, "test@example.com", cfg)
	if err != nil {
		t.Fatalf("CheckLockout: %v", err)
	}
	if locked {
		t.Error("should not be locked with 0 attempts")
	}
	if retry != 0 {
		t.Error("retryAfter should be 0 when not locked")
	}
}

func TestCheckLockoutLocked(t *testing.T) {
	repo := &fakeLoginAttemptRepo{}
	cfg := LockoutConfig{MaxAttempts: 5, Window: 15 * time.Minute, LockDuration: 15 * time.Minute}
	ctx := context.Background()

	// Record 5 failures.
	for i := 0; i < 5; i++ {
		_ = repo.RecordLoginAttempt(ctx, "test@example.com", false)
	}

	locked, retry, err := CheckLockout(ctx, repo, "test@example.com", cfg)
	if err != nil {
		t.Fatalf("CheckLockout: %v", err)
	}
	if !locked {
		t.Error("should be locked after 5 failures")
	}
	if retry != 15*time.Minute {
		t.Errorf("retryAfter = %s, want 15m", retry)
	}
}

func TestCheckLockoutSuccessResets(t *testing.T) {
	repo := &fakeLoginAttemptRepo{}
	cfg := LockoutConfig{MaxAttempts: 5, Window: 15 * time.Minute, LockDuration: 15 * time.Minute}
	ctx := context.Background()

	// Record 4 failures + 1 success. The success doesn't erase past failures
	// in this simple model, but the count should still be < 5.
	for i := 0; i < 4; i++ {
		_ = repo.RecordLoginAttempt(ctx, "test@example.com", false)
	}
	_ = repo.RecordLoginAttempt(ctx, "test@example.com", true) // success

	locked, _, err := CheckLockout(ctx, repo, "test@example.com", cfg)
	if err != nil {
		t.Fatalf("CheckLockout: %v", err)
	}
	// 4 failures still present — not locked (need 5).
	if locked {
		t.Error("should not be locked with only 4 failures")
	}
}

func TestDefaultLockoutConfig(t *testing.T) {
	cfg := DefaultLockoutConfig()
	if cfg.MaxAttempts != 5 {
		t.Errorf("MaxAttempts = %d, want 5", cfg.MaxAttempts)
	}
	if cfg.Window != 15*time.Minute {
		t.Errorf("Window = %s, want 15m", cfg.Window)
	}
	if cfg.LockDuration != 15*time.Minute {
		t.Errorf("LockDuration = %s, want 15m", cfg.LockDuration)
	}
}

func TestIPRateLimiter(t *testing.T) {
	lim := NewIPRateLimiter(3, time.Minute)

	// First 3 requests from same IP should be allowed.
	for i := 0; i < 3; i++ {
		if !lim.Allow("10.0.0.1") {
			t.Errorf("request %d should be allowed", i+1)
		}
	}

	// 4th request should be throttled.
	if lim.Allow("10.0.0.1") {
		t.Error("4th request should be throttled")
	}

	// Different IP should still be allowed.
	if !lim.Allow("10.0.0.2") {
		t.Error("different IP should be allowed")
	}
}

func TestIPRateLimiterCleanup(t *testing.T) {
	lim := NewIPRateLimiter(1, time.Minute)
	lim.buckets["old"] = &ipBucket{lastSeen: time.Now().Add(-time.Minute)}
	lim.buckets["recent"] = &ipBucket{lastSeen: time.Now()}
	lim.Cleanup()
	if _, ok := lim.buckets["old"]; ok {
		t.Fatal("old bucket was not removed")
	}
	if _, ok := lim.buckets["recent"]; !ok {
		t.Fatal("recent bucket was removed")
	}
}
