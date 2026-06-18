-- 007_meal_templates: reusable meal templates and usage log.

CREATE TABLE IF NOT EXISTS meal_templates (
    id         TEXT PRIMARY KEY,
    user_id    TEXT NOT NULL REFERENCES users(id),
    name       TEXT NOT NULL,
    items_json TEXT NOT NULL DEFAULT '[]',
    created_at TEXT NOT NULL,
    last_used  TEXT NOT NULL DEFAULT ''
);

CREATE TABLE IF NOT EXISTS template_logs (
    id          TEXT PRIMARY KEY,
    user_id     TEXT NOT NULL REFERENCES users(id),
    template_id TEXT NOT NULL REFERENCES meal_templates(id),
    logged_at   TEXT NOT NULL
);
