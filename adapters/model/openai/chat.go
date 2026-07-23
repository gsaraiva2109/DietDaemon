package openai

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/gsaraiva2109/dietdaemon/core/ports"
)

// Compile-time interface check.
var _ ports.ChatAdapter = (*ChatAdapter)(nil)

// ChatAdapter satisfies ports.ChatAdapter via OpenAI's streaming chat/completions API.
type ChatAdapter struct {
	apiKey  string
	model   string
	client  *http.Client
	baseURL string
}

// NewChatAdapter returns a ready ChatAdapter for the given API key and model.
// baseURL is the API base, e.g. "https://api.openai.com/v1" (no trailing slash).
func NewChatAdapter(baseURL, apiKey, model string, timeout time.Duration) *ChatAdapter {
	return &ChatAdapter{
		apiKey:  apiKey,
		model:   model,
		client:  &http.Client{Timeout: timeout},
		baseURL: strings.TrimRight(baseURL, "/"),
	}
}

// --- request types ---

type openaiToolSpec struct {
	Type     string `json:"type"`
	Function struct {
		Name        string `json:"name"`
		Description string `json:"description"`
		Parameters  struct {
			Type       string `json:"type"`
			Properties struct {
				Args struct {
					Type        string `json:"type"`
					Description string `json:"description"`
				} `json:"args"`
			} `json:"properties"`
			Required []string `json:"required"`
		} `json:"parameters"`
	} `json:"function"`
}

type openaiToolCall struct {
	ID       string `json:"id"`
	Type     string `json:"type"`
	Function struct {
		Name      string `json:"name"`
		Arguments string `json:"arguments"`
	} `json:"function"`
}

type openaiMessage struct {
	Role       string           `json:"role"`
	Content    string           `json:"content,omitempty"`
	ToolCalls  []openaiToolCall `json:"tool_calls,omitempty"`
	ToolCallID string           `json:"tool_call_id,omitempty"`
}

type openaiStreamRequest struct {
	Model     string           `json:"model"`
	MaxTokens int              `json:"max_tokens"`
	System    string           `json:"-"`
	Messages  []openaiMessage  `json:"messages"`
	Tools     []openaiToolSpec `json:"tools,omitempty"`
	Stream    bool             `json:"stream"`
}

// MarshalJSON openaiStreamRequest marshals system as a separate message because OpenAI puts
// it in the messages array, not as a top-level "system" field.
func (r openaiStreamRequest) MarshalJSON() ([]byte, error) {
	msgs := make([]openaiMessage, 0, len(r.Messages)+1)
	if r.System != "" {
		msgs = append(msgs, openaiMessage{Role: "system", Content: r.System})
	}
	msgs = append(msgs, r.Messages...)

	type alias openaiStreamRequest
	return json.Marshal(&struct {
		Messages []openaiMessage `json:"messages"`
		*alias
	}{
		Messages: msgs,
		alias:    (*alias)(&r),
	})
}

// toWireMessages converts generic ports.ChatMessage history into OpenAI format.
// OpenAI: "tool" messages use role "tool" + tool_call_id, assistant messages
// preserve tool_calls for history, and regular user/assistant messages pass through.
func toWireMessages(msgs []ports.ChatMessage) []openaiMessage {
	out := make([]openaiMessage, 0, len(msgs))
	for _, m := range msgs {
		switch m.Role {
		case "tool":
			out = append(out, openaiMessage{
				Role:       "tool",
				Content:    m.Content,
				ToolCallID: m.ToolCallID,
			})
		case "assistant":
			om := openaiMessage{Role: "assistant", Content: m.Content}
			if len(m.ToolCalls) > 0 {
				om.ToolCalls = make([]openaiToolCall, len(m.ToolCalls))
				for i, tc := range m.ToolCalls {
					om.ToolCalls[i] = openaiToolCall{
						ID:   tc.ID,
						Type: "function",
					}
					om.ToolCalls[i].Function.Name = tc.Name
					// OpenAI expects arguments as a JSON string.
					argsJSON, _ := json.Marshal(map[string]string{"args": tc.Args})
					om.ToolCalls[i].Function.Arguments = string(argsJSON)
				}
			}
			out = append(out, om)
		default: // "user"
			out = append(out, openaiMessage{Role: "user", Content: m.Content})
		}
	}
	return out
}

// --- SSE event types ---

type openaiStreamChoice struct {
	Index        int         `json:"index"`
	Delta        openaiDelta `json:"delta"`
	FinishReason string      `json:"finish_reason"`
}

type openaiDelta struct {
	Content   string                `json:"content,omitempty"`
	ToolCalls []openaiToolCallDelta `json:"tool_calls,omitempty"`
}

type openaiToolCallDelta struct {
	Index    int    `json:"index"`
	ID       string `json:"id,omitempty"`
	Type     string `json:"type,omitempty"`
	Function struct {
		Name      string `json:"name,omitempty"`
		Arguments string `json:"arguments,omitempty"`
	} `json:"function,omitempty"`
}

type openaiStreamResponse struct {
	Choices []openaiStreamChoice `json:"choices"`
}

