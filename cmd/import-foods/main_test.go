package main

import (
	"context"
	"os"
	"testing"

	"github.com/gsaraiva2109/dietdaemon/adapters/nutrition/taco"
	"github.com/gsaraiva2109/dietdaemon/core/ports"
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
