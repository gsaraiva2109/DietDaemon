package api

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gsaraiva2109/dietdaemon/core/types"
)

// --- fakes ---

type fakeMealStore struct {
	meals       map[string]types.Meal
	recentMeals []types.Meal
	rollup      types.DailyRollup
	rollups     []types.DailyRollup
	user        types.User
	tokens      map[string]string // token → userID
	correctErr  error
	getMealErr  error
	rollupErr   error
	rollupsErr  error
	recentErr   error
	validateErr error
	getUserErr  error
	targets     types.DailyTargets
	targetsErr  error
	addErr      error
	deleteErr   error

	// Phase 1.
	latestMealTime    string
	latestMealTimeErr error

	// Phase 2.
	foodList       []types.FoodDetail
	foodListErr    error
	foodDetail     types.FoodDetail
	foodDetailErr  error
	addAliasErr    error
	deleteAliasErr error

	// Phase 3.
	templates         []types.MealTemplate
	templatesErr      error
	template          types.MealTemplate
	templateErr       error
	saveTemplateErr   error
	deleteTemplateErr error
	logTemplateErr    error
	saveMealErr       error

	// Phase 4.
	weights              []types.WeightEntry
	weightsErr           error
	logWeightErr         error
	deleteWeightErr      error
	weightTrend          []types.WeightTrend
	weightTrendErr       error
	measurements         []types.MeasurementEntry
	measurementsErr      error
	logMeasurementErr    error
	deleteMeasurementErr error
	photoMetadata        []types.ProgressPhoto
	photoMetadataErr     error
	photoData            types.ProgressPhoto
	photoDataErr         error
	uploadPhotoErr       error
	deletePhotoErr       error

	// Phase 5.
	profile          types.UserProfile
	profileErr       error
	upsertProfileErr error

	// Phase 4 / shared.
	mealsInRange    []types.Meal
	mealsInRangeErr error
}

func newFakeMealStore() *fakeMealStore {
	return &fakeMealStore{
		meals:  map[string]types.Meal{},
		tokens: map[string]string{},
	}
}

func (s *fakeMealStore) GetMeal(_ context.Context, mealID string) (types.Meal, error) {
	if s.getMealErr != nil {
		return types.Meal{}, s.getMealErr
	}
	if m, ok := s.meals[mealID]; ok {
		return m, nil
	}
	return types.Meal{}, types.ErrNotFound
}
func (s *fakeMealStore) RecentMeals(_ context.Context, _ string, _ int) ([]types.Meal, error) {
	if s.recentErr != nil {
		return nil, s.recentErr
	}
	return s.recentMeals, nil
}
func (s *fakeMealStore) GetRollup(_ context.Context, _, _ string) (types.DailyRollup, error) {
	if s.rollupErr != nil {
		return types.DailyRollup{}, s.rollupErr
	}
	return s.rollup, nil
}
func (s *fakeMealStore) GetRollups(_ context.Context, _, _, _ string) ([]types.DailyRollup, error) {
	if s.rollupsErr != nil {
		return nil, s.rollupsErr
	}
	return s.rollups, nil
}
func (s *fakeMealStore) CorrectMealItem(_ context.Context, _ string, _ string, _ int, _ types.ResolvedItem) error {
	return s.correctErr
}
func (s *fakeMealStore) AddMealItem(_ context.Context, _, mealID string, item types.ResolvedItem) error {
	if s.addErr != nil {
		return s.addErr
	}
	m, ok := s.meals[mealID]
	if !ok {
		return types.ErrNotFound
	}
	m.Items = append(m.Items, item)
	s.meals[mealID] = m
	return nil
}
func (s *fakeMealStore) DeleteMealItem(_ context.Context, _, mealID string, idx int) error {
	if s.deleteErr != nil {
		return s.deleteErr
	}
	m, ok := s.meals[mealID]
	if !ok {
		return types.ErrNotFound
	}
	if idx < 0 || idx >= len(m.Items) {
		return types.ErrNotFound
	}
	m.Items = append(m.Items[:idx], m.Items[idx+1:]...)
	s.meals[mealID] = m
	return nil
}
func (s *fakeMealStore) GetTargets(_ context.Context, _ string) (types.DailyTargets, error) {
	if s.targetsErr != nil {
		return types.DailyTargets{}, s.targetsErr
	}
	return s.targets, nil
}
func (s *fakeMealStore) SetTargets(_ context.Context, t types.DailyTargets) error {
	if s.targetsErr != nil {
		return s.targetsErr
	}
	s.targets = t
	return nil
}
func (s *fakeMealStore) UpdateRollupTargets(_ context.Context, _, _ string, t types.Macros) error {
	s.rollup.Targets = t
	return nil
}
func (s *fakeMealStore) GetUser(_ context.Context, _ string) (types.User, error) {
	if s.getUserErr != nil {
		return types.User{}, s.getUserErr
	}
	return s.user, nil
}
func (s *fakeMealStore) ValidateToken(_ context.Context, token string) (string, error) {
	if s.validateErr != nil {
		return "", s.validateErr
	}
	if uid, ok := s.tokens[token]; ok {
		return uid, nil
	}
	return "", types.ErrNotFound
}

// Phase 1.
func (s *fakeMealStore) LatestMealTime(_ context.Context, _ string) (string, error) {
	return s.latestMealTime, s.latestMealTimeErr
}

// Phase 2.
func (s *fakeMealStore) ListFoods(_ context.Context, _, _ string, _, _ int) ([]types.FoodDetail, error) {
	return s.foodList, s.foodListErr
}
func (s *fakeMealStore) SearchFoods(_ context.Context, _, _ string) ([]types.FoodDetail, error) {
	return s.foodList, s.foodListErr
}
func (s *fakeMealStore) FrequentFoods(_ context.Context, _ string, _ int) ([]types.FoodDetail, error) {
	return s.foodList, s.foodListErr
}
func (s *fakeMealStore) GetFoodDetail(_ context.Context, _, _ string) (types.FoodDetail, error) {
	return s.foodDetail, s.foodDetailErr
}
func (s *fakeMealStore) AddFoodAlias(_ context.Context, _, _, _ string) error {
	return s.addAliasErr
}
func (s *fakeMealStore) DeleteFoodAlias(_ context.Context, _, _, _ string) error {
	return s.deleteAliasErr
}

