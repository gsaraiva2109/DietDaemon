package openfoodfacts

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gsaraiva2109/dietdaemon/core/types"
)

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

		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{
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
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"count":0,"products":[]}`))
	}))
	defer srv.Close()

	s := New()
	s.baseURL = srv.URL

	_, err := s.Resolve(context.Background(), types.ParsedItem{RawPhrase: "nonexistent"})
	if err != types.ErrNoMatch {
		t.Errorf("expected ErrNoMatch, got %v", err)
	}
}

func TestResolveProductWithoutEnergy(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{
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
	if err != types.ErrNoMatch {
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
	if err != types.ErrNoMatch {
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
