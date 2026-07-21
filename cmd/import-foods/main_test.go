package main

import (
	"context"
	"os"
	"testing"

	"github.com/gsaraiva2109/dietdaemon/adapters/nutrition/taco"
	"github.com/gsaraiva2109/dietdaemon/core/ports"
	"github.com/gsaraiva2109/dietdaemon/core/types"
	"github.com/gsaraiva2109/dietdaemon/internal/store"
)

// tempStore opens a real temp-file SQLite store (store.tempDB isn't exported
// outside package store, so this mirrors it locally).
func tempStore(t *testing.T) *store.Store {
	t.Helper()
	f, err := os.CreateTemp("", "import-foods-test-*.db")
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

func TestRunImport_TACO(t *testing.T) {
	src, err := taco.New("")
	if err != nil {
		t.Fatalf("taco.New: %v", err)
	}
	st := tempStore(t)

	rows, err := runImport(context.Background(), src, ports.BulkFilter{}, st, false)
	if err != nil {
		t.Fatalf("runImport: %v", err)
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

	// Row count alone previously let a value-mapping bug ship undetected
	// (issue #111: 598/598 TACO rows had correct row counts but shuffled
	// macros). Assert actual per-100g values roundtrip correctly too.
	var kcal, protein, carbs, fat, fiber float64
	err = st.DB().QueryRow(
		`SELECT kcal_100g, protein_100g, carbs_100g, fat_100g, fiber_100g FROM foods WHERE food_id = ?`,
		"TACO558",
	).Scan(&kcal, &protein, &carbs, &fat, &fiber)
	if err != nil {
		t.Fatalf("query TACO558: %v", err)
	}
	if kcal != 606 || protein != 22.5 || carbs != 18.7 || fat != 54.0 || fiber != 7.8 {
		t.Fatalf("TACO558 macros = kcal=%v protein=%v carbs=%v fat=%v fiber=%v, want 606/22.5/18.7/54/7.8",
			kcal, protein, carbs, fat, fiber)
	}
}

func TestRunImport_DryRunWritesNothing(t *testing.T) {
	src, err := taco.New("")
	if err != nil {
		t.Fatalf("taco.New: %v", err)
	}
	st := tempStore(t)

	rows, err := runImport(context.Background(), src, ports.BulkFilter{}, st, true)
	if err != nil {
		t.Fatalf("runImport: %v", err)
	}
	if rows == 0 {
		t.Fatal("expected rows > 0 even in dry-run (rows counted, not written)")
	}

	var count int
	if err := st.DB().QueryRow("SELECT COUNT(*) FROM foods").Scan(&count); err != nil {
		t.Fatalf("count foods: %v", err)
	}
	if count != 0 {
		t.Fatalf("dry-run should write nothing, foods table has %d rows", count)
	}
}

// TestRunRepair_FixesStaleFoodIDRow reproduces issue #111: a catalog row
// written under a legacy food_id ("558" instead of "TACO558") with shuffled
// macros. A normal re-import can't reach it (ON CONFLICT keys on food_id),
// but runRepair matches by (source, name) and must fix it in place without
// changing food_id, so any existing meal_items/food_aliases referencing that
// id stay valid.
func TestRunRepair_FixesStaleFoodIDRow(t *testing.T) {
	st := tempStore(t)

	const name = "Amendoim, torrado, salgado"
	_, err := st.DB().Exec(
		`INSERT INTO foods (food_id, name, source, kcal_100g, protein_100g, carbs_100g, fat_100g, fiber_100g, created_at, updated_at)
		 VALUES ('558', ?, 'taco', 1.7, 606.0, 2535.0, 22.5, 54.0, '2026-01-01T00:00:00Z', '2026-01-01T00:00:00Z')`,
		name,
	)
	if err != nil {
		t.Fatalf("seed stale row: %v", err)
	}

	src, err := taco.New("")
	if err != nil {
		t.Fatalf("taco.New: %v", err)
	}
	var batch []types.FoodMatch
	if err := src.FetchBulk(context.Background(), ports.BulkFilter{}, func(fm types.FoodMatch) error {
		batch = append(batch, fm)
		return nil
	}); err != nil {
		t.Fatalf("FetchBulk: %v", err)
	}

	fixed, err := st.RepairFoodMacros(context.Background(), batch)
	if err != nil {
		t.Fatalf("RepairFoodMacros: %v", err)
	}
	if fixed == 0 {
		t.Fatal("expected at least one row fixed")
	}

	var foodID string
	var kcal, protein, carbs, fat, fiber float64
	err = st.DB().QueryRow(
		`SELECT food_id, kcal_100g, protein_100g, carbs_100g, fat_100g, fiber_100g FROM foods WHERE name = ?`,
		name,
	).Scan(&foodID, &kcal, &protein, &carbs, &fat, &fiber)
	if err != nil {
		t.Fatalf("query repaired row: %v", err)
	}
	if foodID != "558" {
		t.Fatalf("food_id changed to %q, want unchanged '558'", foodID)
	}
	if kcal != 606 || protein != 22.5 || carbs != 18.7 || fat != 54.0 || fiber != 7.8 {
		t.Fatalf("repaired macros = kcal=%v protein=%v carbs=%v fat=%v fiber=%v, want 606/22.5/18.7/54/7.8",
			kcal, protein, carbs, fat, fiber)
	}
}
