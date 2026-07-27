package api

import (
	"errors"
	"net/http"
	"strings"
	"testing"
	"time"

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

func TestHandleUpdatePlanInvalidJSON(t *testing.T) {
	store := newFakeMealStore()
	store.plans = map[string]types.DietPlan{"p1": {ID: "p1", UserID: "test-user"}}
	h := newHandler(store, &fakeMealLogger{})

	rec := doRequestRawBody(h, "PUT", "/api/v1/plans/p1", "{not json")
	if rec.Code != http.StatusBadRequest {
		t.Errorf("invalid JSON expected 400, got %d", rec.Code)
	}
}

func TestHandleUpdatePlanMissingName(t *testing.T) {
	store := newFakeMealStore()
	store.plans = map[string]types.DietPlan{"p1": {ID: "p1", UserID: "test-user"}}
	h := newHandler(store, &fakeMealLogger{})

	body := map[string]any{"valid_from": "2026-01-05", "cycle_anchor_date": "2026-01-05"}
	rec := doRequest(h, "PUT", "/api/v1/plans/p1", body, nil)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("missing name expected 400, got %d", rec.Code)
	}
}

func TestHandleUpdatePlanBadDates(t *testing.T) {
	store := newFakeMealStore()
	store.plans = map[string]types.DietPlan{"p1": {ID: "p1", UserID: "test-user"}}
	h := newHandler(store, &fakeMealLogger{})

	body := map[string]any{"name": "x", "valid_from": "not-a-date", "cycle_anchor_date": "2026-01-05"}
	rec := doRequest(h, "PUT", "/api/v1/plans/p1", body, nil)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("bad valid_from expected 400, got %d", rec.Code)
	}
}

func TestHandleUpdatePlanNotFound(t *testing.T) {
	store := newFakeMealStore()
	h := newHandler(store, &fakeMealLogger{})

	body := map[string]any{"name": "x", "valid_from": "2026-01-05", "cycle_anchor_date": "2026-01-05"}
	rec := doRequest(h, "PUT", "/api/v1/plans/missing", body, nil)
	if rec.Code != http.StatusNotFound {
		t.Errorf("update of missing plan expected 404, got %d", rec.Code)
	}
}

func TestHandleUpdatePlanCyclePatternBundleErr(t *testing.T) {
	store := newFakeMealStore()
	store.plans = map[string]types.DietPlan{"p1": {ID: "p1", UserID: "test-user"}}
	store.getPlanBundleErr = errors.New("db down")
	h := newHandler(store, &fakeMealLogger{})

	body := map[string]any{
		"name": "New", "valid_from": "2026-01-05", "cycle_anchor_date": "2026-01-05",
		"cycle_pattern": []string{"dt-1"},
	}
	rec := doRequest(h, "PUT", "/api/v1/plans/p1", body, nil)
	if rec.Code != http.StatusInternalServerError {
		t.Errorf("GetPlanBundle failure expected 500, got %d", rec.Code)
	}
}

func TestHandleUpdatePlanStoreErr(t *testing.T) {
	store := newFakeMealStore()
	store.plans = map[string]types.DietPlan{"p1": {ID: "p1", UserID: "test-user"}}
	store.updatePlanErr = errors.New("db down")
	h := newHandler(store, &fakeMealLogger{})

	body := map[string]any{"name": "New", "valid_from": "2026-01-05", "cycle_anchor_date": "2026-01-05"}
	rec := doRequest(h, "PUT", "/api/v1/plans/p1", body, nil)
	if rec.Code != http.StatusInternalServerError {
		t.Errorf("UpdatePlan failure expected 500, got %d", rec.Code)
	}
}

