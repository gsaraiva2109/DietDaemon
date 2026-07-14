package commands

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/gsaraiva2109/dietdaemon/core/types"
)

// LogMealEngine is the subset of pipeline.Engine the /logmeal command needs.
type LogMealEngine interface {
	ParseAndResolve(ctx context.Context, userID, text, locale string) ([]types.ResolvedItem, int, error)
	LogMealFromItems(ctx context.Context, userID string, at time.Time, rawText string, confidence float64, items []types.ResolvedItem) (types.Meal, error)
}

// LogMealCommand handles /logmeal — log a meal from a natural-language description.
type LogMealCommand struct{ engine LogMealEngine }

// NewLogMealCommand creates a LogMealCommand.
func NewLogMealCommand(e LogMealEngine) *LogMealCommand { return &LogMealCommand{engine: e} }

func (c *LogMealCommand) Name() string        { return "/logmeal" }
func (c *LogMealCommand) Aliases() []string   { return nil }
func (c *LogMealCommand) Help() types.I18nKey { return "cmd.logmeal.title" }

func (c *LogMealCommand) Handle(ctx context.Context, msg types.InboundMessage, args string) (types.Reply, error) {
	text := strings.TrimSpace(args)
	if text == "" {
		return types.Reply{Text: "Tell me what you ate, e.g. \"200g grilled chicken and a banana\"."}, nil
	}

	items, needsClarification, err := c.engine.ParseAndResolve(ctx, msg.UserID, text, msg.Locale)
	if err != nil {
		return types.Reply{}, fmt.Errorf("logmeal: parse: %w", err)
	}
	if len(items) == 0 {
		return types.Reply{Text: "Couldn't read any food in that. Try \"200g rice, 100g beans\"."}, nil
	}
	if needsClarification > 0 {
		// Don't reuse pipeline's PendingStore clarification loop here — that's
		// built for the multi-turn bot flow. A tool call just needs to report
		// what's ambiguous in plain text; the model relays it conversationally
		// and the user's next message becomes a fresh tool call with corrected
		// wording (e.g. an explicit gram amount).
		return types.Reply{Text: describeAmbiguity(items)}, nil
	}

	meal, err := c.engine.LogMealFromItems(ctx, msg.UserID, time.Now().UTC(), text, 1.0, items)
	if err != nil {
		return types.Reply{}, fmt.Errorf("logmeal: save: %w", err)
	}
	return types.Reply{Text: summaryText(meal)}, nil
}

// describeAmbiguity lists which items still need a portion/correction, in
// plain text the model can relay conversationally.
func describeAmbiguity(items []types.ResolvedItem) string {
	var b strings.Builder
	b.WriteString("A few items need clarification before I can log this:\n")
	for _, it := range items {
		if it.Match.FoodID == "" {
			fmt.Fprintf(&b, "- %s (not recognized)\n", it.Parsed.RawPhrase)
		} else if it.Parsed.NormalizedGrams <= 0 {
			fmt.Fprintf(&b, "- %s (needs portion in grams)\n", it.Match.Name)
		}
	}
	b.WriteString("\nReply with the missing details (e.g. exact grams) and I'll log it.")
	return b.String()
}

// summaryText formats a successfully-logged meal for the tool-result reply.
// Matches pipeline.Engine's existing summary() formatting style/tone
// (internal/pipeline/pipeline.go) so /logmeal and the Telegram/Discord/Matrix
// free-text path read the same to a user who uses both.
func summaryText(meal types.Meal) string {
	total := meal.Total()
	var b strings.Builder
	fmt.Fprintf(&b, "Logged %d item(s).\n", len(meal.Items))
	fmt.Fprintf(&b, "~%.0f kcal | P %.0fg · C %.0fg · F %.0fg", total.Calories, total.Protein, total.Carbs, total.Fat)
	for _, it := range meal.Items {
		// A resolved item with no explicit parsed grams only ever reaches a
		// logged meal because resolver.finalize fell back to the matched
		// food's default serving size — flag that so the user knows it was
		// assumed, not stated (see resolver.defaultServingGrams).
		if it.Parsed.NormalizedGrams <= 0 && it.Match.ServingSize > 0 {
			fmt.Fprintf(&b, "\n- %s: assumed %.0fg serving", it.Match.Name, it.Match.ServingSize)
		}
	}
	return b.String()
}
