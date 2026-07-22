package usda

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"testing"

	"github.com/gsaraiva2109/dietdaemon/core/ports"
	"github.com/gsaraiva2109/dietdaemon/core/types"
)

func TestResolve(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("api_key") != "test-key" {
			t.Errorf("missing or wrong api_key")
		}
		_ = json.NewEncoder(w).Encode(searchResponse{
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
	if !errors.Is(err, types.ErrNoMatch) {
		t.Errorf("expected ErrNoMatch, got %v", err)
	}
}

func TestResolveNoResults(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(searchResponse{Foods: []food{}})
	}))
	defer srv.Close()

	s := &Source{client: &http.Client{}, baseURL: srv.URL, apiKey: "k"}
	_, err := s.Resolve(t.Context(), types.ParsedItem{RawPhrase: "nonexistent"})
	if !errors.Is(err, types.ErrNoMatch) {
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

// ---------------------------------------------------------------------------
// FetchBulk — API mode
// ---------------------------------------------------------------------------

func TestFetchBulkAPI(t *testing.T) {
	orig := bulkPageDelay
	bulkPageDelay = 0
	defer func() { bulkPageDelay = orig }()

	pages := [][]food{
		{
			{FdcID: 1, Description: "Food A", FoodNutrients: []foodNutrient{{NutrientID: nutrientEnergy, Amount: 100}}},
			{FdcID: 2, Description: "Food B", FoodNutrients: []foodNutrient{{NutrientID: nutrientEnergy, Amount: 200}}},
		},
		{
			{FdcID: 3, Description: "Food C", FoodNutrients: []foodNutrient{{NutrientID: nutrientEnergy, Amount: 300}}},
			{FdcID: 4, Description: "Food D", FoodNutrients: []foodNutrient{{NutrientID: nutrientEnergy, Amount: 400}}},
		},
		{}, // empty page terminates the loop
	}
	calls := 0

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if got := r.URL.Query().Get("query"); got != "" {
			t.Errorf("query param = %q, want empty (bulk browse, not a name search)", got)
		}
		if got := r.URL.Query().Get("dataType"); got != "Foundation,SR Legacy" {
			t.Errorf("dataType = %q, want default %q", got, "Foundation,SR Legacy")
		}
		if got := r.URL.Query().Get("pageSize"); got != "200" {
			t.Errorf("pageSize = %q, want 200", got)
		}

		page, _ := strconv.Atoi(r.URL.Query().Get("pageNumber"))
		var foods []food
		if page >= 1 && page <= len(pages) {
			foods = pages[page-1]
		}
		_ = json.NewEncoder(w).Encode(searchResponse{Foods: foods})
	}))
	defer srv.Close()

	s := &Source{client: &http.Client{}, baseURL: srv.URL, apiKey: "test-key"}

	var got []types.FoodMatch
	err := s.FetchBulk(t.Context(), ports.BulkFilter{}, func(fm types.FoodMatch) error {
		got = append(got, fm)
		return nil
	})
	if err != nil {
		t.Fatalf("FetchBulk: %v", err)
	}
	if len(got) != 4 {
		t.Errorf("emitted %d matches, want 4", len(got))
	}
	if calls != 3 {
		t.Errorf("server called %d times, want 3 (2 data pages + empty terminator)", calls)
	}
}

func TestFetchBulkAPIMaxRows(t *testing.T) {
	orig := bulkPageDelay
	bulkPageDelay = 0
	defer func() { bulkPageDelay = orig }()

	calls := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		page, _ := strconv.Atoi(r.URL.Query().Get("pageNumber"))
		foods := []food{
			{FdcID: page*10 + 1, Description: "A", FoodNutrients: []foodNutrient{{NutrientID: nutrientEnergy, Amount: 100}}},
			{FdcID: page*10 + 2, Description: "B", FoodNutrients: []foodNutrient{{NutrientID: nutrientEnergy, Amount: 100}}},
		}
		_ = json.NewEncoder(w).Encode(searchResponse{Foods: foods}) // never-ending pages
	}))
	defer srv.Close()

	s := &Source{client: &http.Client{}, baseURL: srv.URL, apiKey: "k"}

	var got []types.FoodMatch
	err := s.FetchBulk(t.Context(), ports.BulkFilter{MaxRows: 3}, func(fm types.FoodMatch) error {
		got = append(got, fm)
		return nil
	})
	if err != nil {
		t.Fatalf("FetchBulk: %v", err)
	}
	if len(got) != 3 {
		t.Errorf("emitted %d matches, want 3 (MaxRows)", len(got))
	}
	if calls != 2 {
		t.Errorf("server called %d times, want 2 (must stop before requesting a 3rd page)", calls)
	}
}