// Phase 3.
func (s *fakeMealStore) SaveTemplate(_ context.Context, _ types.MealTemplate) error {
	return s.saveTemplateErr
}
func (s *fakeMealStore) GetTemplates(_ context.Context, _ string) ([]types.MealTemplate, error) {
	return s.templates, s.templatesErr
}
func (s *fakeMealStore) GetTemplate(_ context.Context, _ string) (types.MealTemplate, error) {
	return s.template, s.templateErr
}
func (s *fakeMealStore) DeleteTemplate(_ context.Context, _, _ string) error {
	return s.deleteTemplateErr
}
func (s *fakeMealStore) LogTemplateUse(_ context.Context, _ types.TemplateLog) error {
	return s.logTemplateErr
}
func (s *fakeMealStore) SaveMeal(_ context.Context, _ types.Meal) error {
	return s.saveMealErr
}

// Phase 4 — Weight.
func (s *fakeMealStore) ListWeight(_ context.Context, _ string, _ int) ([]types.WeightEntry, error) {
	return s.weights, s.weightsErr
}
func (s *fakeMealStore) LogWeight(_ context.Context, _ types.WeightEntry) error {
	return s.logWeightErr
}
func (s *fakeMealStore) DeleteWeight(_ context.Context, _, _ string) error {
	return s.deleteWeightErr
}
func (s *fakeMealStore) WeightTrend(_ context.Context, _ string, _ int) ([]types.WeightTrend, error) {
	return s.weightTrend, s.weightTrendErr
}

// Phase 4 — Measurements.
func (s *fakeMealStore) ListMeasurements(_ context.Context, _ string, _ int) ([]types.MeasurementEntry, error) {
	return s.measurements, s.measurementsErr
}
func (s *fakeMealStore) LogMeasurement(_ context.Context, _ types.MeasurementEntry) error {
	return s.logMeasurementErr
}
func (s *fakeMealStore) DeleteMeasurement(_ context.Context, _, _ string) error {
	return s.deleteMeasurementErr
}

// Phase 4 — Photos.
func (s *fakeMealStore) ListPhotoMetadata(_ context.Context, _ string) ([]types.ProgressPhoto, error) {
	return s.photoMetadata, s.photoMetadataErr
}
func (s *fakeMealStore) GetPhotoData(_ context.Context, _ string) (types.ProgressPhoto, error) {
	return s.photoData, s.photoDataErr
}
func (s *fakeMealStore) UploadPhoto(_ context.Context, _ types.ProgressPhoto) error {
	return s.uploadPhotoErr
}
func (s *fakeMealStore) DeletePhoto(_ context.Context, _, _ string) error {
	return s.deletePhotoErr
}

// Phase 4 / shared.
func (s *fakeMealStore) GetMealsInRange(_ context.Context, _, _, _ string) ([]types.Meal, error) {
	return s.mealsInRange, s.mealsInRangeErr
}

// Phase 5.
func (s *fakeMealStore) GetProfile(_ context.Context, _ string) (types.UserProfile, error) {
	return s.profile, s.profileErr
}
func (s *fakeMealStore) UpsertProfile(_ context.Context, _ types.UserProfile) error {
	return s.upsertProfileErr
}

type fakeMealLogger struct {
	lastMsg  types.InboundMessage
	lastMeal types.Meal
	err      error
}

func (l *fakeMealLogger) Handle(_ context.Context, msg types.InboundMessage) error {
	l.lastMsg = msg
	return l.err
}

func (l *fakeMealLogger) LogMeal(_ context.Context, meal types.Meal) error {
	l.lastMeal = meal
	return l.err
}

// --- helpers ---

func newHandler(store MealStore, logger MealLogger, authToken string, multiUser bool) *Handler {
	return New(store, logger, time.UTC, authToken, multiUser)
}

func doRequest(h *Handler, method, path string, body any, headers map[string]string) *httptest.ResponseRecorder {
	var rBody []byte
	if body != nil {
		rBody, _ = json.Marshal(body)
	}
	req := httptest.NewRequest(method, path, bytes.NewReader(rBody))
	req.Header.Set("Content-Type", "application/json")
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	rec := httptest.NewRecorder()
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)
	mux.ServeHTTP(rec, req)
	return rec
}

func decodeJSON[T any](t *testing.T, rec *httptest.ResponseRecorder) T {
	t.Helper()
	var v T
	if err := json.NewDecoder(rec.Body).Decode(&v); err != nil {
		t.Fatalf("decode JSON: %v (body=%q)", err, rec.Body.String())
	}
	return v
}

// --- auth tests ---

