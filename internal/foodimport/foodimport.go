package foodimport

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"syscall"
	"time"

	"github.com/gsaraiva2109/dietdaemon/core/ports"
	"github.com/gsaraiva2109/dietdaemon/core/types"
)

// batchSize is the number of rows buffered before flushing to the store.
// Mirrors internal/store's own bulkUpsertChunkSize.
const batchSize = 500

// Store is the subset of store.Store this package needs.
type Store interface {
	BulkUpsertFoods(ctx context.Context, foods []types.FoodMatch) error
}

type fingerprintStore interface {
	GetFoodImportFingerprint(ctx context.Context, source string) (string, error)
	SetFoodImportFingerprint(ctx context.Context, source, fingerprint string) error
}

// SourceFactory rebuilds a source when its local dataset has changed. This is
// needed for TACO, whose constructor reads the file into memory.
type SourceFactory func() (ports.BulkSource, error)

// Runner ticks on a fixed interval, re-syncing every configured bulk source
// into the global foods table. Mirrors internal/backup.Runner's ticker shape.
type Runner struct {
	store      Store
	sources    []ports.BulkSource
	filters    map[string]ports.BulkFilter // keyed by Name()
	localPaths map[string]string           // keyed by Name(); empty means API/embedded mode
	refresh    map[string]SourceFactory
	interval   time.Duration
	log        *slog.Logger
}

// NewWithLocalPaths adds zero-read file identity checks for local bulk files.
// refresh is called after a changed local file is detected, before importing.
func NewWithLocalPaths(store Store, sources []ports.BulkSource, filters map[string]ports.BulkFilter, interval time.Duration, log *slog.Logger, localPaths map[string]string, refresh map[string]SourceFactory) *Runner {
	r := New(store, sources, filters, interval, log)
	r.localPaths = localPaths
	r.refresh = refresh
	return r
}

// New builds a Runner. filters is keyed by each source's Name().
func New(store Store, sources []ports.BulkSource, filters map[string]ports.BulkFilter, interval time.Duration, log *slog.Logger) *Runner {
	if interval <= 0 {
		interval = 24 * time.Hour
	}
	if log == nil {
		log = slog.Default()
	}
	return &Runner{
		store:    store,
		sources:  sources,
		filters:  filters,
		interval: interval,
		log:      log,
	}
}

// Run ticks until ctx is cancelled, running immediately on start and then on
// every interval.
func (r *Runner) Run(ctx context.Context) {
	t := time.NewTicker(r.interval)
	defer t.Stop()
	r.RunOnce(ctx)
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			r.RunOnce(ctx)
		}
	}
}

// RunOnce re-syncs every configured source once. One source's failure is
// logged but doesn't stop the others from running.
func (r *Runner) RunOnce(ctx context.Context) {
	for i, src := range r.sources {
		next, err := r.runFor(ctx, src)
		if err != nil {
			r.log.Error("foodimport: run source", "source", src.Name(), "result", "failed", "err", err)
			continue
		}
		r.sources[i] = next
	}
}

// runFor streams src's bulk results into the store in fixed-size batches.
func (r *Runner) runFor(ctx context.Context, src ports.BulkSource) (ports.BulkSource, error) {
	path := r.localPaths[src.Name()]
	var before string
	if path != "" {
		var err error
		before, err = localFingerprint(path, r.filters[src.Name()])
		if err != nil {
			return src, err
		}
		fs, ok := r.store.(fingerprintStore)
		if !ok {
			return src, errors.New("foodimport: local file import requires fingerprint store")
		}
		previous, err := fs.GetFoodImportFingerprint(ctx, src.Name())
		if err != nil && !errors.Is(err, types.ErrNotFound) {
			return src, fmt.Errorf("get fingerprint: %w", err)
		}
		if err == nil && previous == before {
			r.log.Info("foodimport: skipped", "source", src.Name(), "result", "skipped")
			return src, nil
		}
		if makeSource := r.refresh[src.Name()]; makeSource != nil {
			src, err = makeSource()
			if err != nil {
				return src, fmt.Errorf("refresh source: %w", err)
			}
		}
	}

	batch := make([]types.FoodMatch, 0, batchSize)
	flush := func() error {
		if len(batch) == 0 {
			return nil
		}
		err := r.store.BulkUpsertFoods(ctx, batch)
		batch = batch[:0]
		return err
	}
	err := src.FetchBulk(ctx, r.filters[src.Name()], func(fm types.FoodMatch) error {
		batch = append(batch, fm)
		if len(batch) >= batchSize {
			return flush()
		}
		return nil
	})
	if err != nil {
		return src, err
	}
	if err := flush(); err != nil {
		return src, err
	}
	if path == "" {
		r.log.Info("foodimport: imported", "source", src.Name(), "result", "imported")
		return src, nil
	}

	after, err := localFingerprint(path, r.filters[src.Name()])
	if err != nil || after != before {
		r.log.Warn("foodimport: changed during import", "source", src.Name(), "result", "changed_during_import")
		return src, nil
	}
	if err := r.store.(fingerprintStore).SetFoodImportFingerprint(ctx, src.Name(), before); err != nil {
		return src, fmt.Errorf("set fingerprint: %w", err)
	}
	r.log.Info("foodimport: imported", "source", src.Name(), "result", "imported")
	return src, nil
}

func localFingerprint(path string, filter ports.BulkFilter) (string, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", fmt.Errorf("absolute path: %w", err)
	}
	info, err := os.Stat(abs)
	if err != nil {
		return "", fmt.Errorf("stat %q: %w", abs, err)
	}
	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok {
		return "", fmt.Errorf("stat %q: Linux file identity unavailable", abs)
	}
	data, err := json.Marshal(struct {
		Path       string
		Device     uint64
		Inode      uint64
		Size       int64
		ModifiedNS int64
		Filter     ports.BulkFilter
	}{abs, uint64(stat.Dev), stat.Ino, info.Size(), info.ModTime().UnixNano(), filter})
	if err != nil {
		return "", fmt.Errorf("encode fingerprint: %w", err)
	}
	return string(data), nil
}
