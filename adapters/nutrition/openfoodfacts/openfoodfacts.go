// Package openfoodfacts implements ports.NutritionSource by querying the
// Open Food Facts product search API.
package openfoodfacts

import (
	"bufio"
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/gsaraiva2109/dietdaemon/core/ports"
	"github.com/gsaraiva2109/dietdaemon/core/types"
	"github.com/gsaraiva2109/dietdaemon/internal/normalize"
)

// Compile-time interface checks.
var (
	_ ports.NutritionSource = (*Source)(nil)
	_ ports.BulkSource      = (*Source)(nil)
)

// DefaultBaseURL is the standard Open Food Facts search endpoint.
const DefaultBaseURL = "https://world.openfoodfacts.org"

// Source resolves foods by searching the Open Food Facts API.
type Source struct {
	client  *http.Client
	baseURL string

	// bulkFilePath, when set (via NewBulk), makes FetchBulk stream from a local
	// .jsonl.gz export instead of hitting the live v2 search API.
	bulkFilePath string
}

// New returns a Source pointed at the public OFF API.
func New() *Source {
	return &Source{
		client:  &http.Client{},
		baseURL: DefaultBaseURL,
	}
}

// NewBulk returns a Source configured for bulk import. If bulkFilePath is
// non-empty, FetchBulk streams from that local .jsonl.gz file instead of the
// live API; otherwise it paginates the live v2 search endpoint.
func NewBulk(bulkFilePath string) *Source {
	s := New()
	s.bulkFilePath = bulkFilePath
	return s
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
	Code         string     `json:"code"`
	ProductName  string     `json:"product_name"`
	Brands       string     `json:"brands"`
	Categories   string     `json:"categories"`
	ImageURL     string     `json:"image_url"`
	Quantity     string     `json:"quantity"`
	UniqueScansN int        `json:"unique_scans_n"`
	Nutriments   nutriments `json:"nutriments"`
}

// toFoodMatch maps an OFF product to the canonical types.FoodMatch shape.
// Shared by Resolve and both bulk-fetch paths so the field mapping lives in
// one place.
func (p product) toFoodMatch() types.FoodMatch {
	servingSize, servingUnit := parseQuantity(p.Quantity)
	return types.FoodMatch{
		FoodID:      p.Code,
		Name:        p.ProductName,
		Source:      "openfoodfacts",
		Per100g:     p.Nutriments.toMacros(),
		Category:    p.Categories,
		Brand:       p.Brands,
		Barcode:     p.Code,
		ImageURL:    p.ImageURL,
		ServingSize: servingSize,
		ServingUnit: servingUnit,
	}
}

