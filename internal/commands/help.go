package commands

import (
	"context"
	"fmt"
	"strings"

	"github.com/gsaraiva2109/dietdaemon/core/types"
)

// HelpCommand handles /help — lists commands or shows detail for one command.
type HelpCommand struct {
	registry *Registry
}

// NewHelpCommand creates a HelpCommand that queries the registry for commands.
func NewHelpCommand(r *Registry) *HelpCommand {
	return &HelpCommand{registry: r}
}

func (c *HelpCommand) Name() string        { return "/help" }
func (c *HelpCommand) Aliases() []string   { return []string{"/h", "/commands"} }
func (c *HelpCommand) Help() types.I18nKey { return "cmd.help.description" }

func (c *HelpCommand) Handle(ctx context.Context, msg types.InboundMessage, args string) (types.Reply, error) {
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
				text := fmt.Sprintf("%s — %s", name, cmd.Help())
				if aliases != "" {
					text = fmt.Sprintf("%s\nAliases: %s", name, aliases)
				}
				return types.Reply{
					Text:        text,
					ChannelMeta: msg.ChannelMeta,
				}, nil
			}
		}
		return types.Reply{
			Text:        fmt.Sprintf("Unknown command: %s", args),
			ChannelMeta: msg.ChannelMeta,
		}, nil
	}

	// List all commands.
	var b strings.Builder
	b.WriteString("Commands\n\n")
	for _, cmd := range cmds {
		aliases := ""
		if len(cmd.Aliases()) > 0 {
			aliases = fmt.Sprintf(" (%s)", strings.Join(cmd.Aliases(), ", "))
		}
		fmt.Fprintf(&b, "%s%s — %s\n", cmd.Name(), aliases, cmd.Help())
	}
	b.WriteString("\nType /help <command> for details.")
	return types.Reply{Text: b.String(), ChannelMeta: msg.ChannelMeta}, nil
}
