package store

import (
	"context"
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

	s, err := New("sqlite", path, SQLiteDialect())
	if err != nil {
		t.Fatalf("New(%q): %v", path, err)
	}
	return s, func() {
		_ = s.Close()
		_ = os.Remove(path)
	}
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
// Targets set / get
// ---------------------------------------------------------------------------

func TestTargetsSetGet(t *testing.T) {
	s, cleanup := tempDB(t)
	defer cleanup()

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

	// Verify upsert semantics (replace, not duplicate).
	targets.Targets.Calories = 3200
	if err := s.SetTargets(ctx(), targets); err != nil {
		t.Fatalf("SetTargets (update): %v", err)
	}
	got, _ = s.GetTargets(ctx(), "u1")
	if got.Targets.Calories != 3200 {
		t.Fatalf("expected updated calories 3200, got %f", got.Targets.Calories)
	}
}

// ---------------------------------------------------------------------------
// Rollup upsert / get
// ---------------------------------------------------------------------------

func TestRollupUpsertGet(t *testing.T) {
	s, cleanup := tempDB(t)
	defer cleanup()

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
