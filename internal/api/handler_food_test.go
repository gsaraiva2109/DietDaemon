package api

import (
	"errors"
	"net/http"
	"testing"

	"github.com/gsaraiva2109/dietdaemon/core/types"
)

// TestFoodHandlersRequireAuth is a table-driven sweep asserting every
// food-discovery/catalog/alias/library/precedence route 401s with no
// Authorization header, before any store method is ever reached.
func TestFoodHandlersRequireAuth(t *testing.T) {
	routes := []struct {
		method string
		path   string
	}{
		{"GET", "/api/v1/foods"},
		{"GET", "/api/v1/foods/search?q=chicken"},
		{"GET", "/api/v1/foods/frequent"},
		{"GET", "/api/v1/foods/f1"},
		{"POST", "/api/v1/foods/custom"},
		{"PUT", "/api/v1/foods/f1/custom"},
		{"DELETE", "/api/v1/foods/f1/custom"},
		{"POST", "/api/v1/foods/f1/units"},
		{"DELETE", "/api/v1/foods/f1/units/u1"},
		{"POST", "/api/v1/foods/f1/aliases"},
		{"DELETE", "/api/v1/foods/f1/aliases/a1"},
		{"GET", "/api/v1/catalog/search"},
		{"DELETE", "/api/v1/foods/f1/library"},
		{"POST", "/api/v1/foods/f1/library"},
		{"GET", "/api/v1/aliases/pending"},
		{"POST", "/api/v1/aliases/pending/p1/confirm"},
		{"DELETE", "/api/v1/aliases/pending/p1"},
		{"GET", "/api/v1/settings/precedence"},
		{"PUT", "/api/v1/settings/precedence"},
		{"GET", "/api/v1/food-import/status"},
	}
	store := newFakeMealStore()
	h := newHandler(store, &fakeMealLogger{})
	for _, rt := range routes {
		t.Run(rt.method+" "+rt.path, func(t *testing.T) {
			rec := doRequest(h, rt.method, rt.path, nil, map[string]string{"Authorization": ""})
			if rec.Code != http.StatusUnauthorized {
				t.Errorf("expected 401, got %d: %s", rec.Code, rec.Body.String())
			}
		})
	}
}

// --- handleListFoods ---

func TestHandleListFoods(t *testing.T) {
	store := newFakeMealStore()
	store.foodList = []types.FoodDetail{{FoodID: "f1", Name: "Chicken"}}
	h := newHandler(store, &fakeMealLogger{})

	rec := doRequest(h, "GET", "/api/v1/foods", nil, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	foods := decodeJSON[[]types.FoodDetail](t, rec)
	if len(foods) != 1 || foods[0].Name != "Chicken" {
		t.Errorf("unexpected foods: %+v", foods)
	}
}

func TestHandleListFoodsInvalidLimit(t *testing.T) {
	store := newFakeMealStore()
	h := newHandler(store, &fakeMealLogger{})

	rec := doRequest(h, "GET", "/api/v1/foods?limit=999", nil, nil)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("out-of-range limit expected 400, got %d", rec.Code)
	}
}

// --- handleSearchFoods ---

func TestHandleSearchFoods(t *testing.T) {
	store := newFakeMealStore()
	store.foodList = []types.FoodDetail{{FoodID: "f1", Name: "Chicken"}}
	h := newHandler(store, &fakeMealLogger{})

	rec := doRequest(h, "GET", "/api/v1/foods/search?q=chicken", nil, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	foods := decodeJSON[[]types.FoodDetail](t, rec)
	if len(foods) != 1 {
		t.Errorf("foods count = %d, want 1", len(foods))
	}
}

func TestHandleSearchFoodsMissingQuery(t *testing.T) {
	store := newFakeMealStore()
	h := newHandler(store, &fakeMealLogger{})

	rec := doRequest(h, "GET", "/api/v1/foods/search", nil, nil)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("missing q expected 400, got %d", rec.Code)
	}
}

