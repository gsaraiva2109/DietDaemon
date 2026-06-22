package commands

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/gsaraiva2109/dietdaemon/core/types"
)

// WorkoutStore is the subset of store methods needed by /workout.
type WorkoutStore interface {
	LogWorkout(ctx context.Context, w types.Workout) error
	ListWorkouts(ctx context.Context, userID string, limit int) ([]types.Workout, error)
}

// WorkoutCommand handles /workout -- log a workout or list recent ones.
type WorkoutCommand struct {
	store WorkoutStore
}

// NewWorkoutCommand creates a WorkoutCommand.
func NewWorkoutCommand(s WorkoutStore) *WorkoutCommand {
	return &WorkoutCommand{store: s}
}

func (c *WorkoutCommand) Name() string        { return "/workout" }
func (c *WorkoutCommand) Aliases() []string   { return nil }
func (c *WorkoutCommand) Help() types.I18nKey { return "cmd.workout.usage" }

func (c *WorkoutCommand) Handle(ctx context.Context, msg types.InboundMessage, args string) (types.Reply, error) {
	args = strings.TrimSpace(args)

	if args == "" {
		return types.Reply{
			Text:        "Usage: /workout <name> <minutes> [intensity]\nIntensity: light, moderate, heavy\nExample: /workout Bench Press 45 heavy",
			ChannelMeta: msg.ChannelMeta,
		}, nil
	}

	if args == "list" {
		workouts, err := c.store.ListWorkouts(ctx, msg.UserID, 10)
		if err != nil || len(workouts) == 0 {
			return types.Reply{
				Text:        "No workouts logged yet.",
				ChannelMeta: msg.ChannelMeta,
			}, nil
		}
		var b strings.Builder
		b.WriteString("Recent workouts:\n\n")
		for _, w := range workouts {
			calStr := ""
			if w.CaloriesBurned != nil {
				calStr = fmt.Sprintf(" (~%d kcal)", *w.CaloriesBurned)
			}
			fmt.Fprintf(&b, "  - %s — %d min, %s%s\n", w.Name, w.DurationMin, w.Intensity, calStr)
		}
		return types.Reply{Text: b.String(), ChannelMeta: msg.ChannelMeta}, nil
	}

	// Parse: <name> <duration_min> [intensity] [note...]
	//
	// Find the first numeric token: everything before it is the workout name.
	parts := strings.Fields(args)

	durationIdx := -1
	for i, p := range parts {
		if n, err := strconv.Atoi(p); err == nil && n > 0 {
			durationIdx = i
			break
		}
	}

	if durationIdx < 1 {
		// Need at least one word before the number (the name).
		return types.Reply{
			Text:        "Usage: /workout <name> <minutes> [intensity]\nExample: /workout Bench Press 45 heavy",
			ChannelMeta: msg.ChannelMeta,
		}, nil
	}

	name := strings.Join(parts[:durationIdx], " ")
	durationMin, _ := strconv.Atoi(parts[durationIdx])
	if durationMin <= 0 || durationMin > 1440 {
		return types.Reply{
			Text:        "Invalid duration. Use minutes (1-1440).",
			ChannelMeta: msg.ChannelMeta,
		}, nil
	}

	// Optional intensity (next token after duration).
	intensity := "moderate"
	noteStart := durationIdx + 1
	if noteStart < len(parts) {
		switch strings.ToLower(parts[noteStart]) {
		case "light", "moderate", "heavy":
			intensity = strings.ToLower(parts[noteStart])
			noteStart++
		}
	}

	// Optional note (everything after intensity).
	note := ""
	if noteStart < len(parts) {
		note = strings.Join(parts[noteStart:], " ")
	}

	now := time.Now().UTC()
	entry := types.Workout{
		ID:          randomID(),
		UserID:      msg.UserID,
		Name:        name,
		DurationMin: durationMin,
		Intensity:   intensity,
		Note:        note,
		LoggedAt:    now.Format("2006-01-02 15:04:05"),
	}
	if err := c.store.LogWorkout(ctx, entry); err != nil {
		return types.Reply{}, fmt.Errorf("log workout: %w", err)
	}

	return types.Reply{
		Text:        fmt.Sprintf("Workout logged: %s — %d min, %s", name, durationMin, intensity),
		ChannelMeta: msg.ChannelMeta,
	}, nil
}
