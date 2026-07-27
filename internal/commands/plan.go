package commands

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/gsaraiva2109/dietdaemon/core/types"
)

// PlanStore is the subset of store methods needed by /plan. Every method
// here is a read: /plan has no write path to the diet-plan tables at all --
// entering or editing a plan's day-types, slots, and options belongs to the
// builder UI, not chat (see ResolveDayType's doc for why a bot-side
// re-implementation of the day-type resolution would be a trap).
type PlanStore interface {
	GetUser(ctx context.Context, userID string) (types.User, error)
	ResolveDayType(ctx context.Context, userID, date string) (types.DietPlanDayType, bool, error)
	GetPlanBundle(ctx context.Context, planID string) (types.PlanBundle, error)
	GetTemplate(ctx context.Context, templateID string) (types.MealTemplate, error)
}

// PlanCommand handles /plan -- read-only lookups against the user's active
// diet plan: today's day-type, the next prescribed meal slot, and what's
// prescribed for a named meal. It never creates, edits, or deletes plan rows;
// that nested day-types -> slots -> options structure is painful to build
// over chat and belongs to the builder UI.
type PlanCommand struct {
	store PlanStore
	loc   *time.Location
}

// NewPlanCommand creates a PlanCommand. The loc parameter is the fallback
// timezone used when the user has not set their own.
func NewPlanCommand(s PlanStore, loc *time.Location) *PlanCommand {
	return &PlanCommand{store: s, loc: loc}
}

func (c *PlanCommand) Name() string        { return "/plan" }
func (c *PlanCommand) Aliases() []string   { return nil }
func (c *PlanCommand) Help() types.I18nKey { return "cmd.plan.usage" }

func (c *PlanCommand) Handle(ctx context.Context, msg types.InboundMessage, args string) (types.Reply, error) {
	loc := c.loc
	if u, err := c.store.GetUser(ctx, msg.UserID); err == nil && u.Timezone != "" {
		if l, err := time.LoadLocation(u.Timezone); err == nil {
			loc = l
		}
	}
	now := time.Now().In(loc)
	today := now.Format("2006-01-02")

	dt, ok, err := c.store.ResolveDayType(ctx, msg.UserID, today)
	if err != nil {
		return types.Reply{}, fmt.Errorf("resolve day type: %w", err)
	}
	if !ok {
		return types.Reply{
			Text:        "No diet plan governs today. Build one from the dashboard, or use /status for your daily targets.",
			ChannelMeta: msg.ChannelMeta,
		}, nil
	}

	bundle, err := c.store.GetPlanBundle(ctx, dt.PlanID)
	if err != nil {
		return types.Reply{}, fmt.Errorf("get plan bundle: %w", err)
	}
	slots := slotsForDayType(bundle, dt.ID)

	switch strings.ToLower(strings.TrimSpace(args)) {
	case "", "today", "hoje":
		return c.todayReply(dt, slots, now, msg), nil
	case "next", "próximo", "proximo":
		return c.nextReply(slots, now, msg), nil
	default:
		return c.mealReply(ctx, dt, slots, args, msg), nil
	}
}

// slotsForDayType returns the slots nested under dayTypeID within an
// already-loaded plan bundle, or nil if the day-type isn't in it.
func slotsForDayType(bundle types.PlanBundle, dayTypeID string) []types.DietPlanSlotBundle {
	for _, b := range bundle.DayTypes {
		if b.ID == dayTypeID {
			return b.Slots
		}
	}
	return nil
}

// todayReply renders the day-type's targets and its full meal schedule, with
// the next upcoming slot marked.
func (c *PlanCommand) todayReply(dt types.DietPlanDayType, slots []types.DietPlanSlotBundle, now time.Time, msg types.InboundMessage) types.Reply {
	var b strings.Builder
	fmt.Fprintf(&b, "Today: %s\n%.0f kcal | P %.0fg . C %.0fg . F %.0fg\n", dt.Name, dt.Targets.Calories, dt.Targets.Protein, dt.Targets.Carbs, dt.Targets.Fat)
	if len(slots) == 0 {
		b.WriteString("\nNo meal slots defined for this day-type yet.")
		return types.Reply{Text: b.String(), ChannelMeta: msg.ChannelMeta}
	}
	nextID := nextSlotID(slots, now)
	b.WriteString("\n")
	for _, sl := range slots {
		marker := "  "
		if sl.ID == nextID {
			marker = "->"
		}
		fmt.Fprintf(&b, "%s %s %s -- %s\n", marker, sl.TimeOfDay, sl.Label, optionSummary(sl.Options))
	}
	return types.Reply{Text: b.String(), ChannelMeta: msg.ChannelMeta}
}

