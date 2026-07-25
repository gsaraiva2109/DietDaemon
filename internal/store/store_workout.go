package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/gsaraiva2109/dietdaemon/core/types"
	"github.com/jmoiron/sqlx"
)

// ---------------------------------------------------------------------------
// Workout tracking
// ---------------------------------------------------------------------------

// LogWorkout inserts a workout and its exercises inside a transaction.
func (s *Store) LogWorkout(ctx context.Context, w types.Workout) error {
	// userLoc queries the DB (GetUser), so it must run before BeginTxx: this
	// pool is single-connection (SQLite), and computing it inside the open tx
	// would self-deadlock waiting for a connection the tx is holding.
	localDate := parseUTC(w.LoggedAt).In(s.userLoc(ctx, w.UserID)).Format("2006-01-02")

	tx, err := s.db.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("store: begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	const workoutQ = `
		INSERT INTO workouts (id, user_id, name, duration_min, intensity, calories_burned, note, logged_at, local_date, external_id, created_at)
		VALUES (:id, :user_id, :name, :duration_min, :intensity, :calories_burned, :note, :logged_at, :local_date, :external_id, :created_at)
	`
	workoutQuery, workoutArgs, err := sqlx.Named(workoutQ, map[string]any{
		"id": w.ID, "user_id": w.UserID, "name": w.Name, "duration_min": w.DurationMin,
		"intensity": w.Intensity, "calories_burned": w.CaloriesBurned, "note": nullStr(w.Note),
		"logged_at": w.LoggedAt, "local_date": localDate, "external_id": w.ExternalID, "created_at": utcNow(),
	})
	if err != nil {
		return fmt.Errorf("store: bind insert workout: %w", err)
	}
	if _, err := tx.ExecContext(ctx, s.rewrite(workoutQuery), workoutArgs...); err != nil {
		return fmt.Errorf("store: insert workout: %w", err)
	}

	const exercisePrefix = `
		INSERT INTO workout_exercises (id, workout_id, position, name, sets, reps, weight_kg, note)
		VALUES `
	if err := s.insertWorkoutExercises(ctx, tx, exercisePrefix, w); err != nil {
		return err
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
		WHERE user_id = ? AND local_date >= ? AND local_date <= ?
		ORDER BY logged_at DESC
	`
	var out []types.Workout
	if err := s.db.SelectContext(ctx, &out, s.rewrite(q), userID, startDate, endDate); err != nil {
		return nil, fmt.Errorf("store: list workouts in range: %w", err)
	}
	return out, nil
}

// GetWorkoutsInRangeWithExercises returns every workout between startDate and
// endDate (inclusive, "YYYY-MM-DD" format) with each workout's exercises
// populated, composing ListWorkoutsInRange and a single batched exercises
// query (avoids one exercises query per workout).
func (s *Store) GetWorkoutsInRangeWithExercises(ctx context.Context, userID, startDate, endDate string) ([]types.Workout, error) {
	workouts, err := s.ListWorkoutsInRange(ctx, userID, startDate, endDate)
	if err != nil {
		return nil, err
	}

	ids := make([]string, len(workouts))
	for i, w := range workouts {
		ids[i] = w.ID
	}
	byWorkout, err := s.loadWorkoutExercisesBatch(ctx, ids)
	if err != nil {
		return nil, err
	}
	for i := range workouts {
		workouts[i].Exercises = byWorkout[workouts[i].ID]
	}
	return workouts, nil
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

// loadWorkoutExercisesBatch loads exercises for every workout ID in one query
// and groups the results by workout_id, avoiding an N+1 query pattern for
// callers that need exercises for many workouts at once (mirrors
// store_meals.go's loadItems).
func (s *Store) loadWorkoutExercisesBatch(ctx context.Context, workoutIDs []string) (map[string][]types.WorkoutExercise, error) {
	if len(workoutIDs) == 0 {
		return nil, nil
	}

	placeholders := make([]string, len(workoutIDs))
	args := make([]any, len(workoutIDs))
	for i, id := range workoutIDs {
		placeholders[i] = s.dialect.Placeholder(i + 1)
		args[i] = id
	}

	// #nosec G201 -- placeholder expansion is ? only, values are args
	q := fmt.Sprintf(`
		SELECT id, workout_id, name, sets, reps, weight_kg, COALESCE(note, '') AS note
		FROM workout_exercises
		WHERE workout_id IN (%s)
		ORDER BY workout_id, position
	`, strings.Join(placeholders, ","))

	var rows []types.WorkoutExercise
	if err := s.db.SelectContext(ctx, &rows, s.rewrite(q), args...); err != nil {
		return nil, fmt.Errorf("store: query exercises: %w", err)
	}

	out := make(map[string][]types.WorkoutExercise)
	for _, r := range rows {
		out[r.WorkoutID] = append(out[r.WorkoutID], r)
	}
	return out, nil
}

// ImportWorkout inserts a workout with its external_id set (for idempotent import).
// Same transactional insert pattern as LogWorkout. On a unique-constraint violation
// (duplicate external_id for the same user — the re-run-safety case), the call is a
// safe no-op and returns nil rather than an error — "import ran twice" is harmless.
func (s *Store) ImportWorkout(ctx context.Context, w types.Workout) error {
	// userLoc queries the DB (GetUser), so it must run before BeginTxx: this
	// pool is single-connection (SQLite), and computing it inside the open tx
	// would self-deadlock waiting for a connection the tx is holding.
	localDate := parseUTC(w.LoggedAt).In(s.userLoc(ctx, w.UserID)).Format("2006-01-02")

	tx, err := s.db.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("store: begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	const workoutQ = `
		INSERT INTO workouts (id, user_id, name, duration_min, intensity, calories_burned, note, logged_at, local_date, external_id, created_at)
		VALUES (:id, :user_id, :name, :duration_min, :intensity, :calories_burned, :note, :logged_at, :local_date, :external_id, :created_at)
	`
	workoutQuery, workoutArgs, err := sqlx.Named(workoutQ, map[string]any{
		"id": w.ID, "user_id": w.UserID, "name": w.Name, "duration_min": w.DurationMin,
		"intensity": w.Intensity, "calories_burned": w.CaloriesBurned, "note": nullStr(w.Note),
		"logged_at": w.LoggedAt, "local_date": localDate, "external_id": w.ExternalID, "created_at": utcNow(),
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

	const exercisePrefix = `
		INSERT INTO workout_exercises (id, workout_id, position, name, sets, reps, weight_kg, note)
		VALUES `
	if err := s.insertWorkoutExercises(ctx, tx, exercisePrefix, w); err != nil {
		return err
	}

	return tx.Commit()
}

func (s *Store) insertWorkoutExercises(ctx context.Context, tx *sqlx.Tx, prefix string, w types.Workout) error {
	rows := make([][]any, 0, len(w.Exercises))
	for i, e := range w.Exercises {
		id := e.ID
		if id == "" {
			id = newID()
		}
		rows = append(rows, []any{id, w.ID, i, e.Name, e.Sets, e.Reps, e.WeightKg, nullStr(e.Note)})
	}
	if err := s.insertRows(ctx, tx, prefix, "", rows); err != nil {
		return fmt.Errorf("store: insert exercises: %w", err)
	}
	return nil
}
