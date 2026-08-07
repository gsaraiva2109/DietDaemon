package store

import (
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/gsaraiva2109/dietdaemon/core/types"
)

// ---------------------------------------------------------------------------
// AddMealItem
// ---------------------------------------------------------------------------

func mkItem(phrase string, kcal float64) types.ResolvedItem {
	return types.ResolvedItem{
		Parsed: types.ParsedItem{RawPhrase: phrase, Quantity: 100, Unit: "g", NormalizedGrams: 100},
		Match:  types.FoodMatch{FoodID: "food-" + phrase, Name: phrase, Source: "test", MatchScore: 1.0},
		Macros: types.Macros{Calories: kcal, Protein: 1, Carbs: 1, Fat: 1, Fiber: 1},
	}
}

func mealsByID(meals []types.Meal) map[string]types.Meal {
	byID := make(map[string]types.Meal, len(meals))
	for _, meal := range meals {
		byID[meal.ID] = meal
	}
	return byID
}

func TestAddMealItem(t *testing.T) {
	s, cleanup := tempDB(t)
	defer cleanup()
	mustUser(t, s, types.User{ID: "u1", CreatedAt: time.Now().UTC()})

	now := time.Date(2026, 6, 17, 12, 0, 0, 0, time.UTC)
	meal := types.Meal{
		ID: "meal-1", UserID: "u1", At: now, RawText: "frango", CreatedAt: now,
		Items: []types.ResolvedItem{mkItem("frango", 100)},
	}
	if err := s.SaveMeal(ctx(), meal); err != nil {
		t.Fatalf("SaveMeal: %v", err)
	}

	// First AddMealItem: rollup already exists (created by SaveMeal path via
	// CorrectMealItem-style upsert isn't run here, so this is the insert path).
	if err := s.AddMealItem(ctx(), "u1", meal.ID, mkItem("arroz", 50)); err != nil {
		t.Fatalf("AddMealItem: %v", err)
	}
	got, err := s.GetMeal(ctx(), meal.ID)
	if err != nil {
		t.Fatalf("GetMeal: %v", err)
	}
	if len(got.Items) != 2 {
		t.Fatalf("expected 2 items after AddMealItem, got %d", len(got.Items))
	}
	if got.Items[1].Parsed.RawPhrase != "arroz" {
		t.Fatalf("expected appended item to be arroz, got %q", got.Items[1].Parsed.RawPhrase)
	}

	rollups, err := s.GetRollups(ctx(), "u1", "2026-06-17", "2026-06-17")
	if err != nil {
		t.Fatalf("GetRollups: %v", err)
	}
	if len(rollups) != 1 {
		t.Fatalf("expected 1 rollup, got %d", len(rollups))
	}
	if rollups[0].Consumed.Calories != 50 {
		t.Fatalf("expected consumed kcal 50 after add, got %f", rollups[0].Consumed.Calories)
	}

	// Second AddMealItem on the same day exercises the ON CONFLICT update path.
	if err := s.AddMealItem(ctx(), "u1", meal.ID, mkItem("feijao", 30)); err != nil {
		t.Fatalf("AddMealItem 2: %v", err)
	}
	rollups, err = s.GetRollups(ctx(), "u1", "2026-06-17", "2026-06-17")
	if err != nil {
		t.Fatalf("GetRollups 2: %v", err)
	}
	if rollups[0].Consumed.Calories != 80 {
		t.Fatalf("expected consumed kcal 80 after second add, got %f", rollups[0].Consumed.Calories)
	}

	// Cross-user ownership check.
	mustUser(t, s, types.User{ID: "u2", CreatedAt: time.Now().UTC()})
	if err := s.AddMealItem(ctx(), "u2", meal.ID, mkItem("x", 1)); !errors.Is(err, types.ErrNotFound) {
		t.Fatalf("AddMealItem cross-user: expected ErrNotFound, got %v", err)
	}

	// Unknown meal.
	if err := s.AddMealItem(ctx(), "u1", "no-such-meal", mkItem("x", 1)); !errors.Is(err, types.ErrNotFound) {
		t.Fatalf("AddMealItem unknown meal: expected ErrNotFound, got %v", err)
	}
}

