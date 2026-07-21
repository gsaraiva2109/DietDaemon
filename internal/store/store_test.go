package store

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/gsaraiva2109/dietdaemon/core/types"
)

func tempDB(t *testing.T) (*Store, func()) {
	t.Helper()
	f, err := os.CreateTemp("", "dietdaemon-test-*.db")
	if err != nil {
		t.Fatalf("create temp file: %v", err)
	}
	path := f.Name()
	_ = f.Close()
	_ = os.Remove(path) // New will create it

	s, err := New("sqlite", path, SQLiteDialect(), nil)
	if err != nil {
		t.Fatalf("New(%q): %v", path, err)
	}
	return s, func() {
		_ = s.Close()
		_ = os.Remove(path)
	}
}

func TestCustomFoodLifecycle(t *testing.T) {
	s, cleanup := tempDB(t)
	defer cleanup()

	mustUser(t, s, types.User{ID: "custom-owner", CreatedAt: time.Now().UTC()})
	mustUser(t, s, types.User{ID: "custom-other", CreatedAt: time.Now().UTC()})

	input := types.CustomFoodInput{
		Name:       "  Homemade Oat Bar  ",
		BasisGrams: 200,
		Macros:     types.Macros{Calories: 400, Protein: 20, Carbs: 60, Fat: 10, Fiber: 8},
	}
	food, err := s.CreateCustomFood(ctx(), "custom-owner", input)
	if err != nil {
		t.Fatalf("CreateCustomFood: %v", err)
	}
	if food.Source != "custom" || food.Name != "Homemade Oat Bar" || food.ServingSize != 200 || food.ServingUnit != "g" {
		t.Fatalf("created food = %+v", food)
	}
	if food.Per100g != (types.Macros{Calories: 200, Protein: 10, Carbs: 30, Fat: 5, Fiber: 4}) {
		t.Fatalf("per 100g = %+v", food.Per100g)
	}
	if !food.InLibrary {
		t.Fatal("custom food is not in its owner's library")
	}

	if _, err := s.LookupFood(ctx(), "custom-owner", "homemade oat bar"); err != nil {
		t.Fatalf("canonical alias lookup: %v", err)
	}
	if _, err := s.GetFoodForUser(ctx(), "custom-other", food.FoodID); err != types.ErrNoMatch {
		t.Fatalf("other user GetFoodForUser error = %v, want ErrNoMatch", err)
	}
	if _, err := s.GetFoodDetail(ctx(), "custom-other", food.FoodID); err != types.ErrNotFound {
		t.Fatalf("other user GetFoodDetail error = %v, want ErrNotFound", err)
	}
	otherCatalog, err := s.SearchCatalog(ctx(), "custom-other", "homemade oat", "", 20, 0)
	if err != nil {
		t.Fatalf("other user catalog search: %v", err)
	}
	if len(otherCatalog) != 0 {
		t.Fatalf("private food leaked into other catalog: %+v", otherCatalog)
	}
	for _, candidate := range mustListFoodsWithoutVectors(t, s) {
		if candidate.FoodID == food.FoodID {
			t.Fatal("custom food was returned for vector indexing")
		}
	}

	if _, err := s.CreateCustomFood(ctx(), "custom-owner", input); err != types.ErrConflict {
		t.Fatalf("duplicate canonical alias error = %v, want ErrConflict", err)
	}
	if _, err := s.CreateCustomFood(ctx(), "custom-owner", types.CustomFoodInput{Name: "bad", BasisGrams: 100, Macros: types.Macros{Calories: -1}}); err == nil {
		t.Fatal("negative custom nutrition unexpectedly succeeded")
	}

	updated, err := s.UpdateCustomFood(ctx(), "custom-owner", food.FoodID, types.CustomFoodInput{
		Name:       "Updated Oat Bar",
		BasisGrams: 50,
		Macros:     types.Macros{Calories: 50, Protein: 5, Carbs: 10, Fat: 2, Fiber: 1},
	})
	if err != nil {
		t.Fatalf("UpdateCustomFood: %v", err)
	}
	if updated.ServingSize != 50 || updated.Per100g != (types.Macros{Calories: 100, Protein: 10, Carbs: 20, Fat: 4, Fiber: 2}) {
		t.Fatalf("updated food = %+v", updated)
	}
	if _, err := s.UpdateCustomFood(ctx(), "custom-other", food.FoodID, input); err != types.ErrNotFound {
		t.Fatalf("other user UpdateCustomFood error = %v, want ErrNotFound", err)
	}
	if err := s.DeleteCustomFood(ctx(), "custom-other", food.FoodID); err != types.ErrNotFound {
		t.Fatalf("other user DeleteCustomFood error = %v, want ErrNotFound", err)
	}
	if err := s.DeleteCustomFood(ctx(), "custom-owner", food.FoodID); err != nil {
		t.Fatalf("DeleteCustomFood: %v", err)
	}
	if _, err := s.GetFoodDetail(ctx(), "custom-owner", food.FoodID); err != types.ErrNotFound {
		t.Fatalf("deleted food detail error = %v, want ErrNotFound", err)
	}
	var remaining int
	if err := s.db.Get(&remaining, `SELECT COUNT(*) FROM food_search WHERE food_id = ?`, food.FoodID); err != nil {
		t.Fatalf("count deleted search rows: %v", err)
	}
	if remaining != 0 {
		t.Fatalf("deleted custom food has %d search rows", remaining)
	}
}

func mustListFoodsWithoutVectors(t *testing.T, s *Store) []types.FoodMatch {
	t.Helper()
	foods, err := s.ListFoodsWithoutVectors(ctx())
	if err != nil {
		t.Fatalf("ListFoodsWithoutVectors: %v", err)
	}
	return foods
}

func ctx() context.Context { return context.Background() }

func mustUser(t *testing.T, s *Store, u types.User) {
	t.Helper()
	if err := s.UpsertUser(ctx(), u); err != nil {
		t.Fatalf("UpsertUser: %v", err)
	}
}

// ---------------------------------------------------------------------------
// User round-trip
// ---------------------------------------------------------------------------

func TestUserUpsertGet(t *testing.T) {
	s, cleanup := tempDB(t)
	defer cleanup()

	u := types.User{
		ID:        "user-1",
		Timezone:  "America/Sao_Paulo",
		CreatedAt: time.Date(2026, 6, 17, 12, 0, 0, 0, time.UTC),
	}

	// Get before insert → ErrNotFound.
	_, err := s.GetUser(ctx(), "user-1")
	if err != types.ErrNotFound {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}

	mustUser(t, s, u)

	got, err := s.GetUser(ctx(), "user-1")
	if err != nil {
		t.Fatalf("GetUser: %v", err)
	}
	if got.ID != u.ID || got.Timezone != u.Timezone {
		t.Fatalf("mismatch: got %+v, want %+v", got, u)
	}

	// Upsert (replace).
	u.Timezone = "Europe/Lisbon"
	mustUser(t, s, u)
	got, _ = s.GetUser(ctx(), "user-1")
	if got.Timezone != "Europe/Lisbon" {
		t.Fatalf("expected updated timezone, got %s", got.Timezone)
	}
}

// ---------------------------------------------------------------------------
// Meal save + read (round-trip)
// ---------------------------------------------------------------------------

