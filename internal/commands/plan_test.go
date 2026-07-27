package commands

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/gsaraiva2109/dietdaemon/core/types"
)

// fakePlanStore is a minimal stub for /plan tests. It implements exactly the
// PlanStore interface -- four read methods, nothing that could write a plan,
// day-type, slot, or option -- so there is no method here a test could even
// call to prove a write happened; the interface itself is the proof /plan
// has no write path to the plan tables.
type fakePlanStore struct {
	user           types.User
	getUserErr     error
	dayType        types.DietPlanDayType
	resolveOK      bool
	resolveErr     error
	bundle         types.PlanBundle
	getBundleErr   error
	templates      map[string]types.MealTemplate
	getTemplateErr error
}

func (f *fakePlanStore) GetUser(_ context.Context, _ string) (types.User, error) {
	if f.getUserErr != nil {
		return types.User{}, f.getUserErr
	}
	return f.user, nil
}

func (f *fakePlanStore) ResolveDayType(_ context.Context, _, _ string) (types.DietPlanDayType, bool, error) {
	if f.resolveErr != nil {
		return types.DietPlanDayType{}, false, f.resolveErr
	}
	return f.dayType, f.resolveOK, nil
}

func (f *fakePlanStore) GetPlanBundle(_ context.Context, _ string) (types.PlanBundle, error) {
	if f.getBundleErr != nil {
		return types.PlanBundle{}, f.getBundleErr
	}
	return f.bundle, nil
}

func (f *fakePlanStore) GetTemplate(_ context.Context, templateID string) (types.MealTemplate, error) {
	if f.getTemplateErr != nil {
		return types.MealTemplate{}, f.getTemplateErr
	}
	if tmpl, ok := f.templates[templateID]; ok {
		return tmpl, nil
	}
	return types.MealTemplate{}, types.ErrNotFound
}

// bundleWithSlots builds a PlanBundle whose single day-type "dt-1" has the
// given slots, for tests that need GetPlanBundle to resolve them.
func bundleWithSlots(slots ...types.DietPlanSlotBundle) types.PlanBundle {
	return types.PlanBundle{
		Plan: types.DietPlan{ID: "p1", UserID: "u1"},
		DayTypes: []types.DietPlanDayTypeBundle{
			{DietPlanDayType: types.DietPlanDayType{ID: "dt-1", PlanID: "p1", Name: "Low-carb"}, Slots: slots},
		},
	}
}

func TestPlanCommand_NoActivePlan(t *testing.T) {
	store := &fakePlanStore{resolveOK: false}
	cmd := NewPlanCommand(store, time.UTC)

	reply, err := cmd.Handle(context.Background(), types.InboundMessage{UserID: "u1"}, "")
	if err != nil {
		t.Fatalf("Handle: %v", err)
	}
	if !strings.Contains(reply.Text, "No diet plan governs today") {
		t.Errorf("expected no-plan reply, got %q", reply.Text)
	}
}

func TestPlanCommand_ResolveDayTypeErr(t *testing.T) {
	store := &fakePlanStore{resolveErr: context.DeadlineExceeded}
	cmd := NewPlanCommand(store, time.UTC)

	_, err := cmd.Handle(context.Background(), types.InboundMessage{UserID: "u1"}, "")
	if err == nil {
		t.Fatal("expected error when ResolveDayType fails")
	}
}

func TestPlanCommand_GetPlanBundleErr(t *testing.T) {
	store := &fakePlanStore{
		resolveOK:    true,
		dayType:      types.DietPlanDayType{ID: "dt-1", PlanID: "p1", Name: "Low-carb"},
		getBundleErr: context.DeadlineExceeded,
	}
	cmd := NewPlanCommand(store, time.UTC)

	_, err := cmd.Handle(context.Background(), types.InboundMessage{UserID: "u1"}, "")
	if err == nil {
		t.Fatal("expected error when GetPlanBundle fails")
	}
}

func TestPlanCommand_TodayOverview(t *testing.T) {
	store := &fakePlanStore{
		resolveOK: true,
		dayType: types.DietPlanDayType{
			ID: "dt-1", PlanID: "p1", Name: "Low-carb",
			Targets: types.Macros{Calories: 1800, Protein: 160, Carbs: 120, Fat: 60},
		},
		bundle: bundleWithSlots(
			types.DietPlanSlotBundle{
				DietPlanSlot: types.DietPlanSlot{ID: "sl-1", DayTypeID: "dt-1", Label: "Almoço", TimeOfDay: "12:30"},
				Options:      []types.DietPlanSlotOption{{ID: "opt-1", SlotID: "sl-1", Label: "Frango e arroz", TemplateID: "tmpl-1"}},
			},
			types.DietPlanSlotBundle{
				DietPlanSlot: types.DietPlanSlot{ID: "sl-2", DayTypeID: "dt-1", Label: "Jantar", TimeOfDay: "19:00"},
			},
		),
	}
	cmd := NewPlanCommand(store, time.UTC)

	reply, err := cmd.Handle(context.Background(), types.InboundMessage{UserID: "u1"}, "")
	if err != nil {
		t.Fatalf("Handle: %v", err)
	}
	if !strings.Contains(reply.Text, "Low-carb") || !strings.Contains(reply.Text, "1800 kcal") {
		t.Errorf("expected day-type name and kcal target, got %q", reply.Text)
	}
	if !strings.Contains(reply.Text, "Almoço") || !strings.Contains(reply.Text, "Jantar") {
		t.Errorf("expected both slots listed, got %q", reply.Text)
	}
	if !strings.Contains(reply.Text, "Frango e arroz") {
		t.Errorf("expected option label in slot summary, got %q", reply.Text)
	}
	// The overview never expands option contents to itemized foods -- that
	// level of detail is reserved for a specific-meal query.
	if strings.Contains(reply.Text, "à vontade") {
		t.Errorf("today overview should not itemize foods, got %q", reply.Text)
	}
}

