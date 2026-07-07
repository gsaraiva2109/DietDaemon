package assistant

import "context"

// Session is a lightweight chat session record.
type Session struct {
	ID        string `db:"id"         json:"id"`
	Title     string `db:"title"      json:"title"`
	CreatedAt string `db:"created_at" json:"created_at"`
	UpdatedAt string `db:"updated_at" json:"updated_at"`
}

// Message is a stored message in a chat session.
type Message struct {
	ID        string `db:"id"         json:"id"`
	SessionID string `db:"session_id" json:"session_id"`
	Role      string `db:"role"       json:"role"`
	Content   string `db:"content"    json:"content"`
	ToolName  string `db:"tool_name"  json:"tool_name,omitempty"`
	CreatedAt string `db:"created_at" json:"created_at"`
}

// Store is the persistence interface the assistant router and HTTP handlers
// need. Satisfied by *store.Store.
type Store interface {
	CreateChatSession(ctx context.Context, id, userID, title string) error
	ListChatSessions(ctx context.Context, userID string) ([]Session, error)
	AppendChatMessage(ctx context.Context, id, sessionID, role, content, toolName string) error
	GetChatMessages(ctx context.Context, sessionID string) ([]Message, error)
	GetAssistantSettings(ctx context.Context, userID string) (customInstructions string, found bool, err error)
	SetAssistantSettings(ctx context.Context, userID, customInstructions string) error
}
