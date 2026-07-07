package store

import (
	"context"
	"fmt"

	"github.com/gsaraiva2109/dietdaemon/internal/assistant"
)

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
func (s *Store) ListChatSessions(ctx context.Context, userID string) ([]assistant.Session, error) {
	const q = `SELECT id, title, created_at, updated_at FROM chat_sessions WHERE user_id = ? ORDER BY updated_at DESC`
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
func (s *Store) AppendChatMessage(ctx context.Context, id, sessionID, role, content, toolName string) error {
	const q = `INSERT INTO chat_messages (id, session_id, role, content, tool_name) VALUES (?, ?, ?, ?, ?)`
	_, err := s.db.ExecContext(ctx, s.rewrite(q), id, sessionID, role, content, toolName)
	if err != nil {
		return fmt.Errorf("store: append chat message: %w", err)
	}

	// Bump session timestamp.
	uq := fmt.Sprintf(`UPDATE chat_sessions SET updated_at = %s WHERE id = ?`, s.dialect.Now())
	_, _ = s.db.ExecContext(ctx, s.rewrite(uq), sessionID)
	return nil
}

// GetChatMessages returns all messages in a session, oldest first.
func (s *Store) GetChatMessages(ctx context.Context, sessionID string) ([]assistant.Message, error) {
	const q = `SELECT id, session_id, role, content, tool_name, created_at FROM chat_messages WHERE session_id = ? ORDER BY created_at ASC`
	var rows []assistant.Message
	err := s.db.SelectContext(ctx, &rows, s.rewrite(q), sessionID)
	if err != nil {
		return nil, fmt.Errorf("store: get chat messages: %w", err)
	}
	if rows == nil {
		rows = []assistant.Message{}
	}
	return rows, nil
}