// StreamChat sends a streaming request to OpenAI's chat/completions API.
func (c *ChatAdapter) StreamChat(ctx context.Context, req ports.ChatRequest) (<-chan ports.ChatEvent, error) {
	msgs := toWireMessages(req.Messages)

	tools := make([]openaiToolSpec, len(req.Tools))
	for i, t := range req.Tools {
		tools[i].Type = "function"
		tools[i].Function.Name = t.Name
		tools[i].Function.Description = t.Description
		tools[i].Function.Parameters.Type = "object"
		tools[i].Function.Parameters.Properties.Args.Type = "string"
		tools[i].Function.Parameters.Properties.Args.Description = "Arguments for the command"
		tools[i].Function.Parameters.Required = []string{"args"}
	}

	body := openaiStreamRequest{
		Model:     c.model,
		MaxTokens: 4096,
		System:    req.System,
		Messages:  msgs,
		Tools:     tools,
		Stream:    true,
	}
	payload, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("openai chat: marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost,
		c.baseURL+"/chat/completions", bytes.NewReader(payload))
	if err != nil {
		return nil, fmt.Errorf("openai chat: build request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	if c.apiKey != "" {
		httpReq.Header.Set("Authorization", "Bearer "+c.apiKey)
	}

	resp, err := c.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("openai chat: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		defer func() { _ = resp.Body.Close() }()
		detail, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
		return nil, fmt.Errorf("openai chat: status %d: %s", resp.StatusCode, bytes.TrimSpace(detail))
	}

	ch := make(chan ports.ChatEvent, 32)
	go c.readStream(ctx, resp.Body, ch)
	return ch, nil
}

// sendEvent delivers evt to ch, or bails if ctx is cancelled first — without
// this, a client that disconnects mid-stream while the channel's buffer is
// full leaks this goroutine (and its open upstream connection) forever.
func sendEvent(ctx context.Context, ch chan<- ports.ChatEvent, evt ports.ChatEvent) bool {
	select {
	case ch <- evt:
		return true
	default:
	}
	select {
	case ch <- evt:
		return true
	case <-ctx.Done():
		return false
	}
}

func (c *ChatAdapter) readStream(ctx context.Context, body io.ReadCloser, ch chan<- ports.ChatEvent) {
	defer close(ch)
	defer func() { _ = body.Close() }()

	scanner := bufio.NewScanner(body)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	pending := make(map[int]*pendingTool)

	for scanner.Scan() {
		if !handleStreamLine(ctx, ch, scanner.Text(), &pending) {
			return
		}
	}

	if err := scanner.Err(); err != nil && err != io.EOF {
		sendEvent(ctx, ch, ports.ChatEvent{
			Kind: "error",
			Err:  fmt.Errorf("openai chat: read stream: %w", err),
		})
	}
}

// pendingTool accumulates OpenAI's incremental tool-call data by stream index.
type pendingTool struct {
	id   string
	name string
	args strings.Builder
}

func handleStreamLine(ctx context.Context, ch chan<- ports.ChatEvent, line string, pending *map[int]*pendingTool) bool {
	if !strings.HasPrefix(line, "data: ") {
		return true
	}
	data := strings.TrimPrefix(line, "data: ")
	if data == "[DONE]" {
		if !flushPendingTools(ctx, ch, *pending) {
			return false
		}
		sendEvent(ctx, ch, ports.ChatEvent{Kind: "done"})
		return false
	}

	var sr openaiStreamResponse
	if err := json.Unmarshal([]byte(data), &sr); err != nil || len(sr.Choices) == 0 {
		return true
	}
	return handleStreamChoice(ctx, ch, sr.Choices[0], pending)
}

func handleStreamChoice(ctx context.Context, ch chan<- ports.ChatEvent, choice openaiStreamChoice, pending *map[int]*pendingTool) bool {
	if choice.Delta.Content != "" && !sendEvent(ctx, ch, ports.ChatEvent{Kind: "text-delta", Text: choice.Delta.Content}) {
		return false
	}
	appendToolCallDeltas(*pending, choice.Delta.ToolCalls)
	if choice.FinishReason == "" {
		return true
	}
	if !flushPendingTools(ctx, ch, *pending) {
		return false
	}
	*pending = make(map[int]*pendingTool)
	return true
}

func appendToolCallDeltas(pending map[int]*pendingTool, calls []openaiToolCallDelta) {
	for _, call := range calls {
		tool := pending[call.Index]
		if tool == nil {
			tool = &pendingTool{}
			pending[call.Index] = tool
		}
		if call.ID != "" {
			tool.id = call.ID
		}
		if call.Function.Name != "" {
			tool.name = call.Function.Name
		}
		tool.args.WriteString(call.Function.Arguments)
	}
}

func flushPendingTools(ctx context.Context, ch chan<- ports.ChatEvent, pending map[int]*pendingTool) bool {
	for _, tool := range pending {
		if !sendEvent(ctx, ch, ports.ChatEvent{
			Kind: "tool-call",
			ToolCall: &ports.ToolCallEvent{
				ID:   tool.id,
				Name: tool.name,
				Args: extractArgs(tool.args.String()),
			},
		}) {
			return false
		}
	}
	return true
}

// extractArgs tries to parse {"args": "..."} from accumulated tool-call arguments.
// Falls back to returning the raw accumulated text if parsing fails.
func extractArgs(raw string) string {
	var obj struct {
		Args string `json:"args"`
	}
	if err := json.Unmarshal([]byte(raw), &obj); err == nil {
		return obj.Args
	}
	return raw
}
