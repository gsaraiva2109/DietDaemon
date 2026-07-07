package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/gsaraiva2109/dietdaemon/core/types"
)

// --- Fasting ---

// StartFast inserts a new fasting window. Callers should ensure no active fast
// exists first (see GetActiveFast).
func (s *Store) StartFast(ctx context.Context, f types.Fast) error {
	const q = `
		INSERT INTO fasts (id, user_id, start_at, end_at, target_hours, completed, created_at)
		VALUES (?, ?, ?, NULL, ?, 0, ?)
	`
	_, err := s.db.ExecContext(ctx, s.rewrite(q), f.ID, f.UserID, utcStr(f.StartAt), f.TargetHours, utcStr(f.CreatedAt))
	if err != nil {
		return fmt.Errorf("store: start fast: %w", err)
	}
	return nil
}

// GetActiveFast returns the user's in-progress fast (end_at IS NULL), or
// ErrNotFound if none is active.
func (s *Store) GetActiveFast(ctx context.Context, userID string) (types.Fast, error) {
	const q = `
		SELECT id, user_id, start_at, end_at, target_hours, completed, created_at
		FROM fasts
		WHERE user_id = ? AND end_at IS NULL
		ORDER BY start_at DESC
		LIMIT 1
	`
	var row fastRow
	if err := s.db.GetContext(ctx, &row, s.rewrite(q), userID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return types.Fast{}, types.ErrNotFound
		}
		return types.Fast{}, err
	}
	return row.toFast(), nil
}

// EndFast closes a fasting window by id, marking its end time and completion.
// Returns the updated fast, or ErrNotFound if no matching active fast exists.
func (s *Store) EndFast(ctx context.Context, userID, fastID string, endAt time.Time, completed bool) (types.Fast, error) {
	const q = `
		UPDATE fasts SET end_at = ?, completed = ?
		WHERE id = ? AND user_id = ? AND end_at IS NULL
	`
	res, err := s.db.ExecContext(ctx, s.rewrite(q), utcStr(endAt), boolToInt(completed), fastID, userID)
	if err != nil {
		return types.Fast{}, fmt.Errorf("store: end fast: %w", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return types.Fast{}, types.ErrNotFound
	}
	const sel = `
		SELECT id, user_id, start_at, end_at, target_hours, completed, created_at
		FROM fasts WHERE id = ? AND user_id = ?
	`
	var row fastRow
	if err := s.db.GetContext(ctx, &row, s.rewrite(sel), fastID, userID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return types.Fast{}, types.ErrNotFound
		}
		return types.Fast{}, err
	}
	return row.toFast(), nil
}

// ListFasts returns the user's most recent fasting windows, newest first.
func (s *Store) ListFasts(ctx context.Context, userID string, limit int) ([]types.Fast, error) {
	const q = `
		SELECT id, user_id, start_at, end_at, target_hours, completed, created_at
		FROM fasts
		WHERE user_id = ?
		ORDER BY start_at DESC
		LIMIT ?
	`
	var rows []fastRow
	if err := s.db.SelectContext(ctx, &rows, s.rewrite(q), userID, limit); err != nil {
		return nil, fmt.Errorf("store: list fasts: %w", err)
	}
	out := make([]types.Fast, len(rows))
	for i, r := range rows {
		out[i] = r.toFast()
	}
	return out, nil
}

// fastRow is the flat DB shape of fasts; types.Fast nests EndAt as *time.Time
// (DB: nullable RFC3339 string) and Completed as bool (DB: int).
type fastRow struct {
	ID          string         `db:"id"`
	UserID      string         `db:"user_id"`
	StartAt     string         `db:"start_at"`
	EndAt       sql.NullString `db:"end_at"`
	TargetHours float64        `db:"target_hours"`
	Completed   int            `db:"completed"`
	CreatedAt   string         `db:"created_at"`
}

func (r fastRow) toFast() types.Fast {
	f := types.Fast{
		ID: r.ID, UserID: r.UserID, StartAt: parseUTC(r.StartAt),
		TargetHours: r.TargetHours, Completed: r.Completed != 0, CreatedAt: parseUTC(r.CreatedAt),
	}
	if r.EndAt.Valid && r.EndAt.String != "" {
		f.EndAt = new(parseUTC(r.EndAt.String))
	}
	return f
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}
