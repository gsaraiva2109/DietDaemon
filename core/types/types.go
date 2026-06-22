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
	ID              string
	AccountID       string
	Email           string
	EmailVerifiedAt *time.Time
	Status          string
	DisplayName     string
	Timezone        string // IANA tz (e.g. "America/Sao_Paulo"); empty falls back to DEFAULT_TIMEZONE
	Locale          string // BCP-47 locale preference (e.g. "en", "pt-BR"); empty = auto-detect
	CreatedAt       time.Time
	WebAuthnHandle  string // base64 stable handle for passkey operations; empty until first passkey
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
	Locale      string       // BCP-47 locale for i18n rendering
	Markup      *ReplyMarkup // optional inline keyboard / components
}

// ReplyMarkup carries platform-agnostic interactive UI elements for a reply.
// Adapters translate this to native controls (inline keyboard, components, text).
type ReplyMarkup struct {
	InlineKeyboard [][]InlineButton
}

// InlineButton is one button in an inline keyboard row.
type InlineButton struct {
	Text         string
	CallbackData string
}

// I18nKey is a translation key used with the i18n bundle to look up a template.
type I18nKey string

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

// ---------------------------------------------------------------------------
// Food library types
// ---------------------------------------------------------------------------

// FoodDetail is a full food library entry with metadata and aliases.
type FoodDetail struct {
	FoodID      string      `json:"food_id"`
	UserID      string      `json:"-"`
	Name        string      `json:"name"`
	Source      string      `json:"source"`
	Per100g     Macros      `json:"per_100g"`
	Category    string      `json:"category"`
	Brand       string      `json:"brand"`
	Barcode     string      `json:"barcode"`
	ImageURL    string      `json:"image_url"`
	ServingSize float64     `json:"serving_size"`
	ServingUnit string      `json:"serving_unit"`
	QueryCount  int         `json:"query_count"`
	LastUsed    string      `json:"last_used"`
	Aliases     []FoodAlias `json:"aliases,omitempty"`
}

// FoodAlias is one alias for a food library entry.
type FoodAlias struct {
	FoodID     string `json:"food_id"`
	Alias      string `json:"alias"`
	Normalized string `json:"normalized"`
}

// ---------------------------------------------------------------------------
// Meal templates
// ---------------------------------------------------------------------------

// MealTemplate is a reusable meal with pre-resolved items.
type MealTemplate struct {
	ID        string         `json:"id"`
	UserID    string         `json:"user_id"`
	Name      string         `json:"name"`
	Items     []ResolvedItem `json:"items"`
	CreatedAt time.Time      `json:"created_at"`
	LastUsed  time.Time      `json:"last_used"`
}

// TemplateLog records a template usage event.
type TemplateLog struct {
	ID         string    `json:"id"`
	UserID     string    `json:"user_id"`
	TemplateID string    `json:"template_id"`
	LoggedAt   time.Time `json:"logged_at"`
}

// ---------------------------------------------------------------------------
// Body tracking types
// ---------------------------------------------------------------------------

// WeightEntry is a single weight measurement.
type WeightEntry struct {
	ID        string    `json:"id"`
	UserID    string    `json:"user_id"`
	Date      string    `json:"date"`
	WeightKg  float64   `json:"weight_kg"`
	Note      string    `json:"note"`
	CreatedAt time.Time `json:"created_at"`
}

// Fast is a single intermittent-fasting window. EndAt is nil while the fast is
// still in progress; Completed is set true at end if the target was reached.
type Fast struct {
	ID          string     `json:"id"`
	UserID      string     `json:"user_id"`
	StartAt     time.Time  `json:"start_at"`
	EndAt       *time.Time `json:"end_at,omitempty"`
	TargetHours float64    `json:"target_hours"`
	Completed   bool       `json:"completed"`
	CreatedAt   time.Time  `json:"created_at"`
}