// ---------------------------------------------------------------------------
// DeleteMealItem
// ---------------------------------------------------------------------------

func TestDeleteMealItem(t *testing.T) {
	s, cleanup := tempDB(t)
	defer cleanup()
	mustUser(t, s, types.User{ID: "u1", CreatedAt: time.Now().UTC()})

	now := time.Date(2026, 6, 17, 12, 0, 0, 0, time.UTC)
	meal := types.Meal{
		ID: "meal-1", UserID: "u1", At: now, RawText: "frango e arroz", CreatedAt: now,
		Items: []types.ResolvedItem{mkItem("frango", 100), mkItem("arroz", 50)},
	}
	if err := s.SaveMeal(ctx(), meal); err != nil {
		t.Fatalf("SaveMeal: %v", err)
	}

	// Out of range.
	if err := s.DeleteMealItem(ctx(), "u1", meal.ID, 5); !errors.Is(err, types.ErrNotFound) {
		t.Fatalf("DeleteMealItem out of range: expected ErrNotFound, got %v", err)
	}
	if err := s.DeleteMealItem(ctx(), "u1", meal.ID, -1); !errors.Is(err, types.ErrNotFound) {
		t.Fatalf("DeleteMealItem negative index: expected ErrNotFound, got %v", err)
	}

	// Cross-user ownership.
	mustUser(t, s, types.User{ID: "u2", CreatedAt: time.Now().UTC()})
	if err := s.DeleteMealItem(ctx(), "u2", meal.ID, 0); !errors.Is(err, types.ErrNotFound) {
		t.Fatalf("DeleteMealItem cross-user: expected ErrNotFound, got %v", err)
	}

	// Delete item 0 (frango, 100 kcal): remaining item's rollup should reflect
	// only arroz (50 kcal).
	if err := s.DeleteMealItem(ctx(), "u1", meal.ID, 0); err != nil {
		t.Fatalf("DeleteMealItem: %v", err)
	}
	got, err := s.GetMeal(ctx(), meal.ID)
	if err != nil {
		t.Fatalf("GetMeal: %v", err)
	}
	if len(got.Items) != 1 || got.Items[0].Parsed.RawPhrase != "arroz" {
		t.Fatalf("expected only arroz left, got %+v", got.Items)
	}
}

// ---------------------------------------------------------------------------
// LatestMealTime
// ---------------------------------------------------------------------------

func TestLatestMealTime(t *testing.T) {
	s, cleanup := tempDB(t)
	defer cleanup()
	mustUser(t, s, types.User{ID: "u1", CreatedAt: time.Now().UTC()})

	if _, err := s.LatestMealTime(ctx(), "u1"); !errors.Is(err, types.ErrNotFound) {
		t.Fatalf("LatestMealTime with no meals: expected ErrNotFound, got %v", err)
	}

	now := time.Date(2026, 6, 17, 12, 0, 0, 0, time.UTC)
	meal := types.Meal{ID: "m1", UserID: "u1", At: now, CreatedAt: now, Items: []types.ResolvedItem{mkItem("x", 1)}}
	if err := s.SaveMeal(ctx(), meal); err != nil {
		t.Fatalf("SaveMeal: %v", err)
	}

	at, err := s.LatestMealTime(ctx(), "u1")
	if err != nil {
		t.Fatalf("LatestMealTime: %v", err)
	}
	if at != utcStr(now) {
		t.Fatalf("LatestMealTime = %q, want %q", at, utcStr(now))
	}
}

// ---------------------------------------------------------------------------
// GetMealsInRange
// ---------------------------------------------------------------------------

