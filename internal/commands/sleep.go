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

// sleepUsage is shown for a bare /sleep.
const sleepUsage = "Usage: /sleep <HH:MM bedtime> <HH:MM wake> [quality]\nQuality: poor, fair, good, great\nExample: /sleep 23:00 07:00 good"

func (c *SleepCommand) Handle(ctx context.Context, msg types.InboundMessage, args string) (types.Reply, error) {
	args = strings.TrimSpace(args)

	switch args {
	case "":
		return types.Reply{Text: sleepUsage, ChannelMeta: msg.ChannelMeta}, nil
	case "status":
		return c.handleStatus(ctx, msg)
	case "list":
		return c.handleList(ctx, msg)
	default:
		return c.handleLog(ctx, msg, args)
	}
}

// handleStatus reports the active sleep session, if any.
func (c *SleepCommand) handleStatus(ctx context.Context, msg types.InboundMessage) (types.Reply, error) {
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

// handleList renders the 10 most recent sleep logs.
func (c *SleepCommand) handleList(ctx context.Context, msg types.InboundMessage) (types.Reply, error) {
	logs, err := c.store.ListSleep(ctx, msg.UserID, 10)
	if err != nil || len(logs) == 0 {
		return types.Reply{Text: "No sleep logs yet.", ChannelMeta: msg.ChannelMeta}, nil
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

// handleLog parses "<HH:MM> <HH:MM> [quality] [note...]" and logs a session.
func (c *SleepCommand) handleLog(ctx context.Context, msg types.InboundMessage, args string) (types.Reply, error) {
	parts := strings.Fields(args)
	if len(parts) < 2 {
		return types.Reply{
			Text:        "Usage: /sleep <HH:MM bedtime> <HH:MM wake> [quality]\nExample: /sleep 23:00 07:00 good",
			ChannelMeta: msg.ChannelMeta,
		}, nil
	}

	sleepAt, wakeAt := parts[0], parts[1]
	if errText := validateSleepTimes(sleepAt, wakeAt); errText != "" {
		return types.Reply{Text: errText, ChannelMeta: msg.ChannelMeta}, nil
	}

	quality, note, errText := parseQualityAndNote(parts[2:])
	if errText != "" {
		return types.Reply{Text: errText, ChannelMeta: msg.ChannelMeta}, nil
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

// validateSleepTimes checks that sleepAt and wakeAt parse as HH:MM, returning
// a user-facing error message for the first one that doesn't (empty if both
// are valid).
func validateSleepTimes(sleepAt, wakeAt string) string {
	if _, err := time.Parse("15:04", sleepAt); err != nil {
		return fmt.Sprintf("Invalid time format: %s. Use HH:MM (e.g. 23:00).", sleepAt)
	}
	if _, err := time.Parse("15:04", wakeAt); err != nil {
		return fmt.Sprintf("Invalid time format: %s. Use HH:MM (e.g. 07:00).", wakeAt)
	}
	return ""
}

// parseQualityAndNote parses the optional quality token and trailing note
// from the fields after bedtime/wake. Quality defaults to "ok" when omitted.
// When given, it must be one of the documented values (poor, fair, good,
// great): the two switch branches used to both fall through to "accept
// whatever string was typed" (SonarQube go:S3923), which silently let typos
// like "godo" through as a bogus quality and swallowed the following word as
// a "note" instead. Rejecting unknown values with a clear error is the
// differentiated behavior the switch was originally meant to have.
func parseQualityAndNote(rest []string) (quality, note, errText string) {
	quality = "ok"
	if len(rest) == 0 {
		return quality, note, ""
	}
	q := strings.ToLower(rest[0])
	switch q {
	case "poor", "fair", "good", "great":
		quality = q
	default:
		return "", "", fmt.Sprintf("Invalid quality: %s. Use one of: poor, fair, good, great.", rest[0])
	}
	if len(rest) > 1 {
		note = strings.Join(rest[1:], " ")
	}
	return quality, note, ""
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
		// AddDate, not Add(-24*time.Hour): calendar-day arithmetic stays
		// correct across a DST transition, where "yesterday" isn't always
		// exactly 24 hours ago in a location-aware time.Time.
		base = base.AddDate(0, 0, -1)
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
