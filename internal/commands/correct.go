package commands

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/gsaraiva2109/dietdaemon/core/types"
)

// gramsRe matches a grams token like "150g" or "150.5g".
var gramsRe = regexp.MustCompile(`^\d+(\.\d+)?g$`)

// CorrectStore is the subset of store methods needed by /correct.
type CorrectStore interface {
	RecentMeals(ctx context.Context, userID string, limit int) ([]types.Meal, error)
	CorrectMealItem(ctx context.Context, userID, mealID string, itemIndex int, corrected types.ResolvedItem) error
}

// CorrectResolver is the subset of resolver methods needed by /correct.
type CorrectResolver interface {
	Resolve(ctx context.Context, userID string, items []types.ParsedItem) ([]types.ResolvedItem, int)
}

// CorrectCommand handles /correct -- fix one item on the user's most recent meal.
type CorrectCommand struct {
	store    CorrectStore
	resolver CorrectResolver
}

// NewCorrectCommand creates a CorrectCommand.
func NewCorrectCommand(s CorrectStore, r CorrectResolver) *CorrectCommand {
	return &CorrectCommand{store: s, resolver: r}
}

func (c *CorrectCommand) Name() string        { return "/correct" }
func (c *CorrectCommand) Aliases() []string   { return nil }
func (c *CorrectCommand) Help() types.I18nKey { return "cmd.correct.usage" }

func (c *CorrectCommand) Handle(ctx context.Context, msg types.InboundMessage, args string) (types.Reply, error) {
	args = strings.TrimSpace(args)
	parts := strings.SplitN(args, " ", 3)
	if len(parts) < 3 {
		return types.Reply{
			Text:        "Usage: /correct <itemIndex> <grams>g <phrase>\nExample: /correct 0 150g grilled chicken breast",
			ChannelMeta: msg.ChannelMeta,
		}, nil
	}

	itemIndex, err := strconv.Atoi(parts[0])
	if err != nil || itemIndex < 0 {
		return types.Reply{
			Text:        "Item index must be a non-negative integer. Usage: /correct <itemIndex> <grams>g <phrase>",
			ChannelMeta: msg.ChannelMeta,
		}, nil
	}

	if !gramsRe.MatchString(parts[1]) {
		return types.Reply{
			Text:        "Grams must look like \"150g\". Usage: /correct <itemIndex> <grams>g <phrase>",
			ChannelMeta: msg.ChannelMeta,
		}, nil
	}
	grams, err := strconv.ParseFloat(strings.TrimSuffix(parts[1], "g"), 64)
	if err != nil {
		return types.Reply{
			Text:        "Grams must look like \"150g\". Usage: /correct <itemIndex> <grams>g <phrase>",
			ChannelMeta: msg.ChannelMeta,
		}, nil
	}

	phrase := strings.TrimSpace(parts[2])
	if phrase == "" {
		return types.Reply{
			Text:        "Missing corrected food phrase. Usage: /correct <itemIndex> <grams>g <phrase>",
			ChannelMeta: msg.ChannelMeta,
		}, nil
	}

	// ponytail: only the single most recent meal is addressable; no meal
	// picker. Extend with an explicit meal ID/date argument if users need to
	// correct an older meal via chat.
	meals, err := c.store.RecentMeals(ctx, msg.UserID, 1)
	if err != nil {
		return types.Reply{}, fmt.Errorf("recent meals: %w", err)
	}
	if len(meals) == 0 {
		return types.Reply{
			Text:        "No recent meal to correct. Log a meal first.",
			ChannelMeta: msg.ChannelMeta,
		}, nil
	}
	meal := meals[0]

	// Resolve always returns exactly one item per input item.
	resolved, _ := c.resolver.Resolve(ctx, msg.UserID, []types.ParsedItem{
		{RawPhrase: phrase, NormalizedGrams: grams},
	})
	item := resolved[0]
	if item.Match.FoodID == "" {
		return types.Reply{
			Text:        fmt.Sprintf("Could not find a food match for %q.", phrase),
			ChannelMeta: msg.ChannelMeta,
		}, nil
	}

	if err := c.store.CorrectMealItem(ctx, msg.UserID, meal.ID, itemIndex, item); err != nil {
		if err == types.ErrNotFound {
			return types.Reply{
				Text:        "Could not find that item on your most recent meal.",
				ChannelMeta: msg.ChannelMeta,
			}, nil
		}
		return types.Reply{}, fmt.Errorf("correct meal item: %w", err)
	}

	m := item.Macros
	return types.Reply{
		Text: fmt.Sprintf("Corrected item %d to \"%s\": %.0f kcal | P %.0fg . C %.0fg . F %.0fg",
			itemIndex, item.Match.Name, m.Calories, m.Protein, m.Carbs, m.Fat),
		ChannelMeta: msg.ChannelMeta,
	}, nil
}
