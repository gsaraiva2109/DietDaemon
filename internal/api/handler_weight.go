package api

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/gsaraiva2109/dietdaemon/core/types"
)

// ---------------------------------------------------------------------------
// Body tracking — weight handlers and cross-domain body summary.
// ---------------------------------------------------------------------------

func (h *Handler) handleListWeight(w http.ResponseWriter, r *http.Request, userID string) {
	days, ok := boundedQueryInt(w, r, "days", 30, 1, 365)
	if !ok {
		return
	}
	entries, err := h.store.ListWeight(r.Context(), userID, days)
	if err != nil {
		h.writeErr(w, err)
		return
	}
	if entries == nil {
		entries = []types.WeightEntry{}
	}
	_ = json.NewEncoder(w).Encode(entries)
}

func (h *Handler) handleLogWeight(w http.ResponseWriter, r *http.Request, userID string) {
	var body struct {
		Date     string  `json:"date"`
		WeightKg float64 `json:"weight_kg"`
		Note     string  `json:"note"`
	}
	if err := decodeRequestJSON(r, &body); err != nil {
		writeValidationError(w, "invalid JSON body")
		return
	}
	if !isFinite(body.WeightKg) || body.WeightKg < 20 || body.WeightKg > 500 {
		writeValidationError(w, "weight_kg must be between 20 and 500")
		return
	}
	if !validDate(body.Date, h.loc) {
		writeValidationError(w, "date must be a non-future YYYY-MM-DD date")
		return
	}
	entry := types.WeightEntry{
		ID:        newHandlerID(),
		UserID:    userID,
		Date:      body.Date,
		WeightKg:  body.WeightKg,
		Note:      body.Note,
		CreatedAt: time.Now().UTC(),
	}
	id, err := h.store.LogWeight(r.Context(), entry)
	if err != nil {
		h.writeErr(w, err)
		return
	}
	// LogWeight upserts by (user_id, date): logging twice the same day
	// overwrites the earlier entry, so the persisted ID may not be the one
	// just generated above — always report back what was actually stored.
	entry.ID = id
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(entry)
}

func (h *Handler) handleWeightTrend(w http.ResponseWriter, r *http.Request, userID string) {
	days, ok := boundedQueryInt(w, r, "days", 30, 1, 365)
	if !ok {
		return
	}
	trend, err := h.store.WeightTrend(r.Context(), userID, days)
	if err != nil {
		h.writeErr(w, err)
		return
	}
	if trend == nil {
		trend = []types.WeightTrend{}
	}
	_ = json.NewEncoder(w).Encode(trend)
}

func (h *Handler) handleDeleteWeight(w http.ResponseWriter, r *http.Request, userID string) {
	entryID := r.PathValue("id")
	if err := h.store.DeleteWeight(r.Context(), userID, entryID); err != nil {
		h.writeErr(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// handleBodySummary reads across weight/fasting/measurements/water/workout/sleep
// domains — it doesn't map cleanly to one sub-domain, so it lives here.
func (h *Handler) handleBodySummary(w http.ResponseWriter, r *http.Request, userID string) {
	// Load all weight entries to compute summary.
	entries, err := h.store.ListWeight(r.Context(), userID, 365)
	if err != nil {
		h.writeErr(w, err)
		return
	}

	summary := types.BodyCompositionSummary{}
	if len(entries) == 0 {
		_ = json.NewEncoder(w).Encode(summary)
		return
	}

	current := entries[len(entries)-1]
	start := entries[0]
	summary.CurrentWeightKg = current.WeightKg
	summary.StartWeightKg = start.WeightKg
	summary.ChangeKg = current.WeightKg - start.WeightKg

	// Compute trend from last 14 days.
	trend, err := h.store.WeightTrend(r.Context(), userID, 14)
	if err == nil && len(trend) > 0 {
		summary.LatestTrendPoint = &trend[len(trend)-1]
		if len(trend) >= 2 {
			first := trend[0].RollingAvg
			last := trend[len(trend)-1].RollingAvg
			diff := last - first
			switch {
			case diff > 0.5:
				summary.TrendDirection = "up"
			case diff < -0.5:
				summary.TrendDirection = "down"
			default:
				summary.TrendDirection = "stable"
			}
		}
	}
	if summary.TrendDirection == "" {
		summary.TrendDirection = "stable"
	}

	_ = json.NewEncoder(w).Encode(summary)
}
