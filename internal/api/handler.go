// Package api implements the REST API for the DietDaemon dashboard. It uses
// the Go standard library net/http and http.ServeMux for routing. All endpoints
// return JSON and are gated behind ENABLE_DASHBOARD=true.
package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gsaraiva2109/dietdaemon/core/types"
)

// MealStore is the subset of the store the API needs.
type MealStore interface {
	// Meals & rollups.
	GetMeal(ctx context.Context, mealID string) (types.Meal, error)
	RecentMeals(ctx context.Context, userID string, limit int) ([]types.Meal, error)
	GetMealsInRange(ctx context.Context, userID, startDate, endDate string) ([]types.Meal, error)
	GetRollup(ctx context.Context, userID, localDate string) (types.DailyRollup, error)
	GetRollups(ctx context.Context, userID, startDate, endDate string) ([]types.DailyRollup, error)
	CorrectMealItem(ctx context.Context, userID string, mealID string, itemIndex int, corrected types.ResolvedItem) error
	AddMealItem(ctx context.Context, userID, mealID string, item types.ResolvedItem) error
	DeleteMealItem(ctx context.Context, userID, mealID string, itemIndex int) error
	SaveMeal(ctx context.Context, m types.Meal) error
	LatestMealTime(ctx context.Context, userID string) (string, error)

	// Targets.
	GetTargets(ctx context.Context, userID string) (types.DailyTargets, error)
	SetTargets(ctx context.Context, t types.DailyTargets) error
	UpdateRollupTargets(ctx context.Context, userID, localDate string, t types.Macros) error

	// Users & auth.
	GetUser(ctx context.Context, userID string) (types.User, error)
	ValidateToken(ctx context.Context, token string) (string, error)

	// Food discovery.
	ListFoods(ctx context.Context, userID, source string, limit, offset int) ([]types.FoodDetail, error)
	SearchFoods(ctx context.Context, userID, query string) ([]types.FoodDetail, error)
	FrequentFoods(ctx context.Context, userID string, limit int) ([]types.FoodDetail, error)
	GetFoodDetail(ctx context.Context, userID, foodID string) (types.FoodDetail, error)
	AddFoodAlias(ctx context.Context, userID, foodID, alias string) error
	DeleteFoodAlias(ctx context.Context, userID, foodID, alias string) error

	// Meal templates.
	SaveTemplate(ctx context.Context, t types.MealTemplate) error
	GetTemplates(ctx context.Context, userID string) ([]types.MealTemplate, error)
	GetTemplate(ctx context.Context, templateID string) (types.MealTemplate, error)
	DeleteTemplate(ctx context.Context, userID, templateID string) error
	LogTemplateUse(ctx context.Context, tl types.TemplateLog) error

	// Body tracking — weight.
	ListWeight(ctx context.Context, userID string, days int) ([]types.WeightEntry, error)
	LogWeight(ctx context.Context, w types.WeightEntry) error
	DeleteWeight(ctx context.Context, userID, entryID string) error
	WeightTrend(ctx context.Context, userID string, days int) ([]types.WeightTrend, error)

	// Body tracking — measurements.
	ListMeasurements(ctx context.Context, userID string, days int) ([]types.MeasurementEntry, error)
	LogMeasurement(ctx context.Context, m types.MeasurementEntry) error
	DeleteMeasurement(ctx context.Context, userID, entryID string) error

	// Body tracking — photos.
	ListPhotoMetadata(ctx context.Context, userID string) ([]types.ProgressPhoto, error)
	GetPhotoData(ctx context.Context, photoID string) (types.ProgressPhoto, error)
	UploadPhoto(ctx context.Context, p types.ProgressPhoto) error
	DeletePhoto(ctx context.Context, userID, photoID string) error

	// Profile & goals.
	GetProfile(ctx context.Context, userID string) (types.UserProfile, error)
	UpsertProfile(ctx context.Context, p types.UserProfile) error
}

