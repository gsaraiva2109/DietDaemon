// Package pendingstore implements ports.PendingStore backed by SQLite.
// It mirrors the behaviour of internal/pending (one pending meal per user,
// lazy TTL expiry) but persists to a BLOB table so open clarification
// survives a process restart.
//
// Design: the whole PendingMeal is JSON-marshalled into a single BLOB column.
// created_at (Unix seconds) is duplicated so expiry can be evaluated without
// unmarshalling the payload.
package pendingstore

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	_ "modernc.org/sqlite"

	"github.com/gsaraiva2109/dietdaemon/core/ports"
	"github.com/gsaraiva2109/dietdaemon/core/types"
)

// Store implements ports.PendingStore with a SQLite-backed durable store.
type Store struct {
	db  *sql.DB
	ttl time.Duration
	now func() time.Time
}

var _ ports.PendingStore = (*Store)(nil)

// New returns a SQLite-backed PendingStore that expires entries older than ttl.
// A non-positive ttl disables expiry (entries live forever). The caller is
// responsible for running migrations before using the store.
func New(db *sql.DB, ttl time.Duration) *Store {
	return &Store{
		db:  db,
		ttl: ttl,
		now: time.Now,
	}
}

// Save stores (replacing any existing) the pending meal for pm.UserID.
func (s *Store) Save(ctx context.Context, pm types.PendingMeal) error {
	payload, err := json.Marshal(pm)
	if err != nil {
		return fmt.Errorf("pendingstore: marshal: %w", err)
	}

	const q = `
		INSERT INTO pending_state (user_id, created_at, payload)
		VALUES (?, ?, ?)
		ON CONFLICT(user_id) DO UPDATE SET
			created_at = excluded.created_at,
			payload    = excluded.payload
	`
	_, err = s.db.ExecContext(ctx, q,
		pm.UserID, pm.CreatedAt.Unix(), payload,
	)
	if err != nil {
		return fmt.Errorf("pendingstore: save: %w", err)
	}
	return nil
}

// Get returns the live pending meal for userID, or types.ErrNotFound when none
// exists or it has expired (expired rows are deleted lazily).
func (s *Store) Get(ctx context.Context, userID string) (types.PendingMeal, error) {
	const q = `SELECT created_at, payload FROM pending_state WHERE user_id = ?`
	row := s.db.QueryRowContext(ctx, q, userID)

	var createdUnix int64
	var raw []byte
	err := row.Scan(&createdUnix, &raw)
	if errors.Is(err, sql.ErrNoRows) {
		return types.PendingMeal{}, types.ErrNotFound
	}
	if err != nil {
		return types.PendingMeal{}, fmt.Errorf("pendingstore: get: %w", err)
	}

	createdAt := time.Unix(createdUnix, 0)
	if s.expired(createdAt) {
		// Lazy delete — same semantic as the in-memory impl.
		_ = s.deleteRow(ctx, userID)
		return types.PendingMeal{}, types.ErrNotFound
	}

	var pm types.PendingMeal
	if err := json.Unmarshal(raw, &pm); err != nil {
		return types.PendingMeal{}, fmt.Errorf("pendingstore: unmarshal: %w", err)
	}
	return pm, nil
}

// Delete removes any pending meal for userID. It is idempotent.
func (s *Store) Delete(ctx context.Context, userID string) error {
	return s.deleteRow(ctx, userID)
}

func (s *Store) deleteRow(ctx context.Context, userID string) error {
	const q = `DELETE FROM pending_state WHERE user_id = ?`
	_, err := s.db.ExecContext(ctx, q, userID)
	if err != nil {
		return fmt.Errorf("pendingstore: delete: %w", err)
	}
	return nil
}

func (s *Store) expired(createdAt time.Time) bool {
	if s.ttl <= 0 {
		return false
	}
	return s.now().Sub(createdAt) > s.ttl
}
