package openfoodfacts

import (
	"compress/gzip"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/gsaraiva2109/dietdaemon/core/ports"
	"github.com/gsaraiva2109/dietdaemon/core/types"
)

// setJSONContentType sets the JSON response header shared by every fake OFF
// server in this file.
func setJSONContentType(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
}

func TestResolve(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Validate query parameters.
		q := r.URL.Query()
		if q.Get("search_terms") != "arroz integral" {
			t.Errorf("search_terms = %q", q.Get("search_terms"))
		}
		if q.Get("json") != "1" {
			t.Error("json param missing")
		}

		setJSONContentType(w)
		_, _ = w.Write([]byte(`{
			"count": 1,
			"page": 1,
			"page_size": 24,
			"products": [
				{
					"code": "7891234567890",
					"product_name": "Arroz Integral Cozido",
					"nutriments": {
						"energy-kcal_100g": 124,
						"proteins_100g": 2.6,
						"carbohydrates_100g": 25.8,
						"fat_100g": 1.0,
						"fiber_100g": 2.7
					}
				}
			]
		}`))
	}))
	defer srv.Close()

	s := New()
	s.baseURL = srv.URL

	fm, err := s.Resolve(context.Background(), types.ParsedItem{RawPhrase: "arroz integral"})
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if fm.FoodID != "7891234567890" {
		t.Errorf("FoodID = %q", fm.FoodID)
	}
	if fm.Name != "Arroz Integral Cozido" {
		t.Errorf("Name = %q", fm.Name)
	}
	if fm.Source != "openfoodfacts" {
		t.Errorf("Source = %q", fm.Source)
	}
	if fm.Per100g.Calories != 124 {
		t.Errorf("Calories = %f", fm.Per100g.Calories)
	}
	if fm.Per100g.Protein != 2.6 {
		t.Errorf("Protein = %f", fm.Per100g.Protein)
	}
	if fm.Per100g.Carbs != 25.8 {
		t.Errorf("Carbs = %f", fm.Per100g.Carbs)
	}
	if fm.Per100g.Fat != 1.0 {
		t.Errorf("Fat = %f", fm.Per100g.Fat)
	}
	if fm.Per100g.Fiber != 2.7 {
		t.Errorf("Fiber = %f", fm.Per100g.Fiber)
	}
}

func TestResolveNoResults(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		setJSONContentType(w)
		_, _ = w.Write([]byte(`{"count":0,"products":[]}`))
	}))
	defer srv.Close()

	s := New()
	s.baseURL = srv.URL

	_, err := s.Resolve(context.Background(), types.ParsedItem{RawPhrase: "nonexistent"})
	if !errors.Is(err, types.ErrNoMatch) {
		t.Errorf("expected ErrNoMatch, got %v", err)
	}
}

func TestResolveProductWithoutEnergy(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		setJSONContentType(w)
		_, _ = w.Write([]byte(`{
			"count": 1,
			"products": [
				{
					"code": "x",
					"product_name": "Empty Food",
					"nutriments": {
						"energy-kcal_100g": 0,
						"proteins_100g": 0,
						"carbohydrates_100g": 0,
						"fat_100g": 0,
						"fiber_100g": 0
					}
				}
			]
		}`))
	}))
	defer srv.Close()

	s := New()
	s.baseURL = srv.URL

	// Product has zero energy → skipped, ErrNoMatch.
	_, err := s.Resolve(context.Background(), types.ParsedItem{RawPhrase: "empty"})
	if !errors.Is(err, types.ErrNoMatch) {
		t.Errorf("expected ErrNoMatch for zero-energy product, got %v", err)
	}
}

func TestName(t *testing.T) {
	s := New()
	if s.Name() != "openfoodfacts" {
		t.Errorf("Name() = %q", s.Name())
	}
}

func TestEmptyPhrase(t *testing.T) {
	s := New()
	_, err := s.Resolve(context.Background(), types.ParsedItem{RawPhrase: ""})
	if !errors.Is(err, types.ErrNoMatch) {
		t.Errorf("expected ErrNoMatch for empty, got %v", err)
	}
}

func TestHTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	s := New()
	s.baseURL = srv.URL

	_, err := s.Resolve(context.Background(), types.ParsedItem{RawPhrase: "test"})
	if err == nil {
		t.Error("expected error on 500")
	}
}

// productAName is the fixture name reused across bulk-test cases.
const productAName = "Product A"

// bulkPage is one fake page of OFF v2-search-shaped JSON for bulk tests.
type bulkPage struct {
	Code   string
	Name   string
	ScansN int
}

