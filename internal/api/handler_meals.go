package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/gsaraiva2109/dietdaemon/core/types"
	"github.com/gsaraiva2109/dietdaemon/internal/scheduler"
)

// ---------------------------------------------------------------------------
// Meals & rollups handlers -- daily rollups, meal CRUD, targets, nudges, budget, meal logging.
// ---------------------------------------------------------------------------

func (h *Handler) handleRollupsToday(w http.ResponseWriter, r *http.Request, userID string) {
	today := time.Now().In(h.loc).Format("2006-01-02")
	rollup, err := h.store.GetRollup(r.Context(), userID, today)
	if err != nil {
		h.writeErr(w, err)
		return
	}
	_ = json.NewEncoder(w).Encode(rollup)
}

func (h *Handler) handleRollupsRange(w http.ResponseWriter, r *http.Request, userID string) {
	start := r.URL.Query().Get("start")
	end := r.URL.Query().Get("end")
	if start == "" || end == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "start and end query params required (YYYY-MM-DD)"})
		return
	}
	rollups, err := h.store.GetRollups(r.Context(), userID, start, end)
	if err != nil {
		h.writeErr(w, err)
		return
	}
	if rollups == nil {
		rollups = []types.DailyRollup{}
	}
	_ = json.NewEncoder(w).Encode(rollups)
}

func (h *Handler) handleMealsList(w http.ResponseWriter, r *http.Request, userID string) {
	limit := 10
	if s := r.URL.Query().Get("limit"); s != "" {
		if n, err := strconv.Atoi(s); err == nil && n > 0 && n <= 100 {
			limit = n
		}
	}
	meals, err := h.store.RecentMeals(r.Context(), userID, limit)
	if err != nil {
		h.writeErr(w, err)
		return
	}
	if meals == nil {
		meals = []types.Meal{}
	}
	_ = json.NewEncoder(w).Encode(meals)
}

func (h *Handler) handleMealDetail(w http.ResponseWriter, r *http.Request, userID string) {
	mealID := r.PathValue("mealID")
	meal, err := h.store.GetMeal(r.Context(), mealID)
	if err != nil {
		h.writeErr(w, err)
		return
	}
	// Only return meals belonging to the authenticated user.
	if meal.UserID != userID {
		h.writeErr(w, types.ErrNotFound)
		return
	}
	_ = json.NewEncoder(w).Encode(meal)
}

func (h *Handler) handleCorrectItem(w http.ResponseWriter, r *http.Request, userID string) {
	mealID := r.PathValue("mealID")
	itemIDStr := r.PathValue("itemID")

	itemIndex, err := strconv.Atoi(itemIDStr)
	if err != nil || itemIndex < 0 {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "itemID must be a non-negative integer index"})
		return
	}

	var corrected types.ResolvedItem
	if err := json.NewDecoder(r.Body).Decode(&corrected); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid JSON body: " + err.Error()})
		return
	}

	if err := h.store.CorrectMealItem(r.Context(), userID, mealID, itemIndex, corrected); err != nil {
		h.writeErr(w, err)
		return
	}

	// Return the updated meal.
	meal, err := h.store.GetMeal(r.Context(), mealID)
	if err != nil {
		h.writeErr(w, err)
		return
	}
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(meal)
}

func (h *Handler) handleAddItem(w http.ResponseWriter, r *http.Request, userID string) {
	mealID := r.PathValue("mealID")

	var item types.ResolvedItem
	if err := json.NewDecoder(r.Body).Decode(&item); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid JSON body: " + err.Error()})
		return
	}
	if err := h.store.AddMealItem(r.Context(), userID, mealID, item); err != nil {
		h.writeErr(w, err)
		return
	}
	h.returnMeal(w, r, mealID, userID)
}

func (h *Handler) handleDeleteItem(w http.ResponseWriter, r *http.Request, userID string) {
	mealID := r.PathValue("mealID")
	itemIndex, err := strconv.Atoi(r.PathValue("itemID"))
	if err != nil || itemIndex < 0 {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "itemID must be a non-negative integer index"})
		return
	}
	if err := h.store.DeleteMealItem(r.Context(), userID, mealID, itemIndex); err != nil {
		h.writeErr(w, err)
		return
	}
	h.returnMeal(w, r, mealID, userID)
}

