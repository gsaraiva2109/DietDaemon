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
	fn   func([]types.ParsedItem) ([]types.ResolvedItem, int) // optional, per-call
}

func (f fakeResolver) Resolve(_ context.Context, _ string, items []types.ParsedItem) ([]types.ResolvedItem, int) {
	if f.fn != nil {
		return f.fn(items)
	}
	return f.out, f.need
}

type fakeStore struct {
	meals    []types.Meal
	rollups  map[string]types.DailyRollup
	targets  map[string]types.Macros
	users    map[string]types.User
	channels map[string]string // "channel:channelUserID" → userID
}

func newFakeStore() *fakeStore {
	return &fakeStore{
		rollups:  map[string]types.DailyRollup{},
		targets:  map[string]types.Macros{},
		users:    map[string]types.User{},
		channels: map[string]string{},
	}
}

func (s *fakeStore) UpsertUser(_ context.Context, u types.User) error { s.users[u.ID] = u; return nil }
func (s *fakeStore) GetUser(_ context.Context, userID string) (types.User, error) {
	if u, ok := s.users[userID]; ok {
		return u, nil
	}
	return types.User{}, types.ErrNotFound
}
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
func (s *fakeStore) GetUserIDByChannel(_ context.Context, channel, channelUserID string) (string, error) {
	key := channel + ":" + channelUserID
	if uid, ok := s.channels[key]; ok {
		return uid, nil
	}
	return "", types.ErrNotFound
}
func (s *fakeStore) MapChannelUser(_ context.Context, channel, channelUserID, userID string) error {
	s.channels[channel+":"+channelUserID] = userID
	return nil
}

type fakeReplier struct{ sent []types.Reply }

func (r *fakeReplier) Send(_ context.Context, reply types.Reply) error {
	r.sent = append(r.sent, reply)
	return nil
}

func (r *fakeReplier) last() string {
	if len(r.sent) == 0 {
		return ""
	}
	return r.sent[len(r.sent)-1].Text
}

// fakePending is a non-expiring in-memory PendingStore for tests.
type fakePending struct{ m map[string]types.PendingMeal }

func newFakePending() *fakePending { return &fakePending{m: map[string]types.PendingMeal{}} }

