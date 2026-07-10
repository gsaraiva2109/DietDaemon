// Package openai implements ports.ModelAdapter against any OpenAI-compatible
// chat completions endpoint (real OpenAI, local vLLM, Gemini's OpenAI-compat
// endpoint). Reached only via COMPLETION_ADAPTER=openai; it never serves
// embeddings — use ollama for those.
package openai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/gsaraiva2109/dietdaemon/adapters/model/internal/jsonfence"
	"github.com/gsaraiva2109/dietdaemon/core/ports"
)

// Compile-time interface check.
var _ ports.ModelAdapter = (*Adapter)(nil)

// Adapter satisfies ports.ModelAdapter via an OpenAI-compatible chat API.
type Adapter struct {
	baseURL string // e.g. "https://api.openai.com/v1", no trailing slash
	apiKey  string // may be empty — local backends like vLLM often need no auth
	model   string
	client  *http.Client
}

// New returns a ready Adapter. baseURL is the API base (no trailing slash),
// apiKey may be empty for backends that require no auth, model is the chat
// model name, timeout applies to the request.
func New(baseURL, apiKey, model string, timeout time.Duration) *Adapter {
	return &Adapter{
		baseURL: strings.TrimRight(baseURL, "/"),
		apiKey:  apiKey,
		model:   model,
		client:  &http.Client{Timeout: timeout},
	}
}

// Embed is not supported by this adapter; embeddings always go through ollama.
func (a *Adapter) Embed(ctx context.Context, text string) ([]float32, error) {
	return nil, fmt.Errorf("openai: embeddings not supported, set EMBED_ADAPTER=ollama")
}

// ---------------------------------------------------------------------------
// Complete — POST {baseURL}/chat/completions
// ---------------------------------------------------------------------------

type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type responseFormat struct {
	Type string `json:"type"`
}

type chatRequest struct {
	Model          string          `json:"model"`
	Messages       []chatMessage   `json:"messages"`
	ResponseFormat *responseFormat `json:"response_format"`
}

type chatResponse struct {
	Choices []struct {
		Message chatMessage `json:"message"`
	} `json:"choices"`
}

// Complete sends prompt as a single user message and returns the model's text
// completion. The model is asked for JSON output via response_format, and the
// returned content is defensively stripped of markdown fences in case a
// backend ignores that.
func (a *Adapter) Complete(ctx context.Context, prompt string) (string, error) {
	body := chatRequest{
		Model:          a.model,
		Messages:       []chatMessage{{Role: "user", Content: prompt}},
		ResponseFormat: &responseFormat{Type: "json_object"},
	}
	payload, err := json.Marshal(body)
	if err != nil {
		return "", fmt.Errorf("openai: marshal chat request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		a.baseURL+"/chat/completions", bytes.NewReader(payload))
	if err != nil {
		return "", fmt.Errorf("openai: build chat request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if a.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+a.apiKey)
	}

	resp, err := a.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("openai: chat completions: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		detail, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
		return "", fmt.Errorf("openai: chat completions status %d: %s", resp.StatusCode, bytes.TrimSpace(detail))
	}

	var cr chatResponse
	if err := json.NewDecoder(resp.Body).Decode(&cr); err != nil {
		return "", fmt.Errorf("openai: decode chat completions: %w", err)
	}
	if len(cr.Choices) == 0 {
		return "", fmt.Errorf("openai: chat completions returned no choices")
	}
	return jsonfence.Strip(cr.Choices[0].Message.Content), nil
}
