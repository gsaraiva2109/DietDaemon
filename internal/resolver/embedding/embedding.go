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

// Match embeds phrase, finds the nearest neighbour in the user's index, and
// returns the corresponding FoodMatch when the similarity meets the threshold.
// Returns types.ErrNoMatch when no neighbour clears the threshold.
func (m *Matcher) Match(ctx context.Context, userID, phrase string) (types.FoodMatch, error) {
	vec, err := m.model.Embed(ctx, phrase)
	if err != nil {
		return types.FoodMatch{}, fmt.Errorf("embedding: embed: %w", err)
	}

	nn, err := m.idx.Nearest(ctx, userID, vec, 1)
	if err != nil {
		return types.FoodMatch{}, fmt.Errorf("embedding: nearest: %w", err)
	}
	if len(nn) == 0 || nn[0].Score < m.threshold {
		return types.FoodMatch{}, types.ErrNoMatch
	}

	fm, err := m.store.GetFood(ctx, userID, nn[0].FoodID)
	if err != nil {
		return types.FoodMatch{}, fmt.Errorf("embedding: get food: %w", err)
	}
	fm.MatchScore = nn[0].Score
	return fm, nil
}

// SetThreshold overrides the match threshold for benchmarking.
func (m *Matcher) SetThreshold(t float64) { m.threshold = t }

// EmbedFood embeds the canonical food name and upserts the vector into the
// index so future embedding queries can match it.
func (m *Matcher) EmbedFood(ctx context.Context, userID, foodID, name string) error {
	vec, err := m.model.Embed(ctx, name)
	if err != nil {
		return fmt.Errorf("embedding: embed food: %w", err)
	}
	return m.idx.Upsert(ctx, userID, foodID, vec)
}
