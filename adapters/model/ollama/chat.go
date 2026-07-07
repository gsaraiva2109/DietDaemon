package ollama

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

// ChatAdapter satisfies ports.ChatAdapter via Ollama's /api/chat endpoint.
// Streaming uses NDJSON (one JSON object per line), not SSE.
type ChatAdapter struct {
	url    string // base URL, e.g. "http://localhost:11434"
	model  string // chat model, e.g. "llama3.1"
	client *http.Client
}

// NewChatAdapter returns a ready ChatAdapter. url is the Ollama base
// (no trailing slash), model is the chat model name, timeout applies to
// every request.
func NewChatAdapter(url, model string, timeout time.Duration) *ChatAdapter {
	return &ChatAdapter{
		url:    strings.TrimRight(url, "/"),
		model:  model,
		client: &http.Client{Timeout: timeout},
	}
}

// --- request types ---

type ollamaToolSpec struct {
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

type ollamaToolCall struct {
	Function struct {
		Name      string          `json:"name"`
		Arguments json.RawMessage `json:"arguments"`
	} `json:"function"`
}

type ollamaMessage struct {
	Role      string           `json:"role"`
	Content   string           `json:"content"`
	ToolCalls []ollamaToolCall `json:"tool_calls,omitempty"`
	ToolName  string           `json:"tool_name,omitempty"`
}

type ollamaChatRequest struct {
	Model    string           `json:"model"`
	Messages []ollamaMessage  `json:"messages"`
	Tools    []ollamaToolSpec `json:"tools,omitempty"`
	Stream   bool             `json:"stream"`
	Options  struct {
		NumPredict int `json:"num_predict"`
	} `json:"options"`
}

// toWireMessages converts generic ports.ChatMessage history into Ollama format.
// Ollama differs from both Anthropic and OpenAI:
//   - Assistant tool_calls: arguments is a parsed JSON OBJECT, not a string.
//   - Tool results: linked by tool_name (function name), not by ID.
func toWireMessages(msgs []ports.ChatMessage) []ollamaMessage {
	out := make([]ollamaMessage, 0, len(msgs))
	for _, m := range msgs {
		switch m.Role {
		case "tool":
			out = append(out, ollamaMessage{
				Role:     "tool",
				Content:  m.Content,
				ToolName: m.ToolName,
			})
		case "assistant":
			om := ollamaMessage{Role: "assistant", Content: m.Content}
			if len(m.ToolCalls) > 0 {
				om.ToolCalls = make([]ollamaToolCall, len(m.ToolCalls))
				for i, tc := range m.ToolCalls {
					// Ollama expects arguments as a parsed JSON object, not a
					// string. Marshal {"args": tc.Args} into RawMessage.
					argsJSON, _ := json.Marshal(map[string]string{"args": tc.Args})
					om.ToolCalls[i].Function.Name = tc.Name
					om.ToolCalls[i].Function.Arguments = argsJSON
				}
			}
			out = append(out, om)
		default: // "user"
			out = append(out, ollamaMessage{Role: "user", Content: m.Content})
		}
	}
	return out
}

// --- NDJSON streaming types ---

type ollamaStreamChunk struct {
	Model      string        `json:"model"`
	CreatedAt  string        `json:"created_at"`
	Message    ollamaMessage `json:"message"`
	Done       bool          `json:"done"`
	DoneReason string        `json:"done_reason"`
}

// StreamChat sends a streaming request to Ollama's /api/chat endpoint.
func (c *ChatAdapter) StreamChat(ctx context.Context, req ports.ChatRequest) (<-chan ports.ChatEvent, error) {
	msgs := toWireMessages(req.Messages)

	// Prepend system prompt as a system message (Ollama places it in the
	// messages array, same convention as OpenAI).
	allMsgs := make([]ollamaMessage, 0, len(msgs)+1)
	if req.System != "" {
		allMsgs = append(allMsgs, ollamaMessage{Role: "system", Content: req.System})
	}
	allMsgs = append(allMsgs, msgs...)

	tools := make([]ollamaToolSpec, len(req.Tools))
	for i, t := range req.Tools {
		tools[i].Type = "function"
		tools[i].Function.Name = t.Name
		tools[i].Function.Description = t.Description
		tools[i].Function.Parameters.Type = "object"
		tools[i].Function.Parameters.Properties.Args.Type = "string"
		tools[i].Function.Parameters.Properties.Args.Description = "Arguments for the command"
		tools[i].Function.Parameters.Required = []string{"args"}
	}

	body := ollamaChatRequest{
		Model:    c.model,
		Messages: allMsgs,
		Tools:    tools,
		Stream:   true,
	}
	body.Options.NumPredict = 4096

	payload, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("ollama chat: marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost,
		c.url+"/api/chat", bytes.NewReader(payload))
	if err != nil {
		return nil, fmt.Errorf("ollama chat: build request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("ollama chat: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return nil, fmt.Errorf("ollama chat: status %d", resp.StatusCode)
	}

	ch := make(chan ports.ChatEvent, 32)
	go c.readStream(resp.Body, ch)
	return ch, nil
}

func (c *ChatAdapter) readStream(body io.ReadCloser, ch chan<- ports.ChatEvent) {
	defer close(ch)
	defer body.Close()

	scanner := bufio.NewScanner(body)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}

		var chunk ollamaStreamChunk
		if err := json.Unmarshal([]byte(line), &chunk); err != nil {
			continue
		}

		msg := chunk.Message

		// Text delta: Ollama streams partial content in message.content.
		if msg.Content != "" && len(msg.ToolCalls) == 0 {
			ch <- ports.ChatEvent{Kind: "text-delta", Text: msg.Content}
		}

		// Tool calls: Ollama delivers them complete in a single chunk
		// (not incrementally like OpenAI).
		for _, tc := range msg.ToolCalls {
			ch <- ports.ChatEvent{
				Kind: "tool-call",
				ToolCall: &ports.ToolCallEvent{
					ID:   tc.Function.Name, // Ollama has no tool-call ID; use name as ID
					Name: tc.Function.Name,
					Args: extractArgsOllama(tc.Function.Arguments),
				},
			}
		}

		if chunk.Done {
			ch <- ports.ChatEvent{Kind: "done"}
			return
		}
	}

	if err := scanner.Err(); err != nil && err != io.EOF {
		ch <- ports.ChatEvent{
			Kind: "error",
			Err:  fmt.Errorf("ollama chat: read stream: %w", err),
		}
	}
}

// extractArgsOllama extracts the "args" string from Ollama's tool-call arguments.
// Ollama returns arguments as a parsed JSON object (e.g. {"args": "some text"}),
// not as a JSON-encoded string like OpenAI.
func extractArgsOllama(raw json.RawMessage) string {
	var obj struct {
		Args string `json:"args"`
	}
	if err := json.Unmarshal(raw, &obj); err == nil && obj.Args != "" {
		return obj.Args
	}
	// ponytail: fall back to the raw JSON if "args" key isn't present.
	return string(raw)
}
