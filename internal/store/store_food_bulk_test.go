package store

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/gsaraiva2109/dietdaemon/core/types"
)

func TestBulkUpsertFoodsRoundTrip(t *testing.T) {
	s, cleanup := tempDB(t)
	defer cleanup()

	const total = 1200
	foods := buildBulkFoods(total)

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

	assertBulkSpotCheck(t, s)
	assertBulkUpsertIsIdempotentUpdate(t, s, total)
	assertBulkImportHasNoUserSideEffects(t, s)
}

// buildBulkFoods returns n synthetic FoodMatch fixtures for bulk-upsert
// round-trip tests.
func buildBulkFoods(n int) []types.FoodMatch {
	foods := make([]types.FoodMatch, n)
	for i := range n {
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
	return foods
}

// assertBulkSpotCheck spot-checks a few of the rows written by
// TestBulkUpsertFoodsRoundTrip.
func assertBulkSpotCheck(t *testing.T, s *Store) {
	t.Helper()
	for _, i := range []int{0, 500, 1199} {
		got, err := s.GetFood(ctx(), fmt.Sprintf("bulk-%d", i))
		if err != nil {
			t.Fatalf("GetFood bulk-%d: %v", i, err)
		}
		if got.Name != fmt.Sprintf("Food %d", i) || got.Per100g.Calories != float64(i%800) {
			t.Fatalf("GetFood bulk-%d: unexpected row %+v", i, got)
		}
	}
}

// assertBulkUpsertIsIdempotentUpdate re-runs BulkUpsertFoods with
// overlapping IDs but changed data — it must update, not duplicate.
func assertBulkUpsertIsIdempotentUpdate(t *testing.T, s *Store, total int) {
	t.Helper()
	updated := []types.FoodMatch{
		{FoodID: "bulk-0", Name: "Updated Food 0", Source: "usda", Per100g: types.Macros{Calories: 899}},
		{FoodID: "bulk-500", Name: "Updated Food 500", Source: "usda", Per100g: types.Macros{Calories: 899}},
	}
	if err := s.BulkUpsertFoods(ctx(), updated); err != nil {
		t.Fatalf("BulkUpsertFoods (update pass): %v", err)
	}

	var count int
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
}

// assertBulkImportHasNoUserSideEffects proves "global-only": no per-user
// side effects from a bulk import.
func assertBulkImportHasNoUserSideEffects(t *testing.T, s *Store) {
	t.Helper()
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

// TestBulkUpsertFoodsWritesServingUnits covers the USDA foodPortions → global
// food_serving_units path (#134/B3): units land with user_id NULL, and a
// re-import replaces the prior set instead of duplicating it (USDA's
// portions for a food can change between runs).
func TestBulkUpsertFoodsWritesServingUnits(t *testing.T) {
	s, cleanup := tempDB(t)
	defer cleanup()

	mustUser(t, s, types.User{ID: "unit-user", CreatedAt: time.Now().UTC()})

	egg := types.FoodMatch{
		FoodID: "usda-egg", Name: "Egg, whole, raw", Source: "usda",
		Per100g:      types.Macros{Calories: 143, Protein: 13, Carbs: 1, Fat: 10},
		ServingUnits: []types.FoodServingUnit{{Label: "1 large", Grams: 50}, {Label: "1 small", Grams: 38}},
	}
	if err := s.BulkUpsertFoods(ctx(), []types.FoodMatch{egg}); err != nil {
		t.Fatalf("BulkUpsertFoods: %v", err)
	}

	if err := s.AddToLibrary(ctx(), "unit-user", egg.FoodID); err != nil {
		t.Fatalf("AddToLibrary: %v", err)
	}
	detail, err := s.GetFoodDetail(ctx(), "unit-user", egg.FoodID)
	if err != nil {
		t.Fatalf("GetFoodDetail: %v", err)
	}
	if len(detail.ServingUnits) != 2 {
		t.Fatalf("ServingUnits = %+v, want 2 system units", detail.ServingUnits)
	}
	for _, u := range detail.ServingUnits {
		if u.Custom {
			t.Fatalf("bulk-imported unit marked Custom: %+v", u)
		}
	}

	// Re-import with a changed portion set — must replace, not accumulate.
	egg.ServingUnits = []types.FoodServingUnit{{Label: "1 jumbo", Grams: 63}}
	if err := s.BulkUpsertFoods(ctx(), []types.FoodMatch{egg}); err != nil {
		t.Fatalf("BulkUpsertFoods (re-import): %v", err)
	}
	detail, err = s.GetFoodDetail(ctx(), "unit-user", egg.FoodID)
	if err != nil {
		t.Fatalf("GetFoodDetail after re-import: %v", err)
	}
	if len(detail.ServingUnits) != 1 || detail.ServingUnits[0].Label != "1 jumbo" {
		t.Fatalf("ServingUnits after re-import = %+v, want only {1 jumbo, 63}", detail.ServingUnits)
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
		 VALUES ('558', 'Amendoim', 'taco', 2, 606, 2535, 23, 54, '2026-01-01T00:00:00Z', '2026-01-01T00:00:00Z'),
		        ('559', 'Feijao', 'taco', 1, 1, 1, 1, 1, '2026-01-01T00:00:00Z', '2026-01-01T00:00:00Z')`,
	); err != nil {
		t.Fatalf("seed legacy rows: %v", err)
	}

	fresh := []types.FoodMatch{
		{
			FoodID: "TACO558", Name: "Amendoim", Source: "taco",
			Per100g:      types.Macros{Calories: 606, Protein: 22.5, Carbs: 18.7, Fat: 54, Fiber: 7.8},
			ServingUnits: []types.FoodServingUnit{{Label: "1 punhado", Grams: 30}},
		},
		{FoodID: "no-match", Name: "Nothing Stored", Source: "taco", Per100g: types.Macros{Calories: 1}},
		// Matches but carries no serving units — covers the "nothing to
		// backfill" path distinct from the "1 punhado" case above.
		{
			FoodID: "TACO559", Name: "Feijao", Source: "taco",
			Per100g: types.Macros{Calories: 333, Protein: 21, Carbs: 60, Fat: 1.2, Fiber: 8.7},
		},
		// Implausible macros: skipped before any DB call, never counted.
		{
			FoodID: "TACOBAD", Name: "Corrupted", Source: "taco",
			Per100g: types.Macros{Calories: 2, Protein: 606, Carbs: 2535, Fat: 23, Fiber: 54},
		},
	}
	fixed, err := s.RepairFoodMacros(ctx(), fresh)
	if err != nil {
		t.Fatalf("RepairFoodMacros: %v", err)
	}
	if fixed != 2 {
		t.Fatalf("expected 2 rows fixed, got %d", fixed)
	}

	got559, err := s.GetFood(ctx(), "559")
	if err != nil {
		t.Fatalf("GetFood 559: %v", err)
	}
	if got559.Per100g.Calories != 333 {
		t.Fatalf("unexpected repaired macros for 559: %+v", got559.Per100g)
	}

	got, err := s.GetFood(ctx(), "558")
	if err != nil {
		t.Fatalf("GetFood 558: %v", err)
	}
	if got.Per100g.Calories != 606 || got.Per100g.Protein != 22.5 || got.Per100g.Carbs != 18.7 || got.Per100g.Fat != 54 || got.Per100g.Fiber != 7.8 {
		t.Fatalf("unexpected repaired macros: %+v", got.Per100g)
	}

	// Serving units land on the stale row's actual food_id ("558"), not the
	// freshly-fetched batch's FoodID ("TACO558") — the whole point of
	// matching by (source, name) instead of food_id.
	mustUser(t, s, types.User{ID: "repair-user", CreatedAt: time.Now().UTC()})
	if err := s.AddToLibrary(ctx(), "repair-user", "558"); err != nil {
		t.Fatalf("AddToLibrary: %v", err)
	}
	detail, err := s.GetFoodDetail(ctx(), "repair-user", "558")
	if err != nil {
		t.Fatalf("GetFoodDetail: %v", err)
	}
	if len(detail.ServingUnits) != 1 || detail.ServingUnits[0].Label != "1 punhado" || detail.ServingUnits[0].Grams != 30 {
		t.Fatalf("ServingUnits = %+v, want [{1 punhado 30}]", detail.ServingUnits)
	}
}

// TestRepairFoodMacrosContinuesPastServingUnitFailure pins down the fix for
// the "partial-commit-then-abandon" bug: a serving-unit-fix failure partway
// through a batch must not silently abandon every food after it. food-a's
// serving units are deliberately invalid (grams <= 0 violates the CHECK
// constraint), which fails only the serving-units transaction for food-a;
// food-a's macro fix (a separate, already-committed statement) must still
// stick, and food-b — later in the same batch — must still be processed in
// full, with an aggregate error surfacing the food-a failure.
func TestRepairFoodMacrosContinuesPastServingUnitFailure(t *testing.T) {
	s, cleanup := tempDB(t)
	defer cleanup()

	seed := func(foodID, name string) {
		t.Helper()
		if _, err := s.db.Exec(
			`INSERT INTO foods (food_id, name, source, kcal_100g, protein_100g, carbs_100g, fat_100g, fiber_100g, created_at, updated_at)
			 VALUES (?, ?, 'taco', 1, 1, 1, 1, 1, '2026-01-01T00:00:00Z', '2026-01-01T00:00:00Z')`,
			foodID, name,
		); err != nil {
			t.Fatalf("seed legacy row %s: %v", foodID, err)
		}
	}
	seed("legacy-a", "FoodA")
	seed("legacy-b", "FoodB")

	fresh := []types.FoodMatch{
		{
			FoodID: "TACOA", Name: "FoodA", Source: "taco",
			Per100g:      types.Macros{Calories: 200, Protein: 10, Carbs: 20, Fat: 5, Fiber: 2},
			ServingUnits: []types.FoodServingUnit{{Label: "invalid", Grams: 0}}, // violates CHECK(grams > 0)
		},
		{
			FoodID: "TACOB", Name: "FoodB", Source: "taco",
			Per100g:      types.Macros{Calories: 300, Protein: 15, Carbs: 30, Fat: 8, Fiber: 3},
			ServingUnits: []types.FoodServingUnit{{Label: "valid", Grams: 50}},
		},
	}

	fixed, err := s.RepairFoodMacros(ctx(), fresh)
	if err == nil {
		t.Fatal("expected an aggregate error from food-a's serving-unit failure")
	}
	if !strings.Contains(err.Error(), "legacy-a") {
		t.Errorf("error %q should identify the food that failed (legacy-a)", err.Error())
	}
	if fixed != 2 {
		t.Fatalf("expected both macro fixes to land (fixed=2), got %d", fixed)
	}

	// food-a's macro fix landed despite its serving-unit failure — the two
	// are independent operations.
	gotA, err := s.GetFood(ctx(), "legacy-a")
	if err != nil {
		t.Fatalf("GetFood legacy-a: %v", err)
	}
	if gotA.Per100g.Calories != 200 {
		t.Fatalf("legacy-a macros not repaired: %+v", gotA.Per100g)
	}

	// food-b, which comes after the failing food-a in the batch, was still
	// fully processed: macros repaired and its serving unit written.
	gotB, err := s.GetFood(ctx(), "legacy-b")
	if err != nil {
		t.Fatalf("GetFood legacy-b: %v", err)
	}
	if gotB.Per100g.Calories != 300 {
		t.Fatalf("legacy-b macros not repaired (batch abandoned after food-a's failure): %+v", gotB.Per100g)
	}
	var unitCount int
	if err := s.db.Get(&unitCount, "SELECT COUNT(*) FROM food_serving_units WHERE food_id = 'legacy-b'"); err != nil {
		t.Fatalf("count legacy-b serving units: %v", err)
	}
	if unitCount != 1 {
		t.Fatalf("legacy-b serving units = %d, want 1 (batch abandoned after food-a's failure)", unitCount)
	}
}

// TestRepairFoodMacrosAggregatesQueryErrors covers the generic (non-
// sql.ErrNoRows) macro-update-query error branch: it must also be collected
// into the aggregate error and skipped over, rather than treated specially.
func TestRepairFoodMacrosAggregatesQueryErrors(t *testing.T) {
	s, cleanup := tempDB(t)
	defer cleanup()

	if _, err := s.db.Exec(
		`INSERT INTO foods (food_id, name, source, kcal_100g, protein_100g, carbs_100g, fat_100g, fiber_100g, created_at, updated_at)
		 VALUES ('legacy-c', 'FoodC', 'taco', 1, 1, 1, 1, 1, '2026-01-01T00:00:00Z', '2026-01-01T00:00:00Z')`,
	); err != nil {
		t.Fatalf("seed legacy row: %v", err)
	}

	cctx, cancel := context.WithCancel(context.Background())
	cancel() // already canceled: the macro-update query fails with a non-ErrNoRows error

	fresh := []types.FoodMatch{
		{FoodID: "TACOC", Name: "FoodC", Source: "taco", Per100g: types.Macros{Calories: 400, Protein: 20, Carbs: 40, Fat: 10, Fiber: 4}},
	}
	fixed, err := s.RepairFoodMacros(cctx, fresh)
	if err == nil {
		t.Fatal("expected an aggregate error from the canceled-context query failure")
	}
	if !strings.Contains(err.Error(), "TACOC") {
		t.Errorf("error %q should identify the food whose query failed (TACOC)", err.Error())
	}
	if fixed != 0 {
		t.Fatalf("fixed = %d, want 0 (the query never completed)", fixed)
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
