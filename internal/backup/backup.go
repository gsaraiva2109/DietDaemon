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
	GetPhotoData(ctx context.Context, photoID string) (types.ProgressPhoto, error)
	GetWaterInRange(ctx context.Context, userID, startDate, endDate string) ([]types.WaterLog, error)
	GetWorkoutsInRangeWithExercises(ctx context.Context, userID, startDate, endDate string) ([]types.Workout, error)
}

// Destination abstracts where a backup file goes. cfg carries the per-user
// destination fields (local_subdir, or s3 bucket/prefix/region/endpoint) so
// implementations can honor a config that differs per user without any
// per-user credential storage.
type Destination interface {
	Write(ctx context.Context, cfg types.BackupConfig, filename string, data []byte) error
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
		cfg, err := r.store.GetBackupConfig(ctx, u.ID)
		if errors.Is(err, types.ErrNotFound) {
			continue // no config == disabled
		}
		if err != nil {
			r.log.Error("backup: get config", "user", u.ID, "err", err)
			continue
		}
		if !cfg.Enabled || !r.due(cfg, now) {
			continue
		}
		if err := r.runFor(ctx, u.ID, cfg, now); err != nil {
			r.log.Error("backup: run", "user", u.ID, "err", err)
		}
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
// Returns types.ErrNotFound if the user has no backup_config.
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
	// Photos last: write every blob before the index, so a recovered
	// photos.csv never references a blob that isn't actually there.
	for _, p := range photos {
		full, err := r.store.GetPhotoData(ctx, p.ID)
		if err != nil {
			return fmt.Errorf("backup: load photo data: %w", err)
		}
		if err := dst.Write(ctx, cfg, exportfmt.PhotoFilename(p.ID), full.Data); err != nil {
			return fmt.Errorf("backup: write photo blob: %w", err)
		}
	}
	return writeCSV(ctx, dst, cfg, "photos", "photos.csv", func() ([]types.ProgressPhoto, error) { return photos, nil }, exportfmt.WritePhotosCSV)
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
