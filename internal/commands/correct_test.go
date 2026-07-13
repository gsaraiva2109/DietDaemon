package commands

import (
	"context"
	"strings"
	"testing"

	"github.com/gsaraiva2109/dietdaemon/core/types"
)

// fakeCorrectStore is a minimal CorrectStore stub for /correct tests.
type fakeCorrectStore struct {
	meals []types.Meal // returned by RecentMeals

	correctCalled bool
	gotUserID     string
	gotMealID     string
	gotItemIndex  int
	gotCorrected  types.ResolvedItem
	correctErr    error
	feedback      types.CorrectionFeedback
	pendingErr    error
}

func (f *fakeCorrectStore) RecentMeals(_ context.Context, _ string, limit int) ([]types.Meal, error) {
	if len(f.meals) > limit {
		return f.meals[:limit], nil
	}
	return f.meals, nil
}

func (f *fakeCorrectStore) CorrectMealItem(_ context.Context, userID, mealID string, itemIndex int, corrected types.ResolvedItem) error {
	f.correctCalled = true
	f.gotUserID = userID
	f.gotMealID = mealID
	f.gotItemIndex = itemIndex
	f.gotCorrected = corrected
	return f.correctErr
}

func (f *fakeCorrectStore) CorrectMealItemWithFeedback(ctx context.Context, userID, mealID string, itemIndex int, corrected types.ResolvedItem) (types.CorrectionFeedback, error) {
	if err := f.CorrectMealItem(ctx, userID, mealID, itemIndex, corrected); err != nil {
		return types.CorrectionFeedback{}, err
	}
	return f.feedback, nil
}
func (f *fakeCorrectStore) ConfirmPendingAlias(context.Context, string, string) error {
	return f.pendingErr
}
func (f *fakeCorrectStore) RejectPendingAlias(context.Context, string, string) error {
	return f.pendingErr
}

// fakeCorrectResolver is a minimal CorrectResolver stub for /correct tests.
type fakeCorrectResolver struct {
	result types.ResolvedItem
}

func (f *fakeCorrectResolver) Resolve(_ context.Context, _ string, items []types.ParsedItem) ([]types.ResolvedItem, int) {
	out := make([]types.ResolvedItem, len(items))
	for i := range items {
		out[i] = f.result
	}
	return out, 0
}

func TestCorrectCommand_HappyPath(t *testing.T) {
	store := &fakeCorrectStore{
		meals: []types.Meal{{ID: "meal-1", UserID: "u1"}},
	}
	resolver := &fakeCorrectResolver{
		result: types.ResolvedItem{
			Parsed: types.ParsedItem{RawPhrase: "grilled chicken breast", NormalizedGrams: 150},
			Match:  types.FoodMatch{FoodID: "chicken", Name: "Grilled Chicken Breast", Source: "taco"},
			Macros: types.Macros{Calories: 247.5, Protein: 46.5, Carbs: 0, Fat: 5.4},
		},
	}
	cmd := NewCorrectCommand(store, resolver)

	reply, err := cmd.Handle(context.Background(), types.InboundMessage{UserID: "u1"}, "0 150g grilled chicken breast")
	if err != nil {
		t.Fatalf("Handle: %v", err)
	}
	if !store.correctCalled {
		t.Fatal("expected CorrectMealItem to be called")
	}
	if store.gotUserID != "u1" || store.gotMealID != "meal-1" || store.gotItemIndex != 0 {
		t.Fatalf("unexpected call args: userID=%s mealID=%s itemIndex=%d", store.gotUserID, store.gotMealID, store.gotItemIndex)
	}
	if store.gotCorrected.Match.Name != "Grilled Chicken Breast" {
		t.Fatalf("unexpected corrected item: %+v", store.gotCorrected)
	}
	if !strings.Contains(reply.Text, "Grilled Chicken Breast") {
		t.Errorf("expected reply to mention corrected food, got %q", reply.Text)
	}
	if !strings.Contains(reply.Text, "248") {
		t.Errorf("expected reply to show corrected kcal, got %q", reply.Text)
	}
}

func TestCorrectCommand_NoRecentMeal(t *testing.T) {
	store := &fakeCorrectStore{meals: nil}
	resolver := &fakeCorrectResolver{}
	cmd := NewCorrectCommand(store, resolver)

	reply, err := cmd.Handle(context.Background(), types.InboundMessage{UserID: "u1"}, "0 150g grilled chicken breast")
	if err != nil {
		t.Fatalf("Handle: %v", err)
	}
	if store.correctCalled {
		t.Fatal("expected CorrectMealItem NOT to be called when there is no recent meal")
	}
	if !strings.Contains(strings.ToLower(reply.Text), "no recent meal") {
		t.Errorf("expected graceful no-meal reply, got %q", reply.Text)
	}
}

func TestCorrectCommand_BadGramsFormat(t *testing.T) {
	store := &fakeCorrectStore{meals: []types.Meal{{ID: "meal-1", UserID: "u1"}}}
	resolver := &fakeCorrectResolver{}
	cmd := NewCorrectCommand(store, resolver)

	reply, err := cmd.Handle(context.Background(), types.InboundMessage{UserID: "u1"}, "0 150 grilled chicken breast")
	if err != nil {
		t.Fatalf("Handle: %v", err)
	}
	if store.correctCalled {
		t.Fatal("expected CorrectMealItem NOT to be called with a bad grams format")
	}
	if !strings.Contains(strings.ToLower(reply.Text), "grams") {
		t.Errorf("expected reply to complain about grams format, got %q", reply.Text)
	}
}

func TestCorrectCommand_ConflictOffersReplacement(t *testing.T) {
	store := &fakeCorrectStore{meals: []types.Meal{{ID: "meal-1"}}, feedback: types.CorrectionFeedback{PendingAliasID: "pending-1"}}
	resolver := &fakeCorrectResolver{result: types.ResolvedItem{Match: types.FoodMatch{FoodID: "chicken", Name: "Chicken"}, Macros: types.Macros{Calories: 1}}}
	reply, err := NewCorrectCommand(store, resolver).Handle(context.Background(), types.InboundMessage{UserID: "u1"}, "0 150g chicken")
	if err != nil || reply.Markup == nil || len(reply.Markup.InlineKeyboard) != 1 {
		t.Fatalf("expected replacement buttons, reply=%+v err=%v", reply, err)
	}
	if got := reply.Markup.InlineKeyboard[0][0].CallbackData; got != "/correct alias accept pending-1" {
		t.Fatalf("accept callback = %q", got)
	}
}
