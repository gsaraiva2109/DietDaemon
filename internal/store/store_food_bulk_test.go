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
			Per100g:     types.Macros{Calories: float64(i % 800), Protein: 1, Carbs: 2, Fat: 3, Fiber: 0.5},
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
		if got.Name != fmt.Sprintf("Food %d", i) || got.Per100g.Calories != float64(i%800) {
			t.Fatalf("GetFood bulk-%d: unexpected row %+v", i, got)
		}
	}

	// Re-run with overlapping IDs but changed data — must update, not duplicate.
	updated := []types.FoodMatch{
		{FoodID: "bulk-0", Name: "Updated Food 0", Source: "usda", Per100g: types.Macros{Calories: 899}},
		{FoodID: "bulk-500", Name: "Updated Food 500", Source: "usda", Per100g: types.Macros{Calories: 899}},
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
	if got.Name != "Updated Food 0" || got.Per100g.Calories != 899 {
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

func TestBulkUpsertFoodsSkipsImplausibleMacros(t *testing.T) {
	s, cleanup := tempDB(t)
	defer cleanup()

	foods := []types.FoodMatch{
		{FoodID: "good-1", Name: "Good Food", Source: "taco", Per100g: types.Macros{Calories: 100, Protein: 5, Carbs: 10, Fat: 2, Fiber: 1}},
		{FoodID: "bad-1", Name: "Corrupted Food", Source: "taco", Per100g: types.Macros{Calories: 2, Protein: 606, Carbs: 2535, Fat: 23, Fiber: 54}},
	}
	if err := s.BulkUpsertFoods(ctx(), foods); err != nil {
		t.Fatalf("BulkUpsertFoods: %v", err)
	}

	var count int
	if err := s.db.Get(&count, "SELECT COUNT(*) FROM foods"); err != nil {
		t.Fatalf("count foods: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected only the plausible row to be written, got %d rows", count)
	}
	if _, err := s.GetFood(ctx(), "good-1"); err != nil {
		t.Fatalf("GetFood good-1: %v", err)
	}
	if _, err := s.GetFood(ctx(), "bad-1"); err == nil {
		t.Fatal("expected bad-1 to be skipped, not written")
	}
}

func TestUpsertFoodRejectsImplausibleMacros(t *testing.T) {
	s, cleanup := tempDB(t)
	defer cleanup()

	match := types.FoodMatch{
		FoodID: "bad-2", Name: "Corrupted Food", Source: "taco",
		Per100g: types.Macros{Calories: 2, Protein: 606, Carbs: 2535, Fat: 23, Fiber: 54},
	}
	if err := s.UpsertFood(ctx(), "user-1", match, nil); err == nil {
		t.Fatal("expected UpsertFood to reject implausible macros")
	}
	if _, err := s.GetFood(ctx(), "bad-2"); err == nil {
		t.Fatal("expected bad-2 to not be written")
	}
}

func TestRepairFoodMacros(t *testing.T) {
	s, cleanup := tempDB(t)
	defer cleanup()

	// Seed a row exactly as issue #111's stale importer left it: a legacy
	// numeric food_id with shuffled, implausible macros. Written directly via
	// SQL since BulkUpsertFoods now rightly refuses to write such a row.
	if _, err := s.db.Exec(
		`INSERT INTO foods (food_id, name, source, kcal_100g, protein_100g, carbs_100g, fat_100g, fiber_100g, created_at, updated_at)
		 VALUES ('558', 'Amendoim', 'taco', 2, 606, 2535, 23, 54, '2026-01-01T00:00:00Z', '2026-01-01T00:00:00Z')`,
	); err != nil {
		t.Fatalf("seed legacy row: %v", err)
	}

	fresh := []types.FoodMatch{
		{FoodID: "TACO558", Name: "Amendoim", Source: "taco", Per100g: types.Macros{Calories: 606, Protein: 22.5, Carbs: 18.7, Fat: 54, Fiber: 7.8}},
		{FoodID: "no-match", Name: "Nothing Stored", Source: "taco", Per100g: types.Macros{Calories: 1}},
	}
	fixed, err := s.RepairFoodMacros(ctx(), fresh)
	if err != nil {
		t.Fatalf("RepairFoodMacros: %v", err)
	}
	if fixed != 1 {
		t.Fatalf("expected 1 row fixed, got %d", fixed)
	}

	got, err := s.GetFood(ctx(), "558")
	if err != nil {
		t.Fatalf("GetFood 558: %v", err)
	}
	if got.Per100g.Calories != 606 || got.Per100g.Protein != 22.5 || got.Per100g.Carbs != 18.7 || got.Per100g.Fat != 54 || got.Per100g.Fiber != 7.8 {
		t.Fatalf("unexpected repaired macros: %+v", got.Per100g)
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
