package api

import (
	"errors"
	"net/http"
	"testing"

	"github.com/gsaraiva2109/dietdaemon/core/types"
)

// --- handleListPlans / handleCreatePlan ---

func TestHandleListPlans(t *testing.T) {
	store := newFakeMealStore()
	store.plans = map[string]types.DietPlan{
		"p1": {ID: "p1", UserID: "test-user", Name: "Cutting cycle"},
	}
	h := newHandler(store, &fakeMealLogger{})

	rec := doRequest(h, "GET", "/api/v1/plans", nil, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	got := decodeJSON[[]types.DietPlan](t, rec)
	if len(got) != 1 || got[0].ID != "p1" {
		t.Errorf("unexpected plans: %+v", got)
	}
}

func TestHandleListPlansUnauthorized(t *testing.T) {
	store := newFakeMealStore()
	h := newHandler(store, &fakeMealLogger{})

	rec := doRequest(h, "GET", "/api/v1/plans", nil, map[string]string{"Authorization": ""})
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rec.Code)
	}
}

func TestHandleCreatePlan(t *testing.T) {
	store := newFakeMealStore()
	h := newHandler(store, &fakeMealLogger{})

	body := map[string]any{
		"name": "Cutting cycle", "valid_from": "2026-01-05", "cycle_anchor_date": "2026-01-05",
	}
	rec := doRequest(h, "POST", "/api/v1/plans", body, nil)
	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rec.Code, rec.Body.String())
	}
	got := decodeJSON[types.DietPlan](t, rec)
	if got.UserID != "test-user" || got.Name != "Cutting cycle" {
		t.Errorf("unexpected plan: %+v", got)
	}
}

func TestHandleCreatePlanMissingName(t *testing.T) {
	store := newFakeMealStore()
	h := newHandler(store, &fakeMealLogger{})

	body := map[string]any{"valid_from": "2026-01-05", "cycle_anchor_date": "2026-01-05"}
	rec := doRequest(h, "POST", "/api/v1/plans", body, nil)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("missing name expected 400, got %d", rec.Code)
	}
}

func TestHandleCreatePlanBadDates(t *testing.T) {
	store := newFakeMealStore()
	h := newHandler(store, &fakeMealLogger{})

	body := map[string]any{"name": "x", "valid_from": "not-a-date", "cycle_anchor_date": "2026-01-05"}
	rec := doRequest(h, "POST", "/api/v1/plans", body, nil)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("bad valid_from expected 400, got %d", rec.Code)
	}
}

func TestHandleCreatePlanNonEmptyCyclePattern(t *testing.T) {
	store := newFakeMealStore()
	h := newHandler(store, &fakeMealLogger{})

	body := map[string]any{
		"name": "x", "valid_from": "2026-01-05", "cycle_anchor_date": "2026-01-05",
		"cycle_pattern": []string{"dt-1"},
	}
	rec := doRequest(h, "POST", "/api/v1/plans", body, nil)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("non-empty cycle_pattern on create expected 400, got %d", rec.Code)
	}
}

func TestHandleCreatePlanCallsRefreshTodayTargets(t *testing.T) {
	store := newFakeMealStore()
	store.refreshTodayTargetsErr = errors.New("refresh failed")
	h := newHandler(store, &fakeMealLogger{})

	body := map[string]any{"name": "x", "valid_from": "2026-01-05", "cycle_anchor_date": "2026-01-05"}
	rec := doRequest(h, "POST", "/api/v1/plans", body, nil)
	if rec.Code != http.StatusInternalServerError {
		t.Errorf("RefreshTodayTargets failure expected 500, got %d", rec.Code)
	}
}

// --- handleGetPlan ---