// --- handleFrequentFoods ---

func TestHandleFrequentFoods(t *testing.T) {
	store := newFakeMealStore()
	store.foodList = []types.FoodDetail{{FoodID: "f1", Name: "Chicken"}}
	h := newHandler(store, &fakeMealLogger{})

	rec := doRequest(h, "GET", "/api/v1/foods/frequent", nil, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	foods := decodeJSON[[]types.FoodDetail](t, rec)
	if len(foods) != 1 {
		t.Errorf("foods count = %d, want 1", len(foods))
	}
}

func TestHandleFrequentFoodsInvalidLimit(t *testing.T) {
	store := newFakeMealStore()
	h := newHandler(store, &fakeMealLogger{})

	rec := doRequest(h, "GET", "/api/v1/foods/frequent?limit=999", nil, nil)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("out-of-range limit expected 400, got %d", rec.Code)
	}
}

// --- handleGetFood (additional cases beyond the two already covered) ---

func TestHandleGetFoodInLibrarySuccess(t *testing.T) {
	store := newFakeMealStore()
	store.foodDetail = types.FoodDetail{
		FoodID:    "f1",
		Name:      "Chicken Breast",
		InLibrary: true,
		Aliases:   []types.FoodAlias{{FoodID: "f1", Alias: "chkn"}},
	}
	h := newHandler(store, &fakeMealLogger{})

	rec := doRequest(h, "GET", "/api/v1/foods/f1", nil, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	fd := decodeJSON[types.FoodDetail](t, rec)
	if fd.Name != "Chicken Breast" || !fd.InLibrary {
		t.Errorf("unexpected food detail: %+v", fd)
	}
	if len(fd.Aliases) != 1 {
		t.Errorf("expected 1 alias, got %v", fd.Aliases)
	}
}

func TestHandleGetFoodDetailStoreError(t *testing.T) {
	store := newFakeMealStore()
	store.foodDetailErr = errors.New("db unavailable")
	h := newHandler(store, &fakeMealLogger{})

	rec := doRequest(h, "GET", "/api/v1/foods/f1", nil, nil)
	if rec.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d: %s", rec.Code, rec.Body.String())
	}
}

// --- handleCreateCustomFood ---

func validCustomFoodBody() map[string]any {
	return map[string]any{
		"name": "Test Food", "calories": 200.0, "protein": 20.0,
		"carbs": 10.0, "fat": 5.0, "fiber": 2.0, "basis_grams": 100.0,
	}
}

func TestHandleCreateCustomFood(t *testing.T) {
	store := newFakeMealStore()
	store.customFood = types.FoodDetail{FoodID: "custom-1", Name: "Test Food"}
	h := newHandler(store, &fakeMealLogger{})

	rec := doRequest(h, "POST", "/api/v1/foods/custom", validCustomFoodBody(), nil)
	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rec.Code, rec.Body.String())
	}
	fd := decodeJSON[types.FoodDetail](t, rec)
	if fd.FoodID != "custom-1" {
		t.Errorf("food_id = %q, want custom-1", fd.FoodID)
	}
	if store.createCustomFoodUser != "test-user" {
		t.Errorf("createCustomFoodUser = %q, want test-user", store.createCustomFoodUser)
	}
	if store.createCustomFoodIn.Name != "Test Food" || store.createCustomFoodIn.BasisGrams != 100.0 {
		t.Errorf("createCustomFoodIn = %+v, unexpected", store.createCustomFoodIn)
	}
}

func TestHandleCreateCustomFoodValidationError(t *testing.T) {
	store := newFakeMealStore()
	h := newHandler(store, &fakeMealLogger{})

	body := map[string]any{"calories": 200.0} // missing name and other required fields
	rec := doRequest(h, "POST", "/api/v1/foods/custom", body, nil)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("missing fields expected 400, got %d: %s", rec.Code, rec.Body.String())
	}
}

// --- handleUpdateCustomFood ---

