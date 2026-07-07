package types

import "testing"

func TestMacrosSub(t *testing.T) {
	targets := Macros{Calories: 2000, Protein: 150, Carbs: 200, Fat: 60, Fiber: 30}
	consumed := Macros{Calories: 1200, Protein: 80, Carbs: 100, Fat: 40, Fiber: 10}

	got := targets.Sub(consumed)
	want := Macros{Calories: 800, Protein: 70, Carbs: 100, Fat: 20, Fiber: 20}
	if got != want {
		t.Errorf("Sub() = %+v, want %+v", got, want)
	}
}

func TestMacrosSubGoesNegativeWhenOverTarget(t *testing.T) {
	targets := Macros{Calories: 2000}
	consumed := Macros{Calories: 2500}

	got := targets.Sub(consumed)
	if got.Calories != -500 {
		t.Errorf("Sub().Calories = %v, want -500", got.Calories)
	}
}