// MeasurementEntry is a single body measurement record.
type MeasurementEntry struct {
	ID           string    `json:"id"`
	UserID       string    `json:"user_id"`
	Date         string    `json:"date"`
	WaistCm      float64   `json:"waist_cm"`
	HipsCm       float64   `json:"hips_cm"`
	ChestCm      float64   `json:"chest_cm"`
	LeftArmCm    float64   `json:"left_arm_cm"`
	RightArmCm   float64   `json:"right_arm_cm"`
	LeftThighCm  float64   `json:"left_thigh_cm"`
	RightThighCm float64   `json:"right_thigh_cm"`
	Note         string    `json:"note"`
	CreatedAt    time.Time `json:"created_at"`
}

// ProgressPhoto is a progress photo record (metadata + binary data).
type ProgressPhoto struct {
	ID        string    `json:"id"`
	UserID    string    `json:"user_id"`
	Date      string    `json:"date"`
	View      string    `json:"view"` // front, side, back
	MimeType  string    `json:"mime_type"`
	Data      []byte    `json:"data,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

// WeightTrend is a single data point on the weight trend line.
type WeightTrend struct {
	Date       string  `json:"date"`
	WeightKg   float64 `json:"weight_kg"`
	RollingAvg float64 `json:"rolling_avg"`
}

// BodyCompositionSummary is a snapshot of the user's body composition progress.
type BodyCompositionSummary struct {
	CurrentWeightKg  float64      `json:"current_weight_kg"`
	StartWeightKg    float64      `json:"start_weight_kg"`
	ChangeKg         float64      `json:"change_kg"`
	TrendDirection   string       `json:"trend_direction"` // up, down, stable
	LatestTrendPoint *WeightTrend `json:"latest_trend_point,omitempty"`
}

// ---------------------------------------------------------------------------
// Goals & planning types
// ---------------------------------------------------------------------------

// UserProfile holds body metrics and goals for a user.
type UserProfile struct {
	UserID         string    `json:"user_id"`
	HeightCm       float64   `json:"height_cm"`
	BirthDate      string    `json:"birth_date"`
	Gender         string    `json:"gender"`
	ActivityLevel  string    `json:"activity_level"`
	Goal           string    `json:"goal"`
	TargetWeightKg float64   `json:"target_weight_kg"`
	WeeklyRate     float64   `json:"weekly_rate"`
	Onboarded      bool      `json:"onboarded"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

// TDEEParams is the input for the TDEE calculator.
type TDEEParams struct {
	WeightKg      float64 `json:"weight_kg"`
	HeightCm      float64 `json:"height_cm"`
	Age           int     `json:"age"`
	Gender        string  `json:"gender"`
	ActivityLevel string  `json:"activity_level"`
}

// TDEEResult is the output of the TDEE calculator.
type TDEEResult struct {
	BMR         float64 `json:"bmr"`
	TDEE        float64 `json:"tdee"`
	CutCal      float64 `json:"cut_cal"`
	MaintainCal float64 `json:"maintain_cal"`
	BulkCal     float64 `json:"bulk_cal"`
	Protein     float64 `json:"protein_g"`
	Fat         float64 `json:"fat_g"`
	Carbs       float64 `json:"carbs_g"`
}

// GoalSuggestion is a human-readable recommendation based on current data vs goals.
type GoalSuggestion struct {
	CurrentIntakeKcal float64 `json:"current_intake_kcal"`
	RecommendedKcal   float64 `json:"recommended_kcal"`
	CurrentLossKg     float64 `json:"current_loss_kg"`
	TargetLossKg      float64 `json:"target_loss_kg"`
	Message           string  `json:"message"`
}

// ---------------------------------------------------------------------------
// Auth types — sessions and API keys
// ---------------------------------------------------------------------------

// APIKey is a machine-authentication key. The raw key is returned exactly
// once on creation; only metadata is listed thereafter.
type APIKey struct {
	ID         string     `json:"id"`
	UserID     string     `json:"user_id"`
	Label      string     `json:"label"`
	CreatedAt  time.Time  `json:"created_at"`
	LastUsedAt *time.Time `json:"last_used_at"`
	RevokedAt  *time.Time `json:"revoked_at"`
}

// NewAPIKeyResponse wraps an APIKey with the one-time raw secret.
type NewAPIKeyResponse struct {
	APIKey
	Key string `json:"key"`
}

// AuditEvent is a single entry in the auth audit log.
type AuditEvent struct {
	ID        string `json:"id"`
	AccountID string `json:"account_id,omitempty"`
	UserID    string `json:"user_id,omitempty"`
	Event     string `json:"event"`
	IP        string `json:"ip,omitempty"`
	UserAgent string `json:"user_agent,omitempty"`
	Meta      string `json:"meta,omitempty"`
	CreatedAt time.Time
}

// OIDCIdentity is a linked OIDC provider identity.
type OIDCIdentity struct {
	ID        string    `json:"id"`
	UserID    string    `json:"-"`
	Provider  string    `json:"provider"`
	Subject   string    `json:"-"`
	Email     string    `json:"email"`
	LinkedAt  time.Time `json:"linked_at"`
	CreatedAt time.Time `json:"-"`
}

// Passkey is a user-facing WebAuthn credential summary returned by the API.
type Passkey struct {
	ID         string `json:"id"`
	Label      string `json:"label"`
	CreatedAt  string `json:"created_at"`
	LastUsedAt string `json:"last_used_at"` // empty if never used
}

// WebAuthnCredential is the raw stored credential data used to reconstruct a
// go-webauthn user for ceremony operations.
type WebAuthnCredential struct {
	ID             string `json:"id"` // base64url credential ID
	CredentialJSON string `json:"credential_json"`
}

// LinkingCode is a one-time code for linking a chat platform account to a
// dashboard user. Codes expire after 10 minutes.
type LinkingCode struct {
	Code      string `json:"code"`
	UserID    string `json:"user_id"`
	Platform  string `json:"platform"`
	ExpiresAt string `json:"expires_at"` // UTC datetime string "2006-01-02 15:04:05"
	UsedAt    string `json:"used_at"`    // empty if not yet used
}

// ---------------------------------------------------------------------------
// Water, workout, and sleep tracking types
// ---------------------------------------------------------------------------

// WaterLog tracks water consumption entries.
type WaterLog struct {
	ID       string `json:"id"`
	UserID   string `json:"user_id"`
	AmountML int    `json:"amount_ml"`
	LoggedAt string `json:"logged_at"`
	Note     string `json:"note,omitempty"`
}

// Workout tracks an exercise session.
type Workout struct {
	ID             string            `json:"id"`
	UserID         string            `json:"user_id"`
	Name           string            `json:"name"`
	DurationMin    int               `json:"duration_min"`
	Intensity      string            `json:"intensity"`
	CaloriesBurned *int              `json:"calories_burned,omitempty"`
	Note           string            `json:"note,omitempty"`
	LoggedAt       string            `json:"logged_at"`
	Exercises      []WorkoutExercise `json:"exercises,omitempty"`
}

// WorkoutExercise is an individual exercise within a workout.
type WorkoutExercise struct {
	ID        string   `json:"id,omitempty"`
	WorkoutID string   `json:"workout_id,omitempty"`
	Name      string   `json:"name"`
	Sets      *int     `json:"sets,omitempty"`
	Reps      *int     `json:"reps,omitempty"`
	WeightKg  *float64 `json:"weight_kg,omitempty"`
	Note      string   `json:"note,omitempty"`
}

// SleepLog tracks sleep sessions.
type SleepLog struct {
	ID            string  `json:"id"`
	UserID        string  `json:"user_id"`
	SleepAt       string  `json:"sleep_at"`
	WakeAt        *string `json:"wake_at,omitempty"`
	DurationHours float64 `json:"duration_hours,omitempty"`
	Quality       string  `json:"quality"`
	Note          string  `json:"note,omitempty"`
}

// RegistrationMode enumerates how new accounts may be created.
type RegistrationMode string

const (
	RegistrationOpen     RegistrationMode = "open"
	RegistrationInvite   RegistrationMode = "invite"
	RegistrationOIDCOnly RegistrationMode = "oidc-only"
)