func TestHandleGetPlan(t *testing.T) {
	store := newFakeMealStore()
	store.planBundles = map[string]types.PlanBundle{
		"p1": {Plan: types.DietPlan{ID: "p1", UserID: "test-user", Name: "Cutting cycle"}, DayTypes: []types.DietPlanDayTypeBundle{}},
	}
	h := newHandler(store, &fakeMealLogger{})

	rec := doRequest(h, "GET", "/api/v1/plans/p1", nil, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	got := decodeJSON[types.PlanBundle](t, rec)
	if got.Plan.ID != "p1" {
		t.Errorf("unexpected bundle: %+v", got)
	}
}

func TestHandleGetPlanWrongUser(t *testing.T) {
	store := newFakeMealStore()
	store.planBundles = map[string]types.PlanBundle{
		"p1": {Plan: types.DietPlan{ID: "p1", UserID: "other-user"}},
	}
	h := newHandler(store, &fakeMealLogger{})

	rec := doRequest(h, "GET", "/api/v1/plans/p1", nil, nil)
	if rec.Code != http.StatusNotFound {
		t.Errorf("cross-user plan access expected 404, got %d", rec.Code)
	}
}

// --- handleUpdatePlan / handleDeletePlan ---

func TestHandleUpdatePlan(t *testing.T) {
	store := newFakeMealStore()
	store.plans = map[string]types.DietPlan{"p1": {ID: "p1", UserID: "test-user", Name: "Old"}}
	store.planBundles = map[string]types.PlanBundle{
		"p1": {Plan: types.DietPlan{ID: "p1", UserID: "test-user"}, DayTypes: []types.DietPlanDayTypeBundle{
			{DietPlanDayType: types.DietPlanDayType{ID: "dt-1", PlanID: "p1"}},
		}},
	}
	h := newHandler(store, &fakeMealLogger{})

	body := map[string]any{
		"name": "New", "valid_from": "2026-01-05", "cycle_anchor_date": "2026-01-05",
		"cycle_pattern": []string{"dt-1"},
	}
	rec := doRequest(h, "PUT", "/api/v1/plans/p1", body, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	got := decodeJSON[types.DietPlan](t, rec)
	if got.Name != "New" || len(got.CyclePattern) != 1 {
		t.Errorf("unexpected plan: %+v", got)
	}
}

func TestHandleUpdatePlanUnknownDayTypeInPattern(t *testing.T) {
	store := newFakeMealStore()
	store.plans = map[string]types.DietPlan{"p1": {ID: "p1", UserID: "test-user"}}
	store.planBundles = map[string]types.PlanBundle{
		"p1": {Plan: types.DietPlan{ID: "p1", UserID: "test-user"}, DayTypes: []types.DietPlanDayTypeBundle{}},
	}
	h := newHandler(store, &fakeMealLogger{})

	body := map[string]any{
		"name": "New", "valid_from": "2026-01-05", "cycle_anchor_date": "2026-01-05",
		"cycle_pattern": []string{"nonexistent-day-type"},
	}
	rec := doRequest(h, "PUT", "/api/v1/plans/p1", body, nil)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("cycle_pattern referencing unknown day-type expected 400, got %d", rec.Code)
	}
}

func TestHandleUpdatePlanWrongUser(t *testing.T) {
	store := newFakeMealStore()
	store.plans = map[string]types.DietPlan{"p1": {ID: "p1", UserID: "other-user"}}
	h := newHandler(store, &fakeMealLogger{})

	body := map[string]any{"name": "New", "valid_from": "2026-01-05", "cycle_anchor_date": "2026-01-05"}
	rec := doRequest(h, "PUT", "/api/v1/plans/p1", body, nil)
	if rec.Code != http.StatusNotFound {
		t.Errorf("cross-user plan update expected 404, got %d", rec.Code)
	}
}

func TestHandleDeletePlan(t *testing.T) {
	store := newFakeMealStore()
	h := newHandler(store, &fakeMealLogger{})

	rec := doRequest(h, "DELETE", "/api/v1/plans/p1", nil, nil)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestHandleDeletePlanNotFound(t *testing.T) {
	store := newFakeMealStore()
	store.deletePlanErr = types.ErrNotFound
	h := newHandler(store, &fakeMealLogger{})

	rec := doRequest(h, "DELETE", "/api/v1/plans/missing", nil, nil)
	if rec.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", rec.Code)
	}
}

// --- handleGetActivePlan ---

func TestHandleGetActivePlanNone(t *testing.T) {
	store := newFakeMealStore()
	store.activePlanErr = types.ErrNotFound
	h := newHandler(store, &fakeMealLogger{})

	rec := doRequest(h, "GET", "/api/v1/plans/active", nil, nil)
	if rec.Code != http.StatusNotFound {
		t.Errorf("no active plan expected 404, got %d", rec.Code)
	}
}

// --- handleCreateDayType / handleUpdateDayType / handleDeleteDayType ---

func TestHandleCreateDayType(t *testing.T) {
	store := newFakeMealStore()
	store.plans = map[string]types.DietPlan{"p1": {ID: "p1", UserID: "test-user"}}
	h := newHandler(store, &fakeMealLogger{})

	body := map[string]any{"name": "Low-carb", "position": 0, "targets": map[string]any{"Calories": 1800}, "water_goal_ml": 2000}
	rec := doRequest(h, "POST", "/api/v1/plans/p1/day-types", body, nil)
	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rec.Code, rec.Body.String())
	}
	got := decodeJSON[types.DietPlanDayType](t, rec)
	if got.PlanID != "p1" || got.Name != "Low-carb" {
		t.Errorf("unexpected day type: %+v", got)
	}
}