func TestGetMealsInRange(t *testing.T) {
	s, cleanup := tempDB(t)
	defer cleanup()
	mustUser(t, s, types.User{ID: "u1", CreatedAt: time.Now().UTC()})

	// No meals yet -> empty (non-nil) slice.
	meals, err := s.GetMealsInRange(ctx(), "u1", "2026-06-01", "2026-06-30")
	if err != nil {
		t.Fatalf("GetMealsInRange empty: %v", err)
	}
	if meals == nil || len(meals) != 0 {
		t.Fatalf("expected empty non-nil slice, got %#v", meals)
	}

	in := time.Date(2026, 6, 15, 12, 0, 0, 0, time.UTC)
	out := time.Date(2026, 7, 1, 12, 0, 0, 0, time.UTC)
	mIn := types.Meal{ID: "m-in", UserID: "u1", At: in, CreatedAt: in, Items: []types.ResolvedItem{mkItem("in", 1)}}
	mOut := types.Meal{ID: "m-out", UserID: "u1", At: out, CreatedAt: out, Items: []types.ResolvedItem{mkItem("out", 1)}}
	if err := s.SaveMeal(ctx(), mIn); err != nil {
		t.Fatalf("SaveMeal in-range: %v", err)
	}
	if err := s.SaveMeal(ctx(), mOut); err != nil {
		t.Fatalf("SaveMeal out-of-range: %v", err)
	}

	meals, err = s.GetMealsInRange(ctx(), "u1", "2026-06-01", "2026-06-30")
	if err != nil {
		t.Fatalf("GetMealsInRange: %v", err)
	}
	if len(meals) != 1 || meals[0].ID != "m-in" {
		t.Fatalf("expected only m-in, got %+v", meals)
	}
	if len(meals[0].Items) != 1 || meals[0].Items[0].Parsed.RawPhrase != "in" {
		t.Fatalf("expected items populated for in-range meal, got %+v", meals[0].Items)
	}
}

// ---------------------------------------------------------------------------
// RecentMealTimes
// ---------------------------------------------------------------------------

func TestRecentMealTimes(t *testing.T) {
	s, cleanup := tempDB(t)
	defer cleanup()
	mustUser(t, s, types.User{ID: "u1", CreatedAt: time.Now().UTC()})

	older := time.Date(2026, 6, 1, 8, 0, 0, 0, time.UTC)
	newer := time.Date(2026, 6, 17, 8, 0, 0, 0, time.UTC)
	for i, at := range []time.Time{older, newer} {
		m := types.Meal{ID: "m" + string(rune('0'+i)), UserID: "u1", At: at, CreatedAt: at, Items: []types.ResolvedItem{mkItem("x", 1)}}
		if err := s.SaveMeal(ctx(), m); err != nil {
			t.Fatalf("SaveMeal %d: %v", i, err)
		}
	}

	since := time.Date(2026, 6, 10, 0, 0, 0, 0, time.UTC)
	times, err := s.RecentMealTimes(ctx(), "u1", since)
	if err != nil {
		t.Fatalf("RecentMealTimes: %v", err)
	}
	if len(times) != 1 || !times[0].Equal(newer) {
		t.Fatalf("expected only the newer meal time, got %+v", times)
	}
}

// ---------------------------------------------------------------------------
// GetMeal not found
// ---------------------------------------------------------------------------

func TestGetMealNotFound(t *testing.T) {
	s, cleanup := tempDB(t)
	defer cleanup()

	if _, err := s.GetMeal(ctx(), "does-not-exist"); !errors.Is(err, types.ErrNotFound) {
		t.Fatalf("GetMeal unknown id: expected ErrNotFound, got %v", err)
	}
}

// ---------------------------------------------------------------------------
// PlanSlotID / PlanOptionID persistence
// ---------------------------------------------------------------------------