func TestPlanCommand_MealQuery(t *testing.T) {
	store := &fakePlanStore{
		resolveOK: true,
		dayType:   types.DietPlanDayType{ID: "dt-1", PlanID: "p1", Name: "Low-carb"},
		bundle: bundleWithSlots(types.DietPlanSlotBundle{
			DietPlanSlot: types.DietPlanSlot{ID: "sl-1", DayTypeID: "dt-1", Label: "Almoço", TimeOfDay: "12:30"},
			Options:      []types.DietPlanSlotOption{{ID: "opt-1", SlotID: "sl-1", Label: "Frango e arroz", TemplateID: "tmpl-1"}},
		}),
		templates: map[string]types.MealTemplate{
			"tmpl-1": {
				ID: "tmpl-1", Name: "Frango e arroz",
				Items: []types.ResolvedItem{
					{Parsed: types.ParsedItem{RawPhrase: "frango", NormalizedGrams: 200}, Match: types.FoodMatch{Name: "Frango grelhado"}, Macros: types.Macros{Calories: 330, Protein: 62}},
					{Parsed: types.ParsedItem{RawPhrase: "salada", NormalizedGrams: 0}, Match: types.FoodMatch{Name: "Salada"}, Macros: types.Macros{}},
				},
			},
		},
	}
	cmd := NewPlanCommand(store, time.UTC)

	reply, err := cmd.Handle(context.Background(), types.InboundMessage{UserID: "u1"}, "almoço")
	if err != nil {
		t.Fatalf("Handle: %v", err)
	}
	if !strings.Contains(reply.Text, "Frango e arroz") {
		t.Errorf("expected option label, got %q", reply.Text)
	}
	if !strings.Contains(reply.Text, "200g Frango grelhado") {
		t.Errorf("expected itemized food with grams, got %q", reply.Text)
	}
	if !strings.Contains(reply.Text, "Salada (à vontade)") {
		t.Errorf("expected ad libitum item rendered as such, got %q", reply.Text)
	}
	if !strings.Contains(reply.Text, "330 kcal") {
		t.Errorf("expected option macro total, got %q", reply.Text)
	}
}

func TestPlanCommand_MealQueryNoMatch(t *testing.T) {
	store := &fakePlanStore{
		resolveOK: true,
		dayType:   types.DietPlanDayType{ID: "dt-1", PlanID: "p1", Name: "Low-carb"},
		bundle: bundleWithSlots(types.DietPlanSlotBundle{
			DietPlanSlot: types.DietPlanSlot{ID: "sl-1", DayTypeID: "dt-1", Label: "Almoço", TimeOfDay: "12:30"},
		}),
	}
	cmd := NewPlanCommand(store, time.UTC)

	reply, err := cmd.Handle(context.Background(), types.InboundMessage{UserID: "u1"}, "café da manhã")
	if err != nil {
		t.Fatalf("Handle: %v", err)
	}
	if !strings.Contains(reply.Text, "No meal slot matching") {
		t.Errorf("expected no-match reply, got %q", reply.Text)
	}
}

// TestPlanCommand_NextSlotID exercises the pure time-of-day math directly,
// since Handle's own clock (time.Now) isn't injectable and a test pinned to
// wall-clock time would be flaky.
func TestPlanCommand_NextSlotID(t *testing.T) {
	slots := []types.DietPlanSlotBundle{
		{DietPlanSlot: types.DietPlanSlot{ID: "breakfast", TimeOfDay: "07:00"}},
		{DietPlanSlot: types.DietPlanSlot{ID: "lunch", TimeOfDay: "12:30"}},
		{DietPlanSlot: types.DietPlanSlot{ID: "dinner", TimeOfDay: "19:00"}},
		{DietPlanSlot: types.DietPlanSlot{ID: "malformed", TimeOfDay: "not-a-time"}},
	}

	cases := []struct {
		now  string
		want string
	}{
		{"2026-07-27T06:00:00Z", "breakfast"},
		{"2026-07-27T08:00:00Z", "lunch"},
		{"2026-07-27T13:00:00Z", "dinner"},
		{"2026-07-27T20:00:00Z", ""}, // every slot today has passed
	}
	for _, tc := range cases {
		now, err := time.Parse(time.RFC3339, tc.now)
		if err != nil {
			t.Fatalf("parse test time: %v", err)
		}
		if got := nextSlotID(slots, now); got != tc.want {
			t.Errorf("nextSlotID at %s = %q, want %q", tc.now, got, tc.want)
		}
	}
}

func TestParseTimeOfDay(t *testing.T) {
	if mins, ok := parseTimeOfDay("12:30"); !ok || mins != 750 {
		t.Errorf("parseTimeOfDay(12:30) = (%d, %v), want (750, true)", mins, ok)
	}
	if _, ok := parseTimeOfDay("garbage"); ok {
		t.Error("parseTimeOfDay(garbage) should fail to parse")
	}
}
