-- 026_backup_config: per-user scheduled backup/export settings, local disk or S3.

CREATE TABLE IF NOT EXISTS backup_config (
    user_id       TEXT PRIMARY KEY REFERENCES users(id),
    enabled       INTEGER NOT NULL DEFAULT 0,
    destination   TEXT NOT NULL DEFAULT 'local', -- "local" | "s3"
    local_subdir  TEXT NOT NULL DEFAULT '',
    s3_bucket     TEXT NOT NULL DEFAULT '',
    s3_prefix     TEXT NOT NULL DEFAULT '',
    s3_region     TEXT NOT NULL DEFAULT '',
    s3_endpoint   TEXT NOT NULL DEFAULT '',
    interval_hrs  INTEGER NOT NULL DEFAULT 24,
    last_run_at   TEXT NOT NULL DEFAULT ''
);
