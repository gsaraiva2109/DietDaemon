// Package scheduler periodically checks each user's progress against their
// daily macro targets and fires nudges when they fall behind. It is the
// component that addresses the project's core problem: a bulking user missing
// meals. Evaluation is timezone-correct (per the user's local day) and
// deduplicated so a given rule nudges at most once per local day.
package scheduler

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"math"
	"time"

	"github.com/gsaraiva2109/dietdaemon/core/types"
)

// Store is the read side the scheduler needs. The concrete *store.Store
// satisfies it once it gains ListUsers (its other methods already exist).
type Store interface {
	ListUsers(ctx context.Context) ([]types.User, error)
	GetTargets(ctx context.Context, userID string) (types.DailyTargets, error)
	GetRollup(ctx context.Context, userID, localDate string) (types.DailyRollup, error)
}

// NudgeStore persists which nudges have already fired, keyed by user, local
// date, and rule id, so a rule fires at most once per local day. This is the
// dedupe boundary; the SQLite implementation lives in internal/store.
type NudgeStore interface {
	WasNudged(ctx context.Context, userID, localDate, ruleID string) (bool, error)
	MarkNudged(ctx context.Context, userID, localDate, ruleID string) error
}

// Notifier delivers a nudge. Satisfied by any ports.Notifier.
type Notifier interface {
	Notify(ctx context.Context, n types.Notification) error
}

// HealthStore provides the read side for non-macro health data used by health
// domain nudging rules. The concrete *store.Store will satisfy this interface
// once water, workout, and sleep methods are added; fasting methods already
// exist. Define it here so the scheduler compiles independently of the
// store implementation schedule.
type HealthStore interface {
	// GetWaterToday returns the day's water logs and their total millilitres.
	// Matches *store.Store's real signature, which every other caller
	// (handler.go) already relies on.
	GetWaterToday(ctx context.Context, userID, localDate string) (logs []types.WaterLog, totalML int, err error)

	// ListWorkouts returns the most recent workouts, newest first.
	ListWorkouts(ctx context.Context, userID string, limit int) ([]types.Workout, error)

	// GetActiveSleep returns the user's in-progress sleep (wake_at IS NULL), or
	// types.ErrNotFound if none is active. Matches *store.Store's real
	// signature (pointer return), which every other caller in this codebase
	// (handler.go, commands/sleep.go) already relies on.
	GetActiveSleep(ctx context.Context, userID string) (*types.SleepLog, error)

	// GetActiveFast returns the user's in-progress fast (end_at IS NULL), or
	// types.ErrNotFound if none is active.
	GetActiveFast(ctx context.Context, userID string) (types.Fast, error)

	// ListFasts returns the user's most recent fasting windows, newest first.
	ListFasts(ctx context.Context, userID string, limit int) ([]types.Fast, error)
}

// RuleConfigStore provides per-user overrides of nudge rules (enable/disable,
// tune a rule's fields). The concrete *store.Store satisfies it once
// GetNudgeRuleConfig is added.
type RuleConfigStore interface {
	GetNudgeRuleConfig(ctx context.Context, userID string) ([]types.NudgeRuleConfig, error)
}

// DigestStore provides the read side for composing the weekly digest
// notification. The concrete *store.Store already satisfies this via its
// existing GetRollups, ListWeight, GetWaterDailyTotals, and
// ListWorkoutsInRange methods.
type DigestStore interface {
	GetRollups(ctx context.Context, userID, startDate, endDate string) ([]types.DailyRollup, error)
	ListWeight(ctx context.Context, userID string, days int) ([]types.WeightEntry, error)
	GetWaterDailyTotals(ctx context.Context, userID, startDate, endDate string) ([]types.WaterDayTotal, error)
	ListWorkoutsInRange(ctx context.Context, userID, startDate, endDate string) ([]types.Workout, error)
}

// ChatRouteStore resolves the chat metadata needed to reach a user
// proactively. The concrete *store.Store satisfies it once GetChatRoute is
// added.
type ChatRouteStore interface {
	GetChatRoute(ctx context.Context, userID string) (channel string, meta map[string]string, err error)
}

// ChatSender delivers a Reply to a chat conversation. Satisfied by any
// ports.MessagingAdapter.
type ChatSender interface {
	Send(ctx context.Context, reply types.Reply) error
}

