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
