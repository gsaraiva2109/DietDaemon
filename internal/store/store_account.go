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

// AccountDeletedAt is a narrow view of AccountDeletionStatus exposing only
// DeletedAt. internal/backup needs this to exclude accounts pending deletion
// from scheduled backups, but can't import internal/store directly (this
// package already imports internal/backup, so that would cycle) -- so it
// declares its own interface method against this signature instead of the
// full AccountDeletionStatus type.
func (s *Store) AccountDeletedAt(ctx context.Context, userID string) (*time.Time, error) {
	status, err := s.AccountDeletionStatus(ctx, userID)
	return status.DeletedAt, err
}

// ListAccountsPendingPhotoPurge returns IDs of accounts soft-deleted at or
// before deletedBefore whose photos have not yet been purged. Used by
// PurgeRunner both for day-30 photo-purge candidates and, with an earlier
// cutoff, day-25 reminder candidates.
func (s *Store) ListAccountsPendingPhotoPurge(ctx context.Context, deletedBefore time.Time) ([]string, error) {
	const q = `SELECT id FROM accounts WHERE deleted_at IS NOT NULL AND deleted_at <= ? AND photos_purged_at IS NULL`
	rows, err := s.db.QueryContext(ctx, s.rewrite(q), deletedBefore.UTC().Format(time.RFC3339))
	if err != nil {
		return nil, fmt.Errorf("store: list accounts pending photo purge: %w", err)
	}
	defer func() { _ = rows.Close() }()
	return scanAccountIDs(rows)
}

// PurgeAccountPhotos hard-deletes every progress photo belonging to
// accountID's users and sets accounts.photos_purged_at, all in one
// transaction. Writes an account.photos_purged audit event. Callers filter
// candidates via ListAccountsPendingPhotoPurge (photos_purged_at IS NULL),
// so re-running this on an already-purged account is a harmless no-op.
//
// The photos_purged_at UPDATE is guarded (deleted_at IS NOT NULL AND
// photos_purged_at IS NULL) and runs before the destructive DELETE, closing a
// TOCTOU window: if the account was reactivated concurrently (e.g. a job
// picked it up from ListAccountsPendingPhotoPurge, then the owner reactivated
// before this ran), the guard matches zero rows and the function skips the
// delete entirely rather than destroying photos for an account that's no
// longer marked deleted.
func (s *Store) PurgeAccountPhotos(ctx context.Context, accountID string) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("store: purge account photos tx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	res, err := tx.ExecContext(ctx, s.rewrite(
		`UPDATE accounts SET photos_purged_at = ? WHERE id = ? AND deleted_at IS NOT NULL AND photos_purged_at IS NULL`),
		utcNow(), accountID)
	if err != nil {
		return fmt.Errorf("store: set photos_purged_at: %w", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return nil // reactivated or already purged concurrently: skip, no photos deleted
	}

	userIDs, err := accountUserIDs(ctx, tx, s.rewrite, accountID)
	if err != nil {
		return err
	}
	for _, uid := range userIDs {
		if _, err := tx.ExecContext(ctx, s.rewrite(`DELETE FROM progress_photos WHERE user_id = ?`), uid); err != nil {
			return fmt.Errorf("store: delete progress photos: %w", err)
		}
	}

	if err := insertAuditEventTx(ctx, tx, s.rewrite, types.AuditEvent{
		ID:        newID(),
		AccountID: accountID,
		Event:     "account.photos_purged",
		CreatedAt: time.Now(),
	}); err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("store: commit purge account photos: %w", err)
	}
	return nil
}

// ListAccountsPastDeletion returns IDs of accounts soft-deleted at or before
// deletedBefore, regardless of photo-purge state. Used by PurgeRunner both
// for day-90 final-purge candidates and, with an earlier cutoff, day-85
// reminder candidates.
func (s *Store) ListAccountsPastDeletion(ctx context.Context, deletedBefore time.Time) ([]string, error) {
	const q = `SELECT id FROM accounts WHERE deleted_at IS NOT NULL AND deleted_at <= ?`
	rows, err := s.db.QueryContext(ctx, s.rewrite(q), deletedBefore.UTC().Format(time.RFC3339))
	if err != nil {
		return nil, fmt.Errorf("store: list accounts past deletion: %w", err)
	}
	defer func() { _ = rows.Close() }()
	return scanAccountIDs(rows)
}

