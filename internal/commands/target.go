package commands

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/gsaraiva2109/dietdaemon/core/types"
)

// MealStore is the subset of the store needed by commands that read and write
// user and target data. Defined here to keep the command package free of a
// dependency on ports.Store (whose surface is much larger).
type MealStore interface {
	UpsertUser(ctx context.Context, u types.User) error
	GetUser(ctx context.Context, userID string) (types.User, error)
	SaveMeal(ctx context.Context, m types.Meal) error
	GetTargets(ctx context.Context, userID string) (types.DailyTargets, error)
	SetTargets(ctx context.Context, t types.DailyTargets) error
	GetRollup(ctx context.Context, userID, localDate string) (types.DailyRollup, error)
	UpsertRollup(ctx context.Context, r types.DailyRollup) error
	GetUserIDByChannel(ctx context.Context, channel, channelUserID string) (string, error)
	MapChannelUser(ctx context.Context, channel, channelUserID, userID string) error
}

// TargetCommand handles /target -- set daily macro goals.
type TargetCommand struct {
	store MealStore
}

// NewTargetCommand creates a TargetCommand that persists targets through store.
func NewTargetCommand(s MealStore) *TargetCommand {
	return &TargetCommand{store: s}
}

func (c *TargetCommand) Name() string        { return "/target" }
func (c *TargetCommand) Aliases() []string   { return nil }
func (c *TargetCommand) Help() types.I18nKey { return "cmd.target.usage" }

func (c *TargetCommand) Handle(ctx context.Context, msg types.InboundMessage, args string) (types.Reply, error) {
	if args == "" {
		return types.Reply{
			Text:        "Usage: /target kcal=3000 protein=180 carbs=350 fat=90",
			ChannelMeta: msg.ChannelMeta,
		}, nil
	}
	macros, ok := parseTargetArgs(args)
	if !ok {
		return types.Reply{
			Text:        "Usage: /target kcal=3000 protein=180 carbs=350 fat=90",
			ChannelMeta: msg.ChannelMeta,
		}, nil
	}
	if err := c.store.SetTargets(ctx, types.DailyTargets{UserID: msg.UserID, Targets: macros}); err != nil {
		return types.Reply{}, fmt.Errorf("set targets: %w", err)
	}
	text := fmt.Sprintf("Targets set: %.0f kcal | P %.0fg . C %.0fg . F %.0fg",
		macros.Calories, macros.Protein, macros.Carbs, macros.Fat)
	return types.Reply{Text: text, ChannelMeta: msg.ChannelMeta}, nil
}

// parseTargetArgs reads "key=value" pairs into a Macros. ok is false when no
// recognized key was provided.
func parseTargetArgs(args string) (types.Macros, bool) {
	var m types.Macros
	found := false
	for _, f := range strings.Fields(args) {
		k, v, hasEq := strings.Cut(f, "=")
		if !hasEq {
			continue
		}
		val, err := strconv.ParseFloat(v, 64)
		if err != nil {
			continue
		}
		switch strings.ToLower(k) {
		case "kcal", "calories", "cal":
			m.Calories, found = val, true
		case "protein", "p":
			m.Protein, found = val, true
		case "carbs", "c":
			m.Carbs, found = val, true
		case "fat", "f":
			m.Fat, found = val, true
		case "fiber":
			m.Fiber, found = val, true
		}
	}
	return m, found
}
