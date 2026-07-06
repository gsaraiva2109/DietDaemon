CREATE TABLE IF NOT EXISTS sent_nudges (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL REFERENCES users(id),
    rule_id TEXT NOT NULL,
    sent_at TEXT NOT NULL,
    body TEXT NOT NULL,
    snapshot_json TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'sent',
    resolved_at TEXT
);
