package store

import (
	"errors"
	"testing"
	"time"

	"github.com/gsaraiva2109/dietdaemon/core/types"
)

// TestTargetsForNoPlanFallback confirms that a user with neither an override
// nor an active plan gets exactly what GetTargets returns — the invariant
// that no-plan users are byte-identical to pre-diet-plan behavior.
func TestTargetsForNoPlanFallback(t *testing.T) {
	s, cleanup := tempDB(t)
	defer cleanup()
	mustPlanUser(t, s, "u1")

	if _, err := s.TargetsFor(ctx(), "u1", "2026-03-01"); !errors.Is(err, types.ErrNotFound) {
		t.Fatalf("TargetsFor with nothing set = %v, want ErrNotFound", err)
	}

	flat := types.DailyTargets{UserID: "u1", Targets: types.Macros{Calories: 2000, Protein: 150, Carbs: 200, Fat: 60, Fiber: 30}, WaterGoalMl: 2500}
	if err := s.SetTargets(ctx(), flat); err != nil {
		t.Fatalf("SetTargets: %v", err)
	}
	got, err := s.TargetsFor(ctx(), "u1", "2026-03-01")
	if err != nil {
		t.Fatalf("TargetsFor: %v", err)
	}
	if got.Targets != flat.Targets || got.WaterGoalMl != flat.WaterGoalMl {
		t.Fatalf("TargetsFor no-plan fallback = %+v, want %+v", got, flat)
	}
}

// TestTargetsForCyclePattern exercises the plan's cycle-pattern resolution,
// including dates before cycle_anchor_date, where a naive (offset % len)
// index would be negative instead of Euclidean.
func TestTargetsForCyclePattern(t *testing.T) {
	s, cleanup := tempDB(t)
	defer cleanup()
	mustPlanUser(t, s, "u1")

	p, err := s.CreatePlan(ctx(), types.DietPlan{
		UserID: "u1", Name: "Cycle", ValidFrom: "2020-01-01", ValidTo: "",
		CyclePattern: []string{"placeholder-a", "placeholder-b"}, CycleAnchorDate: "2026-01-05",
	})
	if err != nil {
		t.Fatalf("CreatePlan: %v", err)
	}
	dtLow, err := s.CreateDayType(ctx(), types.DietPlanDayType{PlanID: p.ID, Name: "Low", Targets: types.Macros{Calories: 1800}, WaterGoalMl: 2000})
	if err != nil {
		t.Fatalf("CreateDayType low: %v", err)
	}
	dtHigh, err := s.CreateDayType(ctx(), types.DietPlanDayType{PlanID: p.ID, Name: "High", Targets: types.Macros{Calories: 2600}, WaterGoalMl: 3200})
	if err != nil {
		t.Fatalf("CreateDayType high: %v", err)
	}
	p.CyclePattern = []string{dtLow.ID, dtHigh.ID}
	if err := s.UpdatePlan(ctx(), p); err != nil {
		t.Fatalf("UpdatePlan: %v", err)
	}

	cases := []struct {
		date string
		want types.DietPlanDayType
	}{
		{"2026-01-05", dtLow},  // offset 0 -> index 0
		{"2026-01-06", dtHigh}, // offset 1 -> index 1
		{"2026-01-07", dtLow},  // offset 2 -> index 0 (wraps)
		{"2026-01-04", dtHigh}, // offset -1 -> Euclidean index 1, not -1
		{"2026-01-03", dtLow},  // offset -2 -> Euclidean index 0
	}
	for _, tc := range cases {
		got, err := s.TargetsFor(ctx(), "u1", tc.date)
		if err != nil {
			t.Fatalf("TargetsFor(%s): %v", tc.date, err)
		}
		if got.Targets.Calories != tc.want.Targets.Calories || got.WaterGoalMl != tc.want.WaterGoalMl {
			t.Errorf("TargetsFor(%s) = %+v, want day-type %q (%+v)", tc.date, got, tc.want.Name, tc.want.Targets)
		}
	}
}

// TestTargetsForOverridePrecedence confirms a day override wins over
// whatever the plan's cycle pattern would otherwise resolve for that date.
func TestTargetsForOverridePrecedence(t *testing.T) {
	s, cleanup := tempDB(t)
	defer cleanup()
	mustPlanUser(t, s, "u1")

	p, err := s.CreatePlan(ctx(), types.DietPlan{
		UserID: "u1", Name: "Cycle", ValidFrom: "2020-01-01", ValidTo: "",
		CyclePattern: []string{"x"}, CycleAnchorDate: "2026-01-05",
	})
	if err != nil {
		t.Fatalf("CreatePlan: %v", err)
	}
	dtPattern, err := s.CreateDayType(ctx(), types.DietPlanDayType{PlanID: p.ID, Name: "Pattern day", Targets: types.Macros{Calories: 2000}})
	if err != nil {
		t.Fatalf("CreateDayType pattern: %v", err)
	}
	dtOverride, err := s.CreateDayType(ctx(), types.DietPlanDayType{PlanID: p.ID, Name: "Override day", Targets: types.Macros{Calories: 3200}})
	if err != nil {
		t.Fatalf("CreateDayType override: %v", err)
	}
	p.CyclePattern = []string{dtPattern.ID}
	if err := s.UpdatePlan(ctx(), p); err != nil {
		t.Fatalf("UpdatePlan: %v", err)
	}

	const date = "2026-01-10"
	if got, err := s.TargetsFor(ctx(), "u1", date); err != nil || got.Targets.Calories != 2000 {
		t.Fatalf("TargetsFor before override = %+v, err = %v, want 2000 kcal", got, err)
	}

	if err := s.SetDayOverride(ctx(), types.DietPlanDayOverride{UserID: "u1", Date: date, DayTypeID: dtOverride.ID}); err != nil {
		t.Fatalf("SetDayOverride: %v", err)
	}
	got, err := s.TargetsFor(ctx(), "u1", date)
	if err != nil {
		t.Fatalf("TargetsFor after override: %v", err)
	}
	if got.Targets.Calories != 3200 {
		t.Fatalf("TargetsFor with override = %+v, want the override day-type's 3200 kcal", got)
	}
}

