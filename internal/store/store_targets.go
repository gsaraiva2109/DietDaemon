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
// Targets
// ---------------------------------------------------------------------------

// GetTargets returns the daily targets for a user, or types.ErrNotFound.
func (s *Store) GetTargets(ctx context.Context, userID string) (types.DailyTargets, error) {
	const q = `SELECT user_id, kcal, protein, carbs, fat, fiber FROM daily_targets WHERE user_id = ?`
	var row targetsRow
	if err := s.db.GetContext(ctx, &row, s.rewrite(q), userID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return types.DailyTargets{}, types.ErrNotFound
		}
		return types.DailyTargets{}, err
	}
	return row.toTargets(), nil
}

// targetsRow is the flat DB shape of daily_targets; types.DailyTargets groups
// the macro columns into a nested Macros struct.
type targetsRow struct {
	UserID  string  `db:"user_id"`
	Kcal    float64 `db:"kcal"`
	Protein float64 `db:"protein"`
	Carbs   float64 `db:"carbs"`
	Fat     float64 `db:"fat"`
	Fiber   float64 `db:"fiber"`
}

func (r targetsRow) toTargets() types.DailyTargets {
	return types.DailyTargets{
		UserID:  r.UserID,
		Targets: types.Macros{Calories: r.Kcal, Protein: r.Protein, Carbs: r.Carbs, Fat: r.Fat, Fiber: r.Fiber},
	}
}

// SetTargets inserts or replaces the daily targets row.
func (s *Store) SetTargets(ctx context.Context, t types.DailyTargets) error {
	const q = `
		INSERT INTO daily_targets (user_id, kcal, protein, carbs, fat, fiber)
		VALUES (:user_id, :kcal, :protein, :carbs, :fat, :fiber)
		ON CONFLICT(user_id) DO UPDATE SET
			kcal    = excluded.kcal,
			protein = excluded.protein,
			carbs   = excluded.carbs,
			fat     = excluded.fat,
			fiber   = excluded.fiber
	`
	query, args, err := sqlx.Named(q, map[string]any{
		"user_id": t.UserID,
		"kcal":    t.Targets.Calories, "protein": t.Targets.Protein, "carbs": t.Targets.Carbs,
		"fat": t.Targets.Fat, "fiber": t.Targets.Fiber,
	})
	if err != nil {
		return fmt.Errorf("store: bind set targets: %w", err)
	}
	_, err = s.db.ExecContext(ctx, s.rewrite(query), args...)
	return err
}

// UpdateRollupTargets writes the target columns of a day's rollup (creating the
// row with zero consumption if absent) so a targets change shows immediately on
// the dashboard, which reads targets from the rollup.
func (s *Store) UpdateRollupTargets(ctx context.Context, userID, localDate string, t types.Macros) error {
	const q = `
		INSERT INTO daily_rollups
			(user_id, date,
			 consumed_kcal, consumed_protein, consumed_carbs, consumed_fat, consumed_fiber,
			 target_kcal, target_protein, target_carbs, target_fat, target_fiber)
		VALUES (:user_id, :date, 0, 0, 0, 0, 0, :kcal, :protein, :carbs, :fat, :fiber)
		ON CONFLICT(user_id, date) DO UPDATE SET
			target_kcal    = :kcal,
			target_protein = :protein,
			target_carbs   = :carbs,
			target_fat     = :fat,
			target_fiber   = :fiber
	`
	query, args, err := sqlx.Named(q, map[string]any{
		"user_id": userID, "date": localDate,
		"kcal": t.Calories, "protein": t.Protein, "carbs": t.Carbs, "fat": t.Fat, "fiber": t.Fiber,
	})
	if err != nil {
		return fmt.Errorf("store: bind update rollup targets: %w", err)
	}
	_, err = s.db.ExecContext(ctx, s.rewrite(query), args...)
	return err
}
