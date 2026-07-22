package taco

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/xuri/excelize/v2"

	"github.com/gsaraiva2109/dietdaemon/core/ports"
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
	_ = f.SetCellValue("Sheet1", "A1", "food_id")
	_ = f.SetCellValue("Sheet1", "B1", "name")
	_ = f.SetCellValue("Sheet1", "C1", "kcal")
	_ = f.SetCellValue("Sheet1", "D1", "protein")
	_ = f.SetCellValue("Sheet1", "E1", "carb")
	_ = f.SetCellValue("Sheet1", "F1", "fat")
	_ = f.SetCellValue("Sheet1", "G1", "fiber")
	// Data rows — subset of the CSV fixture.
	rows := [][]any{
		{"TACO005", "Frango grelhado", 165, 31.0, 0.0, 3.6, 0.0},
		{"TACO003", "Feijão carioca cozido", 76, 4.8, 13.6, 0.5, 8.5},
		{"TACO007", "Ovo de galinha cozido", 155, 13.0, 1.1, 10.6, 0.0},
	}
	for i, row := range rows {
		for j, val := range row {
			cell, _ := excelize.CoordinatesToCellName(j+1, i+2)
			_ = f.SetCellValue("Sheet1", cell, val)
		}
	}
	if err := f.SaveAs(path); err != nil {
		t.Fatalf("save xlsx: %v", err)
	}
	_ = f.Close()

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
	_ = f.SetCellValue("Sheet1", "A1", "food_id")
	_ = f.SetCellValue("Sheet1", "B1", "name")
	_ = f.SetCellValue("Sheet1", "C1", "kcal")
	_ = f.SetCellValue("Sheet1", "D1", "protein")
	_ = f.SetCellValue("Sheet1", "E1", "carb")
	_ = f.SetCellValue("Sheet1", "F1", "fat")
	_ = f.SetCellValue("Sheet1", "G1", "fiber")
	_ = f.SetCellValue("Sheet1", "A2", "Y001")
	_ = f.SetCellValue("Sheet1", "B2", "Synthetic Food")
	_ = f.SetCellValue("Sheet1", "C2", 200)
	_ = f.SetCellValue("Sheet1", "D2", 15)
	_ = f.SetCellValue("Sheet1", "E2", 10)
	_ = f.SetCellValue("Sheet1", "F2", 5)
	_ = f.SetCellValue("Sheet1", "G2", 3)
	if err := f.SaveAs(path); err != nil {
		t.Fatalf("save xlsx: %v", err)
	}
	_ = f.Close()

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

// TestOfficialSpreadsheetLayout reproduces the corruption behind issue #111:
// pointing TACO_DATA_PATH at the raw official TACO/NEPA spreadsheet —
// moisture% and kJ columns land between name and protein, plus a
// food-group separator row with only column 0 populated — instead of the
// simplified schema. New must parse this layout correctly (via
// officialRowsToFoods), not silently shuffle macros into the wrong fields.
func TestOfficialSpreadsheetLayout(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "taco_official.xlsx")

	f := excelize.NewFile()
	header := []string{
		"Número do Alimento", "Descrição dos alimentos", "Umidade (%)", "Energia (kcal)",
		"Energia (kJ)", "Proteína (g)", "Lipídeos (g)", "Colesterol (mg)", "Carboidrato (g)", "Fibra Alimentar (g)",
	}
	for i, h := range header {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		_ = f.SetCellValue("Sheet1", cell, h)
	}
	_ = f.SetCellValue("Sheet1", "A2", "Oleaginosas e sementes")
	data := []any{558, "Amendoim, torrado, salgado", 1.7, 606.0, 2535.0, 22.5, 54.0, "NA", 18.7, 7.8}
	for j, val := range data {
		cell, _ := excelize.CoordinatesToCellName(j+1, 3)
		_ = f.SetCellValue("Sheet1", cell, val)
	}
	if err := f.SaveAs(path); err != nil {
		t.Fatalf("save xlsx: %v", err)
	}
	_ = f.Close()

	src, err := New(path)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	fm, err := src.Resolve(context.Background(), types.ParsedItem{RawPhrase: "Amendoim, torrado, salgado"})
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if fm.FoodID != "TACO558" {
		t.Errorf("FoodID = %q, want TACO558", fm.FoodID)
	}
	if fm.Per100g.Calories != 606 || fm.Per100g.Protein != 22.5 || fm.Per100g.Carbs != 18.7 ||
		fm.Per100g.Fat != 54 || fm.Per100g.Fiber != 7.8 {
		t.Errorf("got macros %+v, want 606/22.5/18.7/54/7.8", fm.Per100g)
	}
}

// TestOfficialSpreadsheetLayoutNoDataRows checks the loud-error fallback:
// a file matching neither the simplified schema nor the official layout
// (no row's first column parses as an integer food ID) still fails clearly
// instead of silently returning an empty catalog.
func TestOfficialSpreadsheetLayoutNoDataRows(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "not_taco.xlsx")

	f := excelize.NewFile()
	_ = f.SetCellValue("Sheet1", "A1", "Título do Documento")
	_ = f.SetCellValue("Sheet1", "A2", "Nenhum dado aqui")
	if err := f.SaveAs(path); err != nil {
		t.Fatalf("save xlsx: %v", err)
	}
	_ = f.Close()

	_, err := New(path)
	if err == nil {
		t.Fatal("expected New to reject a file matching neither known layout")
	}
}

// ---------------------------------------------------------------------------
// FetchBulk
// ---------------------------------------------------------------------------

func TestFetchBulk(t *testing.T) {
	path := filepath.Join("..", "..", "..", "testdata", "taco_sample.csv")
	src, err := New(path)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	const wantRows = 15 // data rows in testdata/taco_sample.csv

	t.Run("no filter emits everything", func(t *testing.T) {
		var got []types.FoodMatch
		err := src.FetchBulk(context.Background(), ports.BulkFilter{}, func(fm types.FoodMatch) error {
			got = append(got, fm)
			return nil
		})
		if err != nil {
			t.Fatalf("FetchBulk: %v", err)
		}
		if len(got) != wantRows {
			t.Errorf("got %d rows, want %d", len(got), wantRows)
		}

		names := make(map[string]bool, len(got))
		for _, fm := range got {
			names[fm.Name] = true
		}
		for _, want := range []string{"Frango grelhado", "Feijão carioca cozido"} {
			if !names[want] {
				t.Errorf("expected %q among emitted foods", want)
			}
		}
	})

	t.Run("MaxRows truncates", func(t *testing.T) {
		n := 0
		err := src.FetchBulk(context.Background(), ports.BulkFilter{MaxRows: 3}, func(fm types.FoodMatch) error {
			n++
			return nil
		})
		if err != nil {
			t.Fatalf("FetchBulk: %v", err)
		}
		if n != 3 {
			t.Errorf("got %d rows, want 3 (MaxRows)", n)
		}
	})

	t.Run("emit error aborts early", func(t *testing.T) {
		wantErr := errors.New("boom")
		n := 0
		err := src.FetchBulk(context.Background(), ports.BulkFilter{}, func(fm types.FoodMatch) error {
			n++
			if n == 2 {
				return wantErr
			}
			return nil
		})
		if !errors.Is(err, wantErr) {
			t.Fatalf("FetchBulk error = %v, want %v", err, wantErr)
		}
		if n != 2 {
			t.Errorf("emitted %d items after error, want exactly 2 (stop on error)", n)
		}
	})
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
	if !reflect.DeepEqual(fm1, fm2) {
		t.Error("two loads should return same data")
	}
}