// TestMealPlanAttributionRoundTrip pins that plan_slot_id/plan_option_id
// survive SaveMeal -> RecentMeals/GetMeal/GetMealsInRange round trips, and
// that a meal logged without plan attribution comes back with empty strings
// rather than erroring (NULL-ish default).
func TestMealPlanAttributionRoundTrip(t *testing.T) {
	s, cleanup := tempDB(t)
	defer cleanup()
	mustUser(t, s, types.User{ID: "u1", CreatedAt: time.Now().UTC()})

	now := time.Date(2026, 6, 17, 12, 0, 0, 0, time.UTC)
	attributed := types.Meal{
		ID: "m-attributed", UserID: "u1", At: now, CreatedAt: now,
		PlanSlotID: "slot-1", PlanOptionID: "option-1",
		Items: []types.ResolvedItem{mkItem("frango", 100)},
	}
	plain := types.Meal{
		ID: "m-plain", UserID: "u1", At: now.Add(time.Hour), CreatedAt: now.Add(time.Hour),
		Items: []types.ResolvedItem{mkItem("arroz", 50)},
	}
	if err := s.SaveMeal(ctx(), attributed); err != nil {
		t.Fatalf("SaveMeal attributed: %v", err)
	}
	if err := s.SaveMeal(ctx(), plain); err != nil {
		t.Fatalf("SaveMeal plain: %v", err)
	}

	got, err := s.GetMeal(ctx(), attributed.ID)
	if err != nil {
		t.Fatalf("GetMeal: %v", err)
	}
	if got.PlanSlotID != "slot-1" || got.PlanOptionID != "option-1" {
		t.Fatalf("GetMeal plan attribution = %+v, want slot-1/option-1", got)
	}

	gotPlain, err := s.GetMeal(ctx(), plain.ID)
	if err != nil {
		t.Fatalf("GetMeal plain: %v", err)
	}
	if gotPlain.PlanSlotID != "" || gotPlain.PlanOptionID != "" {
		t.Fatalf("GetMeal plain plan attribution = %+v, want empty", gotPlain)
	}

	recent, err := s.RecentMeals(ctx(), "u1", 10)
	if err != nil {
		t.Fatalf("RecentMeals: %v", err)
	}
	byID := mealsByID(recent)
	if byID[attributed.ID].PlanSlotID != "slot-1" || byID[attributed.ID].PlanOptionID != "option-1" {
		t.Fatalf("RecentMeals plan attribution = %+v", byID[attributed.ID])
	}
	if byID[plain.ID].PlanSlotID != "" {
		t.Fatalf("RecentMeals plain plan attribution = %+v, want empty", byID[plain.ID])
	}

	ranged, err := s.GetMealsInRange(ctx(), "u1", "2026-06-17", "2026-06-17")
	if err != nil {
		t.Fatalf("GetMealsInRange: %v", err)
	}
	byIDRanged := mealsByID(ranged)
	if byIDRanged[attributed.ID].PlanOptionID != "option-1" {
		t.Fatalf("GetMealsInRange plan attribution = %+v", byIDRanged[attributed.ID])
	}
}

// ---------------------------------------------------------------------------
// SaveMealAndAddToRollup (#272: meal save and rollup update must be atomic
// and additive, or concurrent meal logs for the same user/day lose updates)
// ---------------------------------------------------------------------------

