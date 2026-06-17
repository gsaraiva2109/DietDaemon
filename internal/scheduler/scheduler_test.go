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
