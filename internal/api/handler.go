// Package api implements the REST API for the DietDaemon dashboard. It uses
// the Go standard library net/http and http.ServeMux for routing. All endpoints
// return JSON and are gated behind ENABLE_DASHBOARD=true.
package api

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/gsaraiva2109/dietdaemon/core/types"
)

// MealStore is the subset of the store the API needs for meal and rollup reads.
type MealStore interface {
	GetMeal(ctx context.Context, mealID string) (types.Meal, error)
	RecentMeals(ctx context.Context, userID string, limit int) ([]types.Meal, error)
	GetRollup(ctx context.Context, userID, localDate string) (types.DailyRollup, error)
	GetRollups(ctx context.Context, userID, startDate, endDate string) ([]types.DailyRollup, error)
	CorrectMealItem(ctx context.Context, userID string, mealID string, itemIndex int, corrected types.ResolvedItem) error
	AddMealItem(ctx context.Context, userID, mealID string, item types.ResolvedItem) error
	DeleteMealItem(ctx context.Context, userID, mealID string, itemIndex int) error
	GetTargets(ctx context.Context, userID string) (types.DailyTargets, error)
	SetTargets(ctx context.Context, t types.DailyTargets) error
	UpdateRollupTargets(ctx context.Context, userID, localDate string, t types.Macros) error
	GetUser(ctx context.Context, userID string) (types.User, error)
	ValidateToken(ctx context.Context, token string) (string, error)
}

// MealLogger submits raw text through the parsing pipeline. Satisfied by the
// pipeline.Engine.
type MealLogger interface {
	Handle(ctx context.Context, msg types.InboundMessage) error
}

// Handler serves the DietDaemon REST API.
type Handler struct {
	store     MealStore
	logger    MealLogger
	loc       *time.Location
	authToken string // empty = no auth check in single-user mode
	multiUser bool
}

// New returns a ready API Handler. authToken is the static bearer token for
// single-user mode; when empty, requests from localhost skip auth. In multi-user
// mode, tokens are validated against the api_tokens table via the store.
func New(store MealStore, logger MealLogger, loc *time.Location, authToken string, multiUser bool) *Handler {
	if loc == nil {
		loc = time.UTC
	}
	return &Handler{
		store:     store,
		logger:    logger,
		loc:       loc,
		authToken: authToken,
		multiUser: multiUser,
	}
}

// RegisterRoutes mounts all API routes on the given mux.
func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v1/rollups/today", h.wrap(h.handleRollupsToday))
	mux.HandleFunc("GET /api/v1/rollups/range", h.wrap(h.handleRollupsRange))
	mux.HandleFunc("GET /api/v1/meals", h.wrap(h.handleMealsList))
	mux.HandleFunc("GET /api/v1/meals/{mealID}", h.wrap(h.handleMealDetail))
	mux.HandleFunc("POST /api/v1/meals/{mealID}/items/{itemID}/correct", h.wrap(h.handleCorrectItem))
	mux.HandleFunc("POST /api/v1/meals/{mealID}/items", h.wrap(h.handleAddItem))
	mux.HandleFunc("DELETE /api/v1/meals/{mealID}/items/{itemID}", h.wrap(h.handleDeleteItem))
	mux.HandleFunc("POST /api/v1/meals/log", h.wrap(h.handleLogMeal))
	mux.HandleFunc("GET /api/v1/targets", h.wrap(h.handleGetTargets))
	mux.HandleFunc("PUT /api/v1/targets", h.wrap(h.handleSetTargets))
}

// wrap applies auth middleware and JSON content-type headers to a handler.
func (h *Handler) wrap(next func(w http.ResponseWriter, r *http.Request, userID string)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		userID, err := h.authenticate(r)
		if err != nil {
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
			return
		}
		next(w, r, userID)
	}
}

// authenticate extracts and validates the user identity from the request.
// In multi-user mode, validates Bearer tokens against api_tokens.
// In single-user mode, checks API_AUTH_TOKEN if configured, otherwise
// allows localhost requests without auth.
func (h *Handler) authenticate(r *http.Request) (string, error) {
	if h.multiUser {
		return h.authenticateToken(r)
	}
	// Single-user: check static token if configured.
	if h.authToken != "" {
		return h.authenticateStaticToken(r)
	}
	// No auth configured: use "default" user for localhost, or any Origin.
	return "default", nil
}

func (h *Handler) authenticateStaticToken(r *http.Request) (string, error) {
	token := bearerToken(r)
	if token == "" {
		return "", types.ErrNotFound // "token required"
	}
	if token != h.authToken {
		return "", types.ErrNotFound
	}
	return "default", nil
}

func (h *Handler) authenticateToken(r *http.Request) (string, error) {
	token := bearerToken(r)
	if token == "" {
		return "", types.ErrNotFound
	}
	userID, err := h.store.ValidateToken(r.Context(), token)
	if err != nil {
		return "", err
	}
	return userID, nil
}

func bearerToken(r *http.Request) string {
	auth := r.Header.Get("Authorization")
	if len(auth) < 7 || auth[:7] != "Bearer " {
		return ""
	}
	return auth[7:]
}

// ---------------------------------------------------------------------------
// Handlers
// ---------------------------------------------------------------------------

func (h *Handler) handleRollupsToday(w http.ResponseWriter, r *http.Request, userID string) {
	today := time.Now().In(h.loc).Format("2006-01-02")
	rollup, err := h.store.GetRollup(r.Context(), userID, today)
	if err != nil {
		h.writeErr(w, err)
		return
	}
	json.NewEncoder(w).Encode(rollup)
}

