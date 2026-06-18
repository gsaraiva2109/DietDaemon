-- 010_fts5: full-text search on food_library names and food_aliases.
-- Enables fast prefix/partial matching without LIKE scans.
-- Standard FTS5 (not contentless): columns are stored so JOINs work.

CREATE VIRTUAL TABLE IF NOT EXISTS food_search USING fts5(
    food_id,
    user_id,
    name,
    alias
);

-- Populate from existing food_library rows.
INSERT INTO food_search(food_id, user_id, name, alias)
SELECT food_id, user_id, name, ''
FROM food_library;

-- Populate from existing food_aliases rows.
INSERT INTO food_search(food_id, user_id, name, alias)
SELECT fa.food_id, fa.user_id, fl.name, fa.alias_normalized
FROM food_aliases fa
JOIN food_library fl ON fl.user_id = fa.user_id AND fl.food_id = fa.food_id;

-- Triggers: keep food_search in sync with food_library.

CREATE TRIGGER IF NOT EXISTS food_library_fts_insert AFTER INSERT ON food_library BEGIN
    INSERT INTO food_search(food_id, user_id, name, alias)
    VALUES (NEW.food_id, NEW.user_id, NEW.name, '');
END;

CREATE TRIGGER IF NOT EXISTS food_library_fts_update AFTER UPDATE ON food_library BEGIN
    -- Remove old entries for this food and re-insert with updated name.
    DELETE FROM food_search WHERE food_id = OLD.food_id AND user_id = OLD.user_id;
    INSERT INTO food_search(food_id, user_id, name, alias)
    VALUES (NEW.food_id, NEW.user_id, NEW.name, '');
    -- Re-insert alias rows (aliases are unchanged by food_library UPDATE).
    INSERT INTO food_search(food_id, user_id, name, alias)
    SELECT fa.food_id, fa.user_id, NEW.name, fa.alias_normalized
    FROM food_aliases fa
    WHERE fa.user_id = NEW.user_id AND fa.food_id = NEW.food_id;
END;

CREATE TRIGGER IF NOT EXISTS food_library_fts_delete AFTER DELETE ON food_library BEGIN
    DELETE FROM food_search WHERE food_id = OLD.food_id AND user_id = OLD.user_id;
END;

-- Triggers: keep food_search in sync with food_aliases.

CREATE TRIGGER IF NOT EXISTS food_aliases_fts_insert AFTER INSERT ON food_aliases BEGIN
    INSERT INTO food_search(food_id, user_id, name, alias)
    SELECT fl.food_id, fl.user_id, fl.name, NEW.alias_normalized
    FROM food_library fl
    WHERE fl.user_id = NEW.user_id AND fl.food_id = NEW.food_id;
END;

CREATE TRIGGER IF NOT EXISTS food_aliases_fts_delete AFTER DELETE ON food_aliases BEGIN
    DELETE FROM food_search
    WHERE food_id = OLD.food_id AND user_id = OLD.user_id AND alias = OLD.alias_normalized;
END;
