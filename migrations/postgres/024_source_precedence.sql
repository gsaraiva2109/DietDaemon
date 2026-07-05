-- 024_source_precedence: per-user override of the nutrition source resolution
-- order (default order comes from the NUTRITION_SOURCE env var).

CREATE TABLE IF NOT EXISTS source_precedence (
    user_id TEXT NOT NULL REFERENCES users(id),
    source  TEXT NOT NULL,      -- source name, e.g. "openfoodfacts"
    rank    INTEGER NOT NULL,   -- 0 = tried first
    PRIMARY KEY (user_id, source)
);
