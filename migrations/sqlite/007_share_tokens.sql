-- share_tokens: read-only dashboard links. Mirrors api_keys exactly — a
-- share token is just another revocable bearer credential, scoped to GET-only
-- routes at the handler layer rather than by anything stored here.
CREATE TABLE IF NOT EXISTS share_tokens (
    id           TEXT PRIMARY KEY,
    user_id      TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    hashed_token TEXT NOT NULL UNIQUE,   -- SHA-256(hex) of the raw token
    label        TEXT,
    created_at   TEXT NOT NULL DEFAULT (datetime('now')),
    last_used_at TEXT,
    revoked_at   TEXT
);
CREATE INDEX IF NOT EXISTS idx_share_tokens_user ON share_tokens(user_id);
