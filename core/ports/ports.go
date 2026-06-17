// Package ports declares the interfaces (hexagonal "ports") that decouple
// DietDaemon's core from concrete providers and infrastructure. Adapters under
// /adapters and the store under /internal/store implement these interfaces; the
// core depends only on the interfaces, never on a provider SDK or SQL driver.
//
// This package is the design boundary: changing a signature here ripples into
// every adapter, so it is owned deliberately and kept stable. Implementations
// can be filled in independently once these contracts are fixed.
package ports

import (
	"context"

	"github.com/gsaraiva2109/dietdaemon/core/types"
)

// MessagingAdapter ingests messages from a chat provider and delivers replies
// back to it. Implementations: telegram (baseline), discord, matrix.
type MessagingAdapter interface {
	// Receive streams inbound messages until ctx is cancelled. The adapter owns
	// the channel and closes it when it stops producing.
	Receive(ctx context.Context) (<-chan types.InboundMessage, error)
	// Send delivers a reply to the conversation identified by reply.ChannelMeta.
	Send(ctx context.Context, reply types.Reply) error
	// Name returns the adapter identifier (e.g. "telegram") for logs and config.
	Name() string
}

// STTProvider transcribes an audio payload to text. Optional; only used when
// ENABLE_STT is set. Returns the transcript and a detected BCP-47 locale hint
// (empty if undetermined). Implementations: whisper (local), api.
type STTProvider interface {
	Transcribe(ctx context.Context, audio []byte) (text string, locale string, err error)
}

// Parser performs Stage A: extract discrete food items + quantities from free
// text. The deterministic, embedding, and llm tiers all satisfy this interface
// so they are hot-swappable via PARSER_TIER. confidence is 0..1.
type Parser interface {
	Extract(ctx context.Context, text, locale string) (items []types.ParsedItem, confidence float64, err error)
	// Tier reports which strategy this parser implements.
	Tier() types.ParserTier
}

// NutritionSource performs Stage B: resolve a parsed item to a concrete food and
// its macros. Implementations are queried in NUTRITION_SOURCE order, with the
// local food library always tried first by the resolver. Resolve returns
// types.ErrNoMatch when nothing suitable is found so the pipeline can fall
// through to the next source. Implementations: openfoodfacts, taco, usda.
type NutritionSource interface {
	Resolve(ctx context.Context, item types.ParsedItem) (types.FoodMatch, error)
	Name() string
}

// ModelAdapter exposes embedding and completion calls to an inference backend
// (Ollama over HTTP). Optional; only used by Tier-1/Tier-2 parsers when
// PARSER_TIER > 0.
type ModelAdapter interface {
	Embed(ctx context.Context, text string) ([]float32, error)
	Complete(ctx context.Context, prompt string) (string, error)
}

// Notifier delivers a notification to a user's push channel. Implementations:
// ntfy (baseline), gotify, webhook.
type Notifier interface {
	Notify(ctx context.Context, n types.Notification) error
	Name() string
}

// Store is the persistence boundary. The SQLite implementation in
// internal/store is the only code that knows SQL. All methods are keyed by user
// to keep multi-user a later flag rather than a rewrite. Lookups that find no
// row return types.ErrNotFound (or types.ErrNoMatch for food lookups).
type Store interface {
	// Users.
	UpsertUser(ctx context.Context, u types.User) error
	GetUser(ctx context.Context, userID string) (types.User, error)

	// Meals.
	SaveMeal(ctx context.Context, m types.Meal) error
	RecentMeals(ctx context.Context, userID string, limit int) ([]types.Meal, error)

	// Personal food library: the local-first cache for Stage B.
	// LookupFood matches phrase against the user's known foods and aliases,
	// returning types.ErrNoMatch on a miss. UpsertFood stores a resolved food
	// plus the alias phrases it should match. RecordFoodQuery bumps the food's
	// query_count and last_used so frequent foods rank first.
	LookupFood(ctx context.Context, userID, phrase string) (types.FoodMatch, error)
	UpsertFood(ctx context.Context, userID string, match types.FoodMatch, aliases []string) error
	RecordFoodQuery(ctx context.Context, userID, foodID string) error

	// Targets and materialized daily rollups (localDate is "YYYY-MM-DD" in the
	// user's timezone).
	GetTargets(ctx context.Context, userID string) (types.DailyTargets, error)
	SetTargets(ctx context.Context, t types.DailyTargets) error
	GetRollup(ctx context.Context, userID, localDate string) (types.DailyRollup, error)
	UpsertRollup(ctx context.Context, r types.DailyRollup) error

	Close() error
}
