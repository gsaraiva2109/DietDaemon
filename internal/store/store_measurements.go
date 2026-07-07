package store

import (
	"context"
	"fmt"
	"time"

	"github.com/gsaraiva2109/dietdaemon/core/types"
	"github.com/jmoiron/sqlx"
)

// ListMeasurements returns measurement entries for the last N days.
func (s *Store) ListMeasurements(ctx context.Context, userID string, days int) ([]types.MeasurementEntry, error) {
	cutoff := time.Now().AddDate(0, 0, -days).Format("2006-01-02")
	const q = `
		SELECT id, user_id, date, waist_cm, hips_cm, chest_cm, left_arm_cm, right_arm_cm,
		       left_thigh_cm, right_thigh_cm, note, created_at
		FROM measurement_log
		WHERE user_id = ? AND date >= ?
		ORDER BY date ASC
	`
	var rows []measurementRow
	if err := s.db.SelectContext(ctx, &rows, s.rewrite(q), userID, cutoff); err != nil {
		return nil, fmt.Errorf("store: list measurements: %w", err)
	}
	out := make([]types.MeasurementEntry, len(rows))
	for i, r := range rows {
		out[i] = r.toMeasurementEntry()
	}
	return out, nil
}

// LogMeasurement inserts or updates a measurement entry.
func (s *Store) LogMeasurement(ctx context.Context, m types.MeasurementEntry) error {
	const q = `
		INSERT INTO measurement_log
			(id, user_id, date, waist_cm, hips_cm, chest_cm, left_arm_cm, right_arm_cm,
			 left_thigh_cm, right_thigh_cm, note, created_at)
		VALUES (:id, :user_id, :date, :waist_cm, :hips_cm, :chest_cm, :left_arm_cm, :right_arm_cm,
			:left_thigh_cm, :right_thigh_cm, :note, :created_at)
		ON CONFLICT(id) DO UPDATE SET
			date           = excluded.date,
			waist_cm       = excluded.waist_cm,
			hips_cm        = excluded.hips_cm,
			chest_cm       = excluded.chest_cm,
			left_arm_cm    = excluded.left_arm_cm,
			right_arm_cm   = excluded.right_arm_cm,
			left_thigh_cm  = excluded.left_thigh_cm,
			right_thigh_cm = excluded.right_thigh_cm,
			note           = excluded.note
	`
	query, args, err := sqlx.Named(q, map[string]any{
		"id": m.ID, "user_id": m.UserID, "date": m.Date,
		"waist_cm": m.WaistCm, "hips_cm": m.HipsCm, "chest_cm": m.ChestCm,
		"left_arm_cm": m.LeftArmCm, "right_arm_cm": m.RightArmCm,
		"left_thigh_cm": m.LeftThighCm, "right_thigh_cm": m.RightThighCm,
		"note": m.Note, "created_at": utcStr(m.CreatedAt),
	})
	if err != nil {
		return fmt.Errorf("store: bind log measurement: %w", err)
	}
	_, err = s.db.ExecContext(ctx, s.rewrite(query), args...)
	return err
}

// DeleteMeasurement deletes a measurement entry by user + ID. Returns ErrNotFound.
func (s *Store) DeleteMeasurement(ctx context.Context, userID, entryID string) error {
	const q = `DELETE FROM measurement_log WHERE id = ? AND user_id = ?`
	res, err := s.db.ExecContext(ctx, s.rewrite(q), entryID, userID)
	if err != nil {
		return fmt.Errorf("store: delete measurement: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return types.ErrNotFound
	}
	return nil
}

// measurementRow is the flat DB shape of measurement_log; types.MeasurementEntry
// parses CreatedAt from the stored RFC3339 string.
type measurementRow struct {
	ID           string  `db:"id"`
	UserID       string  `db:"user_id"`
	Date         string  `db:"date"`
	WaistCm      float64 `db:"waist_cm"`
	HipsCm       float64 `db:"hips_cm"`
	ChestCm      float64 `db:"chest_cm"`
	LeftArmCm    float64 `db:"left_arm_cm"`
	RightArmCm   float64 `db:"right_arm_cm"`
	LeftThighCm  float64 `db:"left_thigh_cm"`
	RightThighCm float64 `db:"right_thigh_cm"`
	Note         string  `db:"note"`
	CreatedAt    string  `db:"created_at"`
}

func (r measurementRow) toMeasurementEntry() types.MeasurementEntry {
	return types.MeasurementEntry{
		ID: r.ID, UserID: r.UserID, Date: r.Date,
		WaistCm: r.WaistCm, HipsCm: r.HipsCm, ChestCm: r.ChestCm,
		LeftArmCm: r.LeftArmCm, RightArmCm: r.RightArmCm,
		LeftThighCm: r.LeftThighCm, RightThighCm: r.RightThighCm,
		Note: r.Note, CreatedAt: parseUTC(r.CreatedAt),
	}
}
