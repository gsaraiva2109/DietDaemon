package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/gsaraiva2109/dietdaemon/core/types"
	"github.com/jmoiron/sqlx"
)

// ---------------------------------------------------------------------------
// Targets
// ---------------------------------------------------------------------------

// GetTargets returns the daily targets for a user, or types.ErrNotFound.
func (s *Store) GetTargets(ctx context.Context, userID string) (types.DailyTargets, error) {
	const q = `SELECT user_id, kcal, protein, carbs, fat, fiber, water_goal_ml FROM daily_targets WHERE user_id = ?`
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
	UserID      string  `db:"user_id"`
	Kcal        float64 `db:"kcal"`
	Protein     float64 `db:"protein"`
	Carbs       float64 `db:"carbs"`
	Fat         float64 `db:"fat"`
	Fiber       float64 `db:"fiber"`
	WaterGoalMl int     `db:"water_goal_ml"`
}

func (r targetsRow) toTargets() types.DailyTargets {
	return types.DailyTargets{
		UserID:      r.UserID,
		Targets:     types.Macros{Calories: r.Kcal, Protein: r.Protein, Carbs: r.Carbs, Fat: r.Fat, Fiber: r.Fiber},
		WaterGoalMl: r.WaterGoalMl,
	}
}

// SetTargets inserts or replaces the daily targets row.
func (s *Store) SetTargets(ctx context.Context, t types.DailyTargets) error {
	const q = `
		INSERT INTO daily_targets (user_id, kcal, protein, carbs, fat, fiber, water_goal_ml)
		VALUES (:user_id, :kcal, :protein, :carbs, :fat, :fiber, :water_goal_ml)
		ON CONFLICT(user_id) DO UPDATE SET
			kcal          = excluded.kcal,
			protein       = excluded.protein,
			carbs         = excluded.carbs,
			fat           = excluded.fat,
			fiber         = excluded.fiber,
			water_goal_ml = excluded.water_goal_ml
	`
	query, args, err := sqlx.Named(q, map[string]any{
		"user_id": t.UserID,
		"kcal":    t.Targets.Calories, "protein": t.Targets.Protein, "carbs": t.Targets.Carbs,
		"fat": t.Targets.Fat, "fiber": t.Targets.Fiber, "water_goal_ml": t.WaterGoalMl,
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

// TargetsFor resolves the targets in effect for userID on localDate: an
// override for that date, else the active plan's cycle pattern indexed by
// date, else the flat daily_targets fallback (users with no plan hit this
// path and see behavior identical to before the diet-plan feature).
func (s *Store) TargetsFor(ctx context.Context, userID, localDate string) (types.DailyTargets, error) {
	dt, ok, err := s.resolveDayType(ctx, userID, localDate)
	if err != nil {
		return types.DailyTargets{}, err
	}
	if ok {
		return types.DailyTargets{UserID: userID, Targets: dt.Targets, WaterGoalMl: dt.WaterGoalMl}, nil
	}
	return s.GetTargets(ctx, userID)
}

// RefreshTodayTargets mirrors today's resolved targets into the rollup when a
// plan mutation, day-type edit, or override change could have moved them.
// It is the single reusable helper for that write: handleSetTargets performs
// its own direct write for the no-plan path (see handler_meals.go), and every
// plan-mutating handler calls this instead of duplicating the resolution.
// A no-op when no override or plan governs today, leaving the caller's own
// (or no) write as the source of truth for today's rollup targets.
func (s *Store) RefreshTodayTargets(ctx context.Context, userID string) error {
	today := time.Now().In(s.userLoc(ctx, userID)).Format(dateLayout)
	dt, ok, err := s.resolveDayType(ctx, userID, today)
	if err != nil {
		return err
	}
	if !ok {
		return nil
	}
	return s.UpdateRollupTargets(ctx, userID, today, dt.Targets)
}

// ResolveDayType is the exported form of resolveDayType, for callers outside
// this package (the bot's /plan command) that need the day-type itself --
// its name and water goal, not just the macros TargetsFor already flattens
// out. Do not duplicate the override/cycle-pattern precedence at a call
// site; call this instead.
func (s *Store) ResolveDayType(ctx context.Context, userID, date string) (types.DietPlanDayType, bool, error) {
	return s.resolveDayType(ctx, userID, date)
}

// resolveDayType returns the day-type governing userID on date: an override
// pinned to that exact date takes precedence, else the active plan's cycle
// pattern indexed by date. ok is false when neither applies, telling the
// caller to fall back to the flat daily_targets row.
func (s *Store) resolveDayType(ctx context.Context, userID, date string) (types.DietPlanDayType, bool, error) {
	if ov, err := s.GetDayOverride(ctx, userID, date); err == nil {
		dt, err := s.GetDayType(ctx, ov.DayTypeID)
		if err != nil {
			return types.DietPlanDayType{}, false, fmt.Errorf("store: resolve day override: %w", err)
		}
		return dt, true, nil
	} else if !errors.Is(err, types.ErrNotFound) {
		return types.DietPlanDayType{}, false, err
	}

	plan, err := s.GetActivePlan(ctx, userID, date)
	if err != nil {
		if errors.Is(err, types.ErrNotFound) {
			return types.DietPlanDayType{}, false, nil
		}
		return types.DietPlanDayType{}, false, err
	}
	if len(plan.CyclePattern) == 0 {
		return types.DietPlanDayType{}, false, nil
	}
	idx, err := cycleIndex(plan.CycleAnchorDate, date, len(plan.CyclePattern))
	if err != nil {
		return types.DietPlanDayType{}, false, err
	}
	dt, err := s.GetDayType(ctx, plan.CyclePattern[idx])
	if err != nil {
		return types.DietPlanDayType{}, false, fmt.Errorf("store: resolve cycle day-type: %w", err)
	}
	return dt, true, nil
}

// cycleIndex returns the Euclidean-modulo index into a cycle pattern of
// length n for date, counted in whole days from anchor. Go's % returns a
// negative result for a negative dividend, so a date before anchor (a
// negative day offset) needs the sign correction below — day -1 must land on
// index n-1, not on -1.
func cycleIndex(anchor, date string, n int) (int, error) {
	a, err := time.Parse(dateLayout, anchor)
	if err != nil {
		return 0, fmt.Errorf("store: parse cycle_anchor_date %q: %w", anchor, err)
	}
	d, err := time.Parse(dateLayout, date)
	if err != nil {
		return 0, fmt.Errorf("store: parse date %q: %w", date, err)
	}
	offset := int(d.Sub(a).Hours() / 24)
	idx := offset % n
	if idx < 0 {
		idx += n
	}
	return idx, nil
}
