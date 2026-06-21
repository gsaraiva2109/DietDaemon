package resolver

import (
	"context"
	"testing"

	"github.com/gsaraiva2109/dietdaemon/core/types"
)

// fakeStore is an in-memory FoodStore that tracks calls for assertions.
type fakeStore struct {
	lib       map[string]types.FoodMatch // phrase -> match
	upserts   []types.FoodMatch
	aliases   [][]string
	recorded  []string // foodIDs passed to RecordFoodQuery
	upsertErr error
}

func (f *fakeStore) LookupFood(_ context.Context, _, phrase string) (types.FoodMatch, error) {
	if m, ok := f.lib[phrase]; ok {
		return m, nil
	}
	return types.FoodMatch{}, types.ErrNoMatch
}

func (f *fakeStore) GetFood(_ context.Context, _, foodID string) (types.FoodMatch, error) {
	for _, m := range f.lib {
		if m.FoodID == foodID {
			return m, nil
		}
	}
	return types.FoodMatch{}, types.ErrNoMatch
}

func (f *fakeStore) UpsertFood(_ context.Context, _ string, match types.FoodMatch, aliases []string) error {
	f.upserts = append(f.upserts, match)
	f.aliases = append(f.aliases, aliases)
	return f.upsertErr
}

func (f *fakeStore) RecordFoodQuery(_ context.Context, _, foodID string) error {
	f.recorded = append(f.recorded, foodID)
	return nil
}

// fakeSource matches a single phrase, otherwise ErrNoMatch.
type fakeSource struct {
	name  string
	phr   string
	match types.FoodMatch
	calls int
}

func (s *fakeSource) Name() string { return s.name }
func (s *fakeSource) Resolve(_ context.Context, item types.ParsedItem) (types.FoodMatch, error) {
	s.calls++
	if item.RawPhrase == s.phr {
		return s.match, nil
	}
	return types.FoodMatch{}, types.ErrNoMatch
}

// fakeMatcher implements Matcher for tests.
type fakeMatcher struct {
	match map[string]types.FoodMatch // phrase -> match
	err   error
	calls int
}

func (m *fakeMatcher) Match(_ context.Context, _, phrase string) (types.FoodMatch, error) {
	m.calls++
	if m.err != nil {
		return types.FoodMatch{}, m.err
	}
	if fm, ok := m.match[phrase]; ok {
		return fm, nil
	}
	return types.FoodMatch{}, types.ErrNoMatch
}

// fakeEmbedder implements Embedder for tests.
type fakeEmbedder struct {
	embeds []struct{ userID, foodID, name string }
}

func (e *fakeEmbedder) EmbedFood(_ context.Context, userID, foodID, name string) error {
	e.embeds = append(e.embeds, struct{ userID, foodID, name string }{userID, foodID, name})
	return nil
}

func chicken() types.FoodMatch {
	return types.FoodMatch{FoodID: "off:1", Name: "chicken breast", Source: "openfoodfacts",
		Per100g: types.Macros{Calories: 165, Protein: 31, Carbs: 0, Fat: 3.6}}
}

func TestLocalFirstHit(t *testing.T) {
	st := &fakeStore{lib: map[string]types.FoodMatch{"frango": chicken()}}
	src := &fakeSource{name: "off", phr: "frango", match: chicken()}
	r := New(st, nil, nil, 0.92, src)

	items := []types.ParsedItem{{RawPhrase: "frango", NormalizedGrams: 200}}
	res, need := r.Resolve(context.Background(), "u1", items)

	if need != 0 || len(res) != 1 {
		t.Fatalf("need=%d res=%d, want 0/1", need, len(res))
	}
	if src.calls != 0 {
		t.Errorf("external source called %d times on a local hit, want 0", src.calls)
	}
	if len(st.recorded) != 1 || st.recorded[0] != "off:1" {
		t.Errorf("RecordFoodQuery = %v, want [off:1]", st.recorded)
	}
	if got := res[0].Macros.Protein; got != 62 { // 31 * 200/100
		t.Errorf("protein = %v, want 62", got)
	}
}

func TestExternalMissThenWriteBack(t *testing.T) {
	st := &fakeStore{lib: map[string]types.FoodMatch{}}
	miss := &fakeSource{name: "taco", phr: "nope"}
	hit := &fakeSource{name: "off", phr: "frango", match: chicken()}
	r := New(st, nil, nil, 0.92, miss, hit) // order matters: miss first

	items := []types.ParsedItem{{RawPhrase: "frango", NormalizedGrams: 100}}
	res, need := r.Resolve(context.Background(), "u1", items)

	if need != 0 {
		t.Fatalf("need=%d, want 0", need)
	}
	if miss.calls != 1 || hit.calls != 1 {
		t.Errorf("source calls miss=%d hit=%d, want 1/1", miss.calls, hit.calls)
	}
	if len(st.upserts) != 1 || st.upserts[0].FoodID != "off:1" {
		t.Errorf("write-back upserts = %v, want one off:1", st.upserts)
	}
	if got := res[0].Macros.Calories; got != 165 {
		t.Errorf("calories = %v, want 165", got)
	}
}

