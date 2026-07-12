// Command import-foods bulk-loads a nutrition source's food catalog into the
// global foods table so a fresh install has non-empty food search instead of
// relying on lazy per-meal resolution. Runs once and exits — the daemon's own
// internal/foodimport.Runner handles periodic API-mode re-sync separately.
//
// Usage:
//
//	go run ./cmd/import-foods -source taco -db ./data/dietdaemon.db
//	go run ./cmd/import-foods -source usda -db ./data/dietdaemon.db -max-rows 5000
//	go run ./cmd/import-foods -source openfoodfacts -db ./data/dietdaemon.db -dry-run
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/gsaraiva2109/dietdaemon/core/ports"
	"github.com/gsaraiva2109/dietdaemon/core/types"
	"github.com/gsaraiva2109/dietdaemon/internal/config"
	"github.com/gsaraiva2109/dietdaemon/internal/foodimport"
	"github.com/gsaraiva2109/dietdaemon/internal/store"
)

// batchSize is the number of rows buffered before a store write, matching
// BulkUpsertFoods' own internal chunk size.
const batchSize = 500

func main() {
	source := flag.String("source", "", "source to import: usda, openfoodfacts, taco (required)")
	dbPath := flag.String("db", "", "SQLite database path (required)")
	maxRows := flag.Int("max-rows", 0, "cap on rows imported for this run, 0 = use the source's configured default")
	dryRun := flag.Bool("dry-run", false, "fetch and count rows without writing to the store")
	flag.Parse()

	if *source == "" || *dbPath == "" {
		flag.Usage()
		os.Exit(1)
	}

	if err := run(*source, *dbPath, *maxRows, *dryRun); err != nil {
		fmt.Fprintf(os.Stderr, "import-foods: %v\n", err)
		os.Exit(1)
	}
}

func run(source, dbPath string, maxRows int, dryRun bool) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	src, filter, err := foodimport.BuildSource(source, cfg)
	if err != nil {
		return err
	}
	if maxRows > 0 {
		filter.MaxRows = maxRows
	}

	st, err := store.New("sqlite", dbPath, store.SQLiteDialect())
	if err != nil {
		return fmt.Errorf("open store: %w", err)
	}
	defer func() {
		if cerr := st.Close(); cerr != nil {
			fmt.Fprintf(os.Stderr, "import-foods: close store: %v\n", cerr)
		}
	}()

	// A bulk import can page through a live API for minutes; let ctrl-c stop
	// it cleanly (in-flight batch still flushes) rather than killing the
	// process mid-write, matching cmd/dietdaemon's shutdown handling.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	total, err := runImport(ctx, src, filter, st, dryRun)
	if err != nil {
		return fmt.Errorf("import %s: %w", source, err)
	}

	fmt.Printf("import-foods: source=%s dry_run=%v rows=%d\n", source, dryRun, total)
	return nil
}

// bulkUpserter is the subset of *store.Store that runImport needs, so tests
// can swap in a real temp store without depending on the rest of ports.Store.
type bulkUpserter interface {
	BulkUpsertFoods(ctx context.Context, foods []types.FoodMatch) error
}

// runImport streams src through filter in fixed-size batches, writing each
// batch to st (unless dryRun), and returns the total row count seen.
func runImport(ctx context.Context, src ports.BulkSource, filter ports.BulkFilter, st bulkUpserter, dryRun bool) (int, error) {
	batch := make([]types.FoodMatch, 0, batchSize)
	total := 0

	flush := func() error {
		if len(batch) == 0 {
			return nil
		}
		if !dryRun {
			if err := st.BulkUpsertFoods(ctx, batch); err != nil {
				return err
			}
		}
		total += len(batch)
		batch = batch[:0]
		return nil
	}

	err := src.FetchBulk(ctx, filter, func(fm types.FoodMatch) error {
		batch = append(batch, fm)
		if len(batch) >= batchSize {
			return flush()
		}
		return nil
	})
	if err != nil {
		return total, err
	}
	if err := flush(); err != nil {
		return total, err
	}
	return total, nil
}