func TestAuthNoTokenLocalhost(t *testing.T) {
	h := newHandler(newFakeMealStore(), &fakeMealLogger{}, "", false)

	rec := doRequest(h, "GET", "/api/v1/rollups/today", nil, nil)
	if rec.Code != http.StatusOK {
		t.Errorf("no-auth single-user expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestAuthStaticTokenValid(t *testing.T) {
	store := newFakeMealStore()
	h := newHandler(store, &fakeMealLogger{}, "secret-token", false)

	rec := doRequest(h, "GET", "/api/v1/rollups/today", nil, map[string]string{
		"Authorization": "Bearer secret-token",
	})
	if rec.Code != http.StatusOK {
		t.Errorf("valid static token expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestAuthStaticTokenInvalid(t *testing.T) {
	store := newFakeMealStore()
	h := newHandler(store, &fakeMealLogger{}, "secret-token", false)

	rec := doRequest(h, "GET", "/api/v1/rollups/today", nil, map[string]string{
		"Authorization": "Bearer wrong",
	})
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("invalid static token expected 401, got %d", rec.Code)
	}
}

func TestAuthStaticTokenMissing(t *testing.T) {
	store := newFakeMealStore()
	h := newHandler(store, &fakeMealLogger{}, "secret-token", false)

	// No Authorization header at all → token required when API_AUTH_TOKEN is set.
	rec := doRequest(h, "GET", "/api/v1/rollups/today", nil, nil)
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("missing token with API_AUTH_TOKEN set expected 401, got %d", rec.Code)
	}
}

func TestAuthMultiUserTokenValid(t *testing.T) {
	store := newFakeMealStore()
	store.tokens["multi-token"] = "user-42"
	h := newHandler(store, &fakeMealLogger{}, "", true) // multiUser=true

	rec := doRequest(h, "GET", "/api/v1/rollups/today", nil, map[string]string{
		"Authorization": "Bearer multi-token",
	})
	if rec.Code != http.StatusOK {
		t.Errorf("valid multi-user token expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestAuthMultiUserTokenInvalid(t *testing.T) {
	store := newFakeMealStore()
	h := newHandler(store, &fakeMealLogger{}, "", true)

	rec := doRequest(h, "GET", "/api/v1/rollups/today", nil, map[string]string{
		"Authorization": "Bearer bad-token",
	})
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("invalid multi-user token expected 401, got %d", rec.Code)
	}
}

func TestAuthMultiUserNoToken(t *testing.T) {
	store := newFakeMealStore()
	h := newHandler(store, &fakeMealLogger{}, "", true)

	rec := doRequest(h, "GET", "/api/v1/rollups/today", nil, nil)
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("missing token in multi-user mode expected 401, got %d", rec.Code)
	}
}

func TestBearerTokenEdgeCases(t *testing.T) {
	// Shorter than "Bearer "
	if got := bearerToken(&http.Request{Header: http.Header{"Authorization": {"Bear x"}}}); got != "" {
		t.Errorf("short auth header = %q, want empty", got)
	}
	// No Bearer prefix.
	if got := bearerToken(&http.Request{Header: http.Header{"Authorization": {"Basic xyz"}}}); got != "" {
		t.Errorf("Basic auth = %q, want empty", got)
	}
}

// --- handler tests ---

func TestHandleRollupsToday(t *testing.T) {
	store := newFakeMealStore()
	store.rollup = types.DailyRollup{
		UserID:   "default",
		Date:     "2026-06-17",
		Consumed: types.Macros{Calories: 2100, Protein: 140},
		Targets:  types.Macros{Calories: 3000, Protein: 180},
	}
	h := newHandler(store, &fakeMealLogger{}, "", false)

	rec := doRequest(h, "GET", "/api/v1/rollups/today", nil, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	rollup := decodeJSON[types.DailyRollup](t, rec)
	if rollup.Consumed.Calories != 2100 {
		t.Errorf("calories = %v, want 2100", rollup.Consumed.Calories)
	}
}

func TestHandleRollupsRange(t *testing.T) {
	store := newFakeMealStore()
	store.rollups = []types.DailyRollup{
		{UserID: "default", Date: "2026-06-15", Consumed: types.Macros{Calories: 2000}},
		{UserID: "default", Date: "2026-06-16", Consumed: types.Macros{Calories: 2200}},
	}
	h := newHandler(store, &fakeMealLogger{}, "", false)

	rec := doRequest(h, "GET", "/api/v1/rollups/range?start=2026-06-15&end=2026-06-17", nil, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	rollups := decodeJSON[[]types.DailyRollup](t, rec)
	if len(rollups) != 2 {
		t.Errorf("rollups count = %d, want 2", len(rollups))
	}
}

func TestHandleRollupsRangeMissingParams(t *testing.T) {
	store := newFakeMealStore()
	h := newHandler(store, &fakeMealLogger{}, "", false)

	rec := doRequest(h, "GET", "/api/v1/rollups/range", nil, nil)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("missing params expected 400, got %d", rec.Code)
	}
}

func TestHandleRollupsRangeNullReturn(t *testing.T) {
	store := newFakeMealStore()
	// rollups is nil (not initialized) — handler should return [] not null.
	h := newHandler(store, &fakeMealLogger{}, "", false)

	rec := doRequest(h, "GET", "/api/v1/rollups/range?start=2026-06-15&end=2026-06-17", nil, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "[") {
		t.Errorf("expected JSON array, got %s", rec.Body.String())
	}
}

func TestHandleMealsList(t *testing.T) {
	store := newFakeMealStore()
	store.recentMeals = []types.Meal{
		{ID: "m1", UserID: "default", RawText: "200g chicken"},
		{ID: "m2", UserID: "default", RawText: "2 eggs"},
	}
	h := newHandler(store, &fakeMealLogger{}, "", false)

	rec := doRequest(h, "GET", "/api/v1/meals?limit=5", nil, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	meals := decodeJSON[[]types.Meal](t, rec)
	if len(meals) != 2 {
		t.Errorf("meals count = %d, want 2", len(meals))
	}
}

func TestHandleMealsListDefaultLimit(t *testing.T) {
	store := newFakeMealStore()
	store.recentMeals = []types.Meal{}
	h := newHandler(store, &fakeMealLogger{}, "", false)

	rec := doRequest(h, "GET", "/api/v1/meals", nil, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	meals := decodeJSON[[]types.Meal](t, rec)
	if meals == nil {
		t.Error("expected empty array, got null")
	}
}

func TestHandleMealDetail(t *testing.T) {
	store := newFakeMealStore()
	store.meals["m1"] = types.Meal{
		ID:      "m1",
		UserID:  "default",
		RawText: "200g chicken",
		Items:   []types.ResolvedItem{},
	}
	h := newHandler(store, &fakeMealLogger{}, "", false)

	rec := doRequest(h, "GET", "/api/v1/meals/m1", nil, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	meal := decodeJSON[types.Meal](t, rec)
	if meal.ID != "m1" {
		t.Errorf("meal ID = %q, want m1", meal.ID)
	}
}

func TestHandleMealDetailWrongUser(t *testing.T) {
	store := newFakeMealStore()
	store.meals["m1"] = types.Meal{
		ID:     "m1",
		UserID: "other-user",
	}
	h := newHandler(store, &fakeMealLogger{}, "", false)

	rec := doRequest(h, "GET", "/api/v1/meals/m1", nil, nil)
	if rec.Code != http.StatusNotFound {
		t.Errorf("cross-user meal access expected 404, got %d", rec.Code)
	}
}

func TestHandleMealDetailNotFound(t *testing.T) {
	store := newFakeMealStore()
	h := newHandler(store, &fakeMealLogger{}, "", false)

	rec := doRequest(h, "GET", "/api/v1/meals/nonexistent", nil, nil)
	if rec.Code != http.StatusNotFound {
		t.Errorf("missing meal expected 404, got %d", rec.Code)
	}
}

func TestHandleCorrectItem(t *testing.T) {
	store := newFakeMealStore()
	store.meals["m1"] = types.Meal{
		ID:      "m1",
		UserID:  "default",
		RawText: "200g chicken",
		Items: []types.ResolvedItem{
			{Parsed: types.ParsedItem{RawPhrase: "chicken"}},
		},
	}
	h := newHandler(store, &fakeMealLogger{}, "", false)

	body := types.ResolvedItem{
		Parsed: types.ParsedItem{RawPhrase: "chicken", NormalizedGrams: 200},
		Match:  types.FoodMatch{FoodID: "chicken", Name: "Chicken Breast"},
		Macros: types.Macros{Calories: 330, Protein: 62},
	}
	rec := doRequest(h, "POST", "/api/v1/meals/m1/items/0/correct", body, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestHandleCorrectItemBadIndex(t *testing.T) {
	store := newFakeMealStore()
	h := newHandler(store, &fakeMealLogger{}, "", false)

	rec := doRequest(h, "POST", "/api/v1/meals/m1/items/abc/correct", types.ResolvedItem{}, nil)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("bad index expected 400, got %d", rec.Code)
	}
}

func TestHandleCorrectItemNegativeIndex(t *testing.T) {
	store := newFakeMealStore()
	h := newHandler(store, &fakeMealLogger{}, "", false)

	rec := doRequest(h, "POST", "/api/v1/meals/m1/items/-1/correct", types.ResolvedItem{}, nil)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("negative index expected 400, got %d", rec.Code)
	}
}

func TestHandleLogMeal(t *testing.T) {
	logger := &fakeMealLogger{}
	store := newFakeMealStore()
	h := newHandler(store, logger, "", false)

	body := map[string]string{"text": "200g chicken, 2 eggs"}
	rec := doRequest(h, "POST", "/api/v1/meals/log", body, nil)
	if rec.Code != http.StatusAccepted {
		t.Fatalf("expected 202, got %d: %s", rec.Code, rec.Body.String())
	}
	if logger.lastMsg.Text != "200g chicken, 2 eggs" {
		t.Errorf("logged text = %q, want %q", logger.lastMsg.Text, "200g chicken, 2 eggs")
	}
	if logger.lastMsg.UserID != "default" {
		t.Errorf("logged userID = %q, want default", logger.lastMsg.UserID)
	}
}

func TestHandleLogMealEmptyText(t *testing.T) {
	logger := &fakeMealLogger{}
	store := newFakeMealStore()
	h := newHandler(store, logger, "", false)

	body := map[string]string{"text": ""}
	rec := doRequest(h, "POST", "/api/v1/meals/log", body, nil)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("empty text expected 400, got %d", rec.Code)
	}
}

func TestHandleLogMealInvalidJSON(t *testing.T) {
	logger := &fakeMealLogger{}
	store := newFakeMealStore()
	h := newHandler(store, logger, "", false)

	req := httptest.NewRequest("POST", "/api/v1/meals/log", strings.NewReader("not json"))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("invalid JSON expected 400, got %d", rec.Code)
	}
}

func TestHandleLogMealLoggerError(t *testing.T) {
	logger := &fakeMealLogger{err: errors.New("pipeline busy")}
	store := newFakeMealStore()
	h := newHandler(store, logger, "", false)

	body := map[string]string{"text": "200g chicken"}
	rec := doRequest(h, "POST", "/api/v1/meals/log", body, nil)
	if rec.Code != http.StatusInternalServerError {
		t.Errorf("logger error expected 500, got %d", rec.Code)
	}
}

func TestHandleRollupsTodayNotFound(t *testing.T) {
	store := newFakeMealStore()
	store.rollupErr = types.ErrNotFound
	h := newHandler(store, &fakeMealLogger{}, "", false)

	rec := doRequest(h, "GET", "/api/v1/rollups/today", nil, nil)
	if rec.Code != http.StatusNotFound {
		t.Errorf("expected 404 for not-found rollup, got %d", rec.Code)
	}
}

func TestHandlerWriteErrGeneric(t *testing.T) {
	store := newFakeMealStore()
	store.rollupErr = errors.New("db connection lost")
	h := newHandler(store, &fakeMealLogger{}, "", false)

	rec := doRequest(h, "GET", "/api/v1/rollups/today", nil, nil)
	if rec.Code != http.StatusInternalServerError {
		t.Errorf("generic error expected 500, got %d", rec.Code)
	}
}

func TestMealsListNullReturn(t *testing.T) {
	store := newFakeMealStore()
	// recentMeals is nil (not initialized slice).
	h := newHandler(store, &fakeMealLogger{}, "", false)

	rec := doRequest(h, "GET", "/api/v1/meals", nil, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "[") {
		t.Errorf("expected JSON array, got %s", rec.Body.String())
	}
}

func TestAuthMultiUserValidateTokenError(t *testing.T) {
	store := newFakeMealStore()
	store.validateErr = errors.New("db error")
	h := newHandler(store, &fakeMealLogger{}, "", true)

	rec := doRequest(h, "GET", "/api/v1/rollups/today", nil, map[string]string{
		"Authorization": "Bearer any-token",
	})
	// Auth errors always return 401 to avoid leaking internal state.
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("token validation error expected 401, got %d", rec.Code)
	}
}

func TestAuthHeaderEdgeCases(t *testing.T) {
	store := newFakeMealStore()
	h := newHandler(store, &fakeMealLogger{}, "tok", false)

	// Authorization header with exactly "Bearer " and nothing else.
	rec := doRequest(h, "GET", "/api/v1/rollups/today", nil, map[string]string{
		"Authorization": "Bearer ",
	})
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("empty Bearer token expected 401, got %d", rec.Code)
	}

	// No "Bearer" prefix at all.
	rec = doRequest(h, "GET", "/api/v1/rollups/today", nil, map[string]string{
		"Authorization": "Basic dGVzdDp0ZXN0",
	})
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("Basic auth expected 401, got %d", rec.Code)
	}
}

// --- targets + item add/delete tests ---

func TestSetTargets(t *testing.T) {
	store := newFakeMealStore()
	h := newHandler(store, &fakeMealLogger{}, "", false)

	body := types.Macros{Calories: 3000, Protein: 180, Carbs: 360, Fat: 90, Fiber: 38}
	rec := doRequest(h, "PUT", "/api/v1/targets", body, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("PUT targets expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	if store.targets.Targets.Protein != 180 {
		t.Errorf("targets not persisted, got %+v", store.targets)
	}
	if store.rollup.Targets.Calories != 3000 {
		t.Errorf("rollup targets not refreshed, got %+v", store.rollup.Targets)
	}
}

func TestAddMealItem(t *testing.T) {
	store := newFakeMealStore()
	store.meals["m1"] = types.Meal{ID: "m1", UserID: "default", Items: []types.ResolvedItem{}}
	h := newHandler(store, &fakeMealLogger{}, "", false)

	item := types.ResolvedItem{
		Parsed: types.ParsedItem{RawPhrase: "banana", NormalizedGrams: 120},
		Match:  types.FoodMatch{Name: "Banana", Source: "taco"},
		Macros: types.Macros{Calories: 107, Protein: 1, Carbs: 27},
	}
	rec := doRequest(h, "POST", "/api/v1/meals/m1/items", item, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("POST item expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	got := decodeJSON[types.Meal](t, rec)
	if len(got.Items) != 1 || got.Items[0].Match.Name != "Banana" {
		t.Errorf("item not added, got %+v", got.Items)
	}
}

func TestDeleteMealItem(t *testing.T) {
	store := newFakeMealStore()
	store.meals["m1"] = types.Meal{ID: "m1", UserID: "default", Items: []types.ResolvedItem{
		{Match: types.FoodMatch{Name: "A"}},
		{Match: types.FoodMatch{Name: "B"}},
	}}
	h := newHandler(store, &fakeMealLogger{}, "", false)

	rec := doRequest(h, "DELETE", "/api/v1/meals/m1/items/0", nil, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("DELETE item expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	got := decodeJSON[types.Meal](t, rec)
	if len(got.Items) != 1 || got.Items[0].Match.Name != "B" {
		t.Errorf("wrong item deleted, got %+v", got.Items)
	}
}

func TestDeleteMealItemForbiddenOtherUser(t *testing.T) {
	store := newFakeMealStore()
	store.meals["m1"] = types.Meal{ID: "m1", UserID: "someone-else", Items: []types.ResolvedItem{{}}}
	store.tokens["tok"] = "me"
	h := newHandler(store, &fakeMealLogger{}, "", true)

	rec := doRequest(h, "DELETE", "/api/v1/meals/m1/items/0", nil, map[string]string{"Authorization": "Bearer tok"})
	if rec.Code != http.StatusNotFound {
		t.Errorf("deleting another user's item expected 404, got %d", rec.Code)
	}
}

// ---------------------------------------------------------------------------
// Phase 1 — Meals Latest
// ---------------------------------------------------------------------------

func TestMealsLatest(t *testing.T) {
	store := newFakeMealStore()
	store.latestMealTime = "2026-06-17T12:00:00Z"
	h := newHandler(store, &fakeMealLogger{}, "", false)

	rec := doRequest(h, "GET", "/api/v1/meals/latest", nil, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	got := decodeJSON[map[string]string](t, rec)
	if got["latest"] != "2026-06-17T12:00:00Z" {
		t.Errorf("latest = %q, want 2026-06-17T12:00:00Z", got["latest"])
	}
}

func TestMealsLatestEmpty(t *testing.T) {
	store := newFakeMealStore()
	store.latestMealTimeErr = types.ErrNotFound
	h := newHandler(store, &fakeMealLogger{}, "", false)

	rec := doRequest(h, "GET", "/api/v1/meals/latest", nil, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	got := decodeJSON[map[string]string](t, rec)
	if got["latest"] != "" {
		t.Errorf("latest = %q, want empty", got["latest"])
	}
}

// ---------------------------------------------------------------------------
// Phase 2 — Food Discovery
// ---------------------------------------------------------------------------

func TestListFoods(t *testing.T) {
	store := newFakeMealStore()
	store.foodList = []types.FoodDetail{
		{FoodID: "f1", Name: "Chicken", Source: "food_library"},
	}
	h := newHandler(store, &fakeMealLogger{}, "", false)

	rec := doRequest(h, "GET", "/api/v1/foods", nil, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	foods := decodeJSON[[]types.FoodDetail](t, rec)
	if len(foods) != 1 || foods[0].Name != "Chicken" {
		t.Errorf("unexpected foods: %+v", foods)
	}
}

func TestListFoodsNullReturn(t *testing.T) {
	store := newFakeMealStore()
	// foodList is nil.
	h := newHandler(store, &fakeMealLogger{}, "", false)

	rec := doRequest(h, "GET", "/api/v1/foods", nil, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "[") {
		t.Errorf("expected JSON array, got %s", rec.Body.String())
	}
}

func TestSearchFoodsMissingQ(t *testing.T) {
	store := newFakeMealStore()
	h := newHandler(store, &fakeMealLogger{}, "", false)

	rec := doRequest(h, "GET", "/api/v1/foods/search", nil, nil)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for missing q, got %d", rec.Code)
	}
}

func TestSearchFoods(t *testing.T) {
	store := newFakeMealStore()
	store.foodList = []types.FoodDetail{{FoodID: "f1", Name: "Banana"}}
	h := newHandler(store, &fakeMealLogger{}, "", false)

	rec := doRequest(h, "GET", "/api/v1/foods/search?q=banana", nil, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	foods := decodeJSON[[]types.FoodDetail](t, rec)
	if len(foods) != 1 {
		t.Errorf("expected 1 result, got %d", len(foods))
	}
}

func TestFrequentFoods(t *testing.T) {
	store := newFakeMealStore()
	store.foodList = []types.FoodDetail{{FoodID: "f1", Name: "Rice"}, {FoodID: "f2", Name: "Beans"}}
	h := newHandler(store, &fakeMealLogger{}, "", false)

	rec := doRequest(h, "GET", "/api/v1/foods/frequent?limit=5", nil, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	foods := decodeJSON[[]types.FoodDetail](t, rec)
	if len(foods) != 2 {
		t.Errorf("expected 2, got %d", len(foods))
	}
}

func TestGetFoodNotFound(t *testing.T) {
	store := newFakeMealStore()
	store.foodDetailErr = types.ErrNotFound
	h := newHandler(store, &fakeMealLogger{}, "", false)

	rec := doRequest(h, "GET", "/api/v1/foods/missing", nil, nil)
	if rec.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", rec.Code)
	}
}

func TestGetFood(t *testing.T) {
	store := newFakeMealStore()
	store.foodDetail = types.FoodDetail{
		FoodID:  "f1",
		Name:    "Egg",
		Aliases: []types.FoodAlias{{FoodID: "f1", Normalized: "egg"}},
	}
	h := newHandler(store, &fakeMealLogger{}, "", false)

	rec := doRequest(h, "GET", "/api/v1/foods/f1", nil, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	got := decodeJSON[types.FoodDetail](t, rec)
	if got.Name != "Egg" || len(got.Aliases) != 1 {
		t.Errorf("unexpected food detail: %+v", got)
	}
}

func TestAddAlias(t *testing.T) {
	store := newFakeMealStore()
	h := newHandler(store, &fakeMealLogger{}, "", false)

	rec := doRequest(h, "POST", "/api/v1/foods/f1/aliases", map[string]string{"alias": "ovo"}, nil)
	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestAddAliasMissing(t *testing.T) {
	store := newFakeMealStore()
	h := newHandler(store, &fakeMealLogger{}, "", false)

	rec := doRequest(h, "POST", "/api/v1/foods/f1/aliases", map[string]string{"alias": ""}, nil)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rec.Code)
	}
}

func TestDeleteAlias(t *testing.T) {
	store := newFakeMealStore()
	h := newHandler(store, &fakeMealLogger{}, "", false)

	rec := doRequest(h, "DELETE", "/api/v1/foods/f1/aliases/egg", nil, nil)
	if rec.Code != http.StatusNoContent {
		t.Errorf("expected 204, got %d", rec.Code)
	}
}

// ---------------------------------------------------------------------------
// Phase 3 — Meal Templates
// ---------------------------------------------------------------------------

func TestListTemplates(t *testing.T) {
	store := newFakeMealStore()
	store.templates = []types.MealTemplate{{ID: "t1", Name: "Morning"}}
	h := newHandler(store, &fakeMealLogger{}, "", false)

	rec := doRequest(h, "GET", "/api/v1/templates", nil, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	templates := decodeJSON[[]types.MealTemplate](t, rec)
	if len(templates) != 1 || templates[0].Name != "Morning" {
		t.Errorf("unexpected templates: %+v", templates)
	}
}

func TestCreateTemplate(t *testing.T) {
	store := newFakeMealStore()
	logger := &fakeMealLogger{}
	h := newHandler(store, logger, "", false)

	body := map[string]any{
		"name":  "My Template",
		"items": []types.ResolvedItem{{Match: types.FoodMatch{Name: "Egg"}}},
	}
	rec := doRequest(h, "POST", "/api/v1/templates", body, nil)
	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestCreateTemplateValidation(t *testing.T) {
	store := newFakeMealStore()
	h := newHandler(store, &fakeMealLogger{}, "", false)

	rec := doRequest(h, "POST", "/api/v1/templates", map[string]string{"name": ""}, nil)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("empty name expected 400, got %d", rec.Code)
	}
}

func TestGetTemplateNotFound(t *testing.T) {
	store := newFakeMealStore()
	store.templateErr = types.ErrNotFound
	h := newHandler(store, &fakeMealLogger{}, "", false)

	rec := doRequest(h, "GET", "/api/v1/templates/missing", nil, nil)
	if rec.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", rec.Code)
	}
}

func TestDeleteTemplate(t *testing.T) {
	store := newFakeMealStore()
	h := newHandler(store, &fakeMealLogger{}, "", false)

	rec := doRequest(h, "DELETE", "/api/v1/templates/t1", nil, nil)
	if rec.Code != http.StatusNoContent {
		t.Errorf("expected 204, got %d", rec.Code)
	}
}

func TestLogTemplate(t *testing.T) {
	store := newFakeMealStore()
	store.template = types.MealTemplate{
		ID:     "t1",
		UserID: "default",
		Name:   "Morning",
		Items:  []types.ResolvedItem{{Match: types.FoodMatch{Name: "Egg"}}},
	}
	logger := &fakeMealLogger{}
	h := newHandler(store, logger, "", false)

	rec := doRequest(h, "POST", "/api/v1/templates/t1/log", nil, nil)
	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rec.Code, rec.Body.String())
	}
	if logger.lastMeal.UserID != "default" {
		t.Errorf("LogMeal not called")
	}
}

func TestDuplicateMeal(t *testing.T) {
	store := newFakeMealStore()
	store.meals["m1"] = types.Meal{
		ID: "m1", UserID: "default", RawText: "200g chicken",
		Items: []types.ResolvedItem{{Match: types.FoodMatch{Name: "Chicken"}}},
	}
	logger := &fakeMealLogger{}
	h := newHandler(store, logger, "", false)

	rec := doRequest(h, "POST", "/api/v1/meals/m1/duplicate", nil, nil)
	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rec.Code, rec.Body.String())
	}
	if logger.lastMeal.Items == nil {
		t.Errorf("LogMeal not called with items")
	}
}

// ---------------------------------------------------------------------------
// Phase 4 — Body Tracking
// ---------------------------------------------------------------------------

func TestListWeight(t *testing.T) {
	store := newFakeMealStore()
	store.weights = []types.WeightEntry{{ID: "w1", WeightKg: 80.5, Date: "2026-06-17"}}
	h := newHandler(store, &fakeMealLogger{}, "", false)

	rec := doRequest(h, "GET", "/api/v1/body/weight?days=30", nil, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	weights := decodeJSON[[]types.WeightEntry](t, rec)
	if len(weights) != 1 || weights[0].WeightKg != 80.5 {
		t.Errorf("unexpected weights: %+v", weights)
	}
}

func TestLogWeightValidation(t *testing.T) {
	store := newFakeMealStore()
	h := newHandler(store, &fakeMealLogger{}, "", false)

	rec := doRequest(h, "POST", "/api/v1/body/weight", map[string]any{"weight_kg": 0}, nil)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("zero weight expected 400, got %d", rec.Code)
	}
}

func TestLogWeight(t *testing.T) {
	store := newFakeMealStore()
	h := newHandler(store, &fakeMealLogger{}, "", false)

	rec := doRequest(h, "POST", "/api/v1/body/weight", map[string]any{
		"weight_kg": 80.5, "date": "2026-06-17", "note": "morning",
	}, nil)
	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestWeightTrend(t *testing.T) {
	store := newFakeMealStore()
	store.weightTrend = []types.WeightTrend{
		{Date: "2026-06-15", WeightKg: 80, RollingAvg: 80},
		{Date: "2026-06-16", WeightKg: 79.5, RollingAvg: 79.75},
	}
	h := newHandler(store, &fakeMealLogger{}, "", false)

	rec := doRequest(h, "GET", "/api/v1/body/weight/trend?days=14", nil, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	trend := decodeJSON[[]types.WeightTrend](t, rec)
	if len(trend) != 2 {
		t.Errorf("expected 2, got %d", len(trend))
	}
}

func TestDeleteWeight(t *testing.T) {
	store := newFakeMealStore()
	h := newHandler(store, &fakeMealLogger{}, "", false)

	rec := doRequest(h, "DELETE", "/api/v1/body/weight/w1", nil, nil)
	if rec.Code != http.StatusNoContent {
		t.Errorf("expected 204, got %d", rec.Code)
	}
}

func TestListMeasurements(t *testing.T) {
	store := newFakeMealStore()
	store.measurements = []types.MeasurementEntry{{ID: "m1", WaistCm: 90, Date: "2026-06-17"}}
	h := newHandler(store, &fakeMealLogger{}, "", false)

	rec := doRequest(h, "GET", "/api/v1/body/measurements?days=30", nil, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	got := decodeJSON[[]types.MeasurementEntry](t, rec)
	if len(got) != 1 || got[0].WaistCm != 90 {
		t.Errorf("unexpected measurements: %+v", got)
	}
}

func TestLogMeasurements(t *testing.T) {
	store := newFakeMealStore()
	h := newHandler(store, &fakeMealLogger{}, "", false)

	body := types.MeasurementEntry{Date: "2026-06-17", WaistCm: 90, HipsCm: 100}
	rec := doRequest(h, "POST", "/api/v1/body/measurements", body, nil)
	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestDeleteMeasurement(t *testing.T) {
	store := newFakeMealStore()
	h := newHandler(store, &fakeMealLogger{}, "", false)

	rec := doRequest(h, "DELETE", "/api/v1/body/measurements/m1", nil, nil)
	if rec.Code != http.StatusNoContent {
		t.Errorf("expected 204, got %d", rec.Code)
	}
}

func TestListPhotos(t *testing.T) {
	store := newFakeMealStore()
	store.photoMetadata = []types.ProgressPhoto{{ID: "p1", View: "front"}}
	h := newHandler(store, &fakeMealLogger{}, "", false)

	rec := doRequest(h, "GET", "/api/v1/body/photos", nil, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	photos := decodeJSON[[]types.ProgressPhoto](t, rec)
	if len(photos) != 1 {
		t.Errorf("expected 1, got %d", len(photos))
	}
}

func TestPhotoDataNotFound(t *testing.T) {
	store := newFakeMealStore()
	store.photoDataErr = types.ErrNotFound
	h := newHandler(store, &fakeMealLogger{}, "", false)

	rec := doRequest(h, "GET", "/api/v1/body/photos/missing/data", nil, nil)
	if rec.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", rec.Code)
	}
}

func TestDeletePhoto(t *testing.T) {
	store := newFakeMealStore()
	h := newHandler(store, &fakeMealLogger{}, "", false)

	rec := doRequest(h, "DELETE", "/api/v1/body/photos/p1", nil, nil)
	if rec.Code != http.StatusNoContent {
		t.Errorf("expected 204, got %d", rec.Code)
	}
}

func TestBodySummaryEmpty(t *testing.T) {
	store := newFakeMealStore()
	h := newHandler(store, &fakeMealLogger{}, "", false)

	rec := doRequest(h, "GET", "/api/v1/body/summary", nil, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
}

// ---------------------------------------------------------------------------
// Phase 5 — Goals & Profile
// ---------------------------------------------------------------------------

func TestGetProfileNotOnboarded(t *testing.T) {
	store := newFakeMealStore()
	store.profileErr = types.ErrNotFound
	h := newHandler(store, &fakeMealLogger{}, "", false)

	rec := doRequest(h, "GET", "/api/v1/profile", nil, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	profile := decodeJSON[types.UserProfile](t, rec)
	if profile.Onboarded {
		t.Errorf("expected onboarded=false for missing profile")
	}
}

func TestGetProfile(t *testing.T) {
	store := newFakeMealStore()
	store.profile = types.UserProfile{UserID: "default", HeightCm: 175, Onboarded: true}
	h := newHandler(store, &fakeMealLogger{}, "", false)

	rec := doRequest(h, "GET", "/api/v1/profile", nil, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	profile := decodeJSON[types.UserProfile](t, rec)
	if profile.HeightCm != 175 || !profile.Onboarded {
		t.Errorf("unexpected profile: %+v", profile)
	}
}

func TestUpsertProfile(t *testing.T) {
	store := newFakeMealStore()
	h := newHandler(store, &fakeMealLogger{}, "", false)

	body := types.UserProfile{HeightCm: 180, Gender: "male", ActivityLevel: "moderate"}
	rec := doRequest(h, "PUT", "/api/v1/profile", body, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestCalculateTDEEMissingParams(t *testing.T) {
	store := newFakeMealStore()
	h := newHandler(store, &fakeMealLogger{}, "", false)

	rec := doRequest(h, "GET", "/api/v1/tdee", nil, nil)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for missing params, got %d", rec.Code)
	}
}

func TestCalculateTDEE(t *testing.T) {
	store := newFakeMealStore()
	h := newHandler(store, &fakeMealLogger{}, "", false)

	rec := doRequest(h, "GET", "/api/v1/tdee?weight_kg=80&height_cm=175&age=30&gender=male&activity=moderate", nil, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	result := decodeJSON[types.TDEEResult](t, rec)
	if result.BMR <= 0 || result.TDEE <= 0 {
		t.Errorf("unexpected TDEE result: %+v", result)
	}
}

func TestGoalSuggestionsNoProfile(t *testing.T) {
	store := newFakeMealStore()
	store.profileErr = types.ErrNotFound
	h := newHandler(store, &fakeMealLogger{}, "", false)

	rec := doRequest(h, "GET", "/api/v1/goals/suggestions", nil, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
}

// ---------------------------------------------------------------------------
// Phase 6 — Export
// ---------------------------------------------------------------------------

func TestExportMealsJSON(t *testing.T) {
	store := newFakeMealStore()
	store.mealsInRange = []types.Meal{
		{ID: "m1", UserID: "default", RawText: "200g chicken",
			Items: []types.ResolvedItem{{Macros: types.Macros{Calories: 330, Protein: 62}}}},
	}
	h := newHandler(store, &fakeMealLogger{}, "", false)

	rec := doRequest(h, "GET", "/api/v1/export/meals?start=2026-06-01&end=2026-06-17", nil, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestExportMealsCSV(t *testing.T) {
	store := newFakeMealStore()
	store.mealsInRange = []types.Meal{
		{ID: "m1", UserID: "default", RawText: "200g chicken",
			Items: []types.ResolvedItem{{Macros: types.Macros{Calories: 330, Protein: 62}}}},
	}
	h := newHandler(store, &fakeMealLogger{}, "", false)

	rec := doRequest(h, "GET", "/api/v1/export/meals?start=2026-06-01&end=2026-06-17&format=csv", nil, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	if ct := rec.Header().Get("Content-Type"); !strings.Contains(ct, "text/csv") {
		t.Errorf("expected text/csv, got %s", ct)
	}
}

func TestExportRollupsJSON(t *testing.T) {
	store := newFakeMealStore()
	store.rollups = []types.DailyRollup{
		{UserID: "default", Date: "2026-06-17", Consumed: types.Macros{Calories: 2100}},
	}
	h := newHandler(store, &fakeMealLogger{}, "", false)

	rec := doRequest(h, "GET", "/api/v1/export/rollups?start=2026-06-01&end=2026-06-17", nil, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestExportRollupsCSV(t *testing.T) {
	store := newFakeMealStore()
	store.rollups = []types.DailyRollup{
		{UserID: "default", Date: "2026-06-17", Consumed: types.Macros{Calories: 2100}},
	}
	h := newHandler(store, &fakeMealLogger{}, "", false)

	rec := doRequest(h, "GET", "/api/v1/export/rollups?start=2026-06-01&end=2026-06-17&format=csv", nil, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	if ct := rec.Header().Get("Content-Type"); !strings.Contains(ct, "text/csv") {
		t.Errorf("expected text/csv, got %s", ct)
	}
}

func TestExportMissingParams(t *testing.T) {
	store := newFakeMealStore()
	h := newHandler(store, &fakeMealLogger{}, "", false)

	rec := doRequest(h, "GET", "/api/v1/export/meals", nil, nil)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for missing params, got %d", rec.Code)
	}
}
