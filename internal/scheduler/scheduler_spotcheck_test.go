package scheduler

// Spot-check tests for issue #158: evalHealthRules error paths, and thinner
// evalSmartMealRules coverage (disabled override, reminder-window boundaries,
// the "already ate since previous slot" skip, and multi-hour interaction).
// evalUser's nil-store matrix is exercised implicitly throughout the rest of
// this package's tests (most omit health/digest/weeklyBudget/mealHistory
// entirely and still pass), so no dedicated combinatorial test is added here.

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/gsaraiva2109/dietdaemon/core/types"
)

// --- evalHealthRules: real (non-ErrNotFound) store errors must skip the
// domain's rule, not be treated the same as "no data" / misfire.

func TestHealthWaterErrorSkipsRule(t *testing.T) {
	st := &fakeStore{users: []types.User{{ID: "u1", Timezone: "UTC"}}, targets: map[string]types.Macros{"u1": {Protein: 100}}, rollups: map[string]types.Macros{"u1|2026-06-17": {Protein: 0}}}
	hs := &fakeHealthStore{waterErr: errors.New("db unavailable")}
	nt := &fakeNotifier{}

	hr := DefaultHealthRules()
	s := newHealthSched(st, hs, newFakeNudges(), nt, hr)
	s.tick(context.Background(), time.Date(2026, 6, 17, 20, 0, 0, 0, time.UTC)) // past both water CheckHours

	for _, n := range nt.sent {
		if n.Body == hr[0].Message || n.Body == hr[1].Message {
			t.Errorf("a real GetWaterToday error should skip water rules, not misfire, got: %q", n.Body)
		}
	}
}

func TestHealthWorkoutErrorSkipsRule(t *testing.T) {
	st := &fakeStore{users: []types.User{{ID: "u1", Timezone: "UTC"}}, targets: map[string]types.Macros{"u1": {Protein: 100}}, rollups: map[string]types.Macros{"u1|2026-06-17": {Protein: 0}}}
	// A real error (not ErrNotFound) with an empty workouts slice must NOT be
	// treated the same as "never worked out" (which would trigger a nudge).
	hs := &fakeHealthStore{workoutsErr: errors.New("db unavailable")}
	nt := &fakeNotifier{}

	hr := DefaultHealthRules()
	s := newHealthSched(st, hs, newFakeNudges(), nt, hr)
	s.tick(context.Background(), time.Date(2026, 6, 17, 12, 0, 0, 0, time.UTC))

	for _, n := range nt.sent {
		if n.Body == hr[2].Message {
			t.Error("a real ListWorkouts error should skip workout-reminder, not fire it")
		}
	}
}

func TestHealthSleepErrorSkipsRule(t *testing.T) {
	st := &fakeStore{users: []types.User{{ID: "u1", Timezone: "UTC"}}, targets: map[string]types.Macros{"u1": {Protein: 100}}, rollups: map[string]types.Macros{"u1|2026-06-17": {Protein: 0}}}
	hs := &fakeHealthStore{sleepErr: errors.New("db unavailable")}
	nt := &fakeNotifier{}

	hr := DefaultHealthRules()
	s := newHealthSched(st, hs, newFakeNudges(), nt, hr)
	s.tick(context.Background(), time.Date(2026, 6, 17, 22, 30, 0, 0, time.UTC))

	for _, n := range nt.sent {
		if n.Body == hr[3].Message {
			t.Error("a real GetActiveSleep error should skip sleep-reminder, not fire it")
		}
	}
}

func TestHealthFastingErrorSkipsRule(t *testing.T) {
	st := &fakeStore{users: []types.User{{ID: "u1", Timezone: "UTC"}}, targets: map[string]types.Macros{"u1": {Protein: 100}}, rollups: map[string]types.Macros{"u1|2026-06-17": {Protein: 0}}}
	hs := &fakeHealthStore{fastErr: errors.New("db unavailable")}
	nt := &fakeNotifier{}

	hr := DefaultHealthRules()
	s := newHealthSched(st, hs, newFakeNudges(), nt, hr)
	s.tick(context.Background(), time.Date(2026, 6, 17, 14, 0, 0, 0, time.UTC))

	for _, n := range nt.sent {
		if n.Body == hr[4].Message {
			t.Error("a real GetActiveFast error should skip fast-ending, not fire it")
		}
	}
}