func TestUnresolvedAndCountBased(t *testing.T) {
	st := &fakeStore{lib: map[string]types.FoodMatch{"ovo": chicken()}}
	r := New(st, nil, nil, 0.92) // no external sources

	items := []types.ParsedItem{
		{RawPhrase: "ovo", NormalizedGrams: 0},          // matched food, unknown portion
		{RawPhrase: "unknownfood", NormalizedGrams: 50}, // no match anywhere
	}
	res, need := r.Resolve(context.Background(), "u1", items)

	if need != 2 {
		t.Fatalf("need=%d, want 2 (portion-unknown + no-match)", need)
	}
	if res[0].Match.FoodID != "off:1" || res[0].Macros.Calories != 0 {
		t.Errorf("count-based item should match food but have zero macros, got %+v", res[0])
	}
	if res[1].Match.FoodID != "" {
		t.Errorf("no-match item should have empty Match, got %+v", res[1].Match)
	}
}

// --- Matcher tests ---

func TestMatcherHit(t *testing.T) {
	st := &fakeStore{lib: map[string]types.FoodMatch{}}
	m := &fakeMatcher{match: map[string]types.FoodMatch{
		"frango": {FoodID: "off:1", Name: "chicken", Source: "food_library",
			Per100g: types.Macros{Calories: 165, Protein: 31}, MatchScore: 0.85},
	}}
	src := &fakeSource{name: "off", phr: "frango", match: chicken()}
	r := New(st, m, nil, 0.92, src)

	items := []types.ParsedItem{{RawPhrase: "frango", NormalizedGrams: 100}}
	res, need := r.Resolve(context.Background(), "u1", items)
	_ = res

	if need != 0 {
		t.Fatalf("need=%d, want 0", need)
	}
	if m.calls != 1 {
		t.Errorf("matcher called %d times, want 1", m.calls)
	}
	// External source should not be called.
	if src.calls != 0 {
		t.Errorf("external source called %d times after matcher hit, want 0", src.calls)
	}
}

func TestMatcherMissFallsThroughToExternal(t *testing.T) {
	st := &fakeStore{lib: map[string]types.FoodMatch{}}
	m := &fakeMatcher{match: map[string]types.FoodMatch{}} // matches nothing
	src := &fakeSource{name: "off", phr: "frango", match: chicken()}
	r := New(st, m, nil, 0.92, src)

	items := []types.ParsedItem{{RawPhrase: "frango", NormalizedGrams: 100}}
	res, need := r.Resolve(context.Background(), "u1", items)

	if need != 0 {
		t.Fatalf("need=%d, want 0", need)
	}
	if m.calls != 1 {
		t.Errorf("matcher called %d times, want 1", m.calls)
	}
	if src.calls != 1 {
		t.Errorf("external source called %d times, want 1", src.calls)
	}
	if got := res[0].Macros.Calories; got != 165 {
		t.Errorf("calories = %v, want 165", got)
	}
}

func TestMatcherStrongMatchWritesAlias(t *testing.T) {
	st := &fakeStore{lib: map[string]types.FoodMatch{}}
	m := &fakeMatcher{match: map[string]types.FoodMatch{
		"frango": {FoodID: "off:1", Name: "chicken", Source: "food_library",
			Per100g: types.Macros{Calories: 165, Protein: 31}, MatchScore: 0.95},
	}}
	r := New(st, m, nil, 0.92)

	items := []types.ParsedItem{{RawPhrase: "frango", NormalizedGrams: 100}}
	_, need := r.Resolve(context.Background(), "u1", items)

	if need != 0 {
		t.Fatalf("need=%d, want 0", need)
	}
	// Score >= 0.92 → alias write-back.
	if len(st.upserts) != 1 {
		t.Fatalf("expected 1 upsert for alias write-back, got %d", len(st.upserts))
	}
	if len(st.aliases) != 1 || len(st.aliases[0]) != 1 || st.aliases[0][0] != "frango" {
		t.Errorf("aliases = %v, want [[frango]]", st.aliases)
	}
}

func TestMatcherWeakMatchNoAlias(t *testing.T) {
	st := &fakeStore{lib: map[string]types.FoodMatch{}}
	m := &fakeMatcher{match: map[string]types.FoodMatch{
		"frango": {FoodID: "off:1", Name: "chicken", Source: "food_library",
			Per100g: types.Macros{Calories: 165, Protein: 31}, MatchScore: 0.85},
	}}
	r := New(st, m, nil, 0.92)

	items := []types.ParsedItem{{RawPhrase: "frango", NormalizedGrams: 100}}
	_, _ = r.Resolve(context.Background(), "u1", items)

	// Score < 0.92 → no alias write-back.
	if len(st.upserts) != 0 {
		t.Errorf("expected no upsert for weak match, got %d", len(st.upserts))
	}
}