// TestSaveMealAndAddToRollupConcurrent is the -race regression test called
// for in #272's acceptance criteria: N goroutines each log a distinct meal
// for the same user/day concurrently. The old code (separate GetRollup +
// UpsertRollup calls, outside the meal's transaction) could interleave two
// read-modify-write cycles and silently drop one meal's contribution. The
// fix folds both writes into one transaction with an additive SQL upsert
// (consumed_x = consumed_x + delta), so every goroutine's delta lands
// regardless of interleaving. tempDB's sqlite connection pool is capped at
// 1 (see store.go), so this test's goroutines serialize through Go's
// database/sql pool rather than truly overlapping inside SQLite -- it still
// proves the fix, since the bug was the release-then-reacquire gap between
// two separate top-level store calls, which no longer exists now that both
// writes share one transaction. The same additive upsert is what makes this
// safe under genuine concurrent transactions on Postgres, via ordinary
// row-level locking on ON CONFLICT DO UPDATE.
func TestSaveMealAndAddToRollupConcurrent(t *testing.T) {
	s, cleanup := tempDB(t)
	defer cleanup()
	mustUser(t, s, types.User{ID: "u1", CreatedAt: time.Now().UTC()})

	at := time.Date(2026, 6, 17, 12, 0, 0, 0, time.UTC)
	targets := types.Macros{Calories: 2000, Protein: 150, Carbs: 200, Fat: 60, Fiber: 30}

	const n = 12
	var wg sync.WaitGroup
	errs := make(chan error, n)
	for i := range n {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			meal := types.Meal{
				ID:        fmt.Sprintf("meal-%d", i),
				UserID:    "u1",
				At:        at,
				RawText:   "concurrent meal",
				CreatedAt: at,
				Items:     []types.ResolvedItem{mkItem(fmt.Sprintf("item-%d", i), 100)},
			}
			if err := s.SaveMealAndAddToRollup(ctx(), meal, targets); err != nil {
				errs <- err
			}
		}(i)
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		t.Fatalf("SaveMealAndAddToRollup: %v", err)
	}

	rollups, err := s.GetRollups(ctx(), "u1", "2026-06-17", "2026-06-17")
	if err != nil {
		t.Fatalf("GetRollups: %v", err)
	}
	if len(rollups) != 1 {
		t.Fatalf("expected 1 rollup row, got %d", len(rollups))
	}
	if got, want := rollups[0].Consumed.Calories, float64(n*100); got != want {
		t.Fatalf("consumed kcal = %f, want %f (lost update under concurrency)", got, want)
	}
	if rollups[0].Targets.Calories != targets.Calories {
		t.Fatalf("targets.Calories = %f, want %f (seeded once on first insert, must not be clobbered)",
			rollups[0].Targets.Calories, targets.Calories)
	}

	recent, err := s.RecentMeals(ctx(), "u1", n+1)
	if err != nil {
		t.Fatalf("RecentMeals: %v", err)
	}
	if len(recent) != n {
		t.Fatalf("expected %d meals persisted, got %d", n, len(recent))
	}
}

// TestSaveMealAndAddToRollupExistingRow exercises the ON CONFLICT branch
// directly (sequential, not concurrent): a second meal on a day that already
// has a rollup row must add to consumed and leave targets untouched.
func TestSaveMealAndAddToRollupExistingRow(t *testing.T) {
	s, cleanup := tempDB(t)
	defer cleanup()
	mustUser(t, s, types.User{ID: "u1", CreatedAt: time.Now().UTC()})

	at := time.Date(2026, 6, 17, 12, 0, 0, 0, time.UTC)
	first := types.Meal{
		ID: "meal-1", UserID: "u1", At: at, RawText: "frango", CreatedAt: at,
		Items: []types.ResolvedItem{mkItem("frango", 100)},
	}
	if err := s.SaveMealAndAddToRollup(ctx(), first, types.Macros{Calories: 2000}); err != nil {
		t.Fatalf("SaveMealAndAddToRollup 1: %v", err)
	}

	second := types.Meal{
		ID: "meal-2", UserID: "u1", At: at, RawText: "arroz", CreatedAt: at,
		Items: []types.ResolvedItem{mkItem("arroz", 50)},
	}
	// A different targets value must not overwrite the row's existing targets.
	if err := s.SaveMealAndAddToRollup(ctx(), second, types.Macros{Calories: 9999}); err != nil {
		t.Fatalf("SaveMealAndAddToRollup 2: %v", err)
	}

	rollups, err := s.GetRollups(ctx(), "u1", "2026-06-17", "2026-06-17")
	if err != nil {
		t.Fatalf("GetRollups: %v", err)
	}
	if len(rollups) != 1 {
		t.Fatalf("expected 1 rollup row, got %d", len(rollups))
	}
	if rollups[0].Consumed.Calories != 150 {
		t.Fatalf("consumed kcal = %f, want 150", rollups[0].Consumed.Calories)
	}
	if rollups[0].Targets.Calories != 2000 {
		t.Fatalf("targets.Calories = %f, want 2000 (must not be overwritten by second call)", rollups[0].Targets.Calories)
	}
}

