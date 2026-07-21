package store

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/gsaraiva2109/dietdaemon/core/types"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

func postgresDB(t *testing.T) (*Store, func()) {
	t.Helper()
	ctx := context.Background()

	pgContainer, err := postgres.Run(ctx, "postgres:16-alpine",
		postgres.WithDatabase("dietdaemon_test"),
		postgres.WithUsername("test"),
		postgres.WithPassword("test"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(60*time.Second),
		),
	)
	if err != nil {
		t.Fatalf("start postgres container: %v", err)
	}

	dsn, err := pgContainer.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		t.Fatalf("postgres connection string: %v", err)
	}

	s, err := New("postgres", dsn, postgresDialect{}, nil)
	if err != nil {
		t.Fatalf("New(postgres): %v", err)
	}

	return s, func() {
		_ = s.Close()
		if err := pgContainer.Terminate(ctx); err != nil {
			t.Logf("terminate postgres container: %v", err)
		}
	}
}

func TestPostgresUserRoundTrip(t *testing.T) {
	s, cleanup := postgresDB(t)
	defer cleanup()

	u := types.User{
		ID:        "user-pg-1",
		Timezone:  "America/Sao_Paulo",
		CreatedAt: time.Date(2026, 7, 5, 12, 0, 0, 0, time.UTC),
	}

	if _, err := s.GetUser(ctx(), "user-pg-1"); err != types.ErrNotFound {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}

	if err := s.UpsertUser(ctx(), u); err != nil {
		t.Fatalf("UpsertUser: %v", err)
	}

	got, err := s.GetUser(ctx(), "user-pg-1")
	if err != nil {
		t.Fatalf("GetUser: %v", err)
	}
	if got.ID != u.ID || got.Timezone != u.Timezone {
		t.Fatalf("mismatch: got %+v, want %+v", got, u)
	}

	u.Timezone = "Europe/Lisbon"
	if err := s.UpsertUser(ctx(), u); err != nil {
		t.Fatalf("UpsertUser (update): %v", err)
	}
	got, _ = s.GetUser(ctx(), "user-pg-1")
	if got.Timezone != "Europe/Lisbon" {
		t.Fatalf("expected updated timezone, got %s", got.Timezone)
	}
}

func TestPostgresMealLifecycle(t *testing.T) {
	s, cleanup := postgresDB(t)
	defer cleanup()

	u := types.User{ID: "user-pg-2", Timezone: "UTC", CreatedAt: time.Now().UTC()}
	if err := s.UpsertUser(ctx(), u); err != nil {
		t.Fatalf("UpsertUser: %v", err)
	}

	meal := types.Meal{
		ID:      "meal-pg-1",
		UserID:  "user-pg-2",
		At:      time.Date(2026, 7, 5, 12, 0, 0, 0, time.UTC),
		RawText: "200g chicken, 2 eggs",
		Items: []types.ResolvedItem{
			{
				Parsed: types.ParsedItem{RawPhrase: "chicken", Quantity: 200, Unit: "g", NormalizedGrams: 200},
				Match:  types.FoodMatch{FoodID: "f1", Name: "Chicken Breast", Source: "taco"},
				Macros: types.Macros{Calories: 330, Protein: 62, Fat: 7.2},
			},
			{
				Parsed: types.ParsedItem{RawPhrase: "eggs", Quantity: 2, Unit: "un", NormalizedGrams: 100},
				Match:  types.FoodMatch{FoodID: "f2", Name: "Egg", Source: "taco"},
				Macros: types.Macros{Calories: 143, Protein: 13, Fat: 10},
			},
		},
		CreatedAt: time.Now().UTC(),
	}
	if err := s.SaveMeal(ctx(), meal); err != nil {
		t.Fatalf("SaveMeal: %v", err)
	}

	f1 := types.FoodMatch{FoodID: "f1", Name: "Chicken Breast", Source: "taco", Per100g: types.Macros{Calories: 165, Protein: 31, Fat: 3.6}}
	f2 := types.FoodMatch{FoodID: "f2", Name: "Egg", Source: "taco", Per100g: types.Macros{Calories: 143, Protein: 13, Fat: 10}}
	if err := s.UpsertFood(ctx(), "user-pg-2", f1, nil); err != nil {
		t.Fatalf("UpsertFood f1: %v", err)
	}
	if err := s.UpsertFood(ctx(), "user-pg-2", f2, nil); err != nil {
		t.Fatalf("UpsertFood f2: %v", err)
	}

	rollup := types.DailyRollup{
		UserID:   "user-pg-2",
		Date:     "2026-07-05",
		Consumed: types.Macros{Calories: 473, Protein: 75, Fat: 17.2},
		Targets:  types.Macros{Calories: 2000, Protein: 150, Carbs: 200, Fat: 65, Fiber: 30},
	}
	if err := s.UpsertRollup(ctx(), rollup); err != nil {
		t.Fatalf("UpsertRollup: %v", err)
	}

	got, err := s.GetRollup(ctx(), "user-pg-2", "2026-07-05")
	if err != nil {
		t.Fatalf("GetRollup: %v", err)
	}
	if got.Consumed.Calories != 473 {
		t.Errorf("expected 473 kcal in rollup, got %f", got.Consumed.Calories)
	}
}