// nextReply renders just the next slot due today, or a message once the
// day's meals are past. It deliberately does not look ahead to tomorrow --
// a new day can resolve to a different day-type in the cycle, so "next"
// always means "next today".
func (c *PlanCommand) nextReply(slots []types.DietPlanSlotBundle, now time.Time, msg types.InboundMessage) types.Reply {
	id := nextSlotID(slots, now)
	for _, sl := range slots {
		if sl.ID == id {
			text := fmt.Sprintf("Next: %s at %s -- %s", sl.Label, sl.TimeOfDay, optionSummary(sl.Options))
			return types.Reply{Text: text, ChannelMeta: msg.ChannelMeta}
		}
	}
	return types.Reply{Text: "No more planned meals today.", ChannelMeta: msg.ChannelMeta}
}

// mealReply finds the slot whose label matches query (case-insensitive
// substring) and expands its options into their prescribed items, loading
// each option's backing template. Unlike todayReply/nextReply, this is the
// only path that loads templates -- it is also the only one that needs to,
// since "what's prescribed" means the actual foods, not just the slot name.
func (c *PlanCommand) mealReply(ctx context.Context, dt types.DietPlanDayType, slots []types.DietPlanSlotBundle, query string, msg types.InboundMessage) types.Reply {
	sl, ok := findSlotByLabel(slots, query)
	if !ok {
		return types.Reply{
			Text:        fmt.Sprintf("No meal slot matching %q on today's %s plan.", query, dt.Name),
			ChannelMeta: msg.ChannelMeta,
		}
	}

	var b strings.Builder
	fmt.Fprintf(&b, "%s (%s):\n", sl.Label, sl.TimeOfDay)
	if len(sl.Options) == 0 {
		b.WriteString("No options prescribed yet.")
		return types.Reply{Text: b.String(), ChannelMeta: msg.ChannelMeta}
	}
	for _, opt := range sl.Options {
		tmpl, err := c.store.GetTemplate(ctx, opt.TemplateID)
		if err != nil {
			fmt.Fprintf(&b, "  - %s\n", opt.Label)
			continue
		}
		// macrosSum naturally excludes ad libitum items (quantity 0 scales
		// their macros to 0 already) -- no separate flag needed.
		total := macrosSum(tmpl.Items)
		fmt.Fprintf(&b, "  - %s -- %.0f kcal (P%.0f/C%.0f/F%.0f)\n", opt.Label, total.Calories, total.Protein, total.Carbs, total.Fat)
		for _, item := range tmpl.Items {
			if item.Parsed.NormalizedGrams == 0 {
				fmt.Fprintf(&b, "      %s (à vontade)\n", item.Match.Name)
				continue
			}
			fmt.Fprintf(&b, "      %.0fg %s\n", item.Parsed.NormalizedGrams, item.Match.Name)
		}
	}
	return types.Reply{Text: b.String(), ChannelMeta: msg.ChannelMeta}
}

// findSlotByLabel returns the first slot whose label contains query
// case-insensitively.
func findSlotByLabel(slots []types.DietPlanSlotBundle, query string) (types.DietPlanSlotBundle, bool) {
	q := strings.ToLower(strings.TrimSpace(query))
	for _, sl := range slots {
		if strings.Contains(strings.ToLower(sl.Label), q) {
			return sl, true
		}
	}
	return types.DietPlanSlotBundle{}, false
}

// optionSummary joins option labels for the compact today/next views.
func optionSummary(opts []types.DietPlanSlotOption) string {
	if len(opts) == 0 {
		return "no options yet"
	}
	names := make([]string, len(opts))
	for i, o := range opts {
		names[i] = o.Label
	}
	return strings.Join(names, " / ")
}

// nextSlotID returns the ID of the slot with the earliest time_of_day at or
// after now, or "" if every slot today has already passed (or none parse).
func nextSlotID(slots []types.DietPlanSlotBundle, now time.Time) string {
	nowMinutes := now.Hour()*60 + now.Minute()
	bestID := ""
	bestDelta := -1
	for _, sl := range slots {
		mins, ok := parseTimeOfDay(sl.TimeOfDay)
		if !ok || mins < nowMinutes {
			continue
		}
		if delta := mins - nowMinutes; bestDelta == -1 || delta < bestDelta {
			bestDelta = delta
			bestID = sl.ID
		}
	}
	return bestID
}

// parseTimeOfDay parses a "HH:MM" slot time into minutes since midnight.
func parseTimeOfDay(s string) (int, bool) {
	t, err := time.Parse("15:04", s)
	if err != nil {
		return 0, false
	}
	return t.Hour()*60 + t.Minute(), true
}
