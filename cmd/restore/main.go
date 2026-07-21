// Command restore is the CLI counterpart to the scheduled backup runner
// (internal/backup): it reads a backup (local disk or S3) written by
// internal/backup and replays it into a store, idempotently, entity by
// entity, via internal/restore.
//
// This tool is disaster-recovery code: the target -db is expected to be a
// fresh/empty database (or one missing rows you're recovering), so the
// backup config it needs (which destination, bucket, subdir, ...) can never
// come from that database — every setting is taken directly from CLI flags.
//
// Usage:
//
//	go run ./cmd/restore -user <user-id> -db ./data/dietdaemon.db -destination local -dir ./backups -subdir alice
//	go run ./cmd/restore -user <user-id> -db ./data/dietdaemon.db -destination s3 -s3-bucket my-bucket -s3-prefix alice
//	go run ./cmd/restore -user <user-id> -db ./data/dietdaemon.db -destination local -dir ./backups -dry-run
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/gsaraiva2109/dietdaemon/core/types"
	"github.com/gsaraiva2109/dietdaemon/internal/backup/localdisk"
	"github.com/gsaraiva2109/dietdaemon/internal/backup/s3dest"
	"github.com/gsaraiva2109/dietdaemon/internal/restore"
	"github.com/gsaraiva2109/dietdaemon/internal/store"
)

func main() {
	userID := flag.String("user", "", "user ID to restore into (required)")
	dbPath := flag.String("db", "", "SQLite database path for the target store (required)")
	destination := flag.String("destination", "", `backup destination to read from: "local" or "s3" (required)`)
	dir := flag.String("dir", "", "local disk base directory holding the backup (used when -destination=local)")
	subdir := flag.String("subdir", "", "backup subdirectory the user's files live under (BackupConfig.LocalSubdir)")
	s3Bucket := flag.String("s3-bucket", "", "S3 bucket holding the backup (used when -destination=s3)")
	s3Prefix := flag.String("s3-prefix", "", "S3 key prefix the user's files live under (used when -destination=s3)")
	s3Region := flag.String("s3-region", "", "S3 region override (used when -destination=s3)")
	s3Endpoint := flag.String("s3-endpoint", "", "S3-compatible endpoint override, e.g. MinIO (used when -destination=s3)")
	dryRun := flag.Bool("dry-run", false, "list the backup files found without touching the store")
	flag.Parse()

	if *userID == "" || *dbPath == "" || *destination == "" {
		flag.Usage()
		os.Exit(1)
	}
	if *destination != "local" && *destination != "s3" {
		fmt.Fprintf(os.Stderr, "restore: -destination must be \"local\" or \"s3\", got %q\n", *destination)
		os.Exit(1)
	}

	// Disaster recovery can involve a lot of rows/blobs; let ctrl-c stop
	// cleanly rather than killing the process mid-restore, matching
	// cmd/import-mfp.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := run(ctx, *userID, *dbPath, *destination, *dir, *subdir, *s3Bucket, *s3Prefix, *s3Region, *s3Endpoint, *dryRun); err != nil {
		fmt.Fprintf(os.Stderr, "restore: %v\n", err)
		os.Exit(1)
	}
}

func run(ctx context.Context, userID, dbPath, destination, dir, subdir, s3Bucket, s3Prefix, s3Region, s3Endpoint string, dryRun bool) error {
	if destination == "local" && dir == "" {
		return fmt.Errorf("restore: -dir is required for -destination=local")
	}

	cfg := types.BackupConfig{
		UserID:      userID,
		Destination: destination,
		LocalSubdir: subdir,
		S3Bucket:    s3Bucket,
		S3Prefix:    s3Prefix,
		S3Region:    s3Region,
		S3Endpoint:  s3Endpoint,
	}

	var src restore.Source
	switch destination {
	case "local":
		d, err := localdisk.New(dir)
		if err != nil {
			return fmt.Errorf("restore: open local backup dir: %w", err)
		}
		src = d
	case "s3":
		d, err := s3dest.New(ctx)
		if err != nil {
			return fmt.Errorf("restore: init s3: %w", err)
		}
		src = d
	}

	if dryRun {
		files, err := src.List(ctx, cfg)
		if err != nil {
			return fmt.Errorf("restore: list backup files: %w", err)
		}
		for _, f := range files {
			fmt.Println(f)
		}
		fmt.Printf("restore: dry_run=true destination=%s files=%d\n", destination, len(files))
		return nil
	}

	st, err := store.New("sqlite", dbPath, store.SQLiteDialect(), nil)
	if err != nil {
		return fmt.Errorf("restore: open store: %w", err)
	}
	defer func() {
		if cerr := st.Close(); cerr != nil {
			fmt.Fprintf(os.Stderr, "restore: close store: %v\n", cerr)
		}
	}()

	runner := restore.New(st, src)
	sum, rerr := runner.RunOnce(ctx, userID, cfg)

	// Print the summary even on a partial error: RunOnce never aborts early,
	// so a non-nil error here can still carry a mostly-complete Summary,
	// which is exactly the disaster-recovery-friendly behavior operators
	// need to see (rather than silence on any failure at all).
	fmt.Printf("restore: dry_run=false meals=%d rollups=%d weight=%d measurements=%d sleep=%d workouts=%d water=%d fasts=%d photos=%d skipped=%d\n",
		sum.Meals, sum.Rollups, sum.Weight, sum.Measurements, sum.Sleep, sum.Workouts, sum.Water, sum.Fasts, sum.Photos, len(sum.Skipped))

	return rerr
}
