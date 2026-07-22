-- Named ways to log a food beyond grams (#134). user_id NULL = system-provided
-- (from import, e.g. USDA foodPortions); non-NULL = a user's own custom unit,
-- visible only to them. Same table serves both so the UI has one list.
CREATE TABLE food_serving_units (
    id         TEXT PRIMARY KEY,
    user_id    TEXT REFERENCES users(id) ON DELETE CASCADE,
    food_id    TEXT NOT NULL REFERENCES foods(food_id) ON DELETE CASCADE,
    label      TEXT NOT NULL,
    grams      REAL NOT NULL CHECK (grams > 0),
    created_at TEXT NOT NULL DEFAULT ''
);
CREATE INDEX idx_food_serving_units_food ON food_serving_units(food_id);
CREATE INDEX idx_food_serving_units_user ON food_serving_units(user_id);
