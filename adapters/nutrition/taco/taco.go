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
// Two column layouts are recognized: the simplified schema used by the
// embedded dataset (food_id, name, kcal, protein, carb, fat, fiber), and the
// raw official TACO/NEPA spreadsheet an operator may point TACO_DATA_PATH at
// directly. Anything matching neither is a loud error instead of a silent
// misparse (see issue #111).
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

	foods, err := parseRows(rows)
	if err != nil {
		return nil, err
	}
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

// parseRows dispatches between the two column layouts New() accepts. Without
// this, a file that doesn't match the simplified schema (e.g. the raw
// official TACO/NEPA spreadsheet, which has moisture% and kJ columns before
// protein, category-separator rows, and a three-row merged header) got no
// error at all: rowsToFoods just read whichever columns happened to line
// up and wrote silently wrong macros for every row (issue #111).
func parseRows(rows [][]string) (map[string]types.FoodMatch, error) {
	headerErr := checkSimpleHeader(rows)
	if headerErr == nil {
		return rowsToFoods(rows), nil
	}
	if foods := officialRowsToFoods(rows); len(foods) > 0 {
		return foods, nil
	}
	return nil, headerErr
}

// checkSimpleHeader reports whether rows starts with the simplified schema's
// header (food_id, name, ...), used by both the embedded dataset and any
// operator-supplied file meant to match it.
func checkSimpleHeader(rows [][]string) error {
	if len(rows) == 0 || len(rows[0]) < 2 {
		return fmt.Errorf("taco: file has no header row")
	}
	got0 := strings.ToLower(strings.TrimSpace(rows[0][0]))
	got1 := strings.ToLower(strings.TrimSpace(rows[0][1]))
	if got0 != "food_id" || got1 != "name" {
		return fmt.Errorf(
			"taco: unexpected header %v, want columns food_id,name,kcal,protein,carb,fat,fiber "+
				"(simplified schema, see adapters/nutrition/taco/taco.csv) or the official TACO/NEPA "+
				"spreadsheet layout — neither matched",
			rows[0],
		)
	}
	return nil
}

// officialRowsToFoods parses the raw official TACO/NEPA spreadsheet layout:
// a three-row merged header, food-group separator rows (only column 0
// populated, e.g. "Cereais e derivados"), and columns id, name, moisture%,
// kcal, kJ, protein, fat, cholesterol, carbs, fiber, ... — a completely
// different order and width than the simplified schema. Food IDs are
// prefixed with "TACO" to match the embedded dataset's ID scheme, since the
// official file's "Número do Alimento" column is a bare integer.
func officialRowsToFoods(rows [][]string) map[string]types.FoodMatch {
	const (
		colID      = 0
		colName    = 1
		colKcal    = 3
		colProtein = 5
		colFat     = 6
		colCarbs   = 8
		colFiber   = 9
	)
	foods := make(map[string]types.FoodMatch)
	for _, row := range rows {
		if len(row) <= colFiber {
			continue
		}
		id := strings.TrimSpace(row[colID])
		if _, err := strconv.Atoi(id); err != nil {
			continue // header row or food-group separator, not a food row
		}
		name := strings.TrimSpace(row[colName])
		if name == "" {
			continue
		}
		fm := types.FoodMatch{
			FoodID:     "TACO" + id,
			Name:       name,
			Source:     "taco",
			MatchScore: 1.0,
		}
		fm.Per100g.Calories = parseFloat(row[colKcal])
		fm.Per100g.Protein = parseFloat(row[colProtein])
		fm.Per100g.Carbs = parseFloat(row[colCarbs])
		fm.Per100g.Fat = parseFloat(row[colFat])
		fm.Per100g.Fiber = parseFloat(row[colFiber])

		key := normalizePhrase(fm.Name)
		if key != "" {
			foods[key] = fm
		}
	}
	return foods
}

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
