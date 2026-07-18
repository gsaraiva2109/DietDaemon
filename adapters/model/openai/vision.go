package openai

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
// ExtractLabel — POST {baseURL}/chat/completions with an image_url content part
// ---------------------------------------------------------------------------

type imageURL struct {
	URL string `json:"url"`
}

type visionContentPart struct {
	Type     string    `json:"type"`
	Text     string    `json:"text,omitempty"`
	ImageURL *imageURL `json:"image_url,omitempty"`
}

type visionMessage struct {
	Role    string              `json:"role"`
	Content []visionContentPart `json:"content"`
}

type visionRequest struct {
	Model          string          `json:"model"`
	Messages       []visionMessage `json:"messages"`
	ResponseFormat *responseFormat `json:"response_format"`
}

// ExtractLabel sends the photographed nutrition label as a data-URI image_url
// content part to an OpenAI-compatible chat/completions endpoint and parses
// the model's JSON reply.
func (a *Adapter) ExtractLabel(ctx context.Context, image []byte, mimeType string) (types.NutritionLabelDraft, error) {
	dataURI := fmt.Sprintf("data:%s;base64,%s", mimeType, base64.StdEncoding.EncodeToString(image))

	body := visionRequest{
		Model: a.model,
		Messages: []visionMessage{
			{
				Role: "user",
				Content: []visionContentPart{
					{Type: "text", Text: labelextract.Prompt},
					{Type: "image_url", ImageURL: &imageURL{URL: dataURI}},
				},
			},
		},
		ResponseFormat: &responseFormat{Type: "json_object"},
	}
	payload, err := json.Marshal(body)
	if err != nil {
		return types.NutritionLabelDraft{}, fmt.Errorf("openai: marshal vision request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		a.baseURL+"/chat/completions", bytes.NewReader(payload))
	if err != nil {
		return types.NutritionLabelDraft{}, fmt.Errorf("openai: build vision request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if a.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+a.apiKey)
	}

	resp, err := a.client.Do(req)
	if err != nil {
		return types.NutritionLabelDraft{}, fmt.Errorf("openai: vision chat completions: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		detail, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
		return types.NutritionLabelDraft{}, fmt.Errorf("openai: vision chat completions status %d: %s", resp.StatusCode, bytes.TrimSpace(detail))
	}

	var cr chatResponse
	if err := json.NewDecoder(resp.Body).Decode(&cr); err != nil {
		return types.NutritionLabelDraft{}, fmt.Errorf("openai: decode vision chat completions: %w", err)
	}
	if len(cr.Choices) == 0 {
		return types.NutritionLabelDraft{}, fmt.Errorf("openai: vision chat completions returned no choices")
	}

	return labelextract.ParseResponse(cr.Choices[0].Message.Content)
}
