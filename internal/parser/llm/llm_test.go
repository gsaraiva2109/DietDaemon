package llm

import (
	"context"
	"errors"
	"testing"

	"github.com/gsaraiva2109/dietdaemon/core/types"
)

// --- Fakes ---

// stubModel returns canned JSON for Complete and a canned vector for Embed.
type stubModel struct {
	complete string
	compErr  error
}

func (m *stubModel) Embed(_ context.Context, _ string) ([]float32, error) {
	return []float32{0.1, 0.2}, nil
}
func (m *stubModel) Complete(_ context.Context, _ string) (string, error) {
	return m.complete, m.compErr
}

// stubParser is a fake Tier-0 parser for fallback.
type stubParser struct {
	items []types.ParsedItem
	conf  float64
	err   error
}

func (p *stubParser) Extract(_ context.Context, _, _ string) ([]types.ParsedItem, float64, error) {
	return p.items, p.conf, p.err
}
func (p *stubParser) Tier() types.ParserTier { return types.TierDeterministic }

// ---------------------------------------------------------------------------
// Mechanical tests
// ---------------------------------------------------------------------------

func TestExtractValidJSON(t *testing.T) {
	model := &stubModel{
		complete: `{"items":[{"food":"chicken breast","quantity":200,"unit":"g"}]}`,
	}
	fallback := &stubParser{}
	p := New(model, fallback)

	items, conf, err := p.Extract(t.Context(), "200g chicken breast", "en")
	if err != nil {
		t.Fatalf("Extract: %v", err)
	}
	if conf != 0.90 {
		t.Errorf("confidence = %v, want 0.90", conf)
	}
	if len(items) != 1 {
		t.Fatalf("got %d items, want 1", len(items))
	}
	if items[0].RawPhrase != "chicken breast" {
		t.Errorf("RawPhrase = %q", items[0].RawPhrase)
	}
	if items[0].Quantity != 200 {
		t.Errorf("Quantity = %v, want 200", items[0].Quantity)
	}
	if items[0].Unit != "g" {
		t.Errorf("Unit = %q, want g", items[0].Unit)
	}
	if items[0].NormalizedGrams != 200 {
		t.Errorf("NormalizedGrams = %v, want 200", items[0].NormalizedGrams)
	}
}

func TestExtractMultipleItems(t *testing.T) {
	model := &stubModel{
		complete: `{"items":[
			{"food":"eggs","quantity":2,"unit":""},
			{"food":"rice","quantity":100,"unit":"g"}
		]}`,
	}
	fallback := &stubParser{}
	p := New(model, fallback)

	items, conf, err := p.Extract(t.Context(), "2 eggs and 100g rice", "en")
	if err != nil {
		t.Fatalf("Extract: %v", err)
	}
	if conf != 0.90 {
		t.Errorf("confidence = %v, want 0.90", conf)
	}
	if len(items) != 2 {
		t.Fatalf("got %d items, want 2", len(items))
	}
	// Eggs: count-based, grams=0.
	if items[0].RawPhrase != "eggs" || items[0].Quantity != 2 || items[0].NormalizedGrams != 0 {
		t.Errorf("eggs item = %+v, want qty=2 grams=0", items[0])
	}
	// Rice: mass.
	if items[1].RawPhrase != "rice" || items[1].Quantity != 100 || items[1].NormalizedGrams != 100 {
		t.Errorf("rice item = %+v, want qty=100 grams=100", items[1])
	}
}

func TestExtractUnitNormalization(t *testing.T) {
	model := &stubModel{
		complete: `{"items":[{"food":"rice","quantity":1,"unit":"kg"}]}`,
	}
	fallback := &stubParser{}
	p := New(model, fallback)

	items, _, err := p.Extract(t.Context(), "1kg rice", "en")
	if err != nil {
		t.Fatalf("Extract: %v", err)
	}
	if items[0].NormalizedGrams != 1000 {
		t.Errorf("NormalizedGrams = %v, want 1000 (1 kg)", items[0].NormalizedGrams)
	}
}

func TestExtractFallbackOnModelError(t *testing.T) {
	model := &stubModel{compErr: errors.New("ollama down")}
	fallback := &stubParser{
		items: []types.ParsedItem{{RawPhrase: "chicken", Quantity: 200, Unit: "g", NormalizedGrams: 200}},
		conf:  0.85,
	}
	p := New(model, fallback)

	items, conf, err := p.Extract(t.Context(), "200g chicken", "en")
	if err != nil {
		t.Fatalf("Extract: %v", err)
	}
	// Fallback should return its own items + confidence.
	if conf != 0.85 {
		t.Errorf("fallback confidence = %v, want 0.85", conf)
	}
	if len(items) != 1 || items[0].RawPhrase != "chicken" {
		t.Errorf("items = %+v, want fallback result", items)
	}
}