func TestHandleUpdatePlanCallsRefreshTodayTargets(t *testing.T) {
	store := newFakeMealStore()
	store.plans = map[string]types.DietPlan{"p1": {ID: "p1", UserID: "test-user"}}
	store.refreshTodayTargetsErr = errors.New("refresh failed")
	h := newHandler(store, &fakeMealLogger{})

	body := map[string]any{"name": "New", "valid_from": "2026-01-05", "cycle_anchor_date": "2026-01-05"}
	rec := doRequest(h, "PUT", "/api/v1/plans/p1", body, nil)
	if rec.Code != http.StatusInternalServerError {
		t.Errorf("RefreshTodayTargets failure expected 500, got %d", rec.Code)
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

func TestHandleUpdateDayType(t *testing.T) {
	store := newFakeMealStore()
	store.plans = map[string]types.DietPlan{"p1": {ID: "p1", UserID: "test-user"}}
	store.dayTypes = map[string]types.DietPlanDayType{"dt-1": {ID: "dt-1", PlanID: "p1", Name: "Old"}}
	h := newHandler(store, &fakeMealLogger{})

	body := map[string]any{"name": "New", "position": 1, "water_goal_ml": 2500}
	rec := doRequest(h, "PUT", "/api/v1/plans/p1/day-types/dt-1", body, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	got := decodeJSON[types.DietPlanDayType](t, rec)
	if got.ID != "dt-1" || got.PlanID != "p1" || got.Name != "New" || got.WaterGoalMl != 2500 {
		t.Errorf("unexpected day type: %+v", got)
	}
}

func TestHandleUpdateDayTypeInvalidJSON(t *testing.T) {
	store := newFakeMealStore()
	store.plans = map[string]types.DietPlan{"p1": {ID: "p1", UserID: "test-user"}}
	store.dayTypes = map[string]types.DietPlanDayType{"dt-1": {ID: "dt-1", PlanID: "p1"}}
	h := newHandler(store, &fakeMealLogger{})

	rec := doRequestRawBody(h, "PUT", "/api/v1/plans/p1/day-types/dt-1", "{not json")
	if rec.Code != http.StatusBadRequest {
		t.Errorf("invalid JSON expected 400, got %d", rec.Code)
	}
}

func TestHandleUpdateDayTypeInvalidBody(t *testing.T) {
	store := newFakeMealStore()
	store.plans = map[string]types.DietPlan{"p1": {ID: "p1", UserID: "test-user"}}
	store.dayTypes = map[string]types.DietPlanDayType{"dt-1": {ID: "dt-1", PlanID: "p1"}}
	h := newHandler(store, &fakeMealLogger{})

	rec := doRequest(h, "PUT", "/api/v1/plans/p1/day-types/dt-1", map[string]any{"position": -1}, nil)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("missing name / negative position expected 400, got %d", rec.Code)
	}
}

func TestHandleUpdateDayTypeStoreErr(t *testing.T) {
	store := newFakeMealStore()
	store.plans = map[string]types.DietPlan{"p1": {ID: "p1", UserID: "test-user"}}
	store.dayTypes = map[string]types.DietPlanDayType{"dt-1": {ID: "dt-1", PlanID: "p1"}}
	store.updateDayTypeErr = errors.New("db down")
	h := newHandler(store, &fakeMealLogger{})

	body := map[string]any{"name": "New", "position": 0}
	rec := doRequest(h, "PUT", "/api/v1/plans/p1/day-types/dt-1", body, nil)
	if rec.Code != http.StatusInternalServerError {
		t.Errorf("UpdateDayType failure expected 500, got %d", rec.Code)
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

func TestHandleDeleteDayTypeWrongPlan(t *testing.T) {
	store := newFakeMealStore()
	store.plans = map[string]types.DietPlan{"p1": {ID: "p1", UserID: "test-user"}}
	store.dayTypes = map[string]types.DietPlanDayType{"dt-1": {ID: "dt-1", PlanID: "some-other-plan"}}
	h := newHandler(store, &fakeMealLogger{})

	rec := doRequest(h, "DELETE", "/api/v1/plans/p1/day-types/dt-1", nil, nil)
	if rec.Code != http.StatusNotFound {
		t.Errorf("day-type belonging to a different plan expected 404, got %d", rec.Code)
	}
}

func TestHandleDeleteDayTypeStoreErr(t *testing.T) {
	store := newFakeMealStore()
	store.plans = map[string]types.DietPlan{"p1": {ID: "p1", UserID: "test-user"}}
	store.dayTypes = map[string]types.DietPlanDayType{"dt-1": {ID: "dt-1", PlanID: "p1"}}
	store.deleteDayTypeErr = errors.New("db down")
	h := newHandler(store, &fakeMealLogger{})

	rec := doRequest(h, "DELETE", "/api/v1/plans/p1/day-types/dt-1", nil, nil)
	if rec.Code != http.StatusInternalServerError {
		t.Errorf("DeleteDayType failure expected 500, got %d", rec.Code)
	}
}

func TestHandleDeleteDayTypeCallsRefreshTodayTargets(t *testing.T) {
	store := newFakeMealStore()
	store.plans = map[string]types.DietPlan{"p1": {ID: "p1", UserID: "test-user"}}
	store.dayTypes = map[string]types.DietPlanDayType{"dt-1": {ID: "dt-1", PlanID: "p1"}}
	store.refreshTodayTargetsErr = errors.New("refresh failed")
	h := newHandler(store, &fakeMealLogger{})

	rec := doRequest(h, "DELETE", "/api/v1/plans/p1/day-types/dt-1", nil, nil)
	if rec.Code != http.StatusInternalServerError {
		t.Errorf("RefreshTodayTargets failure expected 500, got %d", rec.Code)
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

func TestHandleCreateSlotWrongPlan(t *testing.T) {
	store := newFakeMealStore()
	store.plans = map[string]types.DietPlan{"p1": {ID: "p1", UserID: "test-user"}}
	store.dayTypes = map[string]types.DietPlanDayType{"dt-1": {ID: "dt-1", PlanID: "some-other-plan"}}
	h := newHandler(store, &fakeMealLogger{})

	body := map[string]any{"label": "Café da manhã", "position": 0}
	rec := doRequest(h, "POST", "/api/v1/plans/p1/day-types/dt-1/slots", body, nil)
	if rec.Code != http.StatusNotFound {
		t.Errorf("slot create under mismatched day-type expected 404, got %d", rec.Code)
	}
}

func TestHandleCreateSlotInvalidJSON(t *testing.T) {
	store := newFakeMealStore()
	store.plans = map[string]types.DietPlan{"p1": {ID: "p1", UserID: "test-user"}}
	store.dayTypes = map[string]types.DietPlanDayType{"dt-1": {ID: "dt-1", PlanID: "p1"}}
	h := newHandler(store, &fakeMealLogger{})

	rec := doRequestRawBody(h, "POST", "/api/v1/plans/p1/day-types/dt-1/slots", "{not json")
	if rec.Code != http.StatusBadRequest {
		t.Errorf("invalid JSON expected 400, got %d", rec.Code)
	}
}

func TestHandleCreateSlotStoreErr(t *testing.T) {
	store := newFakeMealStore()
	store.plans = map[string]types.DietPlan{"p1": {ID: "p1", UserID: "test-user"}}
	store.dayTypes = map[string]types.DietPlanDayType{"dt-1": {ID: "dt-1", PlanID: "p1"}}
	store.createSlotErr = errors.New("db down")
	h := newHandler(store, &fakeMealLogger{})

	body := map[string]any{"label": "Café da manhã", "position": 0}
	rec := doRequest(h, "POST", "/api/v1/plans/p1/day-types/dt-1/slots", body, nil)
	if rec.Code != http.StatusInternalServerError {
		t.Errorf("CreateSlot failure expected 500, got %d", rec.Code)
	}
}

func TestHandleUpdateSlot(t *testing.T) {
	store := newFakeMealStore()
	store.planBundles = map[string]types.PlanBundle{
		"p1": {Plan: types.DietPlan{ID: "p1", UserID: "test-user"}, DayTypes: []types.DietPlanDayTypeBundle{
			{DietPlanDayType: types.DietPlanDayType{ID: "dt-1", PlanID: "p1"}, Slots: []types.DietPlanSlotBundle{
				{DietPlanSlot: types.DietPlanSlot{ID: "sl-1", DayTypeID: "dt-1", Label: "Old"}},
			}},
		}},
	}
	h := newHandler(store, &fakeMealLogger{})

	body := map[string]any{"label": "New label", "position": 2}
	rec := doRequest(h, "PUT", "/api/v1/plans/p1/day-types/dt-1/slots/sl-1", body, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	got := decodeJSON[types.DietPlanSlot](t, rec)
	if got.ID != "sl-1" || got.Label != "New label" || got.Position != 2 {
		t.Errorf("unexpected slot: %+v", got)
	}
}

func TestHandleUpdateSlotInvalidJSON(t *testing.T) {
	store := planBundleWithSlot()
	h := newHandler(store, &fakeMealLogger{})

	rec := doRequestRawBody(h, "PUT", "/api/v1/plans/p1/day-types/dt-1/slots/sl-1", "{not json")
	if rec.Code != http.StatusBadRequest {
		t.Errorf("invalid JSON expected 400, got %d", rec.Code)
	}
}

func TestHandleUpdateSlotMissingLabel(t *testing.T) {
	store := planBundleWithSlot()
	h := newHandler(store, &fakeMealLogger{})

	rec := doRequest(h, "PUT", "/api/v1/plans/p1/day-types/dt-1/slots/sl-1", map[string]any{"position": 0}, nil)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("missing label expected 400, got %d", rec.Code)
	}
}

func TestHandleUpdateSlotStoreErr(t *testing.T) {
	store := planBundleWithSlot()
	store.updateSlotErr = errors.New("db down")
	h := newHandler(store, &fakeMealLogger{})

	body := map[string]any{"label": "New label", "position": 0}
	rec := doRequest(h, "PUT", "/api/v1/plans/p1/day-types/dt-1/slots/sl-1", body, nil)
	if rec.Code != http.StatusInternalServerError {
		t.Errorf("UpdateSlot failure expected 500, got %d", rec.Code)
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

func TestHandleCreateSlotOptionWrongSlot(t *testing.T) {
	store := planBundleWithSlot()
	h := newHandler(store, &fakeMealLogger{})

	body := map[string]any{
		"label": "Opção 2",
		"items": []types.ResolvedItem{{Parsed: types.ParsedItem{RawPhrase: "arroz", NormalizedGrams: 150}}},
	}
	rec := doRequest(h, "POST", "/api/v1/plans/p1/day-types/dt-1/slots/missing-slot/options", body, nil)
	if rec.Code != http.StatusNotFound {
		t.Errorf("slot option create under missing slot expected 404, got %d", rec.Code)
	}
}

func TestHandleCreateSlotOptionInvalidJSON(t *testing.T) {
	store := planBundleWithSlot()
	h := newHandler(store, &fakeMealLogger{})

	rec := doRequestRawBody(h, "POST", "/api/v1/plans/p1/day-types/dt-1/slots/sl-1/options", "{not json")
	if rec.Code != http.StatusBadRequest {
		t.Errorf("invalid JSON expected 400, got %d", rec.Code)
	}
}

func TestHandleCreateSlotOptionSaveTemplateErr(t *testing.T) {
	store := planBundleWithSlot()
	store.saveTemplateErr = errors.New("db down")
	h := newHandler(store, &fakeMealLogger{})

	body := map[string]any{
		"label": "Opção 2",
		"items": []types.ResolvedItem{{Parsed: types.ParsedItem{RawPhrase: "arroz", NormalizedGrams: 150}}},
	}
	rec := doRequest(h, "POST", "/api/v1/plans/p1/day-types/dt-1/slots/sl-1/options", body, nil)
	if rec.Code != http.StatusInternalServerError {
		t.Errorf("SaveTemplate failure expected 500, got %d", rec.Code)
	}
}

func TestHandleCreateSlotOptionStoreErr(t *testing.T) {
	store := planBundleWithSlot()
	store.createSlotOptionErr = errors.New("db down")
	h := newHandler(store, &fakeMealLogger{})

	body := map[string]any{
		"label": "Opção 2",
		"items": []types.ResolvedItem{{Parsed: types.ParsedItem{RawPhrase: "arroz", NormalizedGrams: 150}}},
	}
	rec := doRequest(h, "POST", "/api/v1/plans/p1/day-types/dt-1/slots/sl-1/options", body, nil)
	if rec.Code != http.StatusInternalServerError {
		t.Errorf("CreateSlotOption failure expected 500, got %d", rec.Code)
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

func TestHandleUpdateSlotOptionInvalidJSON(t *testing.T) {
	store := planBundleWithSlot()
	h := newHandler(store, &fakeMealLogger{})

	rec := doRequestRawBody(h, "PUT", "/api/v1/plans/p1/day-types/dt-1/slots/sl-1/options/opt-1", "{not json")
	if rec.Code != http.StatusBadRequest {
		t.Errorf("invalid JSON expected 400, got %d", rec.Code)
	}
}

func TestHandleUpdateSlotOptionMissingItems(t *testing.T) {
	store := planBundleWithSlot()
	h := newHandler(store, &fakeMealLogger{})

	rec := doRequest(h, "PUT", "/api/v1/plans/p1/day-types/dt-1/slots/sl-1/options/opt-1", map[string]any{"label": "x"}, nil)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("missing items expected 400, got %d", rec.Code)
	}
}

func TestHandleUpdateSlotOptionNotFound(t *testing.T) {
	store := planBundleWithSlot()
	h := newHandler(store, &fakeMealLogger{})

	body := map[string]any{
		"label": "x",
		"items": []types.ResolvedItem{{Parsed: types.ParsedItem{RawPhrase: "arroz", NormalizedGrams: 150}}},
	}
	rec := doRequest(h, "PUT", "/api/v1/plans/p1/day-types/dt-1/slots/sl-1/options/missing", body, nil)
	if rec.Code != http.StatusNotFound {
		t.Errorf("update of missing option expected 404, got %d", rec.Code)
	}
}

func TestHandleUpdateSlotOptionSaveTemplateErr(t *testing.T) {
	store := planBundleWithSlot()
	store.saveTemplateErr = errors.New("db down")
	h := newHandler(store, &fakeMealLogger{})

	body := map[string]any{
		"label": "x",
		"items": []types.ResolvedItem{{Parsed: types.ParsedItem{RawPhrase: "arroz", NormalizedGrams: 150}}},
	}
	rec := doRequest(h, "PUT", "/api/v1/plans/p1/day-types/dt-1/slots/sl-1/options/opt-1", body, nil)
	if rec.Code != http.StatusInternalServerError {
		t.Errorf("SaveTemplate failure expected 500, got %d", rec.Code)
	}
}

func TestHandleUpdateSlotOptionStoreErr(t *testing.T) {
	store := planBundleWithSlot()
	store.updateSlotOptionErr = errors.New("db down")
	h := newHandler(store, &fakeMealLogger{})

	body := map[string]any{
		"label": "x",
		"items": []types.ResolvedItem{{Parsed: types.ParsedItem{RawPhrase: "arroz", NormalizedGrams: 150}}},
	}
	rec := doRequest(h, "PUT", "/api/v1/plans/p1/day-types/dt-1/slots/sl-1/options/opt-1", body, nil)
	if rec.Code != http.StatusInternalServerError {
		t.Errorf("UpdateSlotOption failure expected 500, got %d", rec.Code)
	}
}

// TestOwnedOptionMismatchedSlotOrDayType confirms ownedOption -- walked from
// the plan bundle rather than a point query -- rejects an option ID that
// exists but under a different slot or day-type than the URL claims, not
// just an option ID that doesn't exist anywhere.
func TestOwnedOptionMismatchedSlotOrDayType(t *testing.T) {
	store := planBundleWithSlot()
	h := newHandler(store, &fakeMealLogger{})

	for _, tc := range []struct {
		name, path string
	}{
		{"wrong day-type", "/api/v1/plans/p1/day-types/wrong-dt/slots/sl-1/options/opt-1"},
		{"wrong slot", "/api/v1/plans/p1/day-types/dt-1/slots/wrong-slot/options/opt-1"},
	} {
		rec := doRequest(h, "DELETE", tc.path, nil, nil)
		if rec.Code != http.StatusNotFound {
			t.Errorf("%s: expected 404, got %d", tc.name, rec.Code)
		}
	}
}

func TestOwnedOptionPlanBundleErr(t *testing.T) {
	store := planBundleWithSlot()
	store.getPlanBundleErr = errors.New("db down")
	h := newHandler(store, &fakeMealLogger{})

	rec := doRequest(h, "DELETE", "/api/v1/plans/p1/day-types/dt-1/slots/sl-1/options/opt-1", nil, nil)
	if rec.Code != http.StatusInternalServerError {
		t.Errorf("GetPlanBundle failure expected 500, got %d", rec.Code)
	}
}

func TestOwnedOptionWrongUser(t *testing.T) {
	store := newFakeMealStore()
	store.planBundles = map[string]types.PlanBundle{
		"p1": {Plan: types.DietPlan{ID: "p1", UserID: "other-user"}, DayTypes: []types.DietPlanDayTypeBundle{
			{DietPlanDayType: types.DietPlanDayType{ID: "dt-1", PlanID: "p1"}, Slots: []types.DietPlanSlotBundle{
				{DietPlanSlot: types.DietPlanSlot{ID: "sl-1", DayTypeID: "dt-1"}, Options: []types.DietPlanSlotOption{
					{ID: "opt-1", SlotID: "sl-1"},
				}},
			}},
		}},
	}
	h := newHandler(store, &fakeMealLogger{})

	rec := doRequest(h, "DELETE", "/api/v1/plans/p1/day-types/dt-1/slots/sl-1/options/opt-1", nil, nil)
	if rec.Code != http.StatusNotFound {
		t.Errorf("cross-user option access expected 404, got %d", rec.Code)
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

func TestHandleGetPlanDayTargetsForErr(t *testing.T) {
	store := newFakeMealStore()
	store.targetsForErr = errors.New("db down")
	h := newHandler(store, &fakeMealLogger{})

	rec := doRequest(h, "GET", "/api/v1/plans/day/2026-07-27", nil, nil)
	if rec.Code != http.StatusInternalServerError {
		t.Errorf("TargetsFor failure expected 500, got %d", rec.Code)
	}
}

func TestHandleGetPlanDayOverrideStoreErr(t *testing.T) {
	store := newFakeMealStore()
	store.getDayOverrideErr = errors.New("db down")
	h := newHandler(store, &fakeMealLogger{})

	rec := doRequest(h, "GET", "/api/v1/plans/day/2026-07-27", nil, nil)
	if rec.Code != http.StatusInternalServerError {
		t.Errorf("GetDayOverride failure expected 500, got %d", rec.Code)
	}
}

func TestHandleGetPlanDayActivePlanErr(t *testing.T) {
	store := newFakeMealStore()
	store.activePlanErr = errors.New("db down")
	h := newHandler(store, &fakeMealLogger{})

	rec := doRequest(h, "GET", "/api/v1/plans/day/2026-07-27", nil, nil)
	if rec.Code != http.StatusInternalServerError {
		t.Errorf("GetActivePlan failure expected 500, got %d", rec.Code)
	}
}

// TestHandleGetPlanDayOverrideDayTypeGone confirms an override pointing at a
// day-type that no longer exists (deleted after the override was set)
// degrades gracefully: no plan/day-type in the response instead of an error.
func TestHandleGetPlanDayOverrideDayTypeGone(t *testing.T) {
	store := newFakeMealStore()
	store.dayOverrides = map[string]types.DietPlanDayOverride{
		"test-user|2026-07-27": {UserID: "test-user", Date: "2026-07-27", DayTypeID: "dt-gone"},
	}
	store.getDayTypeErr = types.ErrNotFound
	h := newHandler(store, &fakeMealLogger{})

	rec := doRequest(h, "GET", "/api/v1/plans/day/2026-07-27", nil, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	got := decodeJSON[planDayView](t, rec)
	if !got.Overridden || got.PlanActive || got.DayType != nil {
		t.Errorf("expected overridden but unresolved day-type, got %+v", got)
	}
}

// TestHandleGetPlanDayOverrideWrongUserBundle confirms resolveDayTypeView
// rejects a day-type whose plan belongs to a different user, even though the
// day-type row itself was found.
func TestHandleGetPlanDayOverrideWrongUserBundle(t *testing.T) {
	store := newFakeMealStore()
	store.dayOverrides = map[string]types.DietPlanDayOverride{
		"test-user|2026-07-27": {UserID: "test-user", Date: "2026-07-27", DayTypeID: "dt-1"},
	}
	store.dayTypes = map[string]types.DietPlanDayType{"dt-1": {ID: "dt-1", PlanID: "p1"}}
	store.planBundles = map[string]types.PlanBundle{
		"p1": {Plan: types.DietPlan{ID: "p1", UserID: "other-user"}, DayTypes: []types.DietPlanDayTypeBundle{
			{DietPlanDayType: types.DietPlanDayType{ID: "dt-1", PlanID: "p1"}},
		}},
	}
	h := newHandler(store, &fakeMealLogger{})

	rec := doRequest(h, "GET", "/api/v1/plans/day/2026-07-27", nil, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	got := decodeJSON[planDayView](t, rec)
	if !got.Overridden || got.PlanActive || got.DayType != nil {
		t.Errorf("expected overridden but unresolved (cross-user) day-type, got %+v", got)
	}
}

// TestHandleGetPlanDayActivePlanBundleErr confirms a GetPlanBundle failure
// while resolving the active plan's cycle day-type degrades gracefully
// rather than failing the whole request -- the targets fallback already
// succeeded, so the day/slots view just stays unresolved.
func TestHandleGetPlanDayActivePlanBundleErr(t *testing.T) {
	store := newFakeMealStore()
	store.activePlan = types.DietPlan{
		ID: "p1", UserID: "test-user",
		CyclePattern: []string{"dt-low"}, CycleAnchorDate: "2026-07-27",
	}
	store.getPlanBundleErr = errors.New("db down")
	h := newHandler(store, &fakeMealLogger{})

	rec := doRequest(h, "GET", "/api/v1/plans/day/2026-07-27", nil, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	got := decodeJSON[planDayView](t, rec)
	if !got.PlanActive || got.DayType != nil {
		t.Errorf("expected plan active but no resolved day-type, got %+v", got)
	}
}

// --- handleGetSharedDayType ---

// sharedToken creates a share token for h's authenticated "test-user", the
// same setup TestShareTokenReadOnlyFlow uses, and returns the raw token for
// hitting the /shared/{token}/... prefix.
func sharedToken(t *testing.T, h *Handler) string {
	t.Helper()
	rec := doRequest(h, "POST", "/api/v1/auth/share-tokens", map[string]string{"label": "test"}, nil)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create share token: expected 201, got %d: %s", rec.Code, rec.Body.String())
	}
	return decodeJSON[types.NewShareTokenResponse](t, rec).Token
}

// TestHandleGetSharedDayTypeActivePlanCycle confirms the shared read-only
// endpoint surfaces the day-type name driving today's numbers -- so a shared
// dashboard doesn't show a stale flat target while a plan is active -- and
// that the response never carries slot/option content.
func TestHandleGetSharedDayTypeActivePlanCycle(t *testing.T) {
	store := newFakeMealStore()
	today := time.Now().UTC().Format(dateLayout)
	store.activePlan = types.DietPlan{
		ID: "p1", UserID: "test-user",
		CyclePattern: []string{"dt-low"}, CycleAnchorDate: today,
	}
	store.planBundles = map[string]types.PlanBundle{
		"p1": {Plan: types.DietPlan{ID: "p1", UserID: "test-user"}, DayTypes: []types.DietPlanDayTypeBundle{
			{DietPlanDayType: types.DietPlanDayType{ID: "dt-low", PlanID: "p1", Name: "Low-carb"}, Slots: []types.DietPlanSlotBundle{
				{DietPlanSlot: types.DietPlanSlot{ID: "sl-1", DayTypeID: "dt-low", Label: "Almoço"}},
			}},
		}},
	}
	store.targetsFor = types.DailyTargets{UserID: "test-user", Targets: types.Macros{Calories: 1800, Protein: 160}, WaterGoalMl: 2500}
	h := newHandler(store, &fakeMealLogger{})
	token := sharedToken(t, h)

	rec := doRequest(h, "GET", "/api/v1/shared/"+token+"/day-type", nil, map[string]string{"Authorization": ""})
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	got := decodeJSON[sharedDayTypeResponse](t, rec)
	if got.DayTypeName != "Low-carb" {
		t.Errorf("day_type_name = %q, want %q", got.DayTypeName, "Low-carb")
	}
	if got.Targets.Calories != 1800 || got.WaterGoalMl != 2500 {
		t.Errorf("targets = %+v water = %d, want 1800 kcal / 2500 ml", got.Targets, got.WaterGoalMl)
	}
	// The prescription itself (the day-type's slot label) must never leak
	// into the shared payload -- only the badge name and the numbers.
	if strings.Contains(rec.Body.String(), "Almo") {
		t.Errorf("shared day-type response leaked slot content: %s", rec.Body.String())
	}
}

// TestHandleGetSharedDayTypeNoPlan confirms the no-plan fallback keeps
// working unchanged: no day-type badge, just the flat targets -- the same
// shape web/src/routes/SharedDashboard.tsx already handles.
func TestHandleGetSharedDayTypeNoPlan(t *testing.T) {
	store := newFakeMealStore()
	store.activePlanErr = types.ErrNotFound
	store.targetsFor = types.DailyTargets{UserID: "test-user", Targets: types.Macros{Calories: 2000}}
	h := newHandler(store, &fakeMealLogger{})
	token := sharedToken(t, h)

	rec := doRequest(h, "GET", "/api/v1/shared/"+token+"/day-type", nil, map[string]string{"Authorization": ""})
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	got := decodeJSON[sharedDayTypeResponse](t, rec)
	if got.DayTypeName != "" {
		t.Errorf("expected no day-type badge without a plan, got %q", got.DayTypeName)
	}
	if got.Targets.Calories != 2000 {
		t.Errorf("targets = %+v, want flat fallback 2000 kcal", got.Targets)
	}
}

// TestHandleGetSharedDayTypeTargetsForErr confirms a TargetsFor failure fails
// the request instead of silently returning zeroed targets.
func TestHandleGetSharedDayTypeTargetsForErr(t *testing.T) {
	store := newFakeMealStore()
	store.targetsForErr = errors.New("db down")
	h := newHandler(store, &fakeMealLogger{})
	token := sharedToken(t, h)

	rec := doRequest(h, "GET", "/api/v1/shared/"+token+"/day-type", nil, map[string]string{"Authorization": ""})
	if rec.Code != http.StatusInternalServerError {
		t.Errorf("TargetsFor failure expected 500, got %d", rec.Code)
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

func TestHandleSetDayOverrideInvalidJSON(t *testing.T) {
	store := newFakeMealStore()
	h := newHandler(store, &fakeMealLogger{})

	rec := doRequestRawBody(h, "PUT", "/api/v1/plans/overrides/2026-07-27", "{not json")
	if rec.Code != http.StatusBadRequest {
		t.Errorf("invalid JSON expected 400, got %d", rec.Code)
	}
}

func TestHandleSetDayOverrideMissingDayTypeID(t *testing.T) {
	store := newFakeMealStore()
	h := newHandler(store, &fakeMealLogger{})

	rec := doRequest(h, "PUT", "/api/v1/plans/overrides/2026-07-27", map[string]any{}, nil)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("missing day_type_id expected 400, got %d", rec.Code)
	}
}

func TestHandleSetDayOverrideDayTypeNotFound(t *testing.T) {
	store := newFakeMealStore()
	h := newHandler(store, &fakeMealLogger{})

	rec := doRequest(h, "PUT", "/api/v1/plans/overrides/2026-07-27", map[string]any{"day_type_id": "missing"}, nil)
	if rec.Code != http.StatusNotFound {
		t.Errorf("override referencing missing day-type expected 404, got %d", rec.Code)
	}
}

func TestHandleSetDayOverridePlanNotFound(t *testing.T) {
	store := newFakeMealStore()
	store.dayTypes = map[string]types.DietPlanDayType{"dt-1": {ID: "dt-1", PlanID: "missing-plan"}}
	h := newHandler(store, &fakeMealLogger{})

	rec := doRequest(h, "PUT", "/api/v1/plans/overrides/2026-07-27", map[string]any{"day_type_id": "dt-1"}, nil)
	if rec.Code != http.StatusNotFound {
		t.Errorf("day-type whose plan is missing expected 404, got %d", rec.Code)
	}
}

func TestHandleSetDayOverrideStoreErr(t *testing.T) {
	store := newFakeMealStore()
	store.dayTypes = map[string]types.DietPlanDayType{"dt-1": {ID: "dt-1", PlanID: "p1"}}
	store.plans = map[string]types.DietPlan{"p1": {ID: "p1", UserID: "test-user"}}
	store.setDayOverrideErr = errors.New("db down")
	h := newHandler(store, &fakeMealLogger{})

	rec := doRequest(h, "PUT", "/api/v1/plans/overrides/2026-07-27", map[string]any{"day_type_id": "dt-1"}, nil)
	if rec.Code != http.StatusInternalServerError {
		t.Errorf("SetDayOverride failure expected 500, got %d", rec.Code)
	}
}

func TestHandleSetDayOverrideCallsRefreshTodayTargets(t *testing.T) {
	store := newFakeMealStore()
	store.dayTypes = map[string]types.DietPlanDayType{"dt-1": {ID: "dt-1", PlanID: "p1"}}
	store.plans = map[string]types.DietPlan{"p1": {ID: "p1", UserID: "test-user"}}
	store.refreshTodayTargetsErr = errors.New("refresh failed")
	h := newHandler(store, &fakeMealLogger{})

	rec := doRequest(h, "PUT", "/api/v1/plans/overrides/2026-07-27", map[string]any{"day_type_id": "dt-1"}, nil)
	if rec.Code != http.StatusInternalServerError {
		t.Errorf("RefreshTodayTargets failure expected 500, got %d", rec.Code)
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