func TestHandleUpdateCustomFood(t *testing.T) {
	store := newFakeMealStore()
	store.customFood = types.FoodDetail{Name: "Updated Food"}
	h := newHandler(store, &fakeMealLogger{})

	rec := doRequest(h, "PUT", "/api/v1/foods/f1/custom", validCustomFoodBody(), nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	fd := decodeJSON[types.FoodDetail](t, rec)
	if fd.Name != "Updated Food" {
		t.Errorf("name = %q, want Updated Food", fd.Name)
	}
	if store.updateCustomFoodUser != "test-user" || store.updateCustomFoodIn.Name != "Test Food" {
		t.Errorf("unexpected update capture: user=%q in=%+v", store.updateCustomFoodUser, store.updateCustomFoodIn)
	}
}

func TestHandleUpdateCustomFoodValidationError(t *testing.T) {
	store := newFakeMealStore()
	h := newHandler(store, &fakeMealLogger{})

	body := map[string]any{"name": "X"} // missing macros/basis_grams
	rec := doRequest(h, "PUT", "/api/v1/foods/f1/custom", body, nil)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("missing fields expected 400, got %d", rec.Code)
	}
}

func TestHandleUpdateCustomFoodNotFound(t *testing.T) {
	store := newFakeMealStore()
	store.updateCustomFoodErr = types.ErrNotFound
	h := newHandler(store, &fakeMealLogger{})

	rec := doRequest(h, "PUT", "/api/v1/foods/f1/custom", validCustomFoodBody(), nil)
	if rec.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d: %s", rec.Code, rec.Body.String())
	}
}

// --- handleDeleteCustomFood ---

func TestHandleDeleteCustomFood(t *testing.T) {
	store := newFakeMealStore()
	h := newHandler(store, &fakeMealLogger{})

	rec := doRequest(h, "DELETE", "/api/v1/foods/f1/custom", nil, nil)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d: %s", rec.Code, rec.Body.String())
	}
	if store.deleteCustomFoodUser != "test-user" {
		t.Errorf("deleteCustomFoodUser = %q, want test-user", store.deleteCustomFoodUser)
	}
}

func TestHandleDeleteCustomFoodNotFound(t *testing.T) {
	store := newFakeMealStore()
	store.deleteCustomFoodErr = types.ErrNotFound
	h := newHandler(store, &fakeMealLogger{})

	rec := doRequest(h, "DELETE", "/api/v1/foods/f1/custom", nil, nil)
	if rec.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d: %s", rec.Code, rec.Body.String())
	}
}

// --- handleCreateFoodServingUnit ---

func TestHandleCreateFoodServingUnit(t *testing.T) {
	store := newFakeMealStore()
	h := newHandler(store, &fakeMealLogger{})

	body := map[string]any{"label": "1 cup", "grams": 240.0}
	rec := doRequest(h, "POST", "/api/v1/foods/f1/units", body, nil)
	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rec.Code, rec.Body.String())
	}
	unit := decodeJSON[types.FoodServingUnit](t, rec)
	if unit.Label != "1 cup" || unit.Grams != 240.0 || !unit.Custom {
		t.Errorf("unexpected unit: %+v", unit)
	}
}

func TestHandleCreateFoodServingUnitValidationError(t *testing.T) {
	store := newFakeMealStore()
	h := newHandler(store, &fakeMealLogger{})

	body := map[string]any{"label": "", "grams": 240.0}
	rec := doRequest(h, "POST", "/api/v1/foods/f1/units", body, nil)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("empty label expected 400, got %d", rec.Code)
	}

	body2 := map[string]any{"label": "1 cup", "grams": 0.0}
	rec2 := doRequest(h, "POST", "/api/v1/foods/f1/units", body2, nil)
	if rec2.Code != http.StatusBadRequest {
		t.Errorf("zero grams expected 400, got %d", rec2.Code)
	}
}

