-- 004_workout_external_id: add external_id column for idempotent imports (Hevy, MyFitnessPal, etc.).
-- Partial unique index — NULL external_id stays non-unique (manually logged workouts, the
-- overwhelming majority of rows); the constraint only applies to imported rows.
ALTER TABLE workouts ADD COLUMN IF NOT EXISTS external_id TEXT;
CREATE UNIQUE INDEX IF NOT EXISTS idx_workouts_user_external
    ON workouts(user_id, external_id) WHERE external_id IS NOT NULL;
