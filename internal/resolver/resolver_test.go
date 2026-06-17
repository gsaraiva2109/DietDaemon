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
	recorded  []string // foodIDs passed to RecordFoodQuery
	upsertErr error
}

func (f *fakeStore) LookupFood(_ context.Context, _, phrase string) (types.FoodMatch, error) {
	if m, ok := f.lib[phrase]; ok {
		return m, nil
	}
	return types.FoodMatch{}, types.ErrNoMatch
}

func (f *fakeStore) UpsertFood(_ context.Context, _ string, match types.FoodMatch, _ []string) error {
	f.upserts = append(f.upserts, match)
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

func chicken() types.FoodMatch {
	return types.FoodMatch{FoodID: "off:1", Name: "chicken breast", Source: "openfoodfacts",
		Per100g: types.Macros{Calories: 165, Protein: 31, Carbs: 0, Fat: 3.6}}
}

func TestLocalFirstHit(t *testing.T) {
	st := &fakeStore{lib: map[string]types.FoodMatch{"frango": chicken()}}
	src := &fakeSource{name: "off", phr: "frango", match: chicken()}
	r := New(st, src)

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
	r := New(st, miss, hit) // order matters: miss first

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
	r := New(st) // no external sources

	items := []types.ParsedItem{
		{RawPhrase: "ovo", NormalizedGrams: 0},     // matched food, unknown portion
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
