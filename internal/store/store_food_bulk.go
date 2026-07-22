package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"

	"github.com/gsaraiva2109/dietdaemon/core/types"
	"github.com/jmoiron/sqlx"
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
	now := utcNow()
	for _, food := range foods {
		if !plausibleMacros(food.Per100g) {
			log.Printf("store: skip bulk upsert of food %q (source=%s): implausible macros %+v", food.FoodID, food.Source, food.Per100g)
			continue
		}
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

	for _, food := range foods {
		if len(food.ServingUnits) == 0 {
			continue
		}
		if err := s.replaceSystemServingUnitsTx(ctx, tx, food.FoodID, food.ServingUnits); err != nil {
			return fmt.Errorf("bulk upsert serving units for %q: %w", food.FoodID, err)
		}
	}

	return tx.Commit()
}

// replaceSystemServingUnitsTx replaces every system-provided (user_id IS
// NULL) serving unit for foodID with units, inside tx. Deleting first keeps
// re-imports idempotent — USDA's foodPortions for a food can change between
// runs, and units have no natural key to upsert against.
func (s *Store) replaceSystemServingUnitsTx(ctx context.Context, tx *sqlx.Tx, foodID string, units []types.FoodServingUnit) error {
	if _, err := tx.ExecContext(ctx, s.rewrite(`DELETE FROM food_serving_units WHERE food_id = ? AND user_id IS NULL`), foodID); err != nil {
		return fmt.Errorf("clear system serving units: %w", err)
	}
	now := utcNow()
	rows := make([][]any, 0, len(units))
	for _, u := range units {
		rows = append(rows, []any{newID(), foodID, u.Label, u.Grams, now})
	}
	const prefix = `INSERT INTO food_serving_units (id, food_id, label, grams, created_at) VALUES `
	if err := s.insertRows(ctx, tx, prefix, "", rows); err != nil {
		return fmt.Errorf("insert system serving units: %w", err)
	}
	return nil
}

// RepairFoodMacros overwrites macros on existing global foods rows that match
// a fresh source row by (source, name), rather than by food_id. It exists
// for one-time repair of catalog rows written by an older/different importer
// under a different food_id scheme, where BulkUpsertFoods' ON CONFLICT(food_id)
// can't reach the stale row at all (see issue #111). Matching by name instead
// of food_id also means the stale row's food_id is never touched, so any
// meal_items/food_aliases/user_food_stats referencing it stay intact — only
// the wrong macro values get corrected in place. Returns the number of source
// rows that matched (and were fixed) an existing catalog row.
// Also backfills serving units (#134/#134-B3) for any matched row that has
// them — the same repair pass USDA foodPortions rides in on to reach catalog
// entries imported before that data was fetched.
func (s *Store) RepairFoodMacros(ctx context.Context, foods []types.FoodMatch) (int, error) {
	const q = `
		UPDATE foods SET kcal_100g = ?, protein_100g = ?, carbs_100g = ?, fat_100g = ?, fiber_100g = ?, updated_at = ?
		WHERE source = ? AND name = ? AND owner_user_id IS NULL
		RETURNING food_id
	`
	now := utcNow()
	fixed := 0
	for _, food := range foods {
		if !plausibleMacros(food.Per100g) {
			log.Printf("store: skip repair of food %q (source=%s): implausible macros %+v", food.FoodID, food.Source, food.Per100g)
			continue
		}
		var matchedID string
		err := s.db.GetContext(ctx, &matchedID, s.rewrite(q),
			food.Per100g.Calories, food.Per100g.Protein, food.Per100g.Carbs, food.Per100g.Fat, food.Per100g.Fiber, now,
			food.Source, food.Name)
		if errors.Is(err, sql.ErrNoRows) {
			continue // no catalog row matched (source, name) — nothing to repair
		}
		if err != nil {
			return fixed, fmt.Errorf("store: repair food macros %q: %w", food.FoodID, err)
		}
		fixed++
		if len(food.ServingUnits) == 0 {
			continue
		}
		tx, err := s.db.BeginTxx(ctx, nil)
		if err != nil {
			return fixed, fmt.Errorf("store: begin repair serving units tx: %w", err)
		}
		if err := s.replaceSystemServingUnitsTx(ctx, tx, matchedID, food.ServingUnits); err != nil {
			_ = tx.Rollback()
			return fixed, fmt.Errorf("store: repair serving units %q: %w", matchedID, err)
		}
		if err := tx.Commit(); err != nil {
			return fixed, fmt.Errorf("store: commit repair serving units %q: %w", matchedID, err)
		}
	}
	return fixed, nil
}
