package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/gsaraiva2109/dietdaemon/core/types"
	"github.com/gsaraiva2109/dietdaemon/internal/auth"
)

// ---------------------------------------------------------------------------
// Account data export (full personal-data download) and account deletion.
// ---------------------------------------------------------------------------

// exportAllStart, exportAllEnd, and exportAllLimit are wide-open bounds used
// to fetch "everything, ever" from the range/limit-scoped store methods
// below. No store method exists for an unbounded query, so we just pass
// bounds wide enough to always cover a real user's history instead of adding
// one.
const (
	exportAllStart = "0001-01-01"
	exportAllEnd   = "9999-12-31"
	exportAllLimit = 1_000_000
)

// UserDataExport bundles every piece of a user's personal data into one
// downloadable JSON document (GDPR/CCPA-style "download my data").
type UserDataExport struct {
	ExportedAt time.Time         `json:"exported_at"`
	User       types.User        `json:"user"`
	Profile    types.UserProfile `json:"profile"`

	Meals   []types.Meal        `json:"meals"`
	Rollups []types.DailyRollup `json:"rollups"`

	Weight       []types.WeightEntry      `json:"weight"`
	Measurements []types.MeasurementEntry `json:"measurements"`
	Sleep        []types.SleepLog         `json:"sleep"`
	Workouts     []types.Workout          `json:"workouts"`
	Fasts        []types.Fast             `json:"fasts"`

	// WaterDailyTotals holds only per-day aggregated totals, not raw log
	// entries: the store has no ranged raw-list method for water logs, only
	// GetWaterDailyTotals. Add a raw export here if that method ever lands.
	WaterDailyTotals []types.WaterDayTotal `json:"water_daily_totals"`

	Photos    []types.ProgressPhoto `json:"photos"`
	Templates []types.MealTemplate  `json:"templates"`
}

func (h *Handler) handleExportAll(w http.ResponseWriter, r *http.Request, userID string) {
	ctx := r.Context()

	user, err := h.store.GetUser(ctx, userID)
	if err != nil {
		if errors.Is(err, types.ErrNotFound) {
			h.writeErr(w, fmt.Errorf("export authenticated user missing: %v", err))
			return
		}
		h.writeErr(w, err)
		return
	}

	profile, err := h.store.GetProfile(ctx, userID)
	if err != nil && !errors.Is(err, types.ErrNotFound) {
		h.writeErr(w, err)
		return
	}
	if errors.Is(err, types.ErrNotFound) {
		profile = types.UserProfile{UserID: userID, Onboarded: false}
	}

	meals, err := h.store.GetMealsInRange(ctx, userID, exportAllStart, exportAllEnd)
	if err != nil {
		h.writeErr(w, err)
		return
	}
	rollups, err := h.store.GetRollups(ctx, userID, exportAllStart, exportAllEnd)
	if err != nil {
		h.writeErr(w, err)
		return
	}
	weight, err := h.store.ListWeight(ctx, userID, exportAllLimit)
	if err != nil {
		h.writeErr(w, err)
		return
	}
	measurements, err := h.store.ListMeasurements(ctx, userID, exportAllLimit)
	if err != nil {
		h.writeErr(w, err)
		return
	}
	sleep, err := h.store.ListSleep(ctx, userID, exportAllLimit)
	if err != nil {
		h.writeErr(w, err)
		return
	}
	workouts, err := h.store.ListWorkouts(ctx, userID, exportAllLimit)
	if err != nil {
		h.writeErr(w, err)
		return
	}
	fasts, err := h.store.ListFasts(ctx, userID, exportAllLimit)
	if err != nil {
		h.writeErr(w, err)
		return
	}
	waterTotals, err := h.store.GetWaterDailyTotals(ctx, userID, exportAllStart, exportAllEnd)
	if err != nil {
		h.writeErr(w, err)
		return
	}
	templates, err := h.store.GetTemplates(ctx, userID)
	if err != nil {
		h.writeErr(w, err)
		return
	}

	photoMeta, err := h.store.ListPhotoMetadata(ctx, userID)
	if err != nil {
		h.writeErr(w, err)
		return
	}
	photos := make([]types.ProgressPhoto, 0, len(photoMeta))
	for _, meta := range photoMeta {
		full, err := h.store.GetPhotoData(ctx, meta.ID)
		if err != nil {
			h.writeErr(w, err)
			return
		}
		photos = append(photos, full)
	}

	if meals == nil {
		meals = []types.Meal{}
	}
	if rollups == nil {
		rollups = []types.DailyRollup{}
	}
	if weight == nil {
		weight = []types.WeightEntry{}
	}
	if measurements == nil {
		measurements = []types.MeasurementEntry{}
	}
	if sleep == nil {
		sleep = []types.SleepLog{}
	}
	if workouts == nil {
		workouts = []types.Workout{}
	}
	if fasts == nil {
		fasts = []types.Fast{}
	}
	if waterTotals == nil {
		waterTotals = []types.WaterDayTotal{}
	}
	if templates == nil {
		templates = []types.MealTemplate{}
	}

	export := UserDataExport{
		ExportedAt:       time.Now().UTC(),
		User:             user,
		Profile:          profile,
		Meals:            meals,
		Rollups:          rollups,
		Weight:           weight,
		Measurements:     measurements,
		Sleep:            sleep,
		Workouts:         workouts,
		Fasts:            fasts,
		WaterDailyTotals: waterTotals,
		Photos:           photos,
		Templates:        templates,
	}

	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=dietdaemon-export-%s.json", userID))
	_ = json.NewEncoder(w).Encode(export)
}

// deleteAccountRequest is the safety-guard body for account deletion: the
// client must echo back the literal string "DELETE" to confirm intent, so a
// stray or CSRF-forged DELETE request can't wipe an account by accident.
type deleteAccountRequest struct {
	Confirm string `json:"confirm"`
}

func (h *Handler) handleDeleteAccount(w http.ResponseWriter, r *http.Request, userID string) {
	var body deleteAccountRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Confirm != "DELETE" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": `confirm must be the literal string "DELETE"`})
		return
	}

	if err := h.authStore.DeleteAccount(r.Context(), userID); err != nil {
		h.writeErr(w, err)
		return
	}

	// Best-effort: the account (and its sessions row, via cascade) is already
	// gone, but also drop the caller's own session cache entry and cookies so
	// this response doesn't leave a stale authenticated cookie behind.
	if c, err := r.Cookie("dd_session"); err == nil && c.Value != "" {
		_ = h.sessions.DeleteSession(r.Context(), auth.HashToken(c.Value))
	}
	h.clearSessionCookies(w)

	w.WriteHeader(http.StatusNoContent)
}
