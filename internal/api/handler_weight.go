package api

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/gsaraiva2109/dietdaemon/core/types"
)

// ---------------------------------------------------------------------------
// Body tracking — weight handlers and cross-domain body summary.
// ---------------------------------------------------------------------------

func (h *Handler) handleListWeight(w http.ResponseWriter, r *http.Request, userID string) {
	days := 30
	if s := r.URL.Query().Get("days"); s != "" {
		if n, err := strconv.Atoi(s); err == nil && n > 0 && n <= 365 {
			days = n
		}
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
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid JSON body: " + err.Error()})
		return
	}
	if body.WeightKg <= 0 {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "weight_kg must be positive"})
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
	if err := h.store.LogWeight(r.Context(), entry); err != nil {
		h.writeErr(w, err)
		return
	}
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(entry)
}

func (h *Handler) handleWeightTrend(w http.ResponseWriter, r *http.Request, userID string) {
	days := 30
	if s := r.URL.Query().Get("days"); s != "" {
		if n, err := strconv.Atoi(s); err == nil && n > 0 && n <= 365 {
			days = n
		}
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
