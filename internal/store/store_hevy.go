package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
)

// GetUserHevyKey returns the encrypted Hevy API key for a user. found=false
// when no key is stored (no error).
func (s *Store) GetUserHevyKey(ctx context.Context, userID string) (encKey string, found bool, err error) {
	const q = `SELECT enc_key FROM user_hevy_keys WHERE user_id = ?`
	var enc string
	err = s.db.QueryRowContext(ctx, s.rewrite(q), userID).Scan(&enc)
	if errors.Is(err, sql.ErrNoRows) {
		return "", false, nil
	}
	if err != nil {
		return "", false, fmt.Errorf("store: get user hevy key: %w", err)
	}
	return enc, true, nil
}

// SetUserHevyKey upserts a per-user Hevy API key. encKey must already be
// encrypted (AES-256-GCM ciphertext, base64-encoded).
func (s *Store) SetUserHevyKey(ctx context.Context, userID, encKey string) error {
	const q = `
		INSERT INTO user_hevy_keys (user_id, enc_key)
		VALUES (?, ?)
		ON CONFLICT(user_id) DO UPDATE SET enc_key = excluded.enc_key
	`
	_, err := s.db.ExecContext(ctx, s.rewrite(q), userID, encKey)
	if err != nil {
		return fmt.Errorf("store: set user hevy key: %w", err)
	}
	return nil
}

// DeleteUserHevyKey removes a user's stored Hevy API key. No error if nothing existed.
func (s *Store) DeleteUserHevyKey(ctx context.Context, userID string) error {
	const q = `DELETE FROM user_hevy_keys WHERE user_id = ?`
	_, err := s.db.ExecContext(ctx, s.rewrite(q), userID)
	if err != nil {
		return fmt.Errorf("store: delete user hevy key: %w", err)
	}
	return nil
}
