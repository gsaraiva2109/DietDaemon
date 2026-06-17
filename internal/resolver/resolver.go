// Package resolver performs Stage B: turning parsed food items into resolved
// items with macros. It is local-first — the user's personal food library
// (persisted in the store) is consulted before any external NutritionSource, so
// a repeating diet resolves offline, instantly, and for free after the first
// time each food is seen. External lookups are written back to the library and
// their query frequency is recorded so common foods rank first.
package resolver

import (
	"context"
	"errors"

	"github.com/gsaraiva2109/dietdaemon/core/types"
)

// FoodStore is the subset of ports.Store the resolver needs. Declaring it here
// (rather than depending on the whole Store) keeps the resolver decoupled and
// trivially testable; the concrete store satisfies it automatically.
type FoodStore interface {
	LookupFood(ctx context.Context, userID, phrase string) (types.FoodMatch, error)
	UpsertFood(ctx context.Context, userID string, match types.FoodMatch, aliases []string) error
	RecordFoodQuery(ctx context.Context, userID, foodID string) error
}

// Source resolves a parsed item against an external nutrition database. It
// matches ports.NutritionSource.
type Source interface {
	Resolve(ctx context.Context, item types.ParsedItem) (types.FoodMatch, error)
	Name() string
}

// Resolver orchestrates local-first resolution over a store and an ordered list
// of external sources.
type Resolver struct {
	store   FoodStore
	sources []Source
}

// New builds a resolver. Sources are queried in the given order, only after the
// local food library misses.
func New(store FoodStore, sources ...Source) *Resolver {
	return &Resolver{store: store, sources: sources}
}

// Resolve resolves every parsed item for a user. It returns the resolved items
// (in input order) and the number that still need clarification — either no
// food matched, or the food matched but the portion is unknown (count-based
// items such as "2 eggs"). The caller's confidence gate / clarification loop
// uses that count. Resolve is resilient: a single failing item or source never
// aborts the batch.
func (r *Resolver) Resolve(ctx context.Context, userID string, items []types.ParsedItem) ([]types.ResolvedItem, int) {
	resolved := make([]types.ResolvedItem, 0, len(items))
	needsClarification := 0
	for _, item := range items {
		ri, ok := r.resolveItem(ctx, userID, item)
		resolved = append(resolved, ri)
		if !ok {
			needsClarification++
		}
	}
	return resolved, needsClarification
}

// resolveItem resolves one item. ok is false when the item needs clarification.
func (r *Resolver) resolveItem(ctx context.Context, userID string, item types.ParsedItem) (types.ResolvedItem, bool) {
	// 1. Local-first: the personal food library.
	if match, err := r.store.LookupFood(ctx, userID, item.RawPhrase); err == nil {
		_ = r.store.RecordFoodQuery(ctx, userID, match.FoodID)
		return finalize(item, match)
	} else if !errors.Is(err, types.ErrNoMatch) {
		// A real store error: degrade gracefully and try external sources.
		_ = err
	}

	// 2. External sources, in configured order. First match wins.
	for _, src := range r.sources {
		match, err := src.Resolve(ctx, item)
		if err != nil { // ErrNoMatch or transient: skip to the next source.
			continue
		}
		// Write back into the personal library so the next lookup is local, and
		// record this query so frequency ranking improves over time.
		_ = r.store.UpsertFood(ctx, userID, match, []string{item.RawPhrase})
		_ = r.store.RecordFoodQuery(ctx, userID, match.FoodID)
		return finalize(item, match)
	}

	// 3. Nothing matched: unresolved, needs clarification.
	return types.ResolvedItem{Parsed: item}, false
}

// finalize attaches a matched food and scales its per-100g macros to the
// portion. Count-based items (grams unknown) keep zero macros and are flagged
// as needing clarification so the portion can be confirmed later.
func finalize(item types.ParsedItem, match types.FoodMatch) (types.ResolvedItem, bool) {
	ri := types.ResolvedItem{Parsed: item, Match: match}
	if item.NormalizedGrams <= 0 {
		return ri, false // food known, portion unknown
	}
	ri.Macros = match.Per100g.Scale(item.NormalizedGrams / 100.0)
	return ri, true
}
