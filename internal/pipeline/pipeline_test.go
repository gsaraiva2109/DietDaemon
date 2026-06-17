package pipeline

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/gsaraiva2109/dietdaemon/core/types"
)

// --- fakes ---

type fakeParser struct {
	items []types.ParsedItem
	conf  float64
}

func (f fakeParser) Extract(context.Context, string, string) ([]types.ParsedItem, float64, error) {
	return f.items, f.conf, nil
}
func (fakeParser) Tier() types.ParserTier { return types.TierDeterministic }

type fakeResolver struct {
	out  []types.ResolvedItem
	need int
}

func (f fakeResolver) Resolve(context.Context, string, []types.ParsedItem) ([]types.ResolvedItem, int) {
	return f.out, f.need
}

type fakeStore struct {
	meals   []types.Meal
	rollups map[string]types.DailyRollup
	targets map[string]types.Macros
	users   int
}

func newFakeStore() *fakeStore {
	return &fakeStore{rollups: map[string]types.DailyRollup{}, targets: map[string]types.Macros{}}
}

func (s *fakeStore) UpsertUser(context.Context, types.User) error { s.users++; return nil }
func (s *fakeStore) SaveMeal(_ context.Context, m types.Meal) error {
	s.meals = append(s.meals, m)
	return nil
}
func (s *fakeStore) GetTargets(_ context.Context, userID string) (types.DailyTargets, error) {
	if m, ok := s.targets[userID]; ok {
		return types.DailyTargets{UserID: userID, Targets: m}, nil
	}
	return types.DailyTargets{}, types.ErrNotFound
}
func (s *fakeStore) SetTargets(_ context.Context, t types.DailyTargets) error {
	s.targets[t.UserID] = t.Targets
	return nil
}
func (s *fakeStore) GetRollup(_ context.Context, _, date string) (types.DailyRollup, error) {
	if r, ok := s.rollups[date]; ok {
		return r, nil
	}
	return types.DailyRollup{}, types.ErrNotFound
}
func (s *fakeStore) UpsertRollup(_ context.Context, r types.DailyRollup) error {
	s.rollups[r.Date] = r
	return nil
}

type fakeReplier struct{ sent []types.Reply }

func (r *fakeReplier) Send(_ context.Context, reply types.Reply) error {
	r.sent = append(r.sent, reply)
	return nil
}

func resolved(name string, m types.Macros) types.ResolvedItem {
	return types.ResolvedItem{
		Parsed: types.ParsedItem{RawPhrase: name, NormalizedGrams: 100},
		Match:  types.FoodMatch{FoodID: name, Name: name},
		Macros: m,
	}
}

// --- tests ---

func TestHandleLogsMealAndReplies(t *testing.T) {
	st := newFakeStore()
	rp := &fakeReplier{}
	e := New(
		fakeParser{items: []types.ParsedItem{{RawPhrase: "chicken"}}, conf: 0.95},
		fakeResolver{out: []types.ResolvedItem{resolved("chicken", types.Macros{Calories: 330, Protein: 62})}},
		st, rp, time.UTC, 0.6,
	)

	msg := types.InboundMessage{
		UserID:      "u1",
		At:          time.Date(2026, 6, 17, 12, 0, 0, 0, time.UTC),
		Kind:        types.MessageText,
		Text:        "200g chicken",
		ChannelMeta: map[string]string{"chat_id": "42"},
	}
	if err := e.Handle(context.Background(), msg); err != nil {
		t.Fatalf("Handle error = %v", err)
	}

	if len(st.meals) != 1 {
		t.Fatalf("meals saved = %d, want 1", len(st.meals))
	}
	if got := st.meals[0].Total().Calories; got != 330 {
		t.Errorf("meal calories = %v, want 330", got)
	}
	r, ok := st.rollups["2026-06-17"]
	if !ok || r.Consumed.Protein != 62 {
		t.Errorf("rollup = %+v, want protein 62 on 2026-06-17", r)
	}
	if len(rp.sent) != 1 || !strings.Contains(rp.sent[0].Text, "330 kcal") {
		t.Errorf("reply = %+v, want a 330 kcal summary", rp.sent)
	}
	if rp.sent[0].ChannelMeta["chat_id"] != "42" {
		t.Errorf("reply must echo ChannelMeta for routing, got %v", rp.sent[0].ChannelMeta)
	}
}

func TestHandleFlagsClarification(t *testing.T) {
	st := newFakeStore()
	rp := &fakeReplier{}
	e := New(
		fakeParser{items: []types.ParsedItem{{RawPhrase: "eggs"}}, conf: 0.9},
		fakeResolver{out: []types.ResolvedItem{resolved("eggs", types.Macros{})}, need: 1},
		st, rp, time.UTC, 0.6,
	)
	msg := types.InboundMessage{UserID: "u1", Text: "2 eggs"}
	if err := e.Handle(context.Background(), msg); err != nil {
		t.Fatalf("Handle error = %v", err)
	}
	if len(rp.sent) != 1 || !strings.Contains(rp.sent[0].Text, "need a portion") {
		t.Errorf("expected clarification nudge, got %+v", rp.sent)
	}
}

func TestHandleEmptyText(t *testing.T) {
	st := newFakeStore()
	rp := &fakeReplier{}
	e := New(fakeParser{}, fakeResolver{}, st, rp, time.UTC, 0.6)
	if err := e.Handle(context.Background(), types.InboundMessage{UserID: "u1", Text: "  "}); err != nil {
		t.Fatalf("Handle error = %v", err)
	}
	if len(st.meals) != 0 {
		t.Errorf("no meal should be saved for empty text, got %d", len(st.meals))
	}
	if len(rp.sent) != 1 {
		t.Errorf("want a guidance reply, got %+v", rp.sent)
	}
}

func TestTargetCommandSetsGoals(t *testing.T) {
	st := newFakeStore()
	rp := &fakeReplier{}
	e := New(fakeParser{}, fakeResolver{}, st, rp, time.UTC, 0.6)

	msg := types.InboundMessage{UserID: "u1", Text: "/target kcal=3000 protein=180 carbs=350 fat=90"}
	if err := e.Handle(context.Background(), msg); err != nil {
		t.Fatalf("Handle error = %v", err)
	}
	if len(st.meals) != 0 {
		t.Errorf("a command must not log a meal, got %d", len(st.meals))
	}
	got := st.targets["u1"]
	if got.Calories != 3000 || got.Protein != 180 || got.Carbs != 350 || got.Fat != 90 {
		t.Errorf("targets = %+v, want 3000/180/350/90", got)
	}
	if len(rp.sent) != 1 || !strings.Contains(rp.sent[0].Text, "Targets set") {
		t.Errorf("reply = %+v, want confirmation", rp.sent)
	}
}

func TestRollupAccumulates(t *testing.T) {
	st := newFakeStore()
	rp := &fakeReplier{}
	e := New(
		fakeParser{items: []types.ParsedItem{{RawPhrase: "rice"}}, conf: 0.95},
		fakeResolver{out: []types.ResolvedItem{resolved("rice", types.Macros{Calories: 100})}},
		st, rp, time.UTC, 0.6,
	)
	at := time.Date(2026, 6, 17, 8, 0, 0, 0, time.UTC)
	for i := 0; i < 3; i++ {
		msg := types.InboundMessage{UserID: "u1", At: at, Text: "100g rice"}
		if err := e.Handle(context.Background(), msg); err != nil {
			t.Fatalf("Handle error = %v", err)
		}
	}
	if got := st.rollups["2026-06-17"].Consumed.Calories; got != 300 {
		t.Errorf("accumulated calories = %v, want 300", got)
	}
}
