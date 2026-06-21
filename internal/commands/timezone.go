package commands

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/gsaraiva2109/dietdaemon/core/types"
)

// TimezoneCommand handles /timezone -- set the user's IANA timezone.
type TimezoneCommand struct {
	store MealStore
	now   func() time.Time
}

// NewTimezoneCommand creates a TimezoneCommand that persists via store.
// The now parameter can be overridden in tests; pass nil to use time.Now.
func NewTimezoneCommand(s MealStore) *TimezoneCommand {
	return &TimezoneCommand{store: s, now: time.Now}
}

func (c *TimezoneCommand) Name() string        { return "/timezone" }
func (c *TimezoneCommand) Aliases() []string   { return nil }
func (c *TimezoneCommand) Help() types.I18nKey { return "cmd.timezone.usage" }

func (c *TimezoneCommand) Handle(ctx context.Context, msg types.InboundMessage, args string) (types.Reply, error) {
	tz := strings.TrimSpace(args)
	if tz == "" {
		return types.Reply{
			Text:        "Usage: /timezone <IANA name> (e.g. /timezone America/Sao_Paulo)",
			ChannelMeta: msg.ChannelMeta,
		}, nil
	}
	loc, err := time.LoadLocation(tz)
	if err != nil {
		return types.Reply{
			Text:        fmt.Sprintf("%q is not a valid IANA timezone.", tz),
			ChannelMeta: msg.ChannelMeta,
		}, nil
	}
	u, err := c.store.GetUser(ctx, msg.UserID)
	if err != nil {
		u = types.User{ID: msg.UserID, CreatedAt: c.now().UTC()}
	}
	u.Timezone = loc.String()
	if err := c.store.UpsertUser(ctx, u); err != nil {
		return types.Reply{}, fmt.Errorf("upsert user timezone: %w", err)
	}
	return types.Reply{
		Text:        fmt.Sprintf("Timezone set to %s.", loc.String()),
		ChannelMeta: msg.ChannelMeta,
	}, nil
}