func TestSaveAndRecentMeals(t *testing.T) {
	s, cleanup := tempDB(t)
	defer cleanup()

	mustUser(t, s, types.User{ID: "u1", CreatedAt: time.Now().UTC()})

	now := time.Date(2026, 6, 17, 18, 30, 0, 0, time.UTC)
	meal := types.Meal{
		ID:         "meal-1",
		UserID:     "u1",
		At:         now,
		RawText:    "200g frango, 2 ovos",
		Confidence: 0.95,
		ParserTier: types.TierDeterministic,
		CreatedAt:  now,
		Items: []types.ResolvedItem{
			{
				Parsed: types.ParsedItem{
					RawPhrase:       "frango",
					Quantity:        200,
					Unit:            "g",
					NormalizedGrams: 200,
				},
				Match: types.FoodMatch{
					FoodID:     "frango-grelhado",
					Name:       "Frango Grelhado",
					Source:     "taco",
					MatchScore: 1.0,
					Per100g: types.Macros{
						Calories: 165, Protein: 31, Carbs: 0, Fat: 3.6, Fiber: 0,
					},
				},
				Macros: types.Macros{
					Calories: 330, Protein: 62, Carbs: 0, Fat: 7.2, Fiber: 0,
				},
			},
			{
				Parsed: types.ParsedItem{
					RawPhrase:       "ovos",
					Quantity:        2,
					Unit:            "un",
					NormalizedGrams: 100,
				},
				Match: types.FoodMatch{
					FoodID:     "ovo-cozido",
					Name:       "Ovo Cozido",
					Source:     "taco",
					MatchScore: 1.0,
					Per100g: types.Macros{
						Calories: 155, Protein: 13, Carbs: 1.1, Fat: 10.6, Fiber: 0,
					},
				},
				Macros: types.Macros{
					Calories: 155, Protein: 13, Carbs: 1.1, Fat: 10.6, Fiber: 0,
				},
			},
		},
	}

	if err := s.SaveMeal(ctx(), meal); err != nil {
		t.Fatalf("SaveMeal: %v", err)
	}

	meals, err := s.RecentMeals(ctx(), "u1", 10)
	if err != nil {
		t.Fatalf("RecentMeals: %v", err)
	}
	if len(meals) != 1 {
		t.Fatalf("expected 1 meal, got %d", len(meals))
	}

	got := meals[0]
	if got.ID != meal.ID || got.RawText != meal.RawText || got.Confidence != meal.Confidence {
		t.Fatalf("meal fields mismatch:\n got  %+v\n want %+v", got, meal)
	}
	if got.ParserTier != meal.ParserTier {
		t.Fatalf("parser tier: got %d, want %d", got.ParserTier, meal.ParserTier)
	}
	if len(got.Items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(got.Items))
	}

	// Verify item 0: macros should round-trip exactly.
	for i, want := range meal.Items {
		gi := got.Items[i]
		if gi.Parsed.RawPhrase != want.Parsed.RawPhrase {
			t.Errorf("item %d raw_phrase: got %q, want %q", i, gi.Parsed.RawPhrase, want.Parsed.RawPhrase)
		}
		if gi.Parsed.Quantity != want.Parsed.Quantity {
			t.Errorf("item %d quantity: got %f, want %f", i, gi.Parsed.Quantity, want.Parsed.Quantity)
		}
		if gi.Parsed.Unit != want.Parsed.Unit {
			t.Errorf("item %d unit: got %q, want %q", i, gi.Parsed.Unit, want.Parsed.Unit)
		}
		if gi.Match.FoodID != want.Match.FoodID {
			t.Errorf("item %d food_id: got %q, want %q", i, gi.Match.FoodID, want.Match.FoodID)
		}
		if gi.Macros != want.Macros {
			t.Errorf("item %d macros: got %+v, want %+v", i, gi.Macros, want.Macros)
		}
		// Per100g should be reconstructed approximately.
		if gi.Match.Per100g.Calories < want.Match.Per100g.Calories-0.01 ||
			gi.Match.Per100g.Calories > want.Match.Per100g.Calories+0.01 {
			t.Errorf("item %d per100g kcal: got %f, want %f", i, gi.Match.Per100g.Calories, want.Match.Per100g.Calories)
		}
	}
}

// ---------------------------------------------------------------------------
// CorrectMealItem ownership check
// ---------------------------------------------------------------------------

// TestCorrectMealItemOwnership verifies that CorrectMealItem refuses to touch
// a meal belonging to a different user, mirroring AddMealItem/DeleteMealItem's
// ownership check. Regression test for a bug where any authenticated user
// could correct another user's meal by guessing/observing its mealID.
func TestCorrectMealItemOwnership(t *testing.T) {
	s, cleanup := tempDB(t)
	defer cleanup()

	mustUser(t, s, types.User{ID: "userA", CreatedAt: time.Now().UTC()})
	mustUser(t, s, types.User{ID: "userB", CreatedAt: time.Now().UTC()})

	now := time.Date(2026, 6, 17, 12, 0, 0, 0, time.UTC)
	meal := types.Meal{
		ID:         "meal-a1",
		UserID:     "userA",
		At:         now,
		RawText:    "200g frango",
		Confidence: 0.95,
		ParserTier: types.TierDeterministic,
		CreatedAt:  now,
		Items: []types.ResolvedItem{
			{
				Parsed: types.ParsedItem{RawPhrase: "frango", Quantity: 200, Unit: "g", NormalizedGrams: 200},
				Match: types.FoodMatch{
					FoodID: "frango-grelhado", Name: "Frango Grelhado", Source: "taco", MatchScore: 1.0,
					Per100g: types.Macros{Calories: 165, Protein: 31, Carbs: 0, Fat: 3.6, Fiber: 0},
				},
				Macros: types.Macros{Calories: 330, Protein: 62, Carbs: 0, Fat: 7.2, Fiber: 0},
			},
		},
	}
	if err := s.SaveMeal(ctx(), meal); err != nil {
		t.Fatalf("SaveMeal: %v", err)
	}

	corrected := types.ResolvedItem{
		Parsed: types.ParsedItem{RawPhrase: "frango grelhado extra", Quantity: 150, Unit: "g", NormalizedGrams: 150},
		Match: types.FoodMatch{
			FoodID: "frango-grelhado", Name: "Frango Grelhado", Source: "taco", MatchScore: 1.0,
			Per100g: types.Macros{Calories: 165, Protein: 31, Carbs: 0, Fat: 3.6, Fiber: 0},
		},
		Macros: types.Macros{Calories: 247.5, Protein: 46.5, Carbs: 0, Fat: 5.4, Fiber: 0},
	}

	// userB attempts to correct userA's meal.
	err := s.CorrectMealItem(ctx(), "userB", meal.ID, 0, corrected)
	if err != types.ErrNotFound {
		t.Fatalf("expected ErrNotFound for cross-user correction, got %v", err)
	}

	// Meal must be unchanged.
	got, err := s.GetMeal(ctx(), meal.ID)
	if err != nil {
		t.Fatalf("GetMeal: %v", err)
	}
	if len(got.Items) != 1 || got.Items[0].Macros != meal.Items[0].Macros {
		t.Fatalf("meal item was modified by cross-user correction: got %+v", got.Items)
	}

	// Rollup must be unchanged (no row should exist, since SaveMeal doesn't
	// write rollups directly and CorrectMealItem must not have run).
	localDate := now.Format("2006-01-02")
	if _, err := s.GetRollup(ctx(), "userA", localDate); err != types.ErrNotFound {
		t.Fatalf("expected no rollup row for userA (CorrectMealItem must not have touched it), got err=%v", err)
	}
}

