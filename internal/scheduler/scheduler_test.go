package scheduler

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"sync/atomic"
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

type fakeNotifier struct {
	sent []types.Notification
	err  error // when set, Notify returns it instead of recording
}

func (f *fakeNotifier) Notify(_ context.Context, n types.Notification) error {
	if f.err != nil {
		return f.err
	}
	f.sent = append(f.sent, n)
	return nil
}

type fakeMealHistory struct{ times []time.Time }

func (f *fakeMealHistory) RecentMealTimes(_ context.Context, _ string, _ time.Time) ([]time.Time, error) {
	return f.times, nil
}

func TestLearnedMealHoursThresholdAndCap(t *testing.T) {
	var times []time.Time
	for day := 1; day <= 7; day++ {
		for _, hour := range []int{8, 12, 18, 21} {
			times = append(times, time.Date(2026, 6, day, hour, 0, 0, 0, time.UTC))
		}
	}
	hours := learnedMealHours(times, time.UTC)
	if len(hours) != 3 || hours[0] != 8 || hours[1] != 12 || hours[2] != 18 {
		t.Fatalf("hours = %v, want top three in hour order on tie", hours)
	}
	if got := learnedMealHours(times[:6], time.UTC); got != nil {
		t.Fatalf("insufficient days = %v, want nil", got)
	}
}

func TestSmartMealReminderMidnightDedupes(t *testing.T) {
	var times []time.Time
	for day := 1; day <= 7; day++ {
		times = append(times, time.Date(2026, 6, day, 0, 0, 0, 0, time.UTC))
	}
	st := &fakeStore{users: []types.User{{ID: "u1", Timezone: "UTC"}}, targets: map[string]types.Macros{}, rollups: map[string]types.Macros{}}
	nd, nt := newFakeNudges(), &fakeNotifier{}
	s := New(st, nd, nt, nil, time.UTC, time.Minute, WithSmartMealRules(&fakeMealHistory{times: times}, DefaultSmartMealRules()))
	now := time.Date(2026, 6, 8, 23, 30, 0, 0, time.UTC)
	s.tick(context.Background(), now)
	s.tick(context.Background(), now)
	if len(nt.sent) != 1 {
		t.Fatalf("sent = %d, want one", len(nt.sent))
	}
	if !nd.marked["u1|2026-06-09|smart-meal-reminders-00"] {
		t.Fatalf("missing next-local-day midnight dedupe key: %#v", nd.marked)
	}
}

func proteinRule() []Rule {
	return []Rule{{ID: "protein-evening", AfterHour: 20, Macro: MacroProtein, MinFraction: 0.8, Message: "p %.0f/%.0f"}}
}

func newSched(st Store, nd NudgeStore, nt Notifier) *Scheduler {
	return New(st, nd, nt, proteinRule(), time.UTC, time.Minute)
}

type blockingStore struct {
	users   []types.User
	started chan struct{}
	release <-chan struct{}
	calls   atomic.Int32
}

func (s *blockingStore) ListUsers(context.Context) ([]types.User, error) { return s.users, nil }
func (s *blockingStore) GetTargets(context.Context, string) (types.DailyTargets, error) {
	s.calls.Add(1)
	s.started <- struct{}{}
	<-s.release
	return types.DailyTargets{}, types.ErrNotFound
}
func (*blockingStore) GetRollup(context.Context, string, string) (types.DailyRollup, error) {
	return types.DailyRollup{}, types.ErrNotFound
}

