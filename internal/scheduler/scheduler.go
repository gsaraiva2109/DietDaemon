// Package scheduler periodically checks each user's progress against their
// daily macro targets and fires nudges when they fall behind. It is the
// component that addresses the project's core problem: a bulking user missing
// meals. Evaluation is timezone-correct (per the user's local day) and
// deduplicated so a given rule nudges at most once per local day.
package scheduler

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
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
	// GetWaterToday returns the total millilitres logged for the given local
	// date, or 0 and nil if none found.
	GetWaterToday(ctx context.Context, userID, localDate string) (totalML int, err error)

	// ListWorkouts returns the most recent workouts, newest first.
	ListWorkouts(ctx context.Context, userID string, limit int) ([]types.Workout, error)

	// GetActiveSleep returns the user's in-progress sleep (wake_at IS NULL), or
	// types.ErrNotFound if none is active.
	GetActiveSleep(ctx context.Context, userID string) (types.SleepLog, error)

	// GetActiveFast returns the user's in-progress fast (end_at IS NULL), or
	// types.ErrNotFound if none is active.
	GetActiveFast(ctx context.Context, userID string) (types.Fast, error)

	// ListFasts returns the user's most recent fasting windows, newest first.
	ListFasts(ctx context.Context, userID string, limit int) ([]types.Fast, error)
}

// Option configures a Scheduler. Used with the variadic New constructor.
type Option func(*Scheduler)

// Scheduler evaluates rules on a fixed interval.
type Scheduler struct {
	store       Store
	nudges      NudgeStore
	notifier    Notifier
	rules       []Rule
	healthStore HealthStore
	healthRules []HealthRule
	defaultLoc  *time.Location
	interval    time.Duration

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

// Run ticks until ctx is cancelled, evaluating immediately on start.
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

	// Macro rules (require targets).
	targets, err := s.store.GetTargets(ctx, user.ID)
	if err == nil {
		rollup, err := s.store.GetRollup(ctx, user.ID, date)
		if err != nil {
			rollup = types.DailyRollup{} // no meals logged yet today
		}

		for _, r := range s.rules {
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
			if err := s.notifier.Notify(ctx, n); err != nil {
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
		s.evalHealthRules(ctx, now, user)
	}
}

// evalHealthRules evaluates every health rule for one user at the given
// instant. It uses the same nudge_log table for deduplication, keyed by
// (user_id, local_date, rule_id), so health rule IDs like "water-afternoon"
// coexist safely with macro rule IDs.
func (s *Scheduler) evalHealthRules(ctx context.Context, now time.Time, user types.User) {
	local := now.In(s.locFor(user))
	date := local.Format("2006-01-02")
	hour := local.Hour()

	for _, r := range s.healthRules {
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
			totalML, err := s.healthStore.GetWaterToday(ctx, user.ID, date)
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
		if err := s.notifier.Notify(ctx, n); err != nil {
			s.log.Error("scheduler: health notify", "rule", r.ID, "err", err)
			continue // not marked: retry next tick
		}
		if err := s.nudges.MarkNudged(ctx, user.ID, date, r.ID); err != nil {
			s.log.Error("scheduler: health mark-nudged", "rule", r.ID, "err", err)
		}
	}
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
