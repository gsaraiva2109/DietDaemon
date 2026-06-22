package scheduler

import (
	"context"
	"testing"
	"time"

	"github.com/gsaraiva2109/dietdaemon/core/types"
)

type fakeStore struct {
	users   []types.User
	targets map[string]types.Macros
	rollups map[string]types.Macros // key: userID|date
}

func (f *fakeStore) ListUsers(context.Context) ([]types.User, error) { return f.users, nil }
func (f *fakeStore) GetTargets(_ context.Context, userID string) (types.DailyTargets, error) {
	if m, ok := f.targets[userID]; ok {
		return types.DailyTargets{UserID: userID, Targets: m}, nil
	}
	return types.DailyTargets{}, types.ErrNotFound
}
func (f *fakeStore) GetRollup(_ context.Context, userID, date string) (types.DailyRollup, error) {
	if m, ok := f.rollups[userID+"|"+date]; ok {
		return types.DailyRollup{UserID: userID, Date: date, Consumed: m}, nil
	}
	return types.DailyRollup{}, types.ErrNotFound
}

type fakeNudges struct{ marked map[string]bool }

func newFakeNudges() *fakeNudges { return &fakeNudges{marked: map[string]bool{}} }
func key(u, d, r string) string  { return u + "|" + d + "|" + r }
func (f *fakeNudges) WasNudged(_ context.Context, u, d, r string) (bool, error) {
	return f.marked[key(u, d, r)], nil
}
func (f *fakeNudges) MarkNudged(_ context.Context, u, d, r string) error {
	f.marked[key(u, d, r)] = true
	return nil
}

type fakeNotifier struct{ sent []types.Notification }

func (f *fakeNotifier) Notify(_ context.Context, n types.Notification) error {
	f.sent = append(f.sent, n)
	return nil
}

func proteinRule() []Rule {
	return []Rule{{ID: "protein-evening", AfterHour: 20, Macro: MacroProtein, MinFraction: 0.8, Message: "p %.0f/%.0f"}}
}

func newSched(st Store, nd NudgeStore, nt Notifier) *Scheduler {
	return New(st, nd, nt, proteinRule(), time.UTC, time.Minute)
}

// --- Macro rule tests ---

func TestFiresWhenBehindAndDedupes(t *testing.T) {
	st := &fakeStore{
		users:   []types.User{{ID: "u1", Timezone: "UTC"}},
		targets: map[string]types.Macros{"u1": {Protein: 180}},
		rollups: map[string]types.Macros{"u1|2026-06-17": {Protein: 100}}, // 100/180 = 0.55 < 0.8
	}
	nd, nt := newFakeNudges(), &fakeNotifier{}
	s := newSched(st, nd, nt)

	evening := time.Date(2026, 6, 17, 21, 0, 0, 0, time.UTC)
	s.tick(context.Background(), evening)
	if len(nt.sent) != 1 {
		t.Fatalf("nudges sent = %d, want 1", len(nt.sent))
	}
	// Second tick same day must dedupe.
	s.tick(context.Background(), evening)
	if len(nt.sent) != 1 {
		t.Errorf("dedupe failed: nudges sent = %d, want still 1", len(nt.sent))
	}
}

func TestNoFireBeforeHour(t *testing.T) {
	st := &fakeStore{
		users:   []types.User{{ID: "u1"}},
		targets: map[string]types.Macros{"u1": {Protein: 180}},
		rollups: map[string]types.Macros{"u1|2026-06-17": {Protein: 10}},
	}
	s := newSched(st, newFakeNudges(), &fakeNotifier{})
	morning := time.Date(2026, 6, 17, 9, 0, 0, 0, time.UTC)
	nt := s.notifier.(*fakeNotifier)
	s.tick(context.Background(), morning)
	if len(nt.sent) != 0 {
		t.Errorf("should not fire before AfterHour, sent %d", len(nt.sent))
	}
}

