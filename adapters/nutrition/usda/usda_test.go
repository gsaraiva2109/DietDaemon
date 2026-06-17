package usda

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gsaraiva2109/dietdaemon/core/types"
)

func TestResolve(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("api_key") != "test-key" {
			t.Errorf("missing or wrong api_key")
		}
		json.NewEncoder(w).Encode(searchResponse{
			Foods: []food{
				{
					FdcID:       12345,
					Description: "Chicken, breast, roasted",
					FoodNutrients: []foodNutrient{
						{NutrientID: nutrientEnergy, Amount: 165},
						{NutrientID: nutrientProtein, Amount: 31},
						{NutrientID: nutrientCarbs, Amount: 0},
						{NutrientID: nutrientFat, Amount: 3.6},
						{NutrientID: nutrientFiber, Amount: 0},
					},
				},
			},
		})
	}))
	defer srv.Close()

	s := &Source{client: &http.Client{}, baseURL: srv.URL, apiKey: "test-key"}
	match, err := s.Resolve(t.Context(), types.ParsedItem{RawPhrase: "chicken breast"})
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if match.FoodID != "12345" {
		t.Errorf("FoodID = %q, want %q", match.FoodID, "12345")
	}
	if match.Source != "usda" {
		t.Errorf("Source = %q, want %q", match.Source, "usda")
	}
	if match.Per100g.Calories != 165 || match.Per100g.Protein != 31 {
		t.Errorf("macros = %+v, want Calories=165 Protein=31", match.Per100g)
	}
}

func TestResolveEmptyPhrase(t *testing.T) {
	s := &Source{client: &http.Client{}, baseURL: DefaultBaseURL, apiKey: "k"}
	_, err := s.Resolve(t.Context(), types.ParsedItem{RawPhrase: ""})
	if err != types.ErrNoMatch {
		t.Errorf("expected ErrNoMatch, got %v", err)
	}
}

func TestResolveNoResults(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		json.NewEncoder(w).Encode(searchResponse{Foods: []food{}})
	}))
	defer srv.Close()

	s := &Source{client: &http.Client{}, baseURL: srv.URL, apiKey: "k"}
	_, err := s.Resolve(t.Context(), types.ParsedItem{RawPhrase: "nonexistent"})
	if err != types.ErrNoMatch {
		t.Errorf("expected ErrNoMatch, got %v", err)
	}
}

func TestResolveHTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	s := &Source{client: &http.Client{}, baseURL: srv.URL, apiKey: "k"}
	if _, err := s.Resolve(t.Context(), types.ParsedItem{RawPhrase: "chicken"}); err == nil {
		t.Error("expected error on 500, got nil")
	}
}
