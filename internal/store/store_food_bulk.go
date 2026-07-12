package store

import (
	"context"
	"fmt"

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

	for _, food := range foods {
		query, args, err := sqlx.Named(foodUpsertQuery, foodNamedArgs(food))
		if err != nil {
			return fmt.Errorf("bind upsert food %q: %w", food.FoodID, err)
		}
		if _, err := tx.ExecContext(ctx, s.rewrite(query), args...); err != nil {
			return fmt.Errorf("upsert food %q: %w", food.FoodID, err)
		}
	}

	return tx.Commit()
}