// MealLogger submits raw text through the parsing pipeline, and can also directly
// log a fully-resolved meal (used by template logging and meal duplication).
// Satisfied by the pipeline.Engine.
type MealLogger interface {
	Handle(ctx context.Context, msg types.InboundMessage) error
	LogMeal(ctx context.Context, meal types.Meal) error
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
	// Existing.
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

	// Phase 1 — Meals Latest.
	mux.HandleFunc("GET /api/v1/meals/latest", h.wrap(h.handleMealsLatest))

	// Phase 2 — Food Discovery.
	mux.HandleFunc("GET /api/v1/foods", h.wrap(h.handleListFoods))
	mux.HandleFunc("GET /api/v1/foods/search", h.wrap(h.handleSearchFoods))
	mux.HandleFunc("GET /api/v1/foods/frequent", h.wrap(h.handleFrequentFoods))
	mux.HandleFunc("GET /api/v1/foods/{foodID}", h.wrap(h.handleGetFood))
	mux.HandleFunc("POST /api/v1/foods/{foodID}/aliases", h.wrap(h.handleAddAlias))
	mux.HandleFunc("DELETE /api/v1/foods/{foodID}/aliases/{alias}", h.wrap(h.handleDeleteAlias))

	// Phase 3 — Meal Templates.
	mux.HandleFunc("GET /api/v1/templates", h.wrap(h.handleListTemplates))
	mux.HandleFunc("POST /api/v1/templates", h.wrap(h.handleCreateTemplate))
	mux.HandleFunc("GET /api/v1/templates/{id}", h.wrap(h.handleGetTemplate))
	mux.HandleFunc("DELETE /api/v1/templates/{id}", h.wrap(h.handleDeleteTemplate))
	mux.HandleFunc("POST /api/v1/templates/{id}/log", h.wrap(h.handleLogTemplate))
	mux.HandleFunc("POST /api/v1/meals/{mealID}/duplicate", h.wrap(h.handleDuplicateMeal))

	// Phase 4 — Body Tracking: Weight.
	mux.HandleFunc("GET /api/v1/body/weight", h.wrap(h.handleListWeight))
	mux.HandleFunc("POST /api/v1/body/weight", h.wrap(h.handleLogWeight))
	mux.HandleFunc("GET /api/v1/body/weight/trend", h.wrap(h.handleWeightTrend))
	mux.HandleFunc("DELETE /api/v1/body/weight/{id}", h.wrap(h.handleDeleteWeight))

	// Phase 4 — Body Tracking: Measurements.
	mux.HandleFunc("GET /api/v1/body/measurements", h.wrap(h.handleListMeasurements))
	mux.HandleFunc("POST /api/v1/body/measurements", h.wrap(h.handleLogMeasurements))
	mux.HandleFunc("DELETE /api/v1/body/measurements/{id}", h.wrap(h.handleDeleteMeasurement))

	// Phase 4 — Body Tracking: Photos.
	mux.HandleFunc("GET /api/v1/body/photos", h.wrap(h.handleListPhotos))
	mux.HandleFunc("GET /api/v1/body/photos/{id}/data", h.wrap(h.handlePhotoData))
	mux.HandleFunc("POST /api/v1/body/photos", h.wrap(h.handleUploadPhoto))
	mux.HandleFunc("DELETE /api/v1/body/photos/{id}", h.wrap(h.handleDeletePhoto))

	// Phase 4 — Body Tracking: Summary.
	mux.HandleFunc("GET /api/v1/body/summary", h.wrap(h.handleBodySummary))

	// Phase 5 — Goals & Profile.
	mux.HandleFunc("GET /api/v1/profile", h.wrap(h.handleGetProfile))
	mux.HandleFunc("PUT /api/v1/profile", h.wrap(h.handleUpsertProfile))
	mux.HandleFunc("GET /api/v1/tdee", h.wrap(h.handleCalculateTDEE))
	mux.HandleFunc("GET /api/v1/goals/suggestions", h.wrap(h.handleGoalSuggestions))

	// Phase 6 — Export.
	mux.HandleFunc("GET /api/v1/export/meals", h.wrap(h.handleExportMeals))
	mux.HandleFunc("GET /api/v1/export/rollups", h.wrap(h.handleExportRollups))
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
// Phase 1 — Meals Latest
// ---------------------------------------------------------------------------

func (h *Handler) handleMealsLatest(w http.ResponseWriter, r *http.Request, userID string) {
	latest, err := h.store.LatestMealTime(r.Context(), userID)
	if err != nil && !errors.Is(err, types.ErrNotFound) {
		h.writeErr(w, err)
		return
	}
	json.NewEncoder(w).Encode(map[string]string{"latest": latest})
}

// ---------------------------------------------------------------------------
// Phase 2 — Food Discovery
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
	json.NewEncoder(w).Encode(foods)
}

