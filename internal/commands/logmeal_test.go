package commands

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/gsaraiva2109/dietdaemon/core/types"
)

// fakeLogMealEngine is a test double for LogMealEngine.
type fakeLogMealEngine struct {
	items              []types.ResolvedItem
	needsClarification int
	parseErr           error
	meal               types.Meal
	saveErr            error
}

func (f *fakeLogMealEngine) ParseAndResolve(_ context.Context, _, _, _ string) ([]types.ResolvedItem, int, error) {
	return f.items, f.needsClarification, f.parseErr
}

func (f *fakeLogMealEngine) LogMealFromItems(_ context.Context, _ string, _ time.Time, _ string, _ float64, _ []types.ResolvedItem) (types.Meal, error) {
	return f.meal, f.saveErr
}

func resolvedItem(name, foodID string, grams float64, macros types.Macros) types.ResolvedItem {
	return types.ResolvedItem{
		Parsed: types.ParsedItem{RawPhrase: name, NormalizedGrams: grams},
		Match:  types.FoodMatch{FoodID: foodID, Name: name},
		Macros: macros,
	}
}

func ambiguousItemNoMatch(phrase string) types.ResolvedItem {
	return types.ResolvedItem{
		Parsed: types.ParsedItem{RawPhrase: phrase, NormalizedGrams: 0},
		Match:  types.FoodMatch{FoodID: ""},
	}
}

func ambiguousItemNoPortion(name, foodID string) types.ResolvedItem {
	return types.ResolvedItem{
		Parsed: types.ParsedItem{RawPhrase: name, NormalizedGrams: 0},
		Match:  types.FoodMatch{FoodID: foodID, Name: name},
	}
}

func TestLogMealCommand_HappyPath(t *testing.T) {
	items := []types.ResolvedItem{
		resolvedItem("grilled chicken", "f1", 200, types.Macros{Calories: 330, Protein: 62, Carbs: 0, Fat: 7}),
		resolvedItem("banana", "f2", 120, types.Macros{Calories: 107, Protein: 1.3, Carbs: 27, Fat: 0.4}),
	}
	engine := &fakeLogMealEngine{
		items: items,
		meal: types.Meal{
			ID:      "m1",
			RawText: "200g grilled chicken and a banana",
			Items:   items,
		},
	}
	cmd := NewLogMealCommand(engine)
	meta := map[string]string{"chat_id": "123"}

	reply, err := cmd.Handle(context.Background(), types.InboundMessage{UserID: "u1", ChannelMeta: meta}, "200g grilled chicken and a banana")
	if err != nil {
		t.Fatalf("Handle: %v", err)
	}
	if !strings.Contains(reply.Text, "Logged 2 item(s)") {
		t.Errorf("expected reply to mention item count, got %q", reply.Text)
	}
	if !strings.Contains(reply.Text, "437 kcal") {
		t.Errorf("expected reply to mention total calories, got %q", reply.Text)
	}
}

func TestLogMealCommand_AmbiguousItems(t *testing.T) {
	items := []types.ResolvedItem{
		resolvedItem("rice", "f1", 100, types.Macros{Calories: 130, Protein: 2.7, Carbs: 28, Fat: 0.3}),
		ambiguousItemNoMatch("frango grelhado"),
		ambiguousItemNoPortion("beans", "f3"),
	}
	engine := &fakeLogMealEngine{
		items:              items,
		needsClarification: 2,
	}
	cmd := NewLogMealCommand(engine)

	reply, err := cmd.Handle(context.Background(), types.InboundMessage{UserID: "u1"}, "100g rice, frango grelhado, beans")
	if err != nil {
		t.Fatalf("Handle: %v", err)
	}
	if !strings.Contains(reply.Text, "need clarification") {
		t.Errorf("expected clarification text, got %q", reply.Text)
	}
	if !strings.Contains(reply.Text, "frango grelhado") {
		t.Errorf("expected ambiguous item in reply, got %q", reply.Text)
	}
	if !strings.Contains(reply.Text, "beans") {
		t.Errorf("expected beans in reply, got %q", reply.Text)
	}
	// Should not call LogMealFromItems when ambiguous.
	if engine.meal.ID != "" {
		t.Error("LogMealFromItems was called despite ambiguous items")
	}
}

