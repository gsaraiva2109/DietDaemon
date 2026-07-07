package store

import (
	"context"
	"time"

	"github.com/gsaraiva2109/dietdaemon/core/types"
)

// ---------------------------------------------------------------------------
// Linking codes
// ---------------------------------------------------------------------------

// CreateLinkingCode inserts a new one-time linking code. The code expires after
// 10 minutes. The caller is responsible for generating the 6-char code.
func (s *Store) CreateLinkingCode(ctx context.Context, userID, platform, code string) error {
	expiresAt := time.Now().UTC().Add(10 * time.Minute).Format("2006-01-02 15:04:05")
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO linking_codes (code, user_id, platform, expires_at) VALUES (?, ?, ?, ?)`,
		code, userID, platform, expiresAt,
	)
	return err
}

// LookupLinkingCode returns an unused linking code by its code string.
func (s *Store) LookupLinkingCode(ctx context.Context, code string) (types.LinkingCode, error) {
	var lc types.LinkingCode
	err := s.db.GetContext(ctx, &lc,
		`SELECT code, user_id, platform, expires_at, COALESCE(used_at, '') AS used_at FROM linking_codes WHERE code = ? AND used_at IS NULL`,
		code,
	)
	return lc, err
}

// LookupLinkingCodeAny returns a linking code regardless of whether it has been
// used. The SSE stream uses this to detect the transition from unused → used
// (LookupLinkingCode filters used_at IS NULL and would miss the transition).
func (s *Store) LookupLinkingCodeAny(ctx context.Context, code string) (types.LinkingCode, error) {
	var lc types.LinkingCode
	err := s.db.GetContext(ctx, &lc,
		`SELECT code, user_id, platform, expires_at, COALESCE(used_at, '') AS used_at FROM linking_codes WHERE code = ?`,
		code,
	)
	return lc, err
}

// ConsumeLinkingCode marks a linking code as used.
func (s *Store) ConsumeLinkingCode(ctx context.Context, code string) error {
	_, err := s.db.ExecContext(ctx,
		s.rewrite(`UPDATE linking_codes SET used_at = `+s.dialect.Now()+` WHERE code = ? AND used_at IS NULL`),
		code,
	)
	return err
}
