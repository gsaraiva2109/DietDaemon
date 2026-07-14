// Package embedding implements the resolver.Matcher and resolver.Embedder
// interfaces using an embedding model and a brute-force cosine index.
package embedding

import (
	"context"
	"fmt"

	"github.com/gsaraiva2109/dietdaemon/core/ports"
	"github.com/gsaraiva2109/dietdaemon/core/types"
	"github.com/gsaraiva2109/dietdaemon/internal/index"
	"github.com/gsaraiva2109/dietdaemon/internal/resolver"
)

// Compile-time interface checks.
var (
	_ resolver.Matcher  = (*Matcher)(nil)
	_ resolver.Embedder = (*Matcher)(nil)
)

// Matcher implements resolver.Matcher and resolver.Embedder.
type Matcher struct {
	model     ports.ModelAdapter
	idx       index.Index
	store     resolver.FoodStore
	threshold float64
}

// New returns a ready Matcher. threshold is the minimum cosine similarity for
// a match to be accepted (e.g. 0.80).
func New(model ports.ModelAdapter, idx index.Index, store resolver.FoodStore, threshold float64) *Matcher {
	return &Matcher{
		model:     model,
		idx:       idx,
		store:     store,
		threshold: threshold,
	}
}

// Match embeds phrase, finds the nearest neighbour in the global embedding
// index, and returns the corresponding FoodMatch when the similarity meets
// the threshold. Returns types.ErrNoMatch when no neighbour clears the
// threshold. userID is unused for the search itself (the index is global,
// shared by every user) but kept in the signature for interface conformance.
func (m *Matcher) Match(ctx context.Context, userID, phrase string) (types.FoodMatch, error) {
	_ = userID
	vec, err := m.model.Embed(ctx, phrase)
	if err != nil {
		return types.FoodMatch{}, fmt.Errorf("embedding: embed: %w", err)
	}

	nn, err := m.idx.Nearest(ctx, vec, 1)
	if err != nil {
		return types.FoodMatch{}, fmt.Errorf("embedding: nearest: %w", err)
	}
	if len(nn) == 0 || nn[0].Score < m.threshold {
		return types.FoodMatch{}, types.ErrNoMatch
	}

	fm, err := m.store.GetFood(ctx, nn[0].FoodID)
	if err != nil {
		return types.FoodMatch{}, fmt.Errorf("embedding: get food: %w", err)
	}
	fm.MatchScore = nn[0].Score
	return fm, nil
}

// SetThreshold overrides the match threshold for benchmarking.
func (m *Matcher) SetThreshold(t float64) { m.threshold = t }

// EmbedFood embeds the canonical food name and upserts the vector into the
// global index so future embedding queries can match it. userID is unused
// (kept for interface conformance): an embedding is a pure function of name,
// so it's computed and stored once per foodID regardless of which user
// triggered the resolution. Skips the (costly) model call entirely when a
// vector for this food already exists.
func (m *Matcher) EmbedFood(ctx context.Context, userID, foodID, name string) error {
	_ = userID
	exists, err := m.idx.Exists(ctx, foodID)
	if err != nil {
		return fmt.Errorf("embedding: check exists: %w", err)
	}
	if exists {
		return nil
	}
	vec, err := m.model.Embed(ctx, name)
	if err != nil {
		return fmt.Errorf("embedding: embed food: %w", err)
	}
	return m.idx.Upsert(ctx, foodID, vec)
}

// backfillStore is the subset of the store needed to enumerate foods missing
// a vector. Declared here rather than added to resolver.FoodStore since only
// the concrete store needs bulk enumeration, not every Matcher caller; the
// concrete store satisfies it and BackfillEmbeddings type-asserts for it.
type backfillStore interface {
	ListFoodsWithoutVectors(ctx context.Context) ([]types.FoodMatch, error)
}

// BackfillEmbeddings embeds every catalog food that has no vector yet, e.g.
// foods written by a bulk import (which never calls EmbedFood) rather than
// the live resolver's embedding-on-write path. It calls EmbedFood
// sequentially per food (the Ollama adapter has no batch-embed call) so one
// failed food is skipped rather than aborting the run; progress, if non-nil,
// is called once per food after it's processed with that food's error (nil
// on success) so the caller can log/aggregate failures instead of only
// seeing an opaque final count. Returns the counts of successfully embedded
// and failed foods.
func (m *Matcher) BackfillEmbeddings(ctx context.Context, progress func(done, total int, itemErr error)) (embedded, failed int, err error) {
	bs, ok := m.store.(backfillStore)
	if !ok {
		return 0, 0, fmt.Errorf("embedding: backfill: store does not support ListFoodsWithoutVectors")
	}

	foods, err := bs.ListFoodsWithoutVectors(ctx)
	if err != nil {
		return 0, 0, fmt.Errorf("embedding: backfill: list foods: %w", err)
	}

	total := len(foods)
	for i, food := range foods {
		itemErr := m.EmbedFood(ctx, "", food.FoodID, food.Name)
		if itemErr != nil {
			failed++
		} else {
			embedded++
		}
		if progress != nil {
			progress(i+1, total, itemErr)
		}
	}
	return embedded, failed, nil
}
