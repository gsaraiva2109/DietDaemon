package anthropic

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

// ChatAdapter satisfies ports.ChatAdapter via Anthropic's streaming Messages API.
type ChatAdapter struct {
	apiKey  string
	model   string
	client  *http.Client
	baseURL string
}

// NewChatAdapter returns a ready ChatAdapter for the given API key and model.
func NewChatAdapter(apiKey, model string, timeout time.Duration) *ChatAdapter {
	return &ChatAdapter{
		apiKey:  apiKey,
		model:   model,
		client:  &http.Client{Timeout: timeout},
		baseURL: defaultBaseURL,
	}
}

// --- request types ---

// chatContentBlock is Anthropic's tagged-union message content block, covering
// the three shapes this adapter needs to emit: plain text, a tool_use
// request (replayed on later turns so its matching tool_result validates),
// and a tool_result answer.
type chatContentBlock struct {
	Type      string          `json:"type"`
	Text      string          `json:"text,omitempty"`
	ID        string          `json:"id,omitempty"`
	Name      string          `json:"name,omitempty"`
	Input     json.RawMessage `json:"input,omitempty"`
	ToolUseID string          `json:"tool_use_id,omitempty"`
	Content   string          `json:"content,omitempty"`
}

type chatMessage struct {
	Role    string             `json:"role"`
	Content []chatContentBlock `json:"content"`
}

// toWireMessages converts generic ports.ChatMessage history into Anthropic's
// message format. Anthropic only accepts role "user"/"assistant": a "tool"
// message becomes a user-role tool_result block, and an assistant message
// that requested tools gets its tool_use blocks replayed alongside any text
// — Anthropic rejects a tool_result whose tool_use_id doesn't appear in the
// immediately preceding assistant turn.
func toWireMessages(msgs []ports.ChatMessage) []chatMessage {
	out := make([]chatMessage, 0, len(msgs))
	for _, m := range msgs {
		switch m.Role {
		case "tool":
			out = append(out, chatMessage{
				Role: "user",
				Content: []chatContentBlock{{
					Type:      "tool_result",
					ToolUseID: m.ToolCallID,
					Content:   m.Content,
				}},
			})
		case "assistant":
			blocks := make([]chatContentBlock, 0, 1+len(m.ToolCalls))
			if m.Content != "" {
				blocks = append(blocks, chatContentBlock{Type: "text", Text: m.Content})
			}
			for _, tc := range m.ToolCalls {
				input, _ := json.Marshal(map[string]string{"args": tc.Args})
				blocks = append(blocks, chatContentBlock{
					Type:  "tool_use",
					ID:    tc.ID,
					Name:  tc.Name,
					Input: input,
				})
			}
			out = append(out, chatMessage{Role: "assistant", Content: blocks})
		default: // "user"
			out = append(out, chatMessage{
				Role:    "user",
				Content: []chatContentBlock{{Type: "text", Text: m.Content}},
			})
		}
	}
	return out
}

type toolSpec struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	InputSchema struct {
		Type       string `json:"type"`
		Properties struct {
			Args struct {
				Type        string `json:"type"`
				Description string `json:"description"`
			} `json:"args"`
		} `json:"properties"`
		Required []string `json:"required"`
	} `json:"input_schema"`
}

type streamRequest struct {
	Model     string        `json:"model"`
	MaxTokens int           `json:"max_tokens"`
	System    string        `json:"system,omitempty"`
	Messages  []chatMessage `json:"messages"`
	Tools     []toolSpec    `json:"tools,omitempty"`
	Stream    bool          `json:"stream"`
}

// --- SSE event types ---

type sseEvent struct {
	Type string          `json:"type"`
	Data json.RawMessage `json:"-"`
}

type contentBlockStartData struct {
	Index        int `json:"index"`
	ContentBlock struct {
		Type string `json:"type"`
		ID   string `json:"id"`
		Name string `json:"name"`
	} `json:"content_block"`
}

type textDelta struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

type inputJSONDelta struct {
	Type        string `json:"type"`
	PartialJSON string `json:"partial_json"`
}

type contentBlockDeltaData struct {
	Index int             `json:"index"`
	Delta json.RawMessage `json:"delta"`
}

type errorData struct {
	Error struct {
		Type    string `json:"type"`
		Message string `json:"message"`
	} `json:"error"`
}

