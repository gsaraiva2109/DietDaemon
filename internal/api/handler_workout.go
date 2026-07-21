package api

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/gsaraiva2109/dietdaemon/core/types"
)

// ---------------------------------------------------------------------------
// Workout tracking handlers.
// ---------------------------------------------------------------------------

func (h *Handler) handleLogWorkout(w http.ResponseWriter, r *http.Request, userID string) {
	var body struct {
		Name           string                  `json:"name"`
		DurationMin    int                     `json:"duration_min"`
		Intensity      string                  `json:"intensity"`
		CaloriesBurned *int                    `json:"calories_burned,omitempty"`
		Note           string                  `json:"note,omitempty"`
		LoggedAt       string                  `json:"loggedAt"`
		Exercises      []types.WorkoutExercise `json:"exercises,omitempty"`
	}
	if err := decodeRequestJSON(r, &body); err != nil {
		writeValidationError(w, "invalid JSON body")
		return
	}
	if body.Name == "" {
		writeValidationError(w, "name is required")
		return
	}
	if body.DurationMin <= 0 {
		writeValidationError(w, "duration_min must be positive")
		return
	}
	if body.Intensity == "" {
		body.Intensity = "moderate"
	}
	if !validWorkoutIntensity(body.Intensity) {
		writeValidationError(w, "intensity is invalid")
		return
	}
	if body.LoggedAt == "" {
		body.LoggedAt = time.Now().UTC().Format(time.RFC3339)
	}
	entry := types.Workout{
		ID:             newHandlerID(),
		UserID:         userID,
		Name:           body.Name,
		DurationMin:    body.DurationMin,
		Intensity:      body.Intensity,
		CaloriesBurned: body.CaloriesBurned,
		Note:           body.Note,
		LoggedAt:       body.LoggedAt,
		Exercises:      body.Exercises,
	}
	if entry.Exercises == nil {
		entry.Exercises = []types.WorkoutExercise{}
	}
	if err := h.store.LogWorkout(r.Context(), entry); err != nil {
		h.writeErr(w, err)
		return
	}
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(entry)
}

func (h *Handler) handleListWorkouts(w http.ResponseWriter, r *http.Request, userID string) {
	limit, ok := boundedQueryInt(w, r, "limit", 10, 1, 100)
	if !ok {
		return
	}
	workouts, err := h.store.ListWorkouts(r.Context(), userID, limit)
	if err != nil {
		h.writeErr(w, err)
		return
	}
	if workouts == nil {
		workouts = []types.Workout{}
	}
	_ = json.NewEncoder(w).Encode(workouts)
}

func (h *Handler) handleGetWorkout(w http.ResponseWriter, r *http.Request, userID string) {
	id := r.PathValue("id")
	workout, err := h.store.GetWorkout(r.Context(), id)
	if err != nil {
		h.writeErr(w, err)
		return
	}
	if workout.UserID != userID {
		h.writeErr(w, types.ErrNotFound)
		return
	}
	_ = json.NewEncoder(w).Encode(workout)
}

func (h *Handler) handleDeleteWorkout(w http.ResponseWriter, r *http.Request, userID string) {
	id := r.PathValue("id")
	if err := h.store.DeleteWorkout(r.Context(), userID, id); err != nil {
		h.writeErr(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