func TestEmbeddingOnWrite(t *testing.T) {
	st := &fakeStore{lib: map[string]types.FoodMatch{}}
	emb := &fakeEmbedder{}
	src := &fakeSource{name: "off", phr: "frango", match: chicken()}
	r := New(st, nil, emb, 0.92, src)

	items := []types.ParsedItem{{RawPhrase: "frango", NormalizedGrams: 100}}
	_, need := r.Resolve(context.Background(), "u1", items)

	if need != 0 {
		t.Fatalf("need=%d, want 0", need)
	}
	if len(emb.embeds) != 1 {
		t.Fatalf("embed called %d times, want 1", len(emb.embeds))
	}
	if emb.embeds[0].foodID != "off:1" || emb.embeds[0].name != "chicken breast" {
		t.Errorf("embed args = %+v, want foodID=off:1 name='chicken breast'", emb.embeds[0])
	}
}

// TestProfileOffRegression is the safety guarantee: with the ai profile
// off (nil matcher, nil embedder — exactly how cmd wires PARSER_TIER=0), no
// We prove the
// bypass by contrast: the same fixture wired WITH a matcher/embedder takes a
// different path (matcher hit, embed-on-write). Tier 0 must ignore both.
func TestProfileOffRegression(t *testing.T) {
	// A matcher/embedder that would visibly change the outcome IF consulted:
	// the matcher maps "frango" to a different food than the external source.
	spyMatch := func() *fakeMatcher {
		return &fakeMatcher{match: map[string]types.FoodMatch{
			"frango": {FoodID: "matcher:wrong", Name: "wrong", Source: "food_library",
				Per100g: types.Macros{Calories: 1}, MatchScore: 0.99},
		}}
	}

	items := []types.ParsedItem{{RawPhrase: "frango", NormalizedGrams: 100}}

	// Tier 0: nil matcher, nil embedder. Resolves via external source only.
	stOff := &fakeStore{lib: map[string]types.FoodMatch{}}
	embOff := &fakeEmbedder{}
	srcOff := &fakeSource{name: "off", phr: "frango", match: chicken()}
	rOff := New(stOff, nil, nil, 0.92, srcOff)
	resOff, needOff := rOff.Resolve(context.Background(), "u1", items)

	if needOff != 0 {
		t.Fatalf("tier-0 need=%d, want 0", needOff)
	}
	if resOff[0].Match.FoodID != "off:1" {
		t.Errorf("tier-0 resolved via %q, want external off:1 (matcher must not run)", resOff[0].Match.FoodID)
	}
	if srcOff.calls != 1 {
		t.Errorf("tier-0 external calls = %d, want 1", srcOff.calls)
	}
	if len(embOff.embeds) != 0 {
		t.Errorf("tier-0 embedder fired %d times, want 0", len(embOff.embeds))
	}

	// Tier 1: same fixture, but matcher + embedder wired. Different path proves
	// the only thing gating the feature is the wiring, not hidden always-on code.
	m := spyMatch()
	stOn := &fakeStore{lib: map[string]types.FoodMatch{}}
	srcOn := &fakeSource{name: "off", phr: "frango", match: chicken()}
	rOn := New(stOn, m, &fakeEmbedder{}, 0.92, srcOn)
	resOn, _ := rOn.Resolve(context.Background(), "u1", items)

	if m.calls != 1 {
		t.Errorf("tier-1 matcher calls = %d, want 1", m.calls)
	}
	if resOn[0].Match.FoodID != "matcher:wrong" {
		t.Errorf("tier-1 resolved via %q, want matcher:wrong (matcher takes precedence over external)", resOn[0].Match.FoodID)
	}
	if srcOn.calls != 0 {
		t.Errorf("tier-1 external calls = %d, want 0 (matcher hit short-circuits)", srcOn.calls)
	}
}

func TestNoEmbedderOnLocalHit(t *testing.T) {
	st := &fakeStore{lib: map[string]types.FoodMatch{"frango": chicken()}}
	emb := &fakeEmbedder{}
	r := New(st, nil, emb, 0.92)

	items := []types.ParsedItem{{RawPhrase: "frango", NormalizedGrams: 200}}
	_, _ = r.Resolve(context.Background(), "u1", items)

	// Embed not called because it was a local hit, not external.
	if len(emb.embeds) != 0 {
		t.Errorf("embed called %d times on local hit, want 0", len(emb.embeds))
	}
}
