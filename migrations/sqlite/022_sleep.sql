-- 022_sleep: sleep session tracking.
-- A sleep log has wake_at NULL while the session is in progress.

CREATE TABLE IF NOT EXISTS sleep_logs (
    id         TEXT PRIMARY KEY,
    user_id    TEXT NOT NULL REFERENCES users(id),
    sleep_at   TEXT NOT NULL,
    wake_at    TEXT,
    quality    TEXT NOT NULL DEFAULT 'ok',
    note       TEXT,
    created_at TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE INDEX IF NOT EXISTS idx_sleep_logs_user_date ON sleep_logs(user_id, sleep_at);