// ---------------------------------------------------------------------------
// Food library: upsert → lookup → record-query (frequency ordering)
// ---------------------------------------------------------------------------

func TestFoodLibraryRoundTrip(t *testing.T) {
	s, cleanup := tempDB(t)
	defer cleanup()

	mustUser(t, s, types.User{ID: "u1", CreatedAt: time.Now().UTC()})

	// Insert two foods.
	frango := types.FoodMatch{
		FoodID: "frango", Name: "Frango Grelhado", Source: "taco",
		Per100g: types.Macros{Calories: 165, Protein: 31, Carbs: 0, Fat: 3.6, Fiber: 0},
	}
	arroz := types.FoodMatch{
		FoodID: "arroz", Name: "Arroz Branco", Source: "taco",
		Per100g: types.Macros{Calories: 130, Protein: 2.7, Carbs: 28, Fat: 0.3, Fiber: 0.4},
	}

	if err := s.UpsertFood(ctx(), "u1", frango, []string{"frango", "Frango Grelhado", "chicken"}); err != nil {
		t.Fatalf("UpsertFood frango: %v", err)
	}
	if err := s.UpsertFood(ctx(), "u1", arroz, []string{"arroz", "Arroz Branco", "rice"}); err != nil {
		t.Fatalf("UpsertFood arroz: %v", err)
	}

	// Lookup via exact alias (normalized).
	match, err := s.LookupFood(ctx(), "u1", "Frângó") // accented — must normalize
	if err != nil {
		t.Fatalf("LookupFood frango: %v", err)
	}
	if match.FoodID != "frango" {
		t.Fatalf("expected frango, got %s", match.FoodID)
	}

	// Lookup via English alias.
	match, err = s.LookupFood(ctx(), "u1", "rice")
	if err != nil {
		t.Fatalf("LookupFood rice: %v", err)
	}
	if match.FoodID != "arroz" {
		t.Fatalf("expected arroz, got %s", match.FoodID)
	}

	// Lookup non-existent → ErrNoMatch.
	_, err = s.LookupFood(ctx(), "u1", "pizza")
	if err != types.ErrNoMatch {
		t.Fatalf("expected ErrNoMatch, got %v", err)
	}

	// Record queries on frango to boost frequency.
	for i := 0; i < 5; i++ {
		if err := s.RecordFoodQuery(ctx(), "u1", "frango"); err != nil {
			t.Fatalf("RecordFoodQuery: %v", err)
		}
	}
	// frango should still be findable after frequency bumps.
	match, err = s.LookupFood(ctx(), "u1", "chicken")
	if err != nil {
		t.Fatalf("LookupFood chicken after record: %v", err)
	}
	if match.FoodID != "frango" {
		t.Fatalf("expected frango, got %s", match.FoodID)
	}

	// arroz with zero queries should also work.
	match, err = s.LookupFood(ctx(), "u1", "arroz")
	if err != nil {
		t.Fatalf("LookupFood arroz: %v", err)
	}
	if match.FoodID != "arroz" {
		t.Fatalf("expected arroz, got %s", match.FoodID)
	}

	// Verify frequency in the DB directly (both foods match the alias "comida",
	// but PK constraint means only one can own it — insert arroz first, then
	// verify frango cannot steal the alias because INSERT OR IGNORE does not
	// replace).
	if err := s.UpsertFood(ctx(), "u1", arroz, []string{"comida"}); err != nil {
		t.Fatalf("add comida alias to arroz: %v", err)
	}
	if err := s.UpsertFood(ctx(), "u1", frango, []string{"comida"}); err != nil {
		t.Fatalf("add comida alias to frango: %v", err)
	}
	// "comida" still maps to arroz (first writer wins with INSERT OR IGNORE).
	match, err = s.LookupFood(ctx(), "u1", "comida")
	if err != nil {
		t.Fatalf("LookupFood comida: %v", err)
	}
	if match.FoodID != "arroz" {
		t.Fatalf("alias should stick to first writer (arroz), got %s", match.FoodID)
	}
}

// ---------------------------------------------------------------------------
// Catalog search + library removal
// ---------------------------------------------------------------------------

func TestSearchCatalogUnscopedToLibrary(t *testing.T) {
	s, cleanup := tempDB(t)
	defer cleanup()
	mustUser(t, s, types.User{ID: "u1", CreatedAt: time.Now().UTC()})

	// Bulk-imported, never logged by anyone — catalog-only.
	if err := s.BulkUpsertFoods(ctx(), []types.FoodMatch{
		{FoodID: "frango-cat", Name: "Frango Grelhado", Source: "taco", Per100g: types.Macros{Calories: 165}},
		{FoodID: "arroz-cat", Name: "Arroz Branco", Source: "usda", Per100g: types.Macros{Calories: 130}},
	}); err != nil {
		t.Fatalf("BulkUpsertFoods: %v", err)
	}

	// Logged by u1 — should show up as in_library with usage stats.
	feijao := types.FoodMatch{FoodID: "feijao-lib", Name: "Feijao Preto", Source: "taco", Per100g: types.Macros{Calories: 90}}
	if err := s.UpsertFood(ctx(), "u1", feijao, nil); err != nil {
		t.Fatalf("UpsertFood feijao: %v", err)
	}
	if err := s.RecordFoodQuery(ctx(), "u1", "feijao-lib"); err != nil {
		t.Fatalf("RecordFoodQuery: %v", err)
	}

	// Browsing with no query returns every catalog food, ordered by name.
	all, err := s.SearchCatalog(ctx(), "u1", "", "", 20, 0)
	if err != nil {
		t.Fatalf("SearchCatalog: %v", err)
	}
	if len(all) != 3 {
		t.Fatalf("expected 3 catalog foods, got %d: %+v", len(all), all)
	}
	byID := map[string]types.FoodDetail{}
	for _, fd := range all {
		byID[fd.FoodID] = fd
	}
	if fd := byID["frango-cat"]; fd.InLibrary || fd.QueryCount != 0 || fd.LastUsed != "" {
		t.Errorf("catalog-only food should not be in library: %+v", fd)
	}
	if fd := byID["feijao-lib"]; !fd.InLibrary || fd.QueryCount != 1 {
		t.Errorf("logged food should be in library with query_count 1: %+v", fd)
	}

	// Source filter.
	tacoOnly, err := s.SearchCatalog(ctx(), "u1", "", "taco", 20, 0)
	if err != nil {
		t.Fatalf("SearchCatalog source=taco: %v", err)
	}
	if len(tacoOnly) != 2 {
		t.Fatalf("expected 2 taco foods, got %d: %+v", len(tacoOnly), tacoOnly)
	}

	// Full-text query against the global catalog, including catalog-only foods.
	matches, err := s.SearchCatalog(ctx(), "u1", "Frango", "", 20, 0)
	if err != nil {
		t.Fatalf("SearchCatalog q=Frango: %v", err)
	}
	if len(matches) != 1 || matches[0].FoodID != "frango-cat" {
		t.Fatalf("expected only frango-cat to match, got %+v", matches)
	}

	// Limit/offset paginate consistently.
	page, err := s.SearchCatalog(ctx(), "u1", "", "", 1, 1)
	if err != nil {
		t.Fatalf("SearchCatalog paged: %v", err)
	}
	if len(page) != 1 {
		t.Fatalf("expected 1 result for limit=1 offset=1, got %d", len(page))
	}
}