func TestHandleCreateDayTypeMissingName(t *testing.T) {
	store := newFakeMealStore()
	store.plans = map[string]types.DietPlan{"p1": {ID: "p1", UserID: "test-user"}}
	h := newHandler(store, &fakeMealLogger{})

	rec := doRequest(h, "POST", "/api/v1/plans/p1/day-types", map[string]any{"position": 0}, nil)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("missing name expected 400, got %d", rec.Code)
	}
}

func TestHandleCreateDayTypeWrongUser(t *testing.T) {
	store := newFakeMealStore()
	store.plans = map[string]types.DietPlan{"p1": {ID: "p1", UserID: "other-user"}}
	h := newHandler(store, &fakeMealLogger{})

	body := map[string]any{"name": "Low-carb", "position": 0}
	rec := doRequest(h, "POST", "/api/v1/plans/p1/day-types", body, nil)
	if rec.Code != http.StatusNotFound {
		t.Errorf("cross-user day-type create expected 404, got %d", rec.Code)
	}
}

func TestHandleUpdateDayTypeCallsRefreshTodayTargets(t *testing.T) {
	store := newFakeMealStore()
	store.plans = map[string]types.DietPlan{"p1": {ID: "p1", UserID: "test-user"}}
	store.dayTypes = map[string]types.DietPlanDayType{"dt-1": {ID: "dt-1", PlanID: "p1"}}
	store.refreshTodayTargetsErr = errors.New("refresh failed")
	h := newHandler(store, &fakeMealLogger{})

	body := map[string]any{"name": "Low-carb", "position": 0}
	rec := doRequest(h, "PUT", "/api/v1/plans/p1/day-types/dt-1", body, nil)
	if rec.Code != http.StatusInternalServerError {
		t.Errorf("RefreshTodayTargets failure expected 500, got %d", rec.Code)
	}
}

func TestHandleUpdateDayTypeWrongPlan(t *testing.T) {
	store := newFakeMealStore()
	store.plans = map[string]types.DietPlan{"p1": {ID: "p1", UserID: "test-user"}}
	store.dayTypes = map[string]types.DietPlanDayType{"dt-1": {ID: "dt-1", PlanID: "some-other-plan"}}
	h := newHandler(store, &fakeMealLogger{})

	body := map[string]any{"name": "Low-carb", "position": 0}
	rec := doRequest(h, "PUT", "/api/v1/plans/p1/day-types/dt-1", body, nil)
	if rec.Code != http.StatusNotFound {
		t.Errorf("day-type belonging to a different plan expected 404, got %d", rec.Code)
	}
}

