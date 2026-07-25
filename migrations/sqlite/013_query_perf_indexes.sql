-- Query-audit follow-up (db-query-audit.md): workouts.local_date closes the
-- same UTC-day bug #143 fixed for meals/water_logs (see
-- 011_local_date_columns.sql) — logged_date is derived from the UTC
-- logged_at, mis-bucketing workouts logged near local midnight in non-UTC
-- timezones. Existing rows backfill from the old logged_date column as a
-- best-effort approximation; new rows compute the true local date at write
-- time via Store.userLoc, same as meals/water_logs.
ALTER TABLE workouts ADD COLUMN local_date TEXT NOT NULL DEFAULT '';
UPDATE workouts SET local_date = logged_date WHERE local_date = '';
CREATE INDEX IF NOT EXISTS idx_workouts_user_local_date ON workouts(user_id, local_date);

-- Covers GetChatMessages' WHERE session_id = ? ORDER BY created_at DESC —
-- without this, a long-lived session pays an O(session-size) sort on every
-- page load before the LIMIT applies.
CREATE INDEX IF NOT EXISTS idx_chat_messages_session_created ON chat_messages(session_id, created_at DESC);

-- Covers RepairFoodMacros' per-row WHERE source = ? AND name = ? lookup
-- (catalog-import maintenance path); idx_foods_source alone can't seek on
-- name within a source.
CREATE INDEX IF NOT EXISTS idx_foods_source_name ON foods(source, name);

-- Covers GetUserByEmail's bare WHERE email = ? (auth hot path); the existing
-- idx_users_account_email is a composite leading with account_id and can't
-- be used for an email-only seek.
CREATE INDEX IF NOT EXISTS idx_users_email ON users(email) WHERE email IS NOT NULL;

-- Dead weight: idx_meals_user_date_at wraps at_utc in date() (superseded by
-- meals.local_date in 011); water_logs/workouts' logged_date generated
-- columns and their indexes are superseded by local_date above and are no
-- longer read by any query.
DROP INDEX IF EXISTS idx_meals_user_date_at;
DROP INDEX IF EXISTS idx_water_logs_user_date;
DROP INDEX IF EXISTS idx_workouts_user_date;
ALTER TABLE water_logs DROP COLUMN logged_date;
ALTER TABLE workouts DROP COLUMN logged_date;
