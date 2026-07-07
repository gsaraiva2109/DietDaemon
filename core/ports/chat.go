package ports

import "context"

// ChatAdapter streams multi-turn chat completions from an LLM backend.
// Unlike ModelAdapter (single-shot, JSON-mode), this interface supports
// token-by-token streaming and tool-calling for the conversational assistant.
type ChatAdapter interface {
	StreamChat(ctx context.Context, req ChatRequest) (<-chan ChatEvent, error)
}

// ChatMessage is a single message in a conversation.
type ChatMessage struct {
	Role    string // "user" | "assistant" | "tool"
	Content string
}

// ToolSpec describes a tool the model may call.
type ToolSpec struct {
	Name        string
	Description string
	// Input schema is a single free-text "args" string, mirroring
	// ports.Command.Handle's `args string` parameter.
}

// ChatRequest bundles everything needed for a streaming chat call.
type ChatRequest struct {
	System   string
	Messages []ChatMessage
	Tools    []ToolSpec
}

// ToolCallEvent represents a model-requested tool invocation.
type ToolCallEvent struct {
	ID   string
	Name string
	Args string
}

// ChatEvent is a single event from a streaming chat response.
type ChatEvent struct {
	Kind     string // "text-delta" | "tool-call" | "done" | "error"
	Text     string
	ToolCall *ToolCallEvent
	Err      error
}
