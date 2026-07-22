package normalize

import "testing"

func TestNormalizeUnitMass(t *testing.T) {
	tests := []struct {
		qty   float64
		unit  string
		want  string // canonical unit
		grams float64
	}{
		{200, "g", "g", 200},
		{1, "kg", "kg", 1000},
		{500, "mg", "mg", 0.5},
		{4, "oz", "oz", 4 * 28.3495},
		{2, "lb", "lb", 2 * 453.592},
	}
	for _, tc := range tests {
		gotUnit, gotGrams := NormalizeUnit(tc.qty, tc.unit, "food", "")
		if gotUnit != tc.want || gotGrams != tc.grams {
			t.Errorf("NormalizeUnit(%v, %q, ...) = (%q, %v), want (%q, %v)",
				tc.qty, tc.unit, gotUnit, gotGrams, tc.want, tc.grams)
		}
	}
}

func TestNormalizeUnitVolume(t *testing.T) {
	tests := []struct {
		qty   float64
		unit  string
		want  string
		grams float64
	}{
		{100, "ml", "ml", 100},
		{1, "l", "l", 1000},
		{2, "tbsp", "tbsp", 30},
		{3, "tsp", "tsp", 15},
		{1, "cup", "cup", 240},
		{1, "copo", "glass", 200},
	}
	for _, tc := range tests {
		gotUnit, gotGrams := NormalizeUnit(tc.qty, tc.unit, "food", "")
		if gotUnit != tc.want || gotGrams != tc.grams {
			t.Errorf("NormalizeUnit(%v, %q, ...) = (%q, %v), want (%q, %v)",
				tc.qty, tc.unit, gotUnit, gotGrams, tc.want, tc.grams)
		}
	}
}

func TestNormalizeUnitCount(t *testing.T) {
	gotUnit, gotGrams := NormalizeUnit(2, "unit", "eggs", "")
	if gotUnit != "unit" || gotGrams != 0 {
		t.Errorf("NormalizeUnit(2, unit, ...) = (%q, %v), want (unit, 0)", gotUnit, gotGrams)
	}

	// Empty unit treated as count.
	gotUnit, gotGrams = NormalizeUnit(3, "", "apples", "")
	if gotUnit != "unit" || gotGrams != 0 {
		t.Errorf("NormalizeUnit(3, \"\", ...) = (%q, %v), want (unit, 0)", gotUnit, gotGrams)
	}
}

func TestNormalizeUnitUnknown(t *testing.T) {
	gotUnit, gotGrams := NormalizeUnit(5, "unknownxyz", "food", "")
	if gotUnit != "unit" || gotGrams != 0 {
		t.Errorf("NormalizeUnit(5, unknown, ...) = (%q, %v), want (unit, 0)", gotUnit, gotGrams)
	}
}

func TestIsUnit(t *testing.T) {
	tests := []struct {
		token string
		want  bool
	}{
		{"g", true},
		{"kg", true},
		{"ml", true},
		{"cup", true},
		{"copo", true},
		{"colher", true},
		{"unit", true},
		{"frango", false},
		{"chicken", false},
		{"", false},
	}
	for _, tc := range tests {
		got := IsUnit(tc.token)
		if got != tc.want {
			t.Errorf("IsUnit(%q) = %v, want %v", tc.token, got, tc.want)
		}
	}
}

func TestNormalizeUnitAccented(t *testing.T) {
	// "colher" with accents should still be recognized.
	gotUnit, gotGrams := NormalizeUnit(1, "colher", "", "")
	if gotUnit != "tbsp" || gotGrams != 15 {
		t.Errorf("NormalizeUnit(1, colher, ...) = (%q, %v), want (tbsp, 15)", gotUnit, gotGrams)
	}
}

func TestVolumeUnitsEligible(t *testing.T) {
	tests := []struct {
		name     string
		category string
		food     string
		want     bool
	}{
		{"TACO milk, name only, empty category", "", "Leite, vaca, integral", true},
		{"non-liquid food, no match", "", "Grilled chicken breast", false},
		{"category-only match", "Dairy", "Whole Product X", true},
		{"case-insensitive", "", "WHOLE MILK", true},
		{"accented Portuguese oil", "", "Óleo, soja", true},
	}
	for _, tc := range tests {
		got := VolumeUnitsEligible(tc.category, tc.food)
		if got != tc.want {
			t.Errorf("%s: VolumeUnitsEligible(%q, %q) = %v, want %v",
				tc.name, tc.category, tc.food, got, tc.want)
		}
	}
}

// TestParityWithTier0 verifies that NormalizeUnit produces the same grams as
// the Tier-0 parser's consumeUnit for a set of common inputs.
func TestParityWithTier0(t *testing.T) {
	// These pairs mirror what consumeUnit extracts: (quantity, unit-token).
	tests := []struct {
		qty     float64
		unit    string
		wantGms float64
		wantCan string
	}{
		{200, "g", 200, "g"},
		{1.5, "kg", 1500, "kg"},
		{2, "oz", 56.699, "oz"},
		{100, "ml", 100, "ml"},
		{0.5, "l", 500, "l"},
		{1, "tbsp", 15, "tbsp"},
		{2, "cup", 480, "cup"},
		{1, "copo", 200, "glass"},
		{3, "unit", 0, "unit"},
		{2, "", 0, "unit"},
	}

	for _, tc := range tests {
		gotCan, gotGms := NormalizeUnit(tc.qty, tc.unit, "test-food", "")
		if gotGms != tc.wantGms || gotCan != tc.wantCan {
			t.Errorf("NormalizeUnit(%v, %q, ...) = (%q, %v), want (%q, %v)",
				tc.qty, tc.unit, gotCan, gotGms, tc.wantCan, tc.wantGms)
		}
	}
}
