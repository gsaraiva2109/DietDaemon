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

// Embedder backfills vector embeddings for catalog foods that don't have one
// yet — satisfied by *embedding.Matcher. Bulk-imported foods (this package's
// whole job) never go through the live resolver's embed-on-write path, so
// without this they'd stay invisible to the Tier 1/2 fuzzy matcher forever.
type Embedder interface {
	BackfillEmbeddings(ctx context.Context, progress func(done, total int, itemErr error)) (embedded, failed int, err error)
}

type fingerprintStore interface {
	GetFoodImportFingerprint(ctx context.Context, source string) (string, error)
	SetFoodImportFingerprint(ctx context.Context, source, fingerprint string) error
	SetFoodImportStatus(ctx context.Context, source, result, lastError string) error
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
	embedder   Embedder
	interval   time.Duration
	log        *slog.Logger
}

// WithEmbedder wires an embedding backfill into the runner: after every
// import pass, any catalog food still missing a vector (fresh from this run
// or left over from an earlier one) gets embedded so it's matchable by the
// Tier 1/2 fuzzy matcher. Optional — nil (the default) skips this step
// entirely, e.g. on Tier 0 where no embedder is configured. Returns r for
// chaining off the New/NewWithLocalPaths call site.
func (r *Runner) WithEmbedder(e Embedder) *Runner {
	r.embedder = e
	return r
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
		next, result, err := r.runFor(ctx, src)
		r.recordStatus(ctx, src.Name(), result, err)
		if err != nil {
			r.log.Error("foodimport: run source", "source", src.Name(), "result", "failed", "err", err)
			continue
		}
		r.sources[i] = next
	}
	r.backfillEmbeddings(ctx)
}

// recordStatus persists runFor's outcome. Best-effort: a status-write
// failure is logged but never blocks or fails the import itself, and is a
// silent no-op if the store doesn't support it.
func (r *Runner) recordStatus(ctx context.Context, source, result string, runErr error) {
	fs, ok := r.store.(fingerprintStore)
	if !ok {
		return
	}
	errMsg := ""
	if runErr != nil {
		errMsg = runErr.Error()
		result = "failed"
	}
	if err := fs.SetFoodImportStatus(ctx, source, result, errMsg); err != nil {
		r.log.Warn("foodimport: record status", "source", source, "err", err)
	}
}

// backfillEmbeddings runs after every import pass so foods this package just
// wrote (or missed on an earlier run) become matchable without a separate
// manual step. A no-op when no embedder is wired (r.embedder == nil).
// maxLoggedBackfillErrors caps how many per-food embed errors get their own
// log line — a systemic failure (bad OLLAMA_URL, model not pulled) fails
// every item identically, so logging past the first few is just noise.
const maxLoggedBackfillErrors = 3

func (r *Runner) backfillEmbeddings(ctx context.Context) {
	if r.embedder == nil {
		return
	}
	var loggedErrs int
	embedded, failed, err := r.embedder.BackfillEmbeddings(ctx, func(_, _ int, itemErr error) {
		if itemErr == nil || loggedErrs >= maxLoggedBackfillErrors {
			return
		}
		loggedErrs++
		r.log.Warn("foodimport: embedding backfill: food failed", "err", itemErr)
	})
	if err != nil {
		r.log.Error("foodimport: embedding backfill", "result", "failed", "err", err)
		return
	}
	if embedded > 0 || failed > 0 {
		r.log.Info("foodimport: embedding backfill", "result", "done", "embedded", embedded, "failed", failed)
	}
}

// runFor streams src's bulk results into the store in fixed-size batches.
// The returned string is the run's outcome ("imported", "skipped", or
// "changed_during_import") for the caller to persist via recordStatus --
// meaningless when err is non-nil (RunOnce always maps that case to "failed").
func (r *Runner) runFor(ctx context.Context, src ports.BulkSource) (ports.BulkSource, string, error) {
	path := r.localPaths[src.Name()]
	var before string
	if path != "" {
		var err error
		before, err = localFingerprint(path, r.filters[src.Name()])
		if err != nil {
			return src, "", err
		}
		fs, ok := r.store.(fingerprintStore)
		if !ok {
			return src, "", errors.New("foodimport: local file import requires fingerprint store")
		}
		previous, err := fs.GetFoodImportFingerprint(ctx, src.Name())
		if err != nil && !errors.Is(err, types.ErrNotFound) {
			return src, "", fmt.Errorf("get fingerprint: %w", err)
		}
		if err == nil && previous == before {
			r.log.Info("foodimport: skipped", "source", src.Name(), "result", "skipped")
			return src, "skipped", nil
		}
		if makeSource := r.refresh[src.Name()]; makeSource != nil {
			src, err = makeSource()
			if err != nil {
				return src, "", fmt.Errorf("refresh source: %w", err)
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
		return src, "", err
	}
	if err := flush(); err != nil {
		return src, "", err
	}
	if path == "" {
		r.log.Info("foodimport: imported", "source", src.Name(), "result", "imported")
		return src, "imported", nil
	}

	after, err := localFingerprint(path, r.filters[src.Name()])
	if err != nil || after != before {
		r.log.Warn("foodimport: changed during import", "source", src.Name(), "result", "changed_during_import")
		return src, "changed_during_import", nil
	}
	if err := r.store.(fingerprintStore).SetFoodImportFingerprint(ctx, src.Name(), before); err != nil {
		return src, "", fmt.Errorf("set fingerprint: %w", err)
	}
	r.log.Info("foodimport: imported", "source", src.Name(), "result", "imported")
	return src, "imported", nil
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
