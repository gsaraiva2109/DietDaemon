package commands

import (
	"context"
	"fmt"
	"html"
	"strings"

	"github.com/gsaraiva2109/dietdaemon/core/types"
	"github.com/gsaraiva2109/dietdaemon/internal/i18n"
)

// HelpCommand handles /help — lists commands or shows detail for one command.
type HelpCommand struct {
	registry *Registry
	i18n     *i18n.Bundle
}

// NewHelpCommand creates a HelpCommand that queries the registry for commands
// and resolves descriptions through the i18n bundle.
func NewHelpCommand(r *Registry, b *i18n.Bundle) *HelpCommand {
	return &HelpCommand{registry: r, i18n: b}
}

func (c *HelpCommand) Name() string        { return "/help" }
func (c *HelpCommand) Aliases() []string   { return []string{"/h", "/commands"} }
func (c *HelpCommand) Help() types.I18nKey { return "cmd.help.description" }

func (c *HelpCommand) Handle(ctx context.Context, msg types.InboundMessage, args string) (types.Reply, error) {
	locale := msg.Locale
	if locale == "" {
		locale = "en"
	}

	cmds := c.registry.List()

	// If args provided, show detail for that command.
	if args != "" {
		name := strings.ToLower(args)
		if !strings.HasPrefix(name, "/") {
			name = "/" + name
		}
		for _, cmd := range cmds {
			if cmd.Name() == name {
				aliases := strings.Join(cmd.Aliases(), ", ")
				desc := html.EscapeString(c.i18n.T(locale, cmd.Help(), nil))
				data := map[string]any{
					"Name":        name,
					"Aliases":     aliases,
					"Description": desc,
				}
				text := c.i18n.T(locale, "cmd.help.detail", data)
				return types.Reply{
					Text:        text,
					ChannelMeta: msg.ChannelMeta,
					ParseMode:   "HTML",
				}, nil
			}
		}
		return types.Reply{
			Text:        c.i18n.T(locale, "cmd.help.unknown", map[string]any{"Command": html.EscapeString(args)}),
			ChannelMeta: msg.ChannelMeta,
			ParseMode:   "HTML",
		}, nil
	}

	// List all commands — single line per command.
	var b strings.Builder
	title := c.i18n.T(locale, "cmd.help.title", nil)
	footer := c.i18n.T(locale, "cmd.help.footer", nil)

	fmt.Fprintf(&b, "<b>%s</b>\n", title)
	for _, cmd := range cmds {
		aliases := ""
		if len(cmd.Aliases()) > 0 {
			aliases = fmt.Sprintf(" <i>(%s)</i>", strings.Join(cmd.Aliases(), ", "))
		}
		desc := html.EscapeString(c.i18n.T(locale, cmd.Help(), nil))
		fmt.Fprintf(&b, "\n<b>%s</b>%s — %s", cmd.Name(), aliases, desc)
	}
	fmt.Fprintf(&b, "\n\n<i>%s</i>", footer)

	return types.Reply{
		Text:        b.String(),
		ChannelMeta: msg.ChannelMeta,
		ParseMode:   "HTML",
	}, nil
}
