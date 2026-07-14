package resolver

import (
	"context"
	"testing"

	"github.com/gsaraiva2109/dietdaemon/core/types"
)

// fakeStore is an in-memory FoodStore + PrecedenceStore that tracks calls for
// assertions. The same fake satisfies both narrow interfaces, mirroring how
// *store.Store does it for real.
type fakeStore struct {
	lib       map[string]types.FoodMatch // phrase -> match
	upserts   []types.FoodMatch
	aliases   [][]string
	recorded  []string // foodIDs passed to RecordFoodQuery
	upsertErr error

	pendingAliases []types.PendingAlias // AddPendingAlias calls

	precedence    []string // GetSourcePrecedence result
	precedenceErr error
}

func (f *fakeStore) LookupFood(_ context.Context, _, phrase string) (types.FoodMatch, error) {
	if m, ok := f.lib[phrase]; ok {
		return m, nil
	}
	return types.FoodMatch{}, types.ErrNoMatch
}

func (f *fakeStore) GetFood(_ context.Context, foodID string) (types.FoodMatch, error) {
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

func (f *fakeStore) AddPendingAlias(_ context.Context, userID, phrase, foodID string, matchScore float64) error {
	f.pendingAliases = append(f.pendingAliases, types.PendingAlias{
		UserID: userID, Phrase: phrase, FoodID: foodID, MatchScore: matchScore,
	})
	return nil
}

func (f *fakeStore) GetSourcePrecedence(_ context.Context, _ string) ([]string, error) {
	if f.precedenceErr != nil {
		return nil, f.precedenceErr
	}
	return f.precedence, nil
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
	return types.FoodMatch{FoodID: "off:1", Name: "frango grelhado", Source: "openfoodfacts",
		Per100g: types.Macros{Calories: 165, Protein: 31, Carbs: 0, Fat: 3.6}}
}

func TestLocalFirstHit(t *testing.T) {
	st := &fakeStore{lib: map[string]types.FoodMatch{"frango": chicken()}}
	src := &fakeSource{name: "off", phr: "frango", match: chicken()}
	r := New(st, nil, nil, 0.92, st, src)

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
	r := New(st, nil, nil, 0.92, st, miss, hit) // order matters: miss first

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

// TestExternalIrrelevantMatchIsRejected reproduces a real production bug: a
// query for "white rice" hit OpenFoodFacts' loose free-text search and got
// back "Tortitas de arroz con chocolate blanco" (a Spanish snack product with
// no relation to rice beyond OFF's own search returning it first) — the
// resolver accepted it unconditionally and permanently aliased it. The first
// source's irrelevant result must be rejected and fall through to the next
// source instead of being written back as ground truth.
func TestExternalIrrelevantMatchIsRejected(t *testing.T) {
	st := &fakeStore{lib: map[string]types.FoodMatch{}}
	bad := &fakeSource{name: "openfoodfacts", phr: "white rice", match: types.FoodMatch{
		FoodID: "off:bad", Name: "Tortitas de arroz con chocolate blanco", Source: "openfoodfacts",
		Per100g: types.Macros{Calories: 467, Protein: 7.3, Carbs: 63, Fat: 20},
	}}
	good := &fakeSource{name: "usda", phr: "white rice", match: types.FoodMatch{
		FoodID: "usda:1", Name: "Rice, white, long-grain, cooked", Source: "usda",
		Per100g: types.Macros{Calories: 130, Protein: 2.7, Carbs: 28, Fat: 0.3},
	}}
	r := New(st, nil, nil, 0.92, st, bad, good)

	items := []types.ParsedItem{{RawPhrase: "white rice", NormalizedGrams: 200}}
	res, need := r.Resolve(context.Background(), "u1", items)

	if need != 0 {
		t.Fatalf("need=%d, want 0", need)
	}
	if bad.calls != 1 {
		t.Errorf("irrelevant source called %d times, want 1 (tried, then rejected)", bad.calls)
	}
	if good.calls != 1 {
		t.Errorf("relevant source called %d times, want 1 (should be tried after rejection)", good.calls)
	}
	if res[0].Match.FoodID != "usda:1" {
		t.Errorf("resolved via %q, want usda:1 (irrelevant off:bad must be rejected)", res[0].Match.FoodID)
	}
	if len(st.upserts) != 1 || st.upserts[0].FoodID != "usda:1" {
		t.Errorf("write-back upserts = %v, want exactly one usda:1 (bad match must never be aliased)", st.upserts)
	}
}

// TestExternalAllIrrelevantNeedsClarification covers the case where every
// configured source's match is rejected: the item must fall through to
// needing clarification rather than the resolver keeping the first bad match
// as a last resort.
func TestExternalAllIrrelevantNeedsClarification(t *testing.T) {
	st := &fakeStore{lib: map[string]types.FoodMatch{}}
	bad := &fakeSource{name: "openfoodfacts", phr: "white rice", match: types.FoodMatch{
		FoodID: "off:bad", Name: "Tortitas de arroz con chocolate blanco", Source: "openfoodfacts",
		Per100g: types.Macros{Calories: 467},
	}}
	r := New(st, nil, nil, 0.92, st, bad)

	items := []types.ParsedItem{{RawPhrase: "white rice", NormalizedGrams: 200}}
	res, need := r.Resolve(context.Background(), "u1", items)

	if need != 1 {
		t.Fatalf("need=%d, want 1 (no relevant match anywhere)", need)
	}
	if res[0].Match.FoodID != "" {
		t.Errorf("expected no match, got %+v", res[0].Match)
	}
	if len(st.upserts) != 0 {
		t.Errorf("expected no upsert, got %v", st.upserts)
	}
}

func TestUnresolvedAndCountBased(t *testing.T) {
	st := &fakeStore{lib: map[string]types.FoodMatch{"ovo": chicken()}}
	r := New(st, nil, nil, 0.92, st) // no external sources

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

// --- Default serving size tests ---
//
// finalize() falls back to a matched food's ServingSize/ServingUnit when
// grams are unspecified, but only when that unit is an explicit gram unit.
// See resolver.defaultServingGrams.

func eggOFF() types.FoodMatch {
	return types.FoodMatch{FoodID: "off:egg", Name: "egg", Source: "openfoodfacts",
		Per100g:     types.Macros{Calories: 155, Protein: 13, Carbs: 1.1, Fat: 11},
		ServingSize: 50, ServingUnit: "g"}
}

func tacoRice() types.FoodMatch {
	return types.FoodMatch{FoodID: "taco:rice", Name: "arroz branco cozido", Source: "taco",
		Per100g: types.Macros{Calories: 130, Protein: 2.7, Carbs: 28, Fat: 0.3}}
	// TACO never populates ServingSize/ServingUnit — it's a per-100g-only table.
}

// TestFinalizeUsesDefaultServingSizeForGramUnit covers the fix: an
// OFF/USDA-sourced match with a gram ServingSize resolves without
// clarification when the parsed item carries no explicit grams (e.g. "1
// egg"), using ServingSize as the effective portion.
func TestFinalizeUsesDefaultServingSizeForGramUnit(t *testing.T) {
	st := &fakeStore{lib: map[string]types.FoodMatch{"egg": eggOFF()}}
	r := New(st, nil, nil, 0.92, st)

	items := []types.ParsedItem{{RawPhrase: "egg", NormalizedGrams: 0}}
	res, need := r.Resolve(context.Background(), "u1", items)

	if need != 0 {
		t.Fatalf("need=%d, want 0 (default 50g serving should resolve without clarification)", need)
	}
	if want := 155 * 0.5; res[0].Macros.Calories != want {
		t.Errorf("calories = %v, want %v (scaled by default 50g serving)", res[0].Macros.Calories, want)
	}
	// The caller (logmeal.go summaryText) derives "assumed" from exactly these
	// two fields, so both must be intact on the returned item.
	if res[0].Parsed.NormalizedGrams != 0 || res[0].Match.ServingSize != 50 {
		t.Errorf("expected Parsed.NormalizedGrams=0 and Match.ServingSize=50 preserved, got %+v", res[0])
	}
}

// TestFinalizeTacoWithoutServingSizeStillNeedsClarification is the
// regression guard: TACO matches never populate ServingSize, so a
// count-based item ("2 eggs" against a TACO food) must keep requiring
// clarification exactly as before this change.
func TestFinalizeTacoWithoutServingSizeStillNeedsClarification(t *testing.T) {
	st := &fakeStore{lib: map[string]types.FoodMatch{"arroz": tacoRice()}}
	r := New(st, nil, nil, 0.92, st)

	items := []types.ParsedItem{{RawPhrase: "arroz", NormalizedGrams: 0}}
	res, need := r.Resolve(context.Background(), "u1", items)

	if need != 1 {
		t.Fatalf("need=%d, want 1 (TACO has no serving-size default, must still ask for grams)", need)
	}
	if res[0].Match.FoodID != "taco:rice" || res[0].Macros != (types.Macros{}) {
		t.Errorf("expected matched food with zero macros (portion unknown), got %+v", res[0])
	}
}

// TestFinalizeNonMassServingUnitStillNeedsClarification ensures a serving
// unit that isn't a mass ("piece", "ml", ...) is never silently treated as a
// gram count.
func TestFinalizeNonMassServingUnitStillNeedsClarification(t *testing.T) {
	cookie := types.FoodMatch{FoodID: "off:cookie", Name: "cookie", Source: "openfoodfacts",
		Per100g:     types.Macros{Calories: 480, Protein: 6, Carbs: 60, Fat: 22},
		ServingSize: 1, ServingUnit: "piece"}
	st := &fakeStore{lib: map[string]types.FoodMatch{"cookie": cookie}}
	r := New(st, nil, nil, 0.92, st)

	items := []types.ParsedItem{{RawPhrase: "cookie", NormalizedGrams: 0}}
	res, need := r.Resolve(context.Background(), "u1", items)

	if need != 1 {
		t.Fatalf("need=%d, want 1 (non-mass serving unit must not be treated as grams)", need)
	}
	if res[0].Macros != (types.Macros{}) {
		t.Errorf("expected zero macros (portion unknown), got %+v", res[0].Macros)
	}
}

// TestDefaultServingGramsUnitAliases covers case-insensitivity and the
// "gram"/"grams" aliases beyond the bare "g" used above, plus rejection of
// non-gram mass units (kg) per spec: only g/gram/grams should be trusted.
func TestDefaultServingGramsUnitAliases(t *testing.T) {
	tests := []struct {
		name   string
		unit   string
		size   float64
		wantG  float64
		wantOK bool
	}{
		{"lowercase g", "g", 30, 30, true},
		{"uppercase GRAMS", "GRAMS", 30, 30, true},
		{"mixed-case Gram", "Gram", 30, 30, true},
		{"kg not trusted", "kg", 0.03, 0, false},
		{"ml not trusted", "ml", 30, 0, false},
		{"piece not trusted", "piece", 1, 0, false},
		{"zero size", "g", 0, 0, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			match := types.FoodMatch{ServingSize: tt.size, ServingUnit: tt.unit}
			g, ok := defaultServingGrams(match)
			if ok != tt.wantOK || g != tt.wantG {
				t.Errorf("defaultServingGrams(size=%v unit=%q) = (%v, %v), want (%v, %v)",
					tt.size, tt.unit, g, ok, tt.wantG, tt.wantOK)
			}
		})
	}
}

// --- Matcher tests ---

func TestMatcherHit(t *testing.T) {
	st := &fakeStore{lib: map[string]types.FoodMatch{}}
	m := &fakeMatcher{match: map[string]types.FoodMatch{
		"frango": {FoodID: "off:1", Name: "frango", Source: "food_library",
			Per100g: types.Macros{Calories: 165, Protein: 31}, MatchScore: 0.85},
	}}
	src := &fakeSource{name: "off", phr: "frango", match: chicken()}
	r := New(st, m, nil, 0.92, st, src)

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
	r := New(st, m, nil, 0.92, st, src)

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
		"frango": {FoodID: "off:1", Name: "frango", Source: "food_library",
			Per100g: types.Macros{Calories: 165, Protein: 31}, MatchScore: 0.95},
	}}
	r := New(st, m, nil, 0.92, st)

	items := []types.ParsedItem{{RawPhrase: "frango", NormalizedGrams: 100}}
	_, need := r.Resolve(context.Background(), "u1", items)

	if need != 0 {
		t.Fatalf("need=%d, want 0", need)
	}
	// Score >= 0.92 → queued for confirmation, never written straight into the
	// library.
	if len(st.upserts) != 0 {
		t.Errorf("expected no direct upsert for a near-miss, got %d", len(st.upserts))
	}
	if len(st.pendingAliases) != 1 {
		t.Fatalf("expected 1 pending alias, got %d", len(st.pendingAliases))
	}
	pa := st.pendingAliases[0]
	if pa.Phrase != "frango" || pa.FoodID != "off:1" || pa.MatchScore != 0.95 {
		t.Errorf("pending alias = %+v, want phrase=frango foodID=off:1 score=0.95", pa)
	}
}

