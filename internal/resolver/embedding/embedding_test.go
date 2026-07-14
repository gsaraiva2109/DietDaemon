package embedding

import (
	"context"
	"database/sql"
	"fmt"
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
	failFor  map[string]bool // texts that should return an error from Embed
}

func (m *stubModel) Embed(_ context.Context, text string) ([]float32, error) {
	if m.failFor[text] {
		return nil, fmt.Errorf("stub: embed failed for %q", text)
	}
	if v, ok := m.embedMap[text]; ok {
		return v, nil
	}
	// Default: orthogonal vector.
	return []float32{0, 1}, nil
}

func (m *stubModel) Complete(_ context.Context, _ string) (string, error) {
	return "", nil
}

// stubStore is a minimal in-memory FoodStore for the embedding matcher. It
// also implements backfillStore so BackfillEmbeddings tests don't need a
// real DB.
type stubStore struct {
	foods map[string]types.FoodMatch // foodID -> match

	missingVectors    []types.FoodMatch
	missingVectorsErr error
}

func (s *stubStore) LookupFood(_ context.Context, _, _ string) (types.FoodMatch, error) {
	return types.FoodMatch{}, types.ErrNoMatch
}
func (s *stubStore) GetFood(_ context.Context, foodID string) (types.FoodMatch, error) {
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
func (s *stubStore) AddPendingAlias(_ context.Context, _, _, _ string, _ float64) error {
	return nil
}

// missingVectors is the set of foods stubStore reports as missing a vector,
// so tests can exercise BackfillEmbeddings without a real store's
// LEFT JOIN food_vectors query.
func (s *stubStore) ListFoodsWithoutVectors(_ context.Context) ([]types.FoodMatch, error) {
	if s.missingVectorsErr != nil {
		return nil, s.missingVectorsErr
	}
	return s.missingVectors, nil
}

func openTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	const q = `
		CREATE TABLE IF NOT EXISTS food_vectors (
			food_id TEXT PRIMARY KEY,
			dim     INTEGER NOT NULL,
			vec     BLOB NOT NULL
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
	requireNoErr(t, idx.Upsert(context.Background(), "chicken-1", []float32{1, 0}))

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
	requireNoErr(t, idx.Upsert(context.Background(), "chicken-1", []float32{1, 0}))

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
	nn, err := idx.Nearest(context.Background(), []float32{1, 0, 0.5}, 1)
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
	requireNoErr(t, idx.Upsert(context.Background(), "chicken-1", []float32{1, 0}))

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

func TestBackfillEmbeddings(t *testing.T) {
	db := openTestDB(t)
	idx := index.New(db)

	st := &stubStore{
		foods: map[string]types.FoodMatch{},
		missingVectors: []types.FoodMatch{
			{FoodID: "arroz-1", Name: "Arroz, tipo 1, cozido"},
			{FoodID: "feijao-1", Name: "Feijao carioca, cozido"},
		},
	}
	model := &stubModel{embedMap: map[string][]float32{
		"Arroz, tipo 1, cozido":  {1, 0},
		"Feijao carioca, cozido": {0, 1},
	}}
	m := New(model, idx, st, 0.80)

	var progressCalls []int
	embedded, failed, err := m.BackfillEmbeddings(context.Background(), func(done, total int) {
		progressCalls = append(progressCalls, done)
		if total != 2 {
			t.Errorf("progress total = %d, want 2", total)
		}
	})
	if err != nil {
		t.Fatalf("BackfillEmbeddings: %v", err)
	}
	if embedded != 2 || failed != 0 {
		t.Fatalf("embedded=%d failed=%d, want 2/0", embedded, failed)
	}
	if len(progressCalls) != 2 || progressCalls[0] != 1 || progressCalls[1] != 2 {
		t.Fatalf("progress calls = %v, want [1 2]", progressCalls)
	}

	for _, foodID := range []string{"arroz-1", "feijao-1"} {
		exists, err := idx.Exists(context.Background(), foodID)
		requireNoErr(t, err)
		if !exists {
			t.Errorf("expected %q to have a vector after backfill", foodID)
		}
	}
}

func TestBackfillEmbeddings_SkipsAlreadyEmbedded(t *testing.T) {
	db := openTestDB(t)
	idx := index.New(db)
	// Pre-seed a vector directly; the store still reports this food as
	// missing-vectors only if it genuinely is missing, so here we prove
	// EmbedFood's own "skip the model call" short-circuit still applies
	// when a food that's already indexed somehow appears in the backfill
	// list (e.g. a race with a concurrent live resolve).
	requireNoErr(t, idx.Upsert(context.Background(), "chicken-1", []float32{1, 0}))

	st := &stubStore{
		foods: map[string]types.FoodMatch{},
		missingVectors: []types.FoodMatch{
			{FoodID: "chicken-1", Name: "Chicken Breast"},
		},
	}
	model := &stubModel{failFor: map[string]bool{"Chicken Breast": true}}
	m := New(model, idx, st, 0.80)

	embedded, failed, err := m.BackfillEmbeddings(context.Background(), nil)
	if err != nil {
		t.Fatalf("BackfillEmbeddings: %v", err)
	}
	// EmbedFood short-circuits on existing vectors before calling the model,
	// so this counts as embedded (no-op success), not failed, even though
	// the model would have errored.
	if embedded != 1 || failed != 0 {
		t.Fatalf("embedded=%d failed=%d, want 1/0", embedded, failed)
	}
}

func TestBackfillEmbeddings_OneFailureDoesNotAbortBatch(t *testing.T) {
	db := openTestDB(t)
	idx := index.New(db)

	st := &stubStore{
		foods: map[string]types.FoodMatch{},
		missingVectors: []types.FoodMatch{
			{FoodID: "bad-1", Name: "Bad Food"},
			{FoodID: "good-1", Name: "Good Food"},
			{FoodID: "good-2", Name: "Good Food 2"},
		},
	}
	model := &stubModel{
		embedMap: map[string][]float32{
			"Good Food":   {1, 0},
			"Good Food 2": {0, 1},
		},
		failFor: map[string]bool{"Bad Food": true},
	}
	m := New(model, idx, st, 0.80)

	embedded, failed, err := m.BackfillEmbeddings(context.Background(), nil)
	if err != nil {
		t.Fatalf("BackfillEmbeddings: %v", err)
	}
	if embedded != 2 || failed != 1 {
		t.Fatalf("embedded=%d failed=%d, want 2/1", embedded, failed)
	}

	for _, foodID := range []string{"good-1", "good-2"} {
		exists, err := idx.Exists(context.Background(), foodID)
		requireNoErr(t, err)
		if !exists {
			t.Errorf("expected %q to have a vector despite an earlier failure", foodID)
		}
	}
	exists, err := idx.Exists(context.Background(), "bad-1")
	requireNoErr(t, err)
	if exists {
		t.Errorf("bad-1 should have no vector since its embed call failed")
	}
}

func TestBackfillEmbeddings_EmptyCatalogIsNoOp(t *testing.T) {
	db := openTestDB(t)
	idx := index.New(db)
	st := &stubStore{foods: map[string]types.FoodMatch{}, missingVectors: nil}
	model := &stubModel{}
	m := New(model, idx, st, 0.80)

	var progressCalled bool
	embedded, failed, err := m.BackfillEmbeddings(context.Background(), func(_, _ int) {
		progressCalled = true
	})
	if err != nil {
		t.Fatalf("BackfillEmbeddings: %v", err)
	}
	if embedded != 0 || failed != 0 {
		t.Fatalf("embedded=%d failed=%d, want 0/0 for empty catalog", embedded, failed)
	}
	if progressCalled {
		t.Errorf("progress should not be called when there is nothing to backfill")
	}
}

func requireNoErr(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
