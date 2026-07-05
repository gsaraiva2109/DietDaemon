-- 024_source_precedence: per-user ordering of external nutrition sources.
-- Lower rank is tried first.

CREATE TABLE IF NOT EXISTS source_precedence (
    user_id TEXT NOT NULL REFERENCES users(id),
    source  TEXT NOT NULL,
    rank    INTEGER NOT NULL,
    PRIMARY KEY (user_id, source)
);
