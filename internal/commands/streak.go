package commands

import (
	"context"
	"fmt"
	"time"

	"github.com/gsaraiva2109/dietdaemon/core/types"
	"github.com/gsaraiva2109/dietdaemon/internal/adherence"
)

// StreakStore is the subset of store methods needed by /streak.
type StreakStore interface {
	GetRollups(ctx context.Context, userID, startDate, endDate string) ([]types.DailyRollup, error)
}

// StreakCommand handles /streak -- shows how many consecutive days the user
// has stayed within 90-110% of their calorie target.
type StreakCommand struct {
	store StreakStore
}

// NewStreakCommand creates a StreakCommand.
func NewStreakCommand(s StreakStore) *StreakCommand {
	return &StreakCommand{store: s}
}

func (c *StreakCommand) Name() string        { return "/streak" }
func (c *StreakCommand) Aliases() []string   { return nil }
func (c *StreakCommand) Help() types.I18nKey { return "cmd.streak.usage" }

func (c *StreakCommand) Handle(ctx context.Context, msg types.InboundMessage, args string) (types.Reply, error) {
	// Look back 180 days ending yesterday.
	end := time.Now().AddDate(0, 0, -1).Format("2006-01-02")
	start := time.Now().AddDate(0, 0, -180).Format("2006-01-02")

	rollups, err := c.store.GetRollups(ctx, msg.UserID, start, end)
	if err != nil {
		return types.Reply{}, fmt.Errorf("get rollups: %w", err)
	}

	days := adherence.Streak(rollups, 0.90, 1.10)

	return types.Reply{
		Text:        fmt.Sprintf("Current streak: %d day(s)", days),
		ChannelMeta: msg.ChannelMeta,
	}, nil
}
