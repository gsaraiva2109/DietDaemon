-- 003_user_ai_keys: per-user AI API keys for BYOK (bring-your-own-key).
-- Keys are encrypted at rest with AES-256-GCM (same pattern as totp_secrets.secret).
CREATE TABLE IF NOT EXISTS user_ai_keys (
    user_id    TEXT PRIMARY KEY REFERENCES users(id),
    provider   TEXT NOT NULL,       -- "anthropic" | "openai"
    enc_key    TEXT NOT NULL,       -- AES-256-GCM ciphertext, base64
    created_at TEXT NOT NULL DEFAULT (datetime('now'))
);
