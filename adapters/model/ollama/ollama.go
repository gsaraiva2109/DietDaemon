// Package ollama implements ports.ModelAdapter by calling the Ollama REST API
// (/api/embeddings and /api/generate). It is the model backend for Tier-1
// (embedding) and Tier-2 (LLM) parsing when PARSER_TIER > 0.
package ollama

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/gsaraiva2109/dietdaemon/core/ports"
)

// Compile-time interface check.
var _ ports.ModelAdapter = (*Adapter)(nil)

// Adapter satisfies ports.ModelAdapter via Ollama's HTTP API.
type Adapter struct {
	url    string // base URL, e.g. "http://localhost:11434"
	model  string // model name, e.g. "llama3.2", "nomic-embed-text"
	client *http.Client
}

// New returns a ready Adapter. url is the Ollama base (no trailing slash), model
// is the model tag to use for all calls.
func New(url, model string) *Adapter {
	return &Adapter{
		url:    strings.TrimRight(url, "/"),
		model:  model,
		client: &http.Client{},
	}
}

// ---------------------------------------------------------------------------
// Embed — POST /api/embeddings
// ---------------------------------------------------------------------------

type embedRequest struct {
	Model  string `json:"model"`
	Prompt string `json:"prompt"`
}

type embedResponse struct {
	Embedding []float64 `json:"embedding"`
}

// Embed returns a floating-point embedding vector for text.
func (a *Adapter) Embed(ctx context.Context, text string) ([]float32, error) {
	body := embedRequest{Model: a.model, Prompt: text}
	payload, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("ollama: marshal embed request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		a.url+"/api/embeddings", bytes.NewReader(payload))
	if err != nil {
		return nil, fmt.Errorf("ollama: build embed request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := a.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("ollama: embeddings: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("ollama: embeddings status %d", resp.StatusCode)
	}

	var er embedResponse
	if err := json.NewDecoder(resp.Body).Decode(&er); err != nil {
		return nil, fmt.Errorf("ollama: decode embeddings: %w", err)
	}

	// Convert []float64 → []float32 for the port signature.
	vec := make([]float32, len(er.Embedding))
	for i, v := range er.Embedding {
		vec[i] = float32(v)
	}
	return vec, nil
}

// ---------------------------------------------------------------------------
// Complete — POST /api/generate
// ---------------------------------------------------------------------------

type generateRequest struct {
	Model  string `json:"model"`
	Prompt string `json:"prompt"`
	Stream bool   `json:"stream"`
}

type generateResponse struct {
	Response string `json:"response"`
}

// Complete sends a prompt to the model and returns its text completion.
func (a *Adapter) Complete(ctx context.Context, prompt string) (string, error) {
	body := generateRequest{Model: a.model, Prompt: prompt, Stream: false}
	payload, err := json.Marshal(body)
	if err != nil {
		return "", fmt.Errorf("ollama: marshal generate request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		a.url+"/api/generate", bytes.NewReader(payload))
	if err != nil {
		return "", fmt.Errorf("ollama: build generate request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := a.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("ollama: generate: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("ollama: generate status %d", resp.StatusCode)
	}

	var gr generateResponse
	if err := json.NewDecoder(resp.Body).Decode(&gr); err != nil {
		return "", fmt.Errorf("ollama: decode generate: %w", err)
	}
	return gr.Response, nil
}
