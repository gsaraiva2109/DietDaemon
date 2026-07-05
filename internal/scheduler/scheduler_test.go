package scheduler

import (
	"context"
	"encoding/json"
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
	rollups []types.DailyRollup
	weights []types.WeightEntry
}

func (f *fakeDigestStore) GetRollups(_ context.Context, _, _, _ string) ([]types.DailyRollup, error) {
	return f.rollups, nil
}
func (f *fakeDigestStore) ListWeight(_ context.Context, _ string, _ int) ([]types.WeightEntry, error) {
	return f.weights, nil
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
	}
	nt := &fakeNotifier{}

	// Same call shape as cmd/dietdaemon/main.go's scheduler.New(...).
	sched := New(full, full, nt, DefaultRules(), time.UTC, time.Minute,
		WithHealthRules(full, DefaultHealthRules()),
		WithRuleConfig(full),
		WithDigestRules(full, DefaultDigestRules()),
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
