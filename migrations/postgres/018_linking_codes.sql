-- 018_linking_codes: one-time codes for bot-to-dashboard account linking.
-- Links a chat account (Telegram/Discord/Matrix) to a dashboard user.
-- Codes expire after 10 minutes and are single-use.

CREATE TABLE IF NOT EXISTS linking_codes (
    code        TEXT PRIMARY KEY,   -- 6-char random alphanumeric
    user_id     TEXT NOT NULL,      -- dashboard user ID (accounts.id)
    platform    TEXT NOT NULL,      -- "telegram", "discord", "matrix"
    created_at  TEXT NOT NULL DEFAULT (NOW()),
    expires_at  TEXT NOT NULL,      -- created_at + 10 minutes
    used_at     TEXT                -- NULL = unused, set on use
);
