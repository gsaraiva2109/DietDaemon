package commands

import (
	"context"
	"strings"

	"github.com/gsaraiva2109/dietdaemon/core/types"
)

// StartCommand handles /start — onboarding wizard.
type StartCommand struct {
	store MealStore
}

// NewStartCommand creates a StartCommand.
func NewStartCommand(s MealStore) *StartCommand {
	return &StartCommand{store: s}
}

func (c *StartCommand) Name() string        { return "/start" }
func (c *StartCommand) Aliases() []string   { return nil }
func (c *StartCommand) Help() types.I18nKey { return "cmd.start.welcome" }

func (c *StartCommand) Handle(ctx context.Context, msg types.InboundMessage, args string) (types.Reply, error) {
	// Language selection callback from inline keyboard.
	if locale, ok := strings.CutPrefix(args, "lang "); ok {
		u, err := c.store.GetUser(ctx, msg.UserID)
		if err != nil {
			u = types.User{ID: msg.UserID}
		}
		u.Locale = locale
		if err := c.store.UpsertUser(ctx, u); err != nil {
			return types.Reply{}, err
		}
		return types.Reply{
			Text:        "Language set to " + locale + "!\n\nUse /help to see what I can do.",
			ChannelMeta: msg.ChannelMeta,
		}, nil
	}

	// Check if user already onboarded.
	u, err := c.store.GetUser(ctx, msg.UserID)
	if err == nil && u.Locale != "" {
		return types.Reply{
			Text:        "You're all set! Type /help to see what I can do.",
			ChannelMeta: msg.ChannelMeta,
		}, nil
	}

	// Language selection — first onboarding step.
	reply := types.Reply{
		Text:        "Welcome to DietDaemon!\n\nI help you track nutrition through chat. Let's get you set up.\n\nFirst, choose your language / Primeiro, escolha seu idioma:",
		ChannelMeta: msg.ChannelMeta,
		Markup: &types.ReplyMarkup{
			InlineKeyboard: [][]types.InlineButton{
				{
					{Text: "English", CallbackData: "/start lang en"},
					{Text: "Portugues", CallbackData: "/start lang pt-BR"},
				},
			},
		},
	}
	return reply, nil
}
