package localdisk

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/gsaraiva2109/dietdaemon/core/types"
)

func TestWrite_PathTraversalRejected(t *testing.T) {
	tmp := t.TempDir()
	base := filepath.Join(tmp, "backups")
	dst, err := New(base)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	for _, subdir := range []string{"../../etc", "../secrets", "../../../../../../etc"} {
		cfg := types.BackupConfig{UserID: "u1", LocalSubdir: subdir}
		if err := dst.Write(context.Background(), cfg, "meals.csv", []byte("data")); err == nil {
			t.Fatalf("Write with local_subdir %q: expected error, got nil", subdir)
		}
	}

	// Nothing outside tmp should exist as a result of the attempted writes.
	if _, err := os.Stat(filepath.Join(tmp, "etc")); !os.IsNotExist(err) {
		t.Fatalf("expected no file created outside base dir, stat err = %v", err)
	}
	if _, err := os.Stat("/etc/meals.csv"); !os.IsNotExist(err) {
		t.Fatalf("path traversal escaped to real filesystem: /etc/meals.csv exists")
	}
}

func TestWrite_ValidSubdirSucceeds(t *testing.T) {
	tmp := t.TempDir()
	dst, err := New(tmp)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	cfg := types.BackupConfig{UserID: "u1", LocalSubdir: "u1"}
	if err := dst.Write(context.Background(), cfg, "meals.csv", []byte("id,date\n")); err != nil {
		t.Fatalf("Write: %v", err)
	}
	got, err := os.ReadFile(filepath.Join(tmp, "u1", "meals.csv"))
	if err != nil {
		t.Fatalf("expected file written under base dir: %v", err)
	}
	if string(got) != "id,date\n" {
		t.Fatalf("content mismatch: %q", got)
	}
}

func TestWrite_EmptySubdirUsesBase(t *testing.T) {
	tmp := t.TempDir()
	dst, err := New(tmp)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	cfg := types.BackupConfig{UserID: "u1"}
	if err := dst.Write(context.Background(), cfg, "rollups.csv", []byte("date\n")); err != nil {
		t.Fatalf("Write: %v", err)
	}
	if _, err := os.Stat(filepath.Join(tmp, "rollups.csv")); err != nil {
		t.Fatalf("expected file directly under base dir: %v", err)
	}
}

func TestList_ReturnsWrittenFiles(t *testing.T) {
	tmp := t.TempDir()
	dst, err := New(tmp)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	cfg := types.BackupConfig{UserID: "u1", LocalSubdir: "u1"}
	want := []string{"meals.csv", "rollups.csv", "goals.csv"}
	for _, name := range want {
		if err := dst.Write(context.Background(), cfg, name, []byte("data")); err != nil {
			t.Fatalf("Write(%s): %v", name, err)
		}
	}

	got, err := dst.List(context.Background(), cfg)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(got) != len(want) {
		t.Fatalf("List returned %d files, want %d: %v", len(got), len(want), got)
	}
	gotSet := make(map[string]bool, len(got))
	for _, name := range got {
		gotSet[name] = true
	}
	for _, name := range want {
		if !gotSet[name] {
			t.Fatalf("List missing expected file %q, got %v", name, got)
		}
	}
}

func TestList_EmptyForNonexistentSubdir(t *testing.T) {
	tmp := t.TempDir()
	dst, err := New(tmp)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	cfg := types.BackupConfig{UserID: "u1", LocalSubdir: "never-written"}
	got, err := dst.List(context.Background(), cfg)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("expected empty result, got %v", got)
	}
}

func TestRead_RoundTripsWrittenData(t *testing.T) {
	tmp := t.TempDir()
	dst, err := New(tmp)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	cfg := types.BackupConfig{UserID: "u1", LocalSubdir: "u1"}
	want := []byte("id,date\n1,2024-01-01\n")
	if err := dst.Write(context.Background(), cfg, "meals.csv", want); err != nil {
		t.Fatalf("Write: %v", err)
	}

	got, err := dst.Read(context.Background(), cfg, "meals.csv")
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	if string(got) != string(want) {
		t.Fatalf("content mismatch: got %q, want %q", got, want)
	}
}

func TestRead_PathTraversalRejected(t *testing.T) {
	tmp := t.TempDir()
	base := filepath.Join(tmp, "backups")
	dst, err := New(base)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	cfg := types.BackupConfig{UserID: "u1", LocalSubdir: "../../etc"}
	if _, err := dst.Read(context.Background(), cfg, "meals.csv"); err == nil {
		t.Fatalf("Read with local_subdir %q: expected error, got nil", cfg.LocalSubdir)
	}
}
