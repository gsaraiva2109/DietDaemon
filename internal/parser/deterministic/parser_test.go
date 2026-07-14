package deterministic

import (
	"context"
	"math"
	"testing"

	"github.com/gsaraiva2109/dietdaemon/core/types"
)

func eq(a, b float64) bool { return math.Abs(a-b) < 1e-6 }

func TestTier(t *testing.T) {
	if New().Tier() != types.TierDeterministic {
		t.Fatalf("Tier() != 0")
	}
}

func TestParsePortugueseAndEnglishMatch(t *testing.T) {
	p := New()
	pt, cpt, _ := p.Extract(context.Background(), "200g frango, 2 ovos", "pt-BR")
	en, cen, _ := p.Extract(context.Background(), "200g chicken, 2 eggs", "en-US")

	if len(pt) != 2 || len(en) != 2 {
		t.Fatalf("want 2 items each, got pt=%d en=%d", len(pt), len(en))
	}
	// Structure must be identical across languages (food phrase aside).
	for i := range pt {
		if !eq(pt[i].Quantity, en[i].Quantity) || !eq(pt[i].NormalizedGrams, en[i].NormalizedGrams) {
			t.Errorf("item %d differs: pt=%+v en=%+v", i, pt[i], en[i])
		}
	}
	if !eq(pt[0].NormalizedGrams, 200) {
		t.Errorf("frango grams = %v, want 200", pt[0].NormalizedGrams)
	}
	if pt[0].RawPhrase != "frango" || en[0].RawPhrase != "chicken" {
		t.Errorf("food phrase: pt=%q en=%q", pt[0].RawPhrase, en[0].RawPhrase)
	}
	// "2 ovos" / "2 eggs": count-based, grams unknown.
	if !eq(pt[1].Quantity, 2) || !eq(pt[1].NormalizedGrams, 0) {
		t.Errorf("eggs item = %+v, want qty 2 grams 0", pt[1])
	}
	if cpt < 0.8 || cen < 0.8 {
		t.Errorf("confidence too low: pt=%v en=%v", cpt, cen)
	}
}

func TestVolumeAndCookingUnits(t *testing.T) {
	p := New()

	items, _, _ := p.Extract(context.Background(), "1,5 xicara de arroz", "pt-BR")
	if len(items) != 1 || !eq(items[0].NormalizedGrams, 360) || items[0].RawPhrase != "arroz" {
		t.Fatalf("xicara case = %+v, want 360g arroz", items)
	}

	items, _, _ = p.Extract(context.Background(), "2 colheres de sopa de azeite", "pt-BR")
	if len(items) != 1 || !eq(items[0].NormalizedGrams, 30) || items[0].RawPhrase != "azeite" {
		t.Fatalf("colher de sopa case = %+v, want 30g azeite", items)
	}

	items, _, _ = p.Extract(context.Background(), "250ml milk", "en-US")
	if len(items) != 1 || !eq(items[0].NormalizedGrams, 250) || items[0].RawPhrase != "milk" {
		t.Fatalf("ml case = %+v, want 250g milk", items)
	}
}

func TestQuantitylessAndEmpty(t *testing.T) {
	p := New()

	items, conf, _ := p.Extract(context.Background(), "cafe", "pt-BR")
	if len(items) != 1 || items[0].RawPhrase != "cafe" {
		t.Fatalf("bare food = %+v", items)
	}
	if conf >= 0.8 {
		t.Errorf("quantity-less confidence should be low, got %v", conf)
	}

	items, conf, _ = p.Extract(context.Background(), "   ", "pt-BR")
	if len(items) != 0 || conf != 0 {
		t.Errorf("empty input = %+v conf %v, want none", items, conf)
	}
}

// TestLeadingVerbStripped reproduces the real bug: "Comi arroz, feijão, bife
// e salada" left "comi" attached to the first item's RawPhrase, polluting
// the embedding query for exactly the first food in the extremely common "I
// ate X, Y and Z" phrasing.
func TestLeadingVerbStripped(t *testing.T) {
	p := New()

	items, _, _ := p.Extract(context.Background(), "Comi arroz, feijao, bife e salada", "pt-BR")
	if len(items) != 4 {
		t.Fatalf("want 4 items, got %d: %+v", len(items), items)
	}
	want := []string{"arroz", "feijao", "bife", "salada"}
	for i, w := range want {
		if items[i].RawPhrase != w {
			t.Errorf("item %d RawPhrase = %q, want %q", i, items[i].RawPhrase, w)
		}
	}

	items, _, _ = p.Extract(context.Background(), "I ate rice, beans, steak and salad", "en-US")
	if len(items) != 4 {
		t.Fatalf("want 4 items, got %d: %+v", len(items), items)
	}
	wantEN := []string{"rice", "beans", "steak", "salad"}
	for i, w := range wantEN {
		if items[i].RawPhrase != w {
			t.Errorf("item %d RawPhrase = %q, want %q", i, items[i].RawPhrase, w)
		}
	}
}

// TestLeadingVerbStrippedOtherPhrasings covers the remaining filler phrases
// (PT/EN) and the mass-unit path ("comi 200g de arroz" strips before the
// quantity is even parsed since it starts with the verb, matching the
// no-quantity fallback path deterministically).
func TestLeadingVerbStrippedOtherPhrasings(t *testing.T) {
	p := New()

	tests := []struct {
		text string
		want string
	}{
		{"comendo arroz", "arroz"},
		{"vou comer arroz", "arroz"},
		{"acabei de comer arroz", "arroz"},
		{"I had rice", "rice"},
		{"eating rice", "rice"},
	}
	for _, tt := range tests {
		items, _, _ := p.Extract(context.Background(), tt.text, "pt-BR")
		if len(items) != 1 || items[0].RawPhrase != tt.want {
			t.Errorf("Extract(%q) = %+v, want RawPhrase %q", tt.text, items, tt.want)
		}
	}
}

// TestNoLeadingVerbUnaffected is the control: a segment with no leading
// filler word must be left exactly as before, and an unusual ordering that
// happens to end with a filler word (not lead with it) must not be mangled
// since stripLeadingFiller only checks the start of the phrase.
func TestNoLeadingVerbUnaffected(t *testing.T) {
	p := New()

	items, _, _ := p.Extract(context.Background(), "200g frango, 2 ovos", "pt-BR")
	if len(items) != 2 || items[0].RawPhrase != "frango" || items[1].RawPhrase != "ovos" {
		t.Fatalf("unaffected case = %+v", items)
	}

	items, _, _ = p.Extract(context.Background(), "arroz comi", "pt-BR")
	if len(items) != 1 || items[0].RawPhrase != "arroz comi" {
		t.Errorf("trailing verb should be left alone, got %+v", items)
	}
}

func TestConjunctionSeparators(t *testing.T) {
	p := New()
	items, _, _ := p.Extract(context.Background(), "100g arroz e 150g feijao", "pt-BR")
	if len(items) != 2 || !eq(items[0].NormalizedGrams, 100) || !eq(items[1].NormalizedGrams, 150) {
		t.Fatalf("conjunction split = %+v, want arroz 100 + feijao 150", items)
	}
}
