package main

import (
	"context"
	"fmt"

	"github.com/gsaraiva2109/dietdaemon/adapters/model/ollama"
	"github.com/gsaraiva2109/dietdaemon/core/types"
	"github.com/gsaraiva2109/dietdaemon/internal/config"
	"github.com/gsaraiva2109/dietdaemon/internal/foodimport"
	"github.com/gsaraiva2109/dietdaemon/internal/index"
	"github.com/gsaraiva2109/dietdaemon/internal/resolver/embedding"
	"github.com/gsaraiva2109/dietdaemon/internal/store"
)

// adminImportBatchSize mirrors cmd/import-foods' own batchSize: the number of
// rows buffered before a store write.
const adminImportBatchSize = 500

// foodImportAdmin implements api.FoodImportRunner against a live daemon's
// store and config, so the admin/food-import/* HTTP endpoints can trigger the
// same bulk import/repair/backfill operations cmd/import-foods runs
// standalone, without requiring shell/volume access to the running
// container's DB (issue #136).
//
// ponytail: duplicates cmd/import-foods' batching loop; dedupe into
// internal/foodimport once a third call site doesn't risk a merge race.
type foodImportAdmin struct {
	store *store.Store
	cfg   *config.Config
}

// ImportSource streams source through the configured filter in fixed-size
// batches, upserting each into the global foods table. maxRows caps the
// number of rows processed; 0 or negative means no cap (use the source's
// configured default), matching cmd/import-foods' -max-rows flag.
func (a *foodImportAdmin) ImportSource(ctx context.Context, source string, maxRows int) (int, error) {
	src, filter, err := foodimport.BuildSource(source, a.cfg)
	if err != nil {
		return 0, err
	}
	if maxRows > 0 {
		filter.MaxRows = maxRows
	}

	batch := make([]types.FoodMatch, 0, adminImportBatchSize)
	total := 0
	flush := func() error {
		if len(batch) == 0 {
			return nil
		}
		if err := a.store.BulkUpsertFoods(ctx, batch); err != nil {
			return err
		}
		total += len(batch)
		batch = batch[:0]
		return nil
	}

	err = src.FetchBulk(ctx, filter, func(fm types.FoodMatch) error {
		batch = append(batch, fm)
		if len(batch) >= adminImportBatchSize {
			return flush()
		}
		return nil
	})
	if err != nil {
		return total, fmt.Errorf("import %s: %w", source, err)
	}
	if err := flush(); err != nil {
		return total, fmt.Errorf("import %s: %w", source, err)
	}
	return total, nil
}

// RepairSource re-fetches source and overwrites macros on existing catalog
// rows matched by (source, name) instead of food_id (see issue #111).
// checked is the number of source rows fetched; fixed is how many matched
// and were corrected.
func (a *foodImportAdmin) RepairSource(ctx context.Context, source string) (checked, fixed int, err error) {
	src, filter, err := foodimport.BuildSource(source, a.cfg)
	if err != nil {
		return 0, 0, err
	}

	var batch []types.FoodMatch
	if err := src.FetchBulk(ctx, filter, func(fm types.FoodMatch) error {
		batch = append(batch, fm)
		return nil
	}); err != nil {
		return 0, 0, fmt.Errorf("fetch %s: %w", source, err)
	}

	fixed, err = a.store.RepairFoodMacros(ctx, batch)
	if err != nil {
		return len(batch), fixed, fmt.Errorf("repair %s: %w", source, err)
	}
	return len(batch), fixed, nil
}

// BackfillEmbeddings embeds every catalog food missing a vector, against a
// live Ollama endpoint. Requires OLLAMA_URL / EMBED_MODEL to be reachable.
func (a *foodImportAdmin) BackfillEmbeddings(ctx context.Context) (embedded, failed int, err error) {
	model := ollama.New(a.cfg.OllamaURL, a.cfg.EmbedModel, "", a.cfg.ModelTimeout)
	idx := index.New(a.store.DB())
	matcher := embedding.New(model, idx, a.store, a.cfg.EmbedMatchThreshold)
	return matcher.BackfillEmbeddings(ctx, nil)
}
