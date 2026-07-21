package main

import (
	"context"
	"os"
	"testing"

	"github.com/gsaraiva2109/dietdaemon/internal/config"
	"github.com/gsaraiva2109/dietdaemon/internal/store"
)

// adminTempStore opens a real temp-file SQLite store, mirroring
// cmd/import-foods/main_test.go's tempStore (store.tempDB isn't exported
// outside package store, and that helper lives in a different `main`
// package so it can't be imported directly).
func adminTempStore(t *testing.T) *store.Store {
	t.Helper()
	f, err := os.CreateTemp("", "admin-import-test-*.db")
	if err != nil {
		t.Fatalf("create temp file: %v", err)
	}
	path := f.Name()
	_ = f.Close()
	_ = os.Remove(path) // store.New creates it

	st, err := store.New("sqlite", path, store.SQLiteDialect())
	if err != nil {
		t.Fatalf("store.New: %v", err)
	}
	t.Cleanup(func() {
		_ = st.Close()
		_ = os.Remove(path)
	})
	return st
}

func TestFoodImportAdmin_ImportSource_TACO(t *testing.T) {
	st := adminTempStore(t)
	admin := &foodImportAdmin{store: st, cfg: &config.Config{}}

	rows, err := admin.ImportSource(context.Background(), "taco", 0)
	if err != nil {
		t.Fatalf("ImportSource: %v", err)
	}
	if rows == 0 {
		t.Fatal("expected rows > 0 from embedded TACO dataset")
	}

	var count int
	if err := st.DB().QueryRow("SELECT COUNT(*) FROM foods").Scan(&count); err != nil {
		t.Fatalf("count foods: %v", err)
	}
	if count != rows {
		t.Fatalf("foods table has %d rows, want %d", count, rows)
	}
}

func TestFoodImportAdmin_ImportSource_MaxRowsCap(t *testing.T) {
	st := adminTempStore(t)
	admin := &foodImportAdmin{store: st, cfg: &config.Config{}}

	rows, err := admin.ImportSource(context.Background(), "taco", 5)
	if err != nil {
		t.Fatalf("ImportSource: %v", err)
	}
	if rows != 5 {
		t.Fatalf("rows = %d, want 5 (max-rows cap)", rows)
	}
}

func TestFoodImportAdmin_RepairSource(t *testing.T) {
	st := adminTempStore(t)

	const name = "Amendoim, torrado, salgado"
	_, err := st.DB().Exec(
		`INSERT INTO foods (food_id, name, source, kcal_100g, protein_100g, carbs_100g, fat_100g, fiber_100g, created_at, updated_at)
		 VALUES ('558', ?, 'taco', 1.7, 606.0, 2535.0, 22.5, 54.0, '2026-01-01T00:00:00Z', '2026-01-01T00:00:00Z')`,
		name,
	)
	if err != nil {
		t.Fatalf("seed stale row: %v", err)
	}

	admin := &foodImportAdmin{store: st, cfg: &config.Config{}}
	checked, fixed, err := admin.RepairSource(context.Background(), "taco")
	if err != nil {
		t.Fatalf("RepairSource: %v", err)
	}
	if checked == 0 {
		t.Fatal("expected checked > 0")
	}
	if fixed == 0 {
		t.Fatal("expected at least one row fixed")
	}

	var foodID string
	var kcal float64
	err = st.DB().QueryRow(
		`SELECT food_id, kcal_100g FROM foods WHERE name = ?`, name,
	).Scan(&foodID, &kcal)
	if err != nil {
		t.Fatalf("query repaired row: %v", err)
	}
	if foodID != "558" {
		t.Fatalf("food_id changed to %q, want unchanged '558'", foodID)
	}
	if kcal != 606 {
		t.Fatalf("kcal = %v, want 606 (repaired)", kcal)
	}
}

func TestFoodImportAdmin_ImportSource_UnknownSource(t *testing.T) {
	st := adminTempStore(t)
	admin := &foodImportAdmin{store: st, cfg: &config.Config{}}

	if _, err := admin.ImportSource(context.Background(), "not-a-real-source", 0); err == nil {
		t.Fatal("expected error for unknown source")
	}
}