func (h *Handler) handleSearchFoods(w http.ResponseWriter, r *http.Request, userID string) {
	q := r.URL.Query().Get("q")
	if q == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "q query param is required"})
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
	json.NewEncoder(w).Encode(foods)
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
	json.NewEncoder(w).Encode(foods)
}

func (h *Handler) handleGetFood(w http.ResponseWriter, r *http.Request, userID string) {
	foodID := r.PathValue("foodID")
	fd, err := h.store.GetFoodDetail(r.Context(), userID, foodID)
	if err != nil {
		h.writeErr(w, err)
		return
	}
	if fd.Aliases == nil {
		fd.Aliases = []types.FoodAlias{}
	}
	json.NewEncoder(w).Encode(fd)
}

func (h *Handler) handleAddAlias(w http.ResponseWriter, r *http.Request, userID string) {
	foodID := r.PathValue("foodID")
	var body struct {
		Alias string `json:"alias"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Alias == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "alias field is required"})
		return
	}
	if err := h.store.AddFoodAlias(r.Context(), userID, foodID, body.Alias); err != nil {
		h.writeErr(w, err)
		return
	}
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{"status": "created"})
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

// ---------------------------------------------------------------------------
// Phase 3 — Meal Templates
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
	json.NewEncoder(w).Encode(templates)
}

func (h *Handler) handleCreateTemplate(w http.ResponseWriter, r *http.Request, userID string) {
	var body struct {
		Name  string               `json:"name"`
		Items []types.ResolvedItem `json:"items"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "invalid JSON body: " + err.Error()})
		return
	}
	if body.Name == "" || len(body.Items) == 0 {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "name and items are required"})
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
	json.NewEncoder(w).Encode(t)
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
	json.NewEncoder(w).Encode(t)
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
	json.NewEncoder(w).Encode(map[string]string{"status": "logged", "meal_id": meal.ID})
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
	json.NewEncoder(w).Encode(map[string]string{"status": "duplicated", "meal_id": newMeal.ID})
}

// ---------------------------------------------------------------------------
// Phase 4 — Body Tracking
// ---------------------------------------------------------------------------

// --- Weight ---

func (h *Handler) handleListWeight(w http.ResponseWriter, r *http.Request, userID string) {
	days := 30
	if s := r.URL.Query().Get("days"); s != "" {
		if n, err := strconv.Atoi(s); err == nil && n > 0 && n <= 365 {
			days = n
		}
	}
	entries, err := h.store.ListWeight(r.Context(), userID, days)
	if err != nil {
		h.writeErr(w, err)
		return
	}
	if entries == nil {
		entries = []types.WeightEntry{}
	}
	json.NewEncoder(w).Encode(entries)
}