func TestMatcherWeakMatchNoAlias(t *testing.T) {
	st := &fakeStore{lib: map[string]types.FoodMatch{}}
	m := &fakeMatcher{match: map[string]types.FoodMatch{
		"frango": {FoodID: "off:1", Name: "chicken", Source: "food_library",
			Per100g: types.Macros{Calories: 165, Protein: 31}, MatchScore: 0.85},
	}}
	r := New(st, m, nil, 0.92, st)

	items := []types.ParsedItem{{RawPhrase: "frango", NormalizedGrams: 100}}
	_, _ = r.Resolve(context.Background(), "u1", items)

	// Score < 0.92 → no pending alias, no upsert either.
	if len(st.upserts) != 0 {
		t.Errorf("expected no upsert for weak match, got %d", len(st.upserts))
	}
	if len(st.pendingAliases) != 0 {
		t.Errorf("expected no pending alias for weak match, got %d", len(st.pendingAliases))
	}
}

// TestMatcherIrrelevantMatchRejectedFallsThrough is the mechanical proof for
// the embedding relevance gate: the matcher's top hit clears the similarity
// threshold but shares no real word with the query, so it must be treated as
// a non-match (same as types.ErrNoMatch) and the resolver must fall through
// to external sources exactly as it already does for irrelevant external
// results.
func TestMatcherIrrelevantMatchRejectedFallsThrough(t *testing.T) {
	st := &fakeStore{lib: map[string]types.FoodMatch{}}
	m := &fakeMatcher{match: map[string]types.FoodMatch{
		"leite": {FoodID: "food_library:bad", Name: "Chocolate amargo 70%", Source: "food_library",
			Per100g: types.Macros{Calories: 598}, MatchScore: 0.95},
	}}
	src := &fakeSource{name: "usda", phr: "leite", match: types.FoodMatch{
		FoodID: "usda:milk", Name: "Leite integral pasteurizado", Source: "usda",
		Per100g: types.Macros{Calories: 61, Protein: 3.2, Carbs: 4.8, Fat: 3.3},
	}}
	r := New(st, m, nil, 0.92, st, src)

	items := []types.ParsedItem{{RawPhrase: "leite", NormalizedGrams: 200}}
	res, need := r.Resolve(context.Background(), "u1", items)

	if need != 0 {
		t.Fatalf("need=%d, want 0", need)
	}
	if m.calls != 1 {
		t.Errorf("matcher called %d times, want 1", m.calls)
	}
	if src.calls != 1 {
		t.Errorf("external source called %d times, want 1 (matcher hit must be rejected and fall through)", src.calls)
	}
	if res[0].Match.FoodID != "usda:milk" {
		t.Errorf("resolved via %q, want usda:milk (irrelevant matcher hit must be rejected)", res[0].Match.FoodID)
	}
	if len(st.pendingAliases) != 0 {
		t.Errorf("expected no pending alias for a rejected matcher hit, got %v", st.pendingAliases)
	}
}

