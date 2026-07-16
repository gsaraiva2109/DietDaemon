package api

import (
	"encoding/json"
	"errors"
	"math"
	"net/http"
	"strconv"
	"strings"

	"github.com/gsaraiva2109/dietdaemon/core/types"
)

// ---------------------------------------------------------------------------
// Food discovery handlers -- list, search, frequent foods, suggestions, aliases, pending aliases, source precedence.
// ---------------------------------------------------------------------------

func (h *Handler) handleListFoods(w http.ResponseWriter, r *http.Request, userID string) {
	source := r.URL.Query().Get("source")
	limit := 20
	if s := r.URL.Query().Get("limit"); s != "" {
		if n, err := strconv.Atoi(s); err == nil && n > 0 && n <= 100 {
			limit = n
		}
	}
	offset := 0
	if s := r.URL.Query().Get("offset"); s != "" {
		if n, err := strconv.Atoi(s); err == nil && n >= 0 {
			offset = n
		}
	}
	foods, err := h.store.ListFoods(r.Context(), userID, source, limit, offset)
	if err != nil {
		h.writeErr(w, err)
		return
	}
	if foods == nil {
		foods = []types.FoodDetail{}
	}
	_ = json.NewEncoder(w).Encode(foods)
}

func (h *Handler) handleSearchFoods(w http.ResponseWriter, r *http.Request, userID string) {
	q := r.URL.Query().Get("q")
	if q == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "q query param is required"})
		return
	}
	foods, err := h.store.SearchFoods(r.Context(), userID, q)
	if err != nil {
		h.writeErr(w, err)
		return
	}
	if foods == nil {
		foods = []types.FoodDetail{}
	}
	_ = json.NewEncoder(w).Encode(foods)
}

func (h *Handler) handleFrequentFoods(w http.ResponseWriter, r *http.Request, userID string) {
	limit := 10
	if s := r.URL.Query().Get("limit"); s != "" {
		if n, err := strconv.Atoi(s); err == nil && n > 0 && n <= 50 {
			limit = n
		}
	}
	foods, err := h.store.FrequentFoods(r.Context(), userID, limit)
	if err != nil {
		h.writeErr(w, err)
		return
	}
	if foods == nil {
		foods = []types.FoodDetail{}
	}
	_ = json.NewEncoder(w).Encode(foods)
}

// handleSuggest recommends a next meal from what's left of today's targets.
func (h *Handler) handleSuggest(w http.ResponseWriter, r *http.Request, userID string) {
	ctx := h.injectModelOverride(r.Context(), userID)
	sug, err := h.suggester.Suggest(ctx, userID)
	if err != nil {
		h.writeErr(w, err)
		return
	}
	_ = json.NewEncoder(w).Encode(sug)
}