// --- evalSmartMealRules ---

// mealTimesFor builds one meal timestamp per (day, hour) pair, all in UTC in
// June 2026 — the shape learnedMealHours needs (>=7 distinct days, >=3
// occurrences per hour) to learn the given hours.
func mealTimesFor(days []int, hours []int) []time.Time {
	var out []time.Time
	for _, d := range days {
		for _, h := range hours {
			out = append(out, time.Date(2026, 6, d, h, 0, 0, 0, time.UTC))
		}
	}
	return out
}

func TestSmartMealRuleDisabledOverrideSkips(t *testing.T) {
	times := mealTimesFor([]int{1, 2, 3, 4, 5, 6, 7}, []int{12})
	st := &fakeStore{users: []types.User{{ID: "u1", Timezone: "UTC"}}, targets: map[string]types.Macros{}, rollups: map[string]types.Macros{}}
	rcs := &fakeRuleConfigStore{configs: map[string][]types.NudgeRuleConfig{
		"u1": {{UserID: "u1", RuleID: "smart-meal-reminders", Enabled: false}},
	}}
	nt := &fakeNotifier{}
	s := New(st, newFakeNudges(), nt, nil, time.UTC, time.Minute,
		WithSmartMealRules(&fakeMealHistory{times: times}, DefaultSmartMealRules()),
		WithRuleConfig(rcs),
	)

	// Would fire at 11:30 (reminder window for the learned 12:00 slot) if enabled.
	s.tick(context.Background(), time.Date(2026, 6, 10, 11, 30, 0, 0, time.UTC))
	if len(nt.sent) != 0 {
		t.Errorf("disabled smart-meal-reminders override should skip entirely, sent %d", len(nt.sent))
	}
}

// TestSmartMealReminderWindowBoundaryToday pins down the exact [reminder,
// reminder+interval) firing window for the offset=0 (today) occurrence:
// just before it, at its start, just before its end, and at its end.
func TestSmartMealReminderWindowBoundaryToday(t *testing.T) {
	times := mealTimesFor([]int{1, 2, 3, 4, 5, 6, 7}, []int{12})
	// "now" lands on day 10 so "yesterday" (day 9) has no recorded meal,
	// keeping the "already ate since previous slot" check out of the way.
	reminder := time.Date(2026, 6, 10, 11, 30, 0, 0, time.UTC)

	cases := []struct {
		name    string
		now     time.Time
		wantFor bool
	}{
		{"justBeforeStart", reminder.Add(-time.Second), false},
		{"atStart", reminder, true},
		{"justBeforeEnd", reminder.Add(time.Minute - time.Second), true},
		{"atEnd", reminder.Add(time.Minute), false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			st := &fakeStore{users: []types.User{{ID: "u1", Timezone: "UTC"}}, targets: map[string]types.Macros{}, rollups: map[string]types.Macros{}}
			nt := &fakeNotifier{}
			s := New(st, newFakeNudges(), nt, nil, time.UTC, time.Minute,
				WithSmartMealRules(&fakeMealHistory{times: times}, DefaultSmartMealRules()))
			s.tick(context.Background(), tc.now)
			got := len(nt.sent) > 0
			if got != tc.wantFor {
				t.Errorf("now=%v: fired=%v, want %v", tc.now, got, tc.wantFor)
			}
		})
	}
}

