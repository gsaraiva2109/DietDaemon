-- 014_email: single-use tokens for email verification and password reset.
-- Tokens are keyed by SHA-256 hex of the random token sent to the user; only
-- the hash is persisted so a DB leak cannot replay the token.

CREATE TABLE IF NOT EXISTS auth_email_tokens (
    id         TEXT PRIMARY KEY,  -- SHA-256 hex of the random token
    user_id    TEXT NOT NULL REFERENCES users(id),
    purpose    TEXT NOT NULL,     -- 'verify' | 'reset'
    expires_at TEXT NOT NULL,
    created_at TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_auth_email_tokens_expires ON auth_email_tokens(expires_at);
