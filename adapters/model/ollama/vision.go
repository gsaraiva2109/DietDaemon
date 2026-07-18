package ollama

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/gsaraiva2109/dietdaemon/adapters/model/internal/labelextract"
	"github.com/gsaraiva2109/dietdaemon/core/ports"
	"github.com/gsaraiva2109/dietdaemon/core/types"
)

// Compile-time interface check.
var _ ports.VisionAdapter = (*Adapter)(nil)

// ---------------------------------------------------------------------------
// ExtractLabel — POST /api/generate with an embedded image
// ---------------------------------------------------------------------------

type visionRequest struct {
	Model  string   `json:"model"`
	Prompt string   `json:"prompt"`
	Images []string `json:"images"`
	Stream bool     `json:"stream"`
	Format string   `json:"format"`
}

// ExtractLabel sends the photographed nutrition label to Ollama's
// /api/generate endpoint using visionModel (set via SetVisionModel) and
// parses the model's JSON reply. mimeType is unused: Ollama's images array
// takes bare base64 with no data-URI prefix.
func (a *Adapter) ExtractLabel(ctx context.Context, image []byte, mimeType string) (types.NutritionLabelDraft, error) {
	body := visionRequest{
		Model:  a.visionModel,
		Prompt: labelextract.Prompt,
		Images: []string{base64.StdEncoding.EncodeToString(image)},
		Stream: false,
		Format: "json",
	}
	payload, err := json.Marshal(body)
	if err != nil {
		return types.NutritionLabelDraft{}, fmt.Errorf("ollama: marshal vision request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		a.url+"/api/generate", bytes.NewReader(payload))
	if err != nil {
		return types.NutritionLabelDraft{}, fmt.Errorf("ollama: build vision request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := a.client.Do(req)
	if err != nil {
		return types.NutritionLabelDraft{}, fmt.Errorf("ollama: vision generate: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return types.NutritionLabelDraft{}, fmt.Errorf("ollama: vision generate status %d", resp.StatusCode)
	}

	var gr generateResponse
	if err := json.NewDecoder(resp.Body).Decode(&gr); err != nil {
		return types.NutritionLabelDraft{}, fmt.Errorf("ollama: decode vision generate: %w", err)
	}

	return labelextract.ParseResponse(gr.Response)
}