func TestFetchBulkAPIEmitError(t *testing.T) {
	orig := bulkPageDelay
	bulkPageDelay = 0
	defer func() { bulkPageDelay = orig }()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(searchResponse{Foods: []food{
			{FdcID: 1, Description: "A", FoodNutrients: []foodNutrient{{NutrientID: nutrientEnergy, Amount: 100}}},
			{FdcID: 2, Description: "B", FoodNutrients: []foodNutrient{{NutrientID: nutrientEnergy, Amount: 100}}},
		}})
	}))
	defer srv.Close()

	s := &Source{client: &http.Client{}, baseURL: srv.URL, apiKey: "k"}

	wantErr := errors.New("boom")
	var got []types.FoodMatch
	err := s.FetchBulk(t.Context(), ports.BulkFilter{}, func(fm types.FoodMatch) error {
		got = append(got, fm)
		if len(got) == 1 {
			return wantErr
		}
		return nil
	})
	if !errors.Is(err, wantErr) {
		t.Errorf("err = %v, want %v", err, wantErr)
	}
	if len(got) != 1 {
		t.Errorf("emitted %d matches, want 1 (must abort right after emit errors)", len(got))
	}
}

// ---------------------------------------------------------------------------
// FetchBulk — file mode
// ---------------------------------------------------------------------------

func writeBulkFile(t *testing.T, entries []food) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "bulk.json")
	b, err := json.Marshal(entries)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if err := os.WriteFile(path, b, 0o600); err != nil {
		t.Fatalf("write bulk file: %v", err)
	}
	return path
}

func TestFetchBulkFile(t *testing.T) {
	path := writeBulkFile(t, []food{
		{FdcID: 1, Description: "Apple", DataType: "Foundation", FoodNutrients: []foodNutrient{{NutrientID: nutrientEnergy, Amount: 52}}},
		{FdcID: 2, Description: "Banana", DataType: "SR Legacy", FoodNutrients: []foodNutrient{{NutrientID: nutrientEnergy, Amount: 89}}},
		{FdcID: 3, Description: "Branded Bar", DataType: "Branded", FoodNutrients: []foodNutrient{{NutrientID: nutrientEnergy, Amount: 400}}},
		{FdcID: 4, Description: "Survey Item", DataType: "Survey (FNDDS)", FoodNutrients: []foodNutrient{{NutrientID: nutrientEnergy, Amount: 200}}},
		{FdcID: 5, Description: "Carrot", DataType: "Foundation", FoodNutrients: []foodNutrient{{NutrientID: nutrientEnergy, Amount: 41}}},
	})

	s := NewBulk("", path)

	var got []types.FoodMatch
	if err := s.FetchBulk(t.Context(), ports.BulkFilter{}, func(fm types.FoodMatch) error {
		got = append(got, fm)
		return nil
	}); err != nil {
		t.Fatalf("FetchBulk: %v", err)
	}

	if len(got) != 3 {
		t.Fatalf("emitted %d matches, want 3 (Foundation + SR Legacy only)", len(got))
	}
	for _, fm := range got {
		if fm.Name == "Branded Bar" || fm.Name == "Survey Item" {
			t.Errorf("emitted disallowed dataType food: %s", fm.Name)
		}
	}
}

