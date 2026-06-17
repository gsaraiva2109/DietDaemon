// Package pending implements ports.PendingStore in memory: the short-lived
// conversational state for meals awaiting a portion or correction. State is held
// per user behind a mutex and expires after a TTL, which suits the
// "short-lived" requirement and keeps the always-on footprint tiny. A durable
// SQLite-backed implementation can follow behind the same port without touching
// the pipeline.
package pending

import (
	"context"
	"sync"
	"time"

	"github.com/gsaraiva2109/dietdaemon/core/ports"
	"github.com/gsaraiva2109/dietdaemon/core/types"
)

// Store is an in-memory, TTL-expiring PendingStore.
type Store struct {
	mu  sync.Mutex
	ttl time.Duration
	now func() time.Time
	m   map[string]types.PendingMeal
}

var _ ports.PendingStore = (*Store)(nil)

// New returns a pending store that expires entries older than ttl. A non-positive
// ttl disables expiry.
func New(ttl time.Duration) *Store {
	return &Store{
		ttl: ttl,
		now: time.Now,
		m:   make(map[string]types.PendingMeal),
	}
}

// Save stores (replacing any existing) the pending meal for pm.UserID.
func (s *Store) Save(_ context.Context, pm types.PendingMeal) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.m[pm.UserID] = pm
	return nil
}

// Get returns the live pending meal for userID, or types.ErrNotFound when none
// exists or it has expired (expired entries are dropped lazily).
func (s *Store) Get(_ context.Context, userID string) (types.PendingMeal, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	pm, ok := s.m[userID]
	if !ok {
		return types.PendingMeal{}, types.ErrNotFound
	}
	if s.expired(pm) {
		delete(s.m, userID)
		return types.PendingMeal{}, types.ErrNotFound
	}
	return pm, nil
}

// Delete removes any pending meal for userID. It is idempotent.
func (s *Store) Delete(_ context.Context, userID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.m, userID)
	return nil
}

func (s *Store) expired(pm types.PendingMeal) bool {
	if s.ttl <= 0 {
		return false
	}
	return s.now().Sub(pm.CreatedAt) > s.ttl
}
