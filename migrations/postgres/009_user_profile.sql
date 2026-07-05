-- 009_user_profile: per-user profile for TDEE calculation and goal tracking.

CREATE TABLE IF NOT EXISTS user_profiles (
    user_id          TEXT PRIMARY KEY REFERENCES users(id),
    height_cm        DOUBLE PRECISION NOT NULL DEFAULT 0,
    birth_date       TEXT NOT NULL DEFAULT '',
    gender           TEXT NOT NULL DEFAULT '',
    activity_level   TEXT NOT NULL DEFAULT '',
    goal             TEXT NOT NULL DEFAULT '',
    target_weight_kg DOUBLE PRECISION NOT NULL DEFAULT 0,
    weekly_rate      DOUBLE PRECISION NOT NULL DEFAULT 0,
    onboarded        INTEGER NOT NULL DEFAULT 0,
    created_at       TEXT NOT NULL,
    updated_at       TEXT NOT NULL
);
