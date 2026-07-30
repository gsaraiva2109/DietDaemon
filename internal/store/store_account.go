package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/gsaraiva2109/dietdaemon/core/types"
)

// DeleteAccount erases a user's entire account: it resolves userID's
// account_id, then deletes the accounts row. Every per-user table has a
// user_id FK with ON DELETE CASCADE (via users.account_id -> accounts.id),
// so this single delete cascades through users and all their data in one
// step. auth_audit_log is the deliberate exception (ON DELETE SET NULL),
// so audit rows survive with user_id/account_id cleared.
// Returns types.ErrNotFound if userID does not exist.
func (s *Store) DeleteAccount(ctx context.Context, userID string) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("store: delete account tx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	var accountID string
	if err := tx.QueryRowContext(ctx, s.rewrite(`SELECT account_id FROM users WHERE id = ?`), userID).Scan(&accountID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return types.ErrNotFound
		}
		return fmt.Errorf("store: lookup account for user: %w", err)
	}

	if _, err := tx.ExecContext(ctx, s.rewrite(`DELETE FROM accounts WHERE id = ?`), accountID); err != nil {
		return fmt.Errorf("store: delete account: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("store: commit delete account: %w", err)
	}
	return nil
}

// RequestAccountDeletion soft-deletes the account owning userID: it sets
// accounts.deleted_at, deletes every session and revokes every API key for
// every user under the account, and writes an account.delete.requested
// audit event. All in one transaction. The account row (and its data) is
// left in place — a background purge job handles the day-30 photo purge and
// day-90 hard delete described in the tiered retention plan; nothing here
// deletes actual data.
// Returns types.ErrNotFound if userID does not exist.
func (s *Store) RequestAccountDeletion(ctx context.Context, userID string) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("store: request account deletion tx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	var accountID string
	if err := tx.QueryRowContext(ctx, s.rewrite(`SELECT account_id FROM users WHERE id = ?`), userID).Scan(&accountID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return types.ErrNotFound
		}
		return fmt.Errorf("store: lookup account for user: %w", err)
	}

	userIDs, err := accountUserIDs(ctx, tx, s.rewrite, accountID)
	if err != nil {
		return err
	}

	now := utcNow()
	if _, err := tx.ExecContext(ctx, s.rewrite(`UPDATE accounts SET deleted_at = ? WHERE id = ?`), now, accountID); err != nil {
		return fmt.Errorf("store: set account deleted_at: %w", err)
	}

	for _, uid := range userIDs {
		// Mirrors DeleteUserSessions/revokeCred's queries; done inline here
		// (rather than calling those s.db-based methods) so the whole
		// deletion request commits or rolls back as one unit.
		if _, err := tx.ExecContext(ctx, s.rewrite(`DELETE FROM sessions WHERE user_id = ?`), uid); err != nil {
			return fmt.Errorf("store: delete sessions for user: %w", err)
		}
		if _, err := tx.ExecContext(ctx, s.rewrite(`UPDATE api_keys SET revoked_at = ? WHERE user_id = ? AND revoked_at IS NULL`), now, uid); err != nil {
			return fmt.Errorf("store: revoke api keys for user: %w", err)
		}
	}

	if err := insertAuditEventTx(ctx, tx, s.rewrite, types.AuditEvent{
		ID:        newID(),
		AccountID: accountID,
		UserID:    userID,
		Event:     "account.delete.requested",
		CreatedAt: time.Now(),
	}); err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("store: commit request account deletion: %w", err)
	}
	return nil
}

// ReactivateAccount clears accounts.deleted_at for the account owning
// userID, restoring normal access. photos_purged_at is left untouched —
// once the retention purge job has deleted progress photos, reactivation
// cannot bring them back. Writes an account.reactivated audit event.
// Returns types.ErrNotFound if userID does not exist.
func (s *Store) ReactivateAccount(ctx context.Context, userID string) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("store: reactivate account tx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	var accountID string
	if err := tx.QueryRowContext(ctx, s.rewrite(`SELECT account_id FROM users WHERE id = ?`), userID).Scan(&accountID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return types.ErrNotFound
		}
		return fmt.Errorf("store: lookup account for user: %w", err)
	}

	if _, err := tx.ExecContext(ctx, s.rewrite(`UPDATE accounts SET deleted_at = NULL WHERE id = ?`), accountID); err != nil {
		return fmt.Errorf("store: clear account deleted_at: %w", err)
	}

	if err := insertAuditEventTx(ctx, tx, s.rewrite, types.AuditEvent{
		ID:        newID(),
		AccountID: accountID,
		UserID:    userID,
		Event:     "account.reactivated",
		CreatedAt: time.Now(),
	}); err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("store: commit reactivate account: %w", err)
	}
	return nil
}

// AccountDeletionStatus reports the account owning userID's deletion tier:
// whether it's soft-deleted and whether its photos have been purged.
// Returns types.ErrNotFound if userID does not exist.
func (s *Store) AccountDeletionStatus(ctx context.Context, userID string) (AccountDeletionStatus, error) {
	const q = `SELECT a.deleted_at, a.photos_purged_at
		FROM accounts a JOIN users u ON u.account_id = a.id
		WHERE u.id = ?`
	var deletedAt, photosPurgedAt sql.NullString
	err := s.db.QueryRowContext(ctx, s.rewrite(q), userID).Scan(&deletedAt, &photosPurgedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return AccountDeletionStatus{}, types.ErrNotFound
	}
	if err != nil {
		return AccountDeletionStatus{}, fmt.Errorf("store: account deletion status: %w", err)
	}

	var status AccountDeletionStatus
	if deletedAt.Valid && deletedAt.String != "" {
		status.DeletedAt = ptrTime(parseUTC(deletedAt.String))
	}
	if photosPurgedAt.Valid && photosPurgedAt.String != "" {
		status.PhotosPurgedAt = ptrTime(parseUTC(photosPurgedAt.String))
	}
	return status, nil
}

// AccountDeletionStatus is the return type for Store.AccountDeletionStatus:
// nil fields mean "not set" (account active / photos not purged).
type AccountDeletionStatus struct {
	DeletedAt      *time.Time
	PhotosPurgedAt *time.Time
}

// accountUserIDs returns the IDs of every user under accountID, within tx.
func accountUserIDs(ctx context.Context, tx *sql.Tx, rewrite func(string) string, accountID string) ([]string, error) {
	rows, err := tx.QueryContext(ctx, rewrite(`SELECT id FROM users WHERE account_id = ?`), accountID)
	if err != nil {
		return nil, fmt.Errorf("store: list account users: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("store: scan account user id: %w", err)
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

// insertAuditEventTx writes an auth_audit_log row within tx, matching the
// column set and helpers WriteAuditEvent uses outside a transaction.
func insertAuditEventTx(ctx context.Context, tx *sql.Tx, rewrite func(string) string, ev types.AuditEvent) error {
	const q = `INSERT INTO auth_audit_log (id, account_id, user_id, event, ip, user_agent, meta, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)`
	if _, err := tx.ExecContext(ctx, rewrite(q),
		ev.ID, nullStr(ev.AccountID), nullStr(ev.UserID), ev.Event,
		nullStr(ev.IP), nullStr(ev.UserAgent), nullStr(ev.Meta), utcStr(ev.CreatedAt),
	); err != nil {
		return fmt.Errorf("store: write audit event: %w", err)
	}
	return nil
}
