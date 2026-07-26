package api

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/gsaraiva2109/dietdaemon/core/types"
)

// ---------------------------------------------------------------------------
// Diet plan handlers -- CRUD for plans, day-types, slots, slot options, and
// day overrides, plus the resolved day-type+slots+targets view the
// dashboard, week strip, and bot all read from.
//
// The app never generates a plan: every macro number here is typed in by the
// user from what a nutritionist prescribed. Mutations that can change what
// today's targets resolve to (plan create/edit/delete, day-type edit,
// override set/clear) call store.RefreshTodayTargets so the dashboard's
// mirrored daily_rollups row stays in sync -- see the package doc on
// TargetsFor for the fallback invariant this maintains.
// ---------------------------------------------------------------------------

// ---------------------------------------------------------------------------
// Plans
// ---------------------------------------------------------------------------

func (h *Handler) handleListPlans(w http.ResponseWriter, r *http.Request, userID string) {
	plans, err := h.store.ListPlans(r.Context(), userID)
	if err != nil {
		h.writeErr(w, err)
		return
	}
	if plans == nil {
		plans = []types.DietPlan{}
	}
	_ = json.NewEncoder(w).Encode(plans)
}

func (h *Handler) handleCreatePlan(w http.ResponseWriter, r *http.Request, userID string) {
	var body types.DietPlan
	if err := decodeRequestJSON(r, &body); err != nil {
		writeValidationError(w, "invalid JSON body")
		return
	}
	if body.Name == "" {
		writeValidationError(w, "name is required")
		return
	}
	if !validPlanDates(body.ValidFrom, body.ValidTo, body.CycleAnchorDate) {
		writeValidationError(w, "valid_from and cycle_anchor_date are required YYYY-MM-DD dates; valid_to must be empty or a YYYY-MM-DD date on/after valid_from")
		return
	}
	if len(body.CyclePattern) > 0 {
		// No day-type can exist yet for a plan that doesn't exist yet.
		writeValidationError(w, "cycle_pattern must be empty on create; add day-types first, then set the pattern via update")
		return
	}
	body.UserID = userID

	created, err := h.store.CreatePlan(r.Context(), body)
	if err != nil {
		h.writeErr(w, err)
		return
	}
	if err := h.store.RefreshTodayTargets(r.Context(), userID); err != nil {
		h.writeErr(w, err)
		return
	}
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(created)
}

func (h *Handler) handleGetActivePlan(w http.ResponseWriter, r *http.Request, userID string) {
	today := time.Now().In(h.loc).Format(dateLayout)
	p, err := h.store.GetActivePlan(r.Context(), userID, today)
	if err != nil {
		h.writeErr(w, err) // ErrNotFound → 404; frontend treats as "no active plan".
		return
	}
	_ = json.NewEncoder(w).Encode(p)
}

// handleGetPlan returns the full plan tree (day-types, slots, options) in
// one shot, since a builder UI editing a plan needs all of it.
func (h *Handler) handleGetPlan(w http.ResponseWriter, r *http.Request, userID string) {
	planID := r.PathValue("planID")
	bundle, err := h.store.GetPlanBundle(r.Context(), planID)
	if err != nil {
		h.writeErr(w, err)
		return
	}
	if bundle.Plan.UserID != userID {
		h.writeErr(w, types.ErrNotFound)
		return
	}
	_ = json.NewEncoder(w).Encode(bundle)
}

