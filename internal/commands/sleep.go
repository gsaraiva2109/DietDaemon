package commands

import (
	"context"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/gsaraiva2109/dietdaemon/core/types"
)

// SleepStore is the subset of store methods needed by /sleep.
type SleepStore interface {
	LogSleep(ctx context.Context, sl types.SleepLog) error
	GetActiveSleep(ctx context.Context, userID string) (*types.SleepLog, error)
	ListSleep(ctx context.Context, userID string, limit int) ([]types.SleepLog, error)
}

// SleepCommand handles /sleep -- log sleep, check status, or list recent logs.
type SleepCommand struct {
	store SleepStore
}

// NewSleepCommand creates a SleepCommand.
func NewSleepCommand(s SleepStore) *SleepCommand {
	return &SleepCommand{store: s}
}

func (c *SleepCommand) Name() string        { return "/sleep" }
func (c *SleepCommand) Aliases() []string   { return nil }
func (c *SleepCommand) Help() types.I18nKey { return "cmd.sleep.usage" }

func (c *SleepCommand) Handle(ctx context.Context, msg types.InboundMessage, args string) (types.Reply, error) {
	args = strings.TrimSpace(args)

	if args == "" {
		return types.Reply{
			Text:        "Usage: /sleep <HH:MM bedtime> <HH:MM wake> [quality]\nQuality: poor, fair, good, great\nExample: /sleep 23:00 07:00 good",
			ChannelMeta: msg.ChannelMeta,
		}, nil
	}

	if args == "status" {
		active, err := c.store.GetActiveSleep(ctx, msg.UserID)
		if err != nil || active == nil {
			return types.Reply{
				Text:        "No active sleep session. Use /sleep <HH:MM> <HH:MM> [quality] to log one.",
				ChannelMeta: msg.ChannelMeta,
			}, nil
		}

		// Parse sleep_at to compute elapsed time. Since sleep_at is just HH:MM,
		// assume it refers to today (or yesterday if it's in the future).
		elapsed := computeSleepDuration(active.SleepAt, time.Now())
		return types.Reply{
			Text:        fmt.Sprintf("Sleeping since %s (%s elapsed)", active.SleepAt, formatDuration(elapsed)),
			ChannelMeta: msg.ChannelMeta,
		}, nil
	}

	if args == "list" {
		logs, err := c.store.ListSleep(ctx, msg.UserID, 10)
		if err != nil || len(logs) == 0 {
			return types.Reply{
				Text:        "No sleep logs yet.",
				ChannelMeta: msg.ChannelMeta,
			}, nil
		}
		var b strings.Builder
		b.WriteString("Recent sleep:\n\n")
		for _, sl := range logs {
			wakeStr := "active"
			if sl.WakeAt != nil {
				wakeStr = *sl.WakeAt
			}
			hours := calcSleepHours(sl.SleepAt, sl.WakeAt)
			fmt.Fprintf(&b, "  - %s to %s (%.1fh) — %s\n", sl.SleepAt, wakeStr, hours, sl.Quality)
			if sl.Note != "" {
				fmt.Fprintf(&b, "    %s\n", sl.Note)
			}
		}
		return types.Reply{Text: b.String(), ChannelMeta: msg.ChannelMeta}, nil
	}

	// Parse: <HH:MM> <HH:MM> [quality]
	parts := strings.Fields(args)
	if len(parts) < 2 {
		return types.Reply{
			Text:        "Usage: /sleep <HH:MM bedtime> <HH:MM wake> [quality]\nExample: /sleep 23:00 07:00 good",
			ChannelMeta: msg.ChannelMeta,
		}, nil
	}

	sleepAt := parts[0]
	wakeAt := parts[1]

	// Validate time formats.
	if _, err := time.Parse("15:04", sleepAt); err != nil {
		return types.Reply{
			Text:        fmt.Sprintf("Invalid time format: %s. Use HH:MM (e.g. 23:00).", sleepAt),
			ChannelMeta: msg.ChannelMeta,
		}, nil
	}
	if _, err := time.Parse("15:04", wakeAt); err != nil {
		return types.Reply{
			Text:        fmt.Sprintf("Invalid time format: %s. Use HH:MM (e.g. 07:00).", wakeAt),
			ChannelMeta: msg.ChannelMeta,
		}, nil
	}

	// Quality is optional; default "ok".
	quality := "ok"
	if len(parts) > 2 {
		q := strings.ToLower(parts[2])
		switch q {
		case "poor", "fair", "good", "great":
			quality = q
		default:
			quality = q // accept any string
		}
	}

	// Note is whatever remains after quality.
	note := ""
	if len(parts) > 3 {
		note = strings.Join(parts[3:], " ")
	}

	sl := types.SleepLog{
		ID:      randomID(),
		UserID:  msg.UserID,
		SleepAt: sleepAt,
		WakeAt:  &wakeAt,
		Quality: quality,
		Note:    note,
	}
	if err := c.store.LogSleep(ctx, sl); err != nil {
		return types.Reply{}, fmt.Errorf("log sleep: %w", err)
	}

	hours := calcSleepHours(sleepAt, &wakeAt)
	return types.Reply{
		Text:        fmt.Sprintf("Sleep logged: %.1fh from %s to %s (%s)", hours, sleepAt, wakeAt, quality),
		ChannelMeta: msg.ChannelMeta,
	}, nil
}

// computeSleepDuration calculates how long ago the sleep started. Since SleepAt
// is an HH:MM string without a date, we assume it refers to today. If the time
// is in the future (e.g. 23:00 at 20:00), we assume it refers to yesterday.
func computeSleepDuration(sleepAt string, now time.Time) time.Duration {
	t, err := time.Parse("15:04", sleepAt)
	if err != nil {
		return 0
	}
	base := time.Date(now.Year(), now.Month(), now.Day(), t.Hour(), t.Minute(), 0, 0, now.Location())
	if base.After(now) {
		base = base.Add(-24 * time.Hour)
	}
	return now.Sub(base)
}

// calcSleepHours returns the number of hours between sleep and wake. Assumes
// overnight if sleep is later than wake. Returns 0 when wakeAt is nil.
func calcSleepHours(sleepAt string, wakeAt *string) float64 {
	if wakeAt == nil {
		return 0
	}
	s, err1 := time.Parse("15:04", sleepAt)
	w, err2 := time.Parse("15:04", *wakeAt)
	if err1 != nil || err2 != nil {
		return 0
	}
	// Assume same day.
	end := time.Date(2000, 1, 1, w.Hour(), w.Minute(), 0, 0, time.UTC)
	start := time.Date(2000, 1, 1, s.Hour(), s.Minute(), 0, 0, time.UTC)
	d := end.Sub(start)
	if d <= 0 {
		d += 24 * time.Hour
	}
	return math.Round(d.Hours()*10) / 10
}

// formatDuration formats a duration as a human-readable string (e.g. "2h 30m").
func formatDuration(d time.Duration) string {
	h := int(d.Hours())
	m := int(d.Minutes()) % 60
	if h > 0 {
		return fmt.Sprintf("%dh %dm", h, m)
	}
	return fmt.Sprintf("%dm", m)
}
