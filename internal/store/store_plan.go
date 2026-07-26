package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/gsaraiva2109/dietdaemon/core/types"
	"github.com/jmoiron/sqlx"
)

// ---------------------------------------------------------------------------
// Diet plans
// ---------------------------------------------------------------------------

// planRow is the flat DB shape of diet_plans; cycle_pattern is stored as a
// JSON array of diet_plan_day_types.id and unmarshaled into types.DietPlan.
type planRow struct {
	ID              string `db:"id"`
	UserID          string `db:"user_id"`
	Name            string `db:"name"`
	Notes           string `db:"notes"`
	ValidFrom       string `db:"valid_from"`
	ValidTo         string `db:"valid_to"`
	CyclePattern    string `db:"cycle_pattern"`
	CycleAnchorDate string `db:"cycle_anchor_date"`
	CreatedAt       string `db:"created_at"`
	UpdatedAt       string `db:"updated_at"`
}

func (r planRow) toPlan() (types.DietPlan, error) {
	var pattern []string
	if r.CyclePattern != "" {
		if err := json.Unmarshal([]byte(r.CyclePattern), &pattern); err != nil {
			return types.DietPlan{}, fmt.Errorf("store: unmarshal cycle_pattern: %w", err)
		}
	}
	return types.DietPlan{
		ID: r.ID, UserID: r.UserID, Name: r.Name, Notes: r.Notes,
		ValidFrom: r.ValidFrom, ValidTo: r.ValidTo,
		CyclePattern: pattern, CycleAnchorDate: r.CycleAnchorDate,
		CreatedAt: parseUTC(r.CreatedAt), UpdatedAt: parseUTC(r.UpdatedAt),
	}, nil
}

// CreatePlan inserts a new diet plan and returns it with its generated ID and
// timestamps set.
func (s *Store) CreatePlan(ctx context.Context, p types.DietPlan) (types.DietPlan, error) {
	pattern, err := json.Marshal(p.CyclePattern)
	if err != nil {
		return types.DietPlan{}, fmt.Errorf("store: marshal cycle_pattern: %w", err)
	}
	p.ID = newID()
	p.CreatedAt = parseUTC(utcNow())
	p.UpdatedAt = p.CreatedAt

	const q = `
		INSERT INTO diet_plans
			(id, user_id, name, notes, valid_from, valid_to, cycle_pattern, cycle_anchor_date, created_at, updated_at)
		VALUES (:id, :user_id, :name, :notes, :valid_from, :valid_to, :cycle_pattern, :cycle_anchor_date, :created_at, :updated_at)
	`
	query, args, err := sqlx.Named(q, map[string]any{
		"id": p.ID, "user_id": p.UserID, "name": p.Name, "notes": p.Notes,
		"valid_from": p.ValidFrom, "valid_to": p.ValidTo,
		"cycle_pattern": string(pattern), "cycle_anchor_date": p.CycleAnchorDate,
		"created_at": utcStr(p.CreatedAt), "updated_at": utcStr(p.UpdatedAt),
	})
	if err != nil {
		return types.DietPlan{}, fmt.Errorf("store: bind create plan: %w", err)
	}
	if _, err := s.db.ExecContext(ctx, s.rewrite(query), args...); err != nil {
		return types.DietPlan{}, fmt.Errorf("store: create plan: %w", err)
	}
	return p, nil
}

// GetPlan returns a single plan by ID, or types.ErrNotFound.
func (s *Store) GetPlan(ctx context.Context, planID string) (types.DietPlan, error) {
	const q = `
		SELECT id, user_id, name, notes, valid_from, valid_to, cycle_pattern, cycle_anchor_date, created_at, updated_at
		FROM diet_plans WHERE id = ?
	`
	var row planRow
	if err := s.db.GetContext(ctx, &row, s.rewrite(q), planID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return types.DietPlan{}, types.ErrNotFound
		}
		return types.DietPlan{}, fmt.Errorf("store: get plan: %w", err)
	}
	return row.toPlan()
}