func TestFetchBulkFileMaxRows(t *testing.T) {
	path := writeBulkFile(t, []food{
		{FdcID: 1, Description: "Apple", DataType: "Foundation", FoodNutrients: []foodNutrient{{NutrientID: nutrientEnergy, Amount: 52}}},
		{FdcID: 2, Description: "Banana", DataType: "SR Legacy", FoodNutrients: []foodNutrient{{NutrientID: nutrientEnergy, Amount: 89}}},
		{FdcID: 3, Description: "Carrot", DataType: "Foundation", FoodNutrients: []foodNutrient{{NutrientID: nutrientEnergy, Amount: 41}}},
	})

	s := NewBulk("", path)

	var got []types.FoodMatch
	err := s.FetchBulk(t.Context(), ports.BulkFilter{MaxRows: 1}, func(fm types.FoodMatch) error {
		got = append(got, fm)
		return nil
	})
	if err != nil {
		t.Fatalf("FetchBulk: %v", err)
	}
	if len(got) != 1 {
		t.Errorf("emitted %d matches, want 1 (MaxRows)", len(got))
	}
}

// TestFetchBulkFileWrappedObject covers the shape USDA's real FoodData
// Central full downloads actually use: the array is wrapped in an object
// under a key like "FoundationFoods", not a bare top-level array.
func TestFetchBulkFileWrappedObject(t *testing.T) {
	entries := []food{
		{FdcID: 1, Description: "Apple", DataType: "Foundation", FoodNutrients: []foodNutrient{{NutrientID: nutrientEnergy, Amount: 52}}},
		{FdcID: 2, Description: "Banana", DataType: "SR Legacy", FoodNutrients: []foodNutrient{{NutrientID: nutrientEnergy, Amount: 89}}},
		{FdcID: 3, Description: "Branded Bar", DataType: "Branded", FoodNutrients: []foodNutrient{{NutrientID: nutrientEnergy, Amount: 400}}},
	}
	b, err := json.Marshal(map[string]any{"FoundationFoods": entries})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	path := filepath.Join(t.TempDir(), "wrapped.json")
	if err := os.WriteFile(path, b, 0o600); err != nil {
		t.Fatalf("write bulk file: %v", err)
	}

	s := NewBulk("", path)

	var got []types.FoodMatch
	if err := s.FetchBulk(t.Context(), ports.BulkFilter{}, func(fm types.FoodMatch) error {
		got = append(got, fm)
		return nil
	}); err != nil {
		t.Fatalf("FetchBulk: %v", err)
	}

	if len(got) != 2 {
		t.Fatalf("emitted %d matches, want 2 (Foundation + SR Legacy only)", len(got))
	}
	for _, fm := range got {
		if fm.Name == "Branded Bar" {
			t.Errorf("emitted disallowed dataType food: %s", fm.Name)
		}
	}
}

// TestFoodCategoryUnmarshalObjectShape covers the actual crash seen against a
// real USDA export: foodCategory is an object for Foundation/SR Legacy/Survey
// foods (e.g. {"id":53,"code":"0100","description":"Dairy and Egg Products"}),
// not the plain string the field was originally typed as.
func TestFoodCategoryUnmarshalObjectShape(t *testing.T) {
	raw := `{
		"fdcId": 42,
		"description": "Milk, whole",
		"dataType": "Foundation",
		"foodCategory": {"id": 53, "code": "0100", "description": "Dairy and Egg Products"},
		"foodNutrients": [{"nutrientId": 1008, "amount": 61}]
	}`
	var f food
	if err := json.Unmarshal([]byte(raw), &f); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	fm, ok := foodToMatch(f)
	if !ok {
		t.Fatal("foodToMatch: ok = false, want true")
	}
	if fm.Category != "Dairy and Egg Products" {
		t.Errorf("Category = %q, want %q", fm.Category, "Dairy and Egg Products")
	}
}

