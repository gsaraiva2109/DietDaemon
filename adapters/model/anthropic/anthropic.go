// Package anthropic implements ports.ModelAdapter by calling Anthropic's
// Messages API (/v1/messages). It is the completion backend selected via
// COMPLETION_ADAPTER=anthropic; it never serves embeddings.
package anthropic

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/gsaraiva2109/dietdaemon/adapters/model/internal/jsonfence"
	"github.com/gsaraiva2109/dietdaemon/core/ports"
)

// Compile-time interface check.
var _ ports.ModelAdapter = (*Adapter)(nil)

const defaultBaseURL = "https://api.anthropic.com"

// Adapter satisfies ports.ModelAdapter via Anthropic's Messages API.
type Adapter struct {
	apiKey  string
	model   string
	client  *http.Client
	baseURL string // unexported; defaults to defaultBaseURL, overridable in tests
}

// New returns a ready Adapter for the given API key and model, e.g.
// "claude-haiku-4-5-20251001".
func New(apiKey, model string, timeout time.Duration) *Adapter {
	return &Adapter{
		apiKey:  apiKey,
		model:   model,
		client:  &http.Client{Timeout: timeout},
		baseURL: defaultBaseURL,
	}
}

// Embed is not supported by this adapter: Anthropic does not offer an
// embeddings endpoint, so embeddings must go through a separate provider.
func (a *Adapter) Embed(ctx context.Context, text string) ([]float32, error) {
	return nil, fmt.Errorf("anthropic: embeddings not supported, set EMBED_ADAPTER=ollama")
}

// ---------------------------------------------------------------------------
// Complete — POST /v1/messages
// ---------------------------------------------------------------------------

type message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type messagesRequest struct {
	Model     string    `json:"model"`
	MaxTokens int       `json:"max_tokens"`
	Messages  []message `json:"messages"`
}

type contentBlock struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

type messagesResponse struct {
	Content []contentBlock `json:"content"`
}

// Complete sends a prompt to the model and returns its text completion,
// stripped of any markdown fences the model may have wrapped it in.
func (a *Adapter) Complete(ctx context.Context, prompt string) (string, error) {
	body := messagesRequest{
		Model:     a.model,
		MaxTokens: 1024,
		Messages: []message{
			{Role: "user", Content: prompt + "\n\nRespond with ONLY valid JSON. No markdown fences, no commentary."},
		},
	}
	payload, err := json.Marshal(body)
	if err != nil {
		return "", fmt.Errorf("anthropic: marshal messages request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		a.baseURL+"/v1/messages", bytes.NewReader(payload))
	if err != nil {
		return "", fmt.Errorf("anthropic: build messages request: %w", err)
	}
	req.Header.Set("x-api-key", a.apiKey)
	req.Header.Set("anthropic-version", "2023-06-01")
	req.Header.Set("content-type", "application/json")

	resp, err := a.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("anthropic: messages: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("anthropic: messages status %d", resp.StatusCode)
	}

	var mr messagesResponse
	if err := json.NewDecoder(resp.Body).Decode(&mr); err != nil {
		return "", fmt.Errorf("anthropic: decode messages: %w", err)
	}
	if len(mr.Content) == 0 {
		return "", fmt.Errorf("anthropic: empty content in response")
	}

	return jsonfence.Strip(mr.Content[0].Text), nil
}