// WeeklyBudgetStore provides the read side for weekly rolling budget
// compensation. The concrete *store.Store already satisfies this via its
// existing GetRollups method.
type WeeklyBudgetStore interface {
	GetRollups(ctx context.Context, userID, startDate, endDate string) ([]types.DailyRollup, error)
}

// SentNudgeStore records delivered nudges so they can be undone later.
type SentNudgeStore interface {
	RecordSentNudge(ctx context.Context, n types.SentNudge) error
}

// Option configures a Scheduler. Used with the variadic New constructor.
type Option func(*Scheduler)

// Scheduler evaluates rules on a fixed interval.
type Scheduler struct {
	store             Store
	nudges            NudgeStore
	notifier          Notifier
	rules             []Rule
	healthStore       HealthStore
	healthRules       []HealthRule
	ruleConfig        RuleConfigStore
	digestStore       DigestStore
	digestRules       []DigestRule
	chatRoutes        ChatRouteStore
	chatSender        ChatSender
	weeklyBudgetStore WeeklyBudgetStore
	weeklyBudgetRules []WeeklyBudgetRule
	sentNudges        SentNudgeStore
	defaultLoc        *time.Location
	interval          time.Duration

	now func() time.Time
	log *slog.Logger
}

// New builds a Scheduler. defaultLoc is used for users without an explicit
// timezone; interval is the tick period (e.g. 5 minutes). Pass zero or more
// Option values to attach optional behaviour such as WithHealthRules.
func New(store Store, nudges NudgeStore, notifier Notifier, rules []Rule, defaultLoc *time.Location, interval time.Duration, opts ...Option) *Scheduler {
	if defaultLoc == nil {
		defaultLoc = time.UTC
	}
	if interval <= 0 {
		interval = 5 * time.Minute
	}
	s := &Scheduler{
		store:      store,
		nudges:     nudges,
		notifier:   notifier,
		rules:      rules,
		defaultLoc: defaultLoc,
		interval:   interval,
		now:        time.Now,
		log:        slog.Default(),
	}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

// WithHealthRules attaches health-domain rules and their data source to the
// scheduler. When nil is passed for healthRules no health-domain nudges are
// evaluated. This is a functional option intended for use with New.
func WithHealthRules(hs HealthStore, healthRules []HealthRule) Option {
	return func(s *Scheduler) {
		s.healthStore = hs
		s.healthRules = healthRules
	}
}

// WithRuleConfig attaches a per-user rule override source. When not passed,
// every rule runs with its hardcoded defaults (fully backward compatible).
func WithRuleConfig(rcs RuleConfigStore) Option {
	return func(s *Scheduler) {
		s.ruleConfig = rcs
	}
}

// WithDigestRules attaches the weekly digest rules and their data source to
// the scheduler. When nil is passed for digestRules no digest is evaluated.
func WithDigestRules(ds DigestStore, digestRules []DigestRule) Option {
	return func(s *Scheduler) {
		s.digestStore = ds
		s.digestRules = digestRules
	}
}

// WithChatSender attaches a chat-routing store and a MessagingAdapter so
// nudges can be delivered as chat messages instead of only plain text via
// Notifier — this is the prerequisite for buttons/undo on nudges (later
// features build on top of this). When not passed, or when no chat route is
// known yet for a given user, delivery falls back to Notifier unchanged, so
// this is fully backward compatible.
func WithChatSender(routes ChatRouteStore, sender ChatSender) Option {
	return func(s *Scheduler) {
		s.chatRoutes = routes
		s.chatSender = sender
	}
}

// WithSentNudges attaches a SentNudgeStore so the scheduler records every
// delivered nudge and attaches an Undo button to chat-delivered messages.
// When not passed, no sent-nudge rows are written and no undo button appears.
func WithSentNudges(sns SentNudgeStore) Option {
	return func(s *Scheduler) {
		s.sentNudges = sns
	}
}

// WithWeeklyBudgetRules attaches weekly rolling budget rules and their data
// source to the scheduler. When not passed, no weekly budget nudges are
// evaluated. Unlike macro/health/digest rules, weekly budget rules are OFF
// by default — the user must opt in via nudge rule config.
func WithWeeklyBudgetRules(wbs WeeklyBudgetStore, budgetRules []WeeklyBudgetRule) Option {
	return func(s *Scheduler) {
		s.weeklyBudgetStore = wbs
		s.weeklyBudgetRules = budgetRules
	}
}

// EffectiveWeeklyTarget computes the rolling effective daily target for a
// macro, self-correcting for over-/under-eating earlier in the calendar week.
//
//	weeklyTarget = plainDaily * 7
//	effective = (weeklyTarget - consumedPriorDays) / daysRemaining
//
// The result is clamped to [floorPct*plainDaily, ceilPct*plainDaily].
// daysRemaining includes today (1-7). Monday: consumedPriorDays=0,
// daysRemaining=7 → effective=plainDaily.
func EffectiveWeeklyTarget(plainDaily, consumedPriorDays float64, daysRemaining int, floorPct, ceilPct float64) float64 {
	weeklyTarget := plainDaily * 7
	effective := (weeklyTarget - consumedPriorDays) / float64(daysRemaining)
	floor := floorPct * plainDaily
	ceil := ceilPct * plainDaily
	if effective < floor {
		return floor
	}
	if effective > ceil {
		return ceil
	}
	return effective
}
func (s *Scheduler) Run(ctx context.Context) {
	t := time.NewTicker(s.interval)
	defer t.Stop()
	s.tick(ctx, s.now())
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			s.tick(ctx, s.now())
		}
	}
}

