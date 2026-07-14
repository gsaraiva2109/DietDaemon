package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/gsaraiva2109/dietdaemon/core/types"
)

// GetFoodImportFingerprint returns the last successfully imported fingerprint
// for source, or types.ErrNotFound when the source has not completed an import
// yet.
func (s *Store) GetFoodImportFingerprint(ctx context.Context, source string) (string, error) {
	const q = `SELECT fingerprint FROM food_import_fingerprints WHERE source = ?`
	var fingerprint string
	err := s.db.QueryRowContext(ctx, s.rewrite(q), source).Scan(&fingerprint)
	if errors.Is(err, sql.ErrNoRows) {
		return "", types.ErrNotFound
	}
	if err != nil {
		return "", fmt.Errorf("store: get food import fingerprint: %w", err)
	}
	return fingerprint, nil
}

// SetFoodImportFingerprint records source's most recently successful import.
func (s *Store) SetFoodImportFingerprint(ctx context.Context, source, fingerprint string) error {
	const q = `
		INSERT INTO food_import_fingerprints (source, fingerprint)
		VALUES (?, ?)
		ON CONFLICT(source) DO UPDATE SET fingerprint = excluded.fingerprint
	`
	if _, err := s.db.ExecContext(ctx, s.rewrite(q), source, fingerprint); err != nil {
		return fmt.Errorf("store: set food import fingerprint: %w", err)
	}
	return nil
}

// SetFoodImportStatus records source's last run outcome. Deliberately
// never touches the fingerprint column -- a source without a local file (no
// fingerprint concept at all) still gets a status row, seeded with an empty
// fingerprint placeholder that a real SetFoodImportFingerprint call (if any)
// fills in independently.
func (s *Store) SetFoodImportStatus(ctx context.Context, source, result, lastError string) error {
	const q = `
		INSERT INTO food_import_fingerprints (source, fingerprint, last_result, last_run_at, last_error)
		VALUES (?, '', ?, ?, ?)
		ON CONFLICT(source) DO UPDATE SET
			last_result = excluded.last_result,
			last_run_at = excluded.last_run_at,
			last_error  = excluded.last_error
	`
	if _, err := s.db.ExecContext(ctx, s.rewrite(q), source, result, utcNow(), nullStr(lastError)); err != nil {
		return fmt.Errorf("store: set food import status: %w", err)
	}
	return nil
}

// GetFoodImportStatuses returns the last recorded run outcome for every
// source that has one, most-recent first. A source that has never run (no
// row yet) is simply absent -- this is a status feed, not a source registry.
func (s *Store) GetFoodImportStatuses(ctx context.Context) ([]types.FoodImportStatus, error) {
	const q = `
		SELECT source, fingerprint, last_result, last_run_at, last_error
		FROM food_import_fingerprints
		WHERE last_result IS NOT NULL
		ORDER BY last_run_at DESC
	`
	rows, err := s.db.QueryContext(ctx, s.rewrite(q))
	if err != nil {
		return nil, fmt.Errorf("store: get food import statuses: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var out []types.FoodImportStatus
	for rows.Next() {
		var st types.FoodImportStatus
		var runAt string
		var lastErr sql.NullString
		if err := rows.Scan(&st.Source, &st.Fingerprint, &st.LastResult, &runAt, &lastErr); err != nil {
			return nil, fmt.Errorf("store: scan food import status: %w", err)
		}
		st.LastRunAt = parseUTC(runAt)
		st.LastError = lastErr.String
		out = append(out, st)
	}
	return out, rows.Err()
}
