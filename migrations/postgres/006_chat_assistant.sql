-- 006_chat_assistant: chat assistant persistence tables (sessions, messages, per-user settings).
-- Stores conversation history and custom system-prompt instructions for DietDaemon's
-- AI chat assistant (internal/assistant). Distinct from chat_routes (proactive nudges).
CREATE TABLE IF NOT EXISTS chat_sessions (
    id         TEXT PRIMARY KEY,
    user_id    TEXT NOT NULL REFERENCES users(id),
    title      TEXT NOT NULL DEFAULT '',
    created_at TEXT NOT NULL DEFAULT (NOW()),
    updated_at TEXT NOT NULL DEFAULT (NOW())
);

CREATE TABLE IF NOT EXISTS chat_messages (
    id         TEXT PRIMARY KEY,
    session_id TEXT NOT NULL REFERENCES chat_sessions(id),
    role       TEXT NOT NULL,
    content    TEXT NOT NULL,
    tool_name  TEXT,
    created_at TEXT NOT NULL DEFAULT (NOW())
);

CREATE TABLE IF NOT EXISTS user_assistant_settings (
    user_id            TEXT PRIMARY KEY REFERENCES users(id),
    custom_instructions TEXT NOT NULL DEFAULT '',
    updated_at         TEXT NOT NULL DEFAULT (NOW())
);
