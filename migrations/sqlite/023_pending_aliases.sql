-- 023_pending_aliases: alias candidates from embedding near-misses awaiting
-- user confirmation before they are promoted into food_aliases.

CREATE TABLE IF NOT EXISTS pending_aliases (
    id          TEXT PRIMARY KEY,
    user_id     TEXT NOT NULL REFERENCES users(id),
    phrase      TEXT NOT NULL,
    food_id     TEXT NOT NULL,
    match_score REAL NOT NULL,
    created_at  TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE INDEX IF NOT EXISTS idx_pending_aliases_user ON pending_aliases(user_id);
