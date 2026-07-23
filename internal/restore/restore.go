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
	Skipped      []string // filenames absent from the backup
}

// Runner replays a backup into the store.
type Runner struct {
	store Store
	src   Source
	log   *slog.Logger
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

	sum.Meals = restoreCSV(ctx, cfg, r.src, has, userID, "meals.csv", "save meal", exportfmt.ReadMealsCSV,
		func(m types.Meal, userID string) types.Meal { m.UserID = userID; return m },
		func(m types.Meal) string { return m.ID }, r.store.SaveMeal, &sum, &errs)
	sum.Rollups = restoreCSV(ctx, cfg, r.src, has, userID, "rollups.csv", "upsert rollup", exportfmt.ReadRollupsCSV,
		func(rr types.DailyRollup, userID string) types.DailyRollup { rr.UserID = userID; return rr },
		func(rr types.DailyRollup) string { return rr.Date }, r.store.UpsertRollup, &sum, &errs)
	sum.Weight = restoreCSV(ctx, cfg, r.src, has, userID, "weight.csv", "log weight", exportfmt.ReadWeightCSV,
		func(w types.WeightEntry, userID string) types.WeightEntry { w.UserID = userID; return w },
		func(w types.WeightEntry) string { return w.ID }, func(ctx context.Context, w types.WeightEntry) error {
			_, err := r.store.LogWeight(ctx, w)
			return err
		}, &sum, &errs)
	sum.Measurements = restoreCSV(ctx, cfg, r.src, has, userID, "measurements.csv", "log measurement", exportfmt.ReadMeasurementsCSV,
		func(m types.MeasurementEntry, userID string) types.MeasurementEntry { m.UserID = userID; return m },
		func(m types.MeasurementEntry) string { return m.ID }, func(ctx context.Context, m types.MeasurementEntry) error {
			_, err := r.store.LogMeasurement(ctx, m)
			return err
		}, &sum, &errs)
	sum.Sleep = restoreCSV(ctx, cfg, r.src, has, userID, "sleep.csv", "restore sleep", exportfmt.ReadSleepCSV,
		func(s types.SleepLog, userID string) types.SleepLog { s.UserID = userID; return s },
		func(s types.SleepLog) string { return s.ID }, r.store.RestoreSleep, &sum, &errs)
	sum.Workouts = restoreCSV(ctx, cfg, r.src, has, userID, "workouts.csv", "import workout", exportfmt.ReadWorkoutsCSV,
		func(w types.Workout, userID string) types.Workout { w.UserID = userID; return w },
		func(w types.Workout) string { return w.ID }, r.store.ImportWorkout, &sum, &errs)
	sum.Water = restoreCSV(ctx, cfg, r.src, has, userID, "water.csv", "restore water", exportfmt.ReadWaterCSV,
		func(w types.WaterLog, userID string) types.WaterLog { w.UserID = userID; return w },
		func(w types.WaterLog) string { return w.ID }, r.store.RestoreWater, &sum, &errs)
	sum.Fasts = restoreCSV(ctx, cfg, r.src, has, userID, "fasts.csv", "restore fast", exportfmt.ReadFastsCSV,
		func(f types.Fast, userID string) types.Fast { f.UserID = userID; return f },
		func(f types.Fast) string { return f.ID }, r.store.RestoreFast, &sum, &errs)
	sum.Photos = restorePhotos(ctx, cfg, r.src, r.store, has, userID, &sum, &errs)

	return sum, errors.Join(errs...)
}

func restoreCSV[T any](ctx context.Context, cfg types.BackupConfig, src Source, has map[string]bool, userID, filename, action string, parse func(io.Reader) ([]T, error), setUser func(T, string) T, key func(T) string, save func(context.Context, T) error, sum *Summary, errs *[]error) int {
	data, skipped, err := readBackupFile(ctx, cfg, src, has, filename)
	if skipped {
		sum.Skipped = append(sum.Skipped, filename)
		return 0
	}
	if err != nil {
		*errs = append(*errs, err)
		return 0
	}
	rows, err := parse(bytes.NewReader(data))
	if err != nil {
		*errs = append(*errs, fmt.Errorf("restore: parse %s: %w", filename, err))
		return 0
	}
	count := 0
	for _, row := range rows {
		row = setUser(row, userID)
		if err := save(ctx, row); err != nil {
			*errs = append(*errs, fmt.Errorf("restore: %s %s: %w", action, key(row), err))
			continue
		}
		count++
	}
	return count
}

func restorePhotos(ctx context.Context, cfg types.BackupConfig, src Source, store Store, has map[string]bool, userID string, sum *Summary, errs *[]error) int {
	data, skipped, err := readBackupFile(ctx, cfg, src, has, "photos.csv")
	if skipped {
		sum.Skipped = append(sum.Skipped, "photos.csv")
		return 0
	}
	if err != nil {
		*errs = append(*errs, err)
		return 0
	}
	index, err := exportfmt.ReadPhotosCSV(bytes.NewReader(data))
	if err != nil {
		*errs = append(*errs, fmt.Errorf("restore: parse photos.csv: %w", err))
		return 0
	}
	count := 0
	for _, entry := range index {
		blob, err := src.Read(ctx, cfg, entry.Filename)
		if err != nil {
			*errs = append(*errs, fmt.Errorf("restore: read photo blob %s: %w", entry.Filename, err))
			continue
		}
		entry.Photo.UserID = userID
		entry.Photo.Data = blob
		if err := store.RestorePhoto(ctx, entry.Photo); err != nil {
			*errs = append(*errs, fmt.Errorf("restore: restore photo %s: %w", entry.Photo.ID, err))
			continue
		}
		count++
	}
	return count
}

func readBackupFile(ctx context.Context, cfg types.BackupConfig, src Source, has map[string]bool, filename string) ([]byte, bool, error) {
	if !has[filename] {
		return nil, true, nil
	}
	data, err := src.Read(ctx, cfg, filename)
	if err != nil {
		return nil, false, fmt.Errorf("restore: read %s: %w", filename, err)
	}
	return data, false, nil
}
