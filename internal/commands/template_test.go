package commands

import (
	"context"
	"strings"
	"testing"

	"github.com/gsaraiva2109/dietdaemon/core/types"
)

// fakeTemplateStore is a minimal stub for /template tests.
type fakeTemplateStore struct {
	templates     []types.MealTemplate
	savedTemplate *types.MealTemplate
	saveErr       error
}

func (f *fakeTemplateStore) GetTemplates(_ context.Context, _ string) ([]types.MealTemplate, error) {
	return f.templates, nil
}
func (f *fakeTemplateStore) GetTemplate(_ context.Context, _ string) (types.MealTemplate, error) {
	return types.MealTemplate{}, types.ErrNotFound
}
func (f *fakeTemplateStore) SaveTemplate(_ context.Context, tmpl types.MealTemplate) error {
	if f.saveErr != nil {
		return f.saveErr
	}
	f.savedTemplate = &tmpl
	return nil
}
func (f *fakeTemplateStore) LogTemplateUse(_ context.Context, _ types.TemplateLog) error { return nil }

// fakeTemplateComposer is a minimal TemplateComposer stub.
type fakeTemplateComposer struct {
	items              []types.ResolvedItem
	needsClarification int
	err                error
}

func (f *fakeTemplateComposer) ParseAndResolve(_ context.Context, _, _, _ string) ([]types.ResolvedItem, int, error) {
	return f.items, f.needsClarification, f.err
}

func TestTemplateSave_FullResolution(t *testing.T) {
	store := &fakeTemplateStore{}
	composer := &fakeTemplateComposer{
		items: []types.ResolvedItem{
			{
				Parsed: types.ParsedItem{RawPhrase: "chicken breast", NormalizedGrams: 200},
				Match:  types.FoodMatch{FoodID: "chicken", Name: "Chicken Breast", Source: "taco"},
				Macros: types.Macros{Calories: 330, Protein: 62, Carbs: 0, Fat: 7.2},
			},
			{
				Parsed: types.ParsedItem{RawPhrase: "white rice", NormalizedGrams: 150},
				Match:  types.FoodMatch{FoodID: "rice", Name: "White Rice", Source: "taco"},
				Macros: types.Macros{Calories: 195, Protein: 3.9, Carbs: 42, Fat: 0.4},
			},
		},
	}
	cmd := NewTemplateCommand(store, nil, composer)

	reply, err := cmd.Handle(context.Background(), types.InboundMessage{UserID: "u1"}, "save My Recipe: 200g chicken breast, 150g white rice")
	if err != nil {
		t.Fatalf("Handle: %v", err)
	}
	if store.savedTemplate == nil {
		t.Fatal("expected SaveTemplate to be called")
	}
	if store.savedTemplate.Name != "My Recipe" {
		t.Errorf("template name = %q, want %q", store.savedTemplate.Name, "My Recipe")
	}
	if len(store.savedTemplate.Items) != 2 {
		t.Fatalf("items = %d, want 2", len(store.savedTemplate.Items))
	}
	if store.savedTemplate.Items[0].Match.Name != "Chicken Breast" {
		t.Errorf("first item = %q, want %q", store.savedTemplate.Items[0].Match.Name, "Chicken Breast")
	}
	if !strings.Contains(reply.Text, "525") {
		t.Errorf("expected reply to show total kcal (~525), got %q", reply.Text)
	}
}

func TestTemplateSave_PartialResolution(t *testing.T) {
	store := &fakeTemplateStore{}
	composer := &fakeTemplateComposer{
		items:              nil,
		needsClarification: 1, // one item couldn't be resolved
	}
	cmd := NewTemplateCommand(store, nil, composer)

	reply, err := cmd.Handle(context.Background(), types.InboundMessage{UserID: "u1"}, "save Bad Recipe: 200g unknownfood")
	if err != nil {
		t.Fatalf("Handle: %v", err)
	}
	if store.savedTemplate != nil {
		t.Fatal("expected SaveTemplate NOT to be called on partial resolution")
	}
	if !strings.Contains(strings.ToLower(reply.Text), "couldn't fully resolve") {
		t.Errorf("expected clarification reply, got %q", reply.Text)
	}
}

func TestTemplateSave_NoItems(t *testing.T) {
	store := &fakeTemplateStore{}
	composer := &fakeTemplateComposer{
		items:              nil,
		needsClarification: 0,
	}
	cmd := NewTemplateCommand(store, nil, composer)

	reply, err := cmd.Handle(context.Background(), types.InboundMessage{UserID: "u1"}, "save Empty: xyz")
	if err != nil {
		t.Fatalf("Handle: %v", err)
	}
	if store.savedTemplate != nil {
		t.Fatal("expected SaveTemplate NOT to be called with empty items")
	}
	if !strings.Contains(strings.ToLower(reply.Text), "couldn't fully resolve") {
		t.Errorf("expected rejection reply, got %q", reply.Text)
	}
}

func TestTemplateSave_MissingColon(t *testing.T) {
	store := &fakeTemplateStore{}
	composer := &fakeTemplateComposer{}
	cmd := NewTemplateCommand(store, nil, composer)

	reply, err := cmd.Handle(context.Background(), types.InboundMessage{UserID: "u1"}, "save NoColonHere")
	if err != nil {
		t.Fatalf("Handle: %v", err)
	}
	if store.savedTemplate != nil {
		t.Fatal("expected SaveTemplate NOT to be called without colon separator")
	}
	if !strings.Contains(reply.Text, "Usage:") {
		t.Errorf("expected usage reply, got %q", reply.Text)
	}
}