// TestNameMatchesQueryArrozSubstringLimitation documents a known limitation
// of the relevance gate rather than asserting it's fully fixed: the real
// production near-miss was query "arroz" matching the unrelated Spanish
// snack "Tortitas de arroz y legumbres". nameMatchesQuery treats any 3+-char
// substring overlap as relevant, and "arroz" literally is a substring of
// that candidate's name, so this specific example still clears the gate.
// The gate added in resolveItem reuses this exact function, so it is
// defense-in-depth, not a guarantee against this precise example — fully
// avoiding it depends on the embedding index surfacing the correct TACO
// "arroz" candidate in the first place (a separate backfill fix).
func TestNameMatchesQueryArrozSubstringLimitation(t *testing.T) {
	if !nameMatchesQuery("arroz", "Tortitas de arroz y legumbres") {
		t.Fatalf("expected nameMatchesQuery to still accept this substring overlap (documents gate limitation, not a regression)")
	}
}

func TestEmbeddingOnWrite(t *testing.T) {
	st := &fakeStore{lib: map[string]types.FoodMatch{}}
	emb := &fakeEmbedder{}
	src := &fakeSource{name: "off", phr: "frango", match: chicken()}
	r := New(st, nil, emb, 0.92, st, src)

	items := []types.ParsedItem{{RawPhrase: "frango", NormalizedGrams: 100}}
	_, need := r.Resolve(context.Background(), "u1", items)

	if need != 0 {
		t.Fatalf("need=%d, want 0", need)
	}
	if len(emb.embeds) != 1 {
		t.Fatalf("embed called %d times, want 1", len(emb.embeds))
	}
	if emb.embeds[0].foodID != "off:1" || emb.embeds[0].name != "frango grelhado" {
		t.Errorf("embed args = %+v, want foodID=off:1 name='frango grelhado'", emb.embeds[0])
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
			// Name shares "frango" with the query so it clears the relevance
			// gate; FoodID/Calories differ from the external chicken() fixture
			// so the two paths remain distinguishable.
			"frango": {FoodID: "matcher:wrong", Name: "frango errado", Source: "food_library",
				Per100g: types.Macros{Calories: 1}, MatchScore: 0.99},
		}}
	}

	items := []types.ParsedItem{{RawPhrase: "frango", NormalizedGrams: 100}}

	// Tier 0: nil matcher, nil embedder. Resolves via external source only.
	stOff := &fakeStore{lib: map[string]types.FoodMatch{}}
	embOff := &fakeEmbedder{}
	srcOff := &fakeSource{name: "off", phr: "frango", match: chicken()}
	rOff := New(stOff, nil, nil, 0.92, stOff, srcOff)
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
	rOn := New(stOn, m, &fakeEmbedder{}, 0.92, stOn, srcOn)
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
	r := New(st, nil, emb, 0.92, st)

	items := []types.ParsedItem{{RawPhrase: "frango", NormalizedGrams: 200}}
	_, _ = r.Resolve(context.Background(), "u1", items)

	// Embed not called because it was a local hit, not external.
	if len(emb.embeds) != 0 {
		t.Errorf("embed called %d times on local hit, want 0", len(emb.embeds))
	}
}

