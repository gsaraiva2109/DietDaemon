// Package hevy implements one-time Hevy workout import (API client + data transform).
// Schema types mirror Hevy's public OpenAPI spec exactly so json.Unmarshal works out of the box.
package hevy

import "time"

// HevyWorkout is a single workout as returned by GET /v1/workouts and GET /v1/workouts/{id}.
// Field names/types match Hevy's confirmed API response (OpenAPI spec).
type HevyWorkout struct {
	ID        string         `json:"id"`
	Title     string         `json:"title"`
	StartTime time.Time      `json:"start_time"`
	EndTime   time.Time      `json:"end_time"`
	Exercises []HevyExercise `json:"exercises"`
}

// HevyExercise is one exercise within a Hevy workout.
type HevyExercise struct {
	Index              int       `json:"index"`
	Title              string    `json:"title"`
	ExerciseTemplateID string    `json:"exercise_template_id"`
	Notes              *string   `json:"notes"`
	Sets               []HevySet `json:"sets"`
}

// HevySet is one set within a Hevy exercise.
type HevySet struct {
	Index    int      `json:"index"`
	Type     string   `json:"type"` // "warmup" | "normal" | "failure" | "dropset"
	WeightKg *float64 `json:"weight_kg"`
	Reps     *int     `json:"reps"`
}
