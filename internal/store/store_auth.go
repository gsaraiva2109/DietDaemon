package store

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/gsaraiva2109/dietdaemon/internal/auth"

	"github.com/gsaraiva2109/dietdaemon/core/types"
)

// ---------------------------------------------------------------------------
// Auth persistence — accounts, users with passwords, sessions, API keys,
// login attempts, and audit events. All SQL is portable (ANSIIsh, ON CONFLICT
// upserts, TEXT timestamps). No SQLite-isms.
// ---------------------------------------------------------------------------

// --- Accounts + users with credentials ---

// CreateAccount inserts an account row. Idempotent (ON CONFLICT).
func (s *Store) CreateAccount(ctx context.Context, id string) error {
	const q = `INSERT INTO accounts (id, created_at) VALUES (?, ?) ON CONFLICT DO NOTHING`
	_, err := s.db.ExecContext(ctx, q, id, utcNow())
	return err
}

// CreateUserWithPassword creates an account (if needed), inserts the user
// row into users, and stores the password_credentials row — all in one
// transaction. The email is already lowercased by the caller.
func (s *Store) CreateUserWithPassword(ctx context.Context, accountID, userID, email, displayName, phcHash string) (types.User, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return types.User{}, fmt.Errorf("store: create user tx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	// Ensure the account exists.
	if _, err := tx.ExecContext(ctx,
		`INSERT INTO accounts (id, created_at) VALUES (?, ?) ON CONFLICT DO NOTHING`,
		accountID, utcNow(),
	); err != nil {
		return types.User{}, fmt.Errorf("store: insert account: %w", err)
	}

	now := utcNow()
	u := types.User{
		ID:          userID,
		AccountID:   accountID,
		Email:       email,
		Status:      "active",
		DisplayName: displayName,
		Timezone:    "UTC",
		CreatedAt:   time.Now().UTC(),
	}

	if _, err := tx.ExecContext(ctx,
		`INSERT INTO users (id, account_id, email, email_verified_at, status, display_name, timezone, created_at)
		 VALUES (?, ?, ?, NULL, ?, ?, ?, ?)`,
		u.ID, u.AccountID, u.Email, u.Status, nullStr(u.DisplayName), u.Timezone, now,
	); err != nil {
		return types.User{}, fmt.Errorf("store: insert user: %w", err)
	}

	if _, err := tx.ExecContext(ctx,
		`INSERT INTO password_credentials (user_id, phc_hash, updated_at) VALUES (?, ?, ?)`,
		u.ID, phcHash, now,
	); err != nil {
		return types.User{}, fmt.Errorf("store: insert password: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return types.User{}, fmt.Errorf("store: commit user: %w", err)
	}

	return u, nil
}

// GetUserByEmail returns the user for a given lowercase email. Returns
// types.ErrNotFound when no user matches.
func (s *Store) GetUserByEmail(ctx context.Context, email string) (types.User, error) {
	const q = `SELECT id, account_id, email, email_verified_at, status, display_name, timezone, created_at
		FROM users WHERE email = ?`
	row := s.db.QueryRowContext(ctx, q, email)
	u, err := scanUser(row)
	if err == sql.ErrNoRows {
		return types.User{}, types.ErrNotFound
	}
	return u, err
}

// GetPasswordHash returns the PHC string for userID, or types.ErrNotFound.
func (s *Store) GetPasswordHash(ctx context.Context, userID string) (string, error) {
	const q = `SELECT phc_hash FROM password_credentials WHERE user_id = ?`
	row := s.db.QueryRowContext(ctx, q, userID)
	var hash string
	if err := row.Scan(&hash); err == sql.ErrNoRows {
		return "", types.ErrNotFound
	} else if err != nil {
		return "", fmt.Errorf("store: get password hash: %w", err)
	}
	return hash, nil
}

// SetPasswordHash updates (or creates) the password_credentials row.
func (s *Store) SetPasswordHash(ctx context.Context, userID, phcHash string) error {
	const q = `INSERT INTO password_credentials (user_id, phc_hash, updated_at)
		VALUES (?, ?, ?) ON CONFLICT(user_id) DO UPDATE SET phc_hash = excluded.phc_hash, updated_at = excluded.updated_at`
	_, err := s.db.ExecContext(ctx, q, userID, phcHash, utcNow())
	return err
}

// CountUsers returns the total number of user rows. Used for invite-mode
// bootstrap (only first user may register).
func (s *Store) CountUsers(ctx context.Context) (int, error) {
	const q = `SELECT COUNT(*) FROM users`
	row := s.db.QueryRowContext(ctx, q)
	var n int
	if err := row.Scan(&n); err != nil {
		return 0, fmt.Errorf("store: count users: %w", err)
	}
	return n, nil
}

// GetUserByAPIKey resolves a hashed API key to its owning user. Touches
// last_used_at on success. Skips revoked keys. Returns ErrNotFound when
// the key is invalid or revoked.
func (s *Store) GetUserByAPIKey(ctx context.Context, hashedKey string) (types.User, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return types.User{}, fmt.Errorf("store: api key lookup tx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	// Find the key (must not be revoked).
	const keyQ = `SELECT user_id FROM api_keys WHERE hashed_key = ? AND revoked_at IS NULL`
	var userID string
	if err := tx.QueryRowContext(ctx, keyQ, hashedKey).Scan(&userID); err == sql.ErrNoRows {
		return types.User{}, types.ErrNotFound
	} else if err != nil {
		return types.User{}, fmt.Errorf("store: lookup api key: %w", err)
	}

	// Touch last_used_at.
	_, _ = tx.ExecContext(ctx, `UPDATE api_keys SET last_used_at = ? WHERE hashed_key = ?`, utcNow(), hashedKey)

	const userQ = `SELECT id, account_id, email, email_verified_at, status, display_name, timezone, created_at
		FROM users WHERE id = ?`
	row := tx.QueryRowContext(ctx, userQ, userID)
	u, err := scanUser(row)
	if err == sql.ErrNoRows {
		return types.User{}, types.ErrNotFound
	}
	if err != nil {
		return types.User{}, fmt.Errorf("store: get user by api key: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return types.User{}, fmt.Errorf("store: commit api key lookup: %w", err)
	}
	return u, nil
}

// --- Sessions (implements auth.SessionRepo) ---

func (s *Store) CreateSession(ctx context.Context, sess auth.Session) error {
	remember := 0
	if sess.Remember {
		remember = 1
	}
	const q = `INSERT INTO sessions
		(id, user_id, csrf_token, created_at, last_seen_at, idle_expires_at, absolute_expires_at, remember, ip, user_agent)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`
	_, err := s.db.ExecContext(ctx, q,
		sess.ID, sess.UserID, sess.CSRFToken,
		utcStr(sess.CreatedAt), utcStr(sess.LastSeenAt),
		utcStr(sess.IdleExpiresAt), utcStr(sess.AbsoluteExpiresAt),
		remember, nullStr(sess.IP), nullStr(sess.UserAgent),
	)
	return err
}

func (s *Store) GetSession(ctx context.Context, id string) (auth.Session, error) {
	const q = `SELECT id, user_id, csrf_token, created_at, last_seen_at,
		idle_expires_at, absolute_expires_at, remember, ip, user_agent
		FROM sessions WHERE id = ?`
	row := s.db.QueryRowContext(ctx, q, id)
	return scanSession(row)
}

func (s *Store) TouchSession(ctx context.Context, id string, lastSeen, idleExpires time.Time) error {
	const q = `UPDATE sessions SET last_seen_at = ?, idle_expires_at = ? WHERE id = ?`
	_, err := s.db.ExecContext(ctx, q, utcStr(lastSeen), utcStr(idleExpires), id)
	return err
}

// #nosec G202 — id is generated by the app (SHA-256 hex), not user input
func (s *Store) DeleteSession(ctx context.Context, id string) error {
	const q = `DELETE FROM sessions WHERE id = ?`
	_, err := s.db.ExecContext(ctx, q, id)
	return err
}

func (s *Store) DeleteUserSessions(ctx context.Context, userID string) error {
	const q = `DELETE FROM sessions WHERE user_id = ?`
	_, err := s.db.ExecContext(ctx, q, userID)
	return err
}

func scanSession(row *sql.Row) (auth.Session, error) {
	var s auth.Session
	var ca, lsa, iea, aea, ip, ua string
	var remember int
	if err := row.Scan(&s.ID, &s.UserID, &s.CSRFToken, &ca, &lsa, &iea, &aea, &remember, &ip, &ua); err == sql.ErrNoRows {
		return auth.Session{}, fmt.Errorf("store: session not found: %w", err)
	} else if err != nil {
		return auth.Session{}, fmt.Errorf("store: scan session: %w", err)
	}
	s.CreatedAt = parseUTC(ca)
	s.LastSeenAt = parseUTC(lsa)
	s.IdleExpiresAt = parseUTC(iea)
	s.AbsoluteExpiresAt = parseUTC(aea)
	s.Remember = remember != 0
	s.IP = ip
	s.UserAgent = ua
	return s, nil
}

// --- API keys ---

// CreateAPIKey inserts a new machine API key.
func (s *Store) CreateAPIKey(ctx context.Context, id, userID, hashedKey, label string) error {
	const q = `INSERT INTO api_keys (id, user_id, hashed_key, label, created_at)
		VALUES (?, ?, ?, ?, ?)`
	_, err := s.db.ExecContext(ctx, q, id, userID, hashedKey, label, utcNow())
	return err
}

// ListAPIKeys returns all non-revoked API keys for a user.
func (s *Store) ListAPIKeys(ctx context.Context, userID string) ([]types.APIKey, error) {
	const q = `SELECT id, user_id, label, created_at, last_used_at, revoked_at
		FROM api_keys WHERE user_id = ? AND revoked_at IS NULL ORDER BY created_at DESC`
	rows, err := s.db.QueryContext(ctx, q, userID)
	if err != nil {
		return nil, fmt.Errorf("store: list api keys: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var out []types.APIKey
	for rows.Next() {
		var k types.APIKey
		var ca, lua, ra string
		if err := rows.Scan(&k.ID, &k.UserID, &k.Label, &ca, &lua, &ra); err != nil {
			return nil, fmt.Errorf("store: scan api key: %w", err)
		}
		k.CreatedAt = parseUTC(ca)
		if lua != "" {
			t := parseUTC(lua)
			k.LastUsedAt = &t
		}
		if ra != "" {
			t := parseUTC(ra)
			k.RevokedAt = &t
		}
		out = append(out, k)
	}
	return out, rows.Err()
}

// RevokeAPIKey marks an API key as revoked. Returns ErrNotFound if the key
// does not exist or is already revoked.
func (s *Store) RevokeAPIKey(ctx context.Context, userID, keyID string) error {
	const q = `UPDATE api_keys SET revoked_at = ? WHERE id = ? AND user_id = ? AND revoked_at IS NULL`
	res, err := s.db.ExecContext(ctx, q, utcNow(), keyID, userID)
	if err != nil {
		return fmt.Errorf("store: revoke api key: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return types.ErrNotFound
	}
	return nil
}

// --- Login attempts ---

func (s *Store) RecordLoginAttempt(ctx context.Context, identifier string, succeeded bool) error {
	success := 0
	if succeeded {
		success = 1
	}
	const q = `INSERT INTO login_attempts (id, identifier, succeeded, created_at) VALUES (?, ?, ?, ?)`
	_, err := s.db.ExecContext(ctx, q, newID(), identifier, success, utcNow())
	return err
}

func (s *Store) RecentFailedAttempts(ctx context.Context, identifier string, since time.Time) (int, error) {
	const q = `SELECT COUNT(*) FROM login_attempts
		WHERE identifier = ? AND succeeded = 0 AND created_at > ?`
	row := s.db.QueryRowContext(ctx, q, identifier, utcStr(since))
	var n int
	if err := row.Scan(&n); err != nil {
		return 0, fmt.Errorf("store: recent failed attempts: %w", err)
	}
	return n, nil
}

// --- Audit ---

func (s *Store) WriteAuditEvent(ctx context.Context, ev types.AuditEvent) error {
	const q = `INSERT INTO auth_audit_log (id, account_id, user_id, event, ip, user_agent, meta, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)`
	_, err := s.db.ExecContext(ctx, q,
		ev.ID, nullStr(ev.AccountID), nullStr(ev.UserID), ev.Event,
		nullStr(ev.IP), nullStr(ev.UserAgent), nullStr(ev.Meta), utcStr(ev.CreatedAt),
	)
	return err
}

// --- TOTP secrets ---

// UpsertTOTPSecret inserts or updates the encrypted TOTP secret for a user.
// The secret is already AES-256-GCM encrypted by the caller.
func (s *Store) UpsertTOTPSecret(ctx context.Context, userID, encSecret string) error {
	const q = `INSERT INTO totp_secrets (user_id, secret) VALUES (?, ?)
		ON CONFLICT(user_id) DO UPDATE SET secret = excluded.secret`
	_, err := s.db.ExecContext(ctx, q, userID, encSecret)
	return err
}

// ConfirmTOTP marks the TOTP secret as confirmed (enrollment verified).
func (s *Store) ConfirmTOTP(ctx context.Context, userID string) error {
	const q = `UPDATE totp_secrets SET confirmed_at = ? WHERE user_id = ?`
	_, err := s.db.ExecContext(ctx, q, utcNow(), userID)
	return err
}

// GetTOTPSecret returns the encrypted secret and whether enrollment was
// confirmed. Returns types.ErrNotFound when no secret exists.
func (s *Store) GetTOTPSecret(ctx context.Context, userID string) (secret string, confirmed bool, err error) {
	const q = `SELECT secret, confirmed_at FROM totp_secrets WHERE user_id = ?`
	row := s.db.QueryRowContext(ctx, q, userID)
	var ca sql.NullString
	if err := row.Scan(&secret, &ca); err == sql.ErrNoRows {
		return "", false, types.ErrNotFound
	} else if err != nil {
		return "", false, fmt.Errorf("store: get totp secret: %w", err)
	}
	return secret, ca.Valid, nil
}

// DeleteTOTP removes the TOTP secret for a user.
func (s *Store) DeleteTOTP(ctx context.Context, userID string) error {
	const q = `DELETE FROM totp_secrets WHERE user_id = ?`
	_, err := s.db.ExecContext(ctx, q, userID)
	return err
}

// HasConfirmedTOTP reports whether the user has a confirmed TOTP factor.
func (s *Store) HasConfirmedTOTP(ctx context.Context, userID string) (bool, error) {
	const q = `SELECT COUNT(*) FROM totp_secrets WHERE user_id = ? AND confirmed_at IS NOT NULL`
	row := s.db.QueryRowContext(ctx, q, userID)
	var n int
	if err := row.Scan(&n); err != nil {
		return false, fmt.Errorf("store: has confirmed totp: %w", err)
	}
	return n > 0, nil
}

// --- Recovery codes ---

// ReplaceRecoveryCodes deletes all existing recovery codes for the user and
// inserts new ones in a single transaction.
func (s *Store) ReplaceRecoveryCodes(ctx context.Context, userID string, hashes []string) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("store: replace recovery codes tx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	if _, err := tx.ExecContext(ctx, `DELETE FROM recovery_codes WHERE user_id = ?`, userID); err != nil {
		return fmt.Errorf("store: delete old recovery codes: %w", err)
	}

	now := utcNow()
	for _, h := range hashes {
		if _, err := tx.ExecContext(ctx,
			`INSERT INTO recovery_codes (id, user_id, code_hash, created_at) VALUES (?, ?, ?, ?)`,
			newID(), userID, h, now,
		); err != nil {
			return fmt.Errorf("store: insert recovery code: %w", err)
		}
	}

	return tx.Commit()
}

// ConsumeRecoveryCode marks a recovery code as used (sets used_at). Returns
// true if the code existed and was unused, false otherwise.
func (s *Store) ConsumeRecoveryCode(ctx context.Context, userID, hash string) (bool, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return false, fmt.Errorf("store: consume recovery code tx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	const q = `UPDATE recovery_codes SET used_at = ?
		WHERE user_id = ? AND code_hash = ? AND used_at IS NULL`
	res, err := tx.ExecContext(ctx, q, utcNow(), userID, hash)
	if err != nil {
		return false, fmt.Errorf("store: consume recovery code: %w", err)
	}

	n, _ := res.RowsAffected()
	if n == 0 {
		return false, tx.Rollback()
	}

	return true, tx.Commit()
}

// --- MFA challenges ---

// CreateMFAChallenge inserts a hashed MFA challenge token.
func (s *Store) CreateMFAChallenge(ctx context.Context, id, userID string, remember bool, expiresAt string) error {
	rem := 0
	if remember {
		rem = 1
	}
	const q = `INSERT INTO mfa_challenges (id, user_id, remember, expires_at, created_at)
		VALUES (?, ?, ?, ?, ?)`
	_, err := s.db.ExecContext(ctx, q, id, userID, rem, expiresAt, utcNow())
	return err
}

// GetMFAChallenge retrieves a challenge by its hashed token ID.
// Returns types.ErrNotFound when the challenge does not exist.
func (s *Store) GetMFAChallenge(ctx context.Context, id string) (userID string, remember bool, expiresAt string, err error) {
	const q = `SELECT user_id, remember, expires_at FROM mfa_challenges WHERE id = ?`
	row := s.db.QueryRowContext(ctx, q, id)
	var rem int
	if err := row.Scan(&userID, &rem, &expiresAt); err == sql.ErrNoRows {
		return "", false, "", types.ErrNotFound
	} else if err != nil {
		return "", false, "", fmt.Errorf("store: get mfa challenge: %w", err)
	}
	return userID, rem != 0, expiresAt, nil
}

// DeleteMFAChallenge removes a challenge by its hashed token ID.
func (s *Store) DeleteMFAChallenge(ctx context.Context, id string) error {
	const q = `DELETE FROM mfa_challenges WHERE id = ?`
	_, err := s.db.ExecContext(ctx, q, id)
	return err
}

// Compile-time checks that *Store satisfies the auth interfaces.
var (
	_ auth.SessionRepo      = (*Store)(nil)
	_ auth.LoginAttemptRepo = (*Store)(nil)
	_ auth.TOTPRepo         = (*Store)(nil)
	_ auth.MFAChallengeRepo = (*Store)(nil)
	_ auth.RecoveryCodeRepo = (*Store)(nil)
)
