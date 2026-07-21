package api

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gsaraiva2109/dietdaemon/core/types"
)

func assertValidationError(t *testing.T, rec *httptest.ResponseRecorder) {
	t.Helper()
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400: %s", rec.Code, rec.Body.String())
	}
	if got := decodeJSON[errorEnvelope](t, rec); got.Error.Code != ErrorValidation {
		t.Fatalf("error code = %q, want %q", got.Error.Code, ErrorValidation)
	}
}

func TestValidationRejectsInvalidTDEEInputs(t *testing.T) {
	h := newHandler(newFakeMealStore(), &fakeMealLogger{})
	for name, path := range map[string]string{
		"weight":     "/api/v1/tdee?weight_kg=nope&height_cm=180&age=30&gender=male&activity=moderate",
		"height":     "/api/v1/tdee?weight_kg=80&height_cm=301&age=30&gender=male&activity=moderate",
		"age":        "/api/v1/tdee?weight_kg=80&height_cm=180&age=121&gender=male&activity=moderate",
		"activity":   "/api/v1/tdee?weight_kg=80&height_cm=180&age=30&gender=male&activity=unknown",
		"pagination": "/api/v1/foods?limit=invalid",
	} {
		t.Run(name, func(t *testing.T) {
			assertValidationError(t, doRequest(h, http.MethodGet, path, nil, nil))
		})
	}
}

func TestValidationRejectsMalformedOptionalFastBody(t *testing.T) {
	h := newHandler(newFakeMealStore(), &fakeMealLogger{})
	rec := httptest.NewRecorder()
	h.handleStartFast(rec, httptest.NewRequest(http.MethodPost, "/", strings.NewReader("{")), "test-user")
	assertValidationError(t, rec)
}

func TestValidationRejectsMalformedOptionalPasskeyBody(t *testing.T) {
	h := newHandler(newFakeMealStore(), &fakeMealLogger{})
	rec := httptest.NewRecorder()
	h.handlePasskeyLoginBegin(rec, httptest.NewRequest(http.MethodPost, "/", strings.NewReader("{")))
	assertValidationError(t, rec)
}

func TestValidationRejectsInvalidPersistedInputs(t *testing.T) {
	store := newFakeMealStore()
	store.foodsByID = map[string]types.FoodMatch{"food": {FoodID: "food", Per100g: types.Macros{Calories: 100}}}
	h := newHandler(store, &fakeMealLogger{})
	tomorrow := time.Now().UTC().AddDate(0, 0, 1).Format("2006-01-02")

	for name, tc := range map[string]struct {
		method string
		path   string
		body   any
	}{
		"negative targets":      {http.MethodPut, "/api/v1/targets", types.Macros{Calories: -1}},
		"empty measurements":    {http.MethodPost, "/api/v1/body/measurements", map[string]any{"date": "2026-01-01"}},
		"future weight":         {http.MethodPost, "/api/v1/body/weight", map[string]any{"date": tomorrow, "weight_kg": 80}},
		"profile enum":          {http.MethodPut, "/api/v1/profile", types.UserProfile{HeightCm: 180, Gender: "invalid"}},
		"sleep enum":            {http.MethodPost, "/api/v1/body/sleep", map[string]any{"quality": "invalid"}},
		"workout enum":          {http.MethodPost, "/api/v1/body/workouts", map[string]any{"name": "Walk", "duration_min": 30, "intensity": "invalid"}},
		"precedence source":     {http.MethodPut, "/api/v1/settings/precedence", map[string]any{"order": []string{"invalid"}}},
		"email":                 {http.MethodPost, "/api/v1/auth/email/change", map[string]any{"email": "invalid", "current_password": "password"}},
		"structured meal grams": {http.MethodPost, "/api/v1/meals", map[string]any{"items": []map[string]any{{"food_id": "food", "grams": 0}}}},
		"template grams":        {http.MethodPost, "/api/v1/templates/compose", map[string]any{"name": "Meal", "items": []map[string]any{{"food_id": "food", "grams": 0}}}},
	} {
		t.Run(name, func(t *testing.T) {
			assertValidationError(t, doRequest(h, tc.method, tc.path, tc.body, nil))
		})
	}
}

func TestValidDateUsesLocalCalendarDay(t *testing.T) {
	loc := time.FixedZone("UTC-3", -3*60*60)
	today := time.Now().In(loc).Format("2006-01-02")
	if !validDate(today, loc) {
		t.Fatalf("today %q should be valid in %s", today, loc)
	}
}