// TestSaveMealAndAddToRollupDuplicateSkipsRollup confirms a duplicate
// external_id (safe no-op re-import, see insertMealTx) doesn't double-count
// the rollup, and that the failed insert leaves nothing behind: no partial
// items row and no rollup delta, matching the pre-existing SaveMeal contract.
func TestSaveMealAndAddToRollupDuplicateSkipsRollup(t *testing.T) {
	s, cleanup := tempDB(t)
	defer cleanup()
	mustUser(t, s, types.User{ID: "u1", CreatedAt: time.Now().UTC()})

	at := time.Date(2026, 6, 17, 12, 0, 0, 0, time.UTC)
	ext := "mfp-123"
	meal := types.Meal{
		ID: "meal-1", UserID: "u1", At: at, RawText: "frango", CreatedAt: at,
		ExternalID: &ext,
		Items:      []types.ResolvedItem{mkItem("frango", 100)},
	}
	if err := s.SaveMealAndAddToRollup(ctx(), meal, types.Macros{Calories: 2000}); err != nil {
		t.Fatalf("SaveMealAndAddToRollup 1: %v", err)
	}

	// Re-importing the same external_id under a new meal ID is the dup path.
	dupe := meal
	dupe.ID = "meal-1-retry"
	if err := s.SaveMealAndAddToRollup(ctx(), dupe, types.Macros{Calories: 2000}); err != nil {
		t.Fatalf("SaveMealAndAddToRollup dup: %v", err)
	}

	rollups, err := s.GetRollups(ctx(), "u1", "2026-06-17", "2026-06-17")
	if err != nil {
		t.Fatalf("GetRollups: %v", err)
	}
	if rollups[0].Consumed.Calories != 100 {
		t.Fatalf("consumed kcal = %f, want 100 (dup retry must not double-count)", rollups[0].Consumed.Calories)
	}

	recent, err := s.RecentMeals(ctx(), "u1", 10)
	if err != nil {
		t.Fatalf("RecentMeals: %v", err)
	}
	if len(recent) != 1 {
		t.Fatalf("expected 1 meal persisted (dup skipped), got %d", len(recent))
	}
}

// TestSaveMealAndAddToRollupOnClosedDB exercises the "begin tx" error-wrap
// branch by closing the store's real DB connection first, mirroring
// TestAccountMethodsWrapDBErrorsWhenClosed in store_account_test.go.
func TestSaveMealAndAddToRollupOnClosedDB(t *testing.T) {
	s, cleanup := tempDB(t)
	defer cleanup()
	if err := s.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	meal := types.Meal{ID: "meal-1", UserID: "u1", At: time.Now(), CreatedAt: time.Now()}
	if err := s.SaveMealAndAddToRollup(ctx(), meal, types.Macros{}); err == nil {
		t.Error("SaveMealAndAddToRollup on closed db: want error")
	}
}

// TestSaveMealItemCountLimit confirms a meal over maxMealItems is rejected
// before a transaction is opened, so an oversized payload can't hold the DB
// connection for an unbounded bulk insert (see checkMealItemCount).
func TestSaveMealItemCountLimit(t *testing.T) {
	s, cleanup := tempDB(t)
	defer cleanup()
	mustUser(t, s, types.User{ID: "u1", CreatedAt: time.Now().UTC()})

	items := make([]types.ResolvedItem, maxMealItems+1)
	for i := range items {
		items[i] = mkItem(fmt.Sprintf("item-%d", i), 1)
	}
	meal := types.Meal{ID: "too-big", UserID: "u1", At: time.Now(), CreatedAt: time.Now(), Items: items}

	if err := s.SaveMeal(ctx(), meal); err == nil {
		t.Error("SaveMeal with too many items: want error")
	}
	if err := s.SaveMealAndAddToRollup(ctx(), meal, types.Macros{}); err == nil {
		t.Error("SaveMealAndAddToRollup with too many items: want error")
	}

	recent, err := s.RecentMeals(ctx(), "u1", 10)
	if err != nil {
		t.Fatalf("RecentMeals: %v", err)
	}
	if len(recent) != 0 {
		t.Fatalf("expected no meal persisted when over the item limit, got %d", len(recent))
	}
}
