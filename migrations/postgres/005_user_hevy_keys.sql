-- 005_user_hevy_keys: per-user Hevy API keys for workout import.
-- Keys are encrypted at rest with AES-256-GCM (same pattern as user_ai_keys).
CREATE TABLE IF NOT EXISTS user_hevy_keys (
    user_id    TEXT PRIMARY KEY REFERENCES users(id),
    enc_key    TEXT NOT NULL,       -- AES-256-GCM ciphertext, base64
    created_at TEXT NOT NULL DEFAULT (NOW())
);
