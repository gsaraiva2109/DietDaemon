package scheduler

import (
	"time"

	"github.com/gsaraiva2109/dietdaemon/core/types"
)

// Macro selects which nutrient a rule watches.
type Macro string

const (
	MacroCalories Macro = "calories"
	MacroProtein  Macro = "protein"
	MacroCarbs    Macro = "carbs"
	MacroFat      Macro = "fat"
)

// Rule fires a nudge when, after a local hour, a macro is below a fraction of
// its daily target. Each rule fires at most once per user per local day
// (enforced by the NudgeStore via ID).
type Rule struct {
	ID          string  // stable identifier, used for dedupe
	AfterHour   int     // local hour (0-23) the rule becomes eligible
	Macro       Macro   // which macro to check
	MinFraction float64 // fire when consumed/target < MinFraction
	Message     string  // fmt template receiving (consumed, target)

	QuickActions []types.InlineButton // optional inline buttons; nil = none
}

// DefaultRules nudges a bulking user in the evening when protein or calories
// are lagging — the core pain point this project addresses (missed volume).
//
// Quick actions are deliberately omitted on macro rules: a "log usual
// breakfast" button would require a named template that may not exist, and a
// callback that 404s is worse than no button. Water rules in DefaultHealthRules
// get quick actions because "log 500ml water" is safe for every user.
func DefaultRules() []Rule {
	return []Rule{
		{
			ID:          "protein-evening",
			AfterHour:   20,
			Macro:       MacroProtein,
			MinFraction: 0.80,
			Message:     "Protein behind: %.0f/%.0f g. Time for a protein-heavy meal.",
		},
		{
			ID:          "calories-evening",
			AfterHour:   21,
			Macro:       MacroCalories,
			MinFraction: 0.85,
			Message:     "Calories behind: %.0f/%.0f kcal. Add a meal to hit your bulk target.",
		},
	}
}

// macroValue extracts the requested macro from a Macros set.
func macroValue(m types.Macros, which Macro) float64 {
	switch which {
	case MacroCalories:
		return m.Calories
	case MacroProtein:
		return m.Protein
	case MacroCarbs:
		return m.Carbs
	case MacroFat:
		return m.Fat
	default:
		return 0
	}
}

// ---------------------------------------------------------------------------
// Health domain nudging rules
// ---------------------------------------------------------------------------

// HealthRule evaluates a non-macro health domain condition. Each rule fires at
// most once per user per local day (enforced by the NudgeStore via ID).
type HealthRule struct {
	ID      string // stable identifier, used for dedupe
	Domain  string // "water", "workout", "sleep", "fasting"
	Message string // static message sent when triggered

	// CheckHour restricts evaluation to after this local hour (0-23). When 0 the
	// rule is evaluated on every tick (e.g. fast-ending, which is time-of-fast
	// dependent rather than time-of-day dependent).
	CheckHour int

	// MaxGapHours sets the maximum permitted gap in hours for water or sleep
	// logging before a nudge fires. 0 means unused.
	MaxGapHours int

	// MaxGapDays sets the maximum permitted gap in days for workout logging
	// before a nudge fires. 0 means unused.
	MaxGapDays int

	// MinDailyAmount is the minimum daily amount expected for the domain
	// (e.g. millilitres for water). The rule triggers when today's total is
	// below this threshold. 0 means unused / check only for existence.
	MinDailyAmount float64

	QuickActions []types.InlineButton // optional inline buttons; nil = none
}

// DefaultHealthRules returns the built-in health domain nudging rules. They
// nudge about water intake, missed workouts, sleep logging, and fasting
// windows. Nil-able on the Scheduler so they can be opted into independently
// of macro rules.
func DefaultHealthRules() []HealthRule {
	return []HealthRule{
		{
			ID:             "water-afternoon",
			Domain:         "water",
			CheckHour:      16,
			MinDailyAmount: 500,
			Message:        "\U0001f4a7 Don't forget to hydrate! Log your water intake with /water",
			QuickActions: []types.InlineButton{
				{Text: "Log 500ml water", CallbackData: "/water 500"},
			},
		},
		{
			ID:             "water-evening",
			Domain:         "water",
			CheckHour:      20,
			MinDailyAmount: 1600,
			Message:        "\U0001f4a7 Still behind on water — squeeze in a glass before bed!",
			QuickActions: []types.InlineButton{
				{Text: "Log 500ml water", CallbackData: "/water 500"},
			},
		},
		{
			ID:         "workout-reminder",
			Domain:     "workout",
			CheckHour:  10,
			MaxGapDays: 3,
			Message:    "\U0001f3cb️ No workout logged in 3 days. Time to move! Log with /workout",
		},
		{
			ID:        "sleep-reminder",
			Domain:    "sleep",
			CheckHour: 22,
			Message:   "\U0001f634 Ready for bed? Log your sleep with /sleep 23:00 07:00",
		},
		{
			ID:      "fast-ending",
			Domain:  "fasting",
			Message: "⏰ Your fasting window is almost complete! Get ready to break your fast with /fast end",
		},
	}
}

// ---------------------------------------------------------------------------
// Weekly digest
// ---------------------------------------------------------------------------

// DigestRule fires a periodic summary notification on a given weekday, once
// the local hour reaches CheckHour. Deduped via the same nudge_log mechanism
// as Rule/HealthRule, but keyed by ISO year-week instead of local date so it
// fires at most once per week.
type DigestRule struct {
	ID        string // stable identifier, used for dedupe
	CheckHour int    // local hour (0-23) the rule becomes eligible
	Weekday   time.Weekday
}

// DefaultDigestRules returns the built-in weekly digest: Sunday morning.
func DefaultDigestRules() []DigestRule {
	return []DigestRule{
		{ID: "weekly-digest", CheckHour: 9, Weekday: time.Sunday},
	}
}

// ---------------------------------------------------------------------------
// Weekly rolling budget compensation
// ---------------------------------------------------------------------------

// WeeklyBudgetRule fires a nudge when the rolling weekly effective target
// differs materially from the plain daily target, self-correcting for
// over-/under-eating earlier in the calendar week. Deduped via the same
// nudge_log mechanism as Rule/HealthRule, keyed by local date.
type WeeklyBudgetRule struct {
	ID        string // stable identifier, used for dedupe
	Macro     Macro  // which macro to adjust
	CheckHour int    // local hour (0-23) the rule becomes eligible
}

// DefaultWeeklyBudgetRules returns the built-in weekly budget rules.
// Unlike other rule kinds, these are OFF by default (opt-in per user).
func DefaultWeeklyBudgetRules() []WeeklyBudgetRule {
	return []WeeklyBudgetRule{
		{ID: "weekly-budget-calories", Macro: MacroCalories, CheckHour: 9},
		{ID: "weekly-budget-protein", Macro: MacroProtein, CheckHour: 9},
	}
}
