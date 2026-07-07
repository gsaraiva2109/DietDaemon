package store

import (
	"context"
	"fmt"
	"time"

	"github.com/gsaraiva2109/dietdaemon/core/types"
)

// ---------------------------------------------------------------------------
// Body tracking
// ---------------------------------------------------------------------------

// ListWeight returns weight entries for the last N days.
func (s *Store) ListWeight(ctx context.Context, userID string, days int) ([]types.WeightEntry, error) {
	cutoff := time.Now().AddDate(0, 0, -days).Format("2006-01-02")
	const q = `
		SELECT id, user_id, date, weight_kg, note, created_at
		FROM weight_log
		WHERE user_id = ? AND date >= ?
		ORDER BY date ASC
	`
	var rows []weightRow
	if err := s.db.SelectContext(ctx, &rows, s.rewrite(q), userID, cutoff); err != nil {
		return nil, fmt.Errorf("store: list weight: %w", err)
	}
	out := make([]types.WeightEntry, len(rows))
	for i, r := range rows {
		out[i] = r.toWeightEntry()
	}
	return out, nil
}

// LogWeight inserts or updates a weight entry.
func (s *Store) LogWeight(ctx context.Context, w types.WeightEntry) error {
	const q = `
		INSERT INTO weight_log (id, user_id, date, weight_kg, note, created_at)
		VALUES (?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			date      = excluded.date,
			weight_kg = excluded.weight_kg,
			note      = excluded.note
	`
	_, err := s.db.ExecContext(ctx, s.rewrite(q), w.ID, w.UserID, w.Date, w.WeightKg, w.Note, utcStr(w.CreatedAt))
	return err
}

// DeleteWeight deletes a weight entry by user + ID. Returns ErrNotFound if absent.
func (s *Store) DeleteWeight(ctx context.Context, userID, entryID string) error {
	const q = `DELETE FROM weight_log WHERE id = ? AND user_id = ?`
	res, err := s.db.ExecContext(ctx, s.rewrite(q), entryID, userID)
	if err != nil {
		return fmt.Errorf("store: delete weight: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return types.ErrNotFound
	}
	return nil
}

// WeightTrend returns weight entries with 7-day rolling average for the last N days.
func (s *Store) WeightTrend(ctx context.Context, userID string, days int) ([]types.WeightTrend, error) {
	entries, err := s.ListWeight(ctx, userID, days)
	if err != nil {
		return nil, err
	}
	if len(entries) == 0 {
		return []types.WeightTrend{}, nil
	}

	var trend []types.WeightTrend
	for i, e := range entries {
		wt := types.WeightTrend{Date: e.Date, WeightKg: e.WeightKg}

		// 7-day rolling average.
		start := max(i-6, 0)
		sum := 0.0
		count := 0
		for j := start; j <= i; j++ {
			sum += entries[j].WeightKg
			count++
		}
		if count > 0 {
			wt.RollingAvg = sum / float64(count)
		}
		trend = append(trend, wt)
	}
	return trend, nil
}

// weightRow is the flat DB shape of weight_log; types.WeightEntry parses
// CreatedAt from the stored RFC3339 string.
type weightRow struct {
	ID        string  `db:"id"`
	UserID    string  `db:"user_id"`
	Date      string  `db:"date"`
	WeightKg  float64 `db:"weight_kg"`
	Note      string  `db:"note"`
	CreatedAt string  `db:"created_at"`
}

func (r weightRow) toWeightEntry() types.WeightEntry {
	return types.WeightEntry{
		ID: r.ID, UserID: r.UserID, Date: r.Date, WeightKg: r.WeightKg,
		Note: r.Note, CreatedAt: parseUTC(r.CreatedAt),
	}
}
