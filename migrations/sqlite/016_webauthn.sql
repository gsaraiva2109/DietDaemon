-- 016_webauthn: passkeys (WebAuthn credentials), ceremony session storage,
-- per-user stable handle, and email-OTP fallback codes for MFA step-up.

ALTER TABLE users ADD COLUMN webauthn_handle TEXT;  -- base64, NULL until first passkey
CREATE UNIQUE INDEX IF NOT EXISTS idx_users_webauthn_handle ON users(webauthn_handle);

CREATE TABLE IF NOT EXISTS webauthn_credentials (
    id              TEXT PRIMARY KEY,         -- base64url credential ID
    user_id         TEXT NOT NULL REFERENCES users(id),
    label           TEXT NOT NULL,
    credential_json TEXT NOT NULL,            -- full go-webauthn Credential, JSON
    sign_count      INTEGER NOT NULL DEFAULT 0,
    created_at      TEXT NOT NULL,
    last_used_at    TEXT                      -- NULL until first auth
);
CREATE INDEX IF NOT EXISTS idx_webauthn_credentials_user ON webauthn_credentials(user_id);

CREATE TABLE IF NOT EXISTS webauthn_sessions (
    id           TEXT PRIMARY KEY,            -- opaque ceremony id (random), in dd_webauthn cookie
    user_id      TEXT,                        -- NULL for discoverable login (unknown until finish)
    session_data TEXT NOT NULL,               -- go-webauthn SessionData, JSON
    expires_at   TEXT NOT NULL,
    created_at   TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_webauthn_sessions_expires ON webauthn_sessions(expires_at);

CREATE TABLE IF NOT EXISTS mfa_email_codes (
    challenge_id TEXT PRIMARY KEY,            -- = mfa_challenges.id this code is bound to
    code_hash    TEXT NOT NULL,               -- sha256 hex of the 6-digit code
    expires_at   TEXT NOT NULL,
    attempts     INTEGER NOT NULL DEFAULT 0,
    created_at   TEXT NOT NULL
);
