package store

import (
	"context"
	"database/sql"
	"encoding/json"
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
	const q = `SELECT id, account_id, email, email_verified_at, status, display_name, timezone, locale, created_at, webauthn_handle
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

	const userQ = `SELECT id, account_id, email, email_verified_at, status, display_name, timezone, locale, created_at, webauthn_handle
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

// --- API keys & share tokens ---
//
// Both are revocable, user-owned credentials stored in tables with the same
// shape: (id, user_id, <secret column>, label, created_at, last_used_at,
// revoked_at). The only differences are the table name and the secret
// column name, so create/list/revoke share one implementation below;
// CreateAPIKey/ListAPIKeys/RevokeAPIKey and CreateShareToken/ListShareTokens/
// RevokeShareToken just plug in those two names and convert the result to
// their own named type.

// credRow mirrors the common row shape of api_keys and share_tokens. It
// converts directly to types.APIKey or types.ShareToken (identical fields).
type credRow struct {
	ID         string
	UserID     string
	Label      string
	CreatedAt  time.Time
	LastUsedAt *time.Time
	RevokedAt  *time.Time
}

// createCred inserts a new row into table (api_keys or share_tokens).
// secretCol is the column holding the hashed secret ("hashed_key" or
// "hashed_token").
func (s *Store) createCred(ctx context.Context, table, secretCol, id, userID, hashedSecret, label string) error {
	q := fmt.Sprintf(`INSERT INTO %s (id, user_id, %s, label, created_at) VALUES (?, ?, ?, ?, ?)`, table, secretCol)
	_, err := s.db.ExecContext(ctx, q, id, userID, hashedSecret, label, utcNow())
	return err
}

// listCreds returns all non-revoked rows from table for userID.
func (s *Store) listCreds(ctx context.Context, table, userID string) ([]credRow, error) {
	q := fmt.Sprintf(`SELECT id, user_id, label, created_at, last_used_at, revoked_at
		FROM %s WHERE user_id = ? AND revoked_at IS NULL ORDER BY created_at DESC`, table)
	rows, err := s.db.QueryContext(ctx, q, userID)
	if err != nil {
		return nil, fmt.Errorf("store: list %s: %w", table, err)
	}
	defer func() { _ = rows.Close() }()

	var out []credRow
	for rows.Next() {
		var c credRow
		var ca, lua, ra string
		if err := rows.Scan(&c.ID, &c.UserID, &c.Label, &ca, &lua, &ra); err != nil {
			return nil, fmt.Errorf("store: scan %s row: %w", table, err)
		}
		c.CreatedAt = parseUTC(ca)
		if lua != "" {
			c.LastUsedAt = new(parseUTC(lua))
		}
		if ra != "" {
			c.RevokedAt = new(parseUTC(ra))
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

// revokeCred marks a row in table as revoked. The `id = ? AND user_id = ?`
// scoping is a security boundary — it's what stops user A from revoking
// user B's key/token — and must not be dropped or loosened. Returns
// ErrNotFound if the row doesn't exist, is already revoked, or belongs to
// another user.
func (s *Store) revokeCred(ctx context.Context, table, userID, id string) error {
	q := fmt.Sprintf(`UPDATE %s SET revoked_at = ? WHERE id = ? AND user_id = ? AND revoked_at IS NULL`, table)
	res, err := s.db.ExecContext(ctx, q, utcNow(), id, userID)
	if err != nil {
		return fmt.Errorf("store: revoke %s: %w", table, err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return types.ErrNotFound
	}
	return nil
}

// CreateAPIKey inserts a new machine API key.
func (s *Store) CreateAPIKey(ctx context.Context, id, userID, hashedKey, label string) error {
	return s.createCred(ctx, "api_keys", "hashed_key", id, userID, hashedKey, label)
}

// ListAPIKeys returns all non-revoked API keys for a user.
func (s *Store) ListAPIKeys(ctx context.Context, userID string) ([]types.APIKey, error) {
	rows, err := s.listCreds(ctx, "api_keys", userID)
	if err != nil {
		return nil, err
	}
	var out []types.APIKey
	for _, r := range rows {
		out = append(out, types.APIKey(r))
	}
	return out, nil
}

// RevokeAPIKey marks an API key as revoked. Returns ErrNotFound if the key
// does not exist, is already revoked, or belongs to another user.
func (s *Store) RevokeAPIKey(ctx context.Context, userID, keyID string) error {
	return s.revokeCred(ctx, "api_keys", userID, keyID)
}

// CreateShareToken inserts a new read-only share token.
func (s *Store) CreateShareToken(ctx context.Context, id, userID, hashedToken, label string) error {
	return s.createCred(ctx, "share_tokens", "hashed_token", id, userID, hashedToken, label)
}

// ListShareTokens returns all non-revoked share tokens for a user.
func (s *Store) ListShareTokens(ctx context.Context, userID string) ([]types.ShareToken, error) {
	rows, err := s.listCreds(ctx, "share_tokens", userID)
	if err != nil {
		return nil, err
	}
	var out []types.ShareToken
	for _, r := range rows {
		out = append(out, types.ShareToken(r))
	}
	return out, nil
}

// RevokeShareToken marks a share token as revoked. Returns ErrNotFound if the
// token does not exist, is already revoked, or belongs to another user.
func (s *Store) RevokeShareToken(ctx context.Context, userID, tokenID string) error {
	return s.revokeCred(ctx, "share_tokens", userID, tokenID)
}

// GetUserByShareToken resolves a hashed share token to its owning user.
// Touches last_used_at on success. Skips revoked tokens. Returns
// ErrNotFound when the token is invalid or revoked.
func (s *Store) GetUserByShareToken(ctx context.Context, hashedToken string) (types.User, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return types.User{}, fmt.Errorf("store: share token lookup tx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	const tokenQ = `SELECT user_id FROM share_tokens WHERE hashed_token = ? AND revoked_at IS NULL` // #nosec G101 -- SQL query text, not a credential
	var userID string
	if err := tx.QueryRowContext(ctx, tokenQ, hashedToken).Scan(&userID); err == sql.ErrNoRows {
		return types.User{}, types.ErrNotFound
	} else if err != nil {
		return types.User{}, fmt.Errorf("store: lookup share token: %w", err)
	}

	_, _ = tx.ExecContext(ctx, `UPDATE share_tokens SET last_used_at = ? WHERE hashed_token = ?`, utcNow(), hashedToken)

	const userQ = `SELECT id, account_id, email, email_verified_at, status, display_name, timezone, locale, created_at, webauthn_handle
		FROM users WHERE id = ?`
	row := tx.QueryRowContext(ctx, userQ, userID)
	u, err := scanUser(row)
	if err == sql.ErrNoRows {
		return types.User{}, types.ErrNotFound
	}
	if err != nil {
		return types.User{}, fmt.Errorf("store: get user by share token: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return types.User{}, fmt.Errorf("store: commit share token lookup: %w", err)
	}
	return u, nil
}

// --- Login attempts ---

func (s *Store) RecordLoginAttempt(ctx context.Context, identifier string, succeeded bool) error {
	success := 0
	if succeeded {
		success = 1
	}
	const q = `INSERT INTO login_attempts (id, identifier, succeeded, created_at) VALUES (?, ?, ?, ?)`
	_, err := s.db.ExecContext(ctx, s.rewrite(q), newID(), identifier, success, utcNow())
	return err
}

func (s *Store) RecentFailedAttempts(ctx context.Context, identifier string, since time.Time) (int, error) {
	const q = `SELECT COUNT(*) FROM login_attempts
		WHERE identifier = ? AND succeeded = 0 AND created_at > ?`
	row := s.db.QueryRowContext(ctx, s.rewrite(q), identifier, utcStr(since))
	var n int
	if err := row.Scan(&n); err != nil {
		return 0, fmt.Errorf("store: recent failed attempts: %w", err)
	}
	return n, nil
}

// PurgeLoginAttempts deletes login attempts before olderThan. Callers retain
// enough history for every active lockout window before scheduling this purge.
func (s *Store) PurgeLoginAttempts(ctx context.Context, olderThan time.Time) (int, error) {
	const q = `DELETE FROM login_attempts WHERE created_at < ?`
	res, err := s.db.ExecContext(ctx, s.rewrite(q), olderThan.UTC().Format("2006-01-02 15:04:05"))
	if err != nil {
		return 0, fmt.Errorf("store: purge login attempts: %w", err)
	}
	n, _ := res.RowsAffected()
	return int(n), nil
}

// --- Audit ---

func (s *Store) WriteAuditEvent(ctx context.Context, ev types.AuditEvent) error {
	const q = `INSERT INTO auth_audit_log (id, account_id, user_id, event, ip, user_agent, meta, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)`
	_, err := s.db.ExecContext(ctx, s.rewrite(q),
		ev.ID, nullStr(ev.AccountID), nullStr(ev.UserID), ev.Event,
		nullStr(ev.IP), nullStr(ev.UserAgent), nullStr(ev.Meta), utcStr(ev.CreatedAt),
	)
	return err
}

// PurgeAuthAuditEvents deletes audit events before olderThan.
func (s *Store) PurgeAuthAuditEvents(ctx context.Context, olderThan time.Time) (int, error) {
	const q = `DELETE FROM auth_audit_log WHERE created_at < ?`
	res, err := s.db.ExecContext(ctx, s.rewrite(q), olderThan.UTC().Format("2006-01-02 15:04:05"))
	if err != nil {
		return 0, fmt.Errorf("store: purge auth audit events: %w", err)
	}
	n, _ := res.RowsAffected()
	return int(n), nil
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
		_ = tx.Rollback()
		return false, nil
	}

	return true, tx.Commit()
}

// --- MFA challenges (stored in auth_challenges with kind='mfa') ---

// CreateMFAChallenge inserts a new MFA challenge row. The remember flag is
// serialised into the JSON payload column.
func (s *Store) CreateMFAChallenge(ctx context.Context, id, userID string, remember bool, expiresAt string) error {
	payload, _ := json.Marshal(map[string]bool{"remember": remember})
	const q = `INSERT INTO auth_challenges (id, user_id, kind, payload_json, expires_at, created_at)
		VALUES (?, ?, 'mfa', ?, ?, ?)`
	_, err := s.db.ExecContext(ctx, q, id, userID, string(payload), expiresAt, utcNow())
	return err
}

// GetMFAChallenge retrieves a challenge by its hashed token ID.
// Returns types.ErrNotFound when the challenge does not exist.
func (s *Store) GetMFAChallenge(ctx context.Context, id string) (userID string, remember bool, expiresAt string, err error) {
	const q = `SELECT user_id, payload_json, expires_at FROM auth_challenges WHERE id = ? AND kind = 'mfa'`
	row := s.db.QueryRowContext(ctx, q, id)
	var payload string
	if err := row.Scan(&userID, &payload, &expiresAt); err == sql.ErrNoRows {
		return "", false, "", types.ErrNotFound
	} else if err != nil {
		return "", false, "", fmt.Errorf("store: get mfa challenge: %w", err)
	}
	var data struct {
		Remember bool `json:"remember"`
	}
	if json.Unmarshal([]byte(payload), &data) == nil {
		remember = data.Remember
	}
	return userID, remember, expiresAt, nil
}

// DeleteMFAChallenge removes a challenge by its hashed token ID.
func (s *Store) DeleteMFAChallenge(ctx context.Context, id string) error {
	const q = `DELETE FROM auth_challenges WHERE id = ?`
	_, err := s.db.ExecContext(ctx, q, id)
	return err
}

// --- OIDC identities ---

// GetUserByOIDCIdentity returns the user linked to a provider+subject pair.
// Returns types.ErrNotFound when no matching identity exists.
func (s *Store) GetUserByOIDCIdentity(ctx context.Context, provider, subject string) (types.User, error) {
	const q = `SELECT u.id, u.account_id, u.email, u.email_verified_at, u.status, u.display_name, u.timezone, u.locale, u.created_at, u.webauthn_handle
		FROM oidc_identities oi JOIN users u ON u.id = oi.user_id
		WHERE oi.provider = ? AND oi.subject = ?`
	row := s.db.QueryRowContext(ctx, q, provider, subject)
	u, err := scanUser(row)
	if err == sql.ErrNoRows {
		return types.User{}, types.ErrNotFound
	}
	return u, err
}

// LinkOIDCIdentity inserts a new OIDC identity row. The UNIQUE(provider,subject)
// constraint guarantees one identity per provider+subject. On conflict, returns
// types.ErrIdentityLinked.
func (s *Store) LinkOIDCIdentity(ctx context.Context, id, userID, provider, subject, email string) error {
	const q = `INSERT INTO oidc_identities (id, user_id, provider, subject, email, linked_at, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)`
	_, err := s.db.ExecContext(ctx, q, id, userID, provider, subject, nullStr(email), utcNow(), utcNow())
	if err != nil {
		// UNIQUE(provider, subject) violation.
		if isUniqueViolation(err) {
			return types.ErrIdentityLinked
		}
		return fmt.Errorf("store: link oidc identity: %w", err)
	}
	return nil
}

// ListOIDCIdentities returns all OIDC identities for a user.
func (s *Store) ListOIDCIdentities(ctx context.Context, userID string) ([]types.OIDCIdentity, error) {
	const q = `SELECT id, user_id, provider, subject, email, linked_at, created_at
		FROM oidc_identities WHERE user_id = ? ORDER BY linked_at DESC`
	rows, err := s.db.QueryContext(ctx, q, userID)
	if err != nil {
		return nil, fmt.Errorf("store: list oidc identities: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var out []types.OIDCIdentity
	for rows.Next() {
		var oi types.OIDCIdentity
		var la, ca string
		var email sql.NullString
		if err := rows.Scan(&oi.ID, &oi.UserID, &oi.Provider, &oi.Subject, &email, &la, &ca); err != nil {
			return nil, fmt.Errorf("store: scan oidc identity: %w", err)
		}
		if email.Valid {
			oi.Email = email.String
		}
		oi.LinkedAt = parseUTC(la)
		oi.CreatedAt = parseUTC(ca)
		out = append(out, oi)
	}
	return out, rows.Err()
}

// DeleteOIDCIdentity removes one OIDC identity, scoped by user. Returns
// types.ErrNotFound when the identity does not exist or belongs to another user.
func (s *Store) DeleteOIDCIdentity(ctx context.Context, userID, id string) error {
	const q = `DELETE FROM oidc_identities WHERE id = ? AND user_id = ?`
	res, err := s.db.ExecContext(ctx, q, id, userID)
	if err != nil {
		return fmt.Errorf("store: delete oidc identity: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return types.ErrNotFound
	}
	return nil
}

// CreateUserWithOIDC creates an account (if needed), inserts the user with
// email_verified_at set to now (OIDC asserted it), and links the identity row —
// all in one transaction. No password_credentials row is created.
func (s *Store) CreateUserWithOIDC(ctx context.Context, accountID, userID, email, displayName, identityID, provider, subject string) (types.User, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return types.User{}, fmt.Errorf("store: create user with oidc tx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	if _, err := tx.ExecContext(ctx,
		`INSERT INTO accounts (id, created_at) VALUES (?, ?) ON CONFLICT DO NOTHING`,
		accountID, utcNow(),
	); err != nil {
		return types.User{}, fmt.Errorf("store: insert account: %w", err)
	}

	now := utcNow()
	u := types.User{
		ID:              userID,
		AccountID:       accountID,
		Email:           email,
		Status:          "active",
		DisplayName:     displayName,
		Timezone:        "UTC",
		CreatedAt:       parseUTC(now),
		EmailVerifiedAt: ptrTime(parseUTC(now)),
	}

	if _, err := tx.ExecContext(ctx,
		`INSERT INTO users (id, account_id, email, email_verified_at, status, display_name, timezone, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		u.ID, u.AccountID, u.Email, now, u.Status, nullStr(u.DisplayName), u.Timezone, now,
	); err != nil {
		return types.User{}, fmt.Errorf("store: insert user: %w", err)
	}

	if _, err := tx.ExecContext(ctx,
		`INSERT INTO oidc_identities (id, user_id, provider, subject, email, linked_at, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		identityID, u.ID, provider, subject, nullStr(email), now, now,
	); err != nil {
		return types.User{}, fmt.Errorf("store: insert oidc identity: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return types.User{}, fmt.Errorf("store: commit user with oidc: %w", err)
	}

	return u, nil
}

// --- OIDC state tokens ---

// CreateOIDCState persists an OIDC authorization state entry.
func (s *Store) CreateOIDCState(ctx context.Context, id, nonce, pkceVerifier, linkUserID, next, expiresAt string) error {
	const q = `INSERT INTO oidc_states (id, nonce, pkce_verifier, link_user_id, next, expires_at, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)`
	_, err := s.db.ExecContext(ctx, q, id, nonce, pkceVerifier, nullStr(linkUserID), nullStr(next), expiresAt, utcNow())
	return err
}

// ConsumeOIDCState returns the state entry and deletes it in a single
// transaction (single-use). Returns types.ErrNotFound when the state does not
// exist or has expired.
func (s *Store) ConsumeOIDCState(ctx context.Context, id string) (nonce, pkceVerifier, linkUserID, next string, err error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return "", "", "", "", fmt.Errorf("store: consume oidc state tx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	var luID, n sql.NullString
	const q = `SELECT nonce, pkce_verifier, link_user_id, next, expires_at FROM oidc_states WHERE id = ?`
	row := tx.QueryRowContext(ctx, q, id)
	var expiresAt string
	if scanErr := row.Scan(&nonce, &pkceVerifier, &luID, &n, &expiresAt); scanErr == sql.ErrNoRows {
		return "", "", "", "", types.ErrNotFound
	} else if scanErr != nil {
		return "", "", "", "", fmt.Errorf("store: scan oidc state: %w", scanErr)
	}

	// Check expiry.
	exp, parseErr := time.Parse(time.RFC3339, expiresAt)
	if parseErr != nil || time.Now().UTC().After(exp) {
		_ = tx.Commit() // rowsAffected will be 0, caller handles
		return "", "", "", "", types.ErrNotFound
	}

	if _, delErr := tx.ExecContext(ctx, `DELETE FROM oidc_states WHERE id = ?`, id); delErr != nil {
		return "", "", "", "", fmt.Errorf("store: delete oidc state: %w", delErr)
	}

	if err := tx.Commit(); err != nil {
		return "", "", "", "", fmt.Errorf("store: commit consume oidc state: %w", err)
	}

	if luID.Valid {
		linkUserID = luID.String
	}
	if n.Valid {
		next = n.String
	}
	return nonce, pkceVerifier, linkUserID, next, nil
}

// DeleteOIDCState removes an OIDC state row by id (best-effort cleanup).
func (s *Store) DeleteOIDCState(ctx context.Context, id string) error {
	const q = `DELETE FROM oidc_states WHERE id = ?`
	_, err := s.db.ExecContext(ctx, q, id)
	return err
}

// --- Email tokens (stored in auth_verification_codes) ---

// ponytail: Old auth_email_tokens had no UNIQUE; merged auth_verification_codes
// enforces UNIQUE(user_id, purpose). Requesting a second code for the same
// purpose replaces the outstanding one (ON CONFLICT DO UPDATE).

// MarkEmailVerified sets email_verified_at to now for the user.
func (s *Store) MarkEmailVerified(ctx context.Context, userID string) error {
	const q = `UPDATE users SET email_verified_at = ? WHERE id = ?`
	_, err := s.db.ExecContext(ctx, q, utcNow(), userID)
	return err
}

// UpdateUserEmail sets a new email and clears email_verified_at so the user
// must verify the new address.
func (s *Store) UpdateUserEmail(ctx context.Context, userID, email string) error {
	const q = `UPDATE users SET email = ?, email_verified_at = NULL WHERE id = ?`
	_, err := s.db.ExecContext(ctx, q, email, userID)
	return err
}

// CreateEmailToken persists a single-use email token. The caller-provided id is
// already a SHA-256 hash. ON CONFLICT replaces any outstanding token for the
// same (user_id, purpose) — requesting a second code invalidates the first.
func (s *Store) CreateEmailToken(ctx context.Context, id, userID, purpose, expiresAt string) error {
	const q = `INSERT INTO auth_verification_codes (id, user_id, purpose, code_hash, attempts, expires_at, created_at)
		VALUES (?, ?, ?, ?, 0, ?, ?)
		ON CONFLICT(user_id, purpose) DO UPDATE SET id = excluded.id, code_hash = excluded.code_hash, expires_at = excluded.expires_at, attempts = 0, created_at = excluded.created_at`
	// code_hash is set to id (both are the SHA-256 hash of the raw token).
	_, err := s.db.ExecContext(ctx, q, id, userID, purpose, id, expiresAt, utcNow())
	return err
}

// ConsumeEmailToken returns the user_id for a single-use email token and deletes
// it in one transaction. Returns types.ErrNotFound when the token does not exist,
// has expired, or has the wrong purpose.
func (s *Store) ConsumeEmailToken(ctx context.Context, id, purpose string) (userID string, err error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return "", fmt.Errorf("store: consume email token tx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	var storedPurpose, expiresAt string
	const q = `SELECT user_id, purpose, expires_at FROM auth_verification_codes WHERE id = ?`
	row := tx.QueryRowContext(ctx, q, id)
	if scanErr := row.Scan(&userID, &storedPurpose, &expiresAt); scanErr == sql.ErrNoRows {
		return "", types.ErrNotFound
	} else if scanErr != nil {
		return "", fmt.Errorf("store: scan email token: %w", scanErr)
	}

	// Verify purpose matches.
	if storedPurpose != purpose {
		return "", types.ErrNotFound
	}

	// Check expiry.
	exp, parseErr := time.Parse(time.RFC3339, expiresAt)
	if parseErr != nil || time.Now().UTC().After(exp) {
		// Delete expired token.
		_, _ = tx.ExecContext(ctx, `DELETE FROM auth_verification_codes WHERE id = ?`, id)
		_ = tx.Commit()
		return "", types.ErrNotFound
	}

	if _, delErr := tx.ExecContext(ctx, `DELETE FROM auth_verification_codes WHERE id = ?`, id); delErr != nil {
		return "", fmt.Errorf("store: delete email token: %w", delErr)
	}

	if err := tx.Commit(); err != nil {
		return "", fmt.Errorf("store: commit consume email token: %w", err)
	}

	return userID, nil
}

// DeleteEmailTokensByUserAndPurpose removes all email tokens for a user+purpose
// combination. Used to clean up sibling credentials on successful magic sign-in.
func (s *Store) DeleteEmailTokensByUserAndPurpose(ctx context.Context, userID, purpose string) error {
	const q = `DELETE FROM auth_verification_codes WHERE user_id = ? AND purpose = ?`
	_, err := s.db.ExecContext(ctx, q, userID, purpose)
	return err
}

// --- Magic codes (stored in auth_verification_codes with purpose='magic_signin') ---

// UpsertMagicCode inserts or replaces the active magic code for a user.
// One active code per user (UNIQUE(user_id, purpose='magic_signin')); resend
// overwrites.
func (s *Store) UpsertMagicCode(ctx context.Context, userID, codeHash, expiresAt string) error {
	const q = `INSERT INTO auth_verification_codes (id, user_id, purpose, code_hash, attempts, expires_at, created_at)
		VALUES (?, ?, 'magic_signin', ?, 0, ?, ?)
		ON CONFLICT(user_id, purpose) DO UPDATE SET code_hash = excluded.code_hash, expires_at = excluded.expires_at, attempts = 0, created_at = excluded.created_at`
	_, err := s.db.ExecContext(ctx, q, newID(), userID, codeHash, expiresAt, utcNow())
	return err
}

// GetMagicCode returns the stored magic code info for a user, or types.ErrNotFound.
func (s *Store) GetMagicCode(ctx context.Context, userID string) (codeHash, expiresAt string, attempts int, err error) {
	const q = `SELECT code_hash, expires_at, attempts FROM auth_verification_codes WHERE user_id = ? AND purpose = 'magic_signin'`
	row := s.db.QueryRowContext(ctx, q, userID)
	if scanErr := row.Scan(&codeHash, &expiresAt, &attempts); scanErr == sql.ErrNoRows {
		return "", "", 0, types.ErrNotFound
	} else if scanErr != nil {
		return "", "", 0, fmt.Errorf("store: get magic code: %w", scanErr)
	}
	return codeHash, expiresAt, attempts, nil
}

// IncrementMagicCodeAttempts bumps the attempt counter for an active code.
func (s *Store) IncrementMagicCodeAttempts(ctx context.Context, userID string) error {
	const q = `UPDATE auth_verification_codes SET attempts = attempts + 1 WHERE user_id = ? AND purpose = 'magic_signin'`
	_, err := s.db.ExecContext(ctx, q, userID)
	return err
}

// DeleteMagicCode removes the active magic code for a user (consume on success,
// clear on expiry/cap).
func (s *Store) DeleteMagicCode(ctx context.Context, userID string) error {
	const q = `DELETE FROM auth_verification_codes WHERE user_id = ? AND purpose = 'magic_signin'`
	_, err := s.db.ExecContext(ctx, q, userID)
	return err
}

// ---------------------------------------------------------------------------
// WebAuthn passkeys, ceremony sessions, and MFA email codes.
// ---------------------------------------------------------------------------

// --- WebAuthn user handle ---

// GetOrCreateWebAuthnHandle reads the user's webauthn_handle. If it's empty,
// generates a new one, persists it, and returns it.
func (s *Store) GetOrCreateWebAuthnHandle(ctx context.Context, userID string) (string, error) {
	const q = `SELECT webauthn_handle FROM users WHERE id = ?`
	row := s.db.QueryRowContext(ctx, q, userID)
	var handle sql.NullString
	if err := row.Scan(&handle); err == sql.ErrNoRows {
		return "", types.ErrNotFound
	} else if err != nil {
		return "", fmt.Errorf("store: get webauthn handle: %w", err)
	}
	if handle.Valid && handle.String != "" {
		return handle.String, nil
	}

	newHandle := auth.NewWebAuthnHandle()
	const up = `UPDATE users SET webauthn_handle = ? WHERE id = ?`
	if _, err := s.db.ExecContext(ctx, up, newHandle, userID); err != nil {
		return "", fmt.Errorf("store: set webauthn handle: %w", err)
	}
	return newHandle, nil
}

// GetUserByWebAuthnHandle returns the user for a given webauthn handle (used in
// discoverable login to resolve the user from the authenticator response).
func (s *Store) GetUserByWebAuthnHandle(ctx context.Context, handle string) (types.User, error) {
	const q = `SELECT id, account_id, email, email_verified_at, status, display_name, timezone, locale, created_at, webauthn_handle
		FROM users WHERE webauthn_handle = ?`
	row := s.db.QueryRowContext(ctx, q, handle)
	u, err := scanUser(row)
	if err == sql.ErrNoRows {
		return types.User{}, types.ErrNotFound
	}
	return u, err
}

// --- WebAuthn credentials ---

// CreateWebAuthnCredential inserts a new passkey credential row.
func (s *Store) CreateWebAuthnCredential(ctx context.Context, id, userID, label, credentialJSON string, signCount int, createdAt string) error {
	const q = `INSERT INTO webauthn_credentials (id, user_id, label, credential_json, sign_count, created_at)
		VALUES (?, ?, ?, ?, ?, ?)`
	_, err := s.db.ExecContext(ctx, q, id, userID, label, credentialJSON, signCount, createdAt)
	return err
}

// ListWebAuthnCredentials returns the user-visible passkey list for a user.
func (s *Store) ListWebAuthnCredentials(ctx context.Context, userID string) ([]types.Passkey, error) {
	const q = `SELECT id, label, created_at, last_used_at FROM webauthn_credentials WHERE user_id = ? ORDER BY created_at DESC`
	rows, err := s.db.QueryContext(ctx, q, userID)
	if err != nil {
		return nil, fmt.Errorf("store: list webauthn credentials: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var out []types.Passkey
	for rows.Next() {
		var pk types.Passkey
		var lua sql.NullString
		if err := rows.Scan(&pk.ID, &pk.Label, &pk.CreatedAt, &lua); err != nil {
			return nil, fmt.Errorf("store: scan webauthn credential: %w", err)
		}
		pk.LastUsedAt = lua.String
		out = append(out, pk)
	}
	return out, rows.Err()
}

// GetWebAuthnCredentialsRaw returns the raw credential rows needed to build a
// WebAuthnUser for ceremony operations.
func (s *Store) GetWebAuthnCredentialsRaw(ctx context.Context, userID string) ([]types.WebAuthnCredential, error) {
	const q = `SELECT id, credential_json FROM webauthn_credentials WHERE user_id = ?`
	rows, err := s.db.QueryContext(ctx, q, userID)
	if err != nil {
		return nil, fmt.Errorf("store: get webauthn credentials raw: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var out []types.WebAuthnCredential
	for rows.Next() {
		var c types.WebAuthnCredential
		if err := rows.Scan(&c.ID, &c.CredentialJSON); err != nil {
			return nil, fmt.Errorf("store: scan webauthn credential raw: %w", err)
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

// UpdateWebAuthnCredentialOnAuth rewrites the credential JSON, sign count, and
// last_used_at after a successful assertion.
func (s *Store) UpdateWebAuthnCredentialOnAuth(ctx context.Context, id, credentialJSON string, signCount int, lastUsedAt string) error {
	const q = `UPDATE webauthn_credentials SET credential_json = ?, sign_count = ?, last_used_at = ? WHERE id = ?`
	_, err := s.db.ExecContext(ctx, q, credentialJSON, signCount, lastUsedAt, id)
	return err
}

// RenameWebAuthnCredential updates the label for a passkey, scoped by user.
func (s *Store) RenameWebAuthnCredential(ctx context.Context, userID, id, label string) error {
	const q = `UPDATE webauthn_credentials SET label = ? WHERE id = ? AND user_id = ?`
	res, err := s.db.ExecContext(ctx, q, label, id, userID)
	if err != nil {
		return fmt.Errorf("store: rename webauthn credential: %w", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return types.ErrNotFound
	}
	return nil
}

// DeleteWebAuthnCredential removes a passkey, scoped by user.
func (s *Store) DeleteWebAuthnCredential(ctx context.Context, userID, id string) error {
	const q = `DELETE FROM webauthn_credentials WHERE id = ? AND user_id = ?`
	res, err := s.db.ExecContext(ctx, q, id, userID)
	if err != nil {
		return fmt.Errorf("store: delete webauthn credential: %w", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return types.ErrNotFound
	}
	return nil
}

// --- WebAuthn ceremony sessions (stored in auth_challenges with kind='webauthn_ceremony') ---

// CreateWebAuthnSession persists a go-webauthn SessionData. userID may be ""
// for discoverable login (the user is not yet known).
func (s *Store) CreateWebAuthnSession(ctx context.Context, id, userID, sessionDataJSON, expiresAt string) error {
	const q = `INSERT INTO auth_challenges (id, user_id, kind, payload_json, expires_at, created_at)
		VALUES (?, ?, 'webauthn_ceremony', ?, ?, ?)`
	_, err := s.db.ExecContext(ctx, q, id, nullStr(userID), sessionDataJSON, expiresAt, utcNow())
	return err
}

// ConsumeWebAuthnSession reads and deletes a ceremony session in one
// transaction (single-use). Returns types.ErrNotFound when the session is
// absent or expired.
func (s *Store) ConsumeWebAuthnSession(ctx context.Context, id string) (userID, sessionDataJSON string, err error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return "", "", fmt.Errorf("store: consume webauthn session tx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	var uid sql.NullString
	const q = `SELECT user_id, payload_json, expires_at FROM auth_challenges WHERE id = ? AND kind = 'webauthn_ceremony'`
	row := tx.QueryRowContext(ctx, q, id)
	var expiresAt string
	if scanErr := row.Scan(&uid, &sessionDataJSON, &expiresAt); scanErr == sql.ErrNoRows {
		return "", "", types.ErrNotFound
	} else if scanErr != nil {
		return "", "", fmt.Errorf("store: scan webauthn session: %w", scanErr)
	}

	// Check expiry.
	exp, parseErr := time.Parse(time.RFC3339, expiresAt)
	if parseErr != nil || time.Now().UTC().After(exp) {
		_ = tx.Commit()
		return "", "", types.ErrNotFound
	}

	if _, delErr := tx.ExecContext(ctx, `DELETE FROM auth_challenges WHERE id = ?`, id); delErr != nil {
		return "", "", fmt.Errorf("store: delete webauthn session: %w", delErr)
	}

	if err := tx.Commit(); err != nil {
		return "", "", fmt.Errorf("store: commit consume webauthn session: %w", err)
	}

	if uid.Valid {
		userID = uid.String
	}
	return userID, sessionDataJSON, nil
}

// --- MFA email codes (stored in auth_verification_codes with purpose='mfa_email') ---

// ponytail: Old mfa_email_codes was keyed by challenge_id; merged
// auth_verification_codes keys by (user_id, purpose). The caller already has
// the real userID from GetMFAChallenge. UNIQUE(user_id, purpose) guarantees
// one active code per user.

// UpsertMFAEmailCode inserts or replaces the active email code for an MFA step-up.
func (s *Store) UpsertMFAEmailCode(ctx context.Context, userID, codeHash, expiresAt string) error {
	const q = `INSERT INTO auth_verification_codes (id, user_id, purpose, code_hash, attempts, expires_at, created_at)
		VALUES (?, ?, 'mfa_email', ?, 0, ?, ?)
		ON CONFLICT(user_id, purpose) DO UPDATE SET code_hash = excluded.code_hash, expires_at = excluded.expires_at, attempts = 0, created_at = excluded.created_at`
	_, err := s.db.ExecContext(ctx, q, newID(), userID, codeHash, expiresAt, utcNow())
	return err
}

// GetMFAEmailCode returns the stored email code info for a user, or types.ErrNotFound.
func (s *Store) GetMFAEmailCode(ctx context.Context, userID string) (codeHash, expiresAt string, attempts int, err error) {
	const q = `SELECT code_hash, expires_at, attempts FROM auth_verification_codes WHERE user_id = ? AND purpose = 'mfa_email'`
	row := s.db.QueryRowContext(ctx, q, userID)
	if scanErr := row.Scan(&codeHash, &expiresAt, &attempts); scanErr == sql.ErrNoRows {
		return "", "", 0, types.ErrNotFound
	} else if scanErr != nil {
		return "", "", 0, fmt.Errorf("store: get mfa email code: %w", scanErr)
	}
	return codeHash, expiresAt, attempts, nil
}

// IncrementMFAEmailCodeAttempts bumps the attempt counter for an active code.
func (s *Store) IncrementMFAEmailCodeAttempts(ctx context.Context, userID string) error {
	const q = `UPDATE auth_verification_codes SET attempts = attempts + 1 WHERE user_id = ? AND purpose = 'mfa_email'`
	_, err := s.db.ExecContext(ctx, q, userID)
	return err
}

// DeleteMFAEmailCode removes the active email code for a user.
func (s *Store) DeleteMFAEmailCode(ctx context.Context, userID string) error {
	const q = `DELETE FROM auth_verification_codes WHERE user_id = ? AND purpose = 'mfa_email'`
	_, err := s.db.ExecContext(ctx, q, userID)
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
