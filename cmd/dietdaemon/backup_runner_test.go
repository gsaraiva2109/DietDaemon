package main

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/gsaraiva2109/dietdaemon/internal/config"
)

func TestNewBackupRunner_NoLocalDir(t *testing.T) {
	st := adminTempStore(t)
	cfg := &config.Config{BackupCheckInterval: time.Hour}

	runner, localDst, _, err := newBackupRunner(context.Background(), cfg, st)
	if err != nil {
		t.Fatalf("newBackupRunner: %v", err)
	}
	if runner == nil {
		t.Fatal("expected non-nil runner")
	}
	if localDst != nil {
		t.Fatalf("localDst = %v, want nil when BackupLocalDir is empty", localDst)
	}
}

func TestNewBackupRunner_WithLocalDir(t *testing.T) {
	st := adminTempStore(t)
	dir := filepath.Join(t.TempDir(), "backups")
	cfg := &config.Config{BackupLocalDir: dir, BackupCheckInterval: time.Hour}

	runner, localDst, _, err := newBackupRunner(context.Background(), cfg, st)
	if err != nil {
		t.Fatalf("newBackupRunner: %v", err)
	}
	if runner == nil {
		t.Fatal("expected non-nil runner")
	}
	if localDst == nil {
		t.Fatal("expected non-nil localDst when BackupLocalDir is set")
	}
}
