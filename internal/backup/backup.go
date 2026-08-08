// Package backup runs scheduled per-user data backups: on a fixed check
// interval it looks for users whose backup_config is enabled and due (based
// on their own interval_hrs and last_run_at), exports their meals and
// rollups as CSV (reusing internal/exportfmt, the same format the on-demand
// REST export uses), and writes the result to their configured destination
// (local disk or S3).
package backup

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"time"

	"github.com/gsaraiva2109/dietdaemon/core/types"
	"github.com/gsaraiva2109/dietdaemon/internal/exportfmt"
)

// earliestDate bounds the "export everything" range query. Meal/rollup
// history realistically never predates this.
const earliestDate = "1970-01-01"

// defaultIntervalHrs is used when a user's backup_config has a non-positive
// interval_hrs (defensive; the store default is 24).
const defaultIntervalHrs = 24

// largeDaysWindow and largeRowLimit stand in for "export everything" on the
// store methods that take a days/limit window instead of a date range.
const (
	largeDaysWindow = 36600 // ~100 years
	largeRowLimit   = 1_000_000
)

// backupCountDropThreshold is the fraction of previously-seen rows below which
// a backup run logs a warning. 0.5 means a >50% drop triggers a warning.
const backupCountDropThreshold = 0.5

// Store is the read/write side a backup run needs. *store.Store satisfies it.
type Store interface {
	ListUsers(ctx context.Context) ([]types.User, error)
	// AccountDeletedAt reports the deletion timestamp of the account owning
	// userID (nil if the account isn't pending deletion), so tick() can
	// exclude accounts pending deletion from backups entirely.
	AccountDeletedAt(ctx context.Context, userID string) (*time.Time, error)
	GetBackupConfig(ctx context.Context, userID string) (types.BackupConfig, error)
	SetBackupLastRun(ctx context.Context, userID string, t time.Time) error
	SetBackupCounts(ctx context.Context, userID string, mealsCount, rollupsCount int) error
	GetMealsInRange(ctx context.Context, userID, startDate, endDate string) ([]types.Meal, error)
	GetRollups(ctx context.Context, userID, startDate, endDate string) ([]types.DailyRollup, error)
	ListWeight(ctx context.Context, userID string, days int) ([]types.WeightEntry, error)
	ListMeasurements(ctx context.Context, userID string, days int) ([]types.MeasurementEntry, error)
	ListSleep(ctx context.Context, userID string, limit int) ([]types.SleepLog, error)
	ListFasts(ctx context.Context, userID string, limit int) ([]types.Fast, error)
	ListPhotoMetadata(ctx context.Context, userID string) ([]types.ProgressPhoto, error)
	GetPhotosData(ctx context.Context, userID string, photoIDs []string) (map[string][]byte, error)
	GetWaterInRange(ctx context.Context, userID, startDate, endDate string) ([]types.WaterLog, error)
	GetWorkoutsInRangeWithExercises(ctx context.Context, userID, startDate, endDate string) ([]types.Workout, error)

	// Diet plans (and the meal_templates their slot options reference).
	ListPlans(ctx context.Context, userID string) ([]types.DietPlan, error)
	GetPlanBundle(ctx context.Context, planID string) (types.PlanBundle, error)
	ListDayOverrides(ctx context.Context, userID string) ([]types.DietPlanDayOverride, error)
	ListTemplatesForBackup(ctx context.Context, userID string) ([]types.MealTemplate, error)
}

// Destination abstracts where a backup file goes. cfg carries the per-user
// destination fields (local_subdir, or s3 bucket/prefix/region/endpoint) so
// implementations can honor a config that differs per user without any
// per-user credential storage.
type Destination interface {
	Write(ctx context.Context, cfg types.BackupConfig, filename string, data []byte) error

	// Delete removes every file previously written for cfg's user (i.e.
	// everything under their local_subdir / S3 prefix). Used when the
	// owning account is purged, so exported CSVs and photo blobs don't
	// outlive the account they came from.
	Delete(ctx context.Context, cfg types.BackupConfig) error
}

// Runner ticks on a fixed interval, independent of any per-user interval_hrs,
// and checks every user for a due backup. Mirrors scheduler.Scheduler's
// ticker shape (internal/scheduler/scheduler.go).
type Runner struct {
	store    Store
	localDst Destination // nil disables the "local" destination
	s3Dst    Destination // nil disables the "s3" destination
	interval time.Duration

	now func() time.Time
	log *slog.Logger
}

