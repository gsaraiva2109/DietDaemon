// Package labelextract holds the prompt and response contract shared by every
// vision adapter's ExtractLabel implementation (anthropic, openai, ollama), so
// the JSON schema the model is asked for lives in exactly one place.
package labelextract

import (
	"encoding/json"
	"fmt"

	"github.com/gsaraiva2109/dietdaemon/adapters/model/internal/jsonfence"
	"github.com/gsaraiva2109/dietdaemon/core/types"
)

// Prompt instructs the model to read a nutrition label photo — in any
// language — and return the JSON contract ParseResponse expects. It is
// deliberately language-agnostic: a vision-capable model reads English or
// Portuguese labels natively, so no locale branching is needed here.
const Prompt = `You are reading a photo of a food nutrition facts label. The label may be in any language (e.g. English, Portuguese).

Extract these values if, and only if, they are legibly printed on the label:
- name: the food/product name
- basis_grams: the serving size the macros below are stated per, in grams (e.g. "per 100g" -> 100)
- calories: energy in kcal for that serving basis
- protein_g, carbs_g, fat_g, fiber_g: grams for that serving basis

Rules:
- NEVER invent, guess, or estimate a value. If a field is not legibly present on the label, its value must be JSON null.
- If a value is printed but you are not fully confident you read it correctly (blur, glare, partial occlusion), still report your best-effort reading, but list its key in low_confidence_fields.
- If the image contains no readable nutrition label at all, set unreadable to true and set every other field to null.
- Convert units to grams/kcal when the label uses a different unit (e.g. kJ -> kcal, mg -> g).

Respond with ONLY this JSON object, no markdown fences, no commentary:
{
  "name": string or null,
  "basis_grams": number or null,
  "calories": number or null,
  "protein_g": number or null,
  "carbs_g": number or null,
  "fat_g": number or null,
  "fiber_g": number or null,
  "low_confidence_fields": array of field name strings (may be empty),
  "unreadable": boolean
}`

type wireResponse struct {
	Name                *string  `json:"name"`
	BasisGrams          *float64 `json:"basis_grams"`
	Calories            *float64 `json:"calories"`
	ProteinG            *float64 `json:"protein_g"`
	CarbsG              *float64 `json:"carbs_g"`
	FatG                *float64 `json:"fat_g"`
	FiberG              *float64 `json:"fiber_g"`
	LowConfidenceFields []string `json:"low_confidence_fields"`
	Unreadable          bool     `json:"unreadable"`
}

// ParseResponse parses a model's raw text response (optionally markdown-fenced)
// into a NutritionLabelDraft.
func ParseResponse(raw string) (types.NutritionLabelDraft, error) {
	stripped := jsonfence.Strip(raw)

	var wr wireResponse
	if err := json.Unmarshal([]byte(stripped), &wr); err != nil {
		return types.NutritionLabelDraft{}, fmt.Errorf("labelextract: decode response: %w", err)
	}

	return types.NutritionLabelDraft{
		Name:                wr.Name,
		BasisGrams:          wr.BasisGrams,
		Calories:            wr.Calories,
		ProteinG:            wr.ProteinG,
		CarbsG:              wr.CarbsG,
		FatG:                wr.FatG,
		FiberG:              wr.FiberG,
		LowConfidenceFields: wr.LowConfidenceFields,
		Unreadable:          wr.Unreadable,
	}, nil
}
