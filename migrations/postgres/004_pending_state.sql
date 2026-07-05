-- 004_pending_meals: BYTEA-backed durable pending meal store.
-- One row per user; entire PendingMeal JSON-marshalled into payload.
-- created_at is Unix epoch seconds, duplicated out of the payload so
-- expiry can be evaluated in SQL or Go without unmarshalling.
-- Counterpart: internal/pendingstore.

CREATE TABLE IF NOT EXISTS pending_state (
    user_id    TEXT PRIMARY KEY,
    created_at INTEGER NOT NULL,
    payload    BYTEA NOT NULL
);
