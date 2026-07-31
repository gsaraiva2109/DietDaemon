// Package restore is the read-side counterpart to internal/backup: it reads
// a CSV+blob backup (local disk or S3) back off a Source and replays it into
// the store, entity by entity. Every store call it uses is idempotent (safe
// to re-run against a backup that was already restored), and because this is
// disaster-recovery code, a single bad file or row never aborts the rest of
// the run — errors are collected with errors.Join and returned alongside a
// partial Summary.
package restore

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"

	"github.com/gsaraiva2109/dietdaemon/core/types"
	"github.com/gsaraiva2109/dietdaemon/internal/exportfmt"
	"github.com/gsaraiva2109/dietdaemon/internal/store"
)

// Store is the write side a restore run needs. *store.Store satisfies it.
// Every method here is idempotent on the underlying store, so RunOnce is
// safe to call more than once against the same backup.
type Store interface {
	SaveMeal(ctx context.Context, m types.Meal) error
	UpsertRollup(ctx context.Context, r types.DailyRollup) error
	LogWeight(ctx context.Context, w types.WeightEntry) (string, error)
	LogMeasurement(ctx context.Context, m types.MeasurementEntry) (string, error)
	RestoreSleep(ctx context.Context, sl types.SleepLog) error
	ImportWorkout(ctx context.Context, w types.Workout) error
	RestorePhoto(ctx context.Context, p types.ProgressPhoto) error
	RestoreWater(ctx context.Context, w types.WaterLog) error
	RestoreFast(ctx context.Context, f types.Fast) error

	// AccountDeletionStatus reports the account's deletion tier, so RunOnce
	// can refuse to resurrect photos for accounts whose PhotosPurgedAt is set.
	AccountDeletionStatus(ctx context.Context, userID string) (store.AccountDeletionStatus, error)

	// RestoreTemplate Diet plans. Restore order matters: templates before slot options
	// (template_id is a foreign key), plans before day-types before slots
	// before slot options, day-types before day overrides.
	RestoreTemplate(ctx context.Context, t types.MealTemplate) error
	RestorePlan(ctx context.Context, p types.DietPlan) error
	RestoreDayType(ctx context.Context, dt types.DietPlanDayType) error
	RestoreSlot(ctx context.Context, sl types.DietPlanSlot) error
	RestoreSlotOption(ctx context.Context, opt types.DietPlanSlotOption) error
	SetDayOverride(ctx context.Context, o types.DietPlanDayOverride) error
}

// Source abstracts where a backup is read from. Symmetric with
// backup.Destination.Write; satisfied structurally by *localdisk.Dest and
// *s3dest.Dest.
type Source interface {
	List(ctx context.Context, cfg types.BackupConfig) ([]string, error)
	Read(ctx context.Context, cfg types.BackupConfig, filename string) ([]byte, error)
}

// Summary reports how many rows of each entity were restored, plus any
// backup files that were absent (older/partial backups don't always have
// every entity).
type Summary struct {
	Meals        int
	Rollups      int
	Weight       int
	Measurements int
	Sleep        int
	Workouts     int
	Water        int
	Fasts        int
	Photos       int
	Templates    int
	Plans        int
	DayTypes     int
	Slots        int
	SlotOptions  int
	DayOverrides int
	Skipped      []string // filenames absent from the backup
}

// Runner replays a backup into the store.
type Runner struct {
	store Store
	src   Source
	log   *slog.Logger
}

type restoreState struct {
	ctx    context.Context
	cfg    types.BackupConfig
	src    Source
	has    map[string]bool
	userID string
	sum    *Summary
	errs   *[]error
}

// New builds a Runner.
func New(store Store, src Source) *Runner {
	return &Runner{store: store, src: src, log: slog.Default()}
}