func TestRemoveFromLibrary(t *testing.T) {
	s, cleanup := tempDB(t)
	defer cleanup()
	mustUser(t, s, types.User{ID: "u1", CreatedAt: time.Now().UTC()})

	frango := types.FoodMatch{FoodID: "frango", Name: "Frango Grelhado", Source: "taco", Per100g: types.Macros{Calories: 165}}
	if err := s.UpsertFood(ctx(), "u1", frango, nil); err != nil {
		t.Fatalf("UpsertFood: %v", err)
	}
	if _, err := s.GetFoodDetail(ctx(), "u1", "frango"); err != nil {
		t.Fatalf("GetFoodDetail before removal: %v", err)
	}

	if err := s.RemoveFromLibrary(ctx(), "u1", "frango"); err != nil {
		t.Fatalf("RemoveFromLibrary: %v", err)
	}
	if _, err := s.GetFoodDetail(ctx(), "u1", "frango"); !errors.Is(err, types.ErrNotFound) {
		t.Fatalf("GetFoodDetail after removal: expected ErrNotFound, got %v", err)
	}

	// Global catalog row is untouched.
	if _, err := s.GetFood(ctx(), "frango"); err != nil {
		t.Fatalf("GetFood after removal from library: %v", err)
	}

	// Removing again (already gone) is ErrNotFound.
	if err := s.RemoveFromLibrary(ctx(), "u1", "frango"); !errors.Is(err, types.ErrNotFound) {
		t.Fatalf("RemoveFromLibrary again: expected ErrNotFound, got %v", err)
	}

	// Removing a food that was never in the library is ErrNotFound too.
	if err := s.RemoveFromLibrary(ctx(), "u1", "never-logged"); !errors.Is(err, types.ErrNotFound) {
		t.Fatalf("RemoveFromLibrary never-logged: expected ErrNotFound, got %v", err)
	}
}

// ---------------------------------------------------------------------------
// Targets set / get
// ---------------------------------------------------------------------------

func TestTargetsSetGet(t *testing.T) {
	s, cleanup := tempDB(t)
	defer cleanup()

	mustUser(t, s, types.User{ID: "u1"})

	// Get on missing → ErrNotFound.
	_, err := s.GetTargets(ctx(), "u1")
	if err != types.ErrNotFound {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}

	targets := types.DailyTargets{
		UserID: "u1",
		Targets: types.Macros{
			Calories: 3000, Protein: 180, Carbs: 350, Fat: 80, Fiber: 30,
		},
		WaterGoalMl: 2000,
	}
	if err := s.SetTargets(ctx(), targets); err != nil {
		t.Fatalf("SetTargets: %v", err)
	}

	got, err := s.GetTargets(ctx(), "u1")
	if err != nil {
		t.Fatalf("GetTargets: %v", err)
	}
	if got.Targets != targets.Targets {
		t.Fatalf("targets mismatch: got %+v, want %+v", got.Targets, targets.Targets)
	}
	if got.WaterGoalMl != 2000 {
		t.Fatalf("expected water goal 2000, got %d", got.WaterGoalMl)
	}

	// Verify upsert semantics (replace, not duplicate).
	targets.Targets.Calories = 3200
	targets.WaterGoalMl = 2500
	if err := s.SetTargets(ctx(), targets); err != nil {
		t.Fatalf("SetTargets (update): %v", err)
	}
	got, _ = s.GetTargets(ctx(), "u1")
	if got.Targets.Calories != 3200 {
		t.Fatalf("expected updated calories 3200, got %f", got.Targets.Calories)
	}
	if got.WaterGoalMl != 2500 {
		t.Fatalf("expected updated water goal 2500, got %d", got.WaterGoalMl)
	}
}

// TestTargetsWaterGoalDefaultOnLegacyRow simulates a row written before the
// water_goal_ml column existed conceptually: an INSERT that omits the column
// relies on the migration's NOT NULL DEFAULT 2000 to backfill it.
func TestTargetsWaterGoalDefaultOnLegacyRow(t *testing.T) {
	s, cleanup := tempDB(t)
	defer cleanup()

	mustUser(t, s, types.User{ID: "legacy-user"})

	_, err := s.db.ExecContext(ctx(), s.rewrite(
		`INSERT INTO daily_targets (user_id, kcal, protein, carbs, fat, fiber) VALUES (?, ?, ?, ?, ?, ?)`),
		"legacy-user", 2000.0, 150.0, 250.0, 70.0, 25.0)
	if err != nil {
		t.Fatalf("insert legacy row: %v", err)
	}

	got, err := s.GetTargets(ctx(), "legacy-user")
	if err != nil {
		t.Fatalf("GetTargets: %v", err)
	}
	if got.WaterGoalMl != 2000 {
		t.Fatalf("expected column default 2000 for legacy row, got %d", got.WaterGoalMl)
	}
}

// ---------------------------------------------------------------------------
// Rollup upsert / get
// ---------------------------------------------------------------------------

func TestRollupUpsertGet(t *testing.T) {
	s, cleanup := tempDB(t)
	defer cleanup()

	mustUser(t, s, types.User{ID: "u1"})

	// Get on missing → ErrNotFound.
	_, err := s.GetRollup(ctx(), "u1", "2026-06-17")
	if err != types.ErrNotFound {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}

	r := types.DailyRollup{
		UserID: "u1",
		Date:   "2026-06-17",
		Consumed: types.Macros{
			Calories: 2100, Protein: 140, Carbs: 260, Fat: 55, Fiber: 20,
		},
		Targets: types.Macros{
			Calories: 3000, Protein: 180, Carbs: 350, Fat: 80, Fiber: 30,
		},
	}
	if err := s.UpsertRollup(ctx(), r); err != nil {
		t.Fatalf("UpsertRollup: %v", err)
	}

	got, err := s.GetRollup(ctx(), "u1", "2026-06-17")
	if err != nil {
		t.Fatalf("GetRollup: %v", err)
	}
	if got.Consumed != r.Consumed {
		t.Fatalf("consumed mismatch: got %+v, want %+v", got.Consumed, r.Consumed)
	}
	if got.Targets != r.Targets {
		t.Fatalf("targets mismatch: got %+v, want %+v", got.Targets, r.Targets)
	}

	// Upsert (replace) same day.
	r.Consumed.Calories = 2500
	if err := s.UpsertRollup(ctx(), r); err != nil {
		t.Fatalf("UpsertRollup (update): %v", err)
	}
	got, _ = s.GetRollup(ctx(), "u1", "2026-06-17")
	if got.Consumed.Calories != 2500 {
		t.Fatalf("expected updated consumed 2500, got %f", got.Consumed.Calories)
	}
}

// ---------------------------------------------------------------------------
// ErrNotFound paths (user, targets, rollup)
// ---------------------------------------------------------------------------

func TestErrNotFoundPaths(t *testing.T) {
	s, cleanup := tempDB(t)
	defer cleanup()

	if _, err := s.GetUser(ctx(), "no-one"); err != types.ErrNotFound {
		t.Errorf("GetUser: expected ErrNotFound, got %v", err)
	}
	if _, err := s.GetTargets(ctx(), "no-one"); err != types.ErrNotFound {
		t.Errorf("GetTargets: expected ErrNotFound, got %v", err)
	}
	if _, err := s.GetRollup(ctx(), "no-one", "2020-01-01"); err != types.ErrNotFound {
		t.Errorf("GetRollup: expected ErrNotFound, got %v", err)
	}
}

