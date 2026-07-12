// Package taco implements ports.NutritionSource backed by the TACO (Tabela
// Brasileira de Composição de Alimentos) dataset. Supports both CSV and XLSX
// files (chosen by extension). Data is loaded into memory at construction time
// and foods are resolved by exact normalized-name match.
//
// The default taco.csv is embedded into the binary via go:embed so the dataset
// is available with zero configuration. An optional TACO_DATA_PATH overrides
// the embedded data with an external file.
package taco

import (
	"bytes"
	"context"
	_ "embed"
	"encoding/csv"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/xuri/excelize/v2"

	"github.com/gsaraiva2109/dietdaemon/core/ports"
	"github.com/gsaraiva2109/dietdaemon/core/types"
)

//go:embed taco.csv
var defaultTacoCSV []byte

// Compile-time interface checks.
var (
	_ ports.NutritionSource = (*Source)(nil)
	_ ports.BulkSource      = (*Source)(nil)
)

// Source resolves foods from an in-memory TACO dataset.
type Source struct {
	foods map[string]types.FoodMatch // normalized name → match
}

// New loads the dataset and builds the in-memory index. When dataPath is empty
// the embedded taco.csv is used; otherwise the file at dataPath is loaded.
// Format is chosen by file extension: .csv → encoding/csv, .xlsx → excelize.
// Expected columns (in order): food_id, name, kcal, protein, carb, fat, fiber.
func New(dataPath string) (*Source, error) {
	var rows [][]string
	var err error

	if dataPath == "" {
		rows, err = loadEmbeddedCSV()
	} else {
		switch strings.ToLower(filepath.Ext(dataPath)) {
		case ".csv":
			rows, err = loadCSV(dataPath)
		case ".xlsx":
			rows, err = loadXLSX(dataPath)
		default:
			return nil, fmt.Errorf("taco: unsupported format %q (want .csv or .xlsx)", filepath.Ext(dataPath))
		}
	}
	if err != nil {
		return nil, err
	}

	foods := rowsToFoods(rows)
	if len(foods) == 0 {
		src := dataPath
		if src == "" {
			src = "embedded taco.csv"
		}
		return nil, fmt.Errorf("taco: no foods loaded from %s", src)
	}

	return &Source{foods: foods}, nil
}

// Name returns "taco".
func (s *Source) Name() string { return "taco" }

// Resolve matches the parsed item's RawPhrase (case/accent-insensitive)
// against the loaded TACO foods. Returns types.ErrNoMatch on miss.
func (s *Source) Resolve(ctx context.Context, item types.ParsedItem) (types.FoodMatch, error) {
	key := normalizePhrase(item.RawPhrase)
	if key == "" {
		return types.FoodMatch{}, types.ErrNoMatch
	}

	fm, ok := s.foods[key]
	if !ok {
		return types.FoodMatch{}, types.ErrNoMatch
	}
	return fm, nil
}

// FetchBulk emits every loaded TACO food. filter.DataTypes and
// filter.MinPopularity are ignored — TACO has no dataType/popularity concept;
// it's a small, already-curated common-foods list imported whole by design.
func (s *Source) FetchBulk(ctx context.Context, filter ports.BulkFilter, emit func(types.FoodMatch) error) error {
	n := 0
	for _, fm := range s.foods {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if filter.MaxRows > 0 && n >= filter.MaxRows {
			break
		}
		if err := emit(fm); err != nil {
			return err
		}
		n++
	}
	return nil
}

// ---------------------------------------------------------------------------
// Loaders
// ---------------------------------------------------------------------------

func loadEmbeddedCSV() ([][]string, error) {
	r := csv.NewReader(bytes.NewReader(defaultTacoCSV))
	r.TrimLeadingSpace = true
	records, err := r.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("taco: read embedded csv: %w", err)
	}
	if len(records) < 2 {
		return nil, fmt.Errorf("taco: embedded csv has no data rows")
	}
	return records, nil
}

func loadCSV(path string) ([][]string, error) {
	// #nosec G304 -- path provided by operator at CLI, intentional file read
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("taco: open csv: %w", err)
	}
	defer func() { _ = f.Close() }()

	r := csv.NewReader(f)
	r.TrimLeadingSpace = true
	records, err := r.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("taco: read csv: %w", err)
	}
	if len(records) < 2 {
		return nil, fmt.Errorf("taco: csv has no data rows")
	}
	return records, nil
}

func loadXLSX(path string) ([][]string, error) {
	f, err := excelize.OpenFile(path)
	if err != nil {
		return nil, fmt.Errorf("taco: open xlsx: %w", err)
	}
	defer func() { _ = f.Close() }()

	// Read the first sheet.
	sheet := f.GetSheetName(0)
	raw, err := f.GetRows(sheet)
	if err != nil {
		return nil, fmt.Errorf("taco: read xlsx rows: %w", err)
	}
	if len(raw) < 2 {
		return nil, fmt.Errorf("taco: xlsx has no data rows")
	}

	// Convert [][]string from excelize (already strings) — no-op cast through
	// a copy so we return a uniform [][]string.
	rows := make([][]string, len(raw))
	copy(rows, raw)
	return rows, nil
}

// ---------------------------------------------------------------------------
// Row → FoodMatch
// ---------------------------------------------------------------------------

// rowsToFoods converts raw string rows (first row is header) into a normalized
// name → FoodMatch map.
func rowsToFoods(rows [][]string) map[string]types.FoodMatch {
	foods := make(map[string]types.FoodMatch, len(rows)-1)
	for _, row := range rows[1:] {
		if len(row) < 7 {
			continue
		}
		fm := types.FoodMatch{
			FoodID:     strings.TrimSpace(row[0]),
			Name:       strings.TrimSpace(row[1]),
			Source:     "taco",
			MatchScore: 1.0,
		}
		fm.Per100g.Calories = parseFloat(row[2])
		fm.Per100g.Protein = parseFloat(row[3])
		fm.Per100g.Carbs = parseFloat(row[4])
		fm.Per100g.Fat = parseFloat(row[5])
		fm.Per100g.Fiber = parseFloat(row[6])

		key := normalizePhrase(fm.Name)
		if key != "" {
			foods[key] = fm
		}
	}
	return foods
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func parseFloat(s string) float64 {
	v, _ := strconv.ParseFloat(strings.TrimSpace(s), 64)
	return v
}

func normalizePhrase(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	return unaccent(s)
}

func unaccent(s string) string {
	r := strings.NewReplacer(
		"à", "a", "á", "a", "â", "a", "ã", "a", "ä", "a", "å", "a",
		"æ", "ae", "ç", "c",
		"è", "e", "é", "e", "ê", "e", "ë", "e",
		"ì", "i", "í", "i", "î", "i", "ï", "i",
		"ð", "d", "ñ", "n",
		"ò", "o", "ó", "o", "ô", "o", "õ", "o", "ö", "o", "ø", "o",
		"ù", "u", "ú", "u", "û", "u", "ü", "u",
		"ý", "y", "ÿ", "y",
	)
	return r.Replace(s)
}
