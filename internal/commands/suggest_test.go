package commands

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/gsaraiva2109/dietdaemon/core/types"
)

// fakeSuggestEngine is a minimal stub for /suggest tests.
type fakeSuggestEngine struct {
	sug         types.MealSuggestion
	err         error
	lastFoodIDs []string
}

func (f *fakeSuggestEngine) Suggest(_ context.Context, _ string) (types.MealSuggestion, error) {
	return f.sug, f.err
}

// lastFoodIDs records the food IDs passed to the most recent
// SuggestFromIngredients call, so tests can assert on ingredient resolution.
func (f *fakeSuggestEngine) SuggestFromIngredients(_ context.Context, _ string, foodIDs []string) (types.MealSuggestion, error) {
	f.lastFoodIDs = foodIDs
	return f.sug, f.err
}

// fakeFoodSearcher maps a lowercased query to a single top match, mimicking
// the top-result-per-name behavior SuggestCommand relies on.
type fakeFoodSearcher struct {
	matches map[string]types.FoodDetail
}

func (f *fakeFoodSearcher) SearchFoods(_ context.Context, _ string, query string) ([]types.FoodDetail, error) {
	if m, ok := f.matches[query]; ok {
		return []types.FoodDetail{m}, nil
	}
	return nil, nil
}

func TestSuggestCommand_HappyPath(t *testing.T) {
	engine := &fakeSuggestEngine{
		sug: types.MealSuggestion{
			Remaining: types.Macros{Calories: 450, Protein: 30, Carbs: 40, Fat: 15},
			Candidates: []types.SuggestedCombo{
				{
					Items: []types.SuggestedItem{
						{FoodID: "f1", Name: "Grilled chicken breast", Grams: 150},
						{FoodID: "f2", Name: "Rice", Grams: 100},
					},
					Macros: types.Macros{Calories: 420, Protein: 35, Carbs: 38, Fat: 8},
					Score:  0.9,
				},
			},
			Message: "You've got room for a protein-heavy meal.",
			Source:  "rules",
		},
	}
	cmd := NewSuggestCommand(engine, &fakeFoodSearcher{})
	meta := map[string]string{"chat_id": "123"}

	reply, err := cmd.Handle(context.Background(), types.InboundMessage{UserID: "u1", ChannelMeta: meta}, "")
	if err != nil {
		t.Fatalf("Handle: %v", err)
	}
	if !strings.Contains(reply.Text, "You've got room for a protein-heavy meal.") {
		t.Errorf("expected reply to contain engine message, got %q", reply.Text)
	}
	if !strings.Contains(reply.Text, "450 kcal") {
		t.Errorf("expected reply to mention remaining calories, got %q", reply.Text)
	}
	if !strings.Contains(reply.Text, "Grilled chicken breast") {
		t.Errorf("expected reply to mention candidate item, got %q", reply.Text)
	}
	if reply.ChannelMeta["chat_id"] != "123" {
		t.Errorf("expected ChannelMeta to be echoed back, got %v", reply.ChannelMeta)
	}
}

func TestSuggestCommand_EngineError(t *testing.T) {
	engine := &fakeSuggestEngine{err: errors.New("store unreachable")}
	cmd := NewSuggestCommand(engine, &fakeFoodSearcher{})
	meta := map[string]string{"chat_id": "456"}

	reply, err := cmd.Handle(context.Background(), types.InboundMessage{UserID: "u1", ChannelMeta: meta}, "")
	if err != nil {
		t.Fatalf("Handle should not propagate engine errors, got: %v", err)
	}
	if reply.Text != suggestFallback {
		t.Errorf("reply.Text = %q, want fallback %q", reply.Text, suggestFallback)
	}
	if reply.ChannelMeta["chat_id"] != "456" {
		t.Errorf("expected ChannelMeta to be echoed back, got %v", reply.ChannelMeta)
	}
}

func TestSuggestCommand_EmptyMessage(t *testing.T) {
	engine := &fakeSuggestEngine{sug: types.MealSuggestion{}}
	cmd := NewSuggestCommand(engine, &fakeFoodSearcher{})

	reply, err := cmd.Handle(context.Background(), types.InboundMessage{UserID: "u1"}, "")
	if err != nil {
		t.Fatalf("Handle: %v", err)
	}
	if reply.Text != suggestFallback {
		t.Errorf("reply.Text = %q, want fallback %q", reply.Text, suggestFallback)
	}
}

func TestSuggestCommand_Metadata(t *testing.T) {
	cmd := NewSuggestCommand(&fakeSuggestEngine{}, &fakeFoodSearcher{})

	if cmd.Name() != "/suggest" {
		t.Errorf("Name() = %q, want /suggest", cmd.Name())
	}
	aliases := cmd.Aliases()
	if len(aliases) != 1 || aliases[0] != "/eat" {
		t.Errorf("Aliases() = %v, want [/eat]", aliases)
	}
	if cmd.Help() != types.I18nKey("cmd.suggest.usage") {
		t.Errorf("Help() = %q, want cmd.suggest.usage", cmd.Help())
	}
}

func TestSuggestCommand_IngredientArgsResolveViaSearch(t *testing.T) {
	engine := &fakeSuggestEngine{
		sug: types.MealSuggestion{Message: "Use what's on hand."},
	}
	searcher := &fakeFoodSearcher{matches: map[string]types.FoodDetail{
		"chicken": {FoodID: "chicken-id", Name: "Chicken breast"},
		"rice":    {FoodID: "rice-id", Name: "Rice"},
		"eggs":    {FoodID: "eggs-id", Name: "Eggs"},
	}}
	cmd := NewSuggestCommand(engine, searcher)

	reply, err := cmd.Handle(context.Background(), types.InboundMessage{UserID: "u1"}, "chicken, rice, eggs")
	if err != nil {
		t.Fatalf("Handle: %v", err)
	}
	if !strings.Contains(reply.Text, "Use what's on hand.") {
		t.Errorf("expected reply to contain engine message, got %q", reply.Text)
	}

	want := []string{"chicken-id", "rice-id", "eggs-id"}
	if len(engine.lastFoodIDs) != len(want) {
		t.Fatalf("resolved food IDs = %v, want %v", engine.lastFoodIDs, want)
	}
	for i, id := range want {
		if engine.lastFoodIDs[i] != id {
			t.Errorf("resolved food IDs = %v, want %v", engine.lastFoodIDs, want)
			break
		}
	}
}

func TestSuggestCommand_IngredientArgsSkipUnresolvedNames(t *testing.T) {
	engine := &fakeSuggestEngine{sug: types.MealSuggestion{Message: "ok"}}
	searcher := &fakeFoodSearcher{matches: map[string]types.FoodDetail{
		"chicken": {FoodID: "chicken-id", Name: "Chicken breast"},
	}}
	cmd := NewSuggestCommand(engine, searcher)

	_, err := cmd.Handle(context.Background(), types.InboundMessage{UserID: "u1"}, "chicken, nonexistentfood")
	if err != nil {
		t.Fatalf("Handle: %v", err)
	}
	if len(engine.lastFoodIDs) != 1 || engine.lastFoodIDs[0] != "chicken-id" {
		t.Errorf("resolved food IDs = %v, want [chicken-id] (unresolved name skipped)", engine.lastFoodIDs)
	}
}
