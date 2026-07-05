-- 011_auth_foundation: real auth — accounts (tenant root), user credentials,
-- server sessions, machine API keys, audit log, login-attempt tracking.

CREATE TABLE IF NOT EXISTS accounts (
    id         TEXT PRIMARY KEY,
    created_at TEXT NOT NULL
);

-- Extend users (tenant-ready + login identity). SQLite & Postgres both support
-- ALTER TABLE ADD COLUMN. email stored already-lowercased by the app layer.
ALTER TABLE users ADD COLUMN account_id        TEXT REFERENCES accounts(id);
ALTER TABLE users ADD COLUMN email             TEXT;
ALTER TABLE users ADD COLUMN email_verified_at TEXT;
ALTER TABLE users ADD COLUMN status            TEXT NOT NULL DEFAULT 'active';
ALTER TABLE users ADD COLUMN display_name      TEXT;

-- Backfill: one account per existing user, link it.
INSERT INTO accounts (id, created_at) SELECT id, created_at FROM users;
UPDATE users SET account_id = id WHERE account_id IS NULL;

-- Case-insensitive-ish uniqueness: app lowercases email before write/compare.
CREATE UNIQUE INDEX IF NOT EXISTS idx_users_account_email
    ON users(account_id, email) WHERE email IS NOT NULL;

CREATE TABLE IF NOT EXISTS password_credentials (
    user_id    TEXT PRIMARY KEY REFERENCES users(id),
    phc_hash   TEXT NOT NULL,          -- argon2id PHC string
    updated_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS sessions (
    id                  TEXT PRIMARY KEY,  -- SHA-256(hex) of the cookie token
    user_id             TEXT NOT NULL REFERENCES users(id),
    csrf_token          TEXT NOT NULL,
    created_at          TEXT NOT NULL,
    last_seen_at        TEXT NOT NULL,
    idle_expires_at     TEXT NOT NULL,
    absolute_expires_at TEXT NOT NULL,
    remember            INTEGER NOT NULL DEFAULT 0,
    ip                  TEXT,
    user_agent          TEXT
);
CREATE INDEX IF NOT EXISTS idx_sessions_user ON sessions(user_id);

CREATE TABLE IF NOT EXISTS api_keys (
    id           TEXT PRIMARY KEY,
    user_id      TEXT NOT NULL REFERENCES users(id),
    hashed_key   TEXT NOT NULL UNIQUE,   -- SHA-256(hex) of the raw key
    label        TEXT NOT NULL,
    created_at   TEXT NOT NULL,
    last_used_at TEXT,
    revoked_at   TEXT
);
CREATE INDEX IF NOT EXISTS idx_api_keys_user ON api_keys(user_id);

CREATE TABLE IF NOT EXISTS auth_audit_log (
    id         TEXT PRIMARY KEY,
    account_id TEXT,
    user_id    TEXT,
    event      TEXT NOT NULL,   -- e.g. login.success, login.fail, logout, register, apikey.create, apikey.revoke, lockout
    ip         TEXT,
    user_agent TEXT,
    meta       TEXT,            -- optional JSON
    created_at TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_audit_account ON auth_audit_log(account_id, created_at);

CREATE TABLE IF NOT EXISTS login_attempts (
    id          TEXT PRIMARY KEY,
    identifier  TEXT NOT NULL,   -- lowercased email (per-account) or "ip:<addr>"
    succeeded   INTEGER NOT NULL,
    created_at  TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_login_attempts_id ON login_attempts(identifier, created_at);

-- Retire the plaintext token table; machine clients now use api_keys.
-- Messaging adapters are unaffected (they use user_channels, not tokens).
DROP TABLE IF EXISTS api_tokens;