// tick evaluates every user once.
func (s *Scheduler) tick(ctx context.Context, now time.Time) {
	users, err := s.store.ListUsers(ctx)
	if err != nil {
		s.log.Error("scheduler: list users", "err", err)
		return
	}
	for _, u := range users {
		s.evalUser(ctx, now, u)
	}
}

// evalUser checks all rules for one user at the given instant. Health rules
// are evaluated even when the user has no macro targets set (they are
// independent of macro goals).
func (s *Scheduler) evalUser(ctx context.Context, now time.Time, user types.User) {
	local := now.In(s.locFor(user))
	date := local.Format("2006-01-02")

	// Fetch this user's rule overrides once per tick (not once per rule) to
	// avoid N queries. Missing store or no rows: overrides stays nil, and
	// resolveRule treats every rule as un-overridden — fully backward
	// compatible with hardcoded defaults.
	var overrides map[string]types.NudgeRuleConfig
	if s.ruleConfig != nil {
		cfgs, err := s.ruleConfig.GetNudgeRuleConfig(ctx, user.ID)
		if err != nil {
			s.log.Error("scheduler: get rule config", "user", user.ID, "err", err)
		} else {
			overrides = make(map[string]types.NudgeRuleConfig, len(cfgs))
			for _, c := range cfgs {
				overrides[c.RuleID] = c
			}
		}
	}

	// Macro rules (require targets).
	targets, err := s.store.GetTargets(ctx, user.ID)
	if err == nil {
		rollup, err := s.store.GetRollup(ctx, user.ID, date)
		if err != nil {
			rollup = types.DailyRollup{} // no meals logged yet today
		}

		for _, base := range s.rules {
			r, enabled := resolveRule(base, base.ID, overrides)
			if !enabled {
				continue
			}
			if local.Hour() < r.AfterHour {
				continue
			}
			target := macroValue(targets.Targets, r.Macro)
			if target <= 0 {
				continue // no target for this macro
			}
			consumed := macroValue(rollup.Consumed, r.Macro)
			if consumed/target >= r.MinFraction {
				continue // on track
			}

			done, err := s.nudges.WasNudged(ctx, user.ID, date, r.ID)
			if err != nil {
				s.log.Error("scheduler: was-nudged", "rule", r.ID, "err", err)
				continue
			}
			if done {
				continue
			}

			n := types.Notification{
				UserID:   user.ID,
				Title:    "DietDaemon",
				Body:     fmt.Sprintf(r.Message, consumed, target),
				Priority: types.PriorityHigh,
			}
			if err := s.deliver(ctx, user, r.ID, n, &rollup.Consumed, r.QuickActions); err != nil {
				s.log.Error("scheduler: notify", "rule", r.ID, "err", err)
				continue // not marked: retry next tick
			}
			if err := s.nudges.MarkNudged(ctx, user.ID, date, r.ID); err != nil {
				s.log.Error("scheduler: mark-nudged", "rule", r.ID, "err", err)
			}
		}
	}

	// Health rules (independent of macro targets).
	if s.healthStore != nil {
		s.evalHealthRules(ctx, now, user, overrides)
	}

	// Weekly digest (independent of macro targets and health data).
	if s.digestStore != nil {
		s.evalDigestRules(ctx, now, user, overrides)
	}

	// Weekly rolling budget (self-correcting targets).
	if s.weeklyBudgetStore != nil {
		s.evalWeeklyBudgetRules(ctx, now, user, overrides)
	}
}

