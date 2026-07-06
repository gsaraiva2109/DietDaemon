-- 027_chat_routes: reverse routing from user_id to the chat metadata needed
-- to deliver a message proactively (chat id, channel id, room id — whichever
-- the active MessagingAdapter needs). Refreshed on every inbound message so
-- the scheduler can reach a user without waiting for them to message first.

CREATE TABLE IF NOT EXISTS chat_routes (
    user_id    TEXT NOT NULL REFERENCES users(id),
    channel    TEXT NOT NULL,
    meta_json  TEXT NOT NULL,
    updated_at TEXT NOT NULL,
    PRIMARY KEY (user_id, channel)
);
