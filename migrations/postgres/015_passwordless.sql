-- 015_passwordless: magic code table for passwordless email sign-in.
-- Codes are 6-digit low-entropy secrets, so we need an attempt cap and the
-- code_hash is always stored hashed (SHA-256). One active code per user
-- (user_id is the PK); resend overwrites via upsert.

CREATE TABLE IF NOT EXISTS auth_magic_codes (
    user_id    TEXT PRIMARY KEY REFERENCES users(id),
    code_hash  TEXT NOT NULL,   -- SHA-256 hex of the 6-digit code
    expires_at TEXT NOT NULL,
    attempts   INTEGER NOT NULL DEFAULT 0,
    created_at TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_auth_magic_codes_expires ON auth_magic_codes(expires_at);