func TestHandleDeleteDayType(t *testing.T) {
	store := newFakeMealStore()
	store.plans = map[string]types.DietPlan{"p1": {ID: "p1", UserID: "test-user"}}
	store.dayTypes = map[string]types.DietPlanDayType{"dt-1": {ID: "dt-1", PlanID: "p1"}}
	h := newHandler(store, &fakeMealLogger{})

	rec := doRequest(h, "DELETE", "/api/v1/plans/p1/day-types/dt-1", nil, nil)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d: %s", rec.Code, rec.Body.String())
	}
}

// --- handleCreateSlot / handleUpdateSlot / handleDeleteSlot ---

func TestHandleCreateSlot(t *testing.T) {
	store := newFakeMealStore()
	store.plans = map[string]types.DietPlan{"p1": {ID: "p1", UserID: "test-user"}}
	store.dayTypes = map[string]types.DietPlanDayType{"dt-1": {ID: "dt-1", PlanID: "p1"}}
	h := newHandler(store, &fakeMealLogger{})

	body := map[string]any{"label": "Café da manhã", "position": 0}
	rec := doRequest(h, "POST", "/api/v1/plans/p1/day-types/dt-1/slots", body, nil)
	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rec.Code, rec.Body.String())
	}
	got := decodeJSON[types.DietPlanSlot](t, rec)
	if got.DayTypeID != "dt-1" || got.Label != "Café da manhã" {
		t.Errorf("unexpected slot: %+v", got)
	}
}

func TestHandleCreateSlotMissingLabel(t *testing.T) {
	store := newFakeMealStore()
	store.plans = map[string]types.DietPlan{"p1": {ID: "p1", UserID: "test-user"}}
	store.dayTypes = map[string]types.DietPlanDayType{"dt-1": {ID: "dt-1", PlanID: "p1"}}
	h := newHandler(store, &fakeMealLogger{})

	rec := doRequest(h, "POST", "/api/v1/plans/p1/day-types/dt-1/slots", map[string]any{"position": 0}, nil)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("missing label expected 400, got %d", rec.Code)
	}
}

func TestHandleUpdateSlotNotFoundWrongDayType(t *testing.T) {
	store := newFakeMealStore()
	store.plans = map[string]types.DietPlan{"p1": {ID: "p1", UserID: "test-user"}}
	store.planBundles = map[string]types.PlanBundle{
		"p1": {Plan: types.DietPlan{ID: "p1", UserID: "test-user"}, DayTypes: []types.DietPlanDayTypeBundle{
			{DietPlanDayType: types.DietPlanDayType{ID: "dt-1", PlanID: "p1"}, Slots: []types.DietPlanSlotBundle{
				{DietPlanSlot: types.DietPlanSlot{ID: "sl-1", DayTypeID: "dt-1"}},
			}},
		}},
	}
	h := newHandler(store, &fakeMealLogger{})

	// sl-1 belongs to dt-1, not dt-2: the URL claims the wrong parent.
	body := map[string]any{"label": "x", "position": 0}
	rec := doRequest(h, "PUT", "/api/v1/plans/p1/day-types/dt-2/slots/sl-1", body, nil)
	if rec.Code != http.StatusNotFound {
		t.Errorf("slot under mismatched day-type expected 404, got %d", rec.Code)
	}
}

