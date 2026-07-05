-- 003_food_vectors: per-user food embedding vectors for nearest-neighbor
-- matching (Tier-1/Tier-2). vec is a little-endian float32 BLOB; dim records
-- the vector length for sanity checks on load.
-- One row per (user_id, food_id); upsert replaces the vector on food update.

CREATE TABLE IF NOT EXISTS food_vectors (
    user_id TEXT NOT NULL,
    food_id TEXT NOT NULL,
    dim     INTEGER NOT NULL,
    vec     BLOB NOT NULL,
    PRIMARY KEY (user_id, food_id),
    FOREIGN KEY (user_id, food_id) REFERENCES food_library(user_id, food_id)
);