// resolveRule applies a user's override (if any) to a copy of the base rule.
// The second return value is false when the rule should be skipped entirely
// (an explicit disable); otherwise it's true and the returned rule carries
// any tuned fields from the override's Params on top of the base rule's
// defaults. A nil overrides map or no matching entry returns base unchanged.
func resolveRule[T any](base T, ruleID string, overrides map[string]types.NudgeRuleConfig) (T, bool) {
	c, found := overrides[ruleID]
	if !found {
		return base, true
	}
	if !c.Enabled {
		return base, false
	}
	if len(c.Params) > 0 {
		// Unmarshal into a copy of the existing rule (not a zero value) so
		// fields absent from Params keep the base rule's defaults.
		if err := json.Unmarshal(c.Params, &base); err != nil {
			return base, true // malformed override: fall back to defaults
		}
	}
	return base, true
}

// deliver sends a nudge, preferring an interactive chat message over the
// notifier when a chat route is known for the user. Falls back to the
// plain-text Notifier when no chat route is configured/known yet, or when the
// chat send itself fails — so ntfy/gotify remain a fully functional delivery
// path with zero adapter-side changes.
//
// When snapshot is non-nil and a SentNudgeStore is configured, a sent_nudges
// row is recorded and an Undo button is attached to the chat reply.
func (s *Scheduler) deliver(ctx context.Context, user types.User, ruleID string, n types.Notification, snapshot *types.Macros, quickActions []types.InlineButton) error {
	var nudgeID string
	if snapshot != nil && s.sentNudges != nil {
		var b [16]byte
		_, _ = rand.Read(b[:])
		nudgeID = hex.EncodeToString(b[:])
		sn := types.SentNudge{
			ID:       nudgeID,
			UserID:   user.ID,
			RuleID:   ruleID,
			SentAt:   time.Now(),
			Body:     n.Body,
			Snapshot: *snapshot,
			Status:   "sent",
		}
		if err := s.sentNudges.RecordSentNudge(ctx, sn); err != nil {
			s.log.Error("scheduler: record sent nudge", "rule", ruleID, "err", err)
		}
	}

	if s.chatSender != nil && s.chatRoutes != nil && (snapshot != nil || len(quickActions) > 0) {
		if _, meta, err := s.chatRoutes.GetChatRoute(ctx, user.ID); err == nil {
			reply := types.Reply{UserID: user.ID, Text: n.Body, ChannelMeta: meta}
			var row []types.InlineButton
			if nudgeID != "" {
				row = append(row, types.InlineButton{Text: "Not anymore, undo", CallbackData: "/nudge undo " + nudgeID})
			}
			row = append(row, quickActions...)
			if len(row) > 0 {
				reply.Markup = &types.ReplyMarkup{
					InlineKeyboard: [][]types.InlineButton{row},
				}
			}
			sendErr := s.chatSender.Send(ctx, reply)
			if sendErr == nil {
				return nil
			}
			s.log.Warn("scheduler: chat send failed, falling back to notifier", "rule", ruleID, "err", sendErr)
		}
	}
	if s.notifier == nil {
		return fmt.Errorf("scheduler: no delivery channel configured for user %s", user.ID)
	}
	return s.notifier.Notify(ctx, n)
}