func TestHandleCreateFoodServingUnitNotFound(t *testing.T) {
	store := newFakeMealStore()
	store.createServingUnitErr = types.ErrNotFound
	h := newHandler(store, &fakeMealLogger{})

	body := map[string]any{"label": "1 cup", "grams": 240.0}
	rec := doRequest(h, "POST", "/api/v1/foods/f1/units", body, nil)
	if rec.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d: %s", rec.Code, rec.Body.String())
	}
}

// --- handleDeleteFoodServingUnit ---

func TestHandleDeleteFoodServingUnit(t *testing.T) {
	store := newFakeMealStore()
	h := newHandler(store, &fakeMealLogger{})

	rec := doRequest(h, "DELETE", "/api/v1/foods/f1/units/u1", nil, nil)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestHandleDeleteFoodServingUnitNotFound(t *testing.T) {
	store := newFakeMealStore()
	store.deleteServingUnitErr = types.ErrNotFound
	h := newHandler(store, &fakeMealLogger{})

	rec := doRequest(h, "DELETE", "/api/v1/foods/f1/units/u1", nil, nil)
	if rec.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d: %s", rec.Code, rec.Body.String())
	}
}

// --- handleAddAlias ---

func TestHandleAddAlias(t *testing.T) {
	store := newFakeMealStore()
	h := newHandler(store, &fakeMealLogger{})

	rec := doRequest(h, "POST", "/api/v1/foods/f1/aliases", map[string]string{"alias": "chkn"}, nil)
	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestHandleAddAliasValidationError(t *testing.T) {
	store := newFakeMealStore()
	h := newHandler(store, &fakeMealLogger{})

	rec := doRequest(h, "POST", "/api/v1/foods/f1/aliases", map[string]string{"alias": ""}, nil)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("empty alias expected 400, got %d", rec.Code)
	}
}

func TestHandleAddAliasNotFound(t *testing.T) {
	store := newFakeMealStore()
	store.addAliasErr = types.ErrNotFound
	h := newHandler(store, &fakeMealLogger{})

	rec := doRequest(h, "POST", "/api/v1/foods/f1/aliases", map[string]string{"alias": "chkn"}, nil)
	if rec.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d: %s", rec.Code, rec.Body.String())
	}
}

// --- handleDeleteAlias ---

func TestHandleDeleteAlias(t *testing.T) {
	store := newFakeMealStore()
	h := newHandler(store, &fakeMealLogger{})

	rec := doRequest(h, "DELETE", "/api/v1/foods/f1/aliases/chkn", nil, nil)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestHandleDeleteAliasNotFound(t *testing.T) {
	store := newFakeMealStore()
	store.deleteAliasErr = types.ErrNotFound
	h := newHandler(store, &fakeMealLogger{})

	rec := doRequest(h, "DELETE", "/api/v1/foods/f1/aliases/chkn", nil, nil)
	if rec.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d: %s", rec.Code, rec.Body.String())
	}
}

// --- handleSearchCatalog ---

func TestHandleSearchCatalog(t *testing.T) {
	store := newFakeMealStore()
	store.foodList = []types.FoodDetail{{FoodID: "f1", Name: "Chicken"}}
	h := newHandler(store, &fakeMealLogger{})

	rec := doRequest(h, "GET", "/api/v1/catalog/search?q=chicken", nil, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	foods := decodeJSON[[]types.FoodDetail](t, rec)
	if len(foods) != 1 {
		t.Errorf("foods count = %d, want 1", len(foods))
	}
}

func TestHandleSearchCatalogInvalidLimit(t *testing.T) {
	store := newFakeMealStore()
	h := newHandler(store, &fakeMealLogger{})

	rec := doRequest(h, "GET", "/api/v1/catalog/search?limit=999", nil, nil)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("out-of-range limit expected 400, got %d", rec.Code)
	}
}

// --- handleRemoveFromLibrary ---