func TestLogMealCommand_ParseError(t *testing.T) {
	engine := &fakeLogMealEngine{parseErr: errors.New("parser down")}
	cmd := NewLogMealCommand(engine)

	_, err := cmd.Handle(context.Background(), types.InboundMessage{UserID: "u1"}, "some food")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "logmeal: parse") {
		t.Errorf("expected wrapped parse error, got %v", err)
	}
}

func TestLogMealCommand_SaveError(t *testing.T) {
	items := []types.ResolvedItem{
		resolvedItem("chicken", "f1", 200, types.Macros{Calories: 330, Protein: 62, Carbs: 0, Fat: 7}),
	}
	engine := &fakeLogMealEngine{
		items:   items,
		saveErr: errors.New("store unreachable"),
	}
	cmd := NewLogMealCommand(engine)

	_, err := cmd.Handle(context.Background(), types.InboundMessage{UserID: "u1"}, "200g chicken")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "logmeal: save") {
		t.Errorf("expected wrapped save error, got %v", err)
	}
}

func TestLogMealCommand_EmptyArgs(t *testing.T) {
	engine := &fakeLogMealEngine{}
	cmd := NewLogMealCommand(engine)

	reply, err := cmd.Handle(context.Background(), types.InboundMessage{UserID: "u1"}, "")
	if err != nil {
		t.Fatalf("Handle: %v", err)
	}
	if !strings.Contains(reply.Text, "Tell me what you ate") {
		t.Errorf("expected usage hint, got %q", reply.Text)
	}
}

func TestLogMealCommand_NoItems(t *testing.T) {
	engine := &fakeLogMealEngine{items: nil}
	cmd := NewLogMealCommand(engine)

	reply, err := cmd.Handle(context.Background(), types.InboundMessage{UserID: "u1"}, "xyzzy nonsense text")
	if err != nil {
		t.Fatalf("Handle: %v", err)
	}
	if !strings.Contains(reply.Text, "Couldn't read any food") {
		t.Errorf("expected no-food message, got %q", reply.Text)
	}
}

func TestLogMealCommand_NoItemsEmptySlice(t *testing.T) {
	engine := &fakeLogMealEngine{items: []types.ResolvedItem{}}
	cmd := NewLogMealCommand(engine)

	reply, err := cmd.Handle(context.Background(), types.InboundMessage{UserID: "u1"}, "xyzzy")
	if err != nil {
		t.Fatalf("Handle: %v", err)
	}
	if !strings.Contains(reply.Text, "Couldn't read any food") {
		t.Errorf("expected no-food message, got %q", reply.Text)
	}
}

func TestLogMealCommand_Metadata(t *testing.T) {
	cmd := NewLogMealCommand(&fakeLogMealEngine{})

	if cmd.Name() != "/logmeal" {
		t.Errorf("Name() = %q, want /logmeal", cmd.Name())
	}
	if aliases := cmd.Aliases(); aliases != nil {
		t.Errorf("Aliases() = %v, want nil", aliases)
	}
	if cmd.Help() != types.I18nKey("cmd.logmeal.title") {
		t.Errorf("Help() = %q, want cmd.logmeal.title", cmd.Help())
	}
}

func TestSummaryText(t *testing.T) {
	items := []types.ResolvedItem{
		resolvedItem("chicken", "f1", 200, types.Macros{Calories: 330, Protein: 62, Carbs: 0, Fat: 7}),
	}
	meal := types.Meal{RawText: "200g chicken", Items: items}
	got := summaryText(meal)
	if !strings.Contains(got, "Logged 1 item") {
		t.Errorf("expected item count, got %q", got)
	}
	if !strings.Contains(got, "330 kcal") {
		t.Errorf("expected calories, got %q", got)
	}
}

func TestDescribeAmbiguity(t *testing.T) {
	items := []types.ResolvedItem{
		ambiguousItemNoMatch("frango"),
		ambiguousItemNoPortion("rice", "f2"),
	}
	got := describeAmbiguity(items)
	if !strings.Contains(got, "frango") {
		t.Errorf("expected frango in output, got %q", got)
	}
	if !strings.Contains(got, "rice") {
		t.Errorf("expected rice in output, got %q", got)
	}
	if !strings.Contains(got, "not recognized") {
		t.Errorf("expected 'not recognized' for unmatched item, got %q", got)
	}
	if !strings.Contains(got, "needs portion") {
		t.Errorf("expected 'needs portion' for missing grams, got %q", got)
	}
}
