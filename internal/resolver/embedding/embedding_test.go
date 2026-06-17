package embedding

import (
	"context"
	"database/sql"
	"math"
	"testing"

	_ "modernc.org/sqlite"

	"github.com/gsaraiva2109/dietdaemon/core/types"
	"github.com/gsaraiva2109/dietdaemon/internal/index"
)

// --- Fakes ---

// stubModel returns canned embedding vectors. The vector for "chicken" is
// near [1,0]; the vector for "frango" (Portuguese) is also near [1,0] so they
// match across languages. Unrelated phrases map to [0,1].
type stubModel struct {
	embedMap map[string][]float32
}

func (m *stubModel) Embed(_ context.Context, text string) ([]float32, error) {
	if v, ok := m.embedMap[text]; ok {
		return v, nil
	}
	// Default: orthogonal vector.
	return []float32{0, 1}, nil
}

func (m *stubModel) Complete(_ context.Context, _ string) (string, error) {
	return "", nil
}

// stubStore is a minimal in-memory FoodStore for the embedding matcher.
type stubStore struct {
	foods map[string]types.FoodMatch // foodID -> match
}

func (s *stubStore) LookupFood(_ context.Context, _, _ string) (types.FoodMatch, error) {
	return types.FoodMatch{}, types.ErrNoMatch
}
func (s *stubStore) GetFood(_ context.Context, _, foodID string) (types.FoodMatch, error) {
	if fm, ok := s.foods[foodID]; ok {
		return fm, nil
	}
	return types.FoodMatch{}, types.ErrNoMatch
}
func (s *stubStore) UpsertFood(_ context.Context, _ string, _ types.FoodMatch, _ []string) error {
	return nil
}
func (s *stubStore) RecordFoodQuery(_ context.Context, _, _ string) error {
	return nil
}

func openTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	const q = `
		CREATE TABLE IF NOT EXISTS food_vectors (
			user_id TEXT NOT NULL,
			food_id TEXT NOT NULL,
			dim     INTEGER NOT NULL,
			vec     BLOB NOT NULL,
			PRIMARY KEY (user_id, food_id)
		);
	`
	if _, err := db.Exec(q); err != nil {
		t.Fatalf("create table: %v", err)
	}
	return db
}

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

func TestMatchAboveThreshold(t *testing.T) {
	db := openTestDB(t)
	idx := index.New(db)

	// Store a food in the library and its vector in the index.
	st := &stubStore{foods: map[string]types.FoodMatch{
		"chicken-1": {FoodID: "chicken-1", Name: "Chicken Breast", Source: "food_library",
			Per100g: types.Macros{Calories: 165, Protein: 31}},
	}}
	requireNoErr(t, idx.Upsert(context.Background(), "u1", "chicken-1", []float32{1, 0}))

	model := &stubModel{embedMap: map[string][]float32{
		"frango": {0.95, 0.05}, // close to chicken [1,0]
	}}
	m := New(model, idx, st, 0.80)

	// Portuguese "frango" should match chicken via embedding.
	fm, err := m.Match(context.Background(), "u1", "frango")
	if err != nil {
		t.Fatalf("Match: %v", err)
	}
	if fm.FoodID != "chicken-1" {
		t.Errorf("FoodID = %q, want chicken-1", fm.FoodID)
	}
	if fm.MatchScore < 0.80 {
		t.Errorf("MatchScore = %v, want >= 0.80", fm.MatchScore)
	}
	if fm.Name != "Chicken Breast" {
		t.Errorf("Name = %q", fm.Name)
	}
}

