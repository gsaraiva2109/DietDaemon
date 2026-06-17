// Package taco implements ports.NutritionSource backed by the TACO (Tabela
// Brasileira de Composição de Alimentos) dataset. A local CSV file is loaded
// into memory and foods are resolved by exact normalized-name match.
package taco

import (
	"context"
	"encoding/csv"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/gsaraiva2109/dietdaemon/core/ports"
	"github.com/gsaraiva2109/dietdaemon/core/types"
)

// Compile-time interface check.
var _ ports.NutritionSource = (*Source)(nil)

// Source resolves foods from an in-memory TACO dataset.
type Source struct {
	foods map[string]types.FoodMatch // normalized name → match
}

// New loads the CSV at dataPath and builds the in-memory index. Expected
// columns: food_id, name, kcal, protein, carb, fat, fiber.
func New(dataPath string) (*Source, error) {
	f, err := os.Open(dataPath)
	if err != nil {
		return nil, fmt.Errorf("taco: open %s: %w", dataPath, err)
	}
	defer f.Close()

	r := csv.NewReader(f)
	r.TrimLeadingSpace = true

	records, err := r.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("taco: read csv: %w", err)
	}
	if len(records) < 2 {
		return nil, fmt.Errorf("taco: csv has no data rows")
	}

	// Skip header row.
	foods := make(map[string]types.FoodMatch, len(records)-1)
	for _, row := range records[1:] {
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

	if len(foods) == 0 {
		return nil, fmt.Errorf("taco: no foods loaded from %s", dataPath)
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