// evalHealthRules evaluates every health rule for one user at the given
// instant. It uses the same nudge_log table for deduplication, keyed by
// (user_id, local_date, rule_id), so health rule IDs like "water-afternoon"
// coexist safely with macro rule IDs.
func (s *Scheduler) evalHealthRules(ctx context.Context, now time.Time, user types.User, overrides map[string]types.NudgeRuleConfig) {
	local := now.In(s.locFor(user))
	date := local.Format("2006-01-02")
	hour := local.Hour()

	for _, base := range s.healthRules {
		r, enabled := resolveRule(base, base.ID, overrides)
		if !enabled {
			continue
		}
		// Hour gate: CheckHour = 0 means always check (e.g. fast-ending).
		if r.CheckHour > 0 && hour < r.CheckHour {
			continue
		}

		// Deduplication against nudge_log table.
		done, err := s.nudges.WasNudged(ctx, user.ID, date, r.ID)
		if err != nil {
			s.log.Error("scheduler: health was-nudged", "rule", r.ID, "err", err)
			continue
		}
		if done {
			continue
		}

		triggered := false
		switch r.Domain {
		case "water":
			_, totalML, err := s.healthStore.GetWaterToday(ctx, user.ID, date)
			if err != nil {
				s.log.Error("scheduler: get water", "rule", r.ID, "err", err)
				continue
			}
			if totalML < int(r.MinDailyAmount) {
				triggered = true
			}

		case "workout":
			workouts, err := s.healthStore.ListWorkouts(ctx, user.ID, 1)
			if err != nil && !errors.Is(err, types.ErrNotFound) {
				s.log.Error("scheduler: list workouts", "rule", r.ID, "err", err)
				continue
			}
			if len(workouts) == 0 {
				triggered = true // never worked out
				break
			}
			lastTime, parseErr := parseLoggedAt(workouts[0].LoggedAt)
			if parseErr != nil {
				s.log.Error("scheduler: parse workout date", "rule", r.ID, "err", parseErr)
				continue
			}
			if now.Sub(lastTime).Hours() >= float64(r.MaxGapDays)*24 {
				triggered = true
			}

		case "sleep":
			_, err := s.healthStore.GetActiveSleep(ctx, user.ID)
			if errors.Is(err, types.ErrNotFound) {
				triggered = true // no active sleep — nudge
			} else if err != nil {
				s.log.Error("scheduler: get sleep", "rule", r.ID, "err", err)
				continue
			}

		case "fasting":
			activeFast, err := s.healthStore.GetActiveFast(ctx, user.ID)
			if errors.Is(err, types.ErrNotFound) {
				continue // no active fast — nothing to nudge about
			}
			if err != nil {
				s.log.Error("scheduler: get fast", "rule", r.ID, "err", err)
				continue
			}
			elapsed := now.Sub(activeFast.StartAt).Hours()
			remaining := activeFast.TargetHours - elapsed
			if remaining > 0 && remaining <= 0.5 {
				triggered = true // within 30 minutes of target
			}
		}

		if !triggered {
			continue
		}

		n := types.Notification{
			UserID:   user.ID,
			Title:    "DietDaemon",
			Body:     r.Message,
			Priority: types.PriorityHigh,
		}
		var healthSnap *types.Macros
		if len(r.QuickActions) > 0 {
			healthSnap = &types.Macros{}
		}
		if err := s.deliver(ctx, user, r.ID, n, healthSnap, r.QuickActions); err != nil {
			s.log.Error("scheduler: health notify", "rule", r.ID, "err", err)
			continue // not marked: retry next tick
		}
		if err := s.nudges.MarkNudged(ctx, user.ID, date, r.ID); err != nil {
			s.log.Error("scheduler: health mark-nudged", "rule", r.ID, "err", err)
		}
	}
}

// evalDigestRules evaluates the weekly digest rule(s) for one user. Dedupe
// uses the same nudge_log table as macro/health rules, but keyed by ISO
// year-week (e.g. "2026-W27") instead of a daily date, so the unconstrained
// TEXT local_date column naturally dedupes per week with no schema change and
// no format collision with daily "YYYY-MM-DD" keys.
func (s *Scheduler) evalDigestRules(ctx context.Context, now time.Time, user types.User, overrides map[string]types.NudgeRuleConfig) {
	local := now.In(s.locFor(user))

	for _, base := range s.digestRules {
		r, enabled := resolveRule(base, base.ID, overrides)
		if !enabled {
			continue
		}
		if local.Weekday() != r.Weekday || local.Hour() < r.CheckHour {
			continue
		}

		year, week := local.ISOWeek()
		weekKey := fmt.Sprintf("%d-W%02d", year, week)

		done, err := s.nudges.WasNudged(ctx, user.ID, weekKey, r.ID)
		if err != nil {
			s.log.Error("scheduler: digest was-nudged", "rule", r.ID, "err", err)
			continue
		}
		if done {
			continue
		}

		body, err := s.buildDigestBody(ctx, user, local)
		if err != nil {
			s.log.Error("scheduler: build digest", "rule", r.ID, "err", err)
			continue
		}

		n := types.Notification{
			UserID:   user.ID,
			Title:    "DietDaemon Weekly Digest",
			Body:     body,
			Priority: types.PriorityDefault,
		}
		if err := s.deliver(ctx, user, r.ID, n, nil, nil); err != nil {
			s.log.Error("scheduler: digest notify", "rule", r.ID, "err", err)
			continue // not marked: retry next tick
		}
		if err := s.nudges.MarkNudged(ctx, user.ID, weekKey, r.ID); err != nil {
			s.log.Error("scheduler: digest mark-nudged", "rule", r.ID, "err", err)
		}
	}
}

