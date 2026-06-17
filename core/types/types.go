// Package types defines DietDaemon's canonical domain types. Every adapter
// translates to and from these types; the core never speaks a provider's
// native format. Keeping this package free of provider and infrastructure
// imports is what lets messaging, parsing, nutrition, and notification
// backends be swapped behind interfaces.
package types

import "time"

// MessageKind enumerates the payload kinds an inbound message may carry.
type MessageKind string

const (
	MessageText  MessageKind = "text"
	MessageAudio MessageKind = "audio"
	MessageImage MessageKind = "image"
)

// ParserTier identifies which parsing strategy produced a result. It doubles
// as the PARSER_TIER config value.
type ParserTier int

const (
	TierDeterministic ParserTier = 0 // no model: grammar + fuzzy match
	TierEmbedding     ParserTier = 1 // embedding nearest-neighbor matching
	TierLLM           ParserTier = 2 // generative LLM extraction
)

// NotificationPriority controls how loudly a notifier surfaces a message.
type NotificationPriority int

const (
	PriorityLow NotificationPriority = iota
	PriorityDefault
	PriorityHigh
)

// User is a single tracked person. Multi-user is gated behind a feature flag,
// but the schema and core are keyed by user from day one so enabling it later
// is not a rewrite.
type User struct {
	ID        string
	Timezone  string // IANA tz (e.g. "America/Sao_Paulo"); empty falls back to DEFAULT_TIMEZONE
	CreatedAt time.Time
}

// InboundMessage is the canonical, channel-agnostic representation of a message
// received from any MessagingAdapter. Exactly one payload field is populated
// according to Kind; after STT, an audio message gains its Text.
type InboundMessage struct {
	UserID      string
	At          time.Time // arrival time, UTC
	Kind        MessageKind
	Text        string            // set when Kind == MessageText, or filled by STT
	Audio       []byte            // set when Kind == MessageAudio
	Image       []byte            // set when Kind == MessageImage
	Locale      string            // BCP-47 hint (e.g. "pt-BR"); may be empty
	ChannelMeta map[string]string // opaque adapter routing info (chat id, message id, …)
}

// Reply is an outbound message the core asks a MessagingAdapter to deliver. The
// adapter uses ChannelMeta (echoed from the originating InboundMessage) to route
// it back to the right conversation.
type Reply struct {
	UserID      string
	Text        string
	ChannelMeta map[string]string
}

// ParsedItem is one food item extracted from a message in Stage A, before macros
// are resolved. Quantity/Unit are the raw extracted measure; the pipeline fills
// NormalizedGrams via unit normalization (0 when the unit is unknown).
type ParsedItem struct {
	RawPhrase       string  // e.g. "frango grelhado"
	Quantity        float64 // e.g. 200
	Unit            string  // raw unit token, e.g. "g", "colher", "cup"
	NormalizedGrams float64 // canonical grams after normalization (0 if unknown)
	Locale          string
}

// Macros holds nutrition values for a concrete portion of food (absolute, not
// per-100g).
type Macros struct {
	Calories float64 // kcal
	Protein  float64 // grams
	Carbs    float64 // grams
	Fat      float64 // grams
	Fiber    float64 // grams (optional; 0 if unknown)
}

// Add returns the element-wise sum of two macro sets. Used to aggregate items
// into a meal and meals into a daily rollup.
func (m Macros) Add(o Macros) Macros {
	return Macros{
		Calories: m.Calories + o.Calories,
		Protein:  m.Protein + o.Protein,
		Carbs:    m.Carbs + o.Carbs,
		Fat:      m.Fat + o.Fat,
		Fiber:    m.Fiber + o.Fiber,
	}
}

// Scale returns the macros multiplied by factor. Used to scale a per-100g
// FoodMatch down to the actual grams consumed (factor = grams/100).
func (m Macros) Scale(factor float64) Macros {
	return Macros{
		Calories: m.Calories * factor,
		Protein:  m.Protein * factor,
		Carbs:    m.Carbs * factor,
		Fat:      m.Fat * factor,
		Fiber:    m.Fiber * factor,
	}
}

// FoodMatch is a food entry resolved from a NutritionSource or the local food
// library. Per100g holds macros per 100 grams; the resolver scales them to the
// portion actually eaten.
type FoodMatch struct {
	FoodID     string  // stable id within Source
	Name       string  // canonical display name
	Source     string  // "food_library", "openfoodfacts", "taco", "usda", …
	Per100g    Macros  // macros per 100 grams
	MatchScore float64 // 0..1 confidence of the name match
}

// ResolvedItem couples an extracted item with the food it resolved to and the
// macros for the actual portion consumed.
type ResolvedItem struct {
	Parsed ParsedItem
	Match  FoodMatch
	Macros Macros // Match.Per100g scaled to Parsed.NormalizedGrams
}

// Meal is a fully processed logging event: the raw text, what was extracted and
// resolved, and provenance for auditing and later correction.
type Meal struct {
	ID         string
	UserID     string
	At         time.Time // event time, UTC
	RawText    string
	Items      []ResolvedItem
	Confidence float64 // overall parse confidence, 0..1
	ParserTier ParserTier
	CreatedAt  time.Time
}

// Total sums the macros across every resolved item in the meal.
func (m Meal) Total() Macros {
	var sum Macros
	for _, it := range m.Items {
		sum = sum.Add(it.Macros)
	}
	return sum
}

// Notification is a push message destined for a user's notifier channel.
type Notification struct {
	UserID   string
	Title    string
	Body     string
	Priority NotificationPriority
}

// PendingMeal is the short-lived conversational state of a meal that could not
// be fully resolved on first parse. The items that resolved cleanly are held in
// Resolved; the ones the user must clarify (no food match, or a count-based
// portion with unknown grams) are held in Pending, in the order they will be
// asked. The meal is only persisted once Pending is empty, so DietDaemon never
// silently logs a guessed macro. Pending is keyed by user; a newer pending meal
// replaces an older one.
type PendingMeal struct {
	UserID      string
	At          time.Time // event time, UTC
	RawText     string    // the original message text, for the audit trail
	Confidence  float64
	ParserTier  ParserTier
	ChannelMeta map[string]string // echoed so the follow-up routes to the same chat
	Resolved    []ResolvedItem    // already-good items, committed at finalize
	Pending     []ResolvedItem    // open questions; kind derived from Match.FoodID
	CreatedAt   time.Time         // for short-lived expiry
}

// DailyTargets holds a user's daily macro goals.
type DailyTargets struct {
	UserID  string
	Targets Macros
}

// DailyRollup is the materialized sum of a user's macros for one local calendar
// day, alongside the targets in effect, used for dashboard display and nudges.
type DailyRollup struct {
	UserID   string
	Date     string // local date "YYYY-MM-DD" in the user's timezone
	Consumed Macros
	Targets  Macros
}