func TestPostgresSearchFoods(t *testing.T) {
	s, cleanup := postgresDB(t)
	defer cleanup()

	u := types.User{ID: "user-pg-3", Timezone: "UTC", CreatedAt: time.Now().UTC()}
	if err := s.UpsertUser(ctx(), u); err != nil {
		t.Fatalf("UpsertUser: %v", err)
	}

	foods := []types.FoodMatch{
		{FoodID: "sf1", Name: "Chicken Breast", Source: "taco", Per100g: types.Macros{Calories: 165, Protein: 31, Fat: 3.6}},
		{FoodID: "sf2", Name: "Chicken Thigh", Source: "taco", Per100g: types.Macros{Calories: 209, Protein: 26, Fat: 11}},
		{FoodID: "sf3", Name: "Salmon Fillet", Source: "taco", Per100g: types.Macros{Calories: 208, Protein: 20, Fat: 13}},
	}
	for _, f := range foods {
		if err := s.UpsertFood(ctx(), "user-pg-3", f, nil); err != nil {
			t.Fatalf("UpsertFood %s: %v", f.FoodID, err)
		}
	}

	results, err := s.SearchFoods(ctx(), "user-pg-3", "chicken")
	if err != nil {
		t.Fatalf("SearchFoods: %v", err)
	}
	if len(results) != 2 {
		t.Errorf("expected 2 chicken results, got %d", len(results))
	}

	results, err = s.SearchFoods(ctx(), "user-pg-3", "salmon")
	if err != nil {
		t.Fatalf("SearchFoods salmon: %v", err)
	}
	if len(results) != 1 {
		t.Errorf("expected 1 salmon result, got %d", len(results))
	}

	// Test case for duplicate search results when aliases are present
	if err := s.UpsertFood(ctx(), "user-pg-3", types.FoodMatch{
		FoodID: "sf1", Name: "Chicken Breast", Source: "taco", Per100g: types.Macros{Calories: 165, Protein: 31, Fat: 3.6},
	}, []string{"chicken breast grilled"}); err != nil {
		t.Fatalf("UpsertFood with alias: %v", err)
	}

	results, err = s.SearchFoods(ctx(), "user-pg-3", "chicken")
	if err != nil {
		t.Fatalf("SearchFoods after alias: %v", err)
	}
	// Let's count how many times sf1 appears.
	sf1Count := 0
	for _, r := range results {
		if r.FoodID == "sf1" {
			sf1Count++
		}
	}
	if sf1Count > 1 {
		t.Errorf("expected sf1 to appear at most once, but got it %d times in results", sf1Count)
	}
}

func TestPostgresDualDriverSmoke(t *testing.T) {
	drivers := map[string]func() (*Store, func()){
		"sqlite":   func() (*Store, func()) { return tempDB(t) },
		"postgres": func() (*Store, func()) { return postgresDB(t) },
	}

	for name, factory := range drivers {
		t.Run(name, func(t *testing.T) {
			s, cleanup := factory()
			defer cleanup()

			u := types.User{
				ID:        "user-smoke-" + name,
				Timezone:  "UTC",
				CreatedAt: time.Date(2026, 7, 5, 12, 0, 0, 0, time.UTC),
			}
			if err := s.UpsertUser(ctx(), u); err != nil {
				t.Fatalf("UpsertUser: %v", err)
			}

			got, err := s.GetUser(ctx(), u.ID)
			if err != nil {
				t.Fatalf("GetUser: %v", err)
			}
			if got.ID != u.ID {
				t.Errorf("ID mismatch: %q != %q", got.ID, u.ID)
			}

			targets := types.DailyTargets{
				UserID:  u.ID,
				Targets: types.Macros{Calories: 2000, Protein: 150, Carbs: 200, Fat: 65, Fiber: 30},
			}
			if err := s.SetTargets(ctx(), targets); err != nil {
				t.Fatalf("SetTargets: %v", err)
			}

			rollup := types.DailyRollup{
				UserID:  u.ID,
				Date:    "2026-07-05",
				Targets: targets.Targets,
			}
			if err := s.UpsertRollup(ctx(), rollup); err != nil {
				t.Fatalf("UpsertRollup: %v", err)
			}

			gotRollup, err := s.GetRollup(ctx(), u.ID, "2026-07-05")
			if err != nil {
				t.Fatalf("GetRollup: %v", err)
			}
			if gotRollup.Targets.Calories != 2000 {
				t.Errorf("Targets.Calories = %f, want 2000", gotRollup.Targets.Calories)
			}
		})
	}
}

func TestFoodImportFingerprintStore(t *testing.T) {
	drivers := map[string]func(*testing.T) (*Store, func()){
		"sqlite":   func(t *testing.T) (*Store, func()) { return tempDB(t) },
		"postgres": func(t *testing.T) (*Store, func()) { return postgresDB(t) },
	}

	for name, factory := range drivers {
		t.Run(name, func(t *testing.T) {
			s, cleanup := factory(t)
			defer cleanup()

			var migrations int
			if err := s.db.Get(&migrations, s.rewrite(`SELECT COUNT(*) FROM schema_migrations WHERE name = ?`), "004_food_import_fingerprints.sql"); err != nil {
				t.Fatalf("query migration: %v", err)
			}
			if migrations != 1 {
				t.Fatalf("migration count = %d, want 1", migrations)
			}

			if _, err := s.GetFoodImportFingerprint(ctx(), "usda"); !errors.Is(err, types.ErrNotFound) {
				t.Fatalf("empty fingerprint error = %v, want ErrNotFound", err)
			}
			if err := s.SetFoodImportFingerprint(ctx(), "usda", "first"); err != nil {
				t.Fatalf("set first fingerprint: %v", err)
			}
			if got, err := s.GetFoodImportFingerprint(ctx(), "usda"); err != nil || got != "first" {
				t.Fatalf("get first fingerprint = (%q, %v)", got, err)
			}
			if err := s.SetFoodImportFingerprint(ctx(), "usda", "second"); err != nil {
				t.Fatalf("update fingerprint: %v", err)
			}
			if got, err := s.GetFoodImportFingerprint(ctx(), "usda"); err != nil || got != "second" {
				t.Fatalf("get updated fingerprint = (%q, %v)", got, err)
			}
		})
	}
}