// ListPlans returns every plan belonging to a user, most recent valid_from
// first.
func (s *Store) ListPlans(ctx context.Context, userID string) ([]types.DietPlan, error) {
	const q = `
		SELECT id, user_id, name, notes, valid_from, valid_to, cycle_pattern, cycle_anchor_date, created_at, updated_at
		FROM diet_plans WHERE user_id = ? ORDER BY valid_from DESC
	`
	var rows []planRow
	if err := s.db.SelectContext(ctx, &rows, s.rewrite(q), userID); err != nil {
		return nil, fmt.Errorf("store: list plans: %w", err)
	}
	out := make([]types.DietPlan, 0, len(rows))
	for _, r := range rows {
		p, err := r.toPlan()
		if err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, nil
}

// GetActivePlan returns the plan whose [valid_from, valid_to] range contains
// date (valid_to = "" meaning open-ended), or types.ErrNotFound if none
// governs that date. Plans are not expected to overlap; if they do, the one
// with the latest valid_from wins.
func (s *Store) GetActivePlan(ctx context.Context, userID, date string) (types.DietPlan, error) {
	const q = `
		SELECT id, user_id, name, notes, valid_from, valid_to, cycle_pattern, cycle_anchor_date, created_at, updated_at
		FROM diet_plans
		WHERE user_id = ? AND valid_from <= ? AND (valid_to = '' OR valid_to >= ?)
		ORDER BY valid_from DESC
		LIMIT 1
	`
	var row planRow
	if err := s.db.GetContext(ctx, &row, s.rewrite(q), userID, date, date); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return types.DietPlan{}, types.ErrNotFound
		}
		return types.DietPlan{}, fmt.Errorf("store: get active plan: %w", err)
	}
	return row.toPlan()
}