// PurgeAccount permanently deletes accountID: within one transaction it first
// atomically claims the row with a guarded no-op UPDATE (deleted_at IS NOT
// NULL), skipping entirely if a concurrent reactivation already cleared it;
// then, with the claim held for the rest of the transaction, writes the
// account.delete.purged audit event while account_id is still a valid FK
// target, and only then hard-deletes the accounts row. The
// auth_audit_log.account_id FK is ON DELETE SET NULL, so the DELETE nulls the
// just-written audit row's account_id as part of the same statement -- the
// event survives, and the account plus everything under it cascades away for
// good. The audit insert must precede the physical DELETE: inserting it after
// would reference an account_id no longer present in accounts, violating the
// FK.
func (s *Store) PurgeAccount(ctx context.Context, accountID string) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("store: purge account tx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	res, err := tx.ExecContext(ctx, s.rewrite(
		`UPDATE accounts SET deleted_at = deleted_at WHERE id = ? AND deleted_at IS NOT NULL`), accountID)
	if err != nil {
		return fmt.Errorf("store: claim account for purge: %w", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return nil // reactivated concurrently: skip
	}

	if err := insertAuditEventTx(ctx, tx, s.rewrite, types.AuditEvent{
		ID:        newID(),
		AccountID: accountID,
		Event:     "account.delete.purged",
		CreatedAt: time.Now(),
	}); err != nil {
		return err
	}

	if _, err := tx.ExecContext(ctx, s.rewrite(`DELETE FROM accounts WHERE id = ?`), accountID); err != nil {
		return fmt.Errorf("store: delete account: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("store: commit purge account: %w", err)
	}
	return nil
}

// HasAuditEvent reports whether accountID already has an auth_audit_log row
// for event. PurgeRunner uses this to make reminder emails idempotent
// without a dedicated "sent" column.
func (s *Store) HasAuditEvent(ctx context.Context, accountID, event string) (bool, error) {
	const q = `SELECT COUNT(*) FROM auth_audit_log WHERE account_id = ? AND event = ?`
	var n int
	if err := s.db.QueryRowContext(ctx, s.rewrite(q), accountID, event).Scan(&n); err != nil {
		return false, fmt.Errorf("store: has audit event: %w", err)
	}
	return n > 0, nil
}

// AccountEmails returns the non-empty email addresses of every user under
// accountID. PurgeRunner uses this to address retention reminder emails.
func (s *Store) AccountEmails(ctx context.Context, accountID string) ([]string, error) {
	const q = `SELECT email FROM users WHERE account_id = ? AND email IS NOT NULL AND email != ''`
	rows, err := s.db.QueryContext(ctx, s.rewrite(q), accountID)
	if err != nil {
		return nil, fmt.Errorf("store: account emails: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var emails []string
	for rows.Next() {
		var email string
		if err := rows.Scan(&email); err != nil {
			return nil, fmt.Errorf("store: scan account email: %w", err)
		}
		emails = append(emails, email)
	}
	return emails, rows.Err()
}

// ListAccountUserIDs returns the IDs of every user under accountID. Used by
// PurgeRunner to look up each user's backup_config (keyed by user_id, not
// account_id) before the account is purged, so any exported backup files
// can be deleted while the config still exists.
func (s *Store) ListAccountUserIDs(ctx context.Context, accountID string) ([]string, error) {
	var ids []string
	if err := s.db.SelectContext(ctx, &ids, s.rewrite(`SELECT id FROM users WHERE account_id = ?`), accountID); err != nil {
		return nil, fmt.Errorf("store: list account user ids: %w", err)
	}
	return ids, nil
}

// scanAccountIDs drains a single-column (id) result set into a slice.
func scanAccountIDs(rows *sql.Rows) ([]string, error) {
	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("store: scan account id: %w", err)
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
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