func (h *Handler) handleLogWeight(w http.ResponseWriter, r *http.Request, userID string) {
	var body struct {
		Date     string  `json:"date"`
		WeightKg float64 `json:"weight_kg"`
		Note     string  `json:"note"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "invalid JSON body: " + err.Error()})
		return
	}
	if body.WeightKg <= 0 {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "weight_kg must be positive"})
		return
	}
	entry := types.WeightEntry{
		ID:        newHandlerID(),
		UserID:    userID,
		Date:      body.Date,
		WeightKg:  body.WeightKg,
		Note:      body.Note,
		CreatedAt: time.Now().UTC(),
	}
	if err := h.store.LogWeight(r.Context(), entry); err != nil {
		h.writeErr(w, err)
		return
	}
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(entry)
}

func (h *Handler) handleWeightTrend(w http.ResponseWriter, r *http.Request, userID string) {
	days := 30
	if s := r.URL.Query().Get("days"); s != "" {
		if n, err := strconv.Atoi(s); err == nil && n > 0 && n <= 365 {
			days = n
		}
	}
	trend, err := h.store.WeightTrend(r.Context(), userID, days)
	if err != nil {
		h.writeErr(w, err)
		return
	}
	if trend == nil {
		trend = []types.WeightTrend{}
	}
	json.NewEncoder(w).Encode(trend)
}

func (h *Handler) handleDeleteWeight(w http.ResponseWriter, r *http.Request, userID string) {
	entryID := r.PathValue("id")
	if err := h.store.DeleteWeight(r.Context(), userID, entryID); err != nil {
		h.writeErr(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// --- Measurements ---

func (h *Handler) handleListMeasurements(w http.ResponseWriter, r *http.Request, userID string) {
	days := 30
	if s := r.URL.Query().Get("days"); s != "" {
		if n, err := strconv.Atoi(s); err == nil && n > 0 && n <= 365 {
			days = n
		}
	}
	entries, err := h.store.ListMeasurements(r.Context(), userID, days)
	if err != nil {
		h.writeErr(w, err)
		return
	}
	if entries == nil {
		entries = []types.MeasurementEntry{}
	}
	json.NewEncoder(w).Encode(entries)
}

func (h *Handler) handleLogMeasurements(w http.ResponseWriter, r *http.Request, userID string) {
	var body types.MeasurementEntry
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "invalid JSON body: " + err.Error()})
		return
	}
	body.ID = newHandlerID()
	body.UserID = userID
	body.CreatedAt = time.Now().UTC()
	if err := h.store.LogMeasurement(r.Context(), body); err != nil {
		h.writeErr(w, err)
		return
	}
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(body)
}

func (h *Handler) handleDeleteMeasurement(w http.ResponseWriter, r *http.Request, userID string) {
	entryID := r.PathValue("id")
	if err := h.store.DeleteMeasurement(r.Context(), userID, entryID); err != nil {
		h.writeErr(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// --- Photos ---

func (h *Handler) handleListPhotos(w http.ResponseWriter, r *http.Request, userID string) {
	photos, err := h.store.ListPhotoMetadata(r.Context(), userID)
	if err != nil {
		h.writeErr(w, err)
		return
	}
	if photos == nil {
		photos = []types.ProgressPhoto{}
	}
	json.NewEncoder(w).Encode(photos)
}

func (h *Handler) handlePhotoData(w http.ResponseWriter, r *http.Request, userID string) {
	photoID := r.PathValue("id")
	photo, err := h.store.GetPhotoData(r.Context(), photoID)
	if err != nil {
		h.writeErr(w, err)
		return
	}
	if photo.UserID != userID {
		h.writeErr(w, types.ErrNotFound)
		return
	}
	w.Header().Set("Content-Type", photo.MimeType)
	w.Header().Set("Cache-Control", "private, max-age=86400")
	w.Write(photo.Data)
}

func (h *Handler) handleUploadPhoto(w http.ResponseWriter, r *http.Request, userID string) {
	r.Body = http.MaxBytesReader(w, r.Body, 5<<20)
	// #nosec G120 — MaxBytesReader above bounds the body before ParseMultipartForm.
	if err := r.ParseMultipartForm(5 << 20); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "file too large (max 5 MB)"})
		return
	}

	file, _, err := r.FormFile("file")
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "file field required"})
		return
	}
	defer file.Close()

	data, err := io.ReadAll(io.LimitReader(file, 5<<20))
	if err != nil {
		h.writeErr(w, err)
		return
	}

	view := r.FormValue("view")
	if view == "" {
		view = "front"
	}
	date := r.FormValue("date")
	if date == "" {
		date = time.Now().In(h.loc).Format("2006-01-02")
	}

	// Detect mime type from first 512 bytes.
	mimeType := http.DetectContentType(data)

	photo := types.ProgressPhoto{
		ID:        newHandlerID(),
		UserID:    userID,
		Date:      date,
		View:      view,
		MimeType:  mimeType,
		Data:      data,
		CreatedAt: time.Now().UTC(),
	}
	if err := h.store.UploadPhoto(r.Context(), photo); err != nil {
		h.writeErr(w, err)
		return
	}
	// Clear data before JSON response.
	photo.Data = nil
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(photo)
}

func (h *Handler) handleDeletePhoto(w http.ResponseWriter, r *http.Request, userID string) {
	photoID := r.PathValue("id")
	if err := h.store.DeletePhoto(r.Context(), userID, photoID); err != nil {
		h.writeErr(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// --- Body Summary ---

func (h *Handler) handleBodySummary(w http.ResponseWriter, r *http.Request, userID string) {
	// Load all weight entries to compute summary.
	entries, err := h.store.ListWeight(r.Context(), userID, 365)
	if err != nil {
		h.writeErr(w, err)
		return
	}

	summary := types.BodyCompositionSummary{}
	if len(entries) == 0 {
		json.NewEncoder(w).Encode(summary)
		return
	}

	current := entries[len(entries)-1]
	start := entries[0]
	summary.CurrentWeightKg = current.WeightKg
	summary.StartWeightKg = start.WeightKg
	summary.ChangeKg = current.WeightKg - start.WeightKg

	// Compute trend from last 14 days.
	trend, err := h.store.WeightTrend(r.Context(), userID, 14)
	if err == nil && len(trend) > 0 {
		summary.LatestTrendPoint = &trend[len(trend)-1]
		if len(trend) >= 2 {
			first := trend[0].RollingAvg
			last := trend[len(trend)-1].RollingAvg
			diff := last - first
			switch {
			case diff > 0.5:
				summary.TrendDirection = "up"
			case diff < -0.5:
				summary.TrendDirection = "down"
			default:
				summary.TrendDirection = "stable"
			}
		}
	}
	if summary.TrendDirection == "" {
		summary.TrendDirection = "stable"
	}

	json.NewEncoder(w).Encode(summary)
}

// ---------------------------------------------------------------------------
// Phase 5 — Goals & Profile
// ---------------------------------------------------------------------------

func (h *Handler) handleGetProfile(w http.ResponseWriter, r *http.Request, userID string) {
	profile, err := h.store.GetProfile(r.Context(), userID)
	if err != nil && !errors.Is(err, types.ErrNotFound) {
		h.writeErr(w, err)
		return
	}
	if errors.Is(err, types.ErrNotFound) {
		profile = types.UserProfile{UserID: userID, Onboarded: false}
	}
	json.NewEncoder(w).Encode(profile)
}

func (h *Handler) handleUpsertProfile(w http.ResponseWriter, r *http.Request, userID string) {
	var body types.UserProfile
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "invalid JSON body: " + err.Error()})
		return
	}
	now := time.Now().UTC()
	body.UserID = userID
	body.UpdatedAt = now
	if body.CreatedAt.IsZero() {
		body.CreatedAt = now
	}
	if err := h.store.UpsertProfile(r.Context(), body); err != nil {
		h.writeErr(w, err)
		return
	}
	json.NewEncoder(w).Encode(body)
}

func (h *Handler) handleCalculateTDEE(w http.ResponseWriter, r *http.Request, userID string) {
	q := r.URL.Query()
	weightKg, _ := strconv.ParseFloat(q.Get("weight_kg"), 64)
	heightCm, _ := strconv.ParseFloat(q.Get("height_cm"), 64)
	age, _ := strconv.Atoi(q.Get("age"))
	gender := q.Get("gender")
	activity := q.Get("activity")

	if weightKg <= 0 || heightCm <= 0 || age <= 0 || gender == "" || activity == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "weight_kg, height_cm, age, gender, and activity query params are required",
		})
		return
	}

	params := types.TDEEParams{
		WeightKg:      weightKg,
		HeightCm:      heightCm,
		Age:           age,
		Gender:        gender,
		ActivityLevel: activity,
	}
	result := calculateTDEE(params)
	w.Header().Set("Cache-Control", "private, max-age=300")
	json.NewEncoder(w).Encode(result)
}

func (h *Handler) handleGoalSuggestions(w http.ResponseWriter, r *http.Request, userID string) {
	profile, err := h.store.GetProfile(r.Context(), userID)
	if err != nil {
		// No profile yet.
		json.NewEncoder(w).Encode(types.GoalSuggestion{
			Message: "Complete your profile to get personalized goal suggestions.",
		})
		return
	}

	// Get recent rollups for average intake.
	endDate := time.Now().In(h.loc).Format("2006-01-02")
	startDate := time.Now().In(h.loc).AddDate(0, 0, -7).Format("2006-01-02")
	rollups, _ := h.store.GetRollups(r.Context(), userID, startDate, endDate)

	var avgKcal float64
	for _, r := range rollups {
		avgKcal += r.Consumed.Calories
	}
	if len(rollups) > 0 {
		avgKcal /= float64(len(rollups))
	}

	// Get weight trend.
	trend, _ := h.store.WeightTrend(r.Context(), userID, 14)
	var currentLossKg float64
	if len(trend) >= 2 {
		currentLossKg = trend[0].RollingAvg - trend[len(trend)-1].RollingAvg
	}

	// Compute recommended kcal using TDEE.
	now := time.Now()
	birthDate := profile.BirthDate
	age := 30
	if birthDate != "" {
		if parsed, err := time.Parse("2006-01-02", birthDate); err == nil {
			age = int(now.Sub(parsed).Hours() / 8766)
		}
	}

	// Get current weight for TDEE calc.
	var currentWeight float64 = 70
	weights, _ := h.store.ListWeight(r.Context(), userID, 30)
	if len(weights) > 0 {
		currentWeight = weights[len(weights)-1].WeightKg
	}

	params := types.TDEEParams{
		WeightKg:      currentWeight,
		HeightCm:      profile.HeightCm,
		Age:           age,
		Gender:        profile.Gender,
		ActivityLevel: profile.ActivityLevel,
	}
	tdee := calculateTDEE(params)

	var recommendedKcal float64 = tdee.MaintainCal
	switch profile.Goal {
	case "lose":
		recommendedKcal = tdee.CutCal
	case "gain":
		recommendedKcal = tdee.BulkCal
	}

	targetLossKg := currentWeight - profile.TargetWeightKg

	message := "Keep going! Track your meals consistently to reach your goals."
	if profile.Goal == "lose" {
		if currentLossKg > 0 {
			message = fmt.Sprintf("You're losing ~%.1f kg/week. Keep it up!", currentLossKg)
		} else {
			message = "Weight is stable. Try reducing intake slightly to start losing."
		}
	} else if profile.Goal == "gain" {
		message = fmt.Sprintf("Aim for %.0f kcal/day to support muscle gain.", recommendedKcal)
	}

	json.NewEncoder(w).Encode(types.GoalSuggestion{
		CurrentIntakeKcal: avgKcal,
		RecommendedKcal:   recommendedKcal,
		CurrentLossKg:     currentLossKg,
		TargetLossKg:      targetLossKg,
		Message:           message,
	})
}

// ---------------------------------------------------------------------------
// Phase 6 — Export
// ---------------------------------------------------------------------------

func (h *Handler) handleExportMeals(w http.ResponseWriter, r *http.Request, userID string) {
	format := r.URL.Query().Get("format")
	start := r.URL.Query().Get("start")
	end := r.URL.Query().Get("end")

	if start == "" || end == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "start and end query params required (YYYY-MM-DD)"})
		return
	}

	meals, err := h.store.GetMealsInRange(r.Context(), userID, start, end)
	if err != nil {
		h.writeErr(w, err)
		return
	}

	switch format {
	case "csv":
		h.writeMealsCSV(w, meals)
	default:
		// JSON (default).
		w.Header().Set("Content-Disposition", "attachment; filename=meals.json")
		if meals == nil {
			meals = []types.Meal{}
		}
		json.NewEncoder(w).Encode(meals)
	}
}

func (h *Handler) handleExportRollups(w http.ResponseWriter, r *http.Request, userID string) {
	format := r.URL.Query().Get("format")
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

	switch format {
	case "csv":
		h.writeRollupsCSV(w, rollups)
	default:
		w.Header().Set("Content-Disposition", "attachment; filename=rollups.json")
		if rollups == nil {
			rollups = []types.DailyRollup{}
		}
		json.NewEncoder(w).Encode(rollups)
	}
}

func (h *Handler) writeMealsCSV(w http.ResponseWriter, meals []types.Meal) {
	w.Header().Set("Content-Type", "text/csv")
	w.Header().Set("Content-Disposition", "attachment; filename=meals.csv")
	fmt.Fprintln(w, "id,date,raw_text,kcal,protein,carbs,fat,fiber")
	for _, m := range meals {
		total := m.Total()
		escaped := strings.ReplaceAll(m.RawText, `"`, `""`)
		fmt.Fprintf(w, "%s,%s,\"%s\",%.0f,%.1f,%.1f,%.1f,%.1f\n",
			m.ID, m.At.Format("2006-01-02"), escaped,
			total.Calories, total.Protein, total.Carbs, total.Fat, total.Fiber,
		)
	}
}

