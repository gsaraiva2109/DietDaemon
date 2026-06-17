// Package openfoodfacts implements ports.NutritionSource by querying the
// Open Food Facts product search API.
package openfoodfacts

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

// DefaultBaseURL is the standard Open Food Facts search endpoint.
const DefaultBaseURL = "https://world.openfoodfacts.org"

// Source resolves foods by searching the Open Food Facts API.
type Source struct {
	client  *http.Client
	baseURL string
}

// New returns a Source pointed at the public OFF API.
func New() *Source {
	return &Source{
		client:  &http.Client{},
		baseURL: DefaultBaseURL,
	}
}

// Name returns "openfoodfacts".
func (s *Source) Name() string { return "openfoodfacts" }

// ---------------------------------------------------------------------------
// OFF API response shapes
// ---------------------------------------------------------------------------

type searchResponse struct {
	Count    int       `json:"count"`
	Products []product `json:"products"`
}

type product struct {
	Code        string     `json:"code"`
	ProductName string     `json:"product_name"`
	Nutriments  nutriments `json:"nutriments"`
}

type nutriments struct {
	EnergyKcal100g    float64 `json:"energy-kcal_100g"`
	Proteins100g      float64 `json:"proteins_100g"`
	Carbohydrates100g float64 `json:"carbohydrates_100g"`
	Fat100g           float64 `json:"fat_100g"`
	Fiber100g         float64 `json:"fiber_100g"`
}

// Resolve queries the OFF search API with the item's RawPhrase and returns the
// first usable result. Returns types.ErrNoMatch when no products are found or
// none have sufficient nutrition data.
func (s *Source) Resolve(ctx context.Context, item types.ParsedItem) (types.FoodMatch, error) {
	phrase := normalize.Normalize(item.RawPhrase)
	if phrase == "" {
		return types.FoodMatch{}, types.ErrNoMatch
	}

	q := url.Values{}
	q.Set("search_terms", phrase)
	q.Set("search_simple", "1")
	q.Set("json", "1")

	req, err := http.NewRequestWithContext(ctx, http.MethodGet,
		s.baseURL+"/cgi/search.pl?"+q.Encode(), nil)
	if err != nil {
		return types.FoodMatch{}, fmt.Errorf("openfoodfacts: build request: %w", err)
	}
	req.Header.Set("User-Agent", "DietDaemon/0.1 (self-hosted nutrition tracker)")

	resp, err := s.client.Do(req)
	if err != nil {
		return types.FoodMatch{}, fmt.Errorf("openfoodfacts: search: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return types.FoodMatch{}, fmt.Errorf("openfoodfacts: status %d", resp.StatusCode)
	}

	var sr searchResponse
	if err := json.NewDecoder(resp.Body).Decode(&sr); err != nil {
		return types.FoodMatch{}, fmt.Errorf("openfoodfacts: decode: %w", err)
	}

	if sr.Count == 0 || len(sr.Products) == 0 {
		return types.FoodMatch{}, types.ErrNoMatch
	}

	// Pick the first product with a name and non-zero energy.
	for _, p := range sr.Products {
		if p.ProductName == "" || p.Nutriments.EnergyKcal100g == 0 {
			continue
		}
		return types.FoodMatch{
			FoodID:  p.Code,
			Name:    p.ProductName,
			Source:  "openfoodfacts",
			Per100g: p.Nutriments.toMacros(),
		}, nil
	}

	return types.FoodMatch{}, types.ErrNoMatch
}

func (n nutriments) toMacros() types.Macros {
	return types.Macros{
		Calories: n.EnergyKcal100g,
		Protein:  n.Proteins100g,
		Carbs:    n.Carbohydrates100g,
		Fat:      n.Fat100g,
		Fiber:    n.Fiber100g,
	}
}
