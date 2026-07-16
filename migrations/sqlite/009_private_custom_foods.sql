-- Custom foods are private. SQLite cannot add a table CHECK constraint to an
-- existing table, so triggers enforce the same invariant for migrated DBs.
ALTER TABLE foods ADD COLUMN owner_user_id TEXT REFERENCES users(id) ON DELETE CASCADE;
CREATE INDEX IF NOT EXISTS idx_foods_owner ON foods(owner_user_id);

CREATE TRIGGER IF NOT EXISTS foods_owner_source_insert
BEFORE INSERT ON foods
WHEN (NEW.owner_user_id IS NULL AND NEW.source = 'custom')
  OR (NEW.owner_user_id IS NOT NULL AND NEW.source <> 'custom')
BEGIN
    SELECT RAISE(ABORT, 'custom foods require an owner');
END;

CREATE TRIGGER IF NOT EXISTS foods_owner_source_update
BEFORE UPDATE OF owner_user_id, source ON foods
WHEN (NEW.owner_user_id IS NULL AND NEW.source = 'custom')
  OR (NEW.owner_user_id IS NOT NULL AND NEW.source <> 'custom')
BEGIN
    SELECT RAISE(ABORT, 'custom foods require an owner');
END;

DROP TRIGGER IF EXISTS foods_fts_insert;
DROP TRIGGER IF EXISTS foods_fts_update;
DROP TRIGGER IF EXISTS foods_fts_delete;
DELETE FROM food_search;
INSERT INTO food_search(food_id, user_id, name, alias)
SELECT food_id, COALESCE(owner_user_id, ''), name, '' FROM foods;
INSERT INTO food_search(food_id, user_id, name, alias)
SELECT f.food_id, fa.user_id, f.name, fa.alias_normalized
FROM food_aliases fa JOIN foods f ON f.food_id = fa.food_id;

CREATE TRIGGER foods_fts_insert AFTER INSERT ON foods BEGIN
    INSERT INTO food_search(food_id, user_id, name, alias)
    VALUES (NEW.food_id, COALESCE(NEW.owner_user_id, ''), NEW.name, '');
END;

CREATE TRIGGER foods_fts_update AFTER UPDATE ON foods BEGIN
    DELETE FROM food_search WHERE food_id = OLD.food_id AND alias = '';
    INSERT INTO food_search(food_id, user_id, name, alias)
    VALUES (NEW.food_id, COALESCE(NEW.owner_user_id, ''), NEW.name, '');
END;

CREATE TRIGGER foods_fts_delete AFTER DELETE ON foods BEGIN
    DELETE FROM food_search WHERE food_id = OLD.food_id;
END;