func TestHandleDeleteSlot(t *testing.T) {
	store := newFakeMealStore()
	store.planBundles = map[string]types.PlanBundle{
		"p1": {Plan: types.DietPlan{ID: "p1", UserID: "test-user"}, DayTypes: []types.DietPlanDayTypeBundle{
			{DietPlanDayType: types.DietPlanDayType{ID: "dt-1", PlanID: "p1"}, Slots: []types.DietPlanSlotBundle{
				{DietPlanSlot: types.DietPlanSlot{ID: "sl-1", DayTypeID: "dt-1"}},
			}},
		}},
	}
	h := newHandler(store, &fakeMealLogger{})

	rec := doRequest(h, "DELETE", "/api/v1/plans/p1/day-types/dt-1/slots/sl-1", nil, nil)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestHandleDeleteSlotWrongUser(t *testing.T) {
	store := newFakeMealStore()
	store.planBundles = map[string]types.PlanBundle{
		"p1": {Plan: types.DietPlan{ID: "p1", UserID: "other-user"}, DayTypes: []types.DietPlanDayTypeBundle{
			{DietPlanDayType: types.DietPlanDayType{ID: "dt-1", PlanID: "p1"}, Slots: []types.DietPlanSlotBundle{
				{DietPlanSlot: types.DietPlanSlot{ID: "sl-1", DayTypeID: "dt-1"}},
			}},
		}},
	}
	h := newHandler(store, &fakeMealLogger{})

	rec := doRequest(h, "DELETE", "/api/v1/plans/p1/day-types/dt-1/slots/sl-1", nil, nil)
	if rec.Code != http.StatusNotFound {
		t.Errorf("cross-user slot delete expected 404, got %d", rec.Code)
	}
}

// --- handleCreateSlotOption / handleUpdateSlotOption / handleDeleteSlotOption ---

func planBundleWithSlot() *fakeMealStore {
	store := newFakeMealStore()
	store.planBundles = map[string]types.PlanBundle{
		"p1": {Plan: types.DietPlan{ID: "p1", UserID: "test-user"}, DayTypes: []types.DietPlanDayTypeBundle{
			{DietPlanDayType: types.DietPlanDayType{ID: "dt-1", PlanID: "p1"}, Slots: []types.DietPlanSlotBundle{
				{DietPlanSlot: types.DietPlanSlot{ID: "sl-1", DayTypeID: "dt-1"}, Options: []types.DietPlanSlotOption{
					{ID: "opt-1", SlotID: "sl-1", Label: "Opção 1", TemplateID: "tmpl-1"},
				}},
			}},
		}},
	}
	return store
}

func TestHandleCreateSlotOptionCreatesPlanOwnedTemplate(t *testing.T) {
	store := planBundleWithSlot()
	h := newHandler(store, &fakeMealLogger{})

	body := map[string]any{
		"label": "Opção 2",
		"items": []types.ResolvedItem{{Parsed: types.ParsedItem{RawPhrase: "arroz", NormalizedGrams: 150}}},
	}
	rec := doRequest(h, "POST", "/api/v1/plans/p1/day-types/dt-1/slots/sl-1/options", body, nil)
	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rec.Code, rec.Body.String())
	}
	got := decodeJSON[types.DietPlanSlotOption](t, rec)
	if got.SlotID != "sl-1" || got.Label != "Opção 2" || got.TemplateID == "" {
		t.Fatalf("unexpected option: %+v", got)
	}
	if len(store.savedTemplates) != 1 {
		t.Fatalf("expected 1 saved template, got %d", len(store.savedTemplates))
	}
	saved := store.savedTemplates[0]
	if saved.OwnerKind != types.TemplateOwnerPlan {
		t.Errorf("backing template owner_kind = %q, want %q", saved.OwnerKind, types.TemplateOwnerPlan)
	}
	if saved.ID != got.TemplateID {
		t.Errorf("saved template ID = %q, option template_id = %q, want match", saved.ID, got.TemplateID)
	}
}

func TestHandleCreateSlotOptionMissingItems(t *testing.T) {
	store := planBundleWithSlot()
	h := newHandler(store, &fakeMealLogger{})

	rec := doRequest(h, "POST", "/api/v1/plans/p1/day-types/dt-1/slots/sl-1/options", map[string]any{"label": "Opção 2"}, nil)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("missing items expected 400, got %d", rec.Code)
	}
}

func TestHandleUpdateSlotOptionReusesExistingTemplateID(t *testing.T) {
	store := planBundleWithSlot()
	h := newHandler(store, &fakeMealLogger{})

	body := map[string]any{
		"label": "Opção 1 revisada",
		"items": []types.ResolvedItem{{Parsed: types.ParsedItem{RawPhrase: "batata doce", NormalizedGrams: 200}}},
	}
	rec := doRequest(h, "PUT", "/api/v1/plans/p1/day-types/dt-1/slots/sl-1/options/opt-1", body, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	if len(store.savedTemplates) != 1 || store.savedTemplates[0].ID != "tmpl-1" {
		t.Fatalf("expected update to re-save the existing backing template tmpl-1, got %+v", store.savedTemplates)
	}
}

func TestHandleDeleteSlotOptionNotFound(t *testing.T) {
	store := planBundleWithSlot()
	h := newHandler(store, &fakeMealLogger{})

	rec := doRequest(h, "DELETE", "/api/v1/plans/p1/day-types/dt-1/slots/sl-1/options/missing", nil, nil)
	if rec.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", rec.Code)
	}
}

