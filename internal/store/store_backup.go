package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/gsaraiva2109/dietdaemon/core/types"
	"github.com/jmoiron/sqlx"
)

// ---------------------------------------------------------------------------
// Backup / scheduled export
// ---------------------------------------------------------------------------

// GetBackupConfig returns a user's backup settings, or types.ErrNotFound when
// none has been configured (callers treat "not found" as "disabled").
func (s *Store) GetBackupConfig(ctx context.Context, userID string) (types.BackupConfig, error) {
	const q = `
		SELECT user_id, enabled, destination, local_subdir, s3_bucket, s3_prefix, s3_region, s3_endpoint, interval_hrs, last_run_at
		FROM backup_config WHERE user_id = ?
	`
	var row backupConfigRow
	if err := s.db.GetContext(ctx, &row, s.rewrite(q), userID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return types.BackupConfig{}, types.ErrNotFound
		}
		return types.BackupConfig{}, fmt.Errorf("store: get backup config: %w", err)
	}
	return row.toBackupConfig(), nil
}

// backupConfigRow is the flat DB shape of backup_config; types.BackupConfig
// stores Enabled as bool (DB: int) and LastRunAt as time.Time (DB: nullable
// RFC3339 string).
type backupConfigRow struct {
	UserID      string         `db:"user_id"`
	Enabled     int            `db:"enabled"`
	Destination string         `db:"destination"`
	LocalSubdir sql.NullString `db:"local_subdir"`
	S3Bucket    sql.NullString `db:"s3_bucket"`
	S3Prefix    sql.NullString `db:"s3_prefix"`
	S3Region    sql.NullString `db:"s3_region"`
	S3Endpoint  sql.NullString `db:"s3_endpoint"`
	IntervalHrs int            `db:"interval_hrs"`
	LastRunAt   sql.NullString `db:"last_run_at"`
}

func (r backupConfigRow) toBackupConfig() types.BackupConfig {
	cfg := types.BackupConfig{
		UserID:      r.UserID,
		Enabled:     r.Enabled != 0,
		Destination: r.Destination,
		LocalSubdir: r.LocalSubdir.String,
		S3Bucket:    r.S3Bucket.String,
		S3Prefix:    r.S3Prefix.String,
		S3Region:    r.S3Region.String,
		S3Endpoint:  r.S3Endpoint.String,
		IntervalHrs: r.IntervalHrs,
	}
	if r.LastRunAt.Valid && r.LastRunAt.String != "" {
		cfg.LastRunAt = parseUTC(r.LastRunAt.String)
	}
	return cfg
}

// SetBackupConfig inserts or replaces a user's backup settings.
func (s *Store) SetBackupConfig(ctx context.Context, cfg types.BackupConfig) error {
	enabled := 0
	if cfg.Enabled {
		enabled = 1
	}
	const q = `
		INSERT INTO backup_config
			(user_id, enabled, destination, local_subdir, s3_bucket, s3_prefix, s3_region, s3_endpoint, interval_hrs)
		VALUES (:user_id, :enabled, :destination, :local_subdir, :s3_bucket, :s3_prefix, :s3_region, :s3_endpoint, :interval_hrs)
		ON CONFLICT(user_id) DO UPDATE SET
			enabled      = excluded.enabled,
			destination  = excluded.destination,
			local_subdir = excluded.local_subdir,
			s3_bucket    = excluded.s3_bucket,
			s3_prefix    = excluded.s3_prefix,
			s3_region    = excluded.s3_region,
			s3_endpoint  = excluded.s3_endpoint,
			interval_hrs = excluded.interval_hrs
	`
	query, args, err := sqlx.Named(q, map[string]any{
		"user_id": cfg.UserID, "enabled": enabled, "destination": cfg.Destination,
		"local_subdir": cfg.LocalSubdir, "s3_bucket": cfg.S3Bucket, "s3_prefix": cfg.S3Prefix,
		"s3_region": cfg.S3Region, "s3_endpoint": cfg.S3Endpoint, "interval_hrs": cfg.IntervalHrs,
	})
	if err != nil {
		return fmt.Errorf("store: bind set backup config: %w", err)
	}
	_, err = s.db.ExecContext(ctx, s.rewrite(query), args...)
	return err
}

// SetBackupLastRun records when a user's backup last completed.
func (s *Store) SetBackupLastRun(ctx context.Context, userID string, t time.Time) error {
	const q = `UPDATE backup_config SET last_run_at = ? WHERE user_id = ?`
	_, err := s.db.ExecContext(ctx, s.rewrite(q), utcStr(t), userID)
	return err
}