// evalWeeklyBudgetRules evaluates the weekly rolling budget rules for one
// user. Unlike macro/health/digest rules, these are OFF by default (opt-in).
// Dedupe uses the same nudge_log table, keyed by local date.
func (s *Scheduler) evalWeeklyBudgetRules(ctx context.Context, now time.Time, user types.User, overrides map[string]types.NudgeRuleConfig) {
	local := now.In(s.locFor(user))
	date := local.Format("2006-01-02")

	for _, base := range s.weeklyBudgetRules {
		r := base

		// Opt-in gate: this feature is OFF by default.
		cfg, ok := overrides[r.ID]
		if !ok || !cfg.Enabled {
			continue
		}

		// Decode WeeklyBudgetConfig from params.
		var budgetCfg types.WeeklyBudgetConfig
		if len(cfg.Params) > 0 {
			if err := json.Unmarshal(cfg.Params, &budgetCfg); err != nil {
				s.log.Error("scheduler: unmarshal weekly budget config", "rule", r.ID, "err", err)
				continue
			}
		}

		// Apply defaults for clamp values.
		floorPct := budgetCfg.ClampFloorPct
		if floorPct == 0 {
			floorPct = 0.70
		}
		ceilPct := budgetCfg.ClampCeilPct
		if ceilPct == 0 {
			ceilPct = 1.30
		}

		// Hour gate.
		if local.Hour() < r.CheckHour {
			continue
		}

		// Dedupe against nudge_log.
		done, err := s.nudges.WasNudged(ctx, user.ID, date, r.ID)
		if err != nil {
			s.log.Error("scheduler: weekly budget was-nudged", "rule", r.ID, "err", err)
			continue
		}
		if done {
			continue
		}

		// Compute calendar week (Monday-Sunday) bounds.
		weekday := local.Weekday()
		daysFromMonday := int(weekday) - int(time.Monday)
		if weekday == time.Sunday {
			daysFromMonday = 6
		}
		monday := local.AddDate(0, 0, -daysFromMonday)
		sunday := monday.AddDate(0, 0, 6)
		daysRemaining := 7 - daysFromMonday

		// Get rollups for the current week.
		rollups, err := s.weeklyBudgetStore.GetRollups(ctx, user.ID, monday.Format("2006-01-02"), sunday.Format("2006-01-02"))
		if err != nil {
			s.log.Error("scheduler: get weekly rollups", "rule", r.ID, "err", err)
			continue
		}

		// Sum consumed prior days (dates strictly before today).
		var consumedPriorDays float64
		for _, roll := range rollups {
			if roll.Date >= date {
				continue
			}
			consumedPriorDays += macroValue(roll.Consumed, r.Macro)
		}

		// Get daily target for this macro.
		targets, err := s.store.GetTargets(ctx, user.ID)
		if err != nil {
			s.log.Error("scheduler: get targets for weekly budget", "rule", r.ID, "err", err)
			continue
		}
		plainDaily := macroValue(targets.Targets, r.Macro)
		if plainDaily <= 0 {
			continue // no target for this macro
		}

		// Apply weekly target override if configured.
		if budgetCfg.WeeklyTargetOverride > 0 {
			plainDaily = budgetCfg.WeeklyTargetOverride
		}

		// Compute effective target.
		effective := EffectiveWeeklyTarget(plainDaily, consumedPriorDays, daysRemaining, floorPct, ceilPct)

		// Check if delta is negligible (< 3% of daily target).
		delta := effective - plainDaily
		if math.Abs(delta) < plainDaily*0.03 {
			_ = s.nudges.MarkNudged(ctx, user.ID, date, r.ID)
			continue
		}

		// Build notification message.
		unit := "kcal"
		if r.Macro == MacroProtein {
			unit = "g"
		}
		var body string
		if delta > 0 {
			body = fmt.Sprintf("Catch up today, +%.0f%s", delta, unit)
		} else {
			body = fmt.Sprintf("Ease up today, -%.0f%s", -delta, unit)
		}

		n := types.Notification{
			UserID:   user.ID,
			Title:    "DietDaemon",
			Body:     body,
			Priority: types.PriorityHigh,
		}
		if err := s.deliver(ctx, user, r.ID, n, nil, nil); err != nil {
			s.log.Error("scheduler: weekly budget notify", "rule", r.ID, "err", err)
			continue
		}
		if err := s.nudges.MarkNudged(ctx, user.ID, date, r.ID); err != nil {
			s.log.Error("scheduler: weekly budget mark-nudged", "rule", r.ID, "err", err)
		}
	}
}

