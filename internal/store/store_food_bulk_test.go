package store

import (
	"fmt"
	"testing"

	"github.com/gsaraiva2109/dietdaemon/core/types"
)

func TestBulkUpsertFoodsRoundTrip(t *testing.T) {
	s, cleanup := tempDB(t)
	defer cleanup()

	const total = 1200
	foods := make([]types.FoodMatch, total)
	for i := range total {
		foods[i] = types.FoodMatch{
			FoodID:      fmt.Sprintf("bulk-%d", i),
			Name:        fmt.Sprintf("Food %d", i),
			Source:      "usda",
			Per100g:     types.Macros{Calories: float64(i), Protein: 1, Carbs: 2, Fat: 3, Fiber: 0.5},
			Category:    "test-category",
			Brand:       "test-brand",
			Barcode:     fmt.Sprintf("barcode-%d", i),
			ImageURL:    "https://example.com/img.png",
			ServingSize: 100,
			ServingUnit: "g",
		}
	}

	if err := s.BulkUpsertFoods(ctx(), foods); err != nil {
		t.Fatalf("BulkUpsertFoods: %v", err)
	}

	var count int
	if err := s.db.Get(&count, "SELECT COUNT(*) FROM foods"); err != nil {
		t.Fatalf("count foods: %v", err)
	}
	if count != total {
		t.Fatalf("expected %d foods, got %d", total, count)
	}

	// Spot-check a few rows.
	for _, i := range []int{0, 500, 1199} {
		got, err := s.GetFood(ctx(), fmt.Sprintf("bulk-%d", i))
		if err != nil {
			t.Fatalf("GetFood bulk-%d: %v", i, err)
		}
		if got.Name != fmt.Sprintf("Food %d", i) || got.Per100g.Calories != float64(i) {
			t.Fatalf("GetFood bulk-%d: unexpected row %+v", i, got)
		}
	}

	// Re-run with overlapping IDs but changed data — must update, not duplicate.
	updated := []types.FoodMatch{
		{FoodID: "bulk-0", Name: "Updated Food 0", Source: "usda", Per100g: types.Macros{Calories: 999}},
		{FoodID: "bulk-500", Name: "Updated Food 500", Source: "usda", Per100g: types.Macros{Calories: 999}},
	}
	if err := s.BulkUpsertFoods(ctx(), updated); err != nil {
		t.Fatalf("BulkUpsertFoods (update pass): %v", err)
	}

	if err := s.db.Get(&count, "SELECT COUNT(*) FROM foods"); err != nil {
		t.Fatalf("count foods after update: %v", err)
	}
	if count != total {
		t.Fatalf("expected count to stay %d after overlapping upsert, got %d", total, count)
	}

	got, err := s.GetFood(ctx(), "bulk-0")
	if err != nil {
		t.Fatalf("GetFood bulk-0 after update: %v", err)
	}
	if got.Name != "Updated Food 0" || got.Per100g.Calories != 999 {
		t.Fatalf("expected updated row, got %+v", got)
	}

	// Proves "global-only": no per-user side effects from a bulk import.
	var statsCount, aliasCount int
	if err := s.db.Get(&statsCount, "SELECT COUNT(*) FROM user_food_stats"); err != nil {
		t.Fatalf("count user_food_stats: %v", err)
	}
	if statsCount != 0 {
		t.Fatalf("expected 0 user_food_stats rows, got %d", statsCount)
	}
	if err := s.db.Get(&aliasCount, "SELECT COUNT(*) FROM food_aliases"); err != nil {
		t.Fatalf("count food_aliases: %v", err)
	}
	if aliasCount != 0 {
		t.Fatalf("expected 0 food_aliases rows, got %d", aliasCount)
	}
}

func TestListFoodsWithoutVectors(t *testing.T) {
	s, cleanup := tempDB(t)
	defer cleanup()

	foods := []types.FoodMatch{
		{FoodID: "no-vec-1", Name: "Arroz", Source: "taco"},
		{FoodID: "no-vec-2", Name: "Feijao", Source: "taco"},
		{FoodID: "has-vec-1", Name: "Chicken Breast", Source: "usda"},
	}
	if err := s.BulkUpsertFoods(ctx(), foods); err != nil {
		t.Fatalf("BulkUpsertFoods: %v", err)
	}

	// Simulate one food already having a vector (e.g. resolved live via an
	// external source, which embeds on write).
	if _, err := s.db.Exec("INSERT INTO food_vectors (food_id, dim, vec) VALUES (?, 1, ?)",
		"has-vec-1", []byte{0, 0, 0, 0}); err != nil {
		t.Fatalf("seed food_vectors: %v", err)
	}

	got, err := s.ListFoodsWithoutVectors(ctx())
	if err != nil {
		t.Fatalf("ListFoodsWithoutVectors: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("expected 2 foods missing vectors, got %d: %+v", len(got), got)
	}
	ids := map[string]bool{}
	for _, fm := range got {
		ids[fm.FoodID] = true
	}
	if !ids["no-vec-1"] || !ids["no-vec-2"] {
		t.Fatalf("expected no-vec-1 and no-vec-2, got %+v", got)
	}
	if ids["has-vec-1"] {
		t.Fatalf("has-vec-1 already has a vector, should be excluded: %+v", got)
	}
}

func TestListFoodsWithoutVectors_EmptyCatalog(t *testing.T) {
	s, cleanup := tempDB(t)
	defer cleanup()

	got, err := s.ListFoodsWithoutVectors(ctx())
	if err != nil {
		t.Fatalf("ListFoodsWithoutVectors: %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("expected 0 foods for empty catalog, got %d", len(got))
	}
}
