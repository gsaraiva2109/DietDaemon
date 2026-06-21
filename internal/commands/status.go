package commands

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/gsaraiva2109/dietdaemon/core/types"
)

// StatusStore is the subset of store methods needed by /status.
type StatusStore interface {
	GetTargets(ctx context.Context, userID string) (types.DailyTargets, error)
	GetRollup(ctx context.Context, userID, localDate string) (types.DailyRollup, error)
	RecentMeals(ctx context.Context, userID string, limit int) ([]types.Meal, error)
	GetUser(ctx context.Context, userID string) (types.User, error)
}

// StatusCommand handles /status -- shows today's macro progress vs targets.
type StatusCommand struct {
	store StatusStore
	loc   *time.Location
}

// NewStatusCommand creates a StatusCommand. The loc parameter is the fallback
// timezone used when the user has not set their own.
func NewStatusCommand(s StatusStore, loc *time.Location) *StatusCommand {
	return &StatusCommand{store: s, loc: loc}
}

func (c *StatusCommand) Name() string        { return "/status" }
func (c *StatusCommand) Aliases() []string   { return []string{"/summary"} }
func (c *StatusCommand) Help() types.I18nKey { return "cmd.status.title" }

func (c *StatusCommand) Handle(ctx context.Context, msg types.InboundMessage, args string) (types.Reply, error) {
	// Resolve user's timezone for today's date.
	loc := c.loc
	if u, err := c.store.GetUser(ctx, msg.UserID); err == nil && u.Timezone != "" {
		if l, err := time.LoadLocation(u.Timezone); err == nil {
			loc = l
		}
	}
	today := time.Now().In(loc).Format("2006-01-02")

	// Get targets.
	targets, err := c.store.GetTargets(ctx, msg.UserID)
	if err != nil {
		return types.Reply{
			Text:        "No targets set. Use /target to set your daily goals.\nExample: /target kcal=2000 protein=180 carbs=200 fat=60",
			ChannelMeta: msg.ChannelMeta,
		}, nil
	}

	// Get today's rollup.
	rollup, err := c.store.GetRollup(ctx, msg.UserID, today)
	if err != nil {
		rollup = types.DailyRollup{Consumed: types.Macros{}, Targets: targets.Targets}
	}

	// Get recent meals.
	meals, _ := c.store.RecentMeals(ctx, msg.UserID, 5)

	// Format: consumed / target for each macro.
	t := targets.Targets
	con := rollup.Consumed
	calPct := pct(con.Calories, t.Calories)
	proteinPct := pct(con.Protein, t.Protein)
	carbsPct := pct(con.Carbs, t.Carbs)
	fatPct := pct(con.Fat, t.Fat)

	var b strings.Builder
	b.WriteString("Today's Summary\n\n")
	fmt.Fprintf(&b, "Calories: %.0f / %.0f kcal (%.0f%%)\n", con.Calories, t.Calories, calPct)
	fmt.Fprintf(&b, "Protein:  %.0f / %.0f g (%.0f%%)\n", con.Protein, t.Protein, proteinPct)
	fmt.Fprintf(&b, "Carbs:    %.0f / %.0f g (%.0f%%)\n", con.Carbs, t.Carbs, carbsPct)
	fmt.Fprintf(&b, "Fat:      %.0f / %.0f g (%.0f%%)\n", con.Fat, t.Fat, fatPct)

	if len(meals) > 0 {
		b.WriteString("\nRecent meals:\n")
		for _, meal := range meals {
			total := meal.Total()
			fmt.Fprintf(&b, "  - %.0f kcal -- %s\n", total.Calories, meal.RawText)
		}
	} else if calPct < 1 {
		b.WriteString("\nNo meals logged today. Send me what you ate!")
	}

	return types.Reply{Text: b.String(), ChannelMeta: msg.ChannelMeta}, nil
}

// pct returns consumed as a percentage of target. Returns 0 when target is 0
// to avoid division by zero.
func pct(consumed, target float64) float64 {
	if target == 0 {
		return 0
	}
	return consumed / target * 100
}
