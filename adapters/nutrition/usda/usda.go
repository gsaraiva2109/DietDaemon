// Package usda implements ports.NutritionSource by querying the USDA FoodData
// Central REST API. It mirrors the openfoodfacts adapter.
package usda

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/gsaraiva2109/dietdaemon/core/ports"
	"github.com/gsaraiva2109/dietdaemon/core/types"
	"github.com/gsaraiva2109/dietdaemon/internal/normalize"
)

// Compile-time interface checks.
var (
	_ ports.NutritionSource = (*Source)(nil)
	_ ports.BulkSource      = (*Source)(nil)
)

// DefaultBaseURL is the USDA FoodData Central API v1 endpoint.
const DefaultBaseURL = "https://api.nal.usda.gov/fdc/v1"

// defaultBulkDataTypes is the dataType allowlist used when a BulkFilter
// doesn't specify one. Excludes "Branded" — the much larger, noisier
// UPC-scanned dataset — by product decision.
var defaultBulkDataTypes = []string{"Foundation", "SR Legacy"}

// bulkPageDelay throttles fetchBulkAPI's page requests to stay under USDA's
// rate cap. Var (not const) so tests can zero it out.
var bulkPageDelay = time.Second

// bulkAPIPageSize is USDA's max pageSize for /foods/search.
const bulkAPIPageSize = 200

// Source resolves foods by searching the USDA FDC API.
type Source struct {
	client       *http.Client
	baseURL      string
	apiKey       string
	bulkFilePath string // non-empty: FetchBulk streams from this local file instead of the live API
}

// New returns a Source pointed at the USDA FDC API.
func New(apiKey string) *Source {
	return &Source{
		client:  &http.Client{},
		baseURL: DefaultBaseURL,
		apiKey:  apiKey,
	}
}

// NewBulk returns a Source configured for bulk import. If bulkFilePath is
// non-empty, FetchBulk streams from that local file instead of the live API.
func NewBulk(apiKey, bulkFilePath string) *Source {
	s := New(apiKey) // reuse existing constructor for shared setup (http client, api key, etc)
	s.bulkFilePath = bulkFilePath
	return s
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
	FdcID               int            `json:"fdcId"`
	Description         string         `json:"description"`
	DataType            string         `json:"dataType"`
	FoodCategory        foodCategory   `json:"foodCategory"`        // Foundation/SR Legacy/Survey: an object; see foodCategory.UnmarshalJSON
	BrandedFoodCategory string         `json:"brandedFoodCategory"` // Branded dataType only: a plain string
	ServingSize         float64        `json:"servingSize"`
	ServingSizeUnit     string         `json:"servingSizeUnit"`
	BrandOwner          string         `json:"brandOwner"` // Branded dataType only; empty for Foundation/SR Legacy
	GtinUpc             string         `json:"gtinUpc"`    // Branded dataType only (barcode); empty for Foundation/SR Legacy
	FoodNutrients       []foodNutrient `json:"foodNutrients"`
	FoodPortions        []foodPortion  `json:"foodPortions"` // household-measure data; only present with format=full
}

// foodPortion is one household-measure entry from USDA's foodPortions array,
// e.g. {"amount":1,"modifier":"large","portionDescription":"1 large","gramWeight":50}.
type foodPortion struct {
	Amount             float64 `json:"amount"`
	Modifier           string  `json:"modifier"`
	PortionDescription string  `json:"portionDescription"`
	GramWeight         float64 `json:"gramWeight"`
}

// foodCategory absorbs USDA's inconsistent foodCategory shape across
// dataTypes: Foundation/SR Legacy/Survey return an object (e.g.
// {"id":53,"code":"0100","description":"Dairy and Egg Products"}), while some
// other endpoints return a plain string. Either shape decodes into the
// category's display name.
type foodCategory string

