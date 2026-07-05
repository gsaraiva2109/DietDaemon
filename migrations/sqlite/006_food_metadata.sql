-- 006_food_metadata: optional metadata columns on food_library for food discovery.
-- SQLite does not support DROP COLUMN; ADD COLUMN with DEFAULT is safe and
-- backward-compatible.

ALTER TABLE food_library ADD COLUMN category     TEXT NOT NULL DEFAULT '';
ALTER TABLE food_library ADD COLUMN brand        TEXT NOT NULL DEFAULT '';
ALTER TABLE food_library ADD COLUMN barcode      TEXT NOT NULL DEFAULT '';
ALTER TABLE food_library ADD COLUMN image_url    TEXT NOT NULL DEFAULT '';
ALTER TABLE food_library ADD COLUMN serving_size REAL NOT NULL DEFAULT 0;
ALTER TABLE food_library ADD COLUMN serving_unit TEXT NOT NULL DEFAULT '';
