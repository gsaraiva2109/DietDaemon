-- 001_initial: bootstrap DietDaemon schema.
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
    confidence  REAL    NOT NULL DEFAULT 0,
    parser_tier INTEGER NOT NULL DEFAULT 0,
    created_at  TEXT    NOT NULL
);

CREATE TABLE IF NOT EXISTS resolved_items (
    id               TEXT PRIMARY KEY,
    meal_id          TEXT    NOT NULL REFERENCES meals(id),
    raw_phrase       TEXT    NOT NULL,
    quantity         REAL    NOT NULL DEFAULT 0,
    unit             TEXT    NOT NULL DEFAULT '',
    normalized_grams REAL    NOT NULL DEFAULT 0,
    food_id          TEXT    NOT NULL DEFAULT '',
    food_name        TEXT    NOT NULL DEFAULT '',
    source           TEXT    NOT NULL DEFAULT '',
    match_score      REAL    NOT NULL DEFAULT 0,
    kcal             REAL    NOT NULL DEFAULT 0,
    protein          REAL    NOT NULL DEFAULT 0,
    carbs            REAL    NOT NULL DEFAULT 0,
    fat              REAL    NOT NULL DEFAULT 0,
    fiber            REAL    NOT NULL DEFAULT 0
);

CREATE TABLE IF NOT EXISTS food_library (
    food_id       TEXT    NOT NULL,
    user_id       TEXT    NOT NULL,
    name          TEXT    NOT NULL,
    source        TEXT    NOT NULL DEFAULT '',
    kcal_100g     REAL    NOT NULL DEFAULT 0,
    protein_100g  REAL    NOT NULL DEFAULT 0,
    carbs_100g    REAL    NOT NULL DEFAULT 0,
    fat_100g      REAL    NOT NULL DEFAULT 0,
    fiber_100g    REAL    NOT NULL DEFAULT 0,
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
    kcal    REAL NOT NULL DEFAULT 0,
    protein REAL NOT NULL DEFAULT 0,
    carbs   REAL NOT NULL DEFAULT 0,
    fat     REAL NOT NULL DEFAULT 0,
    fiber   REAL NOT NULL DEFAULT 0
);

CREATE TABLE IF NOT EXISTS daily_rollups (
    user_id          TEXT NOT NULL,
    date             TEXT NOT NULL,
    consumed_kcal    REAL NOT NULL DEFAULT 0,
    consumed_protein REAL NOT NULL DEFAULT 0,
    consumed_carbs   REAL NOT NULL DEFAULT 0,
    consumed_fat     REAL NOT NULL DEFAULT 0,
    consumed_fiber   REAL NOT NULL DEFAULT 0,
    target_kcal      REAL NOT NULL DEFAULT 0,
    target_protein   REAL NOT NULL DEFAULT 0,
    target_carbs     REAL NOT NULL DEFAULT 0,
    target_fat       REAL NOT NULL DEFAULT 0,
    target_fiber     REAL NOT NULL DEFAULT 0,
    PRIMARY KEY (user_id, date)
);
