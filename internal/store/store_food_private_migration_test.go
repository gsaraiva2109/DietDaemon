package store

import (
	"testing"
	"time"

	"github.com/gsaraiva2109/dietdaemon/core/types"
)

// assertPrivateFoodMigrationApplied verifies that migration
// 009_private_custom_foods.sql has been applied exactly once.
func assertPrivateFoodMigrationApplied(t *testing.T, s *Store) {
	t.Helper()

	var applied int
	if err := s.db.Get(&applied, s.rewrite(`SELECT COUNT(*) FROM schema_migrations WHERE name = ?`), "009_private_custom_foods.sql"); err != nil {
		t.Fatalf("query migration: %v", err)
	}
	if applied != 1 {
		t.Fatalf("migration count = %d, want 1", applied)
	}
}

// assertPrivateFoodOwnershipConstraints verifies the CHECK constraints added
// by the private-custom-foods migration: a custom food must have an owner, a
// non-custom food must not, and an owned custom food (with its alias) can be
// inserted normally.
func assertPrivateFoodOwnershipConstraints(t *testing.T, s *Store) {
	t.Helper()

	if _, err := s.db.Exec(`INSERT INTO foods (food_id, name, source) VALUES ('global-food', 'Global migration food', 'test')`); err != nil {
		t.Fatalf("insert global food: %v", err)
	}
	if _, err := s.db.Exec(`INSERT INTO foods (food_id, name, source) VALUES ('missing-owner', 'Missing owner', 'custom')`); err == nil {
		t.Fatal("custom food without owner unexpectedly succeeded")
	}
	if _, err := s.db.Exec(`INSERT INTO foods (food_id, owner_user_id, name, source) VALUES ('wrong-source', 'private-owner', 'Wrong source', 'test')`); err == nil {
		t.Fatal("owned non-custom food unexpectedly succeeded")
	}
	if _, err := s.db.Exec(`INSERT INTO foods (food_id, owner_user_id, name, source) VALUES ('private-food', 'private-owner', 'Private migration food', 'custom')`); err != nil {
		t.Fatalf("insert private food: %v", err)
	}
	if _, err := s.db.Exec(`INSERT INTO food_aliases (user_id, alias_normalized, food_id) VALUES ('private-owner', 'private migration alias', 'private-food')`); err != nil {
		t.Fatalf("insert private alias: %v", err)
	}
}

// assertPrivateFoodSearchScoping verifies that the food_search trigger scopes
// the private food to its owner while leaving the global food unscoped, and
// that SearchCatalog respects that scoping.
func assertPrivateFoodSearchScoping(t *testing.T, s *Store) {
	t.Helper()

	var globalSearchOwner, privateSearchOwner string
	if err := s.db.Get(&globalSearchOwner, `SELECT user_id FROM food_search WHERE food_id = 'global-food' AND alias = ''`); err != nil {
		t.Fatalf("get global search owner: %v", err)
	}
	if err := s.db.Get(&privateSearchOwner, `SELECT user_id FROM food_search WHERE food_id = 'private-food' AND alias = ''`); err != nil {
		t.Fatalf("get private search owner: %v", err)
	}
	if globalSearchOwner != "" || privateSearchOwner != "private-owner" {
		t.Fatalf("search owners = global %q, private %q", globalSearchOwner, privateSearchOwner)
	}

	owned, err := s.SearchCatalog(ctx(), "private-owner", "private migration", "", 20, 0)
	if err != nil || len(owned) != 1 || owned[0].FoodID != "private-food" {
		t.Fatalf("owner catalog search = %+v, %v", owned, err)
	}
	other, err := s.SearchCatalog(ctx(), "other-user", "private migration", "", 20, 0)
	if err != nil {
		t.Fatalf("other-user catalog search: %v", err)
	}
	if len(other) != 0 {
		t.Fatalf("private food leaked into other-user catalog: %+v", other)
	}
}

// assertPrivateFoodCascadeDelete verifies that deleting a private food's
// owner cascades to the food row and its search index entries.
func assertPrivateFoodCascadeDelete(t *testing.T, s *Store) {
	t.Helper()

	if _, err := s.db.Exec(`DELETE FROM users WHERE id = 'private-owner'`); err != nil {
		t.Fatalf("delete private owner: %v", err)
	}
	var remaining int
	if err := s.db.Get(&remaining, `SELECT COUNT(*) FROM foods WHERE food_id = 'private-food'`); err != nil {
		t.Fatalf("count private foods: %v", err)
	}
	if remaining != 0 {
		t.Fatalf("private food survived owner deletion: %d rows", remaining)
	}
	if err := s.db.Get(&remaining, `SELECT COUNT(*) FROM food_search WHERE food_id = 'private-food'`); err != nil {
		t.Fatalf("count private search rows: %v", err)
	}
	if remaining != 0 {
		t.Fatalf("private search rows survived owner deletion: %d rows", remaining)
	}
}

// runPrivateFoodMigrationCase runs the full private-custom-foods migration
// scenario against a single driver.
func runPrivateFoodMigrationCase(t *testing.T, s *Store) {
	t.Helper()

	for _, id := range []string{"private-owner", "other-user"} {
		mustUser(t, s, types.User{ID: id, CreatedAt: time.Now().UTC()})
	}

	assertPrivateFoodMigrationApplied(t, s)
	assertPrivateFoodOwnershipConstraints(t, s)
	assertPrivateFoodSearchScoping(t, s)
	assertPrivateFoodCascadeDelete(t, s)
}

func TestPrivateFoodMigration(t *testing.T) {
	drivers := map[string]func(*testing.T) (*Store, func()){
		"sqlite":   func(t *testing.T) (*Store, func()) { return tempDB(t) },
		"postgres": func(t *testing.T) (*Store, func()) { return postgresDB(t) },
	}
	for name, factory := range drivers {
		t.Run(name, func(t *testing.T) {
			s, cleanup := factory(t)
			defer cleanup()
			runPrivateFoodMigrationCase(t, s)
		})
	}
}
