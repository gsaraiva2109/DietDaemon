package scheduler

import "github.com/gsaraiva2109/dietdaemon/core/types"

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
	AfterHour   int      // local hour (0-23) the rule becomes eligible
	Macro       Macro    // which macro to check
	MinFraction float64  // fire when consumed/target < MinFraction
	Message     string   // fmt template receiving (consumed, target)
}

// DefaultRules nudges a bulking user in the evening when protein or calories
// are lagging — the core pain point this project addresses (missed volume).
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
