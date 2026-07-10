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

	// ToolCalls is set on "assistant" messages that requested tool calls this
	// round. Adapters must replay these in history (e.g. Anthropic's tool_use
	// content blocks) — providers require the original call to still be
	// present alongside its result on the next turn.
	ToolCalls []ToolCallEvent

	// ToolCallID is set on "tool" messages: the ID of the ToolCallEvent this
	// result answers, so adapters can link result to call (e.g. Anthropic's
	// tool_result.tool_use_id, OpenAI's tool_call_id).
	ToolCallID string

	// ToolName is set on "tool" messages alongside ToolCallID. Ollama's
	// /api/chat links tool results back to calls by function name
	// (tool_name), not by ID — adapters that need the name read it from
	// here instead of deriving it from the preceding assistant turn.
	ToolName string
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
	Kind        string // "text-delta" | "tool-call" | "done" | "error" | "suggestions"
	Text        string
	ToolCall    *ToolCallEvent
	Suggestions []string // set when Kind == "suggestions"
	Err         error
}