// New builds a Runner. localDst or s3Dst may be nil if that destination isn't
// configured/available; a user whose backup_config selects a nil destination
// gets a clear error at run time instead of a boot-time failure, since
// destination choice is a per-user setting, not a global one.
func New(store Store, localDst, s3Dst Destination, checkInterval time.Duration) *Runner {
	if checkInterval <= 0 {
		checkInterval = time.Hour
	}
	return &Runner{
		store:    store,
		localDst: localDst,
		s3Dst:    s3Dst,
		interval: checkInterval,
		now:      time.Now,
		log:      slog.Default(),
	}
}

// Run ticks until ctx is cancelled, checking immediately on start.
func (r *Runner) Run(ctx context.Context) {
	t := time.NewTicker(r.interval)
	defer t.Stop()
	r.tick(ctx)
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			r.tick(ctx)
		}
	}
}

// tick checks every user's backup_config and runs a backup for anyone due.
func (r *Runner) tick(ctx context.Context) {
	users, err := r.store.ListUsers(ctx)
	if err != nil {
		r.log.Error("backup: list users", "err", err)
		return
	}
	now := r.now()
	for _, u := range users {
		r.tickUser(ctx, u.ID, now)
	}
}

// tickUser runs a backup for one user if they're not pending deletion, have
// backup enabled, and are due. Split out of tick so each skip condition adds
// to this method's complexity instead of the loop's.
func (r *Runner) tickUser(ctx context.Context, userID string, now time.Time) {
	deletedAt, err := r.store.AccountDeletedAt(ctx, userID)
	if err != nil && !errors.Is(err, types.ErrNotFound) {
		r.log.Error("backup: deletion status", "user", userID, "err", err)
		return
	}
	if deletedAt != nil {
		return // account pending/completed deletion: excluded from backup entirely
	}

	cfg, err := r.store.GetBackupConfig(ctx, userID)
	if errors.Is(err, types.ErrNotFound) {
		return // no config == disabled
	}
	if err != nil {
		r.log.Error("backup: get config", "user", userID, "err", err)
		return
	}
	if !cfg.Enabled || !r.due(cfg, now) {
		return
	}
	if err := r.runFor(ctx, userID, cfg, now); err != nil {
		r.log.Error("backup: run", "user", userID, "err", err)
	}
}

// due reports whether enough time has passed since cfg.LastRunAt.
func (r *Runner) due(cfg types.BackupConfig, now time.Time) bool {
	if cfg.LastRunAt.IsZero() {
		return true
	}
	hrs := cfg.IntervalHrs
	if hrs <= 0 {
		hrs = defaultIntervalHrs
	}
	return now.Sub(cfg.LastRunAt) >= time.Duration(hrs)*time.Hour
}

// RunOnce runs a backup for one user immediately, ignoring the interval gate.
// It is the shared entry point for both the manual "run now" API endpoint
// and (via runFor) the ticker, so the two never duplicate the export logic.
// Unlike tick(), it does not itself check AccountDeletedAt: the manual path
// relies on the API layer's own deletion gate (internal/api/handler.go's
// wrap) rejecting requests for accounts pending deletion before this is ever
// called. Returns types.ErrNotFound if the user has no backup_config.
func (r *Runner) RunOnce(ctx context.Context, userID string) error {
	cfg, err := r.store.GetBackupConfig(ctx, userID)
	if err != nil {
		return err
	}
	return r.runFor(ctx, userID, cfg, r.now())
}

