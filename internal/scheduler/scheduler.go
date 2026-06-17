// Package scheduler periodically checks each user's progress against their
// daily macro targets and fires nudges when they fall behind. It is the
// component that addresses the project's core problem: a bulking user missing
// meals. Evaluation is timezone-correct (per the user's local day) and
// deduplicated so a given rule nudges at most once per local day.
package scheduler

import (
	"context"
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

// Scheduler evaluates rules on a fixed interval.
type Scheduler struct {
	store      Store
	nudges     NudgeStore
	notifier   Notifier
	rules      []Rule
	defaultLoc *time.Location
	interval   time.Duration

	now func() time.Time
	log *slog.Logger
}

// New builds a Scheduler. defaultLoc is used for users without an explicit
// timezone; interval is the tick period (e.g. 5 minutes).
func New(store Store, nudges NudgeStore, notifier Notifier, rules []Rule, defaultLoc *time.Location, interval time.Duration) *Scheduler {
	if defaultLoc == nil {
		defaultLoc = time.UTC
	}
	if interval <= 0 {
		interval = 5 * time.Minute
	}
	return &Scheduler{
		store:      store,
		nudges:     nudges,
		notifier:   notifier,
		rules:      rules,
		defaultLoc: defaultLoc,
		interval:   interval,
		now:        time.Now,
		log:        slog.Default(),
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

// evalUser checks all rules for one user at the given instant.
func (s *Scheduler) evalUser(ctx context.Context, now time.Time, user types.User) {
	local := now.In(s.locFor(user))
	date := local.Format("2006-01-02")

	targets, err := s.store.GetTargets(ctx, user.ID)
	if err != nil {
		return // no goals set: nothing to nudge against
	}
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

// locFor resolves a user's timezone, falling back to the default.
func (s *Scheduler) locFor(user types.User) *time.Location {
	if user.Timezone != "" {
		if loc, err := time.LoadLocation(user.Timezone); err == nil {
			return loc
		}
	}
	return s.defaultLoc
}
