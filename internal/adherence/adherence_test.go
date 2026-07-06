package adherence

import (
	"testing"

	"github.com/gsaraiva2109/dietdaemon/core/types"
)

func TestStreak_AllInBand(t *testing.T) {
	rollups := []types.DailyRollup{
		{Date: "2026-07-01", Consumed: types.Macros{Calories: 2200}, Targets: types.Macros{Calories: 2200}},
		{Date: "2026-07-02", Consumed: types.Macros{Calories: 2100}, Targets: types.Macros{Calories: 2200}},
		{Date: "2026-07-03", Consumed: types.Macros{Calories: 2300}, Targets: types.Macros{Calories: 2200}},
	}
	if got := Streak(rollups, 0.90, 1.10); got != 3 {
		t.Errorf("Streak = %d, want 3", got)
	}
}

func TestStreak_OutOfBandStops(t *testing.T) {
	rollups := []types.DailyRollup{
		{Date: "2026-07-01", Consumed: types.Macros{Calories: 2200}, Targets: types.Macros{Calories: 2200}},
		{Date: "2026-07-02", Consumed: types.Macros{Calories: 1500}, Targets: types.Macros{Calories: 2200}}, // 68% — below 90%
		{Date: "2026-07-03", Consumed: types.Macros{Calories: 2200}, Targets: types.Macros{Calories: 2200}},
	}
	if got := Streak(rollups, 0.90, 1.10); got != 1 {
		t.Errorf("Streak = %d, want 1 (stops at out-of-band day)", got)
	}
}

func TestStreak_MissingTarget(t *testing.T) {
	rollups := []types.DailyRollup{
		{Date: "2026-07-01", Consumed: types.Macros{Calories: 2200}, Targets: types.Macros{Calories: 2200}},
		{Date: "2026-07-02", Consumed: types.Macros{Calories: 2200}, Targets: types.Macros{}}, // no target
		{Date: "2026-07-03", Consumed: types.Macros{Calories: 2200}, Targets: types.Macros{Calories: 2200}},
	}
	if got := Streak(rollups, 0.90, 1.10); got != 1 {
		t.Errorf("Streak = %d, want 1 (stops at missing target)", got)
	}
}

func TestStreak_DateGap(t *testing.T) {
	rollups := []types.DailyRollup{
		{Date: "2026-07-01", Consumed: types.Macros{Calories: 2200}, Targets: types.Macros{Calories: 2200}},
		{Date: "2026-07-03", Consumed: types.Macros{Calories: 2200}, Targets: types.Macros{Calories: 2200}}, // gap: 07-02 missing
	}
	if got := Streak(rollups, 0.90, 1.10); got != 1 {
		t.Errorf("Streak = %d, want 1 (stops at date gap)", got)
	}
}

func TestStreak_Empty(t *testing.T) {
	if got := Streak(nil, 0.90, 1.10); got != 0 {
		t.Errorf("Streak = %d, want 0 (empty)", got)
	}
}

func TestStreak_AboveCeilStops(t *testing.T) {
	// Most recent day (07-02) is above ceiling — streak is 0 because the
	// backward walk starts at the end and stops immediately.
	rollups := []types.DailyRollup{
		{Date: "2026-07-01", Consumed: types.Macros{Calories: 2200}, Targets: types.Macros{Calories: 2200}},
		{Date: "2026-07-02", Consumed: types.Macros{Calories: 3000}, Targets: types.Macros{Calories: 2200}}, // 136% — above 110%
	}
	if got := Streak(rollups, 0.90, 1.10); got != 0 {
		t.Errorf("Streak = %d, want 0 (most recent day out of band)", got)
	}
}

func TestStreak_ResumesAfterMidBreak(t *testing.T) {
	// Break in the middle, streak resumes — backward walk from end counts the
	// resumed streak, not the first one. Forward walk would return 2.
	rollups := []types.DailyRollup{
		{Date: "2026-07-01", Consumed: types.Macros{Calories: 2200}, Targets: types.Macros{Calories: 2200}},
		{Date: "2026-07-02", Consumed: types.Macros{Calories: 2200}, Targets: types.Macros{Calories: 2200}},
		{Date: "2026-07-03", Consumed: types.Macros{Calories: 1000}, Targets: types.Macros{Calories: 2200}}, // break
		{Date: "2026-07-04", Consumed: types.Macros{Calories: 2200}, Targets: types.Macros{Calories: 2200}},
		{Date: "2026-07-05", Consumed: types.Macros{Calories: 2200}, Targets: types.Macros{Calories: 2200}},
	}
	if got := Streak(rollups, 0.90, 1.10); got != 2 {
		t.Errorf("Streak = %d, want 2 (backward walk: 07-05, 07-04, stops at 07-03)", got)
	}
}

func TestStreak_ExactBoundary(t *testing.T) {
	// 90% of 2200 = 1980, 110% = 2420
	rollups := []types.DailyRollup{
		{Date: "2026-07-01", Consumed: types.Macros{Calories: 1980}, Targets: types.Macros{Calories: 2200}}, // exactly 90%
		{Date: "2026-07-02", Consumed: types.Macros{Calories: 2420}, Targets: types.Macros{Calories: 2200}}, // exactly 110%
	}
	if got := Streak(rollups, 0.90, 1.10); got != 2 {
		t.Errorf("Streak = %d, want 2 (boundary values count)", got)
	}
}
