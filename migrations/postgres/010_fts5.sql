-- 010_fts5: full-text search on food_library names and food_aliases (Postgres).
-- Uses tsvector column + GIN index instead of FTS5 virtual table.
-- Trigger functions keep the search table in sync with source tables.

CREATE TABLE IF NOT EXISTS food_search (
    food_id TEXT NOT NULL,
    user_id TEXT NOT NULL,
    name    TEXT NOT NULL,
    alias   TEXT NOT NULL DEFAULT '',
    tsv     TSVECTOR
);

CREATE INDEX IF NOT EXISTS idx_food_search_tsv ON food_search USING GIN (tsv);
CREATE INDEX IF NOT EXISTS idx_food_search_food ON food_search (user_id, food_id);

-- Populate from existing food_library rows.
INSERT INTO food_search(food_id, user_id, name, alias, tsv)
SELECT food_id, user_id, name, '',
       to_tsvector('simple', coalesce(name, ''))
FROM food_library;

-- Populate from existing food_aliases rows.
INSERT INTO food_search(food_id, user_id, name, alias, tsv)
SELECT fa.food_id, fa.user_id, fl.name, fa.alias_normalized,
       to_tsvector('simple', coalesce(fl.name, '') || ' ' || coalesce(fa.alias_normalized, ''))
FROM food_aliases fa
JOIN food_library fl ON fl.user_id = fa.user_id AND fl.food_id = fa.food_id;

-- Trigger: keep food_search in sync with food_library INSERT.

CREATE OR REPLACE FUNCTION food_library_fts_insert_fn() RETURNS TRIGGER AS $$
BEGIN
    INSERT INTO food_search(food_id, user_id, name, alias, tsv)
    VALUES (NEW.food_id, NEW.user_id, NEW.name, '',
            to_tsvector('simple', coalesce(NEW.name, '')));
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS food_library_fts_insert ON food_library;
CREATE TRIGGER food_library_fts_insert AFTER INSERT ON food_library
    FOR EACH ROW EXECUTE FUNCTION food_library_fts_insert_fn();

-- Trigger: keep food_search in sync with food_library UPDATE.

CREATE OR REPLACE FUNCTION food_library_fts_update_fn() RETURNS TRIGGER AS $$
BEGIN
    DELETE FROM food_search WHERE food_id = OLD.food_id AND user_id = OLD.user_id;
    INSERT INTO food_search(food_id, user_id, name, alias, tsv)
    VALUES (NEW.food_id, NEW.user_id, NEW.name, '',
            to_tsvector('simple', coalesce(NEW.name, '')));
    INSERT INTO food_search(food_id, user_id, name, alias, tsv)
    SELECT fa.food_id, fa.user_id, NEW.name, fa.alias_normalized,
           to_tsvector('simple', coalesce(NEW.name, '') || ' ' || coalesce(fa.alias_normalized, ''))
    FROM food_aliases fa
    WHERE fa.user_id = NEW.user_id AND fa.food_id = NEW.food_id;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS food_library_fts_update ON food_library;
CREATE TRIGGER food_library_fts_update AFTER UPDATE ON food_library
    FOR EACH ROW EXECUTE FUNCTION food_library_fts_update_fn();

-- Trigger: keep food_search in sync with food_library DELETE.

CREATE OR REPLACE FUNCTION food_library_fts_delete_fn() RETURNS TRIGGER AS $$
BEGIN
    DELETE FROM food_search WHERE food_id = OLD.food_id AND user_id = OLD.user_id;
    RETURN OLD;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS food_library_fts_delete ON food_library;
CREATE TRIGGER food_library_fts_delete AFTER DELETE ON food_library
    FOR EACH ROW EXECUTE FUNCTION food_library_fts_delete_fn();

-- Trigger: keep food_search in sync with food_aliases INSERT.

CREATE OR REPLACE FUNCTION food_aliases_fts_insert_fn() RETURNS TRIGGER AS $$
BEGIN
    INSERT INTO food_search(food_id, user_id, name, alias, tsv)
    SELECT fl.food_id, fl.user_id, fl.name, NEW.alias_normalized,
           to_tsvector('simple', coalesce(fl.name, '') || ' ' || coalesce(NEW.alias_normalized, ''))
    FROM food_library fl
    WHERE fl.user_id = NEW.user_id AND fl.food_id = NEW.food_id;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS food_aliases_fts_insert ON food_aliases;
CREATE TRIGGER food_aliases_fts_insert AFTER INSERT ON food_aliases
    FOR EACH ROW EXECUTE FUNCTION food_aliases_fts_insert_fn();

-- Trigger: keep food_search in sync with food_aliases DELETE.

CREATE OR REPLACE FUNCTION food_aliases_fts_delete_fn() RETURNS TRIGGER AS $$
BEGIN
    DELETE FROM food_search
    WHERE food_id = OLD.food_id AND user_id = OLD.user_id AND alias = OLD.alias_normalized;
    RETURN OLD;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS food_aliases_fts_delete ON food_aliases;
CREATE TRIGGER food_aliases_fts_delete AFTER DELETE ON food_aliases
    FOR EACH ROW EXECUTE FUNCTION food_aliases_fts_delete_fn();