func (h *Handler) handleUpdatePlan(w http.ResponseWriter, r *http.Request, userID string) {
	planID := r.PathValue("planID")
	plan, err := h.ownedPlan(r.Context(), userID, planID)
	if err != nil {
		h.writeErr(w, err)
		return
	}

	var body types.DietPlan
	if err := decodeRequestJSON(r, &body); err != nil {
		writeValidationError(w, "invalid JSON body")
		return
	}
	if body.Name == "" {
		writeValidationError(w, "name is required")
		return
	}
	if !validPlanDates(body.ValidFrom, body.ValidTo, body.CycleAnchorDate) {
		writeValidationError(w, "valid_from and cycle_anchor_date are required YYYY-MM-DD dates; valid_to must be empty or a YYYY-MM-DD date on/after valid_from")
		return
	}
	if len(body.CyclePattern) > 0 {
		bundle, err := h.store.GetPlanBundle(r.Context(), planID)
		if err != nil {
			h.writeErr(w, err)
			return
		}
		known := make(map[string]bool, len(bundle.DayTypes))
		for _, dt := range bundle.DayTypes {
			known[dt.ID] = true
		}
		for _, id := range body.CyclePattern {
			if !known[id] {
				writeValidationError(w, "cycle_pattern references a day-type that does not belong to this plan: "+id)
				return
			}
		}
	}

	body.ID = plan.ID
	body.UserID = userID
	if err := h.store.UpdatePlan(r.Context(), body); err != nil {
		h.writeErr(w, err)
		return
	}
	if err := h.store.RefreshTodayTargets(r.Context(), userID); err != nil {
		h.writeErr(w, err)
		return
	}
	_ = json.NewEncoder(w).Encode(body)
}