// ---------------------------------------------------------------------------
// ErrNoMatch path
// ---------------------------------------------------------------------------

func TestErrNoMatchPath(t *testing.T) {
	s, cleanup := tempDB(t)
	defer cleanup()

	mustUser(t, s, types.User{ID: "u1", CreatedAt: time.Now().UTC()})

	_, err := s.LookupFood(ctx(), "u1", "nonexistent")
	if err != types.ErrNoMatch {
		t.Errorf("LookupFood: expected ErrNoMatch, got %v", err)
	}
}

// ---------------------------------------------------------------------------
// RecentMeals ordering (newest first)
// ---------------------------------------------------------------------------

func TestRecentMealsOrdering(t *testing.T) {
	s, cleanup := tempDB(t)
	defer cleanup()

	mustUser(t, s, types.User{ID: "u1", CreatedAt: time.Now().UTC()})

	base := time.Date(2026, 6, 17, 18, 0, 0, 0, time.UTC)
	for i := 0; i < 3; i++ {
		m := types.Meal{
			ID:        "meal-" + string(rune('a'+i)),
			UserID:    "u1",
			At:        base.Add(time.Duration(i) * time.Hour),
			RawText:   "meal",
			CreatedAt: base.Add(time.Duration(i) * time.Hour),
		}
		if err := s.SaveMeal(ctx(), m); err != nil {
			t.Fatalf("SaveMeal %d: %v", i, err)
		}
	}

	meals, err := s.RecentMeals(ctx(), "u1", 10)
	if err != nil {
		t.Fatalf("RecentMeals: %v", err)
	}
	if len(meals) != 3 {
		t.Fatalf("expected 3 meals, got %d", len(meals))
	}
	// Newest first: c, b, a.
	if meals[0].ID != "meal-c" || meals[1].ID != "meal-b" || meals[2].ID != "meal-a" {
		t.Fatalf("wrong order: %v", []string{meals[0].ID, meals[1].ID, meals[2].ID})
	}

	// Limit respected.
	meals, _ = s.RecentMeals(ctx(), "u1", 1)
	if len(meals) != 1 {
		t.Fatalf("expected 1 meal with limit=1, got %d", len(meals))
	}
}

// ---------------------------------------------------------------------------
// RecentMeals empty result
// ---------------------------------------------------------------------------

func TestRecentMealsEmpty(t *testing.T) {
	s, cleanup := tempDB(t)
	defer cleanup()

	meals, err := s.RecentMeals(ctx(), "u1", 10)
	if err != nil {
		t.Fatalf("RecentMeals: %v", err)
	}
	if len(meals) != 0 {
		t.Fatalf("expected 0 meals, got %d", len(meals))
	}
}

// ---------------------------------------------------------------------------
// Context cancellation
// ---------------------------------------------------------------------------

func TestContextCancellation(t *testing.T) {
	s, cleanup := tempDB(t)
	defer cleanup()

	cctx, cancel := context.WithCancel(context.Background())
	cancel()

	if err := s.UpsertUser(cctx, types.User{ID: "x", CreatedAt: time.Now()}); err == nil {
		t.Error("expected error on cancelled context, got nil")
	}
}

// ---------------------------------------------------------------------------
// Phrase normalization edge cases
// ---------------------------------------------------------------------------

// ---------------------------------------------------------------------------
// ListUsers
// ---------------------------------------------------------------------------

func TestListUsers(t *testing.T) {
	s, cleanup := tempDB(t)
	defer cleanup()

	// Empty store → empty list.
	users, err := s.ListUsers(ctx())
	if err != nil {
		t.Fatalf("ListUsers empty: %v", err)
	}
	if len(users) != 0 {
		t.Fatalf("expected 0 users, got %d", len(users))
	}

	// Upsert two users.
	u1 := types.User{ID: "u1", Timezone: "America/Sao_Paulo", CreatedAt: time.Now().UTC()}
	u2 := types.User{ID: "u2", Timezone: "Europe/Lisbon", CreatedAt: time.Now().UTC()}
	mustUser(t, s, u1)
	mustUser(t, s, u2)

	users, err = s.ListUsers(ctx())
	if err != nil {
		t.Fatalf("ListUsers: %v", err)
	}
	if len(users) != 2 {
		t.Fatalf("expected 2 users, got %d", len(users))
	}
	if users[0].ID != "u1" || users[1].ID != "u2" {
		t.Errorf("order wrong: %v", []string{users[0].ID, users[1].ID})
	}
	if users[1].Timezone != "Europe/Lisbon" {
		t.Errorf("u2 timezone = %q", users[1].Timezone)
	}
}

// ---------------------------------------------------------------------------
// Nudge dedupe
// ---------------------------------------------------------------------------

func TestNudgeDedupe(t *testing.T) {
	s, cleanup := tempDB(t)
	defer cleanup()

	mustUser(t, s, types.User{ID: "u1"})
	mustUser(t, s, types.User{ID: "u2"})

	// Nothing nudged yet.
	done, err := s.WasNudged(ctx(), "u1", "2026-06-17", "rule-1")
	if err != nil {
		t.Fatalf("WasNudged (empty): %v", err)
	}
	if done {
		t.Error("expected false before MarkNudged")
	}

	// Mark it.
	if err := s.MarkNudged(ctx(), "u1", "2026-06-17", "rule-1"); err != nil {
		t.Fatalf("MarkNudged: %v", err)
	}

	// Now it's done.
	done, err = s.WasNudged(ctx(), "u1", "2026-06-17", "rule-1")
	if err != nil {
		t.Fatalf("WasNudged (after): %v", err)
	}
	if !done {
		t.Error("expected true after MarkNudged")
	}

	// MarkNudged twice is idempotent — no error.
	if err := s.MarkNudged(ctx(), "u1", "2026-06-17", "rule-1"); err != nil {
		t.Fatalf("MarkNudged idempotent: %v", err)
	}

	// Different user / date / rule still false.
	done, _ = s.WasNudged(ctx(), "u2", "2026-06-17", "rule-1")
	if done {
		t.Error("different user should not be nudged")
	}
	done, _ = s.WasNudged(ctx(), "u1", "2026-06-18", "rule-1")
	if done {
		t.Error("different date should not be nudged")
	}
	done, _ = s.WasNudged(ctx(), "u1", "2026-06-17", "rule-2")
	if done {
		t.Error("different rule should not be nudged")
	}
}

// ---------------------------------------------------------------------------
// Chat routing (reverse user_id -> channel + delivery metadata)
// ---------------------------------------------------------------------------

func TestChatRouteUpsertThenGet(t *testing.T) {
	s, cleanup := tempDB(t)
	defer cleanup()
	mustUser(t, s, types.User{ID: "u1"})

	if err := s.UpsertChatRoute(ctx(), "u1", "telegram", map[string]string{"chat_id": "123"}); err != nil {
		t.Fatalf("UpsertChatRoute: %v", err)
	}
	channel, meta, err := s.GetChatRoute(ctx(), "u1")
	if err != nil {
		t.Fatalf("GetChatRoute: %v", err)
	}
	if channel != "telegram" || meta["chat_id"] != "123" {
		t.Errorf("got channel=%q meta=%v, want telegram/{chat_id:123}", channel, meta)
	}

	// Upsert again with different metadata: should overwrite, not duplicate.
	if err := s.UpsertChatRoute(ctx(), "u1", "telegram", map[string]string{"chat_id": "456"}); err != nil {
		t.Fatalf("UpsertChatRoute (update): %v", err)
	}
	_, meta, err = s.GetChatRoute(ctx(), "u1")
	if err != nil {
		t.Fatalf("GetChatRoute (after update): %v", err)
	}
	if meta["chat_id"] != "456" {
		t.Errorf("chat_id = %q, want 456 after overwrite", meta["chat_id"])
	}
}

