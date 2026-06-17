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

type fakeMealLogger struct {
	lastMsg types.InboundMessage
	err     error
}

func (l *fakeMealLogger) Handle(_ context.Context, msg types.InboundMessage) error {
	l.lastMsg = msg
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