func TestExtractFallbackOnBadJSON(t *testing.T) {
	model := &stubModel{complete: `not json at all`}
	fallback := &stubParser{
		items: []types.ParsedItem{{RawPhrase: "chicken", Quantity: 200, Unit: "g", NormalizedGrams: 200}},
		conf:  0.80,
	}
	p := New(model, fallback)

	items, conf, err := p.Extract(t.Context(), "200g chicken", "en")
	if err != nil {
		t.Fatalf("Extract: %v", err)
	}
	if conf != 0.80 {
		t.Errorf("fallback confidence = %v, want 0.80", conf)
	}
	if len(items) == 0 {
		t.Error("expected fallback items, got empty")
	}
}

func TestExtractFallbackOnEmptyItems(t *testing.T) {
	model := &stubModel{complete: `{"items":[]}`}
	fallback := &stubParser{
		items: []types.ParsedItem{{RawPhrase: "chicken", Quantity: 200, Unit: "g", NormalizedGrams: 200}},
		conf:  0.75,
	}
	p := New(model, fallback)

	items, conf, err := p.Extract(t.Context(), "200g chicken", "en")
	if err != nil {
		t.Fatalf("Extract: %v", err)
	}
	if conf != 0.75 {
		t.Errorf("fallback confidence = %v, want 0.75", conf)
	}
	if len(items) == 0 {
		t.Error("expected fallback items, got empty")
	}
}

func TestTierIsLLM(t *testing.T) {
	p := New(&stubModel{}, &stubParser{})
	if p.Tier() != types.TierLLM {
		t.Errorf("Tier() = %d, want %d (TierLLM)", p.Tier(), types.TierLLM)
	}
}

func TestExtractEmptyFoodSkipped(t *testing.T) {
	model := &stubModel{
		complete: `{"items":[
			{"food":"","quantity":100,"unit":"g"},
			{"food":"chicken","quantity":200,"unit":"g"}
		]}`,
	}
	fallback := &stubParser{}
	p := New(model, fallback)

	items, _, err := p.Extract(t.Context(), "test", "en")
	if err != nil {
		t.Fatalf("Extract: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("got %d items, want 1 (empty food skipped)", len(items))
	}
	if items[0].RawPhrase != "chicken" {
		t.Errorf("RawPhrase = %q, want chicken", items[0].RawPhrase)
	}
}

// ---------------------------------------------------------------------------
// Opus judgment test — messy prose the deterministic grammar cannot segment.
//
// The other two judgment scenarios scaffolded here (cross-language embedding,
// profile-off regression) were filled where their fakes live, not in this
// parser package: TestTier2CrossLanguageEmbedding in internal/resolver/embedding
// and TestProfileOffRegression in internal/resolver. Keeping each test next to
// the component it exercises avoids duplicating index/store fakes in llm_test.
// ---------------------------------------------------------------------------

// TestTier2MessyProse is the core Tier-2 win: prose the deterministic parser
// cannot segment ("had a couple eggs and some rice this morning"). The model
// untangles it into two items, one with a known count (2 eggs) and one with a
// vague portion (rice). The vague portion MUST stay quantity 0 / grams 0 so the
// resolver flags it portion-unknown and clarification asks the user —
// the parser never guesses a portion.
func TestTier2MessyProse(t *testing.T) {
	model := &stubModel{
		complete: `{"items":[
			{"food":"eggs","quantity":2,"unit":""},
			{"food":"rice","quantity":0,"unit":""}
		]}`,
	}
	p := New(model, &stubParser{})

	items, conf, err := p.Extract(t.Context(), "had a couple eggs and some rice this morning", "en")
	if err != nil {
		t.Fatalf("Extract: %v", err)
	}
	if conf != 0.90 {
		t.Errorf("confidence = %v, want 0.90", conf)
	}
	if len(items) != 2 {
		t.Fatalf("got %d items, want 2", len(items))
	}

	// Eggs: known count, portion still unknown (count-based → grams 0).
	if items[0].RawPhrase != "eggs" || items[0].Quantity != 2 || items[0].NormalizedGrams != 0 {
		t.Errorf("eggs = %+v, want phrase=eggs qty=2 grams=0", items[0])
	}
	// Rice: vague portion. quantity MUST remain 0 (not defaulted to 1) so the
	// clarification loop owns it rather than the parser silently guessing.
	if items[1].RawPhrase != "rice" || items[1].Quantity != 0 || items[1].NormalizedGrams != 0 {
		t.Errorf("rice = %+v, want phrase=rice qty=0 grams=0 (vague portion preserved)", items[1])
	}
}
