-- 001_initial: bootstrap DietDaemon schema (Postgres).
-- All tables are keyed by user_id from day one so multi-user is a flag, not a rewrite.

CREATE TABLE IF NOT EXISTS users (
    id         TEXT PRIMARY KEY,
    timezone   TEXT NOT NULL DEFAULT '',
    created_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS meals (
    id          TEXT PRIMARY KEY,
    user_id     TEXT    NOT NULL REFERENCES users(id),
    at_utc      TEXT    NOT NULL,
    raw_text    TEXT    NOT NULL,
    confidence  DOUBLE PRECISION NOT NULL DEFAULT 0,
    parser_tier INTEGER NOT NULL DEFAULT 0,
    created_at  TEXT    NOT NULL
);

CREATE TABLE IF NOT EXISTS resolved_items (
    id               TEXT PRIMARY KEY,
    meal_id          TEXT    NOT NULL REFERENCES meals(id),
    raw_phrase       TEXT    NOT NULL,
    quantity         DOUBLE PRECISION NOT NULL DEFAULT 0,
    unit             TEXT    NOT NULL DEFAULT '',
    normalized_grams DOUBLE PRECISION NOT NULL DEFAULT 0,
    food_id          TEXT    NOT NULL DEFAULT '',
    food_name        TEXT    NOT NULL DEFAULT '',
    source           TEXT    NOT NULL DEFAULT '',
    match_score      DOUBLE PRECISION NOT NULL DEFAULT 0,
    kcal             DOUBLE PRECISION NOT NULL DEFAULT 0,
    protein          DOUBLE PRECISION NOT NULL DEFAULT 0,
    carbs            DOUBLE PRECISION NOT NULL DEFAULT 0,
    fat              DOUBLE PRECISION NOT NULL DEFAULT 0,
    fiber            DOUBLE PRECISION NOT NULL DEFAULT 0
);

CREATE TABLE IF NOT EXISTS food_library (
    food_id       TEXT    NOT NULL,
    user_id       TEXT    NOT NULL,
    name          TEXT    NOT NULL,
    source        TEXT    NOT NULL DEFAULT '',
    kcal_100g     DOUBLE PRECISION NOT NULL DEFAULT 0,
    protein_100g  DOUBLE PRECISION NOT NULL DEFAULT 0,
    carbs_100g    DOUBLE PRECISION NOT NULL DEFAULT 0,
    fat_100g      DOUBLE PRECISION NOT NULL DEFAULT 0,
    fiber_100g    DOUBLE PRECISION NOT NULL DEFAULT 0,
    query_count   INTEGER NOT NULL DEFAULT 0,
    last_used     TEXT    NOT NULL DEFAULT '',
    PRIMARY KEY (user_id, food_id)
);

CREATE TABLE IF NOT EXISTS food_aliases (
    user_id          TEXT NOT NULL,
    alias_normalized TEXT NOT NULL,
    food_id          TEXT NOT NULL,
    PRIMARY KEY (user_id, alias_normalized),
    FOREIGN KEY (user_id, food_id) REFERENCES food_library(user_id, food_id)
);

CREATE TABLE IF NOT EXISTS daily_targets (
    user_id TEXT PRIMARY KEY,
    kcal    DOUBLE PRECISION NOT NULL DEFAULT 0,
    protein DOUBLE PRECISION NOT NULL DEFAULT 0,
    carbs   DOUBLE PRECISION NOT NULL DEFAULT 0,
    fat     DOUBLE PRECISION NOT NULL DEFAULT 0,
    fiber   DOUBLE PRECISION NOT NULL DEFAULT 0
);

CREATE TABLE IF NOT EXISTS daily_rollups (
    user_id          TEXT NOT NULL,
    date             TEXT NOT NULL,
    consumed_kcal    DOUBLE PRECISION NOT NULL DEFAULT 0,
    consumed_protein DOUBLE PRECISION NOT NULL DEFAULT 0,
    consumed_carbs   DOUBLE PRECISION NOT NULL DEFAULT 0,
    consumed_fat     DOUBLE PRECISION NOT NULL DEFAULT 0,
    consumed_fiber   DOUBLE PRECISION NOT NULL DEFAULT 0,
    target_kcal      DOUBLE PRECISION NOT NULL DEFAULT 0,
    target_protein   DOUBLE PRECISION NOT NULL DEFAULT 0,
    target_carbs     DOUBLE PRECISION NOT NULL DEFAULT 0,
    target_fat       DOUBLE PRECISION NOT NULL DEFAULT 0,
    target_fiber     DOUBLE PRECISION NOT NULL DEFAULT 0,
    PRIMARY KEY (user_id, date)
);