// TestTargetsForPlanBoundaryDates confirms valid_from/valid_to are inclusive
// and that dates outside the range fall through to the flat fallback.
func TestTargetsForPlanBoundaryDates(t *testing.T) {
	s, cleanup := tempDB(t)
	defer cleanup()
	mustPlanUser(t, s, "u1")

	flat := types.DailyTargets{UserID: "u1", Targets: types.Macros{Calories: 1500}}
	if err := s.SetTargets(ctx(), flat); err != nil {
		t.Fatalf("SetTargets: %v", err)
	}

	p, err := s.CreatePlan(ctx(), types.DietPlan{
		UserID: "u1", Name: "Bounded", ValidFrom: "2026-01-01", ValidTo: "2026-01-31",
		CyclePattern: []string{"x"}, CycleAnchorDate: "2026-01-01",
	})
	if err != nil {
		t.Fatalf("CreatePlan: %v", err)
	}
	dt, err := s.CreateDayType(ctx(), types.DietPlanDayType{PlanID: p.ID, Name: "Plan day", Targets: types.Macros{Calories: 2200}})
	if err != nil {
		t.Fatalf("CreateDayType: %v", err)
	}
	p.CyclePattern = []string{dt.ID}
	if err := s.UpdatePlan(ctx(), p); err != nil {
		t.Fatalf("UpdatePlan: %v", err)
	}

	for _, date := range []string{"2026-01-01", "2026-01-31"} {
		got, err := s.TargetsFor(ctx(), "u1", date)
		if err != nil || got.Targets.Calories != 2200 {
			t.Errorf("TargetsFor(%s) = %+v, err = %v, want plan's 2200 kcal (boundary inclusive)", date, got, err)
		}
	}
	for _, date := range []string{"2025-12-31", "2026-02-01"} {
		got, err := s.TargetsFor(ctx(), "u1", date)
		if err != nil || got.Targets.Calories != 1500 {
			t.Errorf("TargetsFor(%s) = %+v, err = %v, want flat fallback 1500 kcal (outside plan range)", date, got, err)
		}
	}
}

// TestRefreshTodayTargets covers both halves of the helper: a no-op when no
// override or plan governs today, and writing today's rollup to the
// governing day-type's numbers (leaving any other day's rollup untouched)
// when one does.
func TestRefreshTodayTargets(t *testing.T) {
	s, cleanup := tempDB(t)
	defer cleanup()
	mustPlanUser(t, s, "u1")

	today := time.Now().UTC().Format(dateLayout)
	yesterday := time.Now().UTC().AddDate(0, 0, -1).Format(dateLayout)

	// No plan, no override: no-op, no rollup row created.
	if err := s.RefreshTodayTargets(ctx(), "u1"); err != nil {
		t.Fatalf("RefreshTodayTargets no-op: %v", err)
	}
	if _, err := s.GetRollup(ctx(), "u1", today); !errors.Is(err, types.ErrNotFound) {
		t.Fatalf("GetRollup after no-op refresh = %v, want ErrNotFound", err)
	}

	// Seed yesterday's rollup with its own frozen targets, to prove the
	// refresh only ever touches today.
	if err := s.UpsertRollup(ctx(), types.DailyRollup{UserID: "u1", Date: yesterday, Targets: types.Macros{Calories: 1111}}); err != nil {
		t.Fatalf("seed yesterday rollup: %v", err)
	}

	p, err := s.CreatePlan(ctx(), types.DietPlan{
		UserID: "u1", Name: "Today's plan", ValidFrom: "2020-01-01", ValidTo: "",
		CyclePattern: []string{"x"}, CycleAnchorDate: today,
	})
	if err != nil {
		t.Fatalf("CreatePlan: %v", err)
	}
	dt, err := s.CreateDayType(ctx(), types.DietPlanDayType{PlanID: p.ID, Name: "Today", Targets: types.Macros{Calories: 2750}})
	if err != nil {
		t.Fatalf("CreateDayType: %v", err)
	}
	p.CyclePattern = []string{dt.ID}
	if err := s.UpdatePlan(ctx(), p); err != nil {
		t.Fatalf("UpdatePlan: %v", err)
	}

	if err := s.RefreshTodayTargets(ctx(), "u1"); err != nil {
		t.Fatalf("RefreshTodayTargets with active plan: %v", err)
	}
	gotToday, err := s.GetRollup(ctx(), "u1", today)
	if err != nil {
		t.Fatalf("GetRollup today after refresh: %v", err)
	}
	if gotToday.Targets.Calories != 2750 {
		t.Fatalf("today's rollup targets = %+v, want the day-type's 2750 kcal", gotToday.Targets)
	}
	gotYesterday, err := s.GetRollup(ctx(), "u1", yesterday)
	if err != nil {
		t.Fatalf("GetRollup yesterday: %v", err)
	}
	if gotYesterday.Targets.Calories != 1111 {
		t.Fatalf("yesterday's rollup targets changed = %+v, want untouched 1111 kcal", gotYesterday.Targets)
	}
}