func TestHandleRemoveFromLibrary(t *testing.T) {
	store := newFakeMealStore()
	h := newHandler(store, &fakeMealLogger{})

	rec := doRequest(h, "DELETE", "/api/v1/foods/f1/library", nil, nil)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestHandleRemoveFromLibraryNotFound(t *testing.T) {
	store := newFakeMealStore()
	store.removeFromLibraryErr = types.ErrNotFound
	h := newHandler(store, &fakeMealLogger{})

	rec := doRequest(h, "DELETE", "/api/v1/foods/f1/library", nil, nil)
	if rec.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d: %s", rec.Code, rec.Body.String())
	}
}

// --- handleAddToLibrary ---

func TestHandleAddToLibrary(t *testing.T) {
	store := newFakeMealStore()
	h := newHandler(store, &fakeMealLogger{})

	rec := doRequest(h, "POST", "/api/v1/foods/f1/library", nil, nil)
	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rec.Code, rec.Body.String())
	}
	got := decodeJSON[map[string]string](t, rec)
	if got["status"] != "created" {
		t.Errorf("status = %q, want created", got["status"])
	}
}

func TestHandleAddToLibraryNotFound(t *testing.T) {
	store := newFakeMealStore()
	store.addToLibraryErr = types.ErrNotFound
	h := newHandler(store, &fakeMealLogger{})

	rec := doRequest(h, "POST", "/api/v1/foods/f1/library", nil, nil)
	if rec.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d: %s", rec.Code, rec.Body.String())
	}
}

// --- handleListPendingAliases ---

func TestHandleListPendingAliases(t *testing.T) {
	store := newFakeMealStore()
	store.pendingAliases = []types.PendingAlias{{ID: "p1", FoodID: "f1", Phrase: "chkn"}}
	store.foodDetail = types.FoodDetail{Name: "Chicken"}
	h := newHandler(store, &fakeMealLogger{})

	rec := doRequest(h, "GET", "/api/v1/aliases/pending", nil, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	views := decodeJSON[[]pendingAliasView](t, rec)
	if len(views) != 1 || views[0].FoodName != "Chicken" {
		t.Errorf("unexpected views: %+v", views)
	}
}

func TestHandleListPendingAliasesStoreError(t *testing.T) {
	store := newFakeMealStore()
	store.pendingAliasesErr = errors.New("db down")
	h := newHandler(store, &fakeMealLogger{})

	rec := doRequest(h, "GET", "/api/v1/aliases/pending", nil, nil)
	if rec.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", rec.Code)
	}
}

// --- handleConfirmPendingAlias ---

func TestHandleConfirmPendingAlias(t *testing.T) {
	store := newFakeMealStore()
	h := newHandler(store, &fakeMealLogger{})

	rec := doRequest(h, "POST", "/api/v1/aliases/pending/p1/confirm", nil, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	got := decodeJSON[map[string]string](t, rec)
	if got["status"] != "confirmed" {
		t.Errorf("status = %q, want confirmed", got["status"])
	}
}

func TestHandleConfirmPendingAliasNotFound(t *testing.T) {
	store := newFakeMealStore()
	store.confirmPendingAliasErr = types.ErrNotFound
	h := newHandler(store, &fakeMealLogger{})

	rec := doRequest(h, "POST", "/api/v1/aliases/pending/p1/confirm", nil, nil)
	if rec.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d: %s", rec.Code, rec.Body.String())
	}
}

// --- handleRejectPendingAlias ---

func TestHandleRejectPendingAlias(t *testing.T) {
	store := newFakeMealStore()
	h := newHandler(store, &fakeMealLogger{})

	rec := doRequest(h, "DELETE", "/api/v1/aliases/pending/p1", nil, nil)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestHandleRejectPendingAliasNotFound(t *testing.T) {
	store := newFakeMealStore()
	store.rejectPendingAliasErr = types.ErrNotFound
	h := newHandler(store, &fakeMealLogger{})

	rec := doRequest(h, "DELETE", "/api/v1/aliases/pending/p1", nil, nil)
	if rec.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d: %s", rec.Code, rec.Body.String())
	}
}

// --- handleGetPrecedence ---

