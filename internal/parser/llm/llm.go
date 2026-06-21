// Package llm implements the Tier-2 parser: an LLM-powered extractor that
// handles messy prose the deterministic grammar cannot segment. It only splits
// — it emits the same ParsedItems the deterministic parser does, then the
// shared unit normalizer turns units into grams. No macros, no foods invented.
//
// On any model error or empty result the parser falls back to an injected
// Tier-0 parser so a flaky model never silently drops a meal.
package llm

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/gsaraiva2109/dietdaemon/core/ports"
	"github.com/gsaraiva2109/dietdaemon/core/types"
	"github.com/gsaraiva2109/dietdaemon/internal/parser/normalize"
)

// Compile-time interface check.
var _ ports.Parser = (*Parser)(nil)

// promptTemplate is the system prompt template. {text} is replaced with the
// user's meal description.
const promptTemplate = `You convert a meal description into a list of food items. Output ONLY JSON:
{"items":[{"food":"<short food name>","quantity":<number>,"unit":"<unit or '' for count>"}]}
Rules:
- One object per distinct food. Split compound descriptions.
- quantity is the number eaten; unit is the measure (g, ml, cup, colher, tbsp...) or "" when it's a count ("2 eggs" -> quantity 2, unit "").
- Do NOT estimate calories, macros, or weights. Only food, quantity, unit.
- Preserve the food language as written (keep "frango" as "frango").
- If a quantity is vague ("some", "a bit"), set quantity to 0 and unit "".
Description: "%s"`

// llmResponse is the expected JSON shape from the model.
type llmResponse struct {
	Items []llmItem `json:"items"`
}

type llmItem struct {
	Food     string  `json:"food"`
	Quantity float64 `json:"quantity"`
	Unit     string  `json:"unit"`
}

// Parser is the Tier-2 LLM parser.
type Parser struct {
	model    ports.ModelAdapter
	fallback ports.Parser
}

// New returns a ready Tier-2 parser. model is used for extraction; fallback is
// invoked when the model returns an error or empty items.
func New(model ports.ModelAdapter, fallback ports.Parser) *Parser {
	return &Parser{model: model, fallback: fallback}
}

// Tier reports that this is the LLM strategy.
func (p *Parser) Tier() types.ParserTier { return types.TierLLM }

// Extract sends the user's text to the LLM and parses the JSON response. On
// any failure (transport error, bad JSON, empty items) it falls back to the
// Tier-0 parser so a flaky model never drops a meal.
func (p *Parser) Extract(ctx context.Context, text, locale string) ([]types.ParsedItem, float64, error) {
	prompt := fmt.Sprintf(promptTemplate, text)
	raw, err := p.model.Complete(ctx, prompt)
	if err != nil {
		// Model unavailable: fall back to deterministic.
		return p.fallback.Extract(ctx, text, locale)
	}

	var resp llmResponse
	if err := json.Unmarshal([]byte(raw), &resp); err != nil {
		// Bad JSON: fall back.
		return p.fallback.Extract(ctx, text, locale)
	}

	if len(resp.Items) == 0 {
		// Model returned valid JSON but no items: fall back.
		return p.fallback.Extract(ctx, text, locale)
	}

	items := make([]types.ParsedItem, 0, len(resp.Items))
	for _, it := range resp.Items {
		food := it.Food
		if food == "" {
			continue
		}
		// quantity==0 is the model's signal for a vague portion ("some rice").
		// Keep it zero: NormalizeUnit yields grams=0, so the resolver flags the
		// item as portion-unknown and the clarification loop owns it.
		// Never invent a default portion here — that would silently guess.
		qty := it.Quantity
		canonicalUnit, grams := normalize.NormalizeUnit(qty, it.Unit, food, locale)

		items = append(items, types.ParsedItem{
			RawPhrase:       food,
			Quantity:        qty,
			Unit:            canonicalUnit,
			NormalizedGrams: grams,
			Locale:          locale,
		})
	}

	if len(items) == 0 {
		return p.fallback.Extract(ctx, text, locale)
	}

	// Tier-2 confidence: 0.90 on valid JSON with ≥1 item.
	return items, 0.90, nil
}
