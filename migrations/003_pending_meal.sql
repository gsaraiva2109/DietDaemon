-- 003_pending_meal: durable store for short-lived conversational meal state.
-- One row per user (replaced on each new partial meal). Complex fields
-- (ChannelMeta, Resolved, Pending) are stored as JSON for simplicity.

CREATE TABLE IF NOT EXISTS pending_meals (
    user_id      TEXT PRIMARY KEY,
    at_utc       TEXT NOT NULL,
    raw_text     TEXT NOT NULL,
    confidence   REAL NOT NULL DEFAULT 0,
    parser_tier  INTEGER NOT NULL DEFAULT 0,
    channel_meta TEXT NOT NULL DEFAULT '{}',
    resolved     TEXT NOT NULL DEFAULT '[]',
    pending      TEXT NOT NULL DEFAULT '[]',
    created_at   TEXT NOT NULL
);