// UpdatePlan overwrites a plan's mutable fields and bumps updated_at. Returns
// types.ErrNotFound if no plan matches id + user_id.
func (s *Store) UpdatePlan(ctx context.Context, p types.DietPlan) error {
	pattern, err := json.Marshal(p.CyclePattern)
	if err != nil {
		return fmt.Errorf("store: marshal cycle_pattern: %w", err)
	}
	const q = `
		UPDATE diet_plans SET
			name = :name, notes = :notes, valid_from = :valid_from, valid_to = :valid_to,
			cycle_pattern = :cycle_pattern, cycle_anchor_date = :cycle_anchor_date, updated_at = :updated_at
		WHERE id = :id AND user_id = :user_id
	`
	query, args, err := sqlx.Named(q, map[string]any{
		"id": p.ID, "user_id": p.UserID, "name": p.Name, "notes": p.Notes,
		"valid_from": p.ValidFrom, "valid_to": p.ValidTo,
		"cycle_pattern": string(pattern), "cycle_anchor_date": p.CycleAnchorDate,
		"updated_at": utcNow(),
	})
	if err != nil {
		return fmt.Errorf("store: bind update plan: %w", err)
	}
	res, err := s.db.ExecContext(ctx, s.rewrite(query), args...)
	if err != nil {
		return fmt.Errorf("store: update plan: %w", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return types.ErrNotFound
	}
	return nil
}

// DeletePlan deletes a plan and all its day-types/slots/options/overrides
// (via ON DELETE CASCADE). Returns types.ErrNotFound if no plan matches.
func (s *Store) DeletePlan(ctx context.Context, userID, planID string) error {
	const q = `DELETE FROM diet_plans WHERE id = ? AND user_id = ?`
	res, err := s.db.ExecContext(ctx, s.rewrite(q), planID, userID)
	if err != nil {
		return fmt.Errorf("store: delete plan: %w", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return types.ErrNotFound
	}
	return nil
}

// ---------------------------------------------------------------------------
// Day types
// ---------------------------------------------------------------------------

type dayTypeRow struct {
	ID          string  `db:"id"`
	PlanID      string  `db:"plan_id"`
	Name        string  `db:"name"`
	Position    int     `db:"position"`
	Kcal        float64 `db:"kcal"`
	Protein     float64 `db:"protein"`
	Carbs       float64 `db:"carbs"`
	Fat         float64 `db:"fat"`
	Fiber       float64 `db:"fiber"`
	WaterGoalMl int     `db:"water_goal_ml"`
}

func (r dayTypeRow) toDayType() types.DietPlanDayType {
	return types.DietPlanDayType{
		ID: r.ID, PlanID: r.PlanID, Name: r.Name, Position: r.Position,
		Targets:     types.Macros{Calories: r.Kcal, Protein: r.Protein, Carbs: r.Carbs, Fat: r.Fat, Fiber: r.Fiber},
		WaterGoalMl: r.WaterGoalMl,
	}
}

// CreateDayType inserts a day-type and returns it with its generated ID set.
func (s *Store) CreateDayType(ctx context.Context, dt types.DietPlanDayType) (types.DietPlanDayType, error) {
	dt.ID = newID()
	const q = `
		INSERT INTO diet_plan_day_types (id, plan_id, name, position, kcal, protein, carbs, fat, fiber, water_goal_ml)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`
	_, err := s.db.ExecContext(ctx, s.rewrite(q),
		dt.ID, dt.PlanID, dt.Name, dt.Position,
		dt.Targets.Calories, dt.Targets.Protein, dt.Targets.Carbs, dt.Targets.Fat, dt.Targets.Fiber,
		dt.WaterGoalMl)
	if err != nil {
		return types.DietPlanDayType{}, fmt.Errorf("store: create day type: %w", err)
	}
	return dt, nil
}

// GetDayType returns a single day-type by ID, or types.ErrNotFound.
func (s *Store) GetDayType(ctx context.Context, dayTypeID string) (types.DietPlanDayType, error) {
	const q = `
		SELECT id, plan_id, name, position, kcal, protein, carbs, fat, fiber, water_goal_ml
		FROM diet_plan_day_types WHERE id = ?
	`
	var row dayTypeRow
	if err := s.db.GetContext(ctx, &row, s.rewrite(q), dayTypeID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return types.DietPlanDayType{}, types.ErrNotFound
		}
		return types.DietPlanDayType{}, fmt.Errorf("store: get day type: %w", err)
	}
	return row.toDayType(), nil
}

// UpdateDayType overwrites a day-type's fields. Returns types.ErrNotFound if
// no row matches id + plan_id.
func (s *Store) UpdateDayType(ctx context.Context, dt types.DietPlanDayType) error {
	const q = `
		UPDATE diet_plan_day_types SET
			name = ?, position = ?, kcal = ?, protein = ?, carbs = ?, fat = ?, fiber = ?, water_goal_ml = ?
		WHERE id = ? AND plan_id = ?
	`
	res, err := s.db.ExecContext(ctx, s.rewrite(q),
		dt.Name, dt.Position, dt.Targets.Calories, dt.Targets.Protein, dt.Targets.Carbs, dt.Targets.Fat, dt.Targets.Fiber,
		dt.WaterGoalMl, dt.ID, dt.PlanID)
	if err != nil {
		return fmt.Errorf("store: update day type: %w", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return types.ErrNotFound
	}
	return nil
}

// DeleteDayType deletes a day-type and its slots/options/overrides (via ON
// DELETE CASCADE). Returns types.ErrNotFound if no row matches.
func (s *Store) DeleteDayType(ctx context.Context, dayTypeID string) error {
	const q = `DELETE FROM diet_plan_day_types WHERE id = ?`
	res, err := s.db.ExecContext(ctx, s.rewrite(q), dayTypeID)
	if err != nil {
		return fmt.Errorf("store: delete day type: %w", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return types.ErrNotFound
	}
	return nil
}

// ---------------------------------------------------------------------------
// Slots
// ---------------------------------------------------------------------------

type slotRow struct {
	ID        string `db:"id"`
	DayTypeID string `db:"day_type_id"`
	Position  int    `db:"position"`
	TimeOfDay string `db:"time_of_day"`
	Label     string `db:"label"`
}

func (r slotRow) toSlot() types.DietPlanSlot {
	return types.DietPlanSlot{ID: r.ID, DayTypeID: r.DayTypeID, Position: r.Position, TimeOfDay: r.TimeOfDay, Label: r.Label}
}

// CreateSlot inserts a slot and returns it with its generated ID set.
func (s *Store) CreateSlot(ctx context.Context, sl types.DietPlanSlot) (types.DietPlanSlot, error) {
	sl.ID = newID()
	const q = `INSERT INTO diet_plan_slots (id, day_type_id, position, time_of_day, label) VALUES (?, ?, ?, ?, ?)`
	if _, err := s.db.ExecContext(ctx, s.rewrite(q), sl.ID, sl.DayTypeID, sl.Position, sl.TimeOfDay, sl.Label); err != nil {
		return types.DietPlanSlot{}, fmt.Errorf("store: create slot: %w", err)
	}
	return sl, nil
}

// UpdateSlot overwrites a slot's fields. Returns types.ErrNotFound if no row
// matches id + day_type_id.
func (s *Store) UpdateSlot(ctx context.Context, sl types.DietPlanSlot) error {
	const q = `UPDATE diet_plan_slots SET position = ?, time_of_day = ?, label = ? WHERE id = ? AND day_type_id = ?`
	res, err := s.db.ExecContext(ctx, s.rewrite(q), sl.Position, sl.TimeOfDay, sl.Label, sl.ID, sl.DayTypeID)
	if err != nil {
		return fmt.Errorf("store: update slot: %w", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return types.ErrNotFound
	}
	return nil
}

// DeleteSlot deletes a slot and its options (via ON DELETE CASCADE). Returns
// types.ErrNotFound if no row matches.
func (s *Store) DeleteSlot(ctx context.Context, slotID string) error {
	const q = `DELETE FROM diet_plan_slots WHERE id = ?`
	res, err := s.db.ExecContext(ctx, s.rewrite(q), slotID)
	if err != nil {
		return fmt.Errorf("store: delete slot: %w", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return types.ErrNotFound
	}
	return nil
}

// ---------------------------------------------------------------------------
// Slot options
// ---------------------------------------------------------------------------

type slotOptionRow struct {
	ID         string `db:"id"`
	SlotID     string `db:"slot_id"`
	Position   int    `db:"position"`
	Label      string `db:"label"`
	TemplateID string `db:"template_id"`
}

func (r slotOptionRow) toOption() types.DietPlanSlotOption {
	return types.DietPlanSlotOption{ID: r.ID, SlotID: r.SlotID, Position: r.Position, Label: r.Label, TemplateID: r.TemplateID}
}

// CreateSlotOption inserts a slot option and returns it with its generated ID
// set.
func (s *Store) CreateSlotOption(ctx context.Context, opt types.DietPlanSlotOption) (types.DietPlanSlotOption, error) {
	opt.ID = newID()
	const q = `INSERT INTO diet_plan_slot_options (id, slot_id, position, label, template_id) VALUES (?, ?, ?, ?, ?)`
	if _, err := s.db.ExecContext(ctx, s.rewrite(q), opt.ID, opt.SlotID, opt.Position, opt.Label, opt.TemplateID); err != nil {
		return types.DietPlanSlotOption{}, fmt.Errorf("store: create slot option: %w", err)
	}
	return opt, nil
}

// UpdateSlotOption overwrites a slot option's fields. Returns
// types.ErrNotFound if no row matches id + slot_id.
func (s *Store) UpdateSlotOption(ctx context.Context, opt types.DietPlanSlotOption) error {
	const q = `UPDATE diet_plan_slot_options SET position = ?, label = ?, template_id = ? WHERE id = ? AND slot_id = ?`
	res, err := s.db.ExecContext(ctx, s.rewrite(q), opt.Position, opt.Label, opt.TemplateID, opt.ID, opt.SlotID)
	if err != nil {
		return fmt.Errorf("store: update slot option: %w", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return types.ErrNotFound
	}
	return nil
}

// DeleteSlotOption deletes a slot option. Returns types.ErrNotFound if no row
// matches. It does not delete the backing meal_templates row; the caller owns
// that lifecycle since a template may still hold adherence history.
func (s *Store) DeleteSlotOption(ctx context.Context, optionID string) error {
	const q = `DELETE FROM diet_plan_slot_options WHERE id = ?`
	res, err := s.db.ExecContext(ctx, s.rewrite(q), optionID)
	if err != nil {
		return fmt.Errorf("store: delete slot option: %w", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return types.ErrNotFound
	}
	return nil
}

// ---------------------------------------------------------------------------
// Day overrides
// ---------------------------------------------------------------------------

// SetDayOverride pins a date to a day-type, upserting on (user_id, date).
func (s *Store) SetDayOverride(ctx context.Context, o types.DietPlanDayOverride) error {
	const q = `
		INSERT INTO diet_plan_day_overrides (user_id, date, day_type_id)
		VALUES (?, ?, ?)
		ON CONFLICT(user_id, date) DO UPDATE SET day_type_id = excluded.day_type_id
	`
	_, err := s.db.ExecContext(ctx, s.rewrite(q), o.UserID, o.Date, o.DayTypeID)
	if err != nil {
		return fmt.Errorf("store: set day override: %w", err)
	}
	return nil
}

// GetDayOverride returns the override pinned to a date, or types.ErrNotFound
// if the date has none.
func (s *Store) GetDayOverride(ctx context.Context, userID, date string) (types.DietPlanDayOverride, error) {
	const q = `SELECT user_id, date, day_type_id FROM diet_plan_day_overrides WHERE user_id = ? AND date = ?`
	var row types.DietPlanDayOverride
	if err := s.db.GetContext(ctx, &row, s.rewrite(q), userID, date); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return types.DietPlanDayOverride{}, types.ErrNotFound
		}
		return types.DietPlanDayOverride{}, fmt.Errorf("store: get day override: %w", err)
	}
	return row, nil
}

// DeleteDayOverride clears the override for a date, if any. Clearing an
// already-absent override is a no-op success.
func (s *Store) DeleteDayOverride(ctx context.Context, userID, date string) error {
	const q = `DELETE FROM diet_plan_day_overrides WHERE user_id = ? AND date = ?`
	_, err := s.db.ExecContext(ctx, s.rewrite(q), userID, date)
	if err != nil {
		return fmt.Errorf("store: delete day override: %w", err)
	}
	return nil
}

// ListDayOverrides returns every day-type override for a user, for backup
// export. GetDayOverride is scoped to a single date and unsuited to bulk
// listing.
func (s *Store) ListDayOverrides(ctx context.Context, userID string) ([]types.DietPlanDayOverride, error) {
	const q = `SELECT user_id, date, day_type_id FROM diet_plan_day_overrides WHERE user_id = ? ORDER BY date`
	var rows []types.DietPlanDayOverride
	if err := s.db.SelectContext(ctx, &rows, s.rewrite(q), userID); err != nil {
		return nil, fmt.Errorf("store: list day overrides: %w", err)
	}
	return rows, nil
}

// ---------------------------------------------------------------------------
// Restore (disaster recovery)
// ---------------------------------------------------------------------------
//
// Unlike the Create* methods above, which always assign a fresh ID, every
// Restore* method here preserves the ID from the backup row directly -- the
// same convention as RestoreSleep/RestoreWater/RestoreFast elsewhere in this
// store. Restoring in plan -> day-type -> slot -> option order (enforced by
// the caller in internal/restore) keeps every foreign key valid as each
// table is replayed. A duplicate ID (already restored) is a safe no-op.
// RestoreDayOverride has no separate method: SetDayOverride already upserts
// on (user_id, date), which is exactly the idempotent behavior restore needs.

// RestorePlan idempotently inserts a plan, preserving its original ID.
func (s *Store) RestorePlan(ctx context.Context, p types.DietPlan) error {
	pattern, err := json.Marshal(p.CyclePattern)
	if err != nil {
		return fmt.Errorf("store: marshal cycle_pattern: %w", err)
	}
	const q = `
		INSERT INTO diet_plans
			(id, user_id, name, notes, valid_from, valid_to, cycle_pattern, cycle_anchor_date, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`
	_, err = s.db.ExecContext(ctx, s.rewrite(q),
		p.ID, p.UserID, p.Name, p.Notes, p.ValidFrom, p.ValidTo, string(pattern), p.CycleAnchorDate,
		utcStr(p.CreatedAt), utcStr(p.UpdatedAt))
	if err != nil {
		if isUniqueViolation(err) {
			return nil
		}
		return fmt.Errorf("store: restore plan: %w", err)
	}
	return nil
}

// RestoreDayType idempotently inserts a day-type, preserving its original ID.
func (s *Store) RestoreDayType(ctx context.Context, dt types.DietPlanDayType) error {
	const q = `
		INSERT INTO diet_plan_day_types (id, plan_id, name, position, kcal, protein, carbs, fat, fiber, water_goal_ml)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`
	_, err := s.db.ExecContext(ctx, s.rewrite(q), dt.ID, dt.PlanID, dt.Name, dt.Position,
		dt.Targets.Calories, dt.Targets.Protein, dt.Targets.Carbs, dt.Targets.Fat, dt.Targets.Fiber, dt.WaterGoalMl)
	if err != nil {
		if isUniqueViolation(err) {
			return nil
		}
		return fmt.Errorf("store: restore day type: %w", err)
	}
	return nil
}

// RestoreSlot idempotently inserts a slot, preserving its original ID.
func (s *Store) RestoreSlot(ctx context.Context, sl types.DietPlanSlot) error {
	const q = `INSERT INTO diet_plan_slots (id, day_type_id, position, time_of_day, label) VALUES (?, ?, ?, ?, ?)`
	_, err := s.db.ExecContext(ctx, s.rewrite(q), sl.ID, sl.DayTypeID, sl.Position, sl.TimeOfDay, sl.Label)
	if err != nil {
		if isUniqueViolation(err) {
			return nil
		}
		return fmt.Errorf("store: restore slot: %w", err)
	}
	return nil
}

// RestoreSlotOption idempotently inserts a slot option, preserving its
// original ID. Requires the referenced template_id to already exist (foreign
// keys are enforced) -- restore templates before slot options.
func (s *Store) RestoreSlotOption(ctx context.Context, opt types.DietPlanSlotOption) error {
	const q = `INSERT INTO diet_plan_slot_options (id, slot_id, position, label, template_id) VALUES (?, ?, ?, ?, ?)`
	_, err := s.db.ExecContext(ctx, s.rewrite(q), opt.ID, opt.SlotID, opt.Position, opt.Label, opt.TemplateID)
	if err != nil {
		if isUniqueViolation(err) {
			return nil
		}
		return fmt.Errorf("store: restore slot option: %w", err)
	}
	return nil
}

// ---------------------------------------------------------------------------
// Plan bundle
// ---------------------------------------------------------------------------

// GetPlanBundle loads a plan and its full day-type/slot/option tree in a
// constant number of queries (one per level, using WHERE ... IN (...) and
// stitching parent/child rows together in Go) regardless of how many
// day-types, slots, or options the plan has.
func (s *Store) GetPlanBundle(ctx context.Context, planID string) (types.PlanBundle, error) {
	plan, err := s.GetPlan(ctx, planID)
	if err != nil {
		return types.PlanBundle{}, err
	}

	var dtRows []dayTypeRow
	const dtQ = `
		SELECT id, plan_id, name, position, kcal, protein, carbs, fat, fiber, water_goal_ml
		FROM diet_plan_day_types WHERE plan_id = ? ORDER BY position
	`
	if err := s.db.SelectContext(ctx, &dtRows, s.rewrite(dtQ), planID); err != nil {
		return types.PlanBundle{}, fmt.Errorf("store: load day types: %w", err)
	}
	if len(dtRows) == 0 {
		return types.PlanBundle{Plan: plan, DayTypes: []types.DietPlanDayTypeBundle{}}, nil
	}

	dayTypeIDs := make([]string, len(dtRows))
	for i, r := range dtRows {
		dayTypeIDs[i] = r.ID
	}
	slotsByDayType, allSlotIDs, err := s.loadPlanSlots(ctx, dayTypeIDs)
	if err != nil {
		return types.PlanBundle{}, err
	}
	optionsBySlot, err := s.loadPlanSlotOptions(ctx, allSlotIDs)
	if err != nil {
		return types.PlanBundle{}, err
	}

	bundle := types.PlanBundle{Plan: plan, DayTypes: make([]types.DietPlanDayTypeBundle, 0, len(dtRows))}
	for _, dtr := range dtRows {
		slots := slotsByDayType[dtr.ID]
		slotBundles := make([]types.DietPlanSlotBundle, 0, len(slots))
		for _, sl := range slots {
			opts := optionsBySlot[sl.ID]
			if opts == nil {
				opts = []types.DietPlanSlotOption{}
			}
			slotBundles = append(slotBundles, types.DietPlanSlotBundle{DietPlanSlot: sl, Options: opts})
		}
		bundle.DayTypes = append(bundle.DayTypes, types.DietPlanDayTypeBundle{DietPlanDayType: dtr.toDayType(), Slots: slotBundles})
	}
	return bundle, nil
}

// loadPlanSlots fetches every slot for the given day-type IDs, grouped by
// day_type_id, plus the flat list of slot IDs found (for the next level's
// IN clause).
func (s *Store) loadPlanSlots(ctx context.Context, dayTypeIDs []string) (map[string][]types.DietPlanSlot, []string, error) {
	// #nosec G201 -- placeholder expansion is ? only, values passed as args
	q := fmt.Sprintf(`
		SELECT id, day_type_id, position, time_of_day, label
		FROM diet_plan_slots WHERE day_type_id IN (%s) ORDER BY day_type_id, position
	`, s.placeholders(len(dayTypeIDs)))
	args := make([]any, len(dayTypeIDs))
	for i, id := range dayTypeIDs {
		args[i] = id
	}
	var rows []slotRow
	if err := s.db.SelectContext(ctx, &rows, s.rewrite(q), args...); err != nil {
		return nil, nil, fmt.Errorf("store: load plan slots: %w", err)
	}
	byDayType := make(map[string][]types.DietPlanSlot)
	ids := make([]string, 0, len(rows))
	for _, r := range rows {
		byDayType[r.DayTypeID] = append(byDayType[r.DayTypeID], r.toSlot())
		ids = append(ids, r.ID)
	}
	return byDayType, ids, nil
}

// loadPlanSlotOptions fetches every option for the given slot IDs, grouped by
// slot_id.
func (s *Store) loadPlanSlotOptions(ctx context.Context, slotIDs []string) (map[string][]types.DietPlanSlotOption, error) {
	if len(slotIDs) == 0 {
		return nil, nil
	}
	// #nosec G201 -- placeholder expansion is ? only, values passed as args
	q := fmt.Sprintf(`
		SELECT id, slot_id, position, label, template_id
		FROM diet_plan_slot_options WHERE slot_id IN (%s) ORDER BY slot_id, position
	`, s.placeholders(len(slotIDs)))
	args := make([]any, len(slotIDs))
	for i, id := range slotIDs {
		args[i] = id
	}
	var rows []slotOptionRow
	if err := s.db.SelectContext(ctx, &rows, s.rewrite(q), args...); err != nil {
		return nil, fmt.Errorf("store: load plan slot options: %w", err)
	}
	bySlot := make(map[string][]types.DietPlanSlotOption)
	for _, r := range rows {
		bySlot[r.SlotID] = append(bySlot[r.SlotID], r.toOption())
	}
	return bySlot, nil
}

// placeholders returns n dialect-correct placeholders joined by commas, for
// building a WHERE ... IN (...) clause with a dynamic argument count.
func (s *Store) placeholders(n int) string {
	ph := make([]string, n)
	for i := range ph {
		ph[i] = s.dialect.Placeholder(i + 1)
	}
	return strings.Join(ph, ",")
}