func TestNoFireWhenOnTrack(t *testing.T) {
	st := &fakeStore{
		users:   []types.User{{ID: "u1"}},
		targets: map[string]types.Macros{"u1": {Protein: 180}},
		rollups: map[string]types.Macros{"u1|2026-06-17": {Protein: 160}}, // 0.88 >= 0.8
	}
	nt := &fakeNotifier{}
	s := newSched(st, newFakeNudges(), nt)
	s.tick(context.Background(), time.Date(2026, 6, 17, 22, 0, 0, 0, time.UTC))
	if len(nt.sent) != 0 {
		t.Errorf("on track should not nudge, sent %d", len(nt.sent))
	}
}

func TestNoFireWithoutTargets(t *testing.T) {
	st := &fakeStore{users: []types.User{{ID: "u1"}}, targets: map[string]types.Macros{}, rollups: map[string]types.Macros{}}
	nt := &fakeNotifier{}
	s := newSched(st, newFakeNudges(), nt)
	s.tick(context.Background(), time.Date(2026, 6, 17, 22, 0, 0, 0, time.UTC))
	if len(nt.sent) != 0 {
		t.Errorf("no targets should not nudge, sent %d", len(nt.sent))
	}
}

// --- Health rule fakes ---

type fakeHealthStore struct {
	waterToday  int
	waterErr    error
	workouts    []types.Workout
	workoutsErr error
	activeSleep types.SleepLog
	sleepErr    error
	activeFast  types.Fast
	fastErr     error
	fasts       []types.Fast
	fastsErr    error
}

func (f *fakeHealthStore) GetWaterToday(_ context.Context, _, _ string) (int, error) {
	return f.waterToday, f.waterErr
}

func (f *fakeHealthStore) ListWorkouts(_ context.Context, _ string, _ int) ([]types.Workout, error) {
	return f.workouts, f.workoutsErr
}

func (f *fakeHealthStore) GetActiveSleep(_ context.Context, _ string) (types.SleepLog, error) {
	return f.activeSleep, f.sleepErr
}

func (f *fakeHealthStore) GetActiveFast(_ context.Context, _ string) (types.Fast, error) {
	return f.activeFast, f.fastErr
}

func (f *fakeHealthStore) ListFasts(_ context.Context, _ string, _ int) ([]types.Fast, error) {
	return f.fasts, f.fastsErr
}

func newHealthSched(st Store, hs HealthStore, nd NudgeStore, nt Notifier, healthRules []HealthRule) *Scheduler {
	return New(st, nd, nt, proteinRule(), time.UTC, time.Minute, WithHealthRules(hs, healthRules))
}

// --- Health water rule tests ---