func (h *Handler) writeRollupsCSV(w http.ResponseWriter, rollups []types.DailyRollup) {
	w.Header().Set("Content-Type", "text/csv")
	w.Header().Set("Content-Disposition", "attachment; filename=rollups.csv")
	fmt.Fprintln(w, "date,consumed_kcal,consumed_protein,consumed_carbs,consumed_fat,consumed_fiber,target_kcal,target_protein,target_carbs,target_fat,target_fiber")
	for _, r := range rollups {
		fmt.Fprintf(w, "%s,%.0f,%.1f,%.1f,%.1f,%.1f,%.0f,%.1f,%.1f,%.1f,%.1f\n",
			r.Date,
			r.Consumed.Calories, r.Consumed.Protein, r.Consumed.Carbs, r.Consumed.Fat, r.Consumed.Fiber,
			r.Targets.Calories, r.Targets.Protein, r.Targets.Carbs, r.Targets.Fat, r.Targets.Fiber,
		)
	}
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// newHandlerID returns a short pseudo-unique ID for API-created entities.
func newHandlerID() string {
	return strconv.FormatInt(time.Now().UnixNano(), 36)
}

// calculateTDEE computes BMR, TDEE, and macro splits using Mifflin-St Jeor.
func calculateTDEE(p types.TDEEParams) types.TDEEResult {
	var bmr float64
	switch p.Gender {
	case "male":
		bmr = 10*p.WeightKg + 6.25*p.HeightCm - 5*float64(p.Age) + 5
	case "female":
		bmr = 10*p.WeightKg + 6.25*p.HeightCm - 5*float64(p.Age) - 161
	default:
		// Average male/female for "other".
		bmrMale := 10*p.WeightKg + 6.25*p.HeightCm - 5*float64(p.Age) + 5
		bmrFemale := 10*p.WeightKg + 6.25*p.HeightCm - 5*float64(p.Age) - 161
		bmr = (bmrMale + bmrFemale) / 2
	}

	multipliers := map[string]float64{
		"sedentary": 1.2, "light": 1.375, "moderate": 1.55,
		"active": 1.725, "very_active": 1.9,
	}
	actMult, ok := multipliers[p.ActivityLevel]
	if !ok {
		actMult = 1.2
	}
	tdee := bmr * actMult

	return types.TDEEResult{
		BMR:         bmr,
		TDEE:        tdee,
		CutCal:      tdee - 500,
		MaintainCal: tdee,
		BulkCal:     tdee + 500,
		Protein:     p.WeightKg * 2.2,
		Fat:         tdee * 0.25 / 9,
		Carbs:       (tdee - (p.WeightKg*2.2*4 + tdee*0.25)) / 4,
	}
}

func (h *Handler) writeErr(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, types.ErrNotFound) || errors.Is(err, types.ErrNoMatch):
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{"error": "not found"})
	default:
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
	}
}