// RunOnce restores every entity present in cfg's backup for userID. It never
// aborts early: a file that's missing is recorded in Summary.Skipped, a file
// that fails to parse or a row that fails to write is collected into the
// returned error, and every other entity still runs. Returns the (possibly
// partial) Summary and the joined error, which is nil if nothing failed.
func (r *Runner) RunOnce(ctx context.Context, userID string, cfg types.BackupConfig) (Summary, error) {
	var sum Summary
	var errs []error

	present, err := r.src.List(ctx, cfg)
	if err != nil {
		return sum, fmt.Errorf("restore: list backup files: %w", err)
	}
	has := make(map[string]bool, len(present))
	for _, f := range present {
		has[f] = true
	}
	state := restoreState{ctx: ctx, cfg: cfg, src: r.src, has: has, userID: userID, sum: &sum, errs: &errs}

	sum.Meals = restoreCSV(&state, "meals.csv", "save meal", exportfmt.ReadMealsCSV,
		func(m types.Meal, userID string) types.Meal { m.UserID = userID; return m },
		func(m types.Meal) string { return m.ID }, r.store.SaveMeal)
	sum.Rollups = restoreCSV(&state, "rollups.csv", "upsert rollup", exportfmt.ReadRollupsCSV,
		func(rr types.DailyRollup, userID string) types.DailyRollup { rr.UserID = userID; return rr },
		func(rr types.DailyRollup) string { return rr.Date }, r.store.UpsertRollup)
	sum.Weight = restoreCSV(&state, "weight.csv", "log weight", exportfmt.ReadWeightCSV,
		func(w types.WeightEntry, userID string) types.WeightEntry { w.UserID = userID; return w },
		func(w types.WeightEntry) string { return w.ID }, func(ctx context.Context, w types.WeightEntry) error {
			_, err := r.store.LogWeight(ctx, w)
			return err
		})
	sum.Measurements = restoreCSV(&state, "measurements.csv", "log measurement", exportfmt.ReadMeasurementsCSV,
		func(m types.MeasurementEntry, userID string) types.MeasurementEntry { m.UserID = userID; return m },
		func(m types.MeasurementEntry) string { return m.ID }, func(ctx context.Context, m types.MeasurementEntry) error {
			_, err := r.store.LogMeasurement(ctx, m)
			return err
		})
	sum.Sleep = restoreCSV(&state, "sleep.csv", "restore sleep", exportfmt.ReadSleepCSV,
		func(s types.SleepLog, userID string) types.SleepLog { s.UserID = userID; return s },
		func(s types.SleepLog) string { return s.ID }, r.store.RestoreSleep)
	sum.Workouts = restoreCSV(&state, "workouts.csv", "import workout", exportfmt.ReadWorkoutsCSV,
		func(w types.Workout, userID string) types.Workout { w.UserID = userID; return w },
		func(w types.Workout) string { return w.ID }, r.store.ImportWorkout)
	sum.Water = restoreCSV(&state, "water.csv", "restore water", exportfmt.ReadWaterCSV,
		func(w types.WaterLog, userID string) types.WaterLog { w.UserID = userID; return w },
		func(w types.WaterLog) string { return w.ID }, r.store.RestoreWater)
	sum.Fasts = restoreCSV(&state, "fasts.csv", "restore fast", exportfmt.ReadFastsCSV,
		func(f types.Fast, userID string) types.Fast { f.UserID = userID; return f },
		func(f types.Fast) string { return f.ID }, r.store.RestoreFast)
	status, err := r.store.AccountDeletionStatus(ctx, userID)
	if err != nil && !errors.Is(err, types.ErrNotFound) {
		errs = append(errs, fmt.Errorf("restore: account deletion status: %w", err))
	} else if status.PhotosPurgedAt != nil {
		sum.Skipped = append(sum.Skipped, "photos: purged per retention policy, not restored")
	} else {
		sum.Photos = restorePhotos(&state, r.store)
	}

	// Diet plans, restored in foreign-key order: templates (slot options
	// reference them) -> plans -> day-types -> slots -> slot options ->
	// day overrides (reference day-types).
	sum.Templates = restoreCSV(&state, "templates.csv", "restore template", exportfmt.ReadTemplatesCSV,
		func(t types.MealTemplate, userID string) types.MealTemplate { t.UserID = userID; return t },
		func(t types.MealTemplate) string { return t.ID }, r.store.RestoreTemplate)
	sum.Plans = restoreCSV(&state, "plans.csv", "restore plan", exportfmt.ReadPlansCSV,
		func(p types.DietPlan, userID string) types.DietPlan { p.UserID = userID; return p },
		func(p types.DietPlan) string { return p.ID }, r.store.RestorePlan)
	sum.DayTypes = restoreCSV(&state, "day_types.csv", "restore day type", exportfmt.ReadDayTypesCSV,
		noSetUser[types.DietPlanDayType], func(dt types.DietPlanDayType) string { return dt.ID }, r.store.RestoreDayType)
	sum.Slots = restoreCSV(&state, "slots.csv", "restore slot", exportfmt.ReadSlotsCSV,
		noSetUser[types.DietPlanSlot], func(sl types.DietPlanSlot) string { return sl.ID }, r.store.RestoreSlot)
	sum.SlotOptions = restoreCSV(&state, "slot_options.csv", "restore slot option", exportfmt.ReadSlotOptionsCSV,
		noSetUser[types.DietPlanSlotOption], func(opt types.DietPlanSlotOption) string { return opt.ID }, r.store.RestoreSlotOption)
	sum.DayOverrides = restoreCSV(&state, "day_overrides.csv", "restore day override", exportfmt.ReadDayOverridesCSV,
		func(o types.DietPlanDayOverride, userID string) types.DietPlanDayOverride {
			o.UserID = userID
			return o
		},
		func(o types.DietPlanDayOverride) string { return o.Date }, r.store.SetDayOverride)

	return sum, errors.Join(errs...)
}