func TestHandleGetPrecedence(t *testing.T) {
	store := newFakeMealStore()
	store.precedence = []string{"usda", "taco"}
	h := newHandler(store, &fakeMealLogger{})

	rec := doRequest(h, "GET", "/api/v1/settings/precedence", nil, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	got := decodeJSON[map[string][]string](t, rec)
	if len(got["order"]) != 2 || got["order"][0] != "usda" {
		t.Errorf("unexpected order: %v", got["order"])
	}
}

func TestHandleGetPrecedenceStoreError(t *testing.T) {
	store := newFakeMealStore()
	store.precedenceErr = errors.New("db down")
	h := newHandler(store, &fakeMealLogger{})

	rec := doRequest(h, "GET", "/api/v1/settings/precedence", nil, nil)
	if rec.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", rec.Code)
	}
}

// --- handleSetPrecedence ---

func TestHandleSetPrecedence(t *testing.T) {
	store := newFakeMealStore()
	h := newHandler(store, &fakeMealLogger{})

	body := map[string][]string{"order": {"usda", "taco"}}
	rec := doRequest(h, "PUT", "/api/v1/settings/precedence", body, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	if len(store.precedence) != 2 || store.precedence[0] != "usda" {
		t.Errorf("stored precedence = %v, unexpected", store.precedence)
	}
}

func TestHandleSetPrecedenceEmptyOrder(t *testing.T) {
	store := newFakeMealStore()
	h := newHandler(store, &fakeMealLogger{})

	rec := doRequest(h, "PUT", "/api/v1/settings/precedence", map[string][]string{"order": {}}, nil)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("empty order expected 400, got %d", rec.Code)
	}
}

func TestHandleSetPrecedenceInvalidSource(t *testing.T) {
	store := newFakeMealStore()
	h := newHandler(store, &fakeMealLogger{})

	rec := doRequest(h, "PUT", "/api/v1/settings/precedence", map[string][]string{"order": {"not-a-real-source"}}, nil)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("invalid source expected 400, got %d", rec.Code)
	}
}

func TestHandleSetPrecedenceDuplicateSource(t *testing.T) {
	store := newFakeMealStore()
	h := newHandler(store, &fakeMealLogger{})

	rec := doRequest(h, "PUT", "/api/v1/settings/precedence", map[string][]string{"order": {"usda", "usda"}}, nil)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("duplicate source expected 400, got %d", rec.Code)
	}
}

// --- handleFoodImportStatus ---

func TestHandleFoodImportStatus(t *testing.T) {
	store := newFakeMealStore()
	store.foodImportStatuses = []types.FoodImportStatus{{Source: "usda", LastResult: "imported"}}
	h := newHandler(store, &fakeMealLogger{})

	rec := doRequest(h, "GET", "/api/v1/food-import/status", nil, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	statuses := decodeJSON[[]types.FoodImportStatus](t, rec)
	if len(statuses) != 1 || statuses[0].Source != "usda" {
		t.Errorf("unexpected statuses: %+v", statuses)
	}
}

func TestHandleFoodImportStatusStoreError(t *testing.T) {
	store := newFakeMealStore()
	store.foodImportStatusesErr = errors.New("db down")
	h := newHandler(store, &fakeMealLogger{})

	rec := doRequest(h, "GET", "/api/v1/food-import/status", nil, nil)
	if rec.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", rec.Code)
	}
}

// --- handleSuggestFromIngredients validation (happy path already covered
// in handler_test.go's TestHandleSuggestFromIngredients) ---

func TestHandleSuggestFromIngredientsEmptyIDs(t *testing.T) {
	store := newFakeMealStore()
	sug := &fakeSuggester{}
	h := newHandler(store, &fakeMealLogger{}, sug)

	rec := doRequest(h, "POST", "/api/v1/suggest/ingredients", map[string]any{"food_ids": []string{}}, nil)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("empty food_ids expected 400, got %d", rec.Code)
	}
}
