package main

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/gsaraiva2109/dietdaemon/internal/importers/mfp"
	"github.com/gsaraiva2109/dietdaemon/internal/store"
)

const testCSV = `Date,Meal,Food,Serving Size,Calories,Fat (g),Carbohydrates (g),Fiber,Protein (g)
2024-01-15,Breakfast,Oatmeal,1 cup,150,3,27,4,5
2024-01-15,Breakfast,Banana,1 medium,105,0.4,27,3.1,1.3
2024-01-15,Lunch,Grilled Chicken Breast,6 oz,280,6,0,0,53
`

// tempStore opens a real temp-file SQLite store (store.tempDB isn't exported
// outside package store, so this mirrors cmd/import-foods' helper).
func tempStore(t *testing.T) *store.Store {
	t.Helper()
	f, err := os.CreateTemp("", "import-mfp-test-*.db")
	if err != nil {
		t.Fatalf("create temp file: %v", err)
	}
	path := f.Name()
	_ = f.Close()
	_ = os.Remove(path) // store.New creates it

	st, err := store.New("sqlite", path, store.SQLiteDialect(), nil)
	if err != nil {
		t.Fatalf("store.New: %v", err)
	}
	t.Cleanup(func() {
		_ = st.Close()
		_ = os.Remove(path)
	})
	return st
}

func TestGroupIntoMeals(t *testing.T) {
	rows, err := mfp.ParseCSV(strings.NewReader(testCSV))
	if err != nil {
		t.Fatalf("ParseCSV: %v", err)
	}

	meals, err := groupIntoMeals("user-1", rows, time.UTC)
	if err != nil {
		t.Fatalf("groupIntoMeals: %v", err)
	}
	if len(meals) != 2 {
		t.Fatalf("len(meals) = %d, want 2 (Breakfast, Lunch)", len(meals))
	}

	breakfast := meals[0]
	if len(breakfast.Items) != 2 {
		t.Fatalf("breakfast items = %d, want 2", len(breakfast.Items))
	}
	if breakfast.ExternalID == nil || *breakfast.ExternalID != "mfp:2024-01-15:breakfast" {
		t.Errorf("ExternalID = %v, want mfp:2024-01-15:breakfast", breakfast.ExternalID)
	}
	wantAt := time.Date(2024, 1, 15, 8, 0, 0, 0, time.UTC)
	if !breakfast.At.Equal(wantAt) {
		t.Errorf("At = %v, want %v", breakfast.At, wantAt)
	}

	lunch := meals[1]
	if len(lunch.Items) != 1 {
		t.Fatalf("lunch items = %d, want 1", len(lunch.Items))
	}
	wantLunchAt := time.Date(2024, 1, 15, 12, 0, 0, 0, time.UTC)
	if !lunch.At.Equal(wantLunchAt) {
		t.Errorf("lunch At = %v, want %v", lunch.At, wantLunchAt)
	}
}

func TestImportMeals_IdempotentReRun(t *testing.T) {
	rows, err := mfp.ParseCSV(strings.NewReader(testCSV))
	if err != nil {
		t.Fatalf("ParseCSV: %v", err)
	}
	meals, err := groupIntoMeals("user-1", rows, time.UTC)
	if err != nil {
		t.Fatalf("groupIntoMeals: %v", err)
	}

	st := tempStore(t)
	// SaveMeal enforces a foreign key to users(id) (and users to accounts(id));
	// insert both directly since this test only exercises the import path,
	// not signup.
	if _, err := st.DB().Exec(`INSERT INTO accounts (id, created_at) VALUES ('acct-1', datetime('now'))`); err != nil {
		t.Fatalf("insert test account: %v", err)
	}
	if _, err := st.DB().Exec(`INSERT INTO users (id, account_id, email, status, display_name, timezone, locale, created_at) VALUES ('user-1', 'acct-1', 'u@example.com', 'active', 'U', 'UTC', 'en', datetime('now'))`); err != nil {
		t.Fatalf("insert test user: %v", err)
	}

	ctx := context.Background()
	if _, err := importMeals(ctx, st, meals); err != nil {
		t.Fatalf("importMeals (first run): %v", err)
	}

	var mealCount, itemCount int
	if err := st.DB().QueryRow("SELECT COUNT(*) FROM meals").Scan(&mealCount); err != nil {
		t.Fatalf("count meals: %v", err)
	}
	if err := st.DB().QueryRow("SELECT COUNT(*) FROM resolved_items").Scan(&itemCount); err != nil {
		t.Fatalf("count resolved_items: %v", err)
	}
	if mealCount != 2 {
		t.Fatalf("meals after first run = %d, want 2", mealCount)
	}
	if itemCount != 3 {
		t.Fatalf("resolved_items after first run = %d, want 3", itemCount)
	}

	// Re-running the same import must be a no-op: SaveMeal no-ops on the
	// external_id unique constraint.
	if _, err := importMeals(ctx, st, meals); err != nil {
		t.Fatalf("importMeals (second run): %v", err)
	}

	var mealCount2, itemCount2 int
	if err := st.DB().QueryRow("SELECT COUNT(*) FROM meals").Scan(&mealCount2); err != nil {
		t.Fatalf("count meals after re-run: %v", err)
	}
	if err := st.DB().QueryRow("SELECT COUNT(*) FROM resolved_items").Scan(&itemCount2); err != nil {
		t.Fatalf("count resolved_items after re-run: %v", err)
	}
	if mealCount2 != mealCount {
		t.Fatalf("meals after re-run = %d, want unchanged %d", mealCount2, mealCount)
	}
	if itemCount2 != itemCount {
		t.Fatalf("resolved_items after re-run = %d, want unchanged %d", itemCount2, itemCount)
	}
}
