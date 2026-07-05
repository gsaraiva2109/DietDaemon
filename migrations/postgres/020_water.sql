-- 020_water: water consumption tracking.
-- Each row records one water entry (e.g. a glass or bottle).

CREATE TABLE IF NOT EXISTS water_logs (
    id         TEXT PRIMARY KEY,
    user_id    TEXT NOT NULL REFERENCES users(id),
    amount_ml  INTEGER NOT NULL,
    logged_at  TEXT NOT NULL,
    note       TEXT,
    created_at TEXT NOT NULL DEFAULT (NOW())
);

CREATE INDEX IF NOT EXISTS idx_water_logs_user_date ON water_logs(user_id, logged_at);
