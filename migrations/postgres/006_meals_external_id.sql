-- external_id supports idempotent one-shot imports (MyFitnessPal, etc.) —
-- NULL for normally logged meals, the overwhelming majority. Mirrors
-- workouts.external_id.
ALTER TABLE meals ADD COLUMN external_id TEXT;
CREATE UNIQUE INDEX IF NOT EXISTS idx_meals_user_external
    ON meals(user_id, external_id) WHERE external_id IS NOT NULL;