// noSetUser is a no-op setUser for restoreCSV entities scoped by a parent ID
// (plan_id, day_type_id, slot_id) rather than carrying their own UserID
// field -- diet_plan_day_types, diet_plan_slots, diet_plan_slot_options.
func noSetUser[T any](t T, _ string) T { return t }

func restoreCSV[T any](state *restoreState, filename, action string, parse func(io.Reader) ([]T, error), setUser func(T, string) T, key func(T) string, save func(context.Context, T) error) int {
	data, skipped, err := readBackupFile(state, filename)
	if skipped {
		state.sum.Skipped = append(state.sum.Skipped, filename)
		return 0
	}
	if err != nil {
		*state.errs = append(*state.errs, err)
		return 0
	}
	rows, err := parse(bytes.NewReader(data))
	if err != nil {
		*state.errs = append(*state.errs, fmt.Errorf("restore: parse %s: %w", filename, err))
		return 0
	}
	count := 0
	for _, row := range rows {
		row = setUser(row, state.userID)
		if err := save(state.ctx, row); err != nil {
			*state.errs = append(*state.errs, fmt.Errorf("restore: %s %s: %w", action, key(row), err))
			continue
		}
		count++
	}
	return count
}

func restorePhotos(state *restoreState, st Store) int {
	data, skipped, err := readBackupFile(state, "photos.csv")
	if skipped {
		state.sum.Skipped = append(state.sum.Skipped, "photos.csv")
		return 0
	}
	if err != nil {
		*state.errs = append(*state.errs, err)
		return 0
	}
	index, err := exportfmt.ReadPhotosCSV(bytes.NewReader(data))
	if err != nil {
		*state.errs = append(*state.errs, fmt.Errorf("restore: parse photos.csv: %w", err))
		return 0
	}
	count := 0
	for _, entry := range index {
		blob, err := state.src.Read(state.ctx, state.cfg, entry.Filename)
		if err != nil {
			*state.errs = append(*state.errs, fmt.Errorf("restore: read photo blob %s: %w", entry.Filename, err))
			continue
		}
		entry.Photo.UserID = state.userID
		entry.Photo.Data = blob
		if err := st.RestorePhoto(state.ctx, entry.Photo); err != nil {
			*state.errs = append(*state.errs, fmt.Errorf("restore: restore photo %s: %w", entry.Photo.ID, err))
			continue
		}
		count++
	}
	return count
}

func readBackupFile(state *restoreState, filename string) ([]byte, bool, error) {
	if !state.has[filename] {
		return nil, true, nil
	}
	data, err := state.src.Read(state.ctx, state.cfg, filename)
	if err != nil {
		return nil, false, fmt.Errorf("restore: read %s: %w", filename, err)
	}
	return data, false, nil
}