func TestWaterAfternoonNudgesWhenBelowThreshold(t *testing.T) {
	st := &fakeStore{users: []types.User{{ID: "u1", Timezone: "UTC"}}, targets: map[string]types.Macros{"u1": {Protein: 100}}, rollups: map[string]types.Macros{"u1|2026-06-17": {Protein: 0}}}
	hs := &fakeHealthStore{waterToday: 200} // 200ml < 500 threshold
	nd := newFakeNudges()
	nt := &fakeNotifier{}

	hr := DefaultHealthRules()
	s := newHealthSched(st, hs, nd, nt, hr)

	afternoon := time.Date(2026, 6, 17, 16, 0, 0, 0, time.UTC)
	s.tick(context.Background(), afternoon)

	found := false
	for _, n := range nt.sent {
		if n.Body == hr[0].Message {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("water-afternoon should nudge when totalML=200 < 500")
	}
}

func TestWaterAfternoonSkipsWhenAboveThreshold(t *testing.T) {
	st := &fakeStore{users: []types.User{{ID: "u1", Timezone: "UTC"}}, targets: map[string]types.Macros{"u1": {Protein: 100}}, rollups: map[string]types.Macros{"u1|2026-06-17": {Protein: 0}}}
	hs := &fakeHealthStore{waterToday: 600} // above threshold
	nt := &fakeNotifier{}

	hr := DefaultHealthRules()
	s := newHealthSched(st, hs, newFakeNudges(), nt, hr)
	s.tick(context.Background(), time.Date(2026, 6, 17, 16, 0, 0, 0, time.UTC))

	for _, n := range nt.sent {
		if n.Body == hr[0].Message {
			t.Errorf("water-afternoon should NOT nudge when totalML=600 >= 500")
		}
	}
}

func TestWaterEveningNudgesWhenBelowThreshold(t *testing.T) {
	st := &fakeStore{users: []types.User{{ID: "u1", Timezone: "UTC"}}, targets: map[string]types.Macros{"u1": {Protein: 100}}, rollups: map[string]types.Macros{"u1|2026-06-17": {Protein: 0}}}
	hs := &fakeHealthStore{waterToday: 1000} // 1000ml < 1600 threshold
	nt := &fakeNotifier{}

	hr := DefaultHealthRules()
	s := newHealthSched(st, hs, newFakeNudges(), nt, hr)
	s.tick(context.Background(), time.Date(2026, 6, 17, 20, 0, 0, 0, time.UTC))

	found := false
	for _, n := range nt.sent {
		if n.Body == hr[1].Message {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("water-evening should nudge when totalML=1000 < 1600")
	}
}

func TestHealthRuleRespectsCheckHour(t *testing.T) {
	st := &fakeStore{users: []types.User{{ID: "u1", Timezone: "UTC"}}, targets: map[string]types.Macros{"u1": {Protein: 100}}, rollups: map[string]types.Macros{"u1|2026-06-17": {Protein: 0}}}
	hs := &fakeHealthStore{waterToday: 0} // way below threshold
	nt := &fakeNotifier{}

	hr := DefaultHealthRules()
	s := newHealthSched(st, hs, newFakeNudges(), nt, hr)
	// Before water-afternoon's CheckHour (16)
	s.tick(context.Background(), time.Date(2026, 6, 17, 15, 0, 0, 0, time.UTC))

	// water-afternoon and water-evening should NOT fire before their respective CheckHours
	for _, n := range nt.sent {
		if n.Body == hr[0].Message || n.Body == hr[1].Message {
			t.Errorf("health rules should not fire before CheckHour, got sent: %q", n.Body)
		}
	}
}

// --- Health workout rule tests ---

func TestWorkoutReminderWhenNoWorkouts(t *testing.T) {
	st := &fakeStore{users: []types.User{{ID: "u1", Timezone: "UTC"}}, targets: map[string]types.Macros{"u1": {Protein: 100}}, rollups: map[string]types.Macros{"u1|2026-06-17": {Protein: 0}}}
	hs := &fakeHealthStore{workouts: nil} // no workouts ever
	nt := &fakeNotifier{}

	hr := DefaultHealthRules()
	s := newHealthSched(st, hs, newFakeNudges(), nt, hr)
	s.tick(context.Background(), time.Date(2026, 6, 17, 12, 0, 0, 0, time.UTC))

	found := false
	for _, n := range nt.sent {
		if n.Body == hr[2].Message {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("workout-reminder should nudge when no workouts exist")
	}
}

func TestWorkoutReminderWhenLastWorkoutOld(t *testing.T) {
	st := &fakeStore{users: []types.User{{ID: "u1", Timezone: "UTC"}}, targets: map[string]types.Macros{"u1": {Protein: 100}}, rollups: map[string]types.Macros{"u1|2026-06-17": {Protein: 0}}}
	// Last workout 4 days ago (2026-06-13), today is 2026-06-17
	hs := &fakeHealthStore{workouts: []types.Workout{{LoggedAt: "2026-06-13 10:00:00"}}}
	nt := &fakeNotifier{}

	hr := DefaultHealthRules()
	s := newHealthSched(st, hs, newFakeNudges(), nt, hr)
	s.tick(context.Background(), time.Date(2026, 6, 17, 12, 0, 0, 0, time.UTC))

	found := false
	for _, n := range nt.sent {
		if n.Body == hr[2].Message {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("workout-reminder should nudge when last workout >3 days ago")
	}
}

func TestWorkoutReminderSkipsWhenRecent(t *testing.T) {
	st := &fakeStore{users: []types.User{{ID: "u1", Timezone: "UTC"}}, targets: map[string]types.Macros{"u1": {Protein: 100}}, rollups: map[string]types.Macros{"u1|2026-06-17": {Protein: 0}}}
	// Last workout 1 day ago (2026-06-16), today is 2026-06-17
	hs := &fakeHealthStore{workouts: []types.Workout{{LoggedAt: "2026-06-16 10:00:00"}}}
	nt := &fakeNotifier{}

	hr := DefaultHealthRules()
	s := newHealthSched(st, hs, newFakeNudges(), nt, hr)
	s.tick(context.Background(), time.Date(2026, 6, 17, 12, 0, 0, 0, time.UTC))

	for _, n := range nt.sent {
		if n.Body == hr[2].Message {
			t.Errorf("workout-reminder should NOT nudge when last workout was 1 day ago")
		}
	}
}

// --- Health sleep rule tests ---

func TestSleepReminderWhenNoActiveSleep(t *testing.T) {
	st := &fakeStore{users: []types.User{{ID: "u1", Timezone: "UTC"}}, targets: map[string]types.Macros{"u1": {Protein: 100}}, rollups: map[string]types.Macros{"u1|2026-06-17": {Protein: 0}}}
	hs := &fakeHealthStore{sleepErr: types.ErrNotFound} // no active sleep
	nt := &fakeNotifier{}

	hr := DefaultHealthRules()
	s := newHealthSched(st, hs, newFakeNudges(), nt, hr)
	s.tick(context.Background(), time.Date(2026, 6, 17, 22, 30, 0, 0, time.UTC))

	found := false
	for _, n := range nt.sent {
		if n.Body == hr[3].Message {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("sleep-reminder should nudge when no active sleep at 22:30")
	}
}

func TestSleepReminderSkipsWhenActiveSleepExists(t *testing.T) {
	st := &fakeStore{users: []types.User{{ID: "u1", Timezone: "UTC"}}, targets: map[string]types.Macros{"u1": {Protein: 100}}, rollups: map[string]types.Macros{"u1|2026-06-17": {Protein: 0}}}
	hs := &fakeHealthStore{activeSleep: types.SleepLog{ID: "s1", UserID: "u1", SleepAt: "2026-06-17 22:00:00"}}
	nt := &fakeNotifier{}

	hr := DefaultHealthRules()
	s := newHealthSched(st, hs, newFakeNudges(), nt, hr)
	s.tick(context.Background(), time.Date(2026, 6, 17, 22, 30, 0, 0, time.UTC))

	for _, n := range nt.sent {
		if n.Body == hr[3].Message {
			t.Errorf("sleep-reminder should NOT nudge when active sleep exists")
		}
	}
}

// --- Health fasting rule tests ---

func TestFastEndingNudgesWithinWindow(t *testing.T) {
	now := time.Date(2026, 6, 17, 14, 0, 0, 0, time.UTC)
	st := &fakeStore{
		users:   []types.User{{ID: "u1", Timezone: "UTC"}},
		targets: map[string]types.Macros{"u1": {Protein: 100}},
		rollups: map[string]types.Macros{"u1|2026-06-17": {Protein: 0}},
	}
	// Fast started at 06:00, target is 8 hours → ends at 14:00. At 14:00,
	// remaining = 0, which is <= 0.5 but NOT > 0. So no nudge.
	// Start at 06:00, target 8h → remaining = 8 - 8 = 0. Not > 0, skip.
	// Let's use start at 06:00, target 8.4h (8h 24min).
	hs := &fakeHealthStore{
		activeFast: types.Fast{
			ID: "f1", UserID: "u1",
			StartAt:     time.Date(2026, 6, 17, 6, 0, 0, 0, time.UTC),
			TargetHours: 8.4, // ends at 14:24 UTC → at 14:00 remaining = 0.4h (24 min)
		},
	}
	nt := &fakeNotifier{}

	hr := DefaultHealthRules()
	s := newHealthSched(st, hs, newFakeNudges(), nt, hr)
	s.tick(context.Background(), now)

	found := false
	for _, n := range nt.sent {
		if n.Body == hr[4].Message {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("fast-ending should nudge within 30 min of target")
	}
}

func TestFastEndingSkipsWhenNotClose(t *testing.T) {
	now := time.Date(2026, 6, 17, 12, 0, 0, 0, time.UTC)
	st := &fakeStore{
		users:   []types.User{{ID: "u1", Timezone: "UTC"}},
		targets: map[string]types.Macros{"u1": {Protein: 100}},
		rollups: map[string]types.Macros{"u1|2026-06-17": {Protein: 0}},
	}
	// Fast started at 06:00, target 8h → ends at 14:00. At 12:00, remaining = 2h.
	hs := &fakeHealthStore{
		activeFast: types.Fast{
			ID: "f1", UserID: "u1",
			StartAt:     time.Date(2026, 6, 17, 6, 0, 0, 0, time.UTC),
			TargetHours: 8,
		},
	}
	nt := &fakeNotifier{}

	hr := DefaultHealthRules()
	s := newHealthSched(st, hs, newFakeNudges(), nt, hr)
	s.tick(context.Background(), now)

	for _, n := range nt.sent {
		if n.Body == hr[4].Message {
			t.Errorf("fast-ending should NOT nudge when remaining is 2h")
		}
	}
}

func TestFastEndingSkipsWhenNoActiveFast(t *testing.T) {
	now := time.Date(2026, 6, 17, 14, 0, 0, 0, time.UTC)
	st := &fakeStore{
		users:   []types.User{{ID: "u1", Timezone: "UTC"}},
		targets: map[string]types.Macros{"u1": {Protein: 100}},
		rollups: map[string]types.Macros{"u1|2026-06-17": {Protein: 0}},
	}
	hs := &fakeHealthStore{fastErr: types.ErrNotFound} // no active fast
	nt := &fakeNotifier{}

	hr := DefaultHealthRules()
	s := newHealthSched(st, hs, newFakeNudges(), nt, hr)
	s.tick(context.Background(), now)

	for _, n := range nt.sent {
		if n.Body == hr[4].Message {
			t.Errorf("fast-ending should NOT nudge when no active fast")
		}
	}
}

// --- Health deduplication tests ---

func TestHealthRuleDedupe(t *testing.T) {
	st := &fakeStore{users: []types.User{{ID: "u1", Timezone: "UTC"}}, targets: map[string]types.Macros{"u1": {Protein: 100}}, rollups: map[string]types.Macros{"u1|2026-06-17": {Protein: 0}}}
	hs := &fakeHealthStore{waterToday: 0} // below threshold
	nd := newFakeNudges()
	nt := &fakeNotifier{}

	hr := DefaultHealthRules()
	s := newHealthSched(st, hs, nd, nt, hr)

	afternoon := time.Date(2026, 6, 17, 17, 0, 0, 0, time.UTC)
	s.tick(context.Background(), afternoon)
	if len(nt.sent) == 0 {
		t.Fatal("expected at least one nudge")
	}
	firstCount := len(nt.sent)

	// Second tick same day must dedupe.
	s.tick(context.Background(), afternoon)
	if len(nt.sent) != firstCount {
		t.Errorf("health dedupe failed: sent %d, want still %d", len(nt.sent), firstCount)
	}
}

// --- Health rule when no macro targets ---

func TestHealthRulesFireWithoutMacroTargets(t *testing.T) {
	st := &fakeStore{users: []types.User{{ID: "u1", Timezone: "UTC"}}, targets: map[string]types.Macros{}, rollups: map[string]types.Macros{}}
	hs := &fakeHealthStore{waterToday: 0} // below threshold
	nd := newFakeNudges()
	nt := &fakeNotifier{}

	hr := DefaultHealthRules()
	s := newHealthSched(st, hs, nd, nt, hr)

	afternoon := time.Date(2026, 6, 17, 17, 0, 0, 0, time.UTC)
	s.tick(context.Background(), afternoon)

	// Should still get water nudge even though no macro targets
	found := false
	for _, n := range nt.sent {
		if n.Body == hr[0].Message {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("health rules should fire even when no macro targets set")
	}
}