func (h *Handler) handleDeletePlan(w http.ResponseWriter, r *http.Request, userID string) {
	planID := r.PathValue("planID")
	if err := h.store.DeletePlan(r.Context(), userID, planID); err != nil {
		h.writeErr(w, err)
		return
	}
	if err := h.store.RefreshTodayTargets(r.Context(), userID); err != nil {
		h.writeErr(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// ---------------------------------------------------------------------------
// Day-types
// ---------------------------------------------------------------------------

func (h *Handler) handleCreateDayType(w http.ResponseWriter, r *http.Request, userID string) {
	planID := r.PathValue("planID")
	if _, err := h.ownedPlan(r.Context(), userID, planID); err != nil {
		h.writeErr(w, err)
		return
	}
	var dt types.DietPlanDayType
	if err := decodeRequestJSON(r, &dt); err != nil {
		writeValidationError(w, "invalid JSON body")
		return
	}
	if !validDayType(dt) {
		writeValidationError(w, "name is required, position must be >= 0, targets must be finite and non-negative, water_goal_ml must be >= 0")
		return
	}
	dt.PlanID = planID

	created, err := h.store.CreateDayType(r.Context(), dt)
	if err != nil {
		h.writeErr(w, err)
		return
	}
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(created)
}

func (h *Handler) handleUpdateDayType(w http.ResponseWriter, r *http.Request, userID string) {
	planID := r.PathValue("planID")
	dayTypeID := r.PathValue("dayTypeID")
	if _, err := h.ownedDayType(r.Context(), userID, planID, dayTypeID); err != nil {
		h.writeErr(w, err)
		return
	}
	var dt types.DietPlanDayType
	if err := decodeRequestJSON(r, &dt); err != nil {
		writeValidationError(w, "invalid JSON body")
		return
	}
	if !validDayType(dt) {
		writeValidationError(w, "name is required, position must be >= 0, targets must be finite and non-negative, water_goal_ml must be >= 0")
		return
	}
	dt.ID = dayTypeID
	dt.PlanID = planID

	if err := h.store.UpdateDayType(r.Context(), dt); err != nil {
		h.writeErr(w, err)
		return
	}
	// A day-type's macros can be what today's targets currently resolve to.
	if err := h.store.RefreshTodayTargets(r.Context(), userID); err != nil {
		h.writeErr(w, err)
		return
	}
	_ = json.NewEncoder(w).Encode(dt)
}

func (h *Handler) handleDeleteDayType(w http.ResponseWriter, r *http.Request, userID string) {
	planID := r.PathValue("planID")
	dayTypeID := r.PathValue("dayTypeID")
	if _, err := h.ownedDayType(r.Context(), userID, planID, dayTypeID); err != nil {
		h.writeErr(w, err)
		return
	}
	if err := h.store.DeleteDayType(r.Context(), dayTypeID); err != nil {
		h.writeErr(w, err)
		return
	}
	if err := h.store.RefreshTodayTargets(r.Context(), userID); err != nil {
		h.writeErr(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// ---------------------------------------------------------------------------
// Slots
// ---------------------------------------------------------------------------

func (h *Handler) handleCreateSlot(w http.ResponseWriter, r *http.Request, userID string) {
	planID := r.PathValue("planID")
	dayTypeID := r.PathValue("dayTypeID")
	if _, err := h.ownedDayType(r.Context(), userID, planID, dayTypeID); err != nil {
		h.writeErr(w, err)
		return
	}
	var sl types.DietPlanSlot
	if err := decodeRequestJSON(r, &sl); err != nil {
		writeValidationError(w, "invalid JSON body")
		return
	}
	if !validSlot(sl) {
		writeValidationError(w, "label is required and position must be >= 0")
		return
	}
	sl.DayTypeID = dayTypeID

	created, err := h.store.CreateSlot(r.Context(), sl)
	if err != nil {
		h.writeErr(w, err)
		return
	}
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(created)
}

func (h *Handler) handleUpdateSlot(w http.ResponseWriter, r *http.Request, userID string) {
	planID := r.PathValue("planID")
	dayTypeID := r.PathValue("dayTypeID")
	slotID := r.PathValue("slotID")
	if _, err := h.ownedSlot(r.Context(), userID, planID, dayTypeID, slotID); err != nil {
		h.writeErr(w, err)
		return
	}
	var sl types.DietPlanSlot
	if err := decodeRequestJSON(r, &sl); err != nil {
		writeValidationError(w, "invalid JSON body")
		return
	}
	if !validSlot(sl) {
		writeValidationError(w, "label is required and position must be >= 0")
		return
	}
	sl.ID = slotID
	sl.DayTypeID = dayTypeID

	if err := h.store.UpdateSlot(r.Context(), sl); err != nil {
		h.writeErr(w, err)
		return
	}
	_ = json.NewEncoder(w).Encode(sl)
}

func (h *Handler) handleDeleteSlot(w http.ResponseWriter, r *http.Request, userID string) {
	planID := r.PathValue("planID")
	dayTypeID := r.PathValue("dayTypeID")
	slotID := r.PathValue("slotID")
	if _, err := h.ownedSlot(r.Context(), userID, planID, dayTypeID, slotID); err != nil {
		h.writeErr(w, err)
		return
	}
	if err := h.store.DeleteSlot(r.Context(), slotID); err != nil {
		h.writeErr(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// ---------------------------------------------------------------------------
// Slot options -- each backed by a meal_templates row (owner_kind = "plan")
// holding the prescribed foods. store.GetTemplates filters those out of the
// user's own Templates list.
// ---------------------------------------------------------------------------

type slotOptionBody struct {
	Label    string               `json:"label"`
	Position int                  `json:"position"`
	Items    []types.ResolvedItem `json:"items"`
}

func validSlotOptionBody(b slotOptionBody) bool {
	return b.Label != "" && len(b.Items) > 0
}

func (h *Handler) handleCreateSlotOption(w http.ResponseWriter, r *http.Request, userID string) {
	planID := r.PathValue("planID")
	dayTypeID := r.PathValue("dayTypeID")
	slotID := r.PathValue("slotID")
	if _, err := h.ownedSlot(r.Context(), userID, planID, dayTypeID, slotID); err != nil {
		h.writeErr(w, err)
		return
	}
	var body slotOptionBody
	if err := decodeRequestJSON(r, &body); err != nil {
		writeValidationError(w, "invalid JSON body")
		return
	}
	if !validSlotOptionBody(body) {
		writeValidationError(w, "label and items are required")
		return
	}

	now := time.Now().UTC()
	tmpl := types.MealTemplate{
		ID:        newHandlerID(),
		UserID:    userID,
		Name:      body.Label,
		Items:     body.Items,
		OwnerKind: types.TemplateOwnerPlan,
		CreatedAt: now,
		LastUsed:  now,
	}
	if err := h.store.SaveTemplate(r.Context(), tmpl); err != nil {
		h.writeErr(w, err)
		return
	}

	created, err := h.store.CreateSlotOption(r.Context(), types.DietPlanSlotOption{
		SlotID: slotID, Position: body.Position, Label: body.Label, TemplateID: tmpl.ID,
	})
	if err != nil {
		h.writeErr(w, err)
		return
	}
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(created)
}

func (h *Handler) handleUpdateSlotOption(w http.ResponseWriter, r *http.Request, userID string) {
	planID := r.PathValue("planID")
	dayTypeID := r.PathValue("dayTypeID")
	slotID := r.PathValue("slotID")
	optionID := r.PathValue("optID")
	existing, err := h.ownedOption(r.Context(), userID, planID, dayTypeID, slotID, optionID)
	if err != nil {
		h.writeErr(w, err)
		return
	}
	var body slotOptionBody
	if err := decodeRequestJSON(r, &body); err != nil {
		writeValidationError(w, "invalid JSON body")
		return
	}
	if !validSlotOptionBody(body) {
		writeValidationError(w, "label and items are required")
		return
	}

	now := time.Now().UTC()
	tmpl := types.MealTemplate{
		ID:        existing.TemplateID,
		UserID:    userID,
		Name:      body.Label,
		Items:     body.Items,
		OwnerKind: types.TemplateOwnerPlan,
		CreatedAt: now,
		LastUsed:  now,
	}
	if err := h.store.SaveTemplate(r.Context(), tmpl); err != nil {
		h.writeErr(w, err)
		return
	}

	opt := types.DietPlanSlotOption{ID: optionID, SlotID: slotID, Position: body.Position, Label: body.Label, TemplateID: existing.TemplateID}
	if err := h.store.UpdateSlotOption(r.Context(), opt); err != nil {
		h.writeErr(w, err)
		return
	}
	_ = json.NewEncoder(w).Encode(opt)
}

func (h *Handler) handleDeleteSlotOption(w http.ResponseWriter, r *http.Request, userID string) {
	planID := r.PathValue("planID")
	dayTypeID := r.PathValue("dayTypeID")
	slotID := r.PathValue("slotID")
	optionID := r.PathValue("optID")
	if _, err := h.ownedOption(r.Context(), userID, planID, dayTypeID, slotID, optionID); err != nil {
		h.writeErr(w, err)
		return
	}
	// The backing meal_templates row is left in place: template_logs
	// (adherence history) may still reference it via ON DELETE CASCADE, and
	// it never surfaces in the user's own Templates list regardless
	// (owner_kind = "plan").
	if err := h.store.DeleteSlotOption(r.Context(), optionID); err != nil {
		h.writeErr(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// ---------------------------------------------------------------------------
// Resolved day view -- what the dashboard, week strip, and bot all read.
// ---------------------------------------------------------------------------

// planDayView is the resolved day-type, its slots, and the targets in effect
// for one date: override, else the active plan's cycle pattern, else the
// no-plan fallback (see TargetsFor). DayType and Slots are empty when no
// plan or override governs the date -- Targets still reflects the flat
// daily_targets fallback either way.
type planDayView struct {
	Date       string                     `json:"date"`
	PlanActive bool                       `json:"plan_active"`
	Overridden bool                       `json:"overridden"`
	DayType    *types.DietPlanDayType     `json:"day_type"`
	Slots      []types.DietPlanSlotBundle `json:"slots"`
	Targets    types.DailyTargets         `json:"targets"`
}

func (h *Handler) handleGetPlanDay(w http.ResponseWriter, r *http.Request, userID string) {
	date := r.PathValue("date")
	if !isPlanDate(date) {
		writeValidationError(w, "date must be YYYY-MM-DD")
		return
	}

	targets, err := h.store.TargetsFor(r.Context(), userID, date)
	if err != nil {
		h.writeErr(w, err)
		return
	}
	view := planDayView{Date: date, Targets: targets, Slots: []types.DietPlanSlotBundle{}}

	override, err := h.store.GetDayOverride(r.Context(), userID, date)
	switch {
	case err == nil:
		view.Overridden = true
		if dt, slots, ok := h.resolveDayTypeView(r.Context(), userID, override.DayTypeID); ok {
			view.DayType = &dt
			view.Slots = slots
			view.PlanActive = true
		}
	case !errors.Is(err, types.ErrNotFound):
		h.writeErr(w, err)
		return
	default:
		if plan, err := h.store.GetActivePlan(r.Context(), userID, date); err == nil {
			view.PlanActive = true
			if len(plan.CyclePattern) > 0 {
				// The plan ID is already known here (unlike the override
				// case), so go straight to its bundle instead of round-
				// tripping through GetDayType to discover it.
				if idx, err := cycleIndex(plan.CycleAnchorDate, date, len(plan.CyclePattern)); err == nil {
					dayTypeID := plan.CyclePattern[idx]
					if bundle, err := h.store.GetPlanBundle(r.Context(), plan.ID); err == nil {
						if dt, slots, ok := findDayTypeInBundle(bundle, dayTypeID); ok {
							view.DayType = &dt
							view.Slots = slots
						}
					}
				}
			}
		} else if !errors.Is(err, types.ErrNotFound) {
			h.writeErr(w, err)
			return
		}
	}

	_ = json.NewEncoder(w).Encode(view)
}

// resolveDayTypeView loads a day-type and its slots for the day view,
// verifying along the way that it belongs to a plan owned by userID. Used
// for the override case, where (unlike the active-plan case) the plan ID
// isn't already known, so GetDayType is needed first to discover it.
func (h *Handler) resolveDayTypeView(ctx context.Context, userID, dayTypeID string) (types.DietPlanDayType, []types.DietPlanSlotBundle, bool) {
	dt, err := h.store.GetDayType(ctx, dayTypeID)
	if err != nil {
		return types.DietPlanDayType{}, nil, false
	}
	bundle, err := h.store.GetPlanBundle(ctx, dt.PlanID)
	if err != nil || bundle.Plan.UserID != userID {
		return types.DietPlanDayType{}, nil, false
	}
	return findDayTypeInBundle(bundle, dayTypeID)
}

// findDayTypeInBundle looks up a day-type (and its slots) by ID within an
// already-loaded plan bundle.
func findDayTypeInBundle(bundle types.PlanBundle, dayTypeID string) (types.DietPlanDayType, []types.DietPlanSlotBundle, bool) {
	for _, b := range bundle.DayTypes {
		if b.ID == dayTypeID {
			return b.DietPlanDayType, b.Slots, true
		}
	}
	return types.DietPlanDayType{}, nil, false
}

// cycleIndex returns (date - anchor) mod patternLen as a Euclidean modulus,
// so dates before anchor resolve to a valid index instead of Go's %
// returning a negative one.
func cycleIndex(anchor, date string, patternLen int) (int, error) {
	a, err := time.Parse(dateLayout, anchor)
	if err != nil {
		return 0, err
	}
	d, err := time.Parse(dateLayout, date)
	if err != nil {
		return 0, err
	}
	days := int(d.Sub(a).Hours() / 24)
	return ((days % patternLen) + patternLen) % patternLen, nil
}

// ---------------------------------------------------------------------------
// Day overrides
// ---------------------------------------------------------------------------

type dayOverrideBody struct {
	DayTypeID string `json:"day_type_id"`
}

func (h *Handler) handleSetDayOverride(w http.ResponseWriter, r *http.Request, userID string) {
	date := r.PathValue("date")
	if !isPlanDate(date) {
		writeValidationError(w, "date must be YYYY-MM-DD")
		return
	}
	var body dayOverrideBody
	if err := decodeRequestJSON(r, &body); err != nil {
		writeValidationError(w, "invalid JSON body")
		return
	}
	if body.DayTypeID == "" {
		writeValidationError(w, "day_type_id is required")
		return
	}
	dt, err := h.store.GetDayType(r.Context(), body.DayTypeID)
	if err != nil {
		h.writeErr(w, err)
		return
	}
	plan, err := h.store.GetPlan(r.Context(), dt.PlanID)
	if err != nil {
		h.writeErr(w, err)
		return
	}
	if plan.UserID != userID {
		h.writeErr(w, types.ErrNotFound)
		return
	}

	if err := h.store.SetDayOverride(r.Context(), types.DietPlanDayOverride{UserID: userID, Date: date, DayTypeID: body.DayTypeID}); err != nil {
		h.writeErr(w, err)
		return
	}
	if err := h.store.RefreshTodayTargets(r.Context(), userID); err != nil {
		h.writeErr(w, err)
		return
	}
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func (h *Handler) handleDeleteDayOverride(w http.ResponseWriter, r *http.Request, userID string) {
	date := r.PathValue("date")
	if !isPlanDate(date) {
		writeValidationError(w, "date must be YYYY-MM-DD")
		return
	}
	if err := h.store.DeleteDayOverride(r.Context(), userID, date); err != nil {
		h.writeErr(w, err)
		return
	}
	if err := h.store.RefreshTodayTargets(r.Context(), userID); err != nil {
		h.writeErr(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// ---------------------------------------------------------------------------
// Ownership checks
// ---------------------------------------------------------------------------

func (h *Handler) ownedPlan(ctx context.Context, userID, planID string) (types.DietPlan, error) {
	p, err := h.store.GetPlan(ctx, planID)
	if err != nil {
		return types.DietPlan{}, err
	}
	if p.UserID != userID {
		return types.DietPlan{}, types.ErrNotFound
	}
	return p, nil
}

func (h *Handler) ownedDayType(ctx context.Context, userID, planID, dayTypeID string) (types.DietPlanDayType, error) {
	if _, err := h.ownedPlan(ctx, userID, planID); err != nil {
		return types.DietPlanDayType{}, err
	}
	dt, err := h.store.GetDayType(ctx, dayTypeID)
	if err != nil {
		return types.DietPlanDayType{}, err
	}
	if dt.PlanID != planID {
		return types.DietPlanDayType{}, types.ErrNotFound
	}
	return dt, nil
}

// ownedSlot and ownedOption walk the plan bundle rather than issuing a point
// query: CreateSlot/DeleteSlot/CreateSlotOption/DeleteSlotOption take no
// ownership parameters of their own (see store_plan.go), so confirming that a
// slot or option actually belongs to the plan/day-type named in the URL --
// not just that some row with a matching ID exists somewhere -- has to
// happen here, on every call, including create.
func (h *Handler) ownedSlot(ctx context.Context, userID, planID, dayTypeID, slotID string) (types.DietPlanSlot, error) {
	bundle, err := h.store.GetPlanBundle(ctx, planID)
	if err != nil {
		return types.DietPlanSlot{}, err
	}
	if bundle.Plan.UserID != userID {
		return types.DietPlanSlot{}, types.ErrNotFound
	}
	for _, dt := range bundle.DayTypes {
		if dt.ID != dayTypeID {
			continue
		}
		for _, sl := range dt.Slots {
			if sl.ID == slotID {
				return sl.DietPlanSlot, nil
			}
		}
	}
	return types.DietPlanSlot{}, types.ErrNotFound
}

func (h *Handler) ownedOption(ctx context.Context, userID, planID, dayTypeID, slotID, optionID string) (types.DietPlanSlotOption, error) {
	bundle, err := h.store.GetPlanBundle(ctx, planID)
	if err != nil {
		return types.DietPlanSlotOption{}, err
	}
	if bundle.Plan.UserID != userID {
		return types.DietPlanSlotOption{}, types.ErrNotFound
	}
	for _, dt := range bundle.DayTypes {
		if dt.ID != dayTypeID {
			continue
		}
		for _, sl := range dt.Slots {
			if sl.ID != slotID {
				continue
			}
			for _, opt := range sl.Options {
				if opt.ID == optionID {
					return opt, nil
				}
			}
		}
	}
	return types.DietPlanSlotOption{}, types.ErrNotFound
}

// ---------------------------------------------------------------------------
// Validation
// ---------------------------------------------------------------------------

func isPlanDate(s string) bool {
	_, err := time.Parse(dateLayout, s)
	return err == nil
}

// validPlanDates checks format only, deliberately not "not in the future"
// (unlike validDate in validation.go): a plan's valid_from is routinely a
// future start date.
func validPlanDates(validFrom, validTo, anchor string) bool {
	if !isPlanDate(validFrom) || !isPlanDate(anchor) {
		return false
	}
	if validTo != "" && (!isPlanDate(validTo) || validTo < validFrom) {
		return false
	}
	return true
}

func validDayType(dt types.DietPlanDayType) bool {
	return dt.Name != "" && dt.Position >= 0 && validMacros(dt.Targets) && dt.WaterGoalMl >= 0
}

func validSlot(sl types.DietPlanSlot) bool {
	return sl.Label != "" && sl.Position >= 0
}
