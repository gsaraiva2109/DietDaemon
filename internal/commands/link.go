package commands

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/gsaraiva2109/dietdaemon/core/types"
)

// LinkCodeStore is the subset of store needed by the link command.
type LinkCodeStore interface {
	LookupLinkingCode(ctx context.Context, code string) (types.LinkingCode, error)
	ConsumeLinkingCode(ctx context.Context, code string) error
}

// LinkCommand handles /link <code> — link a chat account to a dashboard user.
type LinkCommand struct {
	store       LinkCodeStore
	mealStore   MealStore
	channelName string // e.g. "telegram" — set at construction
}

// NewLinkCommand creates a LinkCommand.
func NewLinkCommand(s LinkCodeStore, ms MealStore, channelName string) *LinkCommand {
	return &LinkCommand{store: s, mealStore: ms, channelName: channelName}
}

func (c *LinkCommand) Name() string        { return "/link" }
func (c *LinkCommand) Aliases() []string   { return nil }
func (c *LinkCommand) Help() types.I18nKey { return "cmd.link.usage" }

func (c *LinkCommand) Handle(ctx context.Context, msg types.InboundMessage, args string) (types.Reply, error) {
	code := strings.TrimSpace(args)
	if code == "" {
		return types.Reply{
			Text:        "Usage: /link <code>\nGet your linking code from the dashboard: Settings -> Link Bot.",
			ChannelMeta: msg.ChannelMeta,
		}, nil
	}

	lc, err := c.store.LookupLinkingCode(ctx, code)
	if err != nil {
		return types.Reply{
			Text:        "Invalid or expired linking code. Please generate a new one from the dashboard.",
			ChannelMeta: msg.ChannelMeta,
		}, nil
	}

	// Check expiry.
	expiresAt, parseErr := time.Parse("2006-01-02 15:04:05", lc.ExpiresAt)
	if parseErr != nil || time.Now().UTC().After(expiresAt) {
		return types.Reply{
			Text:        "This linking code has expired. Please generate a new one from the dashboard.",
			ChannelMeta: msg.ChannelMeta,
		}, nil
	}

	// Map the channel user to the dashboard user.
	if err := c.mealStore.MapChannelUser(ctx, c.channelName, msg.UserID, lc.UserID); err != nil {
		return types.Reply{}, fmt.Errorf("link: map channel: %w", err)
	}

	// Mark code as used.
	if err := c.store.ConsumeLinkingCode(ctx, code); err != nil {
		return types.Reply{}, fmt.Errorf("link: consume code: %w", err)
	}

	return types.Reply{
		Text:        "Account linked successfully!\nWelcome! Type /start to begin.",
		ChannelMeta: msg.ChannelMeta,
	}, nil
}
