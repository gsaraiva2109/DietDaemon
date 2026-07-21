-- local_date stores the user's local calendar date at write time (see
-- pipeline.Engine.userLoc / store.Store.userLoc), replacing UTC-based
-- date() / substring() bucketing that silently mis-bucketed meals and water
-- logs near local midnight for non-UTC timezones (#143).
ALTER TABLE meals ADD COLUMN local_date TEXT NOT NULL DEFAULT '';
CREATE INDEX IF NOT EXISTS idx_meals_user_local_date ON meals(user_id, local_date);

ALTER TABLE water_logs ADD COLUMN local_date TEXT NOT NULL DEFAULT '';
CREATE INDEX IF NOT EXISTS idx_water_logs_user_local_date ON water_logs(user_id, local_date);