func (c *foodCategory) UnmarshalJSON(b []byte) error {
	var s string
	if err := json.Unmarshal(b, &s); err == nil {
		*c = foodCategory(s)
		return nil
	}
	var obj struct {
		Description string `json:"description"`
	}
	if err := json.Unmarshal(b, &obj); err != nil {
		return fmt.Errorf("foodCategory: unsupported shape: %w", err)
	}
	*c = foodCategory(obj.Description)
	return nil
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
	q.Set("format", "full") // include foodPortions (household-measure data, #134)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet,
		s.baseURL+"/foods/search?"+q.Encode(), nil)
	if err != nil {
		return types.FoodMatch{}, fmt.Errorf("usda: build request: %w", err)
	}

	resp, err := s.client.Do(req)
	if err != nil {
		return types.FoodMatch{}, fmt.Errorf("usda: search: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

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
		if fm, ok := foodToMatch(f); ok {
			return fm, nil
		}
	}

	return types.FoodMatch{}, types.ErrNoMatch
}

// foodToMatch maps a USDA food search result to a types.FoodMatch. ok is false
// when the food lacks a usable description or has no energy value — the
// caller should skip it. Shared by Resolve, fetchBulkAPI, and fetchBulkFile so
// the response-to-FoodMatch mapping lives in exactly one place.
func foodToMatch(f food) (types.FoodMatch, bool) {
	if f.Description == "" {
		return types.FoodMatch{}, false
	}
	macros := extractMacros(f.FoodNutrients)
	if macros.Calories == 0 {
		return types.FoodMatch{}, false
	}
	category := string(f.FoodCategory)
	if f.BrandedFoodCategory != "" {
		category = f.BrandedFoodCategory // Branded dataType uses this field instead
	}
	return types.FoodMatch{
		FoodID:       fmt.Sprintf("%d", f.FdcID),
		Name:         f.Description,
		Source:       "usda",
		Per100g:      macros,
		Category:     category,
		Brand:        f.BrandOwner,
		Barcode:      f.GtinUpc,
		ServingSize:  f.ServingSize,
		ServingUnit:  f.ServingSizeUnit,
		ServingUnits: portionsToServingUnits(f.FoodPortions),
	}, true
}

