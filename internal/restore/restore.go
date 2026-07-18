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

	read := func(filename string) ([]byte, bool) {
		if !has[filename] {
			sum.Skipped = append(sum.Skipped, filename)
			return nil, false
		}
		data, err := r.src.Read(ctx, cfg, filename)
		if err != nil {
			errs = append(errs, fmt.Errorf("restore: read %s: %w", filename, err))
			return nil, false
		}
		return data, true
	}

	if data, ok := read("meals.csv"); ok {
		meals, err := exportfmt.ReadMealsCSV(bytes.NewReader(data))
		if err != nil {
			errs = append(errs, fmt.Errorf("restore: parse meals.csv: %w", err))
		} else {
			for _, m := range meals {
				m.UserID = userID
				if err := r.store.SaveMeal(ctx, m); err != nil {
					errs = append(errs, fmt.Errorf("restore: save meal %s: %w", m.ID, err))
					continue
				}
				sum.Meals++
			}
		}
	}

	if data, ok := read("rollups.csv"); ok {
		rollups, err := exportfmt.ReadRollupsCSV(bytes.NewReader(data))
		if err != nil {
			errs = append(errs, fmt.Errorf("restore: parse rollups.csv: %w", err))
		} else {
			for _, rr := range rollups {
				rr.UserID = userID
				if err := r.store.UpsertRollup(ctx, rr); err != nil {
					errs = append(errs, fmt.Errorf("restore: upsert rollup %s: %w", rr.Date, err))
					continue
				}
				sum.Rollups++
			}
		}
	}

	if data, ok := read("weight.csv"); ok {
		weight, err := exportfmt.ReadWeightCSV(bytes.NewReader(data))
		if err != nil {
			errs = append(errs, fmt.Errorf("restore: parse weight.csv: %w", err))
		} else {
			for _, w := range weight {
				w.UserID = userID
				if _, err := r.store.LogWeight(ctx, w); err != nil {
					errs = append(errs, fmt.Errorf("restore: log weight %s: %w", w.ID, err))
					continue
				}
				sum.Weight++
			}
		}
	}

	if data, ok := read("measurements.csv"); ok {
		measurements, err := exportfmt.ReadMeasurementsCSV(bytes.NewReader(data))
		if err != nil {
			errs = append(errs, fmt.Errorf("restore: parse measurements.csv: %w", err))
		} else {
			for _, m := range measurements {
				m.UserID = userID
				if _, err := r.store.LogMeasurement(ctx, m); err != nil {
					errs = append(errs, fmt.Errorf("restore: log measurement %s: %w", m.ID, err))
					continue
				}
				sum.Measurements++
			}
		}
	}

	if data, ok := read("sleep.csv"); ok {
		sleep, err := exportfmt.ReadSleepCSV(bytes.NewReader(data))
		if err != nil {
			errs = append(errs, fmt.Errorf("restore: parse sleep.csv: %w", err))
		} else {
			for _, s := range sleep {
				s.UserID = userID
				if err := r.store.RestoreSleep(ctx, s); err != nil {
					errs = append(errs, fmt.Errorf("restore: restore sleep %s: %w", s.ID, err))
					continue
				}
				sum.Sleep++
			}
		}
	}

	if data, ok := read("workouts.csv"); ok {
		workouts, err := exportfmt.ReadWorkoutsCSV(bytes.NewReader(data))
		if err != nil {
			errs = append(errs, fmt.Errorf("restore: parse workouts.csv: %w", err))
		} else {
			for _, w := range workouts {
				w.UserID = userID
				if err := r.store.ImportWorkout(ctx, w); err != nil {
					errs = append(errs, fmt.Errorf("restore: import workout %s: %w", w.ID, err))
					continue
				}
				sum.Workouts++
			}
		}
	}

	if data, ok := read("water.csv"); ok {
		water, err := exportfmt.ReadWaterCSV(bytes.NewReader(data))
		if err != nil {
			errs = append(errs, fmt.Errorf("restore: parse water.csv: %w", err))
		} else {
			for _, w := range water {
				w.UserID = userID
				if err := r.store.RestoreWater(ctx, w); err != nil {
					errs = append(errs, fmt.Errorf("restore: restore water %s: %w", w.ID, err))
					continue
				}
				sum.Water++
			}
		}
	}

	if data, ok := read("fasts.csv"); ok {
		fasts, err := exportfmt.ReadFastsCSV(bytes.NewReader(data))
		if err != nil {
			errs = append(errs, fmt.Errorf("restore: parse fasts.csv: %w", err))
		} else {
			for _, f := range fasts {
				f.UserID = userID
				if err := r.store.RestoreFast(ctx, f); err != nil {
					errs = append(errs, fmt.Errorf("restore: restore fast %s: %w", f.ID, err))
					continue
				}
				sum.Fasts++
			}
		}
	}

	// Photos last: each row needs its blob read separately, so a missing
	// blob only skips that one photo instead of aborting the index.
	if data, ok := read("photos.csv"); ok {
		index, err := exportfmt.ReadPhotosCSV(bytes.NewReader(data))
		if err != nil {
			errs = append(errs, fmt.Errorf("restore: parse photos.csv: %w", err))
		} else {
			for _, entry := range index {
				blob, err := r.src.Read(ctx, cfg, entry.Filename)
				if err != nil {
					errs = append(errs, fmt.Errorf("restore: read photo blob %s: %w", entry.Filename, err))
					continue
				}
				entry.Photo.UserID = userID
				entry.Photo.Data = blob
				if err := r.store.RestorePhoto(ctx, entry.Photo); err != nil {
					errs = append(errs, fmt.Errorf("restore: restore photo %s: %w", entry.Photo.ID, err))
					continue
				}
				sum.Photos++
			}
		}
	}

	return sum, errors.Join(errs...)
}