// --- Precedence tests ---

func TestPrecedenceOrderDeterminesWinner(t *testing.T) {
	// Both sources match the same phrase; whichever comes first in the user's
	// precedence order should win.
	first := &fakeSource{name: "taco", phr: "frango", match: types.FoodMatch{FoodID: "taco:1", Name: "frango taco"}}
	second := &fakeSource{name: "off", phr: "frango", match: chicken()}

	st := &fakeStore{lib: map[string]types.FoodMatch{}, precedence: []string{"off", "taco"}}
	r := New(st, nil, nil, 0.92, st, first, second) // default order: taco, off

	items := []types.ParsedItem{{RawPhrase: "frango", NormalizedGrams: 100}}
	res, need := r.Resolve(context.Background(), "u1", items)

	if need != 0 {
		t.Fatalf("need=%d, want 0", need)
	}
	// User's precedence puts "off" first, so it should win despite "taco" being
	// registered first (the default order).
	if res[0].Match.FoodID != "off:1" {
		t.Errorf("resolved via %q, want off:1 (per-user precedence should override default order)", res[0].Match.FoodID)
	}
	if first.calls != 0 {
		t.Errorf("taco (lower precedence) called %d times, want 0", first.calls)
	}
	if second.calls != 1 {
		t.Errorf("off (higher precedence) called %d times, want 1", second.calls)
	}
}

