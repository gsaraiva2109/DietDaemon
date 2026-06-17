// Package usda implements ports.NutritionSource by querying the USDA FoodData
// Central REST API. It mirrors the openfoodfacts adapter.
package usda

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/gsaraiva2109/dietdaemon/core/ports"
	"github.com/gsaraiva2109/dietdaemon/core/types"
	"github.com/gsaraiva2109/dietdaemon/internal/normalize"
)

// Compile-time interface check.
var _ ports.NutritionSource = (*Source)(nil)

// DefaultBaseURL is the USDA FoodData Central API v1 endpoint.
const DefaultBaseURL = "https://api.nal.usda.gov/fdc/v1"

// Source resolves foods by searching the USDA FDC API.
type Source struct {
	client  *http.Client
	baseURL string
	apiKey  string
}

// New returns a Source pointed at the USDA FDC API.
func New(apiKey string) *Source {
	return &Source{
		client:  &http.Client{},
		baseURL: DefaultBaseURL,
		apiKey:  apiKey,
	}
}

// Name returns "usda".
func (s *Source) Name() string { return "usda" }

// ---------------------------------------------------------------------------
// USDA FDC API response shapes
// ---------------------------------------------------------------------------

type searchResponse struct {
	Foods []food `json:"foods"`
}

type food struct {
	FdcID         int            `json:"fdcId"`
	Description   string         `json:"description"`
	FoodNutrients []foodNutrient `json:"foodNutrients"`
}

type foodNutrient struct {
	NutrientID int     `json:"nutrientId"`
	Name       string  `json:"nutrientName"`
	Amount     float64 `json:"amount"`
}

// USDA nutrient IDs we care about.
const (
	nutrientEnergy  = 1008 // Energy (kcal)
	nutrientProtein = 1003 // Protein
	nutrientCarbs   = 1005 // Carbohydrate, by difference
	nutrientFat     = 1004 // Total lipid (fat)
	nutrientFiber   = 1079 // Fiber, total dietary
)

// Resolve queries the USDA FDC search API with the item's RawPhrase and returns
// the first usable result. Returns types.ErrNoMatch when no foods are found.
func (s *Source) Resolve(ctx context.Context, item types.ParsedItem) (types.FoodMatch, error) {
	phrase := normalize.Normalize(item.RawPhrase)
	if phrase == "" {
		return types.FoodMatch{}, types.ErrNoMatch
	}

	q := url.Values{}
	q.Set("api_key", s.apiKey)
	q.Set("query", phrase)
	q.Set("dataType", "Foundation,SR Legacy,Survey (FNDDS)")
	q.Set("pageSize", "5")

	req, err := http.NewRequestWithContext(ctx, http.MethodGet,
		s.baseURL+"/foods/search?"+q.Encode(), nil)
	if err != nil {
		return types.FoodMatch{}, fmt.Errorf("usda: build request: %w", err)
	}

	resp, err := s.client.Do(req)
	if err != nil {
		return types.FoodMatch{}, fmt.Errorf("usda: search: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return types.FoodMatch{}, fmt.Errorf("usda: status %d", resp.StatusCode)
	}

	var sr searchResponse
	if err := json.NewDecoder(resp.Body).Decode(&sr); err != nil {
		return types.FoodMatch{}, fmt.Errorf("usda: decode: %w", err)
	}

	if len(sr.Foods) == 0 {
		return types.FoodMatch{}, types.ErrNoMatch
	}

	// Pick the first food with a usable description and non-zero energy.
	for _, f := range sr.Foods {
		if f.Description == "" {
			continue
		}
		macros := extractMacros(f.FoodNutrients)
		if macros.Calories == 0 {
			continue
		}
		return types.FoodMatch{
			FoodID:  fmt.Sprintf("%d", f.FdcID),
			Name:    f.Description,
			Source:  "usda",
			Per100g: macros,
		}, nil
	}

	return types.FoodMatch{}, types.ErrNoMatch
}

// extractMacros pulls per-100g macros from the USDA nutrient list. USDA returns
// per-100g values by default for Foundation and SR Legacy foods.
func extractMacros(nutrients []foodNutrient) types.Macros {
	var m types.Macros
	for _, n := range nutrients {
		switch n.NutrientID {
		case nutrientEnergy:
			m.Calories = n.Amount
		case nutrientProtein:
			m.Protein = n.Amount
		case nutrientCarbs:
			m.Carbs = n.Amount
		case nutrientFat:
			m.Fat = n.Amount
		case nutrientFiber:
			m.Fiber = n.Amount
		}
	}
	return m
}