func TestMatchBelowThreshold(t *testing.T) {
	db := openTestDB(t)
	idx := index.New(db)

	st := &stubStore{foods: map[string]types.FoodMatch{
		"chicken-1": {FoodID: "chicken-1", Name: "Chicken", Per100g: types.Macros{Calories: 165}},
	}}
	requireNoErr(t, idx.Upsert(context.Background(), "u1", "chicken-1", []float32{1, 0}))

	// "pizza" maps to [0,1] — cosine with [1,0] is 0.
	model := &stubModel{embedMap: map[string][]float32{
		"pizza": {0, 1},
	}}
	m := New(model, idx, st, 0.80)

	_, err := m.Match(context.Background(), "u1", "pizza")
	if err != types.ErrNoMatch {
		t.Errorf("expected ErrNoMatch for unrelated phrase, got %v", err)
	}
}

func TestMatchEmptyIndex(t *testing.T) {
	db := openTestDB(t)
	idx := index.New(db)
	st := &stubStore{foods: map[string]types.FoodMatch{}}
	model := &stubModel{embedMap: map[string][]float32{
		"frango": {1, 0},
	}}
	m := New(model, idx, st, 0.80)

	_, err := m.Match(context.Background(), "u1", "frango")
	if err != types.ErrNoMatch {
		t.Errorf("expected ErrNoMatch when index is empty, got %v", err)
	}
}

func TestEmbedFood(t *testing.T) {
	db := openTestDB(t)
	idx := index.New(db)
	st := &stubStore{foods: map[string]types.FoodMatch{}}
	model := &stubModel{embedMap: map[string][]float32{
		"Chicken Breast": {1, 0, 0.5},
	}}
	m := New(model, idx, st, 0.80)

	requireNoErr(t, m.EmbedFood(context.Background(), "u1", "chicken-1", "Chicken Breast"))

	// Verify the vector was stored by querying for it.
	nn, err := idx.Nearest(context.Background(), "u1", []float32{1, 0, 0.5}, 1)
	requireNoErr(t, err)
	if len(nn) != 1 || nn[0].FoodID != "chicken-1" {
		t.Fatalf("nearest after EmbedFood = %+v, want chicken-1", nn)
	}
	if math.Abs(nn[0].Score-1.0) > 1e-6 {
		t.Errorf("score = %v, want ~1.0 for identical vector", nn[0].Score)
	}
}

// TestTier2CrossLanguageEmbedding is the judgment-level guarantee for Tier 1/2:
// the embedding matcher recognizes the same library food across languages and
// phrasings, while an unrelated phrase stays below the threshold (never a silent
// weak guess). The user logged "chicken breast" once; later "frango" (PT) and
// "chicken" (EN) must both resolve to it, but "pizza" must not.
func TestTier2CrossLanguageEmbedding(t *testing.T) {
	db := openTestDB(t)
	idx := index.New(db)

	st := &stubStore{foods: map[string]types.FoodMatch{
		"chicken-1": {FoodID: "chicken-1", Name: "Chicken Breast", Source: "food_library",
			Per100g: types.Macros{Calories: 165, Protein: 31}},
	}}
	requireNoErr(t, idx.Upsert(context.Background(), "u1", "chicken-1", []float32{1, 0}))

	// "frango" and "chicken" land near the stored [1,0]; "pizza" is orthogonal.
	model := &stubModel{embedMap: map[string][]float32{
		"frango":  {0.97, 0.04},
		"chicken": {0.99, 0.01},
		"pizza":   {0, 1},
	}}
	m := New(model, idx, st, 0.80)

	for _, phrase := range []string{"frango", "chicken"} {
		fm, err := m.Match(context.Background(), "u1", phrase)
		if err != nil {
			t.Fatalf("Match(%q): %v", phrase, err)
		}
		if fm.FoodID != "chicken-1" {
			t.Errorf("Match(%q) FoodID = %q, want chicken-1", phrase, fm.FoodID)
		}
		if fm.MatchScore < 0.80 {
			t.Errorf("Match(%q) score = %v, want >= 0.80", phrase, fm.MatchScore)
		}
	}

	// Unrelated phrase must stay below threshold — no silent weak match.
	if _, err := m.Match(context.Background(), "u1", "pizza"); err != types.ErrNoMatch {
		t.Errorf("Match(pizza) = %v, want ErrNoMatch", err)
	}
}

func requireNoErr(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