// runFor performs the actual export + write + last-run update for one user.
func (r *Runner) runFor(ctx context.Context, userID string, cfg types.BackupConfig, now time.Time) error {
	dst, err := r.destinationFor(cfg)
	if err != nil {
		return err
	}

	today := now.Format("2006-01-02")

	meals, err := r.store.GetMealsInRange(ctx, userID, earliestDate, today)
	if err != nil {
		return fmt.Errorf("backup: load meals: %w", err)
	}
	if err := r.writeMeals(ctx, dst, cfg, userID, meals); err != nil {
		return err
	}

	rollups, err := r.store.GetRollups(ctx, userID, earliestDate, today)
	if err != nil {
		return fmt.Errorf("backup: load rollups: %w", err)
	}
	if err := r.writeRollups(ctx, dst, cfg, userID, rollups); err != nil {
		return err
	}

	if err := writeCSV(ctx, dst, cfg, "weight", "weight.csv", func() ([]types.WeightEntry, error) {
		return r.store.ListWeight(ctx, userID, largeDaysWindow)
	}, exportfmt.WriteWeightCSV); err != nil {
		return err
	}

	if err := writeCSV(ctx, dst, cfg, "measurements", "measurements.csv", func() ([]types.MeasurementEntry, error) {
		return r.store.ListMeasurements(ctx, userID, largeDaysWindow)
	}, exportfmt.WriteMeasurementsCSV); err != nil {
		return err
	}

	if err := writeCSV(ctx, dst, cfg, "sleep", "sleep.csv", func() ([]types.SleepLog, error) {
		return r.store.ListSleep(ctx, userID, largeRowLimit)
	}, exportfmt.WriteSleepCSV); err != nil {
		return err
	}

	if err := writeCSV(ctx, dst, cfg, "workouts", "workouts.csv", func() ([]types.Workout, error) {
		return r.store.GetWorkoutsInRangeWithExercises(ctx, userID, earliestDate, today)
	}, exportfmt.WriteWorkoutsCSV); err != nil {
		return err
	}

	if err := writeCSV(ctx, dst, cfg, "water", "water.csv", func() ([]types.WaterLog, error) {
		return r.store.GetWaterInRange(ctx, userID, earliestDate, today)
	}, exportfmt.WriteWaterCSV); err != nil {
		return err
	}

	if err := writeCSV(ctx, dst, cfg, "fasts", "fasts.csv", func() ([]types.Fast, error) {
		return r.store.ListFasts(ctx, userID, largeRowLimit)
	}, exportfmt.WriteFastsCSV); err != nil {
		return err
	}

	if err := r.writePhotos(ctx, dst, cfg, userID); err != nil {
		return err
	}

	if err := r.writePlanData(ctx, dst, cfg, userID); err != nil {
		return err
	}

	if err := r.store.SetBackupLastRun(ctx, userID, now); err != nil {
		return fmt.Errorf("backup: set last run: %w", err)
	}
	if err := r.store.SetBackupCounts(ctx, userID, len(meals), len(rollups)); err != nil {
		return fmt.Errorf("backup: set counts: %w", err)
	}
	return nil
}

func (r *Runner) writeMeals(ctx context.Context, dst Destination, cfg types.BackupConfig, userID string, meals []types.Meal) error {
	r.warnCountDrop(userID, "meals", cfg.LastMealsCount, len(meals))
	return writeCSV(ctx, dst, cfg, "meals", "meals.csv", func() ([]types.Meal, error) { return meals, nil }, exportfmt.WriteMealsCSV)
}

func (r *Runner) writeRollups(ctx context.Context, dst Destination, cfg types.BackupConfig, userID string, rollups []types.DailyRollup) error {
	r.warnCountDrop(userID, "rollups", cfg.LastRollupsCount, len(rollups))
	return writeCSV(ctx, dst, cfg, "rollups", "rollups.csv", func() ([]types.DailyRollup, error) { return rollups, nil }, exportfmt.WriteRollupsCSV)
}

func (r *Runner) warnCountDrop(userID, entity string, previous, current int) {
	if previous > 0 && float64(current) < float64(previous)*(1-backupCountDropThreshold) {
		r.log.Warn("backup: row count dropped significantly", "user", userID, "entity", entity, "previous", previous, "current", current)
	}
}

func writeCSV[T any](ctx context.Context, dst Destination, cfg types.BackupConfig, entity, filename string, load func() ([]T, error), write func(io.Writer, []T) error) error {
	values, err := load()
	if err != nil {
		return fmt.Errorf("backup: load %s: %w", entity, err)
	}
	var buf bytes.Buffer
	if err := write(&buf, values); err != nil {
		return fmt.Errorf("backup: write %s csv: %w", entity, err)
	}
	if err := dst.Write(ctx, cfg, filename, buf.Bytes()); err != nil {
		return fmt.Errorf("backup: write %s: %w", entity, err)
	}
	return nil
}

func (r *Runner) writePhotos(ctx context.Context, dst Destination, cfg types.BackupConfig, userID string) error {
	photos, err := r.store.ListPhotoMetadata(ctx, userID)
	if err != nil {
		return fmt.Errorf("backup: load photos: %w", err)
	}

	ids := make([]string, len(photos))
	for i, p := range photos {
		ids[i] = p.ID
	}
	data, err := r.store.GetPhotosData(ctx, userID, ids)
	if err != nil {
		return fmt.Errorf("backup: load photo data: %w", err)
	}

	// Photos last: write every blob before the index, so a recovered
	// photos.csv never references a blob that isn't actually there.
	for _, p := range photos {
		if err := dst.Write(ctx, cfg, exportfmt.PhotoFilename(p.ID), data[p.ID]); err != nil {
			return fmt.Errorf("backup: write photo blob: %w", err)
		}
	}
	return writeCSV(ctx, dst, cfg, "photos", "photos.csv", func() ([]types.ProgressPhoto, error) { return photos, nil }, exportfmt.WritePhotosCSV)
}

