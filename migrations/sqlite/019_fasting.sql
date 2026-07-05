-- 019_fasting: intermittent-fasting windows (Phase 4d).
-- A user has at most one active fast (end_at IS NULL) at a time.

CREATE TABLE IF NOT EXISTS fasts (
    id           TEXT PRIMARY KEY,
    user_id      TEXT NOT NULL REFERENCES users(id),
    start_at     TEXT NOT NULL,
    end_at       TEXT,                       -- NULL = still fasting
    target_hours REAL NOT NULL DEFAULT 16,
    completed    INTEGER NOT NULL DEFAULT 0, -- 1 once target reached at end
    created_at   TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_fasts_user_start ON fasts(user_id, start_at);

-- Fast lookup of the single in-progress fast per user.
CREATE INDEX IF NOT EXISTS idx_fasts_active ON fasts(user_id) WHERE end_at IS NULL;
