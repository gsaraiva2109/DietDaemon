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

func TestMacrosAdd(t *testing.T) {
	a := Macros{Calories: 100, Protein: 10, Carbs: 20, Fat: 5, Fiber: 2}
	b := Macros{Calories: 200, Protein: 15, Carbs: 30, Fat: 8, Fiber: 3}

	got := a.Add(b)
	want := Macros{Calories: 300, Protein: 25, Carbs: 50, Fat: 13, Fiber: 5}
	if got != want {
		t.Errorf("Add() = %+v, want %+v", got, want)
	}
}

func TestMacrosScale(t *testing.T) {
	per100g := Macros{Calories: 200, Protein: 20, Carbs: 10, Fat: 8, Fiber: 4}

	got := per100g.Scale(1.5)
	want := Macros{Calories: 300, Protein: 30, Carbs: 15, Fat: 12, Fiber: 6}
	if got != want {
		t.Errorf("Scale(1.5) = %+v, want %+v", got, want)
	}
}

func TestMacrosScaleByZero(t *testing.T) {
	m := Macros{Calories: 200, Protein: 20, Carbs: 10, Fat: 8, Fiber: 4}

	got := m.Scale(0)
	if got != (Macros{}) {
		t.Errorf("Scale(0) = %+v, want zero value", got)
	}
}

func TestMealTotal(t *testing.T) {
	meal := Meal{
		Items: []ResolvedItem{
			{Macros: Macros{Calories: 100, Protein: 10, Carbs: 5, Fat: 2, Fiber: 1}},
			{Macros: Macros{Calories: 200, Protein: 20, Carbs: 10, Fat: 4, Fiber: 2}},
		},
	}

	got := meal.Total()
	want := Macros{Calories: 300, Protein: 30, Carbs: 15, Fat: 6, Fiber: 3}
	if got != want {
		t.Errorf("Total() = %+v, want %+v", got, want)
	}
}

func TestMealTotalEmpty(t *testing.T) {
	meal := Meal{}

	got := meal.Total()
	if got != (Macros{}) {
		t.Errorf("Total() on empty meal = %+v, want zero value", got)
	}
}
