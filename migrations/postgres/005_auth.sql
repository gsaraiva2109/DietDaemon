-- 005_auth: tokens and channel-to-user mapping for multi-user support.
-- api_tokens are Bearer tokens for REST API authentication.
-- user_channels maps messaging platform (channel + channel_user_id) to an
-- internal user_id so the pipeline can resolve inbound messages to users.

CREATE TABLE IF NOT EXISTS api_tokens (
    token      TEXT PRIMARY KEY,
    user_id    TEXT NOT NULL REFERENCES users(id),
    created_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS user_channels (
    channel         TEXT NOT NULL,
    channel_user_id TEXT NOT NULL,
    user_id         TEXT NOT NULL REFERENCES users(id),
    PRIMARY KEY (channel, channel_user_id)
);
