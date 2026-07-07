package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
)

// GetUserAIKey returns the provider and encrypted key for a user. found=false
// when no key is stored (no error).
func (s *Store) GetUserAIKey(ctx context.Context, userID string) (provider string, encKey string, found bool, err error) {
	const q = `SELECT provider, enc_key FROM user_ai_keys WHERE user_id = ?`
	var prov, enc string
	err = s.db.QueryRowContext(ctx, s.rewrite(q), userID).Scan(&prov, &enc)
	if errors.Is(err, sql.ErrNoRows) {
		return "", "", false, nil
	}
	if err != nil {
		return "", "", false, fmt.Errorf("store: get user ai key: %w", err)
	}
	return prov, enc, true, nil
}

// SetUserAIKey upserts a per-user AI key. encKey must already be encrypted
// (AES-256-GCM ciphertext, base64-encoded).
func (s *Store) SetUserAIKey(ctx context.Context, userID, provider, encKey string) error {
	const q = `
		INSERT INTO user_ai_keys (user_id, provider, enc_key)
		VALUES (?, ?, ?)
		ON CONFLICT(user_id) DO UPDATE SET
			provider = excluded.provider,
			enc_key  = excluded.enc_key
	`
	_, err := s.db.ExecContext(ctx, s.rewrite(q), userID, provider, encKey)
	if err != nil {
		return fmt.Errorf("store: set user ai key: %w", err)
	}
	return nil
}

// DeleteUserAIKey removes a user's stored AI key. No error if nothing existed.
func (s *Store) DeleteUserAIKey(ctx context.Context, userID string) error {
	const q = `DELETE FROM user_ai_keys WHERE user_id = ?`
	_, err := s.db.ExecContext(ctx, s.rewrite(q), userID)
	if err != nil {
		return fmt.Errorf("store: delete user ai key: %w", err)
	}
	return nil
}

// Compile-time check: Store satisfies the MealStore's AI-key subset.
var _ interface {
	GetUserAIKey(ctx context.Context, userID string) (provider string, encKey string, found bool, err error)
	SetUserAIKey(ctx context.Context, userID, provider, encKey string) error
	DeleteUserAIKey(ctx context.Context, userID string) error
} = (*Store)(nil)
