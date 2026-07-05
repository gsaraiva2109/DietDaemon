-- 021_workout: exercise session tracking with individual exercises.

CREATE TABLE IF NOT EXISTS workouts (
    id              TEXT PRIMARY KEY,
    user_id         TEXT NOT NULL REFERENCES users(id),
    name            TEXT NOT NULL,
    duration_min    INTEGER NOT NULL,
    intensity       TEXT NOT NULL DEFAULT 'moderate',
    calories_burned INTEGER,
    note            TEXT,
    logged_at       TEXT NOT NULL,
    created_at      TEXT NOT NULL DEFAULT (NOW())
);

CREATE INDEX IF NOT EXISTS idx_workouts_user_date ON workouts(user_id, logged_at);

CREATE TABLE IF NOT EXISTS workout_exercises (
    id         TEXT PRIMARY KEY,
    workout_id TEXT NOT NULL REFERENCES workouts(id) ON DELETE CASCADE,
    name       TEXT NOT NULL,
    sets       INTEGER,
    reps       INTEGER,
    weight_kg  DOUBLE PRECISION,
    note       TEXT
);

CREATE INDEX IF NOT EXISTS idx_workout_exercises_workout ON workout_exercises(workout_id);