func (h *Handler) handleRollupsRange(w http.ResponseWriter, r *http.Request, userID string) {
	start := r.URL.Query().Get("start")
	end := r.URL.Query().Get("end")
	if start == "" || end == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "start and end query params required (YYYY-MM-DD)"})
		return
	}
	rollups, err := h.store.GetRollups(r.Context(), userID, start, end)
	if err != nil {
		h.writeErr(w, err)
		return
	}
	if rollups == nil {
		rollups = []types.DailyRollup{}
	}
	json.NewEncoder(w).Encode(rollups)
}

func (h *Handler) handleMealsList(w http.ResponseWriter, r *http.Request, userID string) {
	limit := 10
	if s := r.URL.Query().Get("limit"); s != "" {
		if n, err := strconv.Atoi(s); err == nil && n > 0 && n <= 100 {
			limit = n
		}
	}
	meals, err := h.store.RecentMeals(r.Context(), userID, limit)
	if err != nil {
		h.writeErr(w, err)
		return
	}
	if meals == nil {
		meals = []types.Meal{}
	}
	json.NewEncoder(w).Encode(meals)
}

func (h *Handler) handleMealDetail(w http.ResponseWriter, r *http.Request, userID string) {
	mealID := r.PathValue("mealID")
	meal, err := h.store.GetMeal(r.Context(), mealID)
	if err != nil {
		h.writeErr(w, err)
		return
	}
	// Only return meals belonging to the authenticated user.
	if meal.UserID != userID {
		h.writeErr(w, types.ErrNotFound)
		return
	}
	json.NewEncoder(w).Encode(meal)
}

func (h *Handler) handleCorrectItem(w http.ResponseWriter, r *http.Request, userID string) {
	mealID := r.PathValue("mealID")
	itemIDStr := r.PathValue("itemID")

	itemIndex, err := strconv.Atoi(itemIDStr)
	if err != nil || itemIndex < 0 {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "itemID must be a non-negative integer index"})
		return
	}

	var corrected types.ResolvedItem
	if err := json.NewDecoder(r.Body).Decode(&corrected); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "invalid JSON body: " + err.Error()})
		return
	}

	if err := h.store.CorrectMealItem(r.Context(), userID, mealID, itemIndex, corrected); err != nil {
		h.writeErr(w, err)
		return
	}

	// Return the updated meal.
	meal, err := h.store.GetMeal(r.Context(), mealID)
	if err != nil {
		h.writeErr(w, err)
		return
	}
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(meal)
}

func (h *Handler) handleAddItem(w http.ResponseWriter, r *http.Request, userID string) {
	mealID := r.PathValue("mealID")

	var item types.ResolvedItem
	if err := json.NewDecoder(r.Body).Decode(&item); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "invalid JSON body: " + err.Error()})
		return
	}
	if err := h.store.AddMealItem(r.Context(), userID, mealID, item); err != nil {
		h.writeErr(w, err)
		return
	}
	h.returnMeal(w, r, mealID, userID)
}

func (h *Handler) handleDeleteItem(w http.ResponseWriter, r *http.Request, userID string) {
	mealID := r.PathValue("mealID")
	itemIndex, err := strconv.Atoi(r.PathValue("itemID"))
	if err != nil || itemIndex < 0 {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "itemID must be a non-negative integer index"})
		return
	}
	if err := h.store.DeleteMealItem(r.Context(), userID, mealID, itemIndex); err != nil {
		h.writeErr(w, err)
		return
	}
	h.returnMeal(w, r, mealID, userID)
}

// returnMeal writes the meal as JSON, enforcing user ownership.
func (h *Handler) returnMeal(w http.ResponseWriter, r *http.Request, mealID, userID string) {
	meal, err := h.store.GetMeal(r.Context(), mealID)
	if err != nil {
		h.writeErr(w, err)
		return
	}
	if meal.UserID != userID {
		h.writeErr(w, types.ErrNotFound)
		return
	}
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(meal)
}

func (h *Handler) handleGetTargets(w http.ResponseWriter, r *http.Request, userID string) {
	dt, err := h.store.GetTargets(r.Context(), userID)
	if err != nil {
		h.writeErr(w, err)
		return
	}
	json.NewEncoder(w).Encode(dt)
}

func (h *Handler) handleSetTargets(w http.ResponseWriter, r *http.Request, userID string) {
	var body types.Macros
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "invalid JSON body: " + err.Error()})
		return
	}
	dt := types.DailyTargets{UserID: userID, Targets: body}
	if err := h.store.SetTargets(r.Context(), dt); err != nil {
		h.writeErr(w, err)
		return
	}
	// Reflect immediately on the dashboard, which reads targets from the rollup.
	today := time.Now().In(h.loc).Format("2006-01-02")
	if err := h.store.UpdateRollupTargets(r.Context(), userID, today, body); err != nil {
		h.writeErr(w, err)
		return
	}
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(dt)
}

func (h *Handler) handleLogMeal(w http.ResponseWriter, r *http.Request, userID string) {
	var body struct {
		Text string `json:"text"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "invalid JSON body: " + err.Error()})
		return
	}
	if body.Text == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "text field is required"})
		return
	}

	msg := types.InboundMessage{
		UserID: userID,
		Text:   body.Text,
		Kind:   types.MessageText,
	}
	if err := h.logger.Handle(r.Context(), msg); err != nil {
		h.writeErr(w, err)
		return
	}
	w.WriteHeader(http.StatusAccepted)
	json.NewEncoder(w).Encode(map[string]string{"status": "accepted"})
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func (h *Handler) writeErr(w http.ResponseWriter, err error) {
	switch {
	case err.Error() == types.ErrNotFound.Error():
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{"error": "not found"})
	default:
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
	}
}
