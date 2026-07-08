package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
)

// ProviderKey represents a row in user_provider_keys.
type ProviderKey struct {
	UserID    string `db:"user_id"`
	Provider  string `db:"provider"`
	EncKey    string `db:"enc_key"`
	CreatedAt string `db:"created_at"`
}

// UpsertProviderKey creates or updates an encrypted provider API key. encKey
// must already be encrypted (AES-256-GCM ciphertext, base64-encoded).
func (s *Store) UpsertProviderKey(ctx context.Context, userID, provider, encKey string) error {
	const q = `
		INSERT INTO user_provider_keys (user_id, provider, enc_key)
		VALUES (?, ?, ?)
		ON CONFLICT(user_id, provider) DO UPDATE SET enc_key = excluded.enc_key
	`
	_, err := s.db.ExecContext(ctx, s.rewrite(q), userID, provider, encKey)
	if err != nil {
		return fmt.Errorf("store: upsert provider key: %w", err)
	}
	return nil
}

// GetProviderKey returns the encrypted key for a user+provider. found=false
// when no key is stored (no error).
func (s *Store) GetProviderKey(ctx context.Context, userID, provider string) (encKey string, found bool, err error) {
	const q = `SELECT enc_key FROM user_provider_keys WHERE user_id = ? AND provider = ?`
	var enc string
	err = s.db.QueryRowContext(ctx, s.rewrite(q), userID, provider).Scan(&enc)
	if errors.Is(err, sql.ErrNoRows) {
		return "", false, nil
	}
	if err != nil {
		return "", false, fmt.Errorf("store: get provider key: %w", err)
	}
	return enc, true, nil
}

// DeleteProviderKey removes a stored key for a user+provider. No error if
// nothing existed.
func (s *Store) DeleteProviderKey(ctx context.Context, userID, provider string) error {
	const q = `DELETE FROM user_provider_keys WHERE user_id = ? AND provider = ?`
	_, err := s.db.ExecContext(ctx, s.rewrite(q), userID, provider)
	if err != nil {
		return fmt.Errorf("store: delete provider key: %w", err)
	}
	return nil
}

// ListProviderKeys returns all stored provider keys for a user, ordered by
// provider name.
func (s *Store) ListProviderKeys(ctx context.Context, userID string) ([]ProviderKey, error) {
	const q = `SELECT user_id, provider, enc_key, created_at FROM user_provider_keys WHERE user_id = ? ORDER BY provider`
	var keys []ProviderKey
	if err := s.db.SelectContext(ctx, &keys, s.rewrite(q), userID); err != nil {
		return nil, fmt.Errorf("store: list provider keys: %w", err)
	}
	return keys, nil
}

// ---------------------------------------------------------------------------
// Thin wrappers — backward compatible with old caller method signatures.
// ---------------------------------------------------------------------------

// GetUserAIKey returns the provider and encrypted AI key for a user. found=false
// when no key is stored (no error).
func (s *Store) GetUserAIKey(ctx context.Context, userID string) (provider string, encKey string, found bool, err error) {
	const q = `SELECT provider, enc_key FROM user_provider_keys WHERE user_id = ? AND provider != 'hevy' ORDER BY created_at DESC LIMIT 1`
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
	return s.UpsertProviderKey(ctx, userID, provider, encKey)
}

// DeleteUserAIKey removes stored AI keys for a user (excluding Hevy). No error
// if nothing existed.
func (s *Store) DeleteUserAIKey(ctx context.Context, userID string) error {
	const q = `DELETE FROM user_provider_keys WHERE user_id = ? AND provider != 'hevy'`
	_, err := s.db.ExecContext(ctx, s.rewrite(q), userID)
	if err != nil {
		return fmt.Errorf("store: delete user ai key: %w", err)
	}
	return nil
}

// GetUserHevyKey returns the encrypted Hevy API key for a user. found=false
// when no key is stored (no error).
func (s *Store) GetUserHevyKey(ctx context.Context, userID string) (encKey string, found bool, err error) {
	return s.GetProviderKey(ctx, userID, "hevy")
}

// SetUserHevyKey upserts a per-user Hevy API key. encKey must already be
// encrypted (AES-256-GCM ciphertext, base64-encoded).
func (s *Store) SetUserHevyKey(ctx context.Context, userID, encKey string) error {
	return s.UpsertProviderKey(ctx, userID, "hevy", encKey)
}

// DeleteUserHevyKey removes a user's stored Hevy API key. No error if nothing
// existed.
func (s *Store) DeleteUserHevyKey(ctx context.Context, userID string) error {
	return s.DeleteProviderKey(ctx, userID, "hevy")
}
