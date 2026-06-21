package commands

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/gsaraiva2109/dietdaemon/core/types"
)

// WeightStore is the subset of store methods needed by /weight.
type WeightStore interface {
	LogWeight(ctx context.Context, entry types.WeightEntry) error
	WeightTrend(ctx context.Context, userID string, days int) ([]types.WeightTrend, error)
	ListWeight(ctx context.Context, userID string, limit int) ([]types.WeightEntry, error)
}

// WeightCommand handles /weight -- log body weight or show trend.
type WeightCommand struct {
	store WeightStore
}

// NewWeightCommand creates a WeightCommand.
func NewWeightCommand(s WeightStore) *WeightCommand {
	return &WeightCommand{store: s}
}

func (c *WeightCommand) Name() string        { return "/weight" }
func (c *WeightCommand) Aliases() []string   { return nil }
func (c *WeightCommand) Help() types.I18nKey { return "cmd.weight.usage" }

func (c *WeightCommand) Handle(ctx context.Context, msg types.InboundMessage, args string) (types.Reply, error) {
	args = strings.TrimSpace(args)

	if args == "" || args == "trend" {
		// Show weight trend.
		trend, err := c.store.WeightTrend(ctx, msg.UserID, 7)
		if err != nil || len(trend) == 0 {
			return types.Reply{
				Text:        "No weight data yet. Log your first weight with /weight <kg>\nExample: /weight 80.5",
				ChannelMeta: msg.ChannelMeta,
			}, nil
		}
		var b strings.Builder
		b.WriteString("Weight Trend (7 days)\n\n")
		for _, t := range trend {
			marker := ""
			if t.RollingAvg > 0 {
				marker = fmt.Sprintf(" (avg: %.1f)", t.RollingAvg)
			}
			fmt.Fprintf(&b, "%s: %.1f kg%s\n", t.Date, t.WeightKg, marker)
		}
		return types.Reply{Text: b.String(), ChannelMeta: msg.ChannelMeta}, nil
	}

	// Parse weight value.
	kg, err := strconv.ParseFloat(args, 64)
	if err != nil || kg <= 0 || kg > 500 {
		return types.Reply{
			Text:        "Usage: /weight <kg>\nExample: /weight 80.5\nUse /weight trend to see your progress.",
			ChannelMeta: msg.ChannelMeta,
		}, nil
	}

	today := time.Now().Format("2006-01-02")
	entry := types.WeightEntry{
		ID:        randomID(),
		UserID:    msg.UserID,
		Date:      today,
		WeightKg:  kg,
		CreatedAt: time.Now().UTC(),
	}
	if err := c.store.LogWeight(ctx, entry); err != nil {
		return types.Reply{}, fmt.Errorf("log weight: %w", err)
	}

	return types.Reply{
		Text:        fmt.Sprintf("Weight logged: %.1f kg on %s", kg, today),
		ChannelMeta: msg.ChannelMeta,
	}, nil
}