// bulkProductJSON renders a fake OFF product as compact, single-line JSON —
// required so it also works as one line of a .jsonl.gz export.
func bulkProductJSON(p bulkPage) string {
	b, _ := json.Marshal(map[string]any{
		"code":           p.Code,
		"product_name":   p.Name,
		"brands":         "TestBrand",
		"categories":     "Test Category",
		"quantity":       "500 g",
		"image_url":      "https://example.com/img.jpg",
		"unique_scans_n": p.ScansN,
		"nutriments": map[string]any{
			"energy-kcal_100g":   100,
			"proteins_100g":      5,
			"carbohydrates_100g": 10,
			"fat_100g":           2,
			"fiber_100g":         1,
		},
	})
	return string(b)
}

func bulkPageJSON(products []bulkPage) string {
	var items strings.Builder
	for i, p := range products {
		if i > 0 {
			items.WriteString(",")
		}
		items.WriteString(bulkProductJSON(p))
	}
	return fmt.Sprintf(`{"count": %d, "products": [%s]}`, len(products), items.String())
}

// newBulkPagesServer serves OFF v2-search-shaped JSON pages from pages,
// returning an empty page once the requested page is out of range.
func newBulkPagesServer(pages [][]bulkPage) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		page, _ := strconv.Atoi(r.URL.Query().Get("page"))
		setJSONContentType(w)
		if page < 1 || page > len(pages) {
			_, _ = w.Write([]byte(`{"count":0,"products":[]}`))
			return
		}
		_, _ = w.Write([]byte(bulkPageJSON(pages[page-1])))
	}))
}

// assertFirstBulkProductMapped checks the fields toFoodMatch is expected to
// fill in for fixture product "A" shared by this file's bulk tests.
func assertFirstBulkProductMapped(t *testing.T, fm types.FoodMatch) {
	t.Helper()
	if fm.FoodID != "A" || fm.Brand != "TestBrand" || fm.Category != "Test Category" {
		t.Errorf("first product mapped wrong: %+v", fm)
	}
	if fm.ServingSize != 500 || fm.ServingUnit != "g" {
		t.Errorf("serving = %v %v, want 500 g", fm.ServingSize, fm.ServingUnit)
	}
}

// assertOnlyFoodIDs fails the test if got contains any FoodID not in want.
func assertOnlyFoodIDs(t *testing.T, got []types.FoodMatch, want ...string) {
	t.Helper()
	for _, fm := range got {
		if !slices.Contains(want, fm.FoodID) {
			t.Errorf("unexpected product emitted: %q, want one of %v", fm.FoodID, want)
		}
	}
}

func TestFetchBulkAPI(t *testing.T) {
	pages := [][]bulkPage{
		{{Code: "A", Name: productAName, ScansN: 100}, {Code: "B", Name: "Product B", ScansN: 5}},
		{{Code: "C", Name: "Product C", ScansN: 50}, {Code: "D", Name: "Product D", ScansN: 1}},
		{{Code: "E", Name: "Product E", ScansN: 200}, {Code: "F", Name: "Product F", ScansN: 0}},
	}

	srv := newBulkPagesServer(pages)
	defer srv.Close()

	t.Run("no filter emits all", func(t *testing.T) {
		s := New()
		s.baseURL = srv.URL

		var got []types.FoodMatch
		err := s.FetchBulk(context.Background(), ports.BulkFilter{}, func(fm types.FoodMatch) error {
			got = append(got, fm)
			return nil
		})
		requireFetchBulkCount(t, err, got, 6)
		assertFirstBulkProductMapped(t, got[0])
	})

	t.Run("MaxRows caps emitted count and stops early", func(t *testing.T) {
		s := New()
		s.baseURL = srv.URL

		var got []types.FoodMatch
		err := s.FetchBulk(context.Background(), ports.BulkFilter{MaxRows: 3}, func(fm types.FoodMatch) error {
			got = append(got, fm)
			return nil
		})
		requireFetchBulkCount(t, err, got, 3)
	})

	t.Run("MinPopularity filters low-scan products", func(t *testing.T) {
		s := New()
		s.baseURL = srv.URL

		var got []types.FoodMatch
		err := s.FetchBulk(context.Background(), ports.BulkFilter{MinPopularity: 10}, func(fm types.FoodMatch) error {
			got = append(got, fm)
			return nil
		})
		requireFetchBulkCount(t, err, got, 3)
		assertOnlyFoodIDs(t, got, "A", "C", "E")
	})

	t.Run("emit error aborts early", func(t *testing.T) {
		s := New()
		s.baseURL = srv.URL

		wantErr := errors.New("boom")
		n := 0
		err := s.FetchBulk(context.Background(), ports.BulkFilter{}, func(fm types.FoodMatch) error {
			n++
			if n == 1 {
				return wantErr
			}
			return nil
		})
		if !errors.Is(err, wantErr) {
			t.Fatalf("err = %v, want %v", err, wantErr)
		}
		if n != 1 {
			t.Errorf("emit called %d times, want 1 (should abort on first error)", n)
		}
	})
}

