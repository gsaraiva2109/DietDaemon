package api

import (
	"context"
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

	// Plans holds the user's full diet plan history (transcribed
	// prescriptions), each with its day-types/slots/options tree. This is the
	// one piece of data in the app the user cannot reconstruct from memory,
	// so it must never be silently missing from a "download my data" export.
	Plans []types.PlanBundle `json:"plans"`
}

func (h *Handler) handleExportAll(w http.ResponseWriter, r *http.Request, userID string) {
	ctx := r.Context()

	user, profile, err := h.exportUserAndProfile(ctx, userID)
	if err != nil {
		h.writeErr(w, err)
		return
	}

	logs, err := h.exportLogData(ctx, userID)
	if err != nil {
		h.writeErr(w, err)
		return
	}

	photos, err := h.exportPhotos(ctx, userID)
	if err != nil {
		h.writeErr(w, err)
		return
	}

	plans, err := h.exportPlans(ctx, userID)
	if err != nil {
		h.writeErr(w, err)
		return
	}

	export := UserDataExport{
		ExportedAt:       time.Now().UTC(),
		User:             user,
		Profile:          profile,
		Meals:            logs.Meals,
		Rollups:          logs.Rollups,
		Weight:           logs.Weight,
		Measurements:     logs.Measurements,
		Sleep:            logs.Sleep,
		Workouts:         logs.Workouts,
		Fasts:            logs.Fasts,
		WaterDailyTotals: logs.WaterTotals,
		Photos:           photos,
		Templates:        logs.Templates,
		Plans:            plans,
	}

	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=dietdaemon-export-%s.json", userID))
	_ = json.NewEncoder(w).Encode(export)
}

// exportUserAndProfile fetches the exporting user's account and profile. A
// missing profile isn't an error here (an un-onboarded user has none yet) —
// it's reported back as a zero-value UserProfile, same as before this was
// split out of handleExportAll.
func (h *Handler) exportUserAndProfile(ctx context.Context, userID string) (types.User, types.UserProfile, error) {
	user, err := h.store.GetUser(ctx, userID)
	if err != nil {
		if errors.Is(err, types.ErrNotFound) {
			return types.User{}, types.UserProfile{}, fmt.Errorf("export authenticated user missing: %v", err)
		}
		return types.User{}, types.UserProfile{}, err
	}

	profile, err := h.store.GetProfile(ctx, userID)
	if err != nil && !errors.Is(err, types.ErrNotFound) {
		return types.User{}, types.UserProfile{}, err
	}
	if errors.Is(err, types.ErrNotFound) {
		profile = types.UserProfile{UserID: userID, Onboarded: false}
	}
	return user, profile, nil
}

// exportLogData bundles the export's range/limit-scoped store lookups: every
// piece of UserDataExport except the user, profile, and photos.
type exportLogData struct {
	Meals        []types.Meal
	Rollups      []types.DailyRollup
	Weight       []types.WeightEntry
	Measurements []types.MeasurementEntry
	Sleep        []types.SleepLog
	Workouts     []types.Workout
	Fasts        []types.Fast
	WaterTotals  []types.WaterDayTotal
	Templates    []types.MealTemplate
}

func (h *Handler) exportLogData(ctx context.Context, userID string) (exportLogData, error) {
	var d exportLogData
	var err error

	if d.Meals, err = h.store.GetMealsInRange(ctx, userID, exportAllStart, exportAllEnd); err != nil {
		return exportLogData{}, err
	}
	if d.Rollups, err = h.store.GetRollups(ctx, userID, exportAllStart, exportAllEnd); err != nil {
		return exportLogData{}, err
	}
	if d.Weight, err = h.store.ListWeight(ctx, userID, exportAllLimit); err != nil {
		return exportLogData{}, err
	}
	if d.Measurements, err = h.store.ListMeasurements(ctx, userID, exportAllLimit); err != nil {
		return exportLogData{}, err
	}
	if d.Sleep, err = h.store.ListSleep(ctx, userID, exportAllLimit); err != nil {
		return exportLogData{}, err
	}
	if d.Workouts, err = h.store.ListWorkouts(ctx, userID, exportAllLimit); err != nil {
		return exportLogData{}, err
	}
	if d.Fasts, err = h.store.ListFasts(ctx, userID, exportAllLimit); err != nil {
		return exportLogData{}, err
	}
	if d.WaterTotals, err = h.store.GetWaterDailyTotals(ctx, userID, exportAllStart, exportAllEnd); err != nil {
		return exportLogData{}, err
	}
	if d.Templates, err = h.store.GetTemplates(ctx, userID); err != nil {
		return exportLogData{}, err
	}

	d.normalizeNilSlices()
	return d, nil
}

// normalizeNilSlices replaces any nil slice with an empty one so the export
// JSON always has "[]" instead of "null" for a user with no history in that
// category.
func (d *exportLogData) normalizeNilSlices() {
	if d.Meals == nil {
		d.Meals = []types.Meal{}
	}
	if d.Rollups == nil {
		d.Rollups = []types.DailyRollup{}
	}
	if d.Weight == nil {
		d.Weight = []types.WeightEntry{}
	}
	if d.Measurements == nil {
		d.Measurements = []types.MeasurementEntry{}
	}
	if d.Sleep == nil {
		d.Sleep = []types.SleepLog{}
	}
	if d.Workouts == nil {
		d.Workouts = []types.Workout{}
	}
	if d.Fasts == nil {
		d.Fasts = []types.Fast{}
	}
	if d.WaterTotals == nil {
		d.WaterTotals = []types.WaterDayTotal{}
	}
	if d.Templates == nil {
		d.Templates = []types.MealTemplate{}
	}
}

// exportPhotos fetches every progress photo's full data (metadata lookup
// returns thumbnails/refs only; each photo's bytes need a separate fetch).
func (h *Handler) exportPhotos(ctx context.Context, userID string) ([]types.ProgressPhoto, error) {
	photoMeta, err := h.store.ListPhotoMetadata(ctx, userID)
	if err != nil {
		return nil, err
	}
	photos := make([]types.ProgressPhoto, 0, len(photoMeta))
	for _, meta := range photoMeta {
		full, err := h.store.GetPhotoData(ctx, userID, meta.ID)
		if err != nil {
			return nil, err
		}
		photos = append(photos, full)
	}
	return photos, nil
}

// exportPlans fetches every diet plan the user has ever had, each with its
// full day-type/slot/option tree via GetPlanBundle (already the N+1-safe way
// to load one plan; a user has at most a handful of plans in their history).
func (h *Handler) exportPlans(ctx context.Context, userID string) ([]types.PlanBundle, error) {
	list, err := h.store.ListPlans(ctx, userID)
	if err != nil {
		return nil, err
	}
	plans := make([]types.PlanBundle, 0, len(list))
	for _, p := range list {
		bundle, err := h.store.GetPlanBundle(ctx, p.ID)
		if err != nil {
			return nil, err
		}
		plans = append(plans, bundle)
	}
	return plans, nil
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
