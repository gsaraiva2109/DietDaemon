package commands

import (
	"context"
	"fmt"
	"strings"

	"github.com/gsaraiva2109/dietdaemon/core/types"
)

// SuggestEngine is the subset of internal/suggest.Engine needed by /suggest.
type SuggestEngine interface {
	Suggest(ctx context.Context, userID string) (types.MealSuggestion, error)
}

// suggestFallback is returned whenever the engine fails or has nothing to say.
const suggestFallback = "Couldn't put together a suggestion right now. Try again in a bit."

// SuggestCommand handles /suggest -- recommend a next meal from what's left of
// today's targets and foods the user already eats.
type SuggestCommand struct {
	engine SuggestEngine
}

// NewSuggestCommand creates a SuggestCommand.
func NewSuggestCommand(e SuggestEngine) *SuggestCommand {
	return &SuggestCommand{engine: e}
}

func (c *SuggestCommand) Name() string        { return "/suggest" }
func (c *SuggestCommand) Aliases() []string   { return []string{"/eat"} }
func (c *SuggestCommand) Help() types.I18nKey { return "cmd.suggest.usage" }

func (c *SuggestCommand) Handle(ctx context.Context, msg types.InboundMessage, args string) (types.Reply, error) {
	sug, err := c.engine.Suggest(ctx, msg.UserID)
	if err != nil || sug.Message == "" {
		return types.Reply{Text: suggestFallback, ChannelMeta: msg.ChannelMeta}, nil
	}

	var b strings.Builder
	b.WriteString(sug.Message)
	b.WriteString("\n\n")
	fmt.Fprintf(&b, "Left today: %.0f kcal · %.0fg protein · %.0fg carbs · %.0fg fat\n",
		sug.Remaining.Calories, sug.Remaining.Protein, sug.Remaining.Carbs, sug.Remaining.Fat)

	if len(sug.Candidates) > 0 {
		b.WriteString("\n")
		top := sug.Candidates[0]
		b.WriteString("Try:\n")
		for _, item := range top.Items {
			fmt.Fprintf(&b, "  - %s -- %.0fg\n", item.Name, item.Grams)
		}
		fmt.Fprintf(&b, "  %.0f kcal | P%.0fg · C%.0fg · F%.0fg\n",
			top.Macros.Calories, top.Macros.Protein, top.Macros.Carbs, top.Macros.Fat)

		if len(sug.Candidates) > 1 {
			b.WriteString("\nOther options:\n")
			for _, combo := range sug.Candidates[1:] {
				names := make([]string, 0, len(combo.Items))
				for _, item := range combo.Items {
					names = append(names, fmt.Sprintf("%s (%.0fg)", item.Name, item.Grams))
				}
				fmt.Fprintf(&b, "  - %s -- %.0f kcal\n", strings.Join(names, ", "), combo.Macros.Calories)
			}
		}
	}

	return types.Reply{Text: b.String(), ChannelMeta: msg.ChannelMeta}, nil
}
