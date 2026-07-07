package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/gsaraiva2109/dietdaemon/core/types"
	"github.com/jmoiron/sqlx"
)

// ---------------------------------------------------------------------------
// Users
// ---------------------------------------------------------------------------

// UpsertUser inserts or updates a user row. New auth columns are set via
// separate auth-dedicated methods (CreateUserWithPassword); this method
// preserves the existing id/timezone/created_at contract for the pipeline.
func (s *Store) UpsertUser(ctx context.Context, u types.User) error {
	const q = `
		INSERT INTO users (id, account_id, email, email_verified_at, status, display_name, timezone, created_at)
		VALUES (:id, :account_id, :email, :email_verified_at, :status, :display_name, :timezone, :created_at)
		ON CONFLICT(id) DO UPDATE SET timezone = excluded.timezone
	`
	var emailVerifiedAt any
	if u.EmailVerifiedAt != nil {
		emailVerifiedAt = utcStr(*u.EmailVerifiedAt)
	}
	query, args, err := sqlx.Named(q, map[string]any{
		"id":                u.ID,
		"account_id":        nullStr(u.AccountID),
		"email":             nullStr(u.Email),
		"email_verified_at": emailVerifiedAt,
		"status":            u.Status,
		"display_name":      nullStr(u.DisplayName),
		"timezone":          u.Timezone,
		"created_at":        utcStr(u.CreatedAt),
	})
	if err != nil {
		return fmt.Errorf("store: bind upsert user: %w", err)
	}
	_, err = s.db.ExecContext(ctx, s.rewrite(query), args...)
	return err
}

// GetUser returns the user or types.ErrNotFound.
func (s *Store) GetUser(ctx context.Context, userID string) (types.User, error) {
	const q = `SELECT id, account_id, email, email_verified_at, status, display_name, timezone, created_at, webauthn_handle FROM users WHERE id = ?`
	var row userRow
	if err := s.db.GetContext(ctx, &row, s.rewrite(q), userID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return types.User{}, types.ErrNotFound
		}
		return types.User{}, err
	}
	return row.toUser(), nil
}

// ListUsers returns every user. Empty slice, nil error when there are none.
func (s *Store) ListUsers(ctx context.Context) ([]types.User, error) {
	const q = `SELECT id, account_id, email, email_verified_at, status, display_name, timezone, created_at, webauthn_handle FROM users ORDER BY id`
	var rows []userRow
	if err := s.db.SelectContext(ctx, &rows, q); err != nil {
		return nil, fmt.Errorf("store: list users: %w", err)
	}
	var users []types.User
	for _, r := range rows {
		users = append(users, r.toUser())
	}
	return users, nil
}

// userRow is the flat DB shape of the users table; the public types.User
// nests EmailVerifiedAt as *time.Time and applies a default status, neither
// of which maps 1:1 onto a column.
type userRow struct {
	ID              string         `db:"id"`
	AccountID       sql.NullString `db:"account_id"`
	Email           sql.NullString `db:"email"`
	EmailVerifiedAt sql.NullString `db:"email_verified_at"`
	Status          sql.NullString `db:"status"`
	DisplayName     sql.NullString `db:"display_name"`
	Timezone        string         `db:"timezone"`
	CreatedAt       string         `db:"created_at"`
	WebAuthnHandle  sql.NullString `db:"webauthn_handle"`
}

func (r userRow) toUser() types.User {
	u := types.User{
		ID:             r.ID,
		AccountID:      r.AccountID.String,
		Email:          r.Email.String,
		DisplayName:    r.DisplayName.String,
		Status:         r.Status.String,
		Timezone:       r.Timezone,
		CreatedAt:      parseUTC(r.CreatedAt),
		WebAuthnHandle: r.WebAuthnHandle.String,
	}
	if r.EmailVerifiedAt.Valid {
		u.EmailVerifiedAt = new(parseUTC(r.EmailVerifiedAt.String))
	}
	if !r.Status.Valid {
		u.Status = "active"
	}
	return u
}

// ValidateToken looks up a Bearer token in the api_tokens table and returns the
// owning userID. Returns types.ErrNotFound when the token is invalid or expired.
// In single-user mode this method is not called; the static API_AUTH_TOKEN is
// checked directly.

// UpsertUserTimezone updates the users.timezone column for a user.
func (s *Store) UpsertUserTimezone(ctx context.Context, userID, timezone string) error {
	const q = `UPDATE users SET timezone = ? WHERE id = ?`
	_, err := s.db.ExecContext(ctx, s.rewrite(q), timezone, userID)
	return err
}

// MapChannelUser inserts a mapping from a messaging channel + channel_user_id
// to an internal user_id. It is idempotent (INSERT OR IGNORE).
func (s *Store) MapChannelUser(ctx context.Context, channel, channelUserID, userID string) error {
	const q = `
		INSERT INTO user_channels (channel, channel_user_id, user_id)
		VALUES (?, ?, ?)
		ON CONFLICT DO NOTHING
	`
	_, err := s.db.ExecContext(ctx, s.rewrite(q), channel, channelUserID, userID)
	return err
}

// GetUserIDByChannel returns the internal user_id for a given
// (channel, channel_user_id) pair. Returns types.ErrNotFound when no mapping
// exists.
func (s *Store) GetUserIDByChannel(ctx context.Context, channel, channelUserID string) (string, error) {
	const q = `SELECT user_id FROM user_channels WHERE channel = ? AND channel_user_id = ?`
	var userID string
	if err := s.db.GetContext(ctx, &userID, s.rewrite(q), channel, channelUserID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", types.ErrNotFound
		}
		return "", fmt.Errorf("store: get user by channel: %w", err)
	}
	return userID, nil
}

// UpsertChatRoute records the chat metadata needed to reach a user
// proactively (e.g. from the scheduler), refreshed on every inbound message.
// One row per (user, channel).
func (s *Store) UpsertChatRoute(ctx context.Context, userID, channel string, meta map[string]string) error {
	metaJSON, err := json.Marshal(meta)
	if err != nil {
		return fmt.Errorf("store: marshal chat route meta: %w", err)
	}
	const q = `
		INSERT INTO chat_routes (user_id, channel, meta_json, updated_at)
		VALUES (?, ?, ?, ?)
		ON CONFLICT(user_id, channel) DO UPDATE SET
			meta_json  = excluded.meta_json,
			updated_at = excluded.updated_at
	`
	_, err = s.db.ExecContext(ctx, s.rewrite(q), userID, channel, string(metaJSON), utcNow())
	if err != nil {
		return fmt.Errorf("store: upsert chat route: %w", err)
	}
	return nil
}

// GetChatRoute returns the most recently seen channel + delivery metadata for
// a user, so the scheduler can send a message through a MessagingAdapter
// instead of only the plain-text Notifier. Returns types.ErrNotFound when the
// user has never been seen on any channel.
func (s *Store) GetChatRoute(ctx context.Context, userID string) (string, map[string]string, error) {
	const q = `SELECT channel, meta_json FROM chat_routes WHERE user_id = ? ORDER BY updated_at DESC LIMIT 1`
	var row struct {
		Channel  string `db:"channel"`
		MetaJSON string `db:"meta_json"`
	}
	if err := s.db.GetContext(ctx, &row, s.rewrite(q), userID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", nil, types.ErrNotFound
		}
		return "", nil, fmt.Errorf("store: get chat route: %w", err)
	}
	var meta map[string]string
	if err := json.Unmarshal([]byte(row.MetaJSON), &meta); err != nil {
		return "", nil, fmt.Errorf("store: unmarshal chat route meta: %w", err)
	}
	return row.Channel, meta, nil
}
