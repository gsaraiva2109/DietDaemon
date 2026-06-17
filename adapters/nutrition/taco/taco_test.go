package taco

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/gsaraiva2109/dietdaemon/core/types"
)

func TestLoadAndResolve(t *testing.T) {
	// Load the synthetic fixture from testdata/.
	path := filepath.Join("..", "..", "..", "testdata", "taco_sample.csv")
	src, err := New(path)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if src.Name() != "taco" {
		t.Errorf("Name() = %q, want taco", src.Name())
	}

	ctx := context.Background()

	// Exact match.
	fm, err := src.Resolve(ctx, types.ParsedItem{RawPhrase: "Frango grelhado"})
	if err != nil {
		t.Fatalf("Resolve Frango grelhado: %v", err)
	}
	if fm.FoodID != "TACO005" || fm.Name != "Frango grelhado" || fm.Source != "taco" {
		t.Errorf("got %+v", fm)
	}
	if fm.Per100g.Calories != 165 || fm.Per100g.Protein != 31 {
		t.Errorf("macros: calories=%f protein=%f", fm.Per100g.Calories, fm.Per100g.Protein)
	}

	// Accented variant.
	fm, err = src.Resolve(ctx, types.ParsedItem{RawPhrase: "Feijão carioca cozido"})
	if err != nil {
		t.Fatalf("Resolve Feijão: %v", err)
	}
	// "Feijão" normalizes to "feijao"; the CSV has "Feijão carioca cozido"
	// which normalizes to "feijao carioca cozido". Search key is "feijao carioca cozido" — match.
	if fm.FoodID != "TACO003" {
		t.Errorf("expected TACO003, got %s", fm.FoodID)
	}

	// Normalize-insensitive: search with "feijao" (already unaccented) should match "Feijão".
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

func TestNewMissingFile(t *testing.T) {
	_, err := New("/nonexistent/taco.csv")
	if err == nil {
		t.Error("expected error for missing file")
	}
}

func TestLoadMultipleFiles(t *testing.T) {
	// Verify second load works (map is fresh each time).
	path := filepath.Join("..", "..", "..", "testdata", "taco_sample.csv")
	s1, _ := New(path)
	s2, _ := New(path)

	ctx := context.Background()
	fm1, _ := s1.Resolve(ctx, types.ParsedItem{RawPhrase: "arroz branco cozido"})
	fm2, _ := s2.Resolve(ctx, types.ParsedItem{RawPhrase: "arroz branco cozido"})
	if fm1 != fm2 {
		t.Error("two loads should return same data")
	}
}

func TestTinySyntheticCSV(t *testing.T) {
	// Generate a minimal synthetic CSV in a temp location and load it.
	dir := t.TempDir()
	path := filepath.Join(dir, "mini.csv")
	data := "food_id,name,kcal,protein,carb,fat,fiber\nX001,Test Food,100,10,5,2,1\n"
	if err := os.WriteFile(path, []byte(data), 0644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}

	src, err := New(path)
	if err != nil {
		t.Fatalf("New from synthetic: %v", err)
	}

	fm, err := src.Resolve(context.Background(), types.ParsedItem{RawPhrase: "Test Food"})
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if fm.FoodID != "X001" || fm.Per100g.Calories != 100 {
		t.Errorf("unexpected result: %+v", fm)
	}
}