func TestHandleDeleteSlotOption(t *testing.T) {
	store := planBundleWithSlot()
	h := newHandler(store, &fakeMealLogger{})

	rec := doRequest(h, "DELETE", "/api/v1/plans/p1/day-types/dt-1/slots/sl-1/options/opt-1", nil, nil)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d: %s", rec.Code, rec.Body.String())
	}
}

// --- handleGetPlanDay ---

func TestHandleGetPlanDayNoOverrideNoPlan(t *testing.T) {
	store := newFakeMealStore()
	store.activePlanErr = types.ErrNotFound
	store.targetsFor = types.DailyTargets{UserID: "test-user", Targets: types.Macros{Calories: 2000}}
	h := newHandler(store, &fakeMealLogger{})

	rec := doRequest(h, "GET", "/api/v1/plans/day/2026-07-27", nil, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	got := decodeJSON[planDayView](t, rec)
	if got.PlanActive || got.Overridden || got.DayType != nil {
		t.Errorf("expected no plan/override, got %+v", got)
	}
	if got.Targets.Targets.Calories != 2000 {
		t.Errorf("targets = %+v, want fallback 2000 kcal", got.Targets)
	}
}

func TestHandleGetPlanDayInvalidDate(t *testing.T) {
	store := newFakeMealStore()
	h := newHandler(store, &fakeMealLogger{})

	rec := doRequest(h, "GET", "/api/v1/plans/day/not-a-date", nil, nil)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rec.Code)
	}
}

