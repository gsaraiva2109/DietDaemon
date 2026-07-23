package api

import (
	"net/http"
	"testing"

	"github.com/gsaraiva2109/dietdaemon/core/types"
)

func TestHandleLogWorkout(t *testing.T) {
	store := newFakeMealStore()
	h := newHandler(store, &fakeMealLogger{})

	body := map[string]any{"name": "Push day", "duration_min": 45, "intensity": "heavy"}
	rec := doRequest(h, "POST", "/api/v1/body/workouts", body, nil)
	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rec.Code, rec.Body.String())
	}
	entry := decodeJSON[types.Workout](t, rec)
	if entry.Name != "Push day" {
		t.Errorf("name = %q, want Push day", entry.Name)
	}
	if entry.UserID != "test-user" {
		t.Errorf("user_id = %q, want test-user", entry.UserID)
	}
	if entry.Intensity != "heavy" {
		t.Errorf("intensity = %q, want heavy", entry.Intensity)
	}
	if len(store.workouts) != 1 {
		t.Errorf("expected 1 stored workout, got %d", len(store.workouts))
	}
}

func TestHandleLogWorkoutDefaultIntensity(t *testing.T) {
	store := newFakeMealStore()
	h := newHandler(store, &fakeMealLogger{})

	body := map[string]any{"name": "Run", "duration_min": 30}
	rec := doRequest(h, "POST", "/api/v1/body/workouts", body, nil)
	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rec.Code, rec.Body.String())
	}
	entry := decodeJSON[types.Workout](t, rec)
	if entry.Intensity != "moderate" {
		t.Errorf("intensity = %q, want default moderate", entry.Intensity)
	}
}

func TestHandleLogWorkoutMissingName(t *testing.T) {
	store := newFakeMealStore()
	h := newHandler(store, &fakeMealLogger{})

	rec := doRequest(h, "POST", "/api/v1/body/workouts", map[string]any{"duration_min": 30}, nil)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("missing name expected 400, got %d", rec.Code)
	}
}

func TestHandleLogWorkoutInvalidDuration(t *testing.T) {
	store := newFakeMealStore()
	h := newHandler(store, &fakeMealLogger{})

	rec := doRequest(h, "POST", "/api/v1/body/workouts", map[string]any{"name": "Run", "duration_min": 0}, nil)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("zero duration expected 400, got %d", rec.Code)
	}
}

func TestHandleLogWorkoutInvalidIntensity(t *testing.T) {
	store := newFakeMealStore()
	h := newHandler(store, &fakeMealLogger{})

	body := map[string]any{"name": "Run", "duration_min": 30, "intensity": "extreme"}
	rec := doRequest(h, "POST", "/api/v1/body/workouts", body, nil)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("invalid intensity expected 400, got %d", rec.Code)
	}
}

func TestHandleLogWorkoutUnauthorized(t *testing.T) {
	store := newFakeMealStore()
	h := newHandler(store, &fakeMealLogger{})

	body := map[string]any{"name": "Run", "duration_min": 30}
	rec := doRequest(h, "POST", "/api/v1/body/workouts", body, map[string]string{"Authorization": ""})
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rec.Code)
	}
}

func TestHandleListWorkouts(t *testing.T) {
	store := newFakeMealStore()
	store.workouts = []types.Workout{
		{ID: "w1", UserID: "test-user", Name: "Run"},
		{ID: "w2", UserID: "test-user", Name: "Swim"},
	}
	h := newHandler(store, &fakeMealLogger{})

	rec := doRequest(h, "GET", "/api/v1/body/workouts?limit=5", nil, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	workouts := decodeJSON[[]types.Workout](t, rec)
	if len(workouts) != 2 {
		t.Errorf("expected 2 workouts, got %d", len(workouts))
	}
}

func TestHandleListWorkoutsEmpty(t *testing.T) {
	store := newFakeMealStore()
	h := newHandler(store, &fakeMealLogger{})

	rec := doRequest(h, "GET", "/api/v1/body/workouts", nil, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	workouts := decodeJSON[[]types.Workout](t, rec)
	if workouts == nil {
		t.Error("expected empty array, got null")
	}
}

func TestHandleListWorkoutsLimitOutOfRange(t *testing.T) {
	store := newFakeMealStore()
	h := newHandler(store, &fakeMealLogger{})

	rec := doRequest(h, "GET", "/api/v1/body/workouts?limit=0", nil, nil)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("out-of-range limit expected 400, got %d", rec.Code)
	}
}

func TestHandleListWorkoutsUnauthorized(t *testing.T) {
	store := newFakeMealStore()
	h := newHandler(store, &fakeMealLogger{})

	rec := doRequest(h, "GET", "/api/v1/body/workouts", nil, map[string]string{"Authorization": ""})
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rec.Code)
	}
}

func TestHandleGetWorkout(t *testing.T) {
	store := newFakeMealStore()
	store.workoutsByID = map[string]types.Workout{
		"w1": {ID: "w1", UserID: "test-user", Name: "Run"},
	}
	h := newHandler(store, &fakeMealLogger{})

	rec := doRequest(h, "GET", "/api/v1/body/workouts/w1", nil, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	workout := decodeJSON[types.Workout](t, rec)
	if workout.ID != "w1" {
		t.Errorf("id = %q, want w1", workout.ID)
	}
}

func TestHandleGetWorkoutNotFound(t *testing.T) {
	store := newFakeMealStore()
	h := newHandler(store, &fakeMealLogger{})

	rec := doRequest(h, "GET", "/api/v1/body/workouts/missing", nil, nil)
	if rec.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", rec.Code)
	}
}

func TestHandleGetWorkoutWrongUser(t *testing.T) {
	store := newFakeMealStore()
	store.workoutsByID = map[string]types.Workout{
		"w1": {ID: "w1", UserID: "other-user", Name: "Run"},
	}
	h := newHandler(store, &fakeMealLogger{})

	rec := doRequest(h, "GET", "/api/v1/body/workouts/w1", nil, nil)
	if rec.Code != http.StatusNotFound {
		t.Errorf("cross-user workout access expected 404, got %d", rec.Code)
	}
}

func TestHandleGetWorkoutUnauthorized(t *testing.T) {
	store := newFakeMealStore()
	h := newHandler(store, &fakeMealLogger{})

	rec := doRequest(h, "GET", "/api/v1/body/workouts/w1", nil, map[string]string{"Authorization": ""})
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rec.Code)
	}
}

func TestHandleDeleteWorkout(t *testing.T) {
	store := newFakeMealStore()
	h := newHandler(store, &fakeMealLogger{})

	rec := doRequest(h, "DELETE", "/api/v1/body/workouts/w1", nil, nil)
	if rec.Code != http.StatusNoContent {
		t.Errorf("expected 204, got %d", rec.Code)
	}
}

func TestHandleDeleteWorkoutNotFound(t *testing.T) {
	store := newFakeMealStore()
	store.deleteWorkoutErr = types.ErrNotFound
	h := newHandler(store, &fakeMealLogger{})

	rec := doRequest(h, "DELETE", "/api/v1/body/workouts/missing", nil, nil)
	if rec.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", rec.Code)
	}
}

func TestHandleDeleteWorkoutUnauthorized(t *testing.T) {
	store := newFakeMealStore()
	h := newHandler(store, &fakeMealLogger{})

	rec := doRequest(h, "DELETE", "/api/v1/body/workouts/w1", nil, map[string]string{"Authorization": ""})
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rec.Code)
	}
}
