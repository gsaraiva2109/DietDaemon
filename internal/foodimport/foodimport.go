package foodimport

import (
	"context"
	"log/slog"
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

// Runner ticks on a fixed interval, re-syncing every configured bulk source
// into the global foods table. Mirrors internal/backup.Runner's ticker shape.
type Runner struct {
	store    Store
	sources  []ports.BulkSource
	filters  map[string]ports.BulkFilter // keyed by Name()
	interval time.Duration
	log      *slog.Logger
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
	for _, src := range r.sources {
		if err := r.runFor(ctx, src); err != nil {
			r.log.Error("foodimport: run source", "source", src.Name(), "err", err)
		}
	}
}

// runFor streams src's bulk results into the store in fixed-size batches.
func (r *Runner) runFor(ctx context.Context, src ports.BulkSource) error {
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
		return err
	}
	return flush()
}