func TestHandleGetPlanDayOverride(t *testing.T) {
	store := newFakeMealStore()
	store.dayOverrides = map[string]types.DietPlanDayOverride{
		"test-user|2026-07-27": {UserID: "test-user", Date: "2026-07-27", DayTypeID: "dt-1"},
	}
	store.dayTypes = map[string]types.DietPlanDayType{"dt-1": {ID: "dt-1", PlanID: "p1", Name: "Low-carb"}}
	store.planBundles = map[string]types.PlanBundle{
		"p1": {Plan: types.DietPlan{ID: "p1", UserID: "test-user"}, DayTypes: []types.DietPlanDayTypeBundle{
			{DietPlanDayType: types.DietPlanDayType{ID: "dt-1", PlanID: "p1", Name: "Low-carb"}},
		}},
	}
	h := newHandler(store, &fakeMealLogger{})

	rec := doRequest(h, "GET", "/api/v1/plans/day/2026-07-27", nil, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	got := decodeJSON[planDayView](t, rec)
	if !got.Overridden || !got.PlanActive || got.DayType == nil || got.DayType.ID != "dt-1" {
		t.Errorf("expected resolved override day-type, got %+v", got)
	}
}

func TestHandleGetPlanDayActivePlanCycle(t *testing.T) {
	store := newFakeMealStore()
	store.activePlan = types.DietPlan{
		ID: "p1", UserID: "test-user",
		CyclePattern: []string{"dt-low", "dt-high"}, CycleAnchorDate: "2026-07-27",
	}
	store.planBundles = map[string]types.PlanBundle{
		"p1": {Plan: types.DietPlan{ID: "p1", UserID: "test-user"}, DayTypes: []types.DietPlanDayTypeBundle{
			{DietPlanDayType: types.DietPlanDayType{ID: "dt-low", PlanID: "p1", Name: "Low-carb"}},
			{DietPlanDayType: types.DietPlanDayType{ID: "dt-high", PlanID: "p1", Name: "High-carb"}},
		}},
	}
	h := newHandler(store, &fakeMealLogger{})

	// Anchor date itself: index 0 → dt-low.
	rec := doRequest(h, "GET", "/api/v1/plans/day/2026-07-27", nil, nil)
	got := decodeJSON[planDayView](t, rec)
	if got.DayType == nil || got.DayType.ID != "dt-low" {
		t.Fatalf("anchor date expected dt-low, got %+v", got.DayType)
	}

	// One day before anchor: Euclidean mod must resolve to index 1 (dt-high),
	// not panic or return a negative index.
	rec = doRequest(h, "GET", "/api/v1/plans/day/2026-07-26", nil, nil)
	got = decodeJSON[planDayView](t, rec)
	if got.DayType == nil || got.DayType.ID != "dt-high" {
		t.Fatalf("day before anchor expected dt-high, got %+v", got.DayType)
	}
}

// --- handleSetDayOverride / handleDeleteDayOverride ---

func TestHandleSetDayOverride(t *testing.T) {
	store := newFakeMealStore()
	store.dayTypes = map[string]types.DietPlanDayType{"dt-1": {ID: "dt-1", PlanID: "p1"}}
	store.plans = map[string]types.DietPlan{"p1": {ID: "p1", UserID: "test-user"}}
	h := newHandler(store, &fakeMealLogger{})

	rec := doRequest(h, "PUT", "/api/v1/plans/overrides/2026-07-27", map[string]any{"day_type_id": "dt-1"}, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestHandleSetDayOverrideWrongUser(t *testing.T) {
	store := newFakeMealStore()
	store.dayTypes = map[string]types.DietPlanDayType{"dt-1": {ID: "dt-1", PlanID: "p1"}}
	store.plans = map[string]types.DietPlan{"p1": {ID: "p1", UserID: "other-user"}}
	h := newHandler(store, &fakeMealLogger{})

	rec := doRequest(h, "PUT", "/api/v1/plans/overrides/2026-07-27", map[string]any{"day_type_id": "dt-1"}, nil)
	if rec.Code != http.StatusNotFound {
		t.Errorf("override referencing another user's day-type expected 404, got %d", rec.Code)
	}
}

func TestHandleSetDayOverrideInvalidDate(t *testing.T) {
	store := newFakeMealStore()
	h := newHandler(store, &fakeMealLogger{})

	rec := doRequest(h, "PUT", "/api/v1/plans/overrides/not-a-date", map[string]any{"day_type_id": "dt-1"}, nil)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rec.Code)
	}
}

func TestHandleDeleteDayOverride(t *testing.T) {
	store := newFakeMealStore()
	h := newHandler(store, &fakeMealLogger{})

	rec := doRequest(h, "DELETE", "/api/v1/plans/overrides/2026-07-27", nil, nil)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestHandleDeleteDayOverrideCallsRefreshTodayTargets(t *testing.T) {
	store := newFakeMealStore()
	store.refreshTodayTargetsErr = errors.New("refresh failed")
	h := newHandler(store, &fakeMealLogger{})

	rec := doRequest(h, "DELETE", "/api/v1/plans/overrides/2026-07-27", nil, nil)
	if rec.Code != http.StatusInternalServerError {
		t.Errorf("RefreshTodayTargets failure expected 500, got %d", rec.Code)
	}
}

func TestHandlePlansUnauthorized(t *testing.T) {
	store := newFakeMealStore()
	h := newHandler(store, &fakeMealLogger{})

	for _, tc := range []struct{ method, path string }{
		{"GET", "/api/v1/plans/p1"},
		{"PUT", "/api/v1/plans/p1"},
		{"DELETE", "/api/v1/plans/p1"},
		{"GET", "/api/v1/plans/day/2026-07-27"},
		{"PUT", "/api/v1/plans/overrides/2026-07-27"},
	} {
		rec := doRequest(h, tc.method, tc.path, nil, map[string]string{"Authorization": ""})
		if rec.Code != http.StatusUnauthorized {
			t.Errorf("%s %s: expected 401, got %d", tc.method, tc.path, rec.Code)
		}
	}
}
