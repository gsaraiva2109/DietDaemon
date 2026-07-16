ALTER TABLE foods ADD COLUMN owner_user_id TEXT REFERENCES users(id) ON DELETE CASCADE;
ALTER TABLE foods ADD CONSTRAINT foods_owner_source_check CHECK (
    (owner_user_id IS NULL AND source <> 'custom') OR
    (owner_user_id IS NOT NULL AND source = 'custom')
);
CREATE INDEX IF NOT EXISTS idx_foods_owner ON foods(owner_user_id);

CREATE OR REPLACE FUNCTION foods_fts_insert_fn() RETURNS TRIGGER AS $$
BEGIN
    INSERT INTO food_search(food_id, user_id, name, alias, tsv)
    VALUES (NEW.food_id, COALESCE(NEW.owner_user_id, ''), NEW.name, '',
            to_tsvector('simple', coalesce(NEW.name, '')));
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE OR REPLACE FUNCTION foods_fts_update_fn() RETURNS TRIGGER AS $$
BEGIN
    DELETE FROM food_search WHERE food_id = OLD.food_id AND alias = '';
    INSERT INTO food_search(food_id, user_id, name, alias, tsv)
    VALUES (NEW.food_id, COALESCE(NEW.owner_user_id, ''), NEW.name, '',
            to_tsvector('simple', coalesce(NEW.name, '')));
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DELETE FROM food_search;
INSERT INTO food_search(food_id, user_id, name, alias, tsv)
SELECT food_id, COALESCE(owner_user_id, ''), name, '', to_tsvector('simple', coalesce(name, ''))
FROM foods;
INSERT INTO food_search(food_id, user_id, name, alias, tsv)
SELECT f.food_id, fa.user_id, f.name, fa.alias_normalized,
       to_tsvector('simple', f.name || ' ' || coalesce(fa.alias_normalized, ''))
FROM food_aliases fa JOIN foods f ON f.food_id = fa.food_id;