func (p *fakePending) Save(_ context.Context, pm types.PendingMeal) error {
	p.m[pm.UserID] = pm
	return nil
}
func (p *fakePending) Get(_ context.Context, userID string) (types.PendingMeal, error) {
	if pm, ok := p.m[userID]; ok {
		return pm, nil
	}
	return types.PendingMeal{}, types.ErrNotFound
}
func (p *fakePending) Delete(_ context.Context, userID string) error {
	delete(p.m, userID)
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
		st, newFakePending(), rp, time.UTC, 0.6, "telegram", nil,
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

// portionPending returns a resolved item whose food is known but whose portion
// is unknown — exactly what the resolver flags for a count-based "2 eggs".
func portionPending(name string, per100g types.Macros, qty float64, unit string) types.ResolvedItem {
	return types.ResolvedItem{
		Parsed: types.ParsedItem{RawPhrase: name, Quantity: qty, Unit: unit, NormalizedGrams: 0},
		Match:  types.FoodMatch{FoodID: name, Name: name, Per100g: per100g},
	}
}

func TestClarificationHoldsMealAndAsks(t *testing.T) {
	st := newFakeStore()
	rp := &fakeReplier{}
	e := New(
		fakeParser{items: []types.ParsedItem{{RawPhrase: "eggs"}}, conf: 0.9},
		fakeResolver{out: []types.ResolvedItem{portionPending("eggs", types.Macros{Calories: 155, Protein: 13}, 2, "unit")}, need: 1},
		st, newFakePending(), rp, time.UTC, 0.6, "telegram", nil,
	)
	msg := types.InboundMessage{UserID: "u1", Text: "2 eggs", ChannelMeta: map[string]string{"chat_id": "42"}}
	if err := e.Handle(context.Background(), msg); err != nil {
		t.Fatalf("Handle error = %v", err)
	}
	// Nothing logged yet — the meal is held pending a portion.
	if len(st.meals) != 0 {
		t.Fatalf("meal must not be logged before clarification, got %d", len(st.meals))
	}
	if !strings.Contains(rp.last(), "How many grams") {
		t.Errorf("expected a portion question, got %q", rp.last())
	}
}

func TestClarificationPortionCompletesMeal(t *testing.T) {
	st := newFakeStore()
	rp := &fakeReplier{}
	e := New(
		fakeParser{items: []types.ParsedItem{{RawPhrase: "eggs"}}, conf: 0.9},
		fakeResolver{out: []types.ResolvedItem{portionPending("eggs", types.Macros{Calories: 155, Protein: 13}, 2, "unit")}, need: 1},
		st, newFakePending(), rp, time.UTC, 0.6, "telegram", nil,
	)
	ctx := context.Background()
	if err := e.Handle(ctx, types.InboundMessage{UserID: "u1", Text: "2 eggs", ChannelMeta: map[string]string{"chat_id": "42"}}); err != nil {
		t.Fatalf("Handle error = %v", err)
	}
	// User answers with a weight; "100g" of 155kcal/100g → 155 kcal logged.
	if err := e.Handle(ctx, types.InboundMessage{UserID: "u1", Text: "100g", ChannelMeta: map[string]string{"chat_id": "42"}}); err != nil {
		t.Fatalf("Handle error = %v", err)
	}
	if len(st.meals) != 1 {
		t.Fatalf("meal should be logged after the portion answer, got %d", len(st.meals))
	}
	if got := st.meals[0].Total().Calories; got != 155 {
		t.Errorf("logged calories = %v, want 155", got)
	}
	if got := st.meals[0].Items[0].Parsed.NormalizedGrams; got != 100 {
		t.Errorf("normalized grams = %v, want 100", got)
	}
}

func TestClarificationEachMultipliesByQuantity(t *testing.T) {
	st := newFakeStore()
	rp := &fakeReplier{}
	e := New(
		fakeParser{items: []types.ParsedItem{{RawPhrase: "eggs"}}, conf: 0.9},
		fakeResolver{out: []types.ResolvedItem{portionPending("eggs", types.Macros{Calories: 155}, 2, "unit")}, need: 1},
		st, newFakePending(), rp, time.UTC, 0.6, "telegram", nil,
	)
	ctx := context.Background()
	_ = e.Handle(ctx, types.InboundMessage{UserID: "u1", Text: "2 eggs", ChannelMeta: map[string]string{"chat_id": "42"}})
	// "50g each" × 2 eggs = 100g → 155 kcal.
	if err := e.Handle(ctx, types.InboundMessage{UserID: "u1", Text: "50g each", ChannelMeta: map[string]string{"chat_id": "42"}}); err != nil {
		t.Fatalf("Handle error = %v", err)
	}
	if got := st.meals[0].Items[0].Parsed.NormalizedGrams; got != 100 {
		t.Errorf("normalized grams = %v, want 100 (50g each × 2)", got)
	}
}

func TestClarificationCancelDiscards(t *testing.T) {
	st := newFakeStore()
	rp := &fakeReplier{}
	e := New(
		fakeParser{items: []types.ParsedItem{{RawPhrase: "eggs"}}, conf: 0.9},
		fakeResolver{out: []types.ResolvedItem{portionPending("eggs", types.Macros{Calories: 155}, 2, "unit")}, need: 1},
		st, newFakePending(), rp, time.UTC, 0.6, "telegram", nil,
	)
	ctx := context.Background()
	_ = e.Handle(ctx, types.InboundMessage{UserID: "u1", Text: "2 eggs", ChannelMeta: map[string]string{"chat_id": "42"}})
	if err := e.Handle(ctx, types.InboundMessage{UserID: "u1", Text: "cancel", ChannelMeta: map[string]string{"chat_id": "42"}}); err != nil {
		t.Fatalf("Handle error = %v", err)
	}
	if len(st.meals) != 0 {
		t.Errorf("cancel must not log a meal, got %d", len(st.meals))
	}
	if !strings.Contains(rp.last(), "Discarded") {
		t.Errorf("expected a discard confirmation, got %q", rp.last())
	}
}

func TestClarificationUnknownFoodCorrected(t *testing.T) {
	st := newFakeStore()
	rp := &fakeReplier{}
	calls := 0
	res := fakeResolver{fn: func(items []types.ParsedItem) ([]types.ResolvedItem, int) {
		calls++
		if calls == 1 {
			// First parse: food unrecognized.
			return []types.ResolvedItem{{Parsed: items[0]}}, 1
		}
		// Correction re-resolves to a known food with a weight.
		return []types.ResolvedItem{resolved("chicken", types.Macros{Calories: 165})}, 0
	}}
	e := New(
		fakeParser{items: []types.ParsedItem{{RawPhrase: "xyz"}}, conf: 0.5},
		res, st, newFakePending(), rp, time.UTC, 0.6, "telegram", nil,
	)
	ctx := context.Background()
	if err := e.Handle(ctx, types.InboundMessage{UserID: "u1", Text: "xyz", ChannelMeta: map[string]string{"chat_id": "42"}}); err != nil {
		t.Fatalf("Handle error = %v", err)
	}
	if !strings.Contains(rp.last(), "don't recognize") {
		t.Errorf("expected an unrecognized-food question, got %q", rp.last())
	}
	if err := e.Handle(ctx, types.InboundMessage{UserID: "u1", Text: "100g chicken", ChannelMeta: map[string]string{"chat_id": "42"}}); err != nil {
		t.Fatalf("Handle error = %v", err)
	}
	if len(st.meals) != 1 || st.meals[0].Total().Calories != 165 {
		t.Errorf("expected a 165 kcal meal after correction, got %+v", st.meals)
	}
}

func TestHandleEmptyText(t *testing.T) {
	st := newFakeStore()
	rp := &fakeReplier{}
	e := New(fakeParser{}, fakeResolver{}, st, newFakePending(), rp, time.UTC, 0.6, "telegram", nil)
	if err := e.Handle(context.Background(), types.InboundMessage{UserID: "u1", Text: "  ", ChannelMeta: map[string]string{"chat_id": "42"}}); err != nil {
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
	e := New(fakeParser{}, fakeResolver{}, st, newFakePending(), rp, time.UTC, 0.6, "telegram", nil)

	msg := types.InboundMessage{UserID: "u1", Text: "/target kcal=3000 protein=180 carbs=350 fat=90", ChannelMeta: map[string]string{"chat_id": "42"}}
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
		st, newFakePending(), rp, time.UTC, 0.6, "telegram", nil,
	)
	at := time.Date(2026, 6, 17, 8, 0, 0, 0, time.UTC)
	for i := 0; i < 3; i++ {
		msg := types.InboundMessage{UserID: "u1", At: at, Text: "100g rice", ChannelMeta: map[string]string{"chat_id": "42"}}
		if err := e.Handle(context.Background(), msg); err != nil {
			t.Fatalf("Handle error = %v", err)
		}
	}
	if got := st.rollups["2026-06-17"].Consumed.Calories; got != 300 {
		t.Errorf("accumulated calories = %v, want 300", got)
	}
}