func TestPrecedenceFallsBackToDefaultOnEmpty(t *testing.T) {
	first := &fakeSource{name: "taco", phr: "frango", match: types.FoodMatch{FoodID: "taco:1", Name: "frango taco"}}
	second := &fakeSource{name: "off", phr: "frango", match: chicken()}

	st := &fakeStore{lib: map[string]types.FoodMatch{}} // no customized precedence
	r := New(st, nil, nil, 0.92, st, first, second)     // default order: taco, off

	items := []types.ParsedItem{{RawPhrase: "frango", NormalizedGrams: 100}}
	res, _ := r.Resolve(context.Background(), "u1", items)

	if res[0].Match.FoodID != "taco:1" {
		t.Errorf("resolved via %q, want taco:1 (fall back to default order)", res[0].Match.FoodID)
	}
}

func TestPrecedenceFallsBackToDefaultOnError(t *testing.T) {
	first := &fakeSource{name: "taco", phr: "frango", match: types.FoodMatch{FoodID: "taco:1", Name: "frango taco"}}
	second := &fakeSource{name: "off", phr: "frango", match: chicken()}

	st := &fakeStore{lib: map[string]types.FoodMatch{}, precedenceErr: types.ErrNotFound}
	r := New(st, nil, nil, 0.92, st, first, second) // default order: taco, off

	items := []types.ParsedItem{{RawPhrase: "frango", NormalizedGrams: 100}}
	res, _ := r.Resolve(context.Background(), "u1", items)

	if res[0].Match.FoodID != "taco:1" {
		t.Errorf("resolved via %q, want taco:1 (fall back to default order on error)", res[0].Match.FoodID)
	}
}