// TestFoodCategoryUnmarshalStringShape covers a plain-string foodCategory,
// still supported alongside the object shape.
func TestFoodCategoryUnmarshalStringShape(t *testing.T) {
	raw := `{
		"fdcId": 7,
		"description": "Snack Bar",
		"dataType": "Branded",
		"brandedFoodCategory": "Snack Bars",
		"foodNutrients": [{"nutrientId": 1008, "amount": 400}]
	}`
	var f food
	if err := json.Unmarshal([]byte(raw), &f); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	fm, ok := foodToMatch(f)
	if !ok {
		t.Fatal("foodToMatch: ok = false, want true")
	}
	if fm.Category != "Snack Bars" {
		t.Errorf("Category = %q, want %q (brandedFoodCategory should take priority)", fm.Category, "Snack Bars")
	}
}

// TestFoodPortionsToServingUnits covers foodPortions → FoodServingUnit
// mapping (#134/B3): PortionDescription preferred as label, "undetermined"
// and non-positive gramWeight entries skipped, amount+modifier fallback used
// when there's no description.
func TestFoodPortionsToServingUnits(t *testing.T) {
	raw := `{
		"fdcId": 1,
		"description": "Egg, whole, raw",
		"foodNutrients": [{"nutrientId": 1008, "amount": 143}],
		"foodPortions": [
			{"amount": 1, "modifier": "large", "portionDescription": "1 large", "gramWeight": 50},
			{"amount": 1, "modifier": "", "portionDescription": "undetermined", "gramWeight": 44},
			{"amount": 0, "modifier": "", "portionDescription": "", "gramWeight": 0},
			{"amount": 2, "modifier": "small eggs", "portionDescription": "", "gramWeight": 72}
		]
	}`
	var f food
	if err := json.Unmarshal([]byte(raw), &f); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	fm, ok := foodToMatch(f)
	if !ok {
		t.Fatal("foodToMatch: ok = false, want true")
	}
	want := []types.FoodServingUnit{
		{Label: "1 large", Grams: 50},
		{Label: "1", Grams: 44},
		{Label: "2 small eggs", Grams: 72},
	}
	if len(fm.ServingUnits) != len(want) {
		t.Fatalf("ServingUnits = %+v, want %+v", fm.ServingUnits, want)
	}
	for i, u := range fm.ServingUnits {
		if u != want[i] {
			t.Errorf("ServingUnits[%d] = %+v, want %+v", i, u, want[i])
		}
	}
}

// TestResolveRequestsFullFormat confirms Resolve asks USDA for format=full so
// foodPortions is present in the response (#134/B3) — without it USDA omits
// household-measure data entirely.
func TestResolveRequestsFullFormat(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.URL.Query().Get("format"); got != "full" {
			t.Errorf("format = %q, want %q", got, "full")
		}
		_ = json.NewEncoder(w).Encode(searchResponse{Foods: []food{{
			FdcID: 1, Description: "x", FoodNutrients: []foodNutrient{{NutrientID: nutrientEnergy, Amount: 1}},
		}}})
	}))
	defer srv.Close()

	s := &Source{client: &http.Client{}, baseURL: srv.URL, apiKey: "k"}
	if _, err := s.Resolve(t.Context(), types.ParsedItem{RawPhrase: "x"}); err != nil {
		t.Fatalf("Resolve: %v", err)
	}
}

func TestFetchBulkFileEmitError(t *testing.T) {
	path := writeBulkFile(t, []food{
		{FdcID: 1, Description: "Apple", DataType: "Foundation", FoodNutrients: []foodNutrient{{NutrientID: nutrientEnergy, Amount: 52}}},
		{FdcID: 2, Description: "Carrot", DataType: "Foundation", FoodNutrients: []foodNutrient{{NutrientID: nutrientEnergy, Amount: 41}}},
	})

	s := NewBulk("", path)

	wantErr := errors.New("boom")
	var got []types.FoodMatch
	err := s.FetchBulk(t.Context(), ports.BulkFilter{}, func(fm types.FoodMatch) error {
		got = append(got, fm)
		return wantErr
	})
	if !errors.Is(err, wantErr) {
		t.Errorf("err = %v, want %v", err, wantErr)
	}
	if len(got) != 1 {
		t.Errorf("emitted %d matches, want 1 (must abort right after emit errors)", len(got))
	}
}