func TestChatRouteNotFound(t *testing.T) {
	s, cleanup := tempDB(t)
	defer cleanup()

	_, _, err := s.GetChatRoute(ctx(), "nobody")
	if !errors.Is(err, types.ErrNotFound) {
		t.Errorf("GetChatRoute for unknown user: err = %v, want ErrNotFound", err)
	}
}

// ---------------------------------------------------------------------------
// Nudge rule config (per-user overrides)
// ---------------------------------------------------------------------------

func TestNudgeRuleConfigUpsertAndDelete(t *testing.T) {
	s, cleanup := tempDB(t)
	defer cleanup()

	mustUser(t, s, types.User{ID: "u1", CreatedAt: time.Now().UTC()})

	// No overrides yet.
	cfgs, err := s.GetNudgeRuleConfig(ctx(), "u1")
	if err != nil {
		t.Fatalf("GetNudgeRuleConfig (empty): %v", err)
	}
	if len(cfgs) != 0 {
		t.Errorf("expected no overrides, got %d", len(cfgs))
	}

	// Set one.
	params := json.RawMessage(`{"MinFraction":0.5}`)
	if err := s.SetNudgeRuleConfig(ctx(), "u1", "protein-evening", false, params); err != nil {
		t.Fatalf("SetNudgeRuleConfig: %v", err)
	}
	cfgs, err = s.GetNudgeRuleConfig(ctx(), "u1")
	if err != nil {
		t.Fatalf("GetNudgeRuleConfig: %v", err)
	}
	if len(cfgs) != 1 || cfgs[0].RuleID != "protein-evening" || cfgs[0].Enabled {
		t.Fatalf("unexpected config after set: %+v", cfgs)
	}
	if string(cfgs[0].Params) != string(params) {
		t.Errorf("params = %s, want %s", cfgs[0].Params, params)
	}

	// Upsert: same rule, flip enabled, change params.
	if err := s.SetNudgeRuleConfig(ctx(), "u1", "protein-evening", true, json.RawMessage(`{"MinFraction":0.9}`)); err != nil {
		t.Fatalf("SetNudgeRuleConfig (update): %v", err)
	}
	cfgs, _ = s.GetNudgeRuleConfig(ctx(), "u1")
	if len(cfgs) != 1 || !cfgs[0].Enabled {
		t.Fatalf("expected upsert to update in place, got %+v", cfgs)
	}

	// Reset to default: delete the override row.
	if err := s.DeleteNudgeRuleConfig(ctx(), "u1", "protein-evening"); err != nil {
		t.Fatalf("DeleteNudgeRuleConfig: %v", err)
	}
	cfgs, _ = s.GetNudgeRuleConfig(ctx(), "u1")
	if len(cfgs) != 0 {
		t.Errorf("expected no overrides after delete, got %d", len(cfgs))
	}

	// Deleting again (nothing to delete) must not error.
	if err := s.DeleteNudgeRuleConfig(ctx(), "u1", "protein-evening"); err != nil {
		t.Errorf("DeleteNudgeRuleConfig (no-op): %v", err)
	}
}

// ---------------------------------------------------------------------------
// Fasting
// ---------------------------------------------------------------------------

func TestFastingLifecycle(t *testing.T) {
	s, cleanup := tempDB(t)
	defer cleanup()

	mustUser(t, s, types.User{ID: "user-1", CreatedAt: time.Now().UTC()})

	// No active fast yet.
	if _, err := s.GetActiveFast(ctx(), "user-1"); !errors.Is(err, types.ErrNotFound) {
		t.Fatalf("GetActiveFast: expected ErrNotFound, got %v", err)
	}

	start := time.Now().UTC().Add(-17 * time.Hour)
	f := types.Fast{
		ID:          "fast-1",
		UserID:      "user-1",
		StartAt:     start,
		TargetHours: 16,
		CreatedAt:   start,
	}
	if err := s.StartFast(ctx(), f); err != nil {
		t.Fatalf("StartFast: %v", err)
	}

	active, err := s.GetActiveFast(ctx(), "user-1")
	if err != nil {
		t.Fatalf("GetActiveFast: %v", err)
	}
	if active.ID != "fast-1" || active.EndAt != nil {
		t.Errorf("unexpected active fast: %+v", active)
	}

	end := time.Now().UTC()
	ended, err := s.EndFast(ctx(), "user-1", "fast-1", end, true)
	if err != nil {
		t.Fatalf("EndFast: %v", err)
	}
	if ended.EndAt == nil || !ended.Completed {
		t.Errorf("ended fast not closed correctly: %+v", ended)
	}

	// Active is gone.
	if _, err := s.GetActiveFast(ctx(), "user-1"); !errors.Is(err, types.ErrNotFound) {
		t.Errorf("GetActiveFast after end: expected ErrNotFound, got %v", err)
	}

	// Ending again → ErrNotFound.
	if _, err := s.EndFast(ctx(), "user-1", "fast-1", end, false); !errors.Is(err, types.ErrNotFound) {
		t.Errorf("EndFast twice: expected ErrNotFound, got %v", err)
	}

	// History has the one fast.
	hist, err := s.ListFasts(ctx(), "user-1", 10)
	if err != nil {
		t.Fatalf("ListFasts: %v", err)
	}
	if len(hist) != 1 || hist[0].ID != "fast-1" {
		t.Errorf("unexpected history: %+v", hist)
	}
}

// ---------------------------------------------------------------------------
// Pending aliases
// ---------------------------------------------------------------------------