// returnMeal writes the meal as JSON, enforcing user ownership.
func (h *Handler) returnMeal(w http.ResponseWriter, r *http.Request, mealID, userID string) {
	meal, err := h.store.GetMeal(r.Context(), mealID)
	if err != nil {
		h.writeErr(w, err)
		return
	}
	if meal.UserID != userID {
		h.writeErr(w, types.ErrNotFound)
		return
	}
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(meal)
}

func (h *Handler) handleGetTargets(w http.ResponseWriter, r *http.Request, userID string) {
	dt, err := h.store.GetTargets(r.Context(), userID)
	if err != nil {
		h.writeErr(w, err)
		return
	}
	_ = json.NewEncoder(w).Encode(dt)
}

func (h *Handler) handleSetTargets(w http.ResponseWriter, r *http.Request, userID string) {
	var body types.Macros
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid JSON body: " + err.Error()})
		return
	}
	dt := types.DailyTargets{UserID: userID, Targets: body}
	if err := h.store.SetTargets(r.Context(), dt); err != nil {
		h.writeErr(w, err)
		return
	}
	// Reflect immediately on the dashboard, which reads targets from the rollup.
	today := time.Now().In(h.loc).Format("2006-01-02")
	if err := h.store.UpdateRollupTargets(r.Context(), userID, today, body); err != nil {
		h.writeErr(w, err)
		return
	}
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(dt)
}

// nudgeRuleView is the effective (default merged with the user's override)
// state of one nudge rule, as returned by GET /settings/nudges.
type nudgeRuleView struct {
	RuleID  string          `json:"rule_id"`
	Kind    string          `json:"kind"` // "macro", "health", or "digest"
	Enabled bool            `json:"enabled"`
	Rule    json.RawMessage `json:"rule"` // the rule's own fields, with any override applied
}

// buildNudgeRuleView merges a stored override onto a copy of the rule's
// hardcoded default, mirroring scheduler.resolveRule's behavior so the UI
// shows exactly what the scheduler will evaluate.
func buildNudgeRuleView[T any](ruleID, kind string, base T, overrides map[string]types.NudgeRuleConfig) nudgeRuleView {
	enabled := true
	if c, ok := overrides[ruleID]; ok {
		enabled = c.Enabled
		if enabled && len(c.Params) > 0 {
			_ = json.Unmarshal(c.Params, &base)
		}
	}
	ruleJSON, _ := json.Marshal(base)
	return nudgeRuleView{RuleID: ruleID, Kind: kind, Enabled: enabled, Rule: ruleJSON}
}

// buildNudgeRuleViewWeeklyBudget is a sibling of buildNudgeRuleView for the
// weekly-budget rule kind. Unlike macro/health/digest rules, the weekly budget
// is OFF by default (enabled=false) until the user explicitly opts in.
func buildNudgeRuleViewWeeklyBudget(ruleID, kind string, base scheduler.WeeklyBudgetRule, overrides map[string]types.NudgeRuleConfig) nudgeRuleView {
	enabled := false
	if c, ok := overrides[ruleID]; ok {
		enabled = c.Enabled
	}
	ruleJSON, _ := json.Marshal(base)
	return nudgeRuleView{RuleID: ruleID, Kind: kind, Enabled: enabled, Rule: ruleJSON}
}

// handleGetNudgeSettings returns every built-in nudge rule (macro, health,
// digest) merged with the user's stored overrides, so the UI always shows
// the full rule set with each rule's effective enabled/params state.
func (h *Handler) handleGetNudgeSettings(w http.ResponseWriter, r *http.Request, userID string) {
	overrides, err := h.store.GetNudgeRuleConfig(r.Context(), userID)
	if err != nil {
		h.writeErr(w, err)
		return
	}
	byID := make(map[string]types.NudgeRuleConfig, len(overrides))
	for _, c := range overrides {
		byID[c.RuleID] = c
	}

	views := make([]nudgeRuleView, 0, len(scheduler.DefaultRules())+len(scheduler.DefaultHealthRules())+len(scheduler.DefaultDigestRules())+len(scheduler.DefaultWeeklyBudgetRules()))
	for _, base := range scheduler.DefaultRules() {
		views = append(views, buildNudgeRuleView(base.ID, "macro", base, byID))
	}
	for _, base := range scheduler.DefaultHealthRules() {
		views = append(views, buildNudgeRuleView(base.ID, "health", base, byID))
	}
	for _, base := range scheduler.DefaultDigestRules() {
		views = append(views, buildNudgeRuleView(base.ID, "digest", base, byID))
	}
	for _, base := range scheduler.DefaultWeeklyBudgetRules() {
		views = append(views, buildNudgeRuleViewWeeklyBudget(base.ID, "weekly-budget", base, byID))
	}

	_ = json.NewEncoder(w).Encode(views)
}

