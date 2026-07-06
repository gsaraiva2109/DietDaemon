package commands

import (
	"context"
	"fmt"
	"strings"

	"github.com/gsaraiva2109/dietdaemon/core/types"
)

// NudgeStore is the read/update side for sent nudge rows.
type NudgeStore interface {
	GetSentNudge(ctx context.Context, id string) (types.SentNudge, error)
	UpdateSentNudgeStatus(ctx context.Context, id, status string) error
}

// NudgeCommand handles /nudge undo <id>.
type NudgeCommand struct {
	store NudgeStore
}

// NewNudgeCommand creates a NudgeCommand.
func NewNudgeCommand(s NudgeStore) *NudgeCommand {
	return &NudgeCommand{store: s}
}

func (c *NudgeCommand) Name() string        { return "/nudge" }
func (c *NudgeCommand) Aliases() []string   { return nil }
func (c *NudgeCommand) Help() types.I18nKey { return "" }

func (c *NudgeCommand) Handle(ctx context.Context, msg types.InboundMessage, args string) (types.Reply, error) {
	parts := strings.Fields(args)
	if len(parts) < 2 || parts[0] != "undo" {
		return types.Reply{}, nil
	}
	id := parts[1]

	sn, err := c.store.GetSentNudge(ctx, id)
	if err != nil {
		return types.Reply{
			Text:        fmt.Sprintf("Nudge %s not found.", id),
			ChannelMeta: msg.ChannelMeta,
		}, nil
	}
	if sn.UserID != msg.UserID {
		return types.Reply{
			Text:        fmt.Sprintf("Nudge %s is not yours.", id),
			ChannelMeta: msg.ChannelMeta,
		}, nil
	}
	if sn.Status != "sent" {
		return types.Reply{
			Text:        fmt.Sprintf("Nudge %s already handled.", id),
			ChannelMeta: msg.ChannelMeta,
		}, nil
	}

	if err := c.store.UpdateSentNudgeStatus(ctx, id, "dismissed"); err != nil {
		return types.Reply{}, fmt.Errorf("update nudge status: %w", err)
	}

	return types.Reply{
		Text:        fmt.Sprintf("Undone nudge %s.", id),
		ChannelMeta: msg.ChannelMeta,
	}, nil
}
