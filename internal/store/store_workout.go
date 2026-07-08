package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/gsaraiva2109/dietdaemon/core/types"
	"github.com/jmoiron/sqlx"
)

// ---------------------------------------------------------------------------
// Workout tracking
// ---------------------------------------------------------------------------

// LogWorkout inserts a workout and its exercises inside a transaction.
func (s *Store) LogWorkout(ctx context.Context, w types.Workout) error {
	tx, err := s.db.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("store: begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	const workoutQ = `
		INSERT INTO workouts (id, user_id, name, duration_min, intensity, calories_burned, note, logged_at, external_id, created_at)
		VALUES (:id, :user_id, :name, :duration_min, :intensity, :calories_burned, :note, :logged_at, :external_id, :created_at)
	`
	workoutQuery, workoutArgs, err := sqlx.Named(workoutQ, map[string]any{
		"id": w.ID, "user_id": w.UserID, "name": w.Name, "duration_min": w.DurationMin,
		"intensity": w.Intensity, "calories_burned": w.CaloriesBurned, "note": nullStr(w.Note),
		"logged_at": w.LoggedAt, "external_id": w.ExternalID, "created_at": utcNow(),
	})
	if err != nil {
		return fmt.Errorf("store: bind insert workout: %w", err)
	}
	if _, err := tx.ExecContext(ctx, s.rewrite(workoutQuery), workoutArgs...); err != nil {
		return fmt.Errorf("store: insert workout: %w", err)
	}

	const exerciseQ = `
		INSERT INTO workout_exercises (id, workout_id, position, name, sets, reps, weight_kg, note)
		VALUES (:id, :workout_id, :position, :name, :sets, :reps, :weight_kg, :note)
	`
	for i, e := range w.Exercises {
		exID := e.ID
		if exID == "" {
			exID = newID()
		}
		exerciseQuery, exerciseArgs, err := sqlx.Named(exerciseQ, map[string]any{
			"id": exID, "workout_id": w.ID, "position": i, "name": e.Name,
			"sets": e.Sets, "reps": e.Reps, "weight_kg": e.WeightKg, "note": nullStr(e.Note),
		})
		if err != nil {
			return fmt.Errorf("store: bind insert exercise: %w", err)
		}
		if _, err := tx.ExecContext(ctx, s.rewrite(exerciseQuery), exerciseArgs...); err != nil {
			return fmt.Errorf("store: insert exercise: %w", err)
		}
	}

	return tx.Commit()
}

// GetWorkout returns a single workout by ID with its exercises populated.
// Returns types.ErrNotFound when the workout does not exist.
func (s *Store) GetWorkout(ctx context.Context, id string) (types.Workout, error) {
	const q = `
		SELECT id, user_id, name, duration_min, intensity, calories_burned, COALESCE(note, '') AS note, logged_at
		FROM workouts WHERE id = ?
	`
	var w types.Workout
	if err := s.db.GetContext(ctx, &w, s.rewrite(q), id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return types.Workout{}, types.ErrNotFound
		}
		return types.Workout{}, fmt.Errorf("store: get workout: %w", err)
	}

	exercises, err := s.loadWorkoutExercises(ctx, id)
	if err != nil {
		return types.Workout{}, err
	}
	w.Exercises = exercises
	return w, nil
}

// ListWorkouts returns the user's most recent workouts without exercises.
func (s *Store) ListWorkouts(ctx context.Context, userID string, limit int) ([]types.Workout, error) {
	const q = `
		SELECT id, user_id, name, duration_min, intensity, calories_burned, COALESCE(note, '') AS note, logged_at
		FROM workouts
		WHERE user_id = ?
		ORDER BY logged_at DESC
		LIMIT ?
	`
	var out []types.Workout
	if err := s.db.SelectContext(ctx, &out, s.rewrite(q), userID, limit); err != nil {
		return nil, fmt.Errorf("store: list workouts: %w", err)
	}
	return out, nil
}

// DeleteWorkout deletes a workout by user + ID. Exercises are cascade-deleted.
// Returns ErrNotFound if absent.
func (s *Store) DeleteWorkout(ctx context.Context, userID, id string) error {
	const q = `DELETE FROM workouts WHERE id = ? AND user_id = ?`
	res, err := s.db.ExecContext(ctx, s.rewrite(q), id, userID)
	if err != nil {
		return fmt.Errorf("store: delete workout: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return types.ErrNotFound
	}
	return nil
}

// ListWorkoutsInRange returns every workout between startDate and endDate
// (inclusive, "YYYY-MM-DD" format), ordered newest first, with no limit.
func (s *Store) ListWorkoutsInRange(ctx context.Context, userID, startDate, endDate string) ([]types.Workout, error) {
	const q = `
		SELECT id, user_id, name, duration_min, intensity, calories_burned, COALESCE(note, '') AS note, logged_at
		FROM workouts
		WHERE user_id = ? AND logged_date >= ? AND logged_date <= ?
		ORDER BY logged_at DESC
	`
	var out []types.Workout
	if err := s.db.SelectContext(ctx, &out, s.rewrite(q), userID, startDate, endDate); err != nil {
		return nil, fmt.Errorf("store: list workouts in range: %w", err)
	}
	return out, nil
}

func (s *Store) loadWorkoutExercises(ctx context.Context, workoutID string) ([]types.WorkoutExercise, error) {
	const q = `
		SELECT id, workout_id, name, sets, reps, weight_kg, COALESCE(note, '') AS note
		FROM workout_exercises
		WHERE workout_id = ?
		ORDER BY position
	`
	var out []types.WorkoutExercise
	if err := s.db.SelectContext(ctx, &out, s.rewrite(q), workoutID); err != nil {
		return nil, fmt.Errorf("store: query exercises: %w", err)
	}
	return out, nil
}

// ImportWorkout inserts a workout with its external_id set (for idempotent import).
// Same transactional insert pattern as LogWorkout. On a unique-constraint violation
// (duplicate external_id for the same user — the re-run-safety case), the call is a
// safe no-op and returns nil rather than an error — "import ran twice" is harmless.
func (s *Store) ImportWorkout(ctx context.Context, w types.Workout) error {
	tx, err := s.db.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("store: begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	const workoutQ = `
		INSERT INTO workouts (id, user_id, name, duration_min, intensity, calories_burned, note, logged_at, external_id, created_at)
		VALUES (:id, :user_id, :name, :duration_min, :intensity, :calories_burned, :note, :logged_at, :external_id, :created_at)
	`
	workoutQuery, workoutArgs, err := sqlx.Named(workoutQ, map[string]any{
		"id": w.ID, "user_id": w.UserID, "name": w.Name, "duration_min": w.DurationMin,
		"intensity": w.Intensity, "calories_burned": w.CaloriesBurned, "note": nullStr(w.Note),
		"logged_at": w.LoggedAt, "external_id": w.ExternalID, "created_at": utcNow(),
	})
	if err != nil {
		return fmt.Errorf("store: bind insert workout: %w", err)
	}
	if _, err := tx.ExecContext(ctx, s.rewrite(workoutQuery), workoutArgs...); err != nil {
		if isUniqueViolation(err) {
			return nil // safe no-op: already imported
		}
		return fmt.Errorf("store: insert workout: %w", err)
	}

	const exerciseQ = `
		INSERT INTO workout_exercises (id, workout_id, position, name, sets, reps, weight_kg, note)
		VALUES (:id, :workout_id, :position, :name, :sets, :reps, :weight_kg, :note)
	`
	for i, e := range w.Exercises {
		exID := e.ID
		if exID == "" {
			exID = newID()
		}
		exerciseQuery, exerciseArgs, err := sqlx.Named(exerciseQ, map[string]any{
			"id": exID, "workout_id": w.ID, "position": i, "name": e.Name,
			"sets": e.Sets, "reps": e.Reps, "weight_kg": e.WeightKg, "note": nullStr(e.Note),
		})
		if err != nil {
			return fmt.Errorf("store: bind insert exercise: %w", err)
		}
		if _, err := tx.ExecContext(ctx, s.rewrite(exerciseQuery), exerciseArgs...); err != nil {
			return fmt.Errorf("store: insert exercise: %w", err)
		}
	}

	return tx.Commit()
}
