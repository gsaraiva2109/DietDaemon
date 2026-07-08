// Package resolver performs Stage B: turning parsed food items into resolved
// items with macros. It is local-first — the user's personal food library
// (persisted in the store) is consulted before any external NutritionSource, so
// a repeating diet resolves offline, instantly, and for free after the first
// time each food is seen. External lookups are written back to the library and
// their query frequency is recorded so common foods rank first.
//
// When a Matcher is configured (Tier >= 1), the resolver also
// consults an embedding nearest-neighbour search after the exact alias lookup
// and before external sources. On an external hit the resolver embeds the
// canonical food name into the index so future queries can match it via
// embedding (embedding-on-write).
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
	GetFood(ctx context.Context, foodID string) (types.FoodMatch, error)
	UpsertFood(ctx context.Context, userID string, match types.FoodMatch, aliases []string) error
	RecordFoodQuery(ctx context.Context, userID, foodID string) error
	// AddPendingAlias queues an embedding near-miss for user confirmation
	// instead of writing it straight into the personal food library.
	AddPendingAlias(ctx context.Context, userID, phrase, foodID string, matchScore float64) error
}

// PrecedenceStore resolves a user's preferred order of external nutrition
// sources. Implementations return an empty slice (not an error) when the user
// has no customized order; the resolver falls back to the startup-configured
// default order in that case.
type PrecedenceStore interface {
	GetSourcePrecedence(ctx context.Context, userID string) ([]string, error)
}

// Source resolves a parsed item against an external nutrition database. It
// matches ports.NutritionSource.
type Source interface {
	Resolve(ctx context.Context, item types.ParsedItem) (types.FoodMatch, error)
	Name() string
}

// Matcher performs embedding-based nearest-neighbour lookup over the user's
// personal food library. It is optional (nil = today's behaviour).
type Matcher interface {
	// Match returns a library food whose embedding is nearest phrase, or
	// types.ErrNoMatch when nothing clears the similarity threshold.
	Match(ctx context.Context, userID, phrase string) (types.FoodMatch, error)
}

// Embedder is an optional hook for embedding a canonical food name after an
// external source resolves it, so future queries can match via the index.
type Embedder interface {
	EmbedFood(ctx context.Context, userID, foodID, name string) error
}

// Resolver orchestrates local-first resolution over a store, an optional
// embedding matcher, and an ordered list of external sources.
type Resolver struct {
	store          FoodStore
	matcher        Matcher  // nil when Tier 0
	embed          Embedder // nil when Tier 0, called on external write-back
	precedence     PrecedenceStore
	sourcesByName  map[string]Source
	defaultOrder   []string // startup-configured NUTRITION_SOURCE order, keyed by Name()
	aliasThreshold float64  // min similarity for alias write-back (default 0.92)
}

// New builds a resolver. Sources are queried in the given order by default;
// a user with a customized precedence (via precedence.GetSourcePrecedence)
// gets their own order instead. matcher and embedder may be nil for Tier 0
// behaviour.
func New(store FoodStore, matcher Matcher, embed Embedder, aliasThreshold float64, precedence PrecedenceStore, sources ...Source) *Resolver {
	if aliasThreshold <= 0 {
		aliasThreshold = 0.92
	}
	sourcesByName := make(map[string]Source, len(sources))
	defaultOrder := make([]string, 0, len(sources))
	for _, src := range sources {
		sourcesByName[src.Name()] = src
		defaultOrder = append(defaultOrder, src.Name())
	}
	return &Resolver{
		store:          store,
		matcher:        matcher,
		embed:          embed,
		precedence:     precedence,
		sourcesByName:  sourcesByName,
		defaultOrder:   defaultOrder,
		aliasThreshold: aliasThreshold,
	}
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
	// 1. Local-first: exact alias in the personal food library.
	if match, err := r.store.LookupFood(ctx, userID, item.RawPhrase); err == nil {
		_ = r.store.RecordFoodQuery(ctx, userID, match.FoodID)
		return finalize(item, match)
	} else if !errors.Is(err, types.ErrNoMatch) {
		// A real store error: degrade gracefully and try next steps.
		_ = err
	}

	// 2. Embedding matcher (when configured): nearest-neighbour in the library.
	if r.matcher != nil {
		match, err := r.matcher.Match(ctx, userID, item.RawPhrase)
		if err == nil {
			_ = r.store.RecordFoodQuery(ctx, userID, match.FoodID)
			// Queue the new phrasing for user confirmation when the match is
			// strong, rather than writing it straight into the library.
			if match.MatchScore >= r.aliasThreshold {
				_ = r.store.AddPendingAlias(ctx, userID, item.RawPhrase, match.FoodID, match.MatchScore)
			}
			return finalize(item, match)
		} else if !errors.Is(err, types.ErrNoMatch) {
			_ = err // real error, fall through
		}
	}

	// 3. External sources, in the user's precedence order (falling back to the
	// startup-configured default). First match wins.
	order := r.defaultOrder
	if r.precedence != nil {
		if o, err := r.precedence.GetSourcePrecedence(ctx, userID); err == nil && len(o) > 0 {
			order = o
		}
	}
	for _, name := range order {
		src, ok := r.sourcesByName[name]
		if !ok {
			continue // unknown source name: skip
		}
		match, err := src.Resolve(ctx, item)
		if err != nil { // ErrNoMatch or transient: skip to the next source.
			continue
		}
		// Write back into the personal library so the next lookup is local, and
		// record this query so frequency ranking improves over time.
		_ = r.store.UpsertFood(ctx, userID, match, []string{item.RawPhrase})
		_ = r.store.RecordFoodQuery(ctx, userID, match.FoodID)

		// Embedding-on-write: index the canonical name so future embedding
		// queries can match this food.
		if r.embed != nil {
			_ = r.embed.EmbedFood(ctx, userID, match.FoodID, match.Name)
		}

		return finalize(item, match)
	}

	// 4. Nothing matched: unresolved, needs clarification.
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