func TestFetchBulkAPIRetriesTransientStatus(t *testing.T) {
	orig := bulkRetryBackoff
	bulkRetryBackoff = time.Millisecond
	defer func() { bulkRetryBackoff = orig }()

	var page1Attempts int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		page, _ := strconv.Atoi(r.URL.Query().Get("page"))
		setJSONContentType(w)
		switch page {
		case 1:
			page1Attempts++
			if page1Attempts < 3 {
				w.WriteHeader(http.StatusServiceUnavailable)
				return
			}
			_, _ = w.Write([]byte(bulkPageJSON([]bulkPage{{Code: "A", Name: productAName, ScansN: 10}})))
		default:
			_, _ = w.Write([]byte(`{"count":0,"products":[]}`))
		}
	}))
	defer srv.Close()

	s := New()
	s.baseURL = srv.URL

	var got []types.FoodMatch
	err := s.FetchBulk(context.Background(), ports.BulkFilter{}, func(fm types.FoodMatch) error {
		got = append(got, fm)
		return nil
	})
	if err != nil {
		t.Fatalf("FetchBulk: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("emitted %d products, want 1", len(got))
	}
	if page1Attempts != 3 {
		t.Errorf("page 1 was requested %d times, want 3 (2 failures + 1 success)", page1Attempts)
	}
}

func TestFetchBulkAPINonTransientStatusFailsImmediately(t *testing.T) {
	orig := bulkRetryBackoff
	bulkRetryBackoff = time.Millisecond
	defer func() { bulkRetryBackoff = orig }()

	var attempts int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	s := New()
	s.baseURL = srv.URL

	err := s.FetchBulk(context.Background(), ports.BulkFilter{}, func(fm types.FoodMatch) error {
		return nil
	})
	if err == nil {
		t.Fatal("expected error for 404")
	}
	if attempts != 1 {
		t.Errorf("request made %d times, want 1 (404 should not be retried)", attempts)
	}
}

func TestFetchBulkFile(t *testing.T) {
	lines := []string{
		bulkProductJSON(bulkPage{Code: "A", Name: productAName, ScansN: 100}),
		bulkProductJSON(bulkPage{Code: "B", Name: "Product B", ScansN: 5}),
		bulkProductJSON(bulkPage{Code: "C", Name: "Product C", ScansN: 50}),
	}

	path := filepath.Join(t.TempDir(), "off-export.jsonl.gz")
	f, err := os.Create(path)
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	gz := gzip.NewWriter(f)
	for _, l := range lines {
		if _, err := gz.Write([]byte(l + "\n")); err != nil {
			t.Fatalf("write: %v", err)
		}
	}
	if err := gz.Close(); err != nil {
		t.Fatalf("gzip close: %v", err)
	}
	if err := f.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}

	t.Run("streams and filters by popularity", func(t *testing.T) {
		s := NewBulk(path)

		var got []types.FoodMatch
		err := s.FetchBulk(context.Background(), ports.BulkFilter{MinPopularity: 10}, func(fm types.FoodMatch) error {
			got = append(got, fm)
			return nil
		})
		requireFetchBulkCount(t, err, got, 2)
	})

	t.Run("emit error aborts early", func(t *testing.T) {
		s := NewBulk(path)

		wantErr := errors.New("boom")
		n := 0
		err := s.FetchBulk(context.Background(), ports.BulkFilter{}, func(fm types.FoodMatch) error {
			n++
			return wantErr
		})
		if !errors.Is(err, wantErr) {
			t.Fatalf("err = %v, want %v", err, wantErr)
		}
		if n != 1 {
			t.Errorf("emit called %d times, want 1", n)
		}
	})

	t.Run("MaxRows caps emitted count and stops early", func(t *testing.T) {
		s := NewBulk(path)

		var got []types.FoodMatch
		err := s.FetchBulk(context.Background(), ports.BulkFilter{MaxRows: 1}, func(fm types.FoodMatch) error {
			got = append(got, fm)
			return nil
		})
		requireFetchBulkCount(t, err, got, 1)
	})
}

// requireFetchBulkCount fails the test if a FetchBulk call errored or didn't
// emit exactly wantLen matches.
func requireFetchBulkCount(t *testing.T, err error, got []types.FoodMatch, wantLen int) {
	t.Helper()
	if err != nil {
		t.Fatalf("FetchBulk: %v", err)
	}
	if len(got) != wantLen {
		t.Fatalf("emitted %d products, want %d", len(got), wantLen)
	}
}