// handleSuggestFromIngredients recommends a next meal scoped to a caller-
// supplied list of on-hand food IDs, instead of the user's frequently-logged
// foods.
func (h *Handler) handleSuggestFromIngredients(w http.ResponseWriter, r *http.Request, userID string) {
	var body struct {
		FoodIDs []string `json:"food_ids"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || len(body.FoodIDs) == 0 {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "food_ids field is required"})
		return
	}
	ctx := h.injectModelOverride(r.Context(), userID)
	sug, err := h.suggester.SuggestFromIngredients(ctx, userID, body.FoodIDs)
	if err != nil {
		h.writeErr(w, err)
		return
	}
	_ = json.NewEncoder(w).Encode(sug)
}

func (h *Handler) handleGetFood(w http.ResponseWriter, r *http.Request, userID string) {
	foodID := r.PathValue("foodID")
	fd, err := h.store.GetFoodDetail(r.Context(), userID, foodID)
	if err != nil {
		if !errors.Is(err, types.ErrNotFound) {
			h.writeErr(w, err)
			return
		}
		// Not in this user's library — fall back to the global catalog so
		// catalog-only foods (bulk-imported, never logged) are still openable.
		match, matchErr := h.store.GetFoodForUser(r.Context(), userID, foodID)
		if matchErr != nil {
			h.writeErr(w, matchErr)
			return
		}
		fd = types.FoodDetail{
			FoodID:      match.FoodID,
			UserID:      userID,
			Name:        match.Name,
			Source:      match.Source,
			Per100g:     match.Per100g,
			Category:    match.Category,
			Brand:       match.Brand,
			Barcode:     match.Barcode,
			ImageURL:    match.ImageURL,
			ServingSize: match.ServingSize,
			ServingUnit: match.ServingUnit,
			InLibrary:   false,
			QueryCount:  0,
			LastUsed:    "",
			Aliases:     []types.FoodAlias{},
		}
	}
	if fd.Aliases == nil {
		fd.Aliases = []types.FoodAlias{}
	}
	_ = json.NewEncoder(w).Encode(fd)
}

type customFoodRequest struct {
	Name       *string  `json:"name"`
	Calories   *float64 `json:"calories"`
	Protein    *float64 `json:"protein"`
	Carbs      *float64 `json:"carbs"`
	Fat        *float64 `json:"fat"`
	Fiber      *float64 `json:"fiber"`
	BasisGrams *float64 `json:"basis_grams"`
}

func (b customFoodRequest) input() (types.CustomFoodInput, bool) {
	if b.Name == nil || strings.TrimSpace(*b.Name) == "" || b.Calories == nil || b.Protein == nil || b.Carbs == nil || b.Fat == nil || b.Fiber == nil || b.BasisGrams == nil || *b.BasisGrams <= 0 || !finite(*b.BasisGrams) {
		return types.CustomFoodInput{}, false
	}
	macros := types.Macros{Calories: *b.Calories, Protein: *b.Protein, Carbs: *b.Carbs, Fat: *b.Fat, Fiber: *b.Fiber}
	for _, value := range []float64{macros.Calories, macros.Protein, macros.Carbs, macros.Fat, macros.Fiber} {
		if value < 0 || !finite(value) {
			return types.CustomFoodInput{}, false
		}
	}
	return types.CustomFoodInput{Name: strings.TrimSpace(*b.Name), Macros: macros, BasisGrams: *b.BasisGrams}, true
}

func finite(value float64) bool { return !math.IsNaN(value) && !math.IsInf(value, 0) }

func decodeCustomFood(r *http.Request) (types.CustomFoodInput, error) {
	var body customFoodRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		return types.CustomFoodInput{}, err
	}
	input, ok := body.input()
	if !ok {
		return types.CustomFoodInput{}, errors.New("name, calories, protein, carbs, fat, fiber, and positive basis_grams are required")
	}
	return input, nil
}

func (h *Handler) handleCreateCustomFood(w http.ResponseWriter, r *http.Request, userID string) {
	input, err := decodeCustomFood(r)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}
	food, err := h.store.CreateCustomFood(r.Context(), userID, input)
	if err != nil {
		h.writeErr(w, err)
		return
	}
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(food)
}

func (h *Handler) handleUpdateCustomFood(w http.ResponseWriter, r *http.Request, userID string) {
	input, err := decodeCustomFood(r)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}
	food, err := h.store.UpdateCustomFood(r.Context(), userID, r.PathValue("foodID"), input)
	if err != nil {
		h.writeErr(w, err)
		return
	}
	_ = json.NewEncoder(w).Encode(food)
}

func (h *Handler) handleDeleteCustomFood(w http.ResponseWriter, r *http.Request, userID string) {
	if err := h.store.DeleteCustomFood(r.Context(), userID, r.PathValue("foodID")); err != nil {
		h.writeErr(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// handleSearchCatalog browses the full global food catalog, unscoped to the
// user's personal library (unlike handleSearchFoods, q is optional here).
func (h *Handler) handleSearchCatalog(w http.ResponseWriter, r *http.Request, userID string) {
	q := r.URL.Query().Get("q")
	source := r.URL.Query().Get("source")
	limit := 20
	if s := r.URL.Query().Get("limit"); s != "" {
		if n, err := strconv.Atoi(s); err == nil && n > 0 && n <= 100 {
			limit = n
		}
	}
	offset := 0
	if s := r.URL.Query().Get("offset"); s != "" {
		if n, err := strconv.Atoi(s); err == nil && n >= 0 {
			offset = n
		}
	}
	foods, err := h.store.SearchCatalog(r.Context(), userID, q, source, limit, offset)
	if err != nil {
		h.writeErr(w, err)
		return
	}
	if foods == nil {
		foods = []types.FoodDetail{}
	}
	_ = json.NewEncoder(w).Encode(foods)
}

func (h *Handler) handleRemoveFromLibrary(w http.ResponseWriter, r *http.Request, userID string) {
	foodID := r.PathValue("foodID")
	if err := h.store.RemoveFromLibrary(r.Context(), userID, foodID); err != nil {
		h.writeErr(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) handleAddToLibrary(w http.ResponseWriter, r *http.Request, userID string) {
	foodID := r.PathValue("foodID")
	if err := h.store.AddToLibrary(r.Context(), userID, foodID); err != nil {
		h.writeErr(w, err)
		return
	}
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "created"})
}

func (h *Handler) handleAddAlias(w http.ResponseWriter, r *http.Request, userID string) {
	foodID := r.PathValue("foodID")
	var body struct {
		Alias string `json:"alias"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Alias == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "alias field is required"})
		return
	}
	if err := h.store.AddFoodAlias(r.Context(), userID, foodID, body.Alias); err != nil {
		h.writeErr(w, err)
		return
	}
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "created"})
}

