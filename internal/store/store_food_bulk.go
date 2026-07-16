package store

import (
	"context"
	"fmt"

	"github.com/gsaraiva2109/dietdaemon/core/types"
)

// bulkUpsertChunkSize is the number of foods committed per transaction. A
// large import shouldn't hold one giant transaction open.
const bulkUpsertChunkSize = 500

// BulkUpsertFoods writes match rows into the global foods table only — no
// user_food_stats or food_aliases rows are touched, since those are per-user
// and out of scope for a global catalog import (per-user aliasing still
// happens lazily via the normal resolver path the first time a user actually
// logs the food). Rows commit in fixed-size chunks so a large import doesn't
// hold one giant transaction.
func (s *Store) BulkUpsertFoods(ctx context.Context, foods []types.FoodMatch) error {
	for start := 0; start < len(foods); start += bulkUpsertChunkSize {
		end := min(start+bulkUpsertChunkSize, len(foods))
		if err := s.bulkUpsertFoodsChunk(ctx, foods[start:end]); err != nil {
			return fmt.Errorf("store: bulk upsert foods [%d:%d): %w", start, end, err)
		}
	}
	return nil
}

func (s *Store) bulkUpsertFoodsChunk(ctx context.Context, foods []types.FoodMatch) error {
	tx, err := s.db.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	rows := make([][]any, 0, len(foods))
	for _, food := range foods {
		now := utcNow()
		rows = append(rows, []any{
			food.FoodID, food.Name, food.Source,
			food.Per100g.Calories, food.Per100g.Protein, food.Per100g.Carbs, food.Per100g.Fat, food.Per100g.Fiber,
			food.Category, food.Brand, food.Barcode, food.ImageURL, food.ServingSize, food.ServingUnit, now, now,
		})
	}
	const prefix = `INSERT INTO foods
		(food_id, name, source, kcal_100g, protein_100g, carbs_100g, fat_100g, fiber_100g,
		 category, brand, barcode, image_url, serving_size, serving_unit, created_at, updated_at)
		VALUES `
	const suffix = ` ON CONFLICT(food_id) DO UPDATE SET
		name = excluded.name, source = excluded.source, kcal_100g = excluded.kcal_100g,
		protein_100g = excluded.protein_100g, carbs_100g = excluded.carbs_100g,
		fat_100g = excluded.fat_100g, fiber_100g = excluded.fiber_100g,
		category = excluded.category, brand = excluded.brand, barcode = excluded.barcode,
		image_url = excluded.image_url, serving_size = excluded.serving_size,
		serving_unit = excluded.serving_unit, updated_at = excluded.updated_at`
	if err := s.insertRows(ctx, tx, prefix, suffix, rows); err != nil {
		return fmt.Errorf("bulk upsert foods: %w", err)
	}

	return tx.Commit()
}
