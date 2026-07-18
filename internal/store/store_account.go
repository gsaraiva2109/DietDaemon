package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

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
