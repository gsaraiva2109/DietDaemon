package commands

import (
	"context"
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/gsaraiva2109/dietdaemon/core/types"
)

// FastStore is the subset of store methods needed by /fast.
type FastStore interface {
	StartFast(ctx context.Context, f types.Fast) error
	GetActiveFast(ctx context.Context, userID string) (types.Fast, error)
	EndFast(ctx context.Context, userID, fastID string, endAt time.Time, completed bool) (types.Fast, error)
	ListFasts(ctx context.Context, userID string, limit int) ([]types.Fast, error)
}

// FastCommand handles /fast -- manage intermittent-fasting windows.
type FastCommand struct {
	store FastStore
}

// NewFastCommand creates a FastCommand.
func NewFastCommand(s FastStore) *FastCommand {
	return &FastCommand{store: s}
}

func (c *FastCommand) Name() string        { return "/fast" }
func (c *FastCommand) Aliases() []string   { return nil }
func (c *FastCommand) Help() types.I18nKey { return "cmd.fast.usage" }

func (c *FastCommand) Handle(ctx context.Context, msg types.InboundMessage, args string) (types.Reply, error) {
	args = strings.TrimSpace(args)

	if args == "" {
		// Check for active fast; show status if one exists, otherwise show help.
		active, err := c.store.GetActiveFast(ctx, msg.UserID)
		if err == nil && active.StartAt != (time.Time{}) {
			return c.showStatus(ctx, msg, active)
		}
		return types.Reply{
			Text:        "Usage: /fast start — begin fasting window\n/fast end — end current fast\n/fast status — check current fast\n/fast history — view recent fasts",
			ChannelMeta: msg.ChannelMeta,
		}, nil
	}

	parts := strings.Fields(args)
	sub := strings.ToLower(parts[0])

	switch sub {
	case "start":
		return c.handleStart(ctx, msg, parts[1:])
	case "end":
		return c.handleEnd(ctx, msg)
	case "status":
		return c.handleStatus(ctx, msg)
	case "history":
		return c.handleHistory(ctx, msg)
	default:
		return types.Reply{
			Text:        "Usage: /fast start — begin fasting window\n/fast end — end current fast\n/fast status — check current fast\n/fast history — view recent fasts",
			ChannelMeta: msg.ChannelMeta,
		}, nil
	}
}

func (c *FastCommand) handleStart(ctx context.Context, msg types.InboundMessage, args []string) (types.Reply, error) {
	// Check no active fast already.
	if _, err := c.store.GetActiveFast(ctx, msg.UserID); err == nil {
		return types.Reply{
			Text:        "A fast is already in progress. Use /fast end or /fast status.",
			ChannelMeta: msg.ChannelMeta,
		}, nil
	}

	// Determine target hours.
	targetHours := 16.0
	if len(args) > 0 {
		if h, err := strconv.ParseFloat(args[0], 64); err == nil && h > 0 && h <= 72 {
			targetHours = h
		}
	}

	now := time.Now().UTC()
	f := types.Fast{
		ID:          randomID(),
		UserID:      msg.UserID,
		StartAt:     now,
		TargetHours: targetHours,
		CreatedAt:   now,
	}
	if err := c.store.StartFast(ctx, f); err != nil {
		return types.Reply{}, fmt.Errorf("start fast: %w", err)
	}

	return types.Reply{
		Text:        fmt.Sprintf("Fasting started at %s. Target: %.0fh.", now.Format("15:04"), targetHours),
		ChannelMeta: msg.ChannelMeta,
	}, nil
}

func (c *FastCommand) handleEnd(ctx context.Context, msg types.InboundMessage) (types.Reply, error) {
	active, err := c.store.GetActiveFast(ctx, msg.UserID)
	if err != nil {
		if errors.Is(err, types.ErrNotFound) {
			return types.Reply{
				Text:        "No active fast. Use /fast start to begin.",
				ChannelMeta: msg.ChannelMeta,
			}, nil
		}
		return types.Reply{}, fmt.Errorf("get active fast: %w", err)
	}

	now := time.Now().UTC()
	elapsed := now.Sub(active.StartAt).Hours()
	completed := elapsed >= active.TargetHours

	ended, err := c.store.EndFast(ctx, msg.UserID, active.ID, now, completed)
	if err != nil {
		return types.Reply{}, fmt.Errorf("end fast: %w", err)
	}

	// Compute actual duration from the returned fast.
	var durationHours float64
	if ended.EndAt != nil {
		durationHours = ended.EndAt.Sub(active.StartAt).Hours()
	}
	return types.Reply{
		Text:        fmt.Sprintf("Fast ended. Duration: %.1fh.", durationHours),
		ChannelMeta: msg.ChannelMeta,
	}, nil
}

func (c *FastCommand) handleStatus(ctx context.Context, msg types.InboundMessage) (types.Reply, error) {
	active, err := c.store.GetActiveFast(ctx, msg.UserID)
	if err != nil {
		if errors.Is(err, types.ErrNotFound) {
			return types.Reply{
				Text:        "No active fast. Use /fast start to begin.",
				ChannelMeta: msg.ChannelMeta,
			}, nil
		}
		return types.Reply{}, fmt.Errorf("get active fast: %w", err)
	}
	return c.showStatus(ctx, msg, active)
}

func (c *FastCommand) showStatus(ctx context.Context, msg types.InboundMessage, active types.Fast) (types.Reply, error) {
	elapsed := time.Since(active.StartAt)
	remaining := time.Duration(active.TargetHours*float64(time.Hour)) - elapsed
	if remaining < 0 {
		remaining = 0
	}

	elapsedStr := formatDurationShort(elapsed)
	remainingStr := formatDurationShort(remaining)

	return types.Reply{
		Text: fmt.Sprintf("Fasting: %s elapsed of %.0fh target (%s remaining)",
			elapsedStr, active.TargetHours, remainingStr),
		ChannelMeta: msg.ChannelMeta,
	}, nil
}

func (c *FastCommand) handleHistory(ctx context.Context, msg types.InboundMessage) (types.Reply, error) {
	fasts, err := c.store.ListFasts(ctx, msg.UserID, 10)
	if err != nil || len(fasts) == 0 {
		return types.Reply{
			Text:        "No fasts logged yet. Use /fast start to begin one.",
			ChannelMeta: msg.ChannelMeta,
		}, nil
	}

	var b strings.Builder
	b.WriteString("Recent fasts:\n\n")
	for _, f := range fasts {
		endStr := "active"
		var durationHours float64
		if f.EndAt != nil {
			endStr = f.EndAt.Format("2006-01-02 15:04")
			durationHours = f.EndAt.Sub(f.StartAt).Hours()
		} else {
			durationHours = time.Since(f.StartAt).Hours()
		}
		status := ""
		if f.Completed {
			status = " (target met)"
		}
		fmt.Fprintf(&b, "  - %s → %s: %.1fh (target %.0fh)%s\n",
			f.StartAt.Format("2006-01-02 15:04"), endStr,
			math.Round(durationHours*10)/10, f.TargetHours, status)
	}
	return types.Reply{Text: b.String(), ChannelMeta: msg.ChannelMeta}, nil
}

// formatDurationShort formats a duration as "Xh Ym" or "Ym".
func formatDurationShort(d time.Duration) string {
	h := int(d.Hours())
	m := int(d.Minutes()) % 60
	if h > 0 {
		return fmt.Sprintf("%dh %dm", h, m)
	}
	return fmt.Sprintf("%dm", m)
}