func TestPendingAliasRoundTrip(t *testing.T) {
	s, cleanup := tempDB(t)
	defer cleanup()

	mustUser(t, s, types.User{ID: "u1", CreatedAt: time.Now().UTC()})

	frango := types.FoodMatch{
		FoodID: "frango", Name: "Frango Grelhado", Source: "taco",
		Per100g: types.Macros{Calories: 165, Protein: 31},
	}
	arroz := types.FoodMatch{
		FoodID: "arroz", Name: "Arroz Branco", Source: "taco",
		Per100g: types.Macros{Calories: 130, Protein: 2.7},
	}
	if err := s.UpsertFood(ctx(), "u1", frango, nil); err != nil {
		t.Fatalf("UpsertFood frango: %v", err)
	}
	if err := s.UpsertFood(ctx(), "u1", arroz, nil); err != nil {
		t.Fatalf("UpsertFood arroz: %v", err)
	}

	if err := s.AddPendingAlias(ctx(), "u1", "frango grelhado", "frango", 0.95); err != nil {
		t.Fatalf("AddPendingAlias: %v", err)
	}

	list, err := s.ListPendingAliases(ctx(), "u1")
	if err != nil {
		t.Fatalf("ListPendingAliases: %v", err)
	}
	if len(list) != 1 || list[0].Phrase != "frango grelhado" || list[0].FoodID != "frango" || list[0].MatchScore != 0.95 {
		t.Fatalf("unexpected pending list: %+v", list)
	}

	// A different user must not see it.
	mustUser(t, s, types.User{ID: "u2", CreatedAt: time.Now().UTC()})
	otherList, err := s.ListPendingAliases(ctx(), "u2")
	if err != nil {
		t.Fatalf("ListPendingAliases u2: %v", err)
	}
	if len(otherList) != 0 {
		t.Errorf("u2 should have no pending aliases, got %+v", otherList)
	}

	// Confirming as the wrong user fails.
	if err := s.ConfirmPendingAlias(ctx(), "u2", list[0].ID); !errors.Is(err, types.ErrNotFound) {
		t.Errorf("ConfirmPendingAlias wrong user: expected ErrNotFound, got %v", err)
	}

	// Confirm promotes it into food_aliases and removes the pending row.
	if err := s.ConfirmPendingAlias(ctx(), "u1", list[0].ID); err != nil {
		t.Fatalf("ConfirmPendingAlias: %v", err)
	}
	match, err := s.LookupFood(ctx(), "u1", "frango grelhado")
	if err != nil {
		t.Fatalf("LookupFood after confirm: %v", err)
	}
	if match.FoodID != "frango" {
		t.Errorf("expected frango, got %s", match.FoodID)
	}
	list, err = s.ListPendingAliases(ctx(), "u1")
	if err != nil {
		t.Fatalf("ListPendingAliases after confirm: %v", err)
	}
	if len(list) != 0 {
		t.Errorf("pending row should be gone after confirm, got %+v", list)
	}

	// Confirming a gone/unknown ID → ErrNotFound.
	if err := s.ConfirmPendingAlias(ctx(), "u1", "does-not-exist"); !errors.Is(err, types.ErrNotFound) {
		t.Errorf("ConfirmPendingAlias unknown id: expected ErrNotFound, got %v", err)
	}

	// Reject removes the row without promoting it.
	if err := s.AddPendingAlias(ctx(), "u1", "arroz cozido", "arroz", 0.93); err != nil {
		t.Fatalf("AddPendingAlias 2: %v", err)
	}
	list, err = s.ListPendingAliases(ctx(), "u1")
	if err != nil || len(list) != 1 {
		t.Fatalf("ListPendingAliases before reject: %v %+v", err, list)
	}
	if err := s.RejectPendingAlias(ctx(), "u1", list[0].ID); err != nil {
		t.Fatalf("RejectPendingAlias: %v", err)
	}
	if _, err := s.LookupFood(ctx(), "u1", "arroz cozido"); !errors.Is(err, types.ErrNoMatch) {
		t.Errorf("rejected alias should not resolve, got err=%v", err)
	}
	if err := s.RejectPendingAlias(ctx(), "u1", "does-not-exist"); !errors.Is(err, types.ErrNotFound) {
		t.Errorf("RejectPendingAlias unknown id: expected ErrNotFound, got %v", err)
	}
}

func TestCorrectionFeedbackAliases(t *testing.T) {
	s, cleanup := tempDB(t)
	defer cleanup()
	mustUser(t, s, types.User{ID: "u1", CreatedAt: time.Now().UTC()})
	old := types.FoodMatch{FoodID: "old", Name: "Old", Source: "test", Per100g: types.Macros{Calories: 100}}
	newFood := types.FoodMatch{FoodID: "new", Name: "New", Source: "test", Per100g: types.Macros{Calories: 200}}
	if err := s.UpsertFood(ctx(), "u1", old, nil); err != nil {
		t.Fatal(err)
	}
	if err := s.UpsertFood(ctx(), "u1", newFood, nil); err != nil {
		t.Fatal(err)
	}
	meal := types.Meal{ID: "m1", UserID: "u1", At: time.Now().UTC(), CreatedAt: time.Now().UTC(), Items: []types.ResolvedItem{{Parsed: types.ParsedItem{RawPhrase: "usual", NormalizedGrams: 100}, Match: old, Macros: old.Per100g}}}
	if err := s.SaveMeal(ctx(), meal); err != nil {
		t.Fatal(err)
	}
	corrected := types.ResolvedItem{Parsed: types.ParsedItem{RawPhrase: "new food", NormalizedGrams: 100}, Match: newFood, Macros: newFood.Per100g}
	feedback, err := s.CorrectMealItemWithFeedback(ctx(), "u1", "m1", 0, corrected)
	if err != nil || feedback.PendingAliasID != "" {
		t.Fatalf("direct alias feedback=%+v err=%v", feedback, err)
	}
	if got, err := s.LookupFood(ctx(), "u1", "usual"); err != nil || got.FoodID != "new" {
		t.Fatalf("learned alias = %+v, %v", got, err)
	}

	meal.ID = "m2"
	meal.Items[0].Parsed.RawPhrase = "usual"
	meal.Items[0].Match = old
	meal.Items[0].Macros = old.Per100g
	if err := s.SaveMeal(ctx(), meal); err != nil {
		t.Fatal(err)
	}
	if err := s.DeleteFoodAlias(ctx(), "u1", "new", "usual"); err != nil {
		t.Fatal(err)
	}
	if err := s.AddFoodAlias(ctx(), "u1", "old", "usual"); err != nil {
		t.Fatal(err)
	}
	feedback, err = s.CorrectMealItemWithFeedback(ctx(), "u1", "m2", 0, corrected)
	if err != nil || feedback.PendingAliasID == "" {
		t.Fatalf("conflict feedback=%+v err=%v", feedback, err)
	}
	if err := s.ConfirmPendingAlias(ctx(), "u1", feedback.PendingAliasID); err != nil {
		t.Fatal(err)
	}
	if got, err := s.LookupFood(ctx(), "u1", "usual"); err != nil || got.FoodID != "new" {
		t.Fatalf("replaced alias = %+v, %v", got, err)
	}
}

// ---------------------------------------------------------------------------
// Source precedence
// ---------------------------------------------------------------------------

func TestSourcePrecedenceRoundTrip(t *testing.T) {
	s, cleanup := tempDB(t)
	defer cleanup()

	mustUser(t, s, types.User{ID: "u1", CreatedAt: time.Now().UTC()})

	// No customization yet → empty slice, not an error.
	order, err := s.GetSourcePrecedence(ctx(), "u1")
	if err != nil {
		t.Fatalf("GetSourcePrecedence (empty): %v", err)
	}
	if len(order) != 0 {
		t.Errorf("expected empty precedence, got %v", order)
	}

	if err := s.SetSourcePrecedence(ctx(), "u1", []string{"usda", "off", "taco"}); err != nil {
		t.Fatalf("SetSourcePrecedence: %v", err)
	}
	order, err = s.GetSourcePrecedence(ctx(), "u1")
	if err != nil {
		t.Fatalf("GetSourcePrecedence: %v", err)
	}
	want := []string{"usda", "off", "taco"}
	if len(order) != len(want) {
		t.Fatalf("order = %v, want %v", order, want)
	}
	for i := range want {
		if order[i] != want[i] {
			t.Errorf("order[%d] = %q, want %q", i, order[i], want[i])
		}
	}

	// Setting again replaces the previous order entirely.
	if err := s.SetSourcePrecedence(ctx(), "u1", []string{"off", "usda"}); err != nil {
		t.Fatalf("SetSourcePrecedence replace: %v", err)
	}
	order, err = s.GetSourcePrecedence(ctx(), "u1")
	if err != nil {
		t.Fatalf("GetSourcePrecedence after replace: %v", err)
	}
	want = []string{"off", "usda"}
	if len(order) != len(want) {
		t.Fatalf("order = %v, want %v", order, want)
	}
	for i := range want {
		if order[i] != want[i] {
			t.Errorf("order[%d] = %q, want %q", i, order[i], want[i])
		}
	}
}

