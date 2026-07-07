package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/gsaraiva2109/dietdaemon/core/types"
	"github.com/jmoiron/sqlx"
)

// rollupRow is the flat DB shape of daily_rollups; types.DailyRollup groups
// the consumed/target columns into nested Macros structs.
type rollupRow struct {
	UserID          string  `db:"user_id"`
	Date            string  `db:"date"`
	ConsumedKcal    float64 `db:"consumed_kcal"`
	ConsumedProtein float64 `db:"consumed_protein"`
	ConsumedCarbs   float64 `db:"consumed_carbs"`
	ConsumedFat     float64 `db:"consumed_fat"`
	ConsumedFiber   float64 `db:"consumed_fiber"`
	TargetKcal      float64 `db:"target_kcal"`
	TargetProtein   float64 `db:"target_protein"`
	TargetCarbs     float64 `db:"target_carbs"`
	TargetFat       float64 `db:"target_fat"`
	TargetFiber     float64 `db:"target_fiber"`
}

func (r rollupRow) toRollup() types.DailyRollup {
	return types.DailyRollup{
		UserID: r.UserID,
		Date:   r.Date,
		Consumed: types.Macros{
			Calories: r.ConsumedKcal, Protein: r.ConsumedProtein, Carbs: r.ConsumedCarbs,
			Fat: r.ConsumedFat, Fiber: r.ConsumedFiber,
		},
		Targets: types.Macros{
			Calories: r.TargetKcal, Protein: r.TargetProtein, Carbs: r.TargetCarbs,
			Fat: r.TargetFat, Fiber: r.TargetFiber,
		},
	}
}

// ---------------------------------------------------------------------------
// Rollups
// ---------------------------------------------------------------------------

// GetRollup returns a daily rollup, or types.ErrNotFound.
func (s *Store) GetRollup(ctx context.Context, userID, localDate string) (types.DailyRollup, error) {
	const q = `
		SELECT user_id, date,
		       consumed_kcal, consumed_protein, consumed_carbs, consumed_fat, consumed_fiber,
		       target_kcal, target_protein, target_carbs, target_fat, target_fiber
		FROM daily_rollups
		WHERE user_id = ? AND date = ?
	`
	var row rollupRow
	if err := s.db.GetContext(ctx, &row, s.rewrite(q), userID, localDate); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return types.DailyRollup{}, types.ErrNotFound
		}
		return types.DailyRollup{}, err
	}
	return row.toRollup(), nil
}

// UpsertRollup inserts or replaces a daily rollup row.
func (s *Store) UpsertRollup(ctx context.Context, r types.DailyRollup) error {
	const q = `
		INSERT INTO daily_rollups
			(user_id, date,
			 consumed_kcal, consumed_protein, consumed_carbs, consumed_fat, consumed_fiber,
			 target_kcal, target_protein, target_carbs, target_fat, target_fiber)
		VALUES (:user_id, :date,
		        :consumed_kcal, :consumed_protein, :consumed_carbs, :consumed_fat, :consumed_fiber,
		        :target_kcal, :target_protein, :target_carbs, :target_fat, :target_fiber)
		ON CONFLICT(user_id, date) DO UPDATE SET
			consumed_kcal    = excluded.consumed_kcal,
			consumed_protein = excluded.consumed_protein,
			consumed_carbs   = excluded.consumed_carbs,
			consumed_fat     = excluded.consumed_fat,
			consumed_fiber   = excluded.consumed_fiber,
			target_kcal      = excluded.target_kcal,
			target_protein   = excluded.target_protein,
			target_carbs     = excluded.target_carbs,
			target_fat       = excluded.target_fat,
			target_fiber     = excluded.target_fiber
	`
	query, args, err := sqlx.Named(q, map[string]any{
		"user_id": r.UserID, "date": r.Date,
		"consumed_kcal": r.Consumed.Calories, "consumed_protein": r.Consumed.Protein,
		"consumed_carbs": r.Consumed.Carbs, "consumed_fat": r.Consumed.Fat, "consumed_fiber": r.Consumed.Fiber,
		"target_kcal": r.Targets.Calories, "target_protein": r.Targets.Protein,
		"target_carbs": r.Targets.Carbs, "target_fat": r.Targets.Fat, "target_fiber": r.Targets.Fiber,
	})
	if err != nil {
		return fmt.Errorf("store: bind upsert rollup: %w", err)
	}
	_, err = s.db.ExecContext(ctx, s.rewrite(query), args...)
	return err
}