// writePlanData exports diet plans and everything they reference:
// meal_templates first (diet_plan_slot_options.template_id has a foreign key
// to it, so it must exist before options are restored), then plans,
// day-types, slots, and options -- flattened from GetPlanBundle rather than
// queried table-by-table, since a user has at most a handful of plans and
// GetPlanBundle is already the N+1-safe way to load one -- and finally
// per-date overrides.
func (r *Runner) writePlanData(ctx context.Context, dst Destination, cfg types.BackupConfig, userID string) error {
	templates, err := r.store.ListTemplatesForBackup(ctx, userID)
	if err != nil {
		return fmt.Errorf("backup: load templates: %w", err)
	}
	if err := writeCSV(ctx, dst, cfg, "templates", "templates.csv", func() ([]types.MealTemplate, error) { return templates, nil }, exportfmt.WriteTemplatesCSV); err != nil {
		return err
	}

	plans, err := r.store.ListPlans(ctx, userID)
	if err != nil {
		return fmt.Errorf("backup: load plans: %w", err)
	}
	if err := writeCSV(ctx, dst, cfg, "plans", "plans.csv", func() ([]types.DietPlan, error) { return plans, nil }, exportfmt.WritePlansCSV); err != nil {
		return err
	}

	dayTypes, slots, options, err := r.flattenPlanBundles(ctx, plans)
	if err != nil {
		return err
	}
	if err := writeCSV(ctx, dst, cfg, "day_types", "day_types.csv", func() ([]types.DietPlanDayType, error) { return dayTypes, nil }, exportfmt.WriteDayTypesCSV); err != nil {
		return err
	}
	if err := writeCSV(ctx, dst, cfg, "slots", "slots.csv", func() ([]types.DietPlanSlot, error) { return slots, nil }, exportfmt.WriteSlotsCSV); err != nil {
		return err
	}
	if err := writeCSV(ctx, dst, cfg, "slot_options", "slot_options.csv", func() ([]types.DietPlanSlotOption, error) { return options, nil }, exportfmt.WriteSlotOptionsCSV); err != nil {
		return err
	}

	overrides, err := r.store.ListDayOverrides(ctx, userID)
	if err != nil {
		return fmt.Errorf("backup: load day overrides: %w", err)
	}
	return writeCSV(ctx, dst, cfg, "day_overrides", "day_overrides.csv", func() ([]types.DietPlanDayOverride, error) { return overrides, nil }, exportfmt.WriteDayOverridesCSV)
}

// flattenPlanBundles loads each plan's bundle (day-types, slots, options) and
// flattens them into the parallel CSV-ready slices writePlanData exports,
// keeping the nested-loop bundle walk out of that function's own branching.
func (r *Runner) flattenPlanBundles(ctx context.Context, plans []types.DietPlan) ([]types.DietPlanDayType, []types.DietPlanSlot, []types.DietPlanSlotOption, error) {
	var dayTypes []types.DietPlanDayType
	var slots []types.DietPlanSlot
	var options []types.DietPlanSlotOption
	for _, p := range plans {
		bundle, err := r.store.GetPlanBundle(ctx, p.ID)
		if err != nil {
			return nil, nil, nil, fmt.Errorf("backup: load plan bundle %s: %w", p.ID, err)
		}
		for _, dt := range bundle.DayTypes {
			dayTypes = append(dayTypes, dt.DietPlanDayType)
			for _, sl := range dt.Slots {
				slots = append(slots, sl.DietPlanSlot)
				options = append(options, sl.Options...)
			}
		}
	}
	return dayTypes, slots, options, nil
}

func (r *Runner) destinationFor(cfg types.BackupConfig) (Destination, error) {
	switch cfg.Destination {
	case "s3":
		if r.s3Dst == nil {
			return nil, fmt.Errorf("backup: s3 destination not available")
		}
		return r.s3Dst, nil
	case "local", "":
		if r.localDst == nil {
			return nil, fmt.Errorf("backup: local destination not configured (set BACKUP_LOCAL_DIR)")
		}
		return r.localDst, nil
	default:
		return nil, fmt.Errorf("backup: unknown destination %q", cfg.Destination)
	}
}
