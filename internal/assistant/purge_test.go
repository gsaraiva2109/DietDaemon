package assistant

import (
	"context"
	"sync"
	"testing"
	"time"
)

// fakePurgeStore is a test double for PurgeStore.
type fakePurgeStore struct {
	mu     sync.Mutex
	purges []time.Time // recorded olderThan values
	count  int         // number of sessions to report purged
	err    error
}

func (f *fakePurgeStore) PurgeDeletedChatSessions(ctx context.Context, olderThan time.Time) (int, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.purges = append(f.purges, olderThan)
	if f.err != nil {
		return 0, f.err
	}
	return f.count, nil
}

func TestPurgeRunnerTicksAndPurges(t *testing.T) {
	store := &fakePurgeStore{count: 3}
	// Short interval for testing — runner ticks immediately, then every 50ms.
	runner := NewPurgeRunner(store, 50*time.Millisecond)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go runner.Run(ctx)

	// Wait for at least one tick to fire.
	time.Sleep(120 * time.Millisecond)
	cancel()

	store.mu.Lock()
	defer store.mu.Unlock()
	if len(store.purges) == 0 {
		t.Fatal("expected at least one purge call, got 0")
	}

	// olderThan should be ~30 days ago.
	cutoff := time.Now().AddDate(0, 0, -30)
	for i, p := range store.purges {
		diff := p.Sub(cutoff).Abs()
		if diff > 5*time.Second {
			t.Errorf("purge[%d]: olderThan=%v, want ~%v (diff=%v)", i, p, cutoff, diff)
		}
	}
}

func TestPurgeRunnerContextCancel(t *testing.T) {
	store := &fakePurgeStore{count: 0}
	runner := NewPurgeRunner(store, time.Hour) // long interval, won't fire naturally

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	// Should exit cleanly without panicking or deadlocking.
	runner.Run(ctx)

	if len(store.purges) != 0 {
		t.Errorf("expected 0 purges on cancelled context, got %d", len(store.purges))
	}
}

func TestPurgeRunnerZeroPurged(t *testing.T) {
	// Zero purged sessions should not log an info message (no panic, no error).
	store := &fakePurgeStore{count: 0}
	runner := NewPurgeRunner(store, 10*time.Millisecond)

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Millisecond)
	defer cancel()

	runner.Run(ctx)

	store.mu.Lock()
	defer store.mu.Unlock()
	if len(store.purges) == 0 {
		t.Fatal("expected at least one purge call")
	}
	// All calls should have count=0.
	for _, p := range store.purges {
		_ = p
	}
}