// ---------------------------------------------------------------------------
// Restore (backup restore idempotent inserts + range queries)
// ---------------------------------------------------------------------------

func TestRestoreSleep_Idempotent(t *testing.T) {
	s, cleanup := tempDB(t)
	defer cleanup()

	mustUser(t, s, types.User{ID: "user-1", CreatedAt: time.Now().UTC()})

	sl := types.SleepLog{
		ID: "sleep-1", UserID: "user-1",
		SleepAt: "2026-06-17T22:00:00Z", Quality: "good", Note: "slept well",
	}
	if err := s.RestoreSleep(ctx(), sl); err != nil {
		t.Fatalf("RestoreSleep (first): %v", err)
	}
	if err := s.RestoreSleep(ctx(), sl); err != nil {
		t.Fatalf("RestoreSleep (second, idempotent): %v", err)
	}

	got, err := s.ListSleep(ctx(), "user-1", 10)
	if err != nil {
		t.Fatalf("ListSleep: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("len(sleep logs) = %d, want 1", len(got))
	}
}

func TestRestorePhoto_Idempotent(t *testing.T) {
	s, cleanup := tempDB(t)
	defer cleanup()

	mustUser(t, s, types.User{ID: "user-1", CreatedAt: time.Now().UTC()})

	p := types.ProgressPhoto{
		ID: "photo-1", UserID: "user-1", Date: "2026-06-17", View: "front",
		MimeType: "image/jpeg", Data: []byte("fake-jpeg-bytes"), CreatedAt: time.Now().UTC(),
	}
	if err := s.RestorePhoto(ctx(), p); err != nil {
		t.Fatalf("RestorePhoto (first): %v", err)
	}
	if err := s.RestorePhoto(ctx(), p); err != nil {
		t.Fatalf("RestorePhoto (second, idempotent): %v", err)
	}

	got, err := s.ListPhotoMetadata(ctx(), "user-1")
	if err != nil {
		t.Fatalf("ListPhotoMetadata: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("len(photos) = %d, want 1", len(got))
	}
}

func TestRestoreWater_Idempotent(t *testing.T) {
	s, cleanup := tempDB(t)
	defer cleanup()

	mustUser(t, s, types.User{ID: "user-1", CreatedAt: time.Now().UTC()})

	w := types.WaterLog{ID: "water-1", UserID: "user-1", AmountML: 250, LoggedAt: "2026-06-17T08:00:00Z"}
	if err := s.RestoreWater(ctx(), w); err != nil {
		t.Fatalf("RestoreWater (first): %v", err)
	}
	if err := s.RestoreWater(ctx(), w); err != nil {
		t.Fatalf("RestoreWater (second, idempotent): %v", err)
	}

	got, err := s.GetWaterInRange(ctx(), "user-1", "2026-06-17", "2026-06-17")
	if err != nil {
		t.Fatalf("GetWaterInRange: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("len(water logs) = %d, want 1", len(got))
	}
}

func TestGetWaterInRange_ReturnsIndividualRows(t *testing.T) {
	s, cleanup := tempDB(t)
	defer cleanup()

	mustUser(t, s, types.User{ID: "user-1", CreatedAt: time.Now().UTC()})

	entries := []types.WaterLog{
		{ID: "water-1", UserID: "user-1", AmountML: 250, LoggedAt: "2026-06-17T08:00:00Z"},
		{ID: "water-2", UserID: "user-1", AmountML: 300, LoggedAt: "2026-06-17T14:00:00Z"},
		{ID: "water-3", UserID: "user-1", AmountML: 500, LoggedAt: "2026-06-18T09:00:00Z"},
	}
	for _, w := range entries {
		if err := s.RestoreWater(ctx(), w); err != nil {
			t.Fatalf("RestoreWater(%s): %v", w.ID, err)
		}
	}

	got, err := s.GetWaterInRange(ctx(), "user-1", "2026-06-17", "2026-06-18")
	if err != nil {
		t.Fatalf("GetWaterInRange: %v", err)
	}
	if len(got) != 3 {
		t.Fatalf("len(rows) = %d, want 3 (individual, not aggregated)", len(got))
	}
	byID := make(map[string]int)
	for _, w := range got {
		byID[w.ID] = w.AmountML
	}
	for _, want := range entries {
		if byID[want.ID] != want.AmountML {
			t.Errorf("water %s amount = %d, want %d", want.ID, byID[want.ID], want.AmountML)
		}
	}
}

func TestRestoreFast_Idempotent(t *testing.T) {
	s, cleanup := tempDB(t)
	defer cleanup()

	mustUser(t, s, types.User{ID: "user-1", CreatedAt: time.Now().UTC()})

	start := time.Date(2026, 6, 17, 8, 0, 0, 0, time.UTC)
	end := start.Add(16 * time.Hour)
	f := types.Fast{
		ID: "fast-1", UserID: "user-1", StartAt: start, EndAt: &end,
		TargetHours: 16, Completed: true, CreatedAt: start,
	}
	if err := s.RestoreFast(ctx(), f); err != nil {
		t.Fatalf("RestoreFast (first): %v", err)
	}
	if err := s.RestoreFast(ctx(), f); err != nil {
		t.Fatalf("RestoreFast (second, idempotent): %v", err)
	}

	hist, err := s.ListFasts(ctx(), "user-1", 10)
	if err != nil {
		t.Fatalf("ListFasts: %v", err)
	}
	if len(hist) != 1 {
		t.Fatalf("len(fasts) = %d, want 1", len(hist))
	}
	got := hist[0]
	if !got.Completed {
		t.Error("restored fast Completed = false, want true")
	}
	if got.EndAt == nil || !got.EndAt.Equal(end) {
		t.Errorf("restored fast EndAt = %v, want %v", got.EndAt, end)
	}
}

func TestGetWorkoutsInRangeWithExercises_PopulatesExercises(t *testing.T) {
	s, cleanup := tempDB(t)
	defer cleanup()

	mustUser(t, s, types.User{ID: "user-1", CreatedAt: time.Now().UTC()})

	w := types.Workout{
		ID: "workout-1", UserID: "user-1", Name: "Leg day", DurationMin: 60,
		Intensity: "high", LoggedAt: "2026-06-17T18:00:00Z",
		Exercises: []types.WorkoutExercise{
			{Name: "Squat"},
			{Name: "Lunge"},
		},
	}
	if err := s.LogWorkout(ctx(), w); err != nil {
		t.Fatalf("LogWorkout: %v", err)
	}

	got, err := s.GetWorkoutsInRangeWithExercises(ctx(), "user-1", "2026-06-17", "2026-06-17")
	if err != nil {
		t.Fatalf("GetWorkoutsInRangeWithExercises: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("len(workouts) = %d, want 1", len(got))
	}
	if len(got[0].Exercises) != 2 {
		t.Fatalf("len(exercises) = %d, want 2", len(got[0].Exercises))
	}
	names := map[string]bool{}
	for _, e := range got[0].Exercises {
		names[e.Name] = true
	}
	if !names["Squat"] || !names["Lunge"] {
		t.Errorf("exercises = %+v, want Squat and Lunge", got[0].Exercises)
	}
}