// handleSetNudgeSettings accepts one rule's override. Set reset=true to
// remove the override and fall back to the hardcoded default.
func (h *Handler) handleSetNudgeSettings(w http.ResponseWriter, r *http.Request, userID string) {
	var body struct {
		RuleID  string          `json:"rule_id"`
		Enabled bool            `json:"enabled"`
		Params  json.RawMessage `json:"params,omitempty"`
		Reset   bool            `json:"reset"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid JSON body: " + err.Error()})
		return
	}
	if body.RuleID == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "rule_id is required"})
		return
	}

	if body.Reset {
		if err := h.store.DeleteNudgeRuleConfig(r.Context(), userID, body.RuleID); err != nil {
			h.writeErr(w, err)
			return
		}
	} else if err := h.store.SetNudgeRuleConfig(r.Context(), userID, body.RuleID, body.Enabled, body.Params); err != nil {
		h.writeErr(w, err)
		return
	}
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

// handleGetBudgetWeekly returns the plain daily target and effective weekly
// rolling target for calories and protein, computed from the current week's
// consumption data. GET /api/v1/budget/weekly
func (h *Handler) handleGetBudgetWeekly(w http.ResponseWriter, r *http.Request, userID string) {
	now := time.Now().In(h.loc)
	today := now.Format("2006-01-02")

	// Compute calendar week (Monday-Sunday) bounds.
	weekday := now.Weekday()
	daysFromMonday := int(weekday) - int(time.Monday)
	if weekday == time.Sunday {
		daysFromMonday = 6
	}
	monday := now.AddDate(0, 0, -daysFromMonday)
	sunday := monday.AddDate(0, 0, 6)
	daysRemaining := 7 - daysFromMonday

	rollups, err := h.store.GetRollups(r.Context(), userID, monday.Format("2006-01-02"), sunday.Format("2006-01-02"))
	if err != nil {
		h.writeErr(w, err)
		return
	}

	targets, err := h.store.GetTargets(r.Context(), userID)
	if err != nil {
		h.writeErr(w, err)
		return
	}

	// Sum consumed macros from prior days (dates strictly before today).
	var consumedCal, consumedProtein float64
	for _, r := range rollups {
		if r.Date >= today {
			continue
		}
		consumedCal += r.Consumed.Calories
		consumedProtein += r.Consumed.Protein
	}

	resp := map[string]map[string]float64{}

	if targets.Targets.Calories > 0 {
		effective := scheduler.EffectiveWeeklyTarget(targets.Targets.Calories, consumedCal, daysRemaining, 0.70, 1.30)
		resp["calories"] = map[string]float64{"plain": targets.Targets.Calories, "effective": effective}
	} else {
		resp["calories"] = map[string]float64{"plain": 0, "effective": 0}
	}

	if targets.Targets.Protein > 0 {
		effective := scheduler.EffectiveWeeklyTarget(targets.Targets.Protein, consumedProtein, daysRemaining, 0.70, 1.30)
		resp["protein"] = map[string]float64{"plain": targets.Targets.Protein, "effective": effective}
	} else {
		resp["protein"] = map[string]float64{"plain": 0, "effective": 0}
	}

	_ = json.NewEncoder(w).Encode(resp)
}

func (h *Handler) handleLogMeal(w http.ResponseWriter, r *http.Request, userID string) {
	var body struct {
		Text string `json:"text"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid JSON body: " + err.Error()})
		return
	}
	if body.Text == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "text field is required"})
		return
	}

	msg := types.InboundMessage{
		UserID: userID,
		Text:   body.Text,
		Kind:   types.MessageText,
	}
	ctx := h.injectModelOverride(r.Context(), userID)
	if err := h.logger.Handle(ctx, msg); err != nil {
		h.writeErr(w, err)
		return
	}
	w.WriteHeader(http.StatusAccepted)
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "accepted"})
}

// ---------------------------------------------------------------------------
// Meals — latest
// ---------------------------------------------------------------------------

func (h *Handler) handleMealsLatest(w http.ResponseWriter, r *http.Request, userID string) {
	latest, err := h.store.LatestMealTime(r.Context(), userID)
	if err != nil && !errors.Is(err, types.ErrNotFound) {
		h.writeErr(w, err)
		return
	}
	_ = json.NewEncoder(w).Encode(map[string]string{"latest": latest})
}
