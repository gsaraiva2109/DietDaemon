package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/gsaraiva2109/dietdaemon/core/types"
)

// ---------------------------------------------------------------------------
// Meal template handlers -- list, create, compose, get, delete, log, duplicate.
// ---------------------------------------------------------------------------

func (h *Handler) handleListTemplates(w http.ResponseWriter, r *http.Request, userID string) {
	templates, err := h.store.GetTemplates(r.Context(), userID)
	if err != nil {
		h.writeErr(w, err)
		return
	}
	if templates == nil {
		templates = []types.MealTemplate{}
	}
	_ = json.NewEncoder(w).Encode(templates)
}

func (h *Handler) handleCreateTemplate(w http.ResponseWriter, r *http.Request, userID string) {
	var body struct {
		Name  string               `json:"name"`
		Items []types.ResolvedItem `json:"items"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid JSON body: " + err.Error()})
		return
	}
	if body.Name == "" || len(body.Items) == 0 {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "name and items are required"})
		return
	}
	now := time.Now().UTC()
	t := types.MealTemplate{
		ID:        newHandlerID(),
		UserID:    userID,
		Name:      body.Name,
		Items:     body.Items,
		CreatedAt: now,
		LastUsed:  now,
	}
	if err := h.store.SaveTemplate(r.Context(), t); err != nil {
		h.writeErr(w, err)
		return
	}
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(t)
}

func (h *Handler) handleComposeTemplate(w http.ResponseWriter, r *http.Request, userID string) {
	var body struct {
		Name  string `json:"name"`
		Items []struct {
			FoodID string  `json:"food_id"`
			Grams  float64 `json:"grams"`
		} `json:"items"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid JSON body: " + err.Error()})
		return
	}
	if body.Name == "" || len(body.Items) == 0 {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "name and items are required"})
		return
	}

	items := make([]types.ResolvedItem, 0, len(body.Items))
	for _, it := range body.Items {
		food, err := h.store.GetFood(r.Context(), it.FoodID)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "unknown food_id: " + it.FoodID})
			return
		}
		items = append(items, types.ResolvedItem{
			Parsed: types.ParsedItem{RawPhrase: food.Name, NormalizedGrams: it.Grams},
			Match:  food,
			Macros: food.Per100g.Scale(it.Grams / 100.0),
		})
	}

	now := time.Now().UTC()
	t := types.MealTemplate{
		ID:        newHandlerID(),
		UserID:    userID,
		Name:      body.Name,
		Items:     items,
		CreatedAt: now,
		LastUsed:  now,
	}
	if err := h.store.SaveTemplate(r.Context(), t); err != nil {
		h.writeErr(w, err)
		return
	}
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(t)
}

func (h *Handler) handleGetTemplate(w http.ResponseWriter, r *http.Request, userID string) {
	templateID := r.PathValue("id")
	t, err := h.store.GetTemplate(r.Context(), templateID)
	if err != nil {
		h.writeErr(w, err)
		return
	}
	if t.UserID != userID {
		h.writeErr(w, types.ErrNotFound)
		return
	}
	_ = json.NewEncoder(w).Encode(t)
}

func (h *Handler) handleDeleteTemplate(w http.ResponseWriter, r *http.Request, userID string) {
	templateID := r.PathValue("id")
	if err := h.store.DeleteTemplate(r.Context(), userID, templateID); err != nil {
		h.writeErr(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) handleLogTemplate(w http.ResponseWriter, r *http.Request, userID string) {
	templateID := r.PathValue("id")
	t, err := h.store.GetTemplate(r.Context(), templateID)
	if err != nil {
		h.writeErr(w, err)
		return
	}
	if t.UserID != userID {
		h.writeErr(w, types.ErrNotFound)
		return
	}

	now := time.Now().UTC()
	meal := types.Meal{
		ID:         newHandlerID(),
		UserID:     userID,
		At:         now,
		RawText:    fmt.Sprintf("template: %s", t.Name),
		Items:      t.Items,
		Confidence: 1.0,
		CreatedAt:  now,
	}
	if err := h.logger.LogMeal(r.Context(), meal); err != nil {
		h.writeErr(w, err)
		return
	}

	// Record the template usage.
	tl := types.TemplateLog{
		ID:         newHandlerID(),
		UserID:     userID,
		TemplateID: templateID,
		LoggedAt:   now,
	}
	_ = h.store.LogTemplateUse(r.Context(), tl)

	// Update template last_used.
	t.LastUsed = now
	_ = h.store.SaveTemplate(r.Context(), t)

	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "logged", "meal_id": meal.ID})
}

func (h *Handler) handleDuplicateMeal(w http.ResponseWriter, r *http.Request, userID string) {
	mealID := r.PathValue("mealID")
	original, err := h.store.GetMeal(r.Context(), mealID)
	if err != nil {
		h.writeErr(w, err)
		return
	}
	if original.UserID != userID {
		h.writeErr(w, types.ErrNotFound)
		return
	}

	now := time.Now().UTC()
	newMeal := types.Meal{
		ID:         newHandlerID(),
		UserID:     userID,
		At:         now,
		RawText:    fmt.Sprintf("duplicated: %s", original.RawText),
		Items:      original.Items,
		Confidence: 1.0,
		CreatedAt:  now,
	}
	if err := h.logger.LogMeal(r.Context(), newMeal); err != nil {
		h.writeErr(w, err)
		return
	}

	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "duplicated", "meal_id": newMeal.ID})
}