// buildDigestBody composes a short readable summary of the last 7 days:
// average calories/protein, average adherence to target, weight change,
// water intake, and workouts.
func (s *Scheduler) buildDigestBody(ctx context.Context, user types.User, local time.Time) (string, error) {
	end := local.Format("2006-01-02")
	start := local.AddDate(0, 0, -6).Format("2006-01-02")

	rollups, err := s.digestStore.GetRollups(ctx, user.ID, start, end)
	if err != nil {
		return "", fmt.Errorf("get rollups: %w", err)
	}

	var days int
	var sumCal, sumProtein, sumAdherence float64
	for _, r := range rollups {
		days++
		sumCal += r.Consumed.Calories
		sumProtein += r.Consumed.Protein
		if r.Targets.Calories > 0 {
			sumAdherence += r.Consumed.Calories / r.Targets.Calories
		}
	}

	var avgCal, avgProtein, avgAdherencePct float64
	if days > 0 {
		avgCal = sumCal / float64(days)
		avgProtein = sumProtein / float64(days)
		avgAdherencePct = (sumAdherence / float64(days)) * 100
	}

	weightNote := "no weigh-ins logged"
	if weights, err := s.digestStore.ListWeight(ctx, user.ID, 7); err == nil {
		switch len(weights) {
		case 0:
			// keep default
		case 1:
			weightNote = fmt.Sprintf("weight %.1f kg (single entry)", weights[0].WeightKg)
		default:
			delta := weights[len(weights)-1].WeightKg - weights[0].WeightKg
			weightNote = fmt.Sprintf("weight %+.1f kg", delta)
		}
	}

	body := fmt.Sprintf(
		"Weekly digest: avg %.0f kcal/%.0f g protein (%.0f%% of target), %s.",
		avgCal, avgProtein, avgAdherencePct, weightNote,
	)

	waterDaysUnder := 0
	if waterTotals, err := s.digestStore.GetWaterDailyTotals(ctx, user.ID, start, end); err == nil {
		for _, wt := range waterTotals {
			// ponytail: hardcoded 2000ml goal; should become a per-user setting
			if wt.TotalML < 2000 {
				waterDaysUnder++
			}
		}
		body += fmt.Sprintf(" %d/%d days under 2000ml water.", waterDaysUnder, 7)
	}

	if workouts, err := s.digestStore.ListWorkoutsInRange(ctx, user.ID, start, end); err == nil {
		body += fmt.Sprintf(" %d workouts logged.", len(workouts))
	}

	return body, nil
}

// parseLoggedAt attempts to parse a timestamp string stored in a WaterLog,
// Workout, or SleepLog. It tries the internal store format first, then RFC
// 3339, and finally a bare date.
func parseLoggedAt(s string) (time.Time, error) {
	formats := []string{
		"2006-01-02 15:04:05", // internal store format (utcStr)
		time.RFC3339,
		"2006-01-02",
	}
	for _, f := range formats {
		if t, err := time.Parse(f, s); err == nil {
			return t, nil
		}
	}
	return time.Time{}, fmt.Errorf("cannot parse timestamp %q", s)
}

// locFor resolves a user's timezone, falling back to the default.
func (s *Scheduler) locFor(user types.User) *time.Location {
	if user.Timezone != "" {
		if loc, err := time.LoadLocation(user.Timezone); err == nil {
			return loc
		}
	}
	return s.defaultLoc
}
