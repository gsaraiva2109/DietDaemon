-- 008_body_tracking: weight log, measurement log, and progress photos.

CREATE TABLE IF NOT EXISTS weight_log (
    id         TEXT PRIMARY KEY,
    user_id    TEXT NOT NULL REFERENCES users(id),
    date       TEXT NOT NULL,
    weight_kg  REAL NOT NULL,
    note       TEXT NOT NULL DEFAULT '',
    created_at TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_weight_log_user_date ON weight_log(user_id, date);

CREATE TABLE IF NOT EXISTS measurement_log (
    id            TEXT PRIMARY KEY,
    user_id       TEXT NOT NULL REFERENCES users(id),
    date          TEXT NOT NULL,
    waist_cm      REAL NOT NULL DEFAULT 0,
    hips_cm       REAL NOT NULL DEFAULT 0,
    chest_cm      REAL NOT NULL DEFAULT 0,
    left_arm_cm   REAL NOT NULL DEFAULT 0,
    right_arm_cm  REAL NOT NULL DEFAULT 0,
    left_thigh_cm REAL NOT NULL DEFAULT 0,
    right_thigh_cm REAL NOT NULL DEFAULT 0,
    note          TEXT NOT NULL DEFAULT '',
    created_at    TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_measurement_log_user_date ON measurement_log(user_id, date);

CREATE TABLE IF NOT EXISTS progress_photos (
    id         TEXT PRIMARY KEY,
    user_id    TEXT NOT NULL REFERENCES users(id),
    date       TEXT NOT NULL,
    view       TEXT NOT NULL CHECK(view IN ('front', 'side', 'back')),
    mime_type  TEXT NOT NULL,
    data       BLOB NOT NULL,
    created_at TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_progress_photos_user_date ON progress_photos(user_id, date);
