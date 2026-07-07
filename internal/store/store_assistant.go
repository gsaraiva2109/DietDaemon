package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
)

// GetAssistantSettings returns the user's custom assistant instructions.
// found=false means no settings row exists yet (no error).
func (s *Store) GetAssistantSettings(ctx context.Context, userID string) (customInstructions string, found bool, err error) {
	const q = `SELECT custom_instructions FROM user_assistant_settings WHERE user_id = ?`
	var ci string
	err = s.db.QueryRowContext(ctx, s.rewrite(q), userID).Scan(&ci)
	if errors.Is(err, sql.ErrNoRows) {
		return "", false, nil
	}
	if err != nil {
		return "", false, fmt.Errorf("store: get assistant settings: %w", err)
	}
	return ci, true, nil
}

// SetAssistantSettings upserts the user's custom assistant instructions.
// Empty string clears the custom instructions.
func (s *Store) SetAssistantSettings(ctx context.Context, userID, customInstructions string) error {
	q := fmt.Sprintf(`
		INSERT INTO user_assistant_settings (user_id, custom_instructions)
		VALUES (?, ?)
		ON CONFLICT(user_id) DO UPDATE SET
			custom_instructions = excluded.custom_instructions,
			updated_at = %s
	`, s.dialect.Now())
	_, err := s.db.ExecContext(ctx, s.rewrite(q), userID, customInstructions)
	if err != nil {
		return fmt.Errorf("store: set assistant settings: %w", err)
	}
	return nil
}
