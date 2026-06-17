package taco

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/xuri/excelize/v2"

	"github.com/gsaraiva2109/dietdaemon/core/types"
)

// ---------------------------------------------------------------------------
// Shared resolve tests — run against a Source regardless of format.
// ---------------------------------------------------------------------------

func testResolve(t *testing.T, src *Source) {
	t.Helper()
	ctx := context.Background()

	if src.Name() != "taco" {
		t.Errorf("Name() = %q, want taco", src.Name())
	}

	// Exact match.
	fm, err := src.Resolve(ctx, types.ParsedItem{RawPhrase: "Frango grelhado"})
	if err != nil {
		t.Fatalf("Resolve Frango grelhado: %v", err)
	}
	if fm.FoodID != "TACO005" || fm.Per100g.Calories != 165 || fm.Per100g.Protein != 31 {
		t.Errorf("got %+v", fm)
	}

	// Accented variant.
	fm, err = src.Resolve(ctx, types.ParsedItem{RawPhrase: "Feijão carioca cozido"})
	if err != nil {
		t.Fatalf("Resolve Feijão: %v", err)
	}
	if fm.FoodID != "TACO003" {
		t.Errorf("expected TACO003, got %s", fm.FoodID)
	}

	// Unaccented input.
	fm, err = src.Resolve(ctx, types.ParsedItem{RawPhrase: "feijao carioca cozido"})
	if err != nil {
		t.Fatalf("Resolve feijao: %v", err)
	}
	if fm.FoodID != "TACO003" {
		t.Errorf("expected TACO003, got %s", fm.FoodID)
	}

	// Miss.
	_, err = src.Resolve(ctx, types.ParsedItem{RawPhrase: "pizza"})
	if err != types.ErrNoMatch {
		t.Errorf("expected ErrNoMatch, got %v", err)
	}

	// Empty phrase.
	_, err = src.Resolve(ctx, types.ParsedItem{RawPhrase: ""})
	if err != types.ErrNoMatch {
		t.Errorf("expected ErrNoMatch for empty, got %v", err)
	}
}

// ---------------------------------------------------------------------------
// CSV path
// ---------------------------------------------------------------------------

func TestCSVLoadAndResolve(t *testing.T) {
	path := filepath.Join("..", "..", "..", "testdata", "taco_sample.csv")
	src, err := New(path)
	if err != nil {
		t.Fatalf("New csv: %v", err)
	}
	testResolve(t, src)
}

// ---------------------------------------------------------------------------
// XLSX path
// ---------------------------------------------------------------------------

func TestXLSXLoadAndResolve(t *testing.T) {
	// Generate a tiny XLSX fixture with the same schema as taco_sample.csv.
	dir := t.TempDir()
	path := filepath.Join(dir, "taco.xlsx")

	f := excelize.NewFile()
	// Header row.
	f.SetCellValue("Sheet1", "A1", "food_id")
	f.SetCellValue("Sheet1", "B1", "name")
	f.SetCellValue("Sheet1", "C1", "kcal")
	f.SetCellValue("Sheet1", "D1", "protein")
	f.SetCellValue("Sheet1", "E1", "carb")
	f.SetCellValue("Sheet1", "F1", "fat")
	f.SetCellValue("Sheet1", "G1", "fiber")
	// Data rows — subset of the CSV fixture.
	rows := [][]interface{}{
		{"TACO005", "Frango grelhado", 165, 31.0, 0.0, 3.6, 0.0},
		{"TACO003", "Feijão carioca cozido", 76, 4.8, 13.6, 0.5, 8.5},
		{"TACO007", "Ovo de galinha cozido", 155, 13.0, 1.1, 10.6, 0.0},
	}
	for i, row := range rows {
		for j, val := range row {
			cell, _ := excelize.CoordinatesToCellName(j+1, i+2)
			f.SetCellValue("Sheet1", cell, val)
		}
	}
	if err := f.SaveAs(path); err != nil {
		t.Fatalf("save xlsx: %v", err)
	}
	f.Close()

	src, err := New(path)
	if err != nil {
		t.Fatalf("New xlsx: %v", err)
	}
	testResolve(t, src)
}

// ---------------------------------------------------------------------------
// Synthetic mini-files
// ---------------------------------------------------------------------------

func TestTinySyntheticCSV(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "mini.csv")
	data := "food_id,name,kcal,protein,carb,fat,fiber\nX001,Test Food,100,10,5,2,1\n"
	if err := os.WriteFile(path, []byte(data), 0644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}

	src, err := New(path)
	if err != nil {
		t.Fatalf("New from synthetic csv: %v", err)
	}

	fm, err := src.Resolve(context.Background(), types.ParsedItem{RawPhrase: "Test Food"})
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if fm.FoodID != "X001" || fm.Per100g.Calories != 100 {
		t.Errorf("unexpected result: %+v", fm)
	}
}

func TestTinySyntheticXLSX(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "mini.xlsx")

	f := excelize.NewFile()
	f.SetCellValue("Sheet1", "A1", "food_id")
	f.SetCellValue("Sheet1", "B1", "name")
	f.SetCellValue("Sheet1", "C1", "kcal")
	f.SetCellValue("Sheet1", "D1", "protein")
	f.SetCellValue("Sheet1", "E1", "carb")
	f.SetCellValue("Sheet1", "F1", "fat")
	f.SetCellValue("Sheet1", "G1", "fiber")
	f.SetCellValue("Sheet1", "A2", "Y001")
	f.SetCellValue("Sheet1", "B2", "Synthetic Food")
	f.SetCellValue("Sheet1", "C2", 200)
	f.SetCellValue("Sheet1", "D2", 15)
	f.SetCellValue("Sheet1", "E2", 10)
	f.SetCellValue("Sheet1", "F2", 5)
	f.SetCellValue("Sheet1", "G2", 3)
	if err := f.SaveAs(path); err != nil {
		t.Fatalf("save xlsx: %v", err)
	}
	f.Close()

	src, err := New(path)
	if err != nil {
		t.Fatalf("New from synthetic xlsx: %v", err)
	}

	fm, err := src.Resolve(context.Background(), types.ParsedItem{RawPhrase: "Synthetic Food"})
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if fm.FoodID != "Y001" || fm.Per100g.Calories != 200 {
		t.Errorf("unexpected result: %+v", fm)
	}
}

// ---------------------------------------------------------------------------
// Error paths
// ---------------------------------------------------------------------------

func TestNewMissingFile(t *testing.T) {
	_, err := New("/nonexistent/taco.csv")
	if err == nil {
		t.Error("expected error for missing file")
	}
}

func TestUnsupportedExtension(t *testing.T) {
	_, err := New("/tmp/foo.json")
	if err == nil {
		t.Error("expected error for .json extension")
	}
}

// ---------------------------------------------------------------------------
// Independent loads
// ---------------------------------------------------------------------------

func TestIndependentLoads(t *testing.T) {
	path := filepath.Join("..", "..", "..", "testdata", "taco_sample.csv")
	s1, err := New(path)
	if err != nil {
		t.Fatal(err)
	}
	s2, err := New(path)
	if err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()
	fm1, _ := s1.Resolve(ctx, types.ParsedItem{RawPhrase: "arroz branco cozido"})
	fm2, _ := s2.Resolve(ctx, types.ParsedItem{RawPhrase: "arroz branco cozido"})
	if fm1 != fm2 {
		t.Error("two loads should return same data")
	}
}