// parseQuantity splits an OFF "quantity" string (e.g. "500 g", "1L") into a
// numeric size and its unit. Returns (0, "") when q has no leading number.
func parseQuantity(q string) (float64, string) {
	i := 0
	for i < len(q) && (unicode.IsDigit(rune(q[i])) || q[i] == '.' || q[i] == ',') {
		i++
	}
	if i == 0 {
		return 0, ""
	}
	numPart := strings.ReplaceAll(q[:i], ",", ".")
	size, err := strconv.ParseFloat(numPart, 64)
	if err != nil {
		return 0, ""
	}
	return size, strings.TrimSpace(q[i:])
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
	defer func() { _ = resp.Body.Close() }()

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
		return p.toFoodMatch(), nil
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

// ---------------------------------------------------------------------------
// Bulk import
// ---------------------------------------------------------------------------

// bulkFields limits the v2 search response to what FetchBulk needs, keeping
// pages small.
const bulkFields = "code,product_name,brands,categories,quantity,image_url,unique_scans_n,nutriments"

// FetchBulk streams OpenFoodFacts products to emit, from the live v2 search
// API or a local .jsonl.gz export depending on how the Source was
// constructed (see NewBulk).
func (s *Source) FetchBulk(ctx context.Context, filter ports.BulkFilter, emit func(types.FoodMatch) error) error {
	if s.bulkFilePath != "" {
		return s.fetchBulkFile(ctx, filter, emit)
	}
	return s.fetchBulkAPI(ctx, filter, emit)
}

// meetsPopularity reports whether p passes filter.MinPopularity. OFF's
// unique_scans_n (times a barcode was scanned) is the closest available
// popularity signal; filtering is done client-side since the v2 search API
// has no documented server-side minimum-popularity parameter.
func meetsPopularity(p product, filter ports.BulkFilter) bool {
	return filter.MinPopularity <= 0 || p.UniqueScansN >= filter.MinPopularity
}

// fetchBulkAPI pages through the OFF v2 search endpoint, most-popular-first,
// emitting matches until exhausted, filter.MaxRows is reached, ctx is
// cancelled, or emit errors.
//
// NOTE: sort_by=unique_scans_n is assumed to sort descending (most-scanned
// first) per OFF's v2 API historical behavior; this is not re-verified
// against current live docs in this session — if OFF changes/ignores this
// param, bulk import still completes correctly (just not in popularity
// order), since popularity filtering below is done client-side regardless.
// bulkPageMaxRetries is how many times a transient failure (5xx, 429, or a
// transport error) on a single bulk page is retried before giving up.
const bulkPageMaxRetries = 3

// bulkRetryBackoff scales the linear backoff between retry attempts. Var (not
// const) so tests can zero it out.
var bulkRetryBackoff = time.Second

func (s *Source) fetchBulkAPI(ctx context.Context, filter ports.BulkFilter, emit func(types.FoodMatch) error) error {
	const pageSize = 100
	emitted := 0

	for page := 1; ; page++ {
		if ctx.Err() != nil {
			return ctx.Err()
		}

		sr, err := s.fetchBulkPage(ctx, page, pageSize)
		if err != nil {
			return err
		}

		if len(sr.Products) == 0 {
			return nil
		}

		for _, p := range sr.Products {
			if !meetsPopularity(p, filter) {
				continue
			}
			if err := emit(p.toFoodMatch()); err != nil {
				return err
			}
			emitted++
			if filter.MaxRows > 0 && emitted >= filter.MaxRows {
				return nil
			}
		}
	}
}

// fetchBulkPage fetches one page of the bulk search, retrying a transient
// failure (429, 5xx, or a transport error) up to bulkPageMaxRetries times
// with a short linear backoff before giving up. A non-transient (4xx other
// than 429) status fails immediately, no retry.
func (s *Source) fetchBulkPage(ctx context.Context, page, pageSize int) (searchResponse, error) {
	q := url.Values{}
	q.Set("sort_by", "unique_scans_n")
	q.Set("page", strconv.Itoa(page))
	q.Set("page_size", strconv.Itoa(pageSize))
	q.Set("fields", bulkFields)
	reqURL := s.baseURL + "/api/v2/search?" + q.Encode()

	var lastErr error
	for attempt := 0; attempt <= bulkPageMaxRetries; attempt++ {
		if attempt > 0 {
			select {
			case <-ctx.Done():
				return searchResponse{}, ctx.Err()
			case <-time.After(time.Duration(attempt) * bulkRetryBackoff):
			}
		}

		req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
		if err != nil {
			return searchResponse{}, fmt.Errorf("openfoodfacts: build bulk request: %w", err)
		}
		req.Header.Set("User-Agent", "DietDaemon/0.1 (self-hosted nutrition tracker)")

		resp, err := s.client.Do(req)
		if err != nil {
			lastErr = fmt.Errorf("openfoodfacts: bulk fetch page %d: %w", page, err)
			continue
		}

		var sr searchResponse
		decErr := json.NewDecoder(resp.Body).Decode(&sr)
		status := resp.StatusCode
		_ = resp.Body.Close()

		if status == http.StatusOK {
			if decErr != nil {
				return searchResponse{}, fmt.Errorf("openfoodfacts: bulk decode page %d: %w", page, decErr)
			}
			return sr, nil
		}

		lastErr = fmt.Errorf("openfoodfacts: bulk status %d on page %d", status, page)
		if status != http.StatusTooManyRequests && status < http.StatusInternalServerError {
			return searchResponse{}, lastErr // non-transient client error, don't retry
		}
	}
	return searchResponse{}, fmt.Errorf("openfoodfacts: page %d failed after %d retries: %w", page, bulkPageMaxRetries, lastErr)
}

// fetchBulkFile streams a local OFF .jsonl.gz export (one product JSON object
// per line) without ever holding the whole file in memory.
func (s *Source) fetchBulkFile(ctx context.Context, filter ports.BulkFilter, emit func(types.FoodMatch) error) error {
	f, err := os.Open(s.bulkFilePath)
	if err != nil {
		return fmt.Errorf("openfoodfacts: open bulk file: %w", err)
	}
	defer func() { _ = f.Close() }()

	gz, err := gzip.NewReader(f)
	if err != nil {
		return fmt.Errorf("openfoodfacts: gzip reader: %w", err)
	}
	defer func() { _ = gz.Close() }()

	scanner := bufio.NewScanner(gz)
	// OFF product JSON lines can exceed bufio.Scanner's default 64KB token
	// size; grow the buffer so long lines don't silently truncate/error.
	scanner.Buffer(make([]byte, 0, 1<<20), 1<<20)

	emitted := 0
	for scanner.Scan() {
		if ctx.Err() != nil {
			return ctx.Err()
		}

		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}

		var p product
		if err := json.Unmarshal(line, &p); err != nil {
			// ponytail: skip malformed lines rather than abort a multi-GB
			// import over one bad row.
			continue
		}
		if !meetsPopularity(p, filter) {
			continue
		}
		if err := emit(p.toFoodMatch()); err != nil {
			return err
		}
		emitted++
		if filter.MaxRows > 0 && emitted >= filter.MaxRows {
			return nil
		}
	}
	return scanner.Err()
}
