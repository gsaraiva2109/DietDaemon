// Package ollama implements ports.ModelAdapter by calling the Ollama REST API
// (/api/embeddings and /api/generate). It is the model backend for Tier-1
// (embedding) and Tier-2 (LLM) parsing when PARSER_TIER > 0.
package ollama

import (
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
var _ ports.ModelAdapter = (*Adapter)(nil)

// Adapter satisfies ports.ModelAdapter via Ollama's HTTP API.
type Adapter struct {
	url         string // base URL, e.g. "http://localhost:11434"
	embedModel  string // model for embeddings, e.g. "nomic-embed-text"
	llmModel    string // model for completions, e.g. "llama3.1"
	visionModel string // model for ExtractLabel, e.g. "llava"; set via SetVisionModel
	client      *http.Client
}

// SetVisionModel sets the model used by ExtractLabel (e.g. "llava"), which is
// distinct from llmModel since a default chat model like llama3.1 has no
// vision support. A setter avoids changing New's signature and every
// existing call site.
func (a *Adapter) SetVisionModel(model string) {
	a.visionModel = model
}

// New returns a ready Adapter. url is the Ollama base (no trailing slash),
// embedModel is the model for /api/embeddings, llmModel is the model for
// /api/generate, timeout applies to both endpoints.
func New(url, embedModel, llmModel string, timeout time.Duration) *Adapter {
	return &Adapter{
		url:        strings.TrimRight(url, "/"),
		embedModel: embedModel,
		llmModel:   llmModel,
		client:     &http.Client{Timeout: timeout},
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
	body := embedRequest{Model: a.embedModel, Prompt: text}
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
	defer func() { _ = resp.Body.Close() }()

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
	Format string `json:"format"`
}

type generateResponse struct {
	Response string `json:"response"`
}

const maxTagsResponseBytes = 4 << 20

type tagsResponse struct {
	Models []struct {
		Name string `json:"name"`
	} `json:"models"`
}

type pullRequest struct {
	Model  string `json:"model"`
	Stream bool   `json:"stream"`
}

// EnsureModels pulls configured models that are not already present in Ollama.
// Pull requests intentionally have no client timeout: downloading a model can
// take much longer than normal inference. The caller's context still cancels
// the request during shutdown.
func (a *Adapter) EnsureModels(ctx context.Context, models ...string) error {
	required := uniqueModels(models)
	if len(required) == 0 {
		return nil
	}

	installed, err := a.installedModels(ctx)
	if err != nil {
		return err
	}
	for _, model := range required {
		if installed[model] || (!strings.Contains(model, ":") && installed[model+":latest"]) {
			continue
		}
		if err := a.pull(ctx, model); err != nil {
			return err
		}
	}
	return nil
}

func uniqueModels(models []string) []string {
	seen := make(map[string]struct{}, len(models))
	unique := make([]string, 0, len(models))
	for _, model := range models {
		model = strings.TrimSpace(model)
		if model == "" {
			continue
		}
		if _, ok := seen[model]; ok {
			continue
		}
		seen[model] = struct{}{}
		unique = append(unique, model)
	}
	return unique
}

func (a *Adapter) installedModels(ctx context.Context) (map[string]bool, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, a.url+"/api/tags", nil)
	if err != nil {
		return nil, fmt.Errorf("ollama: build tags request: %w", err)
	}
	resp, err := a.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("ollama: list models: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("ollama: list models status %d", resp.StatusCode)
	}

	var tags tagsResponse
	if err := json.NewDecoder(io.LimitReader(resp.Body, maxTagsResponseBytes)).Decode(&tags); err != nil {
		return nil, fmt.Errorf("ollama: decode model list: %w", err)
	}
	installed := make(map[string]bool, len(tags.Models))
	for _, model := range tags.Models {
		installed[model.Name] = true
	}
	return installed, nil
}

func (a *Adapter) pull(ctx context.Context, model string) error {
	payload, err := json.Marshal(pullRequest{Model: model, Stream: false})
	if err != nil {
		return fmt.Errorf("ollama: marshal pull request: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, a.url+"/api/pull", bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("ollama: build pull request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	// Pulls can last minutes; inference timeout must not abort an otherwise
	// healthy download. Shutdown still cancels through ctx.
	resp, err := (&http.Client{}).Do(req)
	if err != nil {
		return fmt.Errorf("ollama: pull %q: %w", model, err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("ollama: pull %q status %d", model, resp.StatusCode)
	}
	return nil
}

// Complete sends a prompt to the model and returns its text completion.
// The model is asked for JSON output via format:json.
func (a *Adapter) Complete(ctx context.Context, prompt string) (string, error) {
	body := generateRequest{Model: a.llmModel, Prompt: prompt, Stream: false, Format: "json"}
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
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("ollama: generate status %d", resp.StatusCode)
	}

	var gr generateResponse
	if err := json.NewDecoder(resp.Body).Decode(&gr); err != nil {
		return "", fmt.Errorf("ollama: decode generate: %w", err)
	}
	return gr.Response, nil
}
