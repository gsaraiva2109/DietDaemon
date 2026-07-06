package scheduler

import (
	"math"
	"testing"
)

func TestEffectiveWeeklyTarget_Monday_PlainTarget(t *testing.T) {
	// Monday: 0 prior days consumed, 7 days remaining → effective = plain.
	got := EffectiveWeeklyTarget(2200, 0, 7, 0.70, 1.30)
	if math.Abs(got-2200) > 0.01 {
		t.Errorf("Monday should equal plain target, got %.1f", got)
	}
}

func TestEffectiveWeeklyTarget_MidweekOvereating_LowersTarget(t *testing.T) {
	// Weekly target = 2200 * 7 = 15400. Ate 5000 over first 2 days (Mon-Tue).
	// Remaining = 15400 - 5000 = 10400 over 5 days = 2080.
	got := EffectiveWeeklyTarget(2200, 5000, 5, 0.70, 1.30)
	if math.Abs(got-2080) > 0.01 {
		t.Errorf("overeating should lower target, got %.1f, want 2080", got)
	}
}

func TestEffectiveWeeklyTarget_MidweekUndereating_RaisesTarget(t *testing.T) {
	// Weekly target = 2200 * 7 = 15400. Ate only 3000 over first 2 days.
	// Remaining = 15400 - 3000 = 12400 over 5 days = 2480.
	got := EffectiveWeeklyTarget(2200, 3000, 5, 0.70, 1.30)
	if math.Abs(got-2480) > 0.01 {
		t.Errorf("undereating should raise target, got %.1f, want 2480", got)
	}
}

func TestEffectiveWeeklyTarget_FloorClamp(t *testing.T) {
	// Weekly target = 2200 * 7 = 15400. Ate 14000 over first 2 days (massive binge).
	// Remaining = 15400 - 14000 = 1400 over 5 days = 280.
	// Floor at 70% of 2200 = 1540 → clamped to 1540.
	got := EffectiveWeeklyTarget(2200, 14000, 5, 0.70, 1.30)
	if math.Abs(got-1540) > 0.01 {
		t.Errorf("floor clamp should apply, got %.1f, want 1540", got)
	}
}

func TestEffectiveWeeklyTarget_CeilingClamp(t *testing.T) {
	// Weekly target = 2200 * 7 = 15400. Ate only 500 over first 2 days (big deficit).
	// Remaining = 15400 - 500 = 14900 over 5 days = 2980.
	// Ceiling at 130% of 2200 = 2860 → clamped.
	got := EffectiveWeeklyTarget(2200, 500, 5, 0.70, 1.30)
	if math.Abs(got-2860) > 0.01 {
		t.Errorf("ceiling clamp should apply, got %.1f, want 2860", got)
	}
}

func TestEffectiveWeeklyTarget_LastDay(t *testing.T) {
	// Sunday: 6 prior days consumed, 1 day remaining.
	// Weekly target = 2200 * 7 = 15400. Ate 12000 over first 6 days.
	// Remaining = 15400 - 12000 = 3400. 1 day → 3400. Ceiling 130% = 2860.
	got := EffectiveWeeklyTarget(2200, 12000, 1, 0.70, 1.30)
	if math.Abs(got-2860) > 0.01 {
		t.Errorf("last day with ceiling, got %.1f, want 2860", got)
	}
}

func TestEffectiveWeeklyTarget_OverrideTarget(t *testing.T) {
	// WeeklyTargetOverride changes the weekly total used in the formula.
	// plainDaily=2200, override=2500 → weekly=2500*7=17500.
	// consumed=5000 over 2 days, 5 remaining.
	// effective = (17500 - 5000) / 5 = 2500.
	got := EffectiveWeeklyTarget(2500, 5000, 5, 0.70, 1.30)
	if math.Abs(got-2500) > 0.01 {
		t.Errorf("override target, got %.1f, want 2500", got)
	}
}