// TestSmartMealReminderTomorrowOffsetJustBeforeWindowSkips complements the
// existing midnight-dedupe test (which checks the offset=1 window's exact
// start) by checking the instant just before that window opens.
func TestSmartMealReminderTomorrowOffsetJustBeforeWindowSkips(t *testing.T) {
	var times []time.Time
	for day := 1; day <= 7; day++ {
		times = append(times, time.Date(2026, 6, day, 0, 0, 0, 0, time.UTC))
	}
	st := &fakeStore{users: []types.User{{ID: "u1", Timezone: "UTC"}}, targets: map[string]types.Macros{}, rollups: map[string]types.Macros{}}
	nt := &fakeNotifier{}
	s := New(st, newFakeNudges(), nt, nil, time.UTC, time.Minute,
		WithSmartMealRules(&fakeMealHistory{times: times}, DefaultSmartMealRules()))

	// Window for tomorrow's (day 9) midnight slot opens at day8 23:30:00; one
	// second before that must not fire.
	justBefore := time.Date(2026, 6, 8, 23, 29, 59, 0, time.UTC)
	s.tick(context.Background(), justBefore)
	if len(nt.sent) != 0 {
		t.Errorf("just before the tomorrow-offset reminder window should not fire, sent %d", len(nt.sent))
	}
}

// TestSmartMealReminderSkipsWhenAlreadyAteSincePreviousSlot verifies the
// "user already ate since the previous learned slot" skip: a meal logged
// between the previous slot's occurrence and now suppresses the reminder
// that would otherwise fire.
func TestSmartMealReminderSkipsWhenAlreadyAteSincePreviousSlot(t *testing.T) {
	times := mealTimesFor([]int{1, 2, 3, 4, 5, 6, 7}, []int{12})
	// Extra meal today, after the previous slot (day9 12:00) and before now.
	times = append(times, time.Date(2026, 6, 10, 8, 0, 0, 0, time.UTC))

	st := &fakeStore{users: []types.User{{ID: "u1", Timezone: "UTC"}}, targets: map[string]types.Macros{}, rollups: map[string]types.Macros{}}
	nt := &fakeNotifier{}
	s := New(st, newFakeNudges(), nt, nil, time.UTC, time.Minute,
		WithSmartMealRules(&fakeMealHistory{times: times}, DefaultSmartMealRules()))

	// Same instant that fires in TestSmartMealReminderWindowBoundaryToday's
	// "atStart" case, absent the extra meal.
	s.tick(context.Background(), time.Date(2026, 6, 10, 11, 30, 0, 0, time.UTC))
	if len(nt.sent) != 0 {
		t.Errorf("a meal logged since the previous learned slot should suppress the reminder, sent %d", len(nt.sent))
	}
}

// TestSmartMealRemindersOnlyFireForHourInWindow learns three hours end to
// end and checks that only the hour whose reminder window is currently open
// fires — the others (past and future slots) stay silent.
func TestSmartMealRemindersOnlyFireForHourInWindow(t *testing.T) {
	times := mealTimesFor([]int{1, 2, 3, 4, 5, 6, 7}, []int{8, 12, 18})
	st := &fakeStore{users: []types.User{{ID: "u1", Timezone: "UTC"}}, targets: map[string]types.Macros{}, rollups: map[string]types.Macros{}}
	nd, nt := newFakeNudges(), &fakeNotifier{}
	s := New(st, nd, nt, nil, time.UTC, time.Minute,
		WithSmartMealRules(&fakeMealHistory{times: times}, DefaultSmartMealRules()))

	// Only the 12:00 slot's reminder window (11:30-11:31) is open here; the
	// 8:00 slot's window closed hours ago and the 18:00 slot's hasn't opened.
	s.tick(context.Background(), time.Date(2026, 6, 10, 11, 30, 0, 0, time.UTC))

	if len(nt.sent) != 1 {
		t.Fatalf("sent = %d, want 1 (only the in-window hour)", len(nt.sent))
	}
	if !nd.marked["u1|2026-06-10|smart-meal-reminders-12"] {
		t.Errorf("expected the 12:00 slot to fire, marked: %#v", nd.marked)
	}
	if nd.marked["u1|2026-06-10|smart-meal-reminders-08"] || nd.marked["u1|2026-06-09|smart-meal-reminders-18"] {
		t.Errorf("out-of-window hours should not fire, marked: %#v", nd.marked)
	}
}
