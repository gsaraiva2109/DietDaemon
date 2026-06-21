-- 012_totp: TOTP two-factor auth with encrypted-at-rest secrets, recovery codes,
-- and MFA challenge tokens for step-up authentication.

CREATE TABLE IF NOT EXISTS totp_secrets (
    user_id      TEXT PRIMARY KEY REFERENCES users(id),
    secret       TEXT NOT NULL,  -- AES-256-GCM ciphertext (base64), NOT raw base32
    confirmed_at TEXT            -- NULL until enrollment is verified
);

CREATE TABLE IF NOT EXISTS recovery_codes (
    id         TEXT PRIMARY KEY,
    user_id    TEXT NOT NULL REFERENCES users(id),
    code_hash  TEXT NOT NULL,   -- SHA-256 hex of the code
    used_at    TEXT,            -- NULL → available; set on first use
    created_at TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_recovery_codes_user ON recovery_codes(user_id);

CREATE TABLE IF NOT EXISTS mfa_challenges (
    id         TEXT PRIMARY KEY,  -- SHA-256 hex of the challenge token
    user_id    TEXT NOT NULL,
    remember   INTEGER NOT NULL DEFAULT 0,
    expires_at TEXT NOT NULL,
    created_at TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_mfa_challenges_user ON mfa_challenges(user_id);
