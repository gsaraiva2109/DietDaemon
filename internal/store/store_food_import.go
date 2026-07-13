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