func TestTickBoundsConcurrentUserEvaluation(t *testing.T) {
	users := make([]types.User, schedulerWorkers+1)
	for i := range users {
		users[i].ID = string(rune('a' + i))
	}
	release := make(chan struct{})
	st := &blockingStore{users: users, started: make(chan struct{}, len(users)), release: release}
	s := New(st, newFakeNudges(), &fakeNotifier{}, nil, time.UTC, time.Minute)
	done := make(chan struct{})
	go func() {
		s.tick(context.Background(), time.Now())
		close(done)
	}()
	for range schedulerWorkers {
		<-st.started
	}
	select {
	case <-st.started:
		t.Fatal("evaluated more users than the worker limit")
	default:
	}
	close(release)
	<-done
	if got := int(st.calls.Load()); got != len(users) {
		t.Fatalf("evaluated %d users, want %d", got, len(users))
	}
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

func (f *fakeHealthStore) GetWaterToday(_ context.Context, _, _ string) ([]types.WaterLog, int, error) {
	return nil, f.waterToday, f.waterErr
}

func (f *fakeHealthStore) ListWorkouts(_ context.Context, _ string, _ int) ([]types.Workout, error) {
	return f.workouts, f.workoutsErr
}

func (f *fakeHealthStore) GetActiveSleep(_ context.Context, _ string) (*types.SleepLog, error) {
	if f.sleepErr != nil {
		return nil, f.sleepErr
	}
	return &f.activeSleep, nil
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

// --- Rule config (per-user overrides) tests ---

type fakeRuleConfigStore struct {
	configs map[string][]types.NudgeRuleConfig // userID -> overrides
}

func (f *fakeRuleConfigStore) GetNudgeRuleConfig(_ context.Context, userID string) ([]types.NudgeRuleConfig, error) {
	return f.configs[userID], nil
}

func TestRuleConfigDisableSkipsRule(t *testing.T) {
	st := &fakeStore{
		users:   []types.User{{ID: "u1", Timezone: "UTC"}},
		targets: map[string]types.Macros{"u1": {Protein: 180}},
		rollups: map[string]types.Macros{"u1|2026-06-17": {Protein: 100}}, // 100/180=0.55 < 0.8, would fire
	}
	rcs := &fakeRuleConfigStore{configs: map[string][]types.NudgeRuleConfig{
		"u1": {{UserID: "u1", RuleID: "protein-evening", Enabled: false}},
	}}
	nt := &fakeNotifier{}
	s := New(st, newFakeNudges(), nt, proteinRule(), time.UTC, time.Minute, WithRuleConfig(rcs))

	s.tick(context.Background(), time.Date(2026, 6, 17, 21, 0, 0, 0, time.UTC))
	if len(nt.sent) != 0 {
		t.Errorf("disabled override should skip rule entirely, sent %d", len(nt.sent))
	}
}

func TestRuleConfigParamOverrideChangesFiring(t *testing.T) {
	st := &fakeStore{
		users:   []types.User{{ID: "u1", Timezone: "UTC"}},
		targets: map[string]types.Macros{"u1": {Protein: 180}},
		rollups: map[string]types.Macros{"u1|2026-06-17": {Protein: 100}}, // 100/180=0.55
	}
	// Lower MinFraction to 0.5: 0.55 >= 0.5 now counts as "on track", so the
	// rule that would otherwise fire (default MinFraction 0.8) must not.
	rcs := &fakeRuleConfigStore{configs: map[string][]types.NudgeRuleConfig{
		"u1": {{UserID: "u1", RuleID: "protein-evening", Enabled: true, Params: json.RawMessage(`{"MinFraction":0.5}`)}},
	}}
	nt := &fakeNotifier{}
	s := New(st, newFakeNudges(), nt, proteinRule(), time.UTC, time.Minute, WithRuleConfig(rcs))

	s.tick(context.Background(), time.Date(2026, 6, 17, 21, 0, 0, 0, time.UTC))
	if len(nt.sent) != 0 {
		t.Errorf("param override lowering MinFraction to already-met should suppress the nudge, sent %d", len(nt.sent))
	}
}

func TestRuleConfigMissingOverrideRunsDefault(t *testing.T) {
	// Backward compatibility: a RuleConfigStore configured but with no row for
	// this user/rule must behave identically to no RuleConfigStore at all.
	st := &fakeStore{
		users:   []types.User{{ID: "u1", Timezone: "UTC"}},
		targets: map[string]types.Macros{"u1": {Protein: 180}},
		rollups: map[string]types.Macros{"u1|2026-06-17": {Protein: 100}},
	}
	rcs := &fakeRuleConfigStore{configs: map[string][]types.NudgeRuleConfig{}}
	nt := &fakeNotifier{}
	s := New(st, newFakeNudges(), nt, proteinRule(), time.UTC, time.Minute, WithRuleConfig(rcs))

	s.tick(context.Background(), time.Date(2026, 6, 17, 21, 0, 0, 0, time.UTC))
	if len(nt.sent) != 1 {
		t.Errorf("no override row should run the rule with hardcoded defaults, sent %d", len(nt.sent))
	}
}

// --- Weekly digest tests ---

type fakeDigestStore struct {
	rollups     []types.DailyRollup
	weights     []types.WeightEntry
	waterTotals []types.WaterDayTotal
	workouts    []types.Workout
}

func (f *fakeDigestStore) GetRollups(_ context.Context, _, _, _ string) ([]types.DailyRollup, error) {
	return f.rollups, nil
}
func (f *fakeDigestStore) ListWeight(_ context.Context, _ string, _ int) ([]types.WeightEntry, error) {
	return f.weights, nil
}
func (f *fakeDigestStore) GetWaterDailyTotals(_ context.Context, _, _, _ string) ([]types.WaterDayTotal, error) {
	return f.waterTotals, nil
}
func (f *fakeDigestStore) ListWorkoutsInRange(_ context.Context, _, _, _ string) ([]types.Workout, error) {
	return f.workouts, nil
}

func TestWeeklyDigestFiresOnceThenDedupesSameISOWeek(t *testing.T) {
	st := &fakeStore{users: []types.User{{ID: "u1", Timezone: "UTC"}}, targets: map[string]types.Macros{}, rollups: map[string]types.Macros{}}
	ds := &fakeDigestStore{
		rollups: []types.DailyRollup{{Consumed: types.Macros{Calories: 2000, Protein: 150}, Targets: types.Macros{Calories: 2200}}},
		weights: []types.WeightEntry{{WeightKg: 80}, {WeightKg: 79.5}},
	}
	nt := &fakeNotifier{}
	s := New(st, newFakeNudges(), nt, nil, time.UTC, time.Minute, WithDigestRules(ds, DefaultDigestRules()))

	// 2026-06-21 is a Sunday.
	sunday9am := time.Date(2026, 6, 21, 9, 0, 0, 0, time.UTC)
	s.tick(context.Background(), sunday9am)
	if len(nt.sent) != 1 {
		t.Fatalf("digest sent = %d, want 1", len(nt.sent))
	}

	// A later tick the same day, and one on the following Sunday's date but
	// still checked before that ISO week actually starts, must both dedupe.
	s.tick(context.Background(), sunday9am.Add(2*time.Hour))
	if len(nt.sent) != 1 {
		t.Errorf("digest should dedupe within the same ISO week, sent %d", len(nt.sent))
	}

	nextSunday := sunday9am.AddDate(0, 0, 7)
	s.tick(context.Background(), nextSunday)
	if len(nt.sent) != 2 {
		t.Errorf("digest should fire again the following ISO week, sent %d", len(nt.sent))
	}
}

func TestWeeklyDigestSkipsBeforeCheckHourOrWrongWeekday(t *testing.T) {
	st := &fakeStore{users: []types.User{{ID: "u1", Timezone: "UTC"}}, targets: map[string]types.Macros{}, rollups: map[string]types.Macros{}}
	ds := &fakeDigestStore{}
	nt := &fakeNotifier{}
	s := New(st, newFakeNudges(), nt, nil, time.UTC, time.Minute, WithDigestRules(ds, DefaultDigestRules()))

	// Sunday but before CheckHour (9).
	s.tick(context.Background(), time.Date(2026, 6, 21, 8, 0, 0, 0, time.UTC))
	// Monday at the check hour.
	s.tick(context.Background(), time.Date(2026, 6, 22, 9, 0, 0, 0, time.UTC))
	if len(nt.sent) != 0 {
		t.Errorf("digest should not fire before CheckHour or on the wrong weekday, sent %d", len(nt.sent))
	}
}

func TestWeeklyDigestDisabledOverrideSkips(t *testing.T) {
	st := &fakeStore{users: []types.User{{ID: "u1", Timezone: "UTC"}}, targets: map[string]types.Macros{}, rollups: map[string]types.Macros{}}
	ds := &fakeDigestStore{}
	rcs := &fakeRuleConfigStore{configs: map[string][]types.NudgeRuleConfig{
		"u1": {{UserID: "u1", RuleID: "weekly-digest", Enabled: false}},
	}}
	nt := &fakeNotifier{}
	s := New(st, newFakeNudges(), nt, nil, time.UTC, time.Minute, WithDigestRules(ds, DefaultDigestRules()), WithRuleConfig(rcs))

	s.tick(context.Background(), time.Date(2026, 6, 21, 9, 0, 0, 0, time.UTC))
	if len(nt.sent) != 0 {
		t.Errorf("disabled digest override should skip it, sent %d", len(nt.sent))
	}
}

func TestWeeklyDigestMentionsMissedWaterDay(t *testing.T) {
	st := &fakeStore{users: []types.User{{ID: "u1", Timezone: "UTC"}}, targets: map[string]types.Macros{}, rollups: map[string]types.Macros{}}
	ds := &fakeDigestStore{
		rollups: []types.DailyRollup{
			{Date: "2026-06-15", Consumed: types.Macros{Calories: 2000, Protein: 150}, Targets: types.Macros{Calories: 2200}},
			{Date: "2026-06-16", Consumed: types.Macros{Calories: 1800, Protein: 120}, Targets: types.Macros{Calories: 2200}},
		},
		weights: []types.WeightEntry{{WeightKg: 80}, {WeightKg: 79.5}},
		waterTotals: []types.WaterDayTotal{
			{Date: "2026-06-15", TotalML: 1500}, // under 2000ml
			{Date: "2026-06-16", TotalML: 2500}, // fine
		},
	}
	nt := &fakeNotifier{}
	s := New(st, newFakeNudges(), nt, nil, time.UTC, time.Minute, WithDigestRules(ds, DefaultDigestRules()))

	s.tick(context.Background(), time.Date(2026, 6, 21, 9, 0, 0, 0, time.UTC))
	if len(nt.sent) != 1 {
		t.Fatalf("digest sent = %d, want 1", len(nt.sent))
	}
	if !strings.Contains(nt.sent[0].Body, "under 2000ml") {
		t.Errorf("digest body should mention missed-water day, got: %q", nt.sent[0].Body)
	}
}

// --- Chat delivery tests ---

type fakeChatRouteStore struct {
	routes map[string]string // userID -> channel; "" meta always {"chat_id": userID+"-chat"}
}

func (f *fakeChatRouteStore) GetChatRoute(_ context.Context, userID string) (string, map[string]string, error) {
	channel, ok := f.routes[userID]
	if !ok {
		return "", nil, types.ErrNotFound
	}
	return channel, map[string]string{"chat_id": userID + "-chat"}, nil
}

type fakeChatSender struct {
	sent    []types.Reply
	sendErr error
}

func (f *fakeChatSender) Send(_ context.Context, reply types.Reply) error {
	if f.sendErr != nil {
		return f.sendErr
	}
	f.sent = append(f.sent, reply)
	return nil
}

func TestDeliverPrefersChatWhenRouteExists(t *testing.T) {
	st := &fakeStore{
		users:   []types.User{{ID: "u1", Timezone: "UTC"}},
		targets: map[string]types.Macros{"u1": {Protein: 180}},
		rollups: map[string]types.Macros{"u1|2026-06-17": {Protein: 100}},
	}
	nt := &fakeNotifier{}
	routes := &fakeChatRouteStore{routes: map[string]string{"u1": "telegram"}}
	sender := &fakeChatSender{}
	s := New(st, newFakeNudges(), nt, proteinRule(), time.UTC, time.Minute, WithChatSender(routes, sender))

	s.tick(context.Background(), time.Date(2026, 6, 17, 21, 0, 0, 0, time.UTC))

	if len(sender.sent) != 1 {
		t.Fatalf("chat sends = %d, want 1", len(sender.sent))
	}
	if len(nt.sent) != 0 {
		t.Errorf("notifier should not be used when chat delivery succeeds, sent %d", len(nt.sent))
	}
}

func TestDeliverFallsBackToNotifierWhenNoRoute(t *testing.T) {
	st := &fakeStore{
		users:   []types.User{{ID: "u1", Timezone: "UTC"}},
		targets: map[string]types.Macros{"u1": {Protein: 180}},
		rollups: map[string]types.Macros{"u1|2026-06-17": {Protein: 100}},
	}
	nt := &fakeNotifier{}
	routes := &fakeChatRouteStore{routes: map[string]string{}} // no route for u1
	sender := &fakeChatSender{}
	s := New(st, newFakeNudges(), nt, proteinRule(), time.UTC, time.Minute, WithChatSender(routes, sender))

	s.tick(context.Background(), time.Date(2026, 6, 17, 21, 0, 0, 0, time.UTC))

	if len(nt.sent) != 1 {
		t.Errorf("notifier sends = %d, want 1 (fallback)", len(nt.sent))
	}
	if len(sender.sent) != 0 {
		t.Errorf("chat sender should not be used without a route, sent %d", len(sender.sent))
	}
}

func TestDeliverFallsBackToNotifierOnChatSendError(t *testing.T) {
	st := &fakeStore{
		users:   []types.User{{ID: "u1", Timezone: "UTC"}},
		targets: map[string]types.Macros{"u1": {Protein: 180}},
		rollups: map[string]types.Macros{"u1|2026-06-17": {Protein: 100}},
	}
	nt := &fakeNotifier{}
	routes := &fakeChatRouteStore{routes: map[string]string{"u1": "telegram"}}
	sender := &fakeChatSender{sendErr: errors.New("boom")}
	s := New(st, newFakeNudges(), nt, proteinRule(), time.UTC, time.Minute, WithChatSender(routes, sender))

	s.tick(context.Background(), time.Date(2026, 6, 17, 21, 0, 0, 0, time.UTC))

	if len(nt.sent) != 1 {
		t.Errorf("notifier sends = %d, want 1 (fallback on chat error)", len(nt.sent))
	}
}

// --- Production wiring test ---
//
// fakeFullStore satisfies every scheduler collaborator interface in one
// value, mirroring how *store.Store satisfies Store, NudgeStore, HealthStore,
// RuleConfigStore, and DigestStore simultaneously. This test calls New with
// the exact option shape used in cmd/dietdaemon/main.go, so a regression that
// breaks that construction call (the historical bug: health rules never
// wired, or an interface mismatch that fails to compile) is caught here
// rather than only in isolated fakes.
type fakeFullStore struct {
	*fakeStore
	*fakeNudges
	*fakeHealthStore
	*fakeRuleConfigStore
	*fakeDigestStore
	*fakeChatRouteStore
}

func TestHealthRulesFireThroughRealConstructionPath(t *testing.T) {
	full := &fakeFullStore{
		fakeStore: &fakeStore{
			users:   []types.User{{ID: "u1", Timezone: "UTC"}},
			targets: map[string]types.Macros{},
			rollups: map[string]types.Macros{},
		},
		fakeNudges:          newFakeNudges(),
		fakeHealthStore:     &fakeHealthStore{waterToday: 0}, // below every water threshold
		fakeRuleConfigStore: &fakeRuleConfigStore{},
		fakeDigestStore:     &fakeDigestStore{},
		fakeChatRouteStore:  &fakeChatRouteStore{routes: map[string]string{}}, // no route: falls back to notifier
	}
	nt := &fakeNotifier{}
	sender := &fakeChatSender{}

	// Same call shape as cmd/dietdaemon/main.go's scheduler.New(...).
	sched := New(full, full, nt, DefaultRules(), time.UTC, time.Minute,
		WithHealthRules(full, DefaultHealthRules()),
		WithRuleConfig(full),
		WithDigestRules(full, DefaultDigestRules()),
		WithChatSender(full, sender),
	)

	afternoon := time.Date(2026, 6, 17, 16, 0, 0, 0, time.UTC)
	sched.tick(context.Background(), afternoon)

	hr := DefaultHealthRules()
	found := false
	for _, n := range nt.sent {
		if n.Body == hr[0].Message { // water-afternoon
			found = true
			break
		}
	}
	if !found {
		t.Errorf("health rules did not fire through the production scheduler.New(...) construction path")
	}
}

// --- Quick-action buttons ---

type fakeSentNudgeStore struct {
	recorded []types.SentNudge
}

func (f *fakeSentNudgeStore) RecordSentNudge(_ context.Context, n types.SentNudge) error {
	f.recorded = append(f.recorded, n)
	return nil
}

func TestWaterRuleQuickActionInMarkup(t *testing.T) {
	st := &fakeStore{
		users:   []types.User{{ID: "u1", Timezone: "UTC"}},
		targets: map[string]types.Macros{"u1": {Protein: 100}},
		rollups: map[string]types.Macros{"u1|2026-06-17": {Protein: 0}},
	}
	hs := &fakeHealthStore{waterToday: 0} // below 500ml threshold
	nd := newFakeNudges()
	nt := &fakeNotifier{}
	routes := &fakeChatRouteStore{routes: map[string]string{"u1": "telegram"}}
	sender := &fakeChatSender{}
	sns := &fakeSentNudgeStore{}

	hr := DefaultHealthRules()
	s := New(st, nd, nt, nil, time.UTC, time.Minute,
		WithHealthRules(hs, hr),
		WithChatSender(routes, sender),
		WithSentNudges(sns),
	)

	afternoon := time.Date(2026, 6, 17, 16, 0, 0, 0, time.UTC)
	s.tick(context.Background(), afternoon)

	if len(sender.sent) != 1 {
		t.Fatalf("chat sends = %d, want 1", len(sender.sent))
	}
	markup := sender.sent[0].Markup
	if markup == nil {
		t.Fatal("expected markup on chat reply")
	}
	if len(markup.InlineKeyboard) != 1 {
		t.Fatalf("inline keyboard rows = %d, want 1", len(markup.InlineKeyboard))
	}
	buttons := markup.InlineKeyboard[0]

	// Water rule should get water quick action + Undo button.
	hasWater := false
	hasUndo := false
	for _, btn := range buttons {
		if btn.Text == "Log 500ml water" && btn.CallbackData == "/water 500" {
			hasWater = true
		}
		if strings.Contains(btn.CallbackData, "/nudge undo ") {
			hasUndo = true
		}
	}
	if !hasWater {
		t.Error("expected water quick-action button in markup")
	}
	if !hasUndo {
		t.Error("expected Undo button in markup")
	}
}

func TestMacroRuleUndoOnlyMarkup(t *testing.T) {
	st := &fakeStore{
		users:   []types.User{{ID: "u1", Timezone: "UTC"}},
		targets: map[string]types.Macros{"u1": {Protein: 180}},
		rollups: map[string]types.Macros{"u1|2026-06-17": {Protein: 100}}, // behind on protein
	}
	nd := newFakeNudges()
	nt := &fakeNotifier{}
	routes := &fakeChatRouteStore{routes: map[string]string{"u1": "telegram"}}
	sender := &fakeChatSender{}
	sns := &fakeSentNudgeStore{}

	s := New(st, nd, nt, proteinRule(), time.UTC, time.Minute,
		WithChatSender(routes, sender),
		WithSentNudges(sns),
	)

	evening := time.Date(2026, 6, 17, 21, 0, 0, 0, time.UTC)
	s.tick(context.Background(), evening)

	if len(sender.sent) != 1 {
		t.Fatalf("chat sends = %d, want 1", len(sender.sent))
	}
	markup := sender.sent[0].Markup
	if markup == nil {
		t.Fatal("expected markup on chat reply")
	}
	if len(markup.InlineKeyboard) != 1 {
		t.Fatalf("inline keyboard rows = %d, want 1", len(markup.InlineKeyboard))
	}
	buttons := markup.InlineKeyboard[0]

	// Macro rule should get Undo button only — no default quick action.
	hasUndo := false
	for _, btn := range buttons {
		if strings.Contains(btn.CallbackData, "/nudge undo ") {
			hasUndo = true
		}
		if btn.Text == "Log 500ml water" {
			t.Error("macro rule should NOT get water quick-action button")
		}
	}
	if !hasUndo {
		t.Error("expected Undo button in macro-rule markup")
	}
}

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

// --- Weekly rolling budget tests ---
//
// CHARACTERIZATION tests for evalWeeklyBudgetRules: pin down current
// behavior only, do not assert on anything the function doesn't already do.

type fakeWeeklyBudgetStore struct {
	rollups  []types.DailyRollup
	err      error
	calls    int
	gotStart string
	gotEnd   string
}

func (f *fakeWeeklyBudgetStore) GetRollups(_ context.Context, _, start, end string) ([]types.DailyRollup, error) {
	f.calls++
	f.gotStart, f.gotEnd = start, end
	return f.rollups, f.err
}

// enabledWeeklyOverride builds a bare "Enabled:true, no Params" override row,
// the common case for tests that don't need to tune clamp/override fields.
func enabledWeeklyOverride(ruleID string) map[string][]types.NudgeRuleConfig {
	return map[string][]types.NudgeRuleConfig{
		"u1": {{UserID: "u1", RuleID: ruleID, Enabled: true}},
	}
}

// (1) Absent override entry: the rule must never fire, even with a wired
// WeeklyBudgetStore and a real deficit — the opposite default from
// macro/health/digest rules (opt-in, not opt-out).
func TestWeeklyBudgetNoOverrideNeverFires(t *testing.T) {
	st := &fakeStore{
		users:   []types.User{{ID: "u1", Timezone: "UTC"}},
		targets: map[string]types.Macros{"u1": {Calories: 2000}},
		rollups: map[string]types.Macros{},
	}
	wbs := &fakeWeeklyBudgetStore{rollups: []types.DailyRollup{
		{Date: "2026-06-15", Consumed: types.Macros{Calories: 500}}, // real deficit vs 2000 target
	}}
	nt := &fakeNotifier{}
	// No WithRuleConfig at all: overrides stays nil for every rule.
	s := New(st, newFakeNudges(), nt, nil, time.UTC, time.Minute, WithWeeklyBudgetRules(wbs, DefaultWeeklyBudgetRules()))

	s.tick(context.Background(), time.Date(2026, 6, 17, 10, 0, 0, 0, time.UTC))

	if len(nt.sent) != 0 {
		t.Fatalf("no-override rule should never fire, sent %d", len(nt.sent))
	}
	if wbs.calls != 0 {
		t.Errorf("GetRollups should not even be called without an override, calls=%d", wbs.calls)
	}
}

// (2) Override present but Enabled:false → skip.
func TestWeeklyBudgetOverrideDisabledSkips(t *testing.T) {
	st := &fakeStore{
		users:   []types.User{{ID: "u1", Timezone: "UTC"}},
		targets: map[string]types.Macros{"u1": {Calories: 2000}},
		rollups: map[string]types.Macros{},
	}
	wbs := &fakeWeeklyBudgetStore{rollups: []types.DailyRollup{{Date: "2026-06-15", Consumed: types.Macros{Calories: 500}}}}
	rcs := &fakeRuleConfigStore{configs: map[string][]types.NudgeRuleConfig{
		"u1": {{UserID: "u1", RuleID: "weekly-budget-calories", Enabled: false}},
	}}
	nt := &fakeNotifier{}
	s := New(st, newFakeNudges(), nt, nil, time.UTC, time.Minute,
		WithWeeklyBudgetRules(wbs, DefaultWeeklyBudgetRules()), WithRuleConfig(rcs))

	s.tick(context.Background(), time.Date(2026, 6, 17, 10, 0, 0, 0, time.UTC))

	if len(nt.sent) != 0 {
		t.Errorf("Enabled:false override should skip the rule, sent %d", len(nt.sent))
	}
}

// (3) Hour gate: CheckHour is 9 for the default rules.
func TestWeeklyBudgetHourGate(t *testing.T) {
	st := &fakeStore{
		users:   []types.User{{ID: "u1", Timezone: "UTC"}},
		targets: map[string]types.Macros{"u1": {Calories: 2000}},
		rollups: map[string]types.Macros{},
	}
	wbs := &fakeWeeklyBudgetStore{rollups: []types.DailyRollup{{Date: "2026-06-15", Consumed: types.Macros{Calories: 500}}}}
	rcs := &fakeRuleConfigStore{configs: enabledWeeklyOverride("weekly-budget-calories")}
	nt := &fakeNotifier{}
	s := New(st, newFakeNudges(), nt, nil, time.UTC, time.Minute,
		WithWeeklyBudgetRules(wbs, DefaultWeeklyBudgetRules()), WithRuleConfig(rcs))

	s.tick(context.Background(), time.Date(2026, 6, 17, 8, 0, 0, 0, time.UTC)) // 1h before CheckHour

	if len(nt.sent) != 0 {
		t.Errorf("should not fire before CheckHour, sent %d", len(nt.sent))
	}
	if wbs.calls != 0 {
		t.Errorf("GetRollups should not be called before the hour gate, calls=%d", wbs.calls)
	}
}

// (4) Dedupe via nudge_log (WasNudged) short-circuits before GetRollups.
func TestWeeklyBudgetDedupeViaNudgeLog(t *testing.T) {
	st := &fakeStore{
		users:   []types.User{{ID: "u1", Timezone: "UTC"}},
		targets: map[string]types.Macros{"u1": {Calories: 2000}},
		rollups: map[string]types.Macros{},
	}
	wbs := &fakeWeeklyBudgetStore{rollups: []types.DailyRollup{{Date: "2026-06-15", Consumed: types.Macros{Calories: 500}}}}
	rcs := &fakeRuleConfigStore{configs: enabledWeeklyOverride("weekly-budget-calories")}
	nd := newFakeNudges()
	nd.marked[key("u1", "2026-06-17", "weekly-budget-calories")] = true
	nt := &fakeNotifier{}
	s := New(st, nd, nt, nil, time.UTC, time.Minute,
		WithWeeklyBudgetRules(wbs, DefaultWeeklyBudgetRules()), WithRuleConfig(rcs))

	s.tick(context.Background(), time.Date(2026, 6, 17, 10, 0, 0, 0, time.UTC))

	if len(nt.sent) != 0 {
		t.Errorf("already-nudged rule should dedupe, sent %d", len(nt.sent))
	}
	if wbs.calls != 0 {
		t.Errorf("GetRollups should not run once dedupe short-circuits, calls=%d", wbs.calls)
	}
}

// (5a) Calendar week bounds on a Monday: monday == today, 7 days remaining.
func TestWeeklyBudgetWeekBoundsMonday(t *testing.T) {
	st := &fakeStore{
		users:   []types.User{{ID: "u1", Timezone: "UTC"}},
		targets: map[string]types.Macros{"u1": {Calories: 2200}},
		rollups: map[string]types.Macros{},
	}
	wbs := &fakeWeeklyBudgetStore{} // no rollups: consumedPriorDays=0
	rcs := &fakeRuleConfigStore{configs: enabledWeeklyOverride("weekly-budget-calories")}
	nd, nt := newFakeNudges(), &fakeNotifier{}
	s := New(st, nd, nt, nil, time.UTC, time.Minute,
		WithWeeklyBudgetRules(wbs, DefaultWeeklyBudgetRules()), WithRuleConfig(rcs))

	// 2026-06-15 is a Monday.
	s.tick(context.Background(), time.Date(2026, 6, 15, 10, 0, 0, 0, time.UTC))

	if wbs.gotStart != "2026-06-15" || wbs.gotEnd != "2026-06-21" {
		t.Errorf("Monday week bounds = [%s, %s], want [2026-06-15, 2026-06-21]", wbs.gotStart, wbs.gotEnd)
	}
	// consumedPriorDays=0, daysRemaining=7 -> effective=plainDaily -> delta=0
	// (negligible): still marks nudged, no delivery.
	if len(nt.sent) != 0 {
		t.Errorf("Monday with zero prior consumption is a negligible delta, sent %d", len(nt.sent))
	}
	if !nd.marked[key("u1", "2026-06-15", "weekly-budget-calories")] {
		t.Error("negligible-delta branch should still mark nudged")
	}
}

// (5b) Calendar week bounds on a Sunday: the daysFromMonday=6 special case
// must resolve to the Monday of the SAME week, not one day in the future
// (naive int(weekday)-int(Monday) would give -1 on Sunday).
func TestWeeklyBudgetWeekBoundsSunday(t *testing.T) {
	st := &fakeStore{
		users:   []types.User{{ID: "u1", Timezone: "UTC"}},
		targets: map[string]types.Macros{"u1": {Calories: 2200}},
		rollups: map[string]types.Macros{},
	}
	wbs := &fakeWeeklyBudgetStore{}
	rcs := &fakeRuleConfigStore{configs: enabledWeeklyOverride("weekly-budget-calories")}
	s := New(st, newFakeNudges(), &fakeNotifier{}, nil, time.UTC, time.Minute,
		WithWeeklyBudgetRules(wbs, DefaultWeeklyBudgetRules()), WithRuleConfig(rcs))

	// 2026-06-21 is a Sunday.
	s.tick(context.Background(), time.Date(2026, 6, 21, 10, 0, 0, 0, time.UTC))

	if wbs.gotStart != "2026-06-15" || wbs.gotEnd != "2026-06-21" {
		t.Errorf("Sunday week bounds = [%s, %s], want [2026-06-15, 2026-06-21]", wbs.gotStart, wbs.gotEnd)
	}
}

// (6) GetRollups error: no panic, no delivery, no mark.
func TestWeeklyBudgetGetRollupsErrorNoDelivery(t *testing.T) {
	st := &fakeStore{
		users:   []types.User{{ID: "u1", Timezone: "UTC"}},
		targets: map[string]types.Macros{"u1": {Calories: 2000}},
		rollups: map[string]types.Macros{},
	}
	wbs := &fakeWeeklyBudgetStore{err: errors.New("boom")}
	rcs := &fakeRuleConfigStore{configs: enabledWeeklyOverride("weekly-budget-calories")}
	nd, nt := newFakeNudges(), &fakeNotifier{}
	s := New(st, nd, nt, nil, time.UTC, time.Minute,
		WithWeeklyBudgetRules(wbs, DefaultWeeklyBudgetRules()), WithRuleConfig(rcs))

	s.tick(context.Background(), time.Date(2026, 6, 17, 10, 0, 0, 0, time.UTC)) // must not panic

	if len(nt.sent) != 0 {
		t.Errorf("GetRollups error should not deliver, sent %d", len(nt.sent))
	}
	if len(nd.marked) != 0 {
		t.Errorf("GetRollups error should not mark nudged, marked=%v", nd.marked)
	}
}

// (7) plainDaily <= 0 (no macro target set for this rule's macro) → skip.
func TestWeeklyBudgetNoTargetSkips(t *testing.T) {
	st := &fakeStore{
		users:   []types.User{{ID: "u1", Timezone: "UTC"}},
		targets: map[string]types.Macros{"u1": {Protein: 150}}, // no Calories target
		rollups: map[string]types.Macros{},
	}
	wbs := &fakeWeeklyBudgetStore{rollups: []types.DailyRollup{{Date: "2026-06-15", Consumed: types.Macros{Calories: 500}}}}
	rcs := &fakeRuleConfigStore{configs: enabledWeeklyOverride("weekly-budget-calories")}
	nt := &fakeNotifier{}
	s := New(st, newFakeNudges(), nt, nil, time.UTC, time.Minute,
		WithWeeklyBudgetRules(wbs, DefaultWeeklyBudgetRules()), WithRuleConfig(rcs))

	s.tick(context.Background(), time.Date(2026, 6, 17, 10, 0, 0, 0, time.UTC))

	if len(nt.sent) != 0 {
		t.Errorf("plainDaily<=0 should skip without delivering, sent %d", len(nt.sent))
	}
}

// (8) WeeklyTargetOverride replaces the store's daily target in the formula.
func TestWeeklyBudgetTargetOverrideChangesEffective(t *testing.T) {
	st := &fakeStore{
		users:   []types.User{{ID: "u1", Timezone: "UTC"}},
		targets: map[string]types.Macros{"u1": {Calories: 2000}}, // ignored in favor of the override
		rollups: map[string]types.Macros{},
	}
	wbs := &fakeWeeklyBudgetStore{rollups: []types.DailyRollup{
		{Date: "2026-06-15", Consumed: types.Macros{Calories: 1500}},
		{Date: "2026-06-16", Consumed: types.Macros{Calories: 1500}},
	}}
	params, err := json.Marshal(types.WeeklyBudgetConfig{WeeklyTargetOverride: 3000})
	if err != nil {
		t.Fatal(err)
	}
	rcs := &fakeRuleConfigStore{configs: map[string][]types.NudgeRuleConfig{
		"u1": {{UserID: "u1", RuleID: "weekly-budget-calories", Enabled: true, Params: params}},
	}}
	nt := &fakeNotifier{}
	s := New(st, newFakeNudges(), nt, nil, time.UTC, time.Minute,
		WithWeeklyBudgetRules(wbs, DefaultWeeklyBudgetRules()), WithRuleConfig(rcs))

	// Wednesday: consumedPriorDays=3000, daysRemaining=5. With the override,
	// plainDaily becomes 3000 (not the store's 2000): weeklyTarget=21000,
	// effective=(21000-3000)/5=3600, delta=+600. Without the override applied
	// the delta would instead be +200 (using the store's 2000) — the distinct
	// number proves the override actually took effect.
	s.tick(context.Background(), time.Date(2026, 6, 17, 10, 0, 0, 0, time.UTC))

	if len(nt.sent) != 1 {
		t.Fatalf("sent = %d, want 1", len(nt.sent))
	}
	if want := "Catch up today, +600kcal"; nt.sent[0].Body != want {
		t.Errorf("body = %q, want %q (proves WeeklyTargetOverride is applied)", nt.sent[0].Body, want)
	}
}

// (9) Negligible delta (<3% of daily target): marks nudged so it doesn't
// recompute every tick, but must NOT call deliver/notifier.
func TestWeeklyBudgetNegligibleDeltaMarksNudgedWithoutDelivery(t *testing.T) {
	st := &fakeStore{
		users:   []types.User{{ID: "u1", Timezone: "UTC"}},
		targets: map[string]types.Macros{"u1": {Calories: 2000}},
		rollups: map[string]types.Macros{},
	}
	wbs := &fakeWeeklyBudgetStore{rollups: []types.DailyRollup{
		{Date: "2026-06-15", Consumed: types.Macros{Calories: 1925}},
		{Date: "2026-06-16", Consumed: types.Macros{Calories: 1925}},
	}}
	rcs := &fakeRuleConfigStore{configs: enabledWeeklyOverride("weekly-budget-calories")}
	nd := newFakeNudges()
	nt := &fakeNotifier{} // asserting sent is empty proves deliver/notifier was never invoked
	s := New(st, nd, nt, nil, time.UTC, time.Minute,
		WithWeeklyBudgetRules(wbs, DefaultWeeklyBudgetRules()), WithRuleConfig(rcs))

	// Wednesday: consumedPriorDays=3850, daysRemaining=5. effective=2030,
	// delta=+30, which is under 3% of 2000 (60) -> negligible.
	s.tick(context.Background(), time.Date(2026, 6, 17, 10, 0, 0, 0, time.UTC))

	if len(nt.sent) != 0 {
		t.Fatalf("negligible delta must not deliver, sent %d", len(nt.sent))
	}
	if !nd.marked[key("u1", "2026-06-17", "weekly-budget-calories")] {
		t.Error("negligible delta should still mark nudged so it doesn't recompute every tick")
	}
}

// (10) Positive delta -> "catch up +Xkcal" message.
func TestWeeklyBudgetPositiveDeltaCatchUpMessage(t *testing.T) {
	st := &fakeStore{
		users:   []types.User{{ID: "u1", Timezone: "UTC"}},
		targets: map[string]types.Macros{"u1": {Calories: 2200}},
		rollups: map[string]types.Macros{},
	}
	wbs := &fakeWeeklyBudgetStore{rollups: []types.DailyRollup{
		{Date: "2026-06-15", Consumed: types.Macros{Calories: 1500}}, // Mon, prior day
		{Date: "2026-06-16", Consumed: types.Macros{Calories: 1500}}, // Tue, prior day
		{Date: "2026-06-17", Consumed: types.Macros{Calories: 900}},  // Wed = today, must NOT count
	}}
	rcs := &fakeRuleConfigStore{configs: enabledWeeklyOverride("weekly-budget-calories")}
	nt := &fakeNotifier{}
	s := New(st, newFakeNudges(), nt, nil, time.UTC, time.Minute,
		WithWeeklyBudgetRules(wbs, DefaultWeeklyBudgetRules()), WithRuleConfig(rcs))

	// Wednesday: consumedPriorDays=3000 (today's rollup excluded), daysRemaining=5.
	// weeklyTarget=2200*7=15400. effective=(15400-3000)/5=2480. delta=+280.
	s.tick(context.Background(), time.Date(2026, 6, 17, 10, 0, 0, 0, time.UTC))

	if len(nt.sent) != 1 {
		t.Fatalf("sent = %d, want 1", len(nt.sent))
	}
	if want := "Catch up today, +280kcal"; nt.sent[0].Body != want {
		t.Errorf("body = %q, want %q", nt.sent[0].Body, want)
	}
}

// (11) Negative delta -> "ease up -Xg" message (also covers the protein "g" unit).
func TestWeeklyBudgetNegativeDeltaEaseUpMessageWithGramsUnit(t *testing.T) {
	st := &fakeStore{
		users:   []types.User{{ID: "u1", Timezone: "UTC"}},
		targets: map[string]types.Macros{"u1": {Protein: 150}},
		rollups: map[string]types.Macros{},
	}
	wbs := &fakeWeeklyBudgetStore{rollups: []types.DailyRollup{
		{Date: "2026-06-15", Consumed: types.Macros{Protein: 175}}, // Mon
		{Date: "2026-06-16", Consumed: types.Macros{Protein: 175}}, // Tue
	}}
	rcs := &fakeRuleConfigStore{configs: enabledWeeklyOverride("weekly-budget-protein")}
	nt := &fakeNotifier{}
	s := New(st, newFakeNudges(), nt, nil, time.UTC, time.Minute,
		WithWeeklyBudgetRules(wbs, DefaultWeeklyBudgetRules()), WithRuleConfig(rcs))

	// Wednesday: consumedPriorDays=350, daysRemaining=5.
	// weeklyTarget=150*7=1050. effective=(1050-350)/5=140. delta=-10 (g).
	s.tick(context.Background(), time.Date(2026, 6, 17, 10, 0, 0, 0, time.UTC))

	if len(nt.sent) != 1 {
		t.Fatalf("sent = %d, want 1", len(nt.sent))
	}
	if want := "Ease up today, -10g"; nt.sent[0].Body != want {
		t.Errorf("body = %q, want %q", nt.sent[0].Body, want)
	}
}

// (12) deliver/notifier error -> NOT marked as nudged, so it retries next tick.
func TestWeeklyBudgetDeliverErrorNotMarkedNudged(t *testing.T) {
	st := &fakeStore{
		users:   []types.User{{ID: "u1", Timezone: "UTC"}},
		targets: map[string]types.Macros{"u1": {Calories: 2200}},
		rollups: map[string]types.Macros{},
	}
	wbs := &fakeWeeklyBudgetStore{rollups: []types.DailyRollup{
		{Date: "2026-06-15", Consumed: types.Macros{Calories: 1500}},
		{Date: "2026-06-16", Consumed: types.Macros{Calories: 1500}},
	}}
	rcs := &fakeRuleConfigStore{configs: enabledWeeklyOverride("weekly-budget-calories")}
	nd := newFakeNudges()
	nt := &fakeNotifier{err: errors.New("delivery boom")}
	s := New(st, nd, nt, nil, time.UTC, time.Minute,
		WithWeeklyBudgetRules(wbs, DefaultWeeklyBudgetRules()), WithRuleConfig(rcs))

	// Same non-negligible +280 delta scenario as the catch-up test, but delivery fails.
	s.tick(context.Background(), time.Date(2026, 6, 17, 10, 0, 0, 0, time.UTC))

	if nd.marked[key("u1", "2026-06-17", "weekly-budget-calories")] {
		t.Error("delivery error must not mark nudged, so it retries next tick")
	}
}
