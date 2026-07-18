package store

import (
	"context"
	"fmt"

	"github.com/gsaraiva2109/dietdaemon/core/types"
)

// ---------------------------------------------------------------------------
// Water tracking
// ---------------------------------------------------------------------------

// LogWater inserts a water consumption entry.
func (s *Store) LogWater(ctx context.Context, w types.WaterLog) error {
	const q = `
		INSERT INTO water_logs (id, user_id, amount_ml, logged_at, note, created_at)
		VALUES (?, ?, ?, ?, ?, ?)
	`
	_, err := s.db.ExecContext(ctx, s.rewrite(q), w.ID, w.UserID, w.AmountML, w.LoggedAt, nullStr(w.Note), utcNow())
	if err != nil {
		return fmt.Errorf("store: log water: %w", err)
	}
	return nil
}

// RestoreWater inserts a water log entry for backup restore. On a
// unique-constraint violation (duplicate id — the re-run-safety case), the
// call is a safe no-op and returns nil rather than an error.
func (s *Store) RestoreWater(ctx context.Context, w types.WaterLog) error {
	const q = `
		INSERT INTO water_logs (id, user_id, amount_ml, logged_at, note, created_at)
		VALUES (?, ?, ?, ?, ?, ?)
	`
	_, err := s.db.ExecContext(ctx, s.rewrite(q), w.ID, w.UserID, w.AmountML, w.LoggedAt, nullStr(w.Note), utcNow())
	if err != nil {
		if isUniqueViolation(err) {
			return nil // safe no-op: already restored
		}
		return fmt.Errorf("store: restore water: %w", err)
	}
	return nil
}

// GetWaterInRange returns individual water log rows for a user within a date
// range (inclusive, "YYYY-MM-DD" format), ordered oldest first. Unlike
// GetWaterDailyTotals, rows are not aggregated per day.
func (s *Store) GetWaterInRange(ctx context.Context, userID, startDate, endDate string) ([]types.WaterLog, error) {
	dateExpr := s.dialect.DateTrunc("logged_at")
	q := fmt.Sprintf(`
		SELECT id, user_id, amount_ml, logged_at, COALESCE(note, '') AS note
		FROM water_logs
		WHERE user_id = ? AND %s >= ? AND %s <= ?
		ORDER BY logged_at ASC
	`, dateExpr, dateExpr)
	var out []types.WaterLog
	if err := s.db.SelectContext(ctx, &out, s.rewrite(q), userID, startDate, endDate); err != nil {
		return nil, fmt.Errorf("store: get water in range: %w", err)
	}
	return out, nil
}

// GetWaterToday returns water logs for a specific local date, along with the
// total ml consumed that day.
func (s *Store) GetWaterToday(ctx context.Context, userID, localDate string) ([]types.WaterLog, int, error) {
	q := fmt.Sprintf(`
		SELECT id, user_id, amount_ml, logged_at, COALESCE(note, '') AS note
		FROM water_logs
		WHERE user_id = ? AND %s = ?
		ORDER BY logged_at DESC
	`, s.dialect.DateTrunc("logged_at"))
	var logs []types.WaterLog
	if err := s.db.SelectContext(ctx, &logs, s.rewrite(q), userID, localDate); err != nil {
		return nil, 0, fmt.Errorf("store: get water today: %w", err)
	}
	total := 0
	for _, w := range logs {
		total += w.AmountML
	}
	return logs, total, nil
}

// GetWaterDailyTotals returns per-day water totals between startDate and endDate
// (inclusive, "YYYY-MM-DD" format). Days with no water logs are not returned.
func (s *Store) GetWaterDailyTotals(ctx context.Context, userID, startDate, endDate string) ([]types.WaterDayTotal, error) {
	dateExpr := s.dialect.DateTrunc("logged_at")
	q := fmt.Sprintf(`
		SELECT %s AS date, SUM(amount_ml) AS total_ml
		FROM water_logs
		WHERE user_id = ? AND %s >= ? AND %s <= ?
		GROUP BY %s
		ORDER BY %s ASC
	`, dateExpr, dateExpr, dateExpr, dateExpr, dateExpr)
	var out []types.WaterDayTotal
	if err := s.db.SelectContext(ctx, &out, s.rewrite(q), userID, startDate, endDate); err != nil {
		return nil, fmt.Errorf("store: get water daily totals: %w", err)
	}
	return out, nil
}

// DeleteWater deletes a water log entry by user + ID. Returns ErrNotFound if absent.
func (s *Store) DeleteWater(ctx context.Context, userID, id string) error {
	const q = `DELETE FROM water_logs WHERE id = ? AND user_id = ?`
	res, err := s.db.ExecContext(ctx, s.rewrite(q), id, userID)
	if err != nil {
		return fmt.Errorf("store: delete water: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return types.ErrNotFound
	}
	return nil
}
