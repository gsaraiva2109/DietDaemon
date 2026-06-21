package commands

import (
	"context"
	"fmt"
	"strings"

	"github.com/gsaraiva2109/dietdaemon/core/types"
)

// FoodStore is the subset of store methods needed by /food.
type FoodStore interface {
	SearchFoods(ctx context.Context, userID, query string) ([]types.FoodDetail, error)
	FrequentFoods(ctx context.Context, userID string, limit int) ([]types.FoodDetail, error)
	GetFoodDetail(ctx context.Context, userID, foodID string) (types.FoodDetail, error)
}

// FoodCommand handles /food -- search the food library or show frequent foods.
type FoodCommand struct {
	store FoodStore
}

// NewFoodCommand creates a FoodCommand.
func NewFoodCommand(s FoodStore) *FoodCommand {
	return &FoodCommand{store: s}
}

func (c *FoodCommand) Name() string        { return "/food" }
func (c *FoodCommand) Aliases() []string   { return []string{"/search"} }
func (c *FoodCommand) Help() types.I18nKey { return "cmd.food.usage" }

func (c *FoodCommand) Handle(ctx context.Context, msg types.InboundMessage, args string) (types.Reply, error) {
	args = strings.TrimSpace(args)

	if args == "" || args == "recent" {
		foods, err := c.store.FrequentFoods(ctx, msg.UserID, 10)
		if err != nil || len(foods) == 0 {
			return types.Reply{
				Text:        "No foods found. Use /food <query> to search your personal library.\nExample: /food chicken",
				ChannelMeta: msg.ChannelMeta,
			}, nil
		}
		var b strings.Builder
		b.WriteString("Frequently used:\n\n")
		for _, f := range foods {
			fmt.Fprintf(&b, "  - %s -- %.0f kcal/100g (P%.1f/C%.1f/F%.1f) [used %dx]\n",
				f.Name, f.Per100g.Calories, f.Per100g.Protein, f.Per100g.Carbs, f.Per100g.Fat, f.QueryCount)
		}
		b.WriteString("\nUse /food <name> for details.")
		return types.Reply{Text: b.String(), ChannelMeta: msg.ChannelMeta}, nil
	}

	// Search by query.
	foods, err := c.store.SearchFoods(ctx, msg.UserID, args)
	if err != nil || len(foods) == 0 {
		return types.Reply{
			Text:        fmt.Sprintf("No foods found for %q.", args),
			ChannelMeta: msg.ChannelMeta,
		}, nil
	}

	var b strings.Builder
	fmt.Fprintf(&b, "Results for %q:\n\n", args)
	for _, f := range foods {
		fmt.Fprintf(&b, "  - %s -- %.0f kcal/100g | P%.1fg . C%.1fg . F%.1fg",
			f.Name, f.Per100g.Calories, f.Per100g.Protein, f.Per100g.Carbs, f.Per100g.Fat)
		if f.Brand != "" {
			fmt.Fprintf(&b, " [%s]", f.Brand)
		}
		b.WriteString("\n")
	}
	return types.Reply{Text: b.String(), ChannelMeta: msg.ChannelMeta}, nil
}