// StreamChat sends a streaming request to Anthropic's Messages API and returns
// a channel of ChatEvents. The channel is closed when the stream ends.
func (c *ChatAdapter) StreamChat(ctx context.Context, req ports.ChatRequest) (<-chan ports.ChatEvent, error) {
	msgs := toWireMessages(req.Messages)

	tools := make([]toolSpec, len(req.Tools))
	for i, t := range req.Tools {
		tools[i] = toolSpec{
			Name:        t.Name,
			Description: t.Description,
		}
		tools[i].InputSchema.Type = "object"
		tools[i].InputSchema.Properties.Args.Type = "string"
		tools[i].InputSchema.Properties.Args.Description = "Arguments for the command"
		tools[i].InputSchema.Required = []string{"args"}
	}

	body := streamRequest{
		Model:     c.model,
		MaxTokens: 4096,
		System:    req.System,
		Messages:  msgs,
		Tools:     tools,
		Stream:    true,
	}
	payload, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("anthropic chat: marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost,
		c.baseURL+"/v1/messages", bytes.NewReader(payload))
	if err != nil {
		return nil, fmt.Errorf("anthropic chat: build request: %w", err)
	}
	httpReq.Header.Set("x-api-key", c.apiKey)
	httpReq.Header.Set("anthropic-version", "2023-06-01")
	httpReq.Header.Set("content-type", "application/json")

	resp, err := c.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("anthropic chat: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		defer func() { _ = resp.Body.Close() }()
		detail, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
		return nil, fmt.Errorf("anthropic chat: status %d: %s", resp.StatusCode, bytes.TrimSpace(detail))
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

type streamState struct {
	currentToolID   string
	currentToolName string
	toolArgs        strings.Builder
	inToolUse       bool
}

func parseSSEEvent(line string) (sseEvent, []byte, bool) {
	if !strings.HasPrefix(line, "data: ") {
		return sseEvent{}, nil, false
	}
	data := []byte(strings.TrimPrefix(line, "data: "))
	var evt sseEvent
	if err := json.Unmarshal(data, &evt); err != nil {
		return sseEvent{}, nil, false
	}
	return evt, data, true
}

func (s *streamState) startToolUse(data []byte) {
	var cb contentBlockStartData
	if err := json.Unmarshal(data, &cb); err != nil || cb.ContentBlock.Type != "tool_use" {
		return
	}
	s.inToolUse = true
	s.currentToolID = cb.ContentBlock.ID
	s.currentToolName = cb.ContentBlock.Name
	s.toolArgs.Reset()
}

func (s *streamState) appendToolArgs(delta json.RawMessage) {
	var input inputJSONDelta
	if err := json.Unmarshal(delta, &input); err == nil && input.Type == "input_json_delta" {
		s.toolArgs.WriteString(input.PartialJSON)
	}
}

func (s *streamState) sendTextDelta(ctx context.Context, ch chan<- ports.ChatEvent, delta json.RawMessage) bool {
	var text textDelta
	if err := json.Unmarshal(delta, &text); err != nil || text.Type != "text_delta" {
		return true
	}
	return sendEvent(ctx, ch, ports.ChatEvent{Kind: "text-delta", Text: text.Text})
}

func (s *streamState) handleDelta(ctx context.Context, ch chan<- ports.ChatEvent, data []byte) bool {
	var delta contentBlockDeltaData
	if err := json.Unmarshal(data, &delta); err != nil {
		return true
	}
	if s.inToolUse {
		s.appendToolArgs(delta.Delta)
		return true
	}
	return s.sendTextDelta(ctx, ch, delta.Delta)
}

func (s *streamState) finishToolUse(ctx context.Context, ch chan<- ports.ChatEvent) bool {
	if !s.inToolUse {
		return true
	}
	if !sendEvent(ctx, ch, ports.ChatEvent{
		Kind: "tool-call",
		ToolCall: &ports.ToolCallEvent{
			ID:   s.currentToolID,
			Name: s.currentToolName,
			Args: extractArgs(s.toolArgs.String()),
		},
	}) {
		return false
	}
	s.inToolUse = false
	return true
}

func sendProviderError(ctx context.Context, ch chan<- ports.ChatEvent, data []byte) bool {
	var providerError errorData
	if err := json.Unmarshal(data, &providerError); err != nil {
		return true
	}
	sendEvent(ctx, ch, ports.ChatEvent{
		Kind: "error",
		Err:  fmt.Errorf("anthropic: %s: %s", providerError.Error.Type, providerError.Error.Message),
	})
	return false
}

func (s *streamState) handleEvent(ctx context.Context, ch chan<- ports.ChatEvent, evt sseEvent, data []byte) bool {
	switch evt.Type {
	case "content_block_start":
		s.startToolUse(data)
	case "content_block_delta":
		return s.handleDelta(ctx, ch, data)
	case "content_block_stop":
		return s.finishToolUse(ctx, ch)
	case "message_stop":
		sendEvent(ctx, ch, ports.ChatEvent{Kind: "done"})
		return false
	case "error":
		return sendProviderError(ctx, ch, data)
	}
	return true
}

func (c *ChatAdapter) readStream(ctx context.Context, body io.ReadCloser, ch chan<- ports.ChatEvent) {
	defer close(ch)
	defer func() { _ = body.Close() }()

	scanner := bufio.NewScanner(body)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024) // 1 MB max token

	var state streamState

	for scanner.Scan() {
		evt, data, ok := parseSSEEvent(scanner.Text())
		if !ok {
			continue
		}
		if !state.handleEvent(ctx, ch, evt, data) {
			return
		}
	}

	if err := scanner.Err(); err != nil && err != io.EOF {
		sendEvent(ctx, ch, ports.ChatEvent{
			Kind: "error",
			Err:  fmt.Errorf("anthropic chat: read stream: %w", err),
		})
	}
}

// extractArgs tries to parse {"args": "..."} from accumulated tool-use JSON.
// Falls back to returning the raw accumulated text if parsing fails.
func extractArgs(raw string) string {
	var obj struct {
		Args string `json:"args"`
	}
	if err := json.Unmarshal([]byte(raw), &obj); err == nil {
		return obj.Args
	}
	// ponytail: partial JSON may not parse; return raw, caller handles it.
	return raw
}
