-- 023_backup_config: per-user scheduled backup/export settings.
-- destination is 'local' or 's3'; only the fields for the chosen destination
-- are used. Credentials are never stored here (ambient AWS credential chain).

CREATE TABLE IF NOT EXISTS backup_config (
    user_id      TEXT PRIMARY KEY REFERENCES users(id),
    enabled      INTEGER NOT NULL DEFAULT 0,
    destination  TEXT NOT NULL DEFAULT 'local',
    local_subdir TEXT,
    s3_bucket    TEXT,
    s3_prefix    TEXT,
    s3_region    TEXT,
    s3_endpoint  TEXT,
    interval_hrs INTEGER NOT NULL DEFAULT 24,
    last_run_at  TEXT
);
