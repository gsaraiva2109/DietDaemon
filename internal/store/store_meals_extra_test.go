package store

import (
	"errors"
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
