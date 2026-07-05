-- 023_pending_aliases: embedding-matched aliases awaiting user confirmation.
-- Replaces the silent write-back into food_aliases for near-miss matches:
-- the match is parked here until the user confirms or rejects it.

CREATE TABLE IF NOT EXISTS pending_aliases (
    id          TEXT PRIMARY KEY,
    user_id     TEXT NOT NULL REFERENCES users(id),
    phrase      TEXT NOT NULL,      -- raw phrase that triggered the match
    food_id     TEXT NOT NULL,      -- candidate food_library.food_id
    match_score REAL NOT NULL,      -- embedding similarity score
    created_at  TEXT NOT NULL DEFAULT (datetime('now'))
);
