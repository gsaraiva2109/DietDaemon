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
	SuggestFromIngredients(ctx context.Context, userID string, foodIDs []string) (types.MealSuggestion, error)
}

// SuggestFoodSearcher is the subset of store methods /suggest needs to turn
// ingredient names typed by the user into food IDs. Satisfied by the same
// store already wired into /food.
type SuggestFoodSearcher interface {
	SearchFoods(ctx context.Context, userID, query string) ([]types.FoodDetail, error)
}

// suggestFallback is returned whenever the engine fails or has nothing to say.
const suggestFallback = "Couldn't put together a suggestion right now. Try again in a bit."

// SuggestCommand handles /suggest -- recommend a next meal from what's left of
// today's targets and foods the user already eats, or from an on-hand
// ingredient list given as "/suggest chicken, rice, eggs".
type SuggestCommand struct {
	engine   SuggestEngine
	searcher SuggestFoodSearcher
}

// NewSuggestCommand creates a SuggestCommand.
func NewSuggestCommand(e SuggestEngine, s SuggestFoodSearcher) *SuggestCommand {
	return &SuggestCommand{engine: e, searcher: s}
}

func (c *SuggestCommand) Name() string        { return "/suggest" }
func (c *SuggestCommand) Aliases() []string   { return []string{"/eat"} }
func (c *SuggestCommand) Help() types.I18nKey { return "cmd.suggest.usage" }

func (c *SuggestCommand) Handle(ctx context.Context, msg types.InboundMessage, args string) (types.Reply, error) {
	args = strings.TrimSpace(args)

	var sug types.MealSuggestion
	var err error
	if args == "" {
		sug, err = c.engine.Suggest(ctx, msg.UserID)
	} else {
		foodIDs := c.resolveIngredients(ctx, msg.UserID, args)
		sug, err = c.engine.SuggestFromIngredients(ctx, msg.UserID, foodIDs)
	}
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

// resolveIngredients turns a comma-separated ingredient list ("chicken, rice,
// eggs") into food IDs by taking the top search match per name. Names that
// match nothing are skipped; SuggestFromIngredients handles an empty result.
func (c *SuggestCommand) resolveIngredients(ctx context.Context, userID, args string) []string {
	names := strings.Split(args, ",")
	ids := make([]string, 0, len(names))
	for _, name := range names {
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}
		foods, err := c.searcher.SearchFoods(ctx, userID, name)
		if err != nil || len(foods) == 0 {
			continue
		}
		ids = append(ids, foods[0].FoodID)
	}
	return ids
}
