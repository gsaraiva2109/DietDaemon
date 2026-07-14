package mfp

import (
	"strings"
	"testing"
)

const sampleCSV = `Date,Meal,Food,Serving Size,Calories,Fat (g),Saturated Fat,Cholesterol,Sodium (mg),Carbohydrates (g),Fiber,Sugar,Protein (g)
2024-01-15,Breakfast,Oatmeal,1 cup,150,3,0.5,0,120,27,4,1,5
2024-01-15,Breakfast,Banana,1 medium,105,0.4,0,0,1,27,3.1,14,1.3
2024-01-15,Lunch,Grilled Chicken Breast,6 oz,280,6,1.7,145,380,0,0,0,53
`

func TestParseCSV(t *testing.T) {
	rows, err := ParseCSV(strings.NewReader(sampleCSV))
	if err != nil {
		t.Fatalf("ParseCSV: %v", err)
	}
	if len(rows) != 3 {
		t.Fatalf("len(rows) = %d, want 3", len(rows))
	}

	first := rows[0]
	if first.Date != "2024-01-15" {
		t.Errorf("Date = %q, want 2024-01-15", first.Date)
	}
	if first.Meal != "Breakfast" {
		t.Errorf("Meal = %q, want Breakfast", first.Meal)
	}
	if first.Food != "Oatmeal" {
		t.Errorf("Food = %q, want Oatmeal", first.Food)
	}
	if first.ServingSize != "1 cup" {
		t.Errorf("ServingSize = %q, want %q", first.ServingSize, "1 cup")
	}
	if first.Calories != 150 {
		t.Errorf("Calories = %v, want 150", first.Calories)
	}
	if first.FatG != 3 {
		t.Errorf("FatG = %v, want 3", first.FatG)
	}
	if first.CarbsG != 27 {
		t.Errorf("CarbsG = %v, want 27", first.CarbsG)
	}
	if first.FiberG != 4 {
		t.Errorf("FiberG = %v, want 4", first.FiberG)
	}
	if first.ProteinG != 5 {
		t.Errorf("ProteinG = %v, want 5", first.ProteinG)
	}

	// Sodium/Cholesterol/Sugar columns exist in the header but aren't part
	// of Row — confirms unmapped columns don't corrupt mapped ones (the
	// col[key] zero-value bug this guards against would misread column 0).
	third := rows[2]
	if third.Food != "Grilled Chicken Breast" {
		t.Errorf("Food = %q, want Grilled Chicken Breast", third.Food)
	}
	if third.ProteinG != 53 {
		t.Errorf("ProteinG = %v, want 53", third.ProteinG)
	}
}

func TestParseCSV_MissingRequiredColumn(t *testing.T) {
	const badCSV = "Meal,Food,Calories\nBreakfast,Oatmeal,150\n"
	if _, err := ParseCSV(strings.NewReader(badCSV)); err == nil {
		t.Fatal("expected error for csv missing Date column, got nil")
	}
}

func TestParseCSV_HeaderVariants(t *testing.T) {
	// Case-insensitive and unit-suffix tolerant, matching different MFP
	// export vintages.
	const variantCSV = "DATE,MEAL,FOOD NAME,Calories,PROTEIN\n2024-02-01,Dinner,Salmon,400,40\n"
	rows, err := ParseCSV(strings.NewReader(variantCSV))
	if err != nil {
		t.Fatalf("ParseCSV: %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("len(rows) = %d, want 1", len(rows))
	}
	if rows[0].Food != "Salmon" || rows[0].ProteinG != 40 {
		t.Errorf("row = %+v, want Food=Salmon ProteinG=40", rows[0])
	}
}

func TestToItem(t *testing.T) {
	row := Row{
		Date: "2024-01-15", Meal: "Breakfast", Food: "Oatmeal", ServingSize: "1 cup",
		Calories: 150, FatG: 3, CarbsG: 27, FiberG: 4, ProteinG: 5,
	}
	item := ToItem(row)

	if item.Parsed.RawPhrase != "Oatmeal" {
		t.Errorf("RawPhrase = %q, want Oatmeal", item.Parsed.RawPhrase)
	}
	if item.Parsed.Unit != "1 cup" {
		t.Errorf("Unit = %q, want %q", item.Parsed.Unit, "1 cup")
	}
	if item.Match.Source != "mfp_import" {
		t.Errorf("Source = %q, want mfp_import", item.Match.Source)
	}
	if item.Match.Name != "Oatmeal" {
		t.Errorf("Match.Name = %q, want Oatmeal", item.Match.Name)
	}
	if item.Macros.Calories != 150 || item.Macros.Protein != 5 || item.Macros.Carbs != 27 || item.Macros.Fat != 3 || item.Macros.Fiber != 4 {
		t.Errorf("Macros = %+v, want {150 5 27 3 4}", item.Macros)
	}
}
