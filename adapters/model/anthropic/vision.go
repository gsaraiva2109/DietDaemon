package anthropic

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/gsaraiva2109/dietdaemon/adapters/model/internal/labelextract"
	"github.com/gsaraiva2109/dietdaemon/core/ports"
	"github.com/gsaraiva2109/dietdaemon/core/types"
)

// Compile-time interface check.
var _ ports.VisionAdapter = (*Adapter)(nil)

// ---------------------------------------------------------------------------
// ExtractLabel — POST /v1/messages with an image content block
// ---------------------------------------------------------------------------

type imageSource struct {
	Type      string `json:"type"`
	MediaType string `json:"media_type"`
	Data      string `json:"data"`
}

type visionContentBlock struct {
	Type   string       `json:"type"`
	Source *imageSource `json:"source,omitempty"`
	Text   string       `json:"text,omitempty"`
}

type visionMessage struct {
	Role    string               `json:"role"`
	Content []visionContentBlock `json:"content"`
}

type visionRequest struct {
	Model     string          `json:"model"`
	MaxTokens int             `json:"max_tokens"`
	Messages  []visionMessage `json:"messages"`
}

// ExtractLabel sends the photographed nutrition label to Anthropic's Messages
// API as an image content block and parses the model's JSON reply.
func (a *Adapter) ExtractLabel(ctx context.Context, image []byte, mimeType string) (types.NutritionLabelDraft, error) {
	body := visionRequest{
		Model:     a.model,
		MaxTokens: 1024,
		Messages: []visionMessage{
			{
				Role: "user",
				Content: []visionContentBlock{
					{Type: "image", Source: &imageSource{
						Type:      "base64",
						MediaType: mimeType,
						Data:      base64.StdEncoding.EncodeToString(image),
					}},
					{Type: "text", Text: labelextract.Prompt},
				},
			},
		},
	}
	payload, err := json.Marshal(body)
	if err != nil {
		return types.NutritionLabelDraft{}, fmt.Errorf("anthropic: marshal vision request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		a.baseURL+"/v1/messages", bytes.NewReader(payload))
	if err != nil {
		return types.NutritionLabelDraft{}, fmt.Errorf("anthropic: build vision request: %w", err)
	}
	req.Header.Set("x-api-key", a.apiKey)
	req.Header.Set("anthropic-version", "2023-06-01")
	req.Header.Set("content-type", "application/json")

	resp, err := a.client.Do(req)
	if err != nil {
		return types.NutritionLabelDraft{}, fmt.Errorf("anthropic: vision messages: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		detail, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
		return types.NutritionLabelDraft{}, fmt.Errorf("anthropic: vision messages status %d: %s", resp.StatusCode, bytes.TrimSpace(detail))
	}

	var mr messagesResponse
	if err := json.NewDecoder(resp.Body).Decode(&mr); err != nil {
		return types.NutritionLabelDraft{}, fmt.Errorf("anthropic: decode vision messages: %w", err)
	}
	if len(mr.Content) == 0 {
		return types.NutritionLabelDraft{}, fmt.Errorf("anthropic: empty content in vision response")
	}

	return labelextract.ParseResponse(mr.Content[0].Text)
}
