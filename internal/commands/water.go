package commands

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/gsaraiva2109/dietdaemon/core/types"
)

// WaterStore is the subset of store methods needed by /water.
type WaterStore interface {
	LogWater(ctx context.Context, w types.WaterLog) error
	GetWaterToday(ctx context.Context, userID, localDate string) ([]types.WaterLog, int, error)
}

// WaterCommand handles /water -- log water intake or show today's total.
type WaterCommand struct {
	store WaterStore
}

// NewWaterCommand creates a WaterCommand.
func NewWaterCommand(s WaterStore) *WaterCommand {
	return &WaterCommand{store: s}
}

func (c *WaterCommand) Name() string        { return "/water" }
func (c *WaterCommand) Aliases() []string   { return nil }
func (c *WaterCommand) Help() types.I18nKey { return "cmd.water.usage" }

func (c *WaterCommand) Handle(ctx context.Context, msg types.InboundMessage, args string) (types.Reply, error) {
	args = strings.TrimSpace(args)

	if args == "" || args == "today" {
		today := time.Now().Format("2006-01-02")
		logs, total, err := c.store.GetWaterToday(ctx, msg.UserID, today)
		if err != nil || len(logs) == 0 {
			return types.Reply{
				Text:        "No water logged today. Drink up! 💧",
				ChannelMeta: msg.ChannelMeta,
			}, nil
		}
		var b strings.Builder
		goalMl := 2000
		fmt.Fprintf(&b, "Today: %d / %d ml\n\n", total, goalMl)
		for _, l := range logs {
			fmt.Fprintf(&b, "  - %d ml", l.AmountML)
			if l.Note != "" {
				fmt.Fprintf(&b, " — %s", l.Note)
			}
			fmt.Fprintf(&b, "\n")
		}
		return types.Reply{Text: b.String(), ChannelMeta: msg.ChannelMeta}, nil
	}

	// Parse amount_ml [note].
	parts := strings.Fields(args)
	amount, err := strconv.Atoi(parts[0])
	if err != nil || amount <= 0 || amount > 10000 {
		return types.Reply{
			Text:        "Usage: /water <ml>\nExample: /water 500",
			ChannelMeta: msg.ChannelMeta,
		}, nil
	}

	note := ""
	if len(parts) > 1 {
		note = strings.Join(parts[1:], " ")
	}

	entry := types.WaterLog{
		ID:       randomID(),
		UserID:   msg.UserID,
		AmountML: amount,
		LoggedAt: time.Now().UTC().Format("2006-01-02 15:04:05"),
		Note:     note,
	}
	if err := c.store.LogWater(ctx, entry); err != nil {
		return types.Reply{}, fmt.Errorf("log water: %w", err)
	}

	return types.Reply{
		Text:        fmt.Sprintf("Water logged: %d ml", amount),
		ChannelMeta: msg.ChannelMeta,
	}, nil
}