// portionsToServingUnits converts USDA's foodPortions household-measure
// entries (e.g. {"amount":1,"modifier":"large","gramWeight":50}) into
// system-provided serving units. Entries with no usable gram weight are
// skipped; PortionDescription is preferred as the label when present since
// it's already human-readable ("1 large"), falling back to "Amount Modifier".
func portionsToServingUnits(portions []foodPortion) []types.FoodServingUnit {
	var out []types.FoodServingUnit
	for _, p := range portions {
		if p.GramWeight <= 0 {
			continue
		}
		label := strings.TrimSpace(p.PortionDescription)
		if label == "" || label == "undetermined" {
			label = strings.TrimSpace(fmt.Sprintf("%s %s", strconv.FormatFloat(p.Amount, 'g', -1, 64), p.Modifier))
		}
		if label == "" {
			continue
		}
		out = append(out, types.FoodServingUnit{Label: label, Grams: p.GramWeight})
	}
	return out
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

// ---------------------------------------------------------------------------
// Bulk import
// ---------------------------------------------------------------------------

// FetchBulk implements ports.BulkSource, dispatching to the live API or a
// local bulk-export file depending on how the Source was constructed.
func (s *Source) FetchBulk(ctx context.Context, filter ports.BulkFilter, emit func(types.FoodMatch) error) error {
	if s.bulkFilePath != "" {
		return s.fetchBulkFile(ctx, filter, emit)
	}
	return s.fetchBulkAPI(ctx, filter, emit)
}

// bulkDataTypes returns filter.DataTypes, defaulting to defaultBulkDataTypes
// when the filter doesn't specify one.
func bulkDataTypes(filter []string) []string {
	if len(filter) == 0 {
		return defaultBulkDataTypes
	}
	return filter
}

// fetchBulkAPI pages through USDA's /foods/search with no query — a bulk
// browse of every food in the allowed dataType(s) — instead of Resolve's
// name search. Stops on an empty page, ctx cancellation, filter.MaxRows, or
// an emit error.
func (s *Source) fetchBulkAPI(ctx context.Context, filter ports.BulkFilter, emit func(types.FoodMatch) error) error {
	dataTypes := strings.Join(bulkDataTypes(filter.DataTypes), ",")

	n := 0
	for page := 1; ; page++ {
		if ctx.Err() != nil {
			return ctx.Err()
		}

		q := url.Values{}
		q.Set("api_key", s.apiKey)
		q.Set("dataType", dataTypes)
		q.Set("pageSize", strconv.Itoa(bulkAPIPageSize))
		q.Set("pageNumber", strconv.Itoa(page))
		q.Set("format", "full") // include foodPortions (household-measure data, #134)

		req, err := http.NewRequestWithContext(ctx, http.MethodGet,
			s.baseURL+"/foods/search?"+q.Encode(), nil)
		if err != nil {
			return fmt.Errorf("usda: build bulk request: %w", err)
		}

		resp, err := s.client.Do(req)
		if err != nil {
			return fmt.Errorf("usda: bulk search: %w", err)
		}
		var sr searchResponse
		decErr := json.NewDecoder(resp.Body).Decode(&sr)
		status := resp.StatusCode
		_ = resp.Body.Close()
		if status != http.StatusOK {
			return fmt.Errorf("usda: bulk status %d", status)
		}
		if decErr != nil {
			return fmt.Errorf("usda: bulk decode: %w", decErr)
		}

		if len(sr.Foods) == 0 {
			return nil
		}

		for _, f := range sr.Foods {
			if filter.MaxRows > 0 && n >= filter.MaxRows {
				return nil
			}
			fm, ok := foodToMatch(f)
			if !ok {
				continue
			}
			if err := emit(fm); err != nil {
				return err
			}
			n++
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(bulkPageDelay):
		}
	}
}

// fetchBulkFile stream-decodes USDA food objects from s.bulkFilePath one
// element at a time, so a multi-GB bulk export never has to fit in memory.
// Elements whose dataType isn't in the filter's allowlist are skipped.
//
// USDA's FoodData Central full downloads aren't a bare top-level array — they
// wrap it in an object under a key like "FoundationFoods"/"SRLegacyFoods"
// (e.g. {"FoundationFoods": [...]}). Rather than special-case the wrapper key
// name, walk tokens until the first '[' delimiter and stream that array's
// elements — this handles a wrapped export and a bare top-level array alike.
func (s *Source) fetchBulkFile(ctx context.Context, filter ports.BulkFilter, emit func(types.FoodMatch) error) error {
	// #nosec G304 -- path is host-configured at startup, not user input
	f, err := os.Open(s.bulkFilePath)
	if err != nil {
		return fmt.Errorf("usda: open bulk file: %w", err)
	}
	defer func() { _ = f.Close() }()

	allow := make(map[string]bool)
	for _, dt := range bulkDataTypes(filter.DataTypes) {
		allow[dt] = true
	}

	dec := json.NewDecoder(f)
	for {
		tok, err := dec.Token()
		if err != nil {
			return fmt.Errorf("usda: bulk file: %w", err)
		}
		if tok == json.Delim('[') {
			break
		}
	}

	n := 0
	for dec.More() {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if filter.MaxRows > 0 && n >= filter.MaxRows {
			return nil
		}

		var item food
		if err := dec.Decode(&item); err != nil {
			return fmt.Errorf("usda: bulk file decode: %w", err)
		}
		if !allow[item.DataType] {
			continue
		}
		fm, ok := foodToMatch(item)
		if !ok {
			continue
		}
		if err := emit(fm); err != nil {
			return err
		}
		n++
	}
	return nil
}
