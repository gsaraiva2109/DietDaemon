package store

import (
	"context"
	"fmt"
	"time"

	"github.com/gsaraiva2109/dietdaemon/core/types"
	"github.com/gsaraiva2109/dietdaemon/internal/assistant"
)

const chatHistoryLimit = 100

// CreateChatSession inserts a new chat session for a user.
func (s *Store) CreateChatSession(ctx context.Context, id, userID, title string) error {
	const q = `INSERT INTO chat_sessions (id, user_id, title) VALUES (?, ?, ?)`
	_, err := s.db.ExecContext(ctx, s.rewrite(q), id, userID, title)
	if err != nil {
		return fmt.Errorf("store: create chat session: %w", err)
	}
	return nil
}

// ListChatSessions returns all chat sessions for a user, newest first.
// Soft-deleted sessions are excluded.
func (s *Store) ListChatSessions(ctx context.Context, userID string) ([]assistant.Session, error) {
	const q = `SELECT id, title, created_at, updated_at FROM chat_sessions WHERE user_id = ? AND deleted_at IS NULL ORDER BY updated_at DESC`
	var rows []assistant.Session
	err := s.db.SelectContext(ctx, &rows, s.rewrite(q), userID)
	if err != nil {
		return nil, fmt.Errorf("store: list chat sessions: %w", err)
	}
	if rows == nil {
		rows = []assistant.Session{}
	}
	return rows, nil
}

// AppendChatMessage inserts a message into a session and bumps the session's
// updated_at timestamp. toolName is optional; pass "" for non-tool messages.
// Returns types.ErrNotFound if sessionID doesn't belong to userID.
func (s *Store) AppendChatMessage(ctx context.Context, id, userID, sessionID, role, content, toolName string) error {
	const q = `
		INSERT INTO chat_messages (id, session_id, role, content, tool_name)
		SELECT ?, ?, ?, ?, ?
		WHERE EXISTS (SELECT 1 FROM chat_sessions WHERE id = ? AND user_id = ?)
	`
	res, err := s.db.ExecContext(ctx, s.rewrite(q), id, sessionID, role, content, toolName, sessionID, userID)
	if err != nil {
		return fmt.Errorf("store: append chat message: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return types.ErrNotFound
	}

	// Bump session timestamp.
	uq := fmt.Sprintf(`UPDATE chat_sessions SET updated_at = %s WHERE id = ?`, s.dialect.Now())
	_, _ = s.db.ExecContext(ctx, s.rewrite(uq), sessionID)
	return nil
}

// GetChatMessages returns the newest chatHistoryLimit messages in a session,
// oldest first. Returns an empty slice (not an error) if sessionID doesn't
// belong to userID.
func (s *Store) GetChatMessages(ctx context.Context, userID, sessionID string) ([]assistant.Message, error) {
	const q = `
		SELECT id, session_id, role, content, tool_name, created_at
		FROM (
			SELECT cm.id, cm.session_id, cm.role, cm.content, cm.tool_name, cm.created_at
			FROM chat_messages cm
			WHERE cm.session_id = ?
			AND EXISTS (SELECT 1 FROM chat_sessions cs WHERE cs.id = cm.session_id AND cs.user_id = ?)
			ORDER BY cm.created_at DESC, cm.id DESC
			LIMIT ?
		)
		ORDER BY created_at ASC, id ASC
	`
	var rows []assistant.Message
	err := s.db.SelectContext(ctx, &rows, s.rewrite(q), sessionID, userID, chatHistoryLimit)
	if err != nil {
		return nil, fmt.Errorf("store: get chat messages: %w", err)
	}
	if rows == nil {
		rows = []assistant.Message{}
	}
	return rows, nil
}

// SoftDeleteChatSession marks a session as deleted (deleted_at = now) rather
// than removing the row, so it can be restored within the retention window.
// Returns types.ErrNotFound if sessionID doesn't belong to userID or is
// already deleted.
func (s *Store) SoftDeleteChatSession(ctx context.Context, userID, sessionID string) error {
	q := fmt.Sprintf(`UPDATE chat_sessions SET deleted_at = %s WHERE id = ? AND user_id = ? AND deleted_at IS NULL`, s.dialect.Now())
	res, err := s.db.ExecContext(ctx, s.rewrite(q), sessionID, userID)
	if err != nil {
		return fmt.Errorf("store: soft delete chat session: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return types.ErrNotFound
	}
	return nil
}

// RestoreChatSession un-deletes a session and bumps updated_at to now, so it
// reappears at the top of the active session list.
func (s *Store) RestoreChatSession(ctx context.Context, userID, sessionID string) error {
	q := fmt.Sprintf(`UPDATE chat_sessions SET deleted_at = NULL, updated_at = %s WHERE id = ? AND user_id = ? AND deleted_at IS NOT NULL`, s.dialect.Now())
	res, err := s.db.ExecContext(ctx, s.rewrite(q), sessionID, userID)
	if err != nil {
		return fmt.Errorf("store: restore chat session: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return types.ErrNotFound
	}
	return nil
}

// ListDeletedChatSessions returns a user's soft-deleted sessions, most
// recently deleted first.
func (s *Store) ListDeletedChatSessions(ctx context.Context, userID string) ([]assistant.Session, error) {
	const q = `SELECT id, title, created_at, updated_at FROM chat_sessions WHERE user_id = ? AND deleted_at IS NOT NULL ORDER BY deleted_at DESC`
	var rows []assistant.Session
	err := s.db.SelectContext(ctx, &rows, s.rewrite(q), userID)
	if err != nil {
		return nil, fmt.Errorf("store: list deleted chat sessions: %w", err)
	}
	if rows == nil {
		rows = []assistant.Session{}
	}
	return rows, nil
}

// PurgeDeletedChatSessions permanently removes sessions soft-deleted before
// olderThan. chat_messages rows cascade via ON DELETE CASCADE.
// Returns the number of sessions purged.
func (s *Store) PurgeDeletedChatSessions(ctx context.Context, olderThan time.Time) (int, error) {
	const q = `DELETE FROM chat_sessions WHERE deleted_at IS NOT NULL AND deleted_at < ?`
	// "2006-01-02 15:04:05" matches sqlite datetime('now') output format,
	// making TEXT comparison lexicographically correct, and is also a valid
	// timestamp literal for PostgreSQL.
	res, err := s.db.ExecContext(ctx, s.rewrite(q), olderThan.UTC().Format("2006-01-02 15:04:05"))
	if err != nil {
		return 0, fmt.Errorf("store: purge deleted chat sessions: %w", err)
	}
	n, _ := res.RowsAffected()
	return int(n), nil
}
