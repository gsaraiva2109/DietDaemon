package api

import (
	"errors"
	"net/http"
	"strings"
	"testing"

	"github.com/gsaraiva2109/dietdaemon/core/types"
)

func TestGoalsRoutesRequireAuth(t *testing.T) {
	store := newFakeMealStore()
	h := newHandler(store, &fakeMealLogger{})

	rec := doRequest(h, "GET", "/api/v1/profile", nil, map[string]string{"Authorization": ""})
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rec.Code)
	}
}

func TestGetProfileStoreError(t *testing.T) {
	store := newFakeMealStore()
	store.profileErr = errors.New("db unavailable")
	h := newHandler(store, &fakeMealLogger{})

	rec := doRequest(h, "GET", "/api/v1/profile", nil, nil)
	if rec.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", rec.Code)
	}
}

func TestUpsertProfileInvalidJSON(t *testing.T) {
	store := newFakeMealStore()
	h := newHandler(store, &fakeMealLogger{})

	rec := doRequestRawBody(h, "PUT", "/api/v1/profile", "not json")
	if rec.Code != http.StatusBadRequest {
		t.Errorf("invalid JSON expected 400, got %d", rec.Code)
	}
}

func TestUpsertProfileValidation(t *testing.T) {
	for name, body := range map[string]types.UserProfile{
		"height too low":        {HeightCm: 10},
		"height too high":       {HeightCm: 400},
		"target weight too low": {TargetWeightKg: 5},
		"negative weekly rate":  {WeeklyRate: -1},
		"invalid gender":        {Gender: "unspecified"},
		"invalid activity":      {ActivityLevel: "extreme"},
		"invalid goal":          {Goal: "shred"},
		"future birth date":     {BirthDate: "9999-12-31"},
	} {
		t.Run(name, func(t *testing.T) {
			rec := doRequest(newHandler(newFakeMealStore(), &fakeMealLogger{}), "PUT", "/api/v1/profile", body, nil)
			if rec.Code != http.StatusBadRequest {
				t.Errorf("expected 400, got %d: %s", rec.Code, rec.Body.String())
			}
		})
	}
}

func TestUpsertProfileStoreError(t *testing.T) {
	store := newFakeMealStore()
	store.upsertProfileErr = errors.New("db unavailable")
	h := newHandler(store, &fakeMealLogger{})

	body := types.UserProfile{HeightCm: 180, Gender: "male", ActivityLevel: "moderate", Goal: "maintain"}
	rec := doRequest(h, "PUT", "/api/v1/profile", body, nil)
	if rec.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", rec.Code)
	}
}

func TestCalculateTDEEInvalidGender(t *testing.T) {
	store := newFakeMealStore()
	h := newHandler(store, &fakeMealLogger{})

	rec := doRequest(h, "GET", "/api/v1/tdee?weight_kg=80&height_cm=175&age=30&gender=alien&activity=moderate", nil, nil)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("invalid gender expected 400, got %d", rec.Code)
	}
}

func TestCalculateTDEEInvalidActivity(t *testing.T) {
	store := newFakeMealStore()
	h := newHandler(store, &fakeMealLogger{})

	rec := doRequest(h, "GET", "/api/v1/tdee?weight_kg=80&height_cm=175&age=30&gender=male&activity=insane", nil, nil)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("invalid activity expected 400, got %d", rec.Code)
	}
}

func TestCalculateTDEEOutOfRangeAge(t *testing.T) {
	store := newFakeMealStore()
	h := newHandler(store, &fakeMealLogger{})

	rec := doRequest(h, "GET", "/api/v1/tdee?weight_kg=80&height_cm=175&age=0&gender=male&activity=moderate", nil, nil)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("age=0 expected 400, got %d", rec.Code)
	}
}

func TestGoalSuggestionsMissingBirthDate(t *testing.T) {
	store := newFakeMealStore()
	store.profile = types.UserProfile{UserID: "test-user", HeightCm: 175}
	h := newHandler(store, &fakeMealLogger{})

	rec := doRequest(h, "GET", "/api/v1/goals/suggestions", nil, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	got := decodeJSON[types.GoalSuggestion](t, rec)
	if !strings.Contains(got.Message, "birth date") {
		t.Errorf("expected birth date prompt, got %q", got.Message)
	}
}

func TestGoalSuggestionsMissingHeight(t *testing.T) {
	store := newFakeMealStore()
	store.profile = types.UserProfile{UserID: "test-user", BirthDate: "1990-01-01"}
	h := newHandler(store, &fakeMealLogger{})

	rec := doRequest(h, "GET", "/api/v1/goals/suggestions", nil, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	got := decodeJSON[types.GoalSuggestion](t, rec)
	if !strings.Contains(got.Message, "height") {
		t.Errorf("expected height prompt, got %q", got.Message)
	}
}

func TestGoalSuggestionsNoWeightLogged(t *testing.T) {
	store := newFakeMealStore()
	store.profile = types.UserProfile{UserID: "test-user", BirthDate: "1990-01-01", HeightCm: 175}
	h := newHandler(store, &fakeMealLogger{})

	rec := doRequest(h, "GET", "/api/v1/goals/suggestions", nil, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	got := decodeJSON[types.GoalSuggestion](t, rec)
	if !strings.Contains(got.Message, "weight") {
		t.Errorf("expected weight prompt, got %q", got.Message)
	}
}

func TestGoalSuggestionsFullHappyPath(t *testing.T) {
	store := newFakeMealStore()
	store.profile = types.UserProfile{
		UserID: "test-user", BirthDate: "1990-01-01", HeightCm: 175,
		Gender: "male", ActivityLevel: "moderate", Goal: "lose", TargetWeightKg: 75,
	}
	store.weights = []types.WeightEntry{{ID: "w1", WeightKg: 80, Date: "2026-06-17"}}
	store.weightTrend = []types.WeightTrend{
		{Date: "2026-06-10", RollingAvg: 81},
		{Date: "2026-06-17", RollingAvg: 80},
	}
	store.rollups = []types.DailyRollup{
		{UserID: "test-user", Date: "2026-06-17", Consumed: types.Macros{Calories: 2000}},
	}
	h := newHandler(store, &fakeMealLogger{})

	rec := doRequest(h, "GET", "/api/v1/goals/suggestions", nil, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	got := decodeJSON[types.GoalSuggestion](t, rec)
	if got.RecommendedKcal <= 0 {
		t.Errorf("expected positive recommended kcal, got %v", got.RecommendedKcal)
	}
	if got.TargetLossKg != 5 {
		t.Errorf("expected target loss 5kg (80-75), got %v", got.TargetLossKg)
	}
	if got.CurrentIntakeKcal != 2000 {
		t.Errorf("expected current intake 2000, got %v", got.CurrentIntakeKcal)
	}
}