func (h *Handler) handleDeleteAlias(w http.ResponseWriter, r *http.Request, userID string) {
	foodID := r.PathValue("foodID")
	alias := r.PathValue("alias")
	if err := h.store.DeleteFoodAlias(r.Context(), userID, foodID, alias); err != nil {
		h.writeErr(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// pendingAliasView adds the matched food's display name to a pending alias so
// the UI can render "phrase -> food name" without a second round-trip per row.
type pendingAliasView struct {
	types.PendingAlias
	FoodName string `json:"food_name"`
}

func (h *Handler) handleListPendingAliases(w http.ResponseWriter, r *http.Request, userID string) {
	pending, err := h.store.ListPendingAliases(r.Context(), userID)
	if err != nil {
		h.writeErr(w, err)
		return
	}
	views := make([]pendingAliasView, 0, len(pending))
	for _, pa := range pending {
		view := pendingAliasView{PendingAlias: pa, FoodName: pa.FoodID}
		if fd, err := h.store.GetFoodDetail(r.Context(), userID, pa.FoodID); err == nil {
			view.FoodName = fd.Name
		}
		views = append(views, view)
	}
	_ = json.NewEncoder(w).Encode(views)
}

func (h *Handler) handleConfirmPendingAlias(w http.ResponseWriter, r *http.Request, userID string) {
	id := r.PathValue("id")
	if err := h.store.ConfirmPendingAlias(r.Context(), userID, id); err != nil {
		h.writeErr(w, err)
		return
	}
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "confirmed"})
}

func (h *Handler) handleRejectPendingAlias(w http.ResponseWriter, r *http.Request, userID string) {
	id := r.PathValue("id")
	if err := h.store.RejectPendingAlias(r.Context(), userID, id); err != nil {
		h.writeErr(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) handleGetPrecedence(w http.ResponseWriter, r *http.Request, userID string) {
	order, err := h.store.GetSourcePrecedence(r.Context(), userID)
	if err != nil {
		h.writeErr(w, err)
		return
	}
	if order == nil {
		order = []string{}
	}
	_ = json.NewEncoder(w).Encode(map[string][]string{"order": order})
}

func (h *Handler) handleFoodImportStatus(w http.ResponseWriter, r *http.Request, userID string) {
	statuses, err := h.store.GetFoodImportStatuses(r.Context())
	if err != nil {
		h.writeErr(w, err)
		return
	}
	if statuses == nil {
		statuses = []types.FoodImportStatus{}
	}
	_ = json.NewEncoder(w).Encode(statuses)
}

func (h *Handler) handleSetPrecedence(w http.ResponseWriter, r *http.Request, userID string) {
	var body struct {
		Order []string `json:"order"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "order field is required"})
		return
	}
	if err := h.store.SetSourcePrecedence(r.Context(), userID, body.Order); err != nil {
		h.writeErr(w, err)
		return
	}
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "updated"})
}
