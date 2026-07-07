package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/gsaraiva2109/dietdaemon/core/types"
	"github.com/jmoiron/sqlx"
)

// ---------------------------------------------------------------------------
// Sleep tracking
// ---------------------------------------------------------------------------

// LogSleep inserts a new sleep log entry.
func (s *Store) LogSleep(ctx context.Context, sl types.SleepLog) error {
	const q = `
		INSERT INTO sleep_logs (id, user_id, sleep_at, wake_at, quality, note, created_at)
		VALUES (:id, :user_id, :sleep_at, :wake_at, :quality, :note, :created_at)
	`
	query, args, err := sqlx.Named(q, map[string]any{
		"id": sl.ID, "user_id": sl.UserID, "sleep_at": sl.SleepAt, "wake_at": sl.WakeAt,
		"quality": sl.Quality, "note": nullStr(sl.Note), "created_at": utcNow(),
	})
	if err != nil {
		return fmt.Errorf("store: bind log sleep: %w", err)
	}
	if _, err := s.db.ExecContext(ctx, s.rewrite(query), args...); err != nil {
		return fmt.Errorf("store: log sleep: %w", err)
	}
	return nil
}

// GetActiveSleep returns the user's in-progress sleep (wake_at IS NULL), or
// ErrNotFound if none is active.
func (s *Store) GetActiveSleep(ctx context.Context, userID string) (*types.SleepLog, error) {
	const q = `
		SELECT id, user_id, sleep_at, wake_at, quality, COALESCE(note, '') AS note
		FROM sleep_logs
		WHERE user_id = ? AND wake_at IS NULL
		ORDER BY sleep_at DESC
		LIMIT 1
	`
	var sl types.SleepLog
	if err := s.db.GetContext(ctx, &sl, s.rewrite(q), userID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, types.ErrNotFound
		}
		return nil, fmt.Errorf("store: get active sleep: %w", err)
	}
	return &sl, nil
}

// EndSleep closes a sleep log by setting wake_at and quality. Returns
// ErrNotFound if no matching active sleep log exists.
func (s *Store) EndSleep(ctx context.Context, userID, id, wakeAt, quality string) error {
	const q = `
		UPDATE sleep_logs SET wake_at = ?, quality = ?
		WHERE id = ? AND user_id = ? AND wake_at IS NULL
	`
	res, err := s.db.ExecContext(ctx, s.rewrite(q), wakeAt, quality, id, userID)
	if err != nil {
		return fmt.Errorf("store: end sleep: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return types.ErrNotFound
	}
	return nil
}

// ListSleep returns the user's most recent sleep logs, newest first.
func (s *Store) ListSleep(ctx context.Context, userID string, limit int) ([]types.SleepLog, error) {
	const q = `
		SELECT id, user_id, sleep_at, wake_at, quality, COALESCE(note, '') AS note
		FROM sleep_logs
		WHERE user_id = ?
		ORDER BY sleep_at DESC
		LIMIT ?
	`
	var out []types.SleepLog
	if err := s.db.SelectContext(ctx, &out, s.rewrite(q), userID, limit); err != nil {
		return nil, fmt.Errorf("store: list sleep: %w", err)
	}
	return out, nil
}

// DeleteSleep deletes a sleep log by user + ID. Returns ErrNotFound if absent.
func (s *Store) DeleteSleep(ctx context.Context, userID, id string) error {
	const q = `DELETE FROM sleep_logs WHERE id = ? AND user_id = ?`
	res, err := s.db.ExecContext(ctx, s.rewrite(q), id, userID)
	if err != nil {
		return fmt.Errorf("store: delete sleep: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return types.ErrNotFound
	}
	return nil
}
