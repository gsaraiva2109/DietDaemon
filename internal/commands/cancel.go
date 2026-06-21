package commands

import (
	"context"

	"github.com/gsaraiva2109/dietdaemon/core/types"
)

// PendingStore is the subset of ports.PendingStore needed by commands that
// interact with pending meal state.
type PendingStore interface {
	Get(ctx context.Context, userID string) (types.PendingMeal, error)
	Delete(ctx context.Context, userID string) error
}

// CancelCommand handles /cancel -- discard the pending meal for the user.
type CancelCommand struct {
	pending PendingStore
}

// NewCancelCommand creates a CancelCommand that reads and deletes from pending.
func NewCancelCommand(p PendingStore) *CancelCommand {
	return &CancelCommand{pending: p}
}

func (c *CancelCommand) Name() string        { return "/cancel" }
func (c *CancelCommand) Aliases() []string   { return nil }
func (c *CancelCommand) Help() types.I18nKey { return "cmd.cancel.done" }

func (c *CancelCommand) Handle(ctx context.Context, msg types.InboundMessage, args string) (types.Reply, error) {
	_, err := c.pending.Get(ctx, msg.UserID)
	if err != nil {
		// No pending meal -- nothing to cancel. This is not an error; the
		// command is idempotent from the user's perspective.
		return types.Reply{
			Text:        "Nothing pending to cancel.",
			ChannelMeta: msg.ChannelMeta,
		}, nil
	}
	if err := c.pending.Delete(ctx, msg.UserID); err != nil {
		return types.Reply{}, err
	}
	return types.Reply{
		Text:        "Discarded the pending meal.",
		ChannelMeta: msg.ChannelMeta,
	}, nil
}
