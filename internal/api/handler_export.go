package api

import (
	"encoding/json"
	"net/http"

	"github.com/gsaraiva2109/dietdaemon/core/types"
	"github.com/gsaraiva2109/dietdaemon/internal/exportfmt"
)

// ---------------------------------------------------------------------------
// Data export handlers -- meals and rollups in JSON or CSV.
// ---------------------------------------------------------------------------

func (h *Handler) handleExportMeals(w http.ResponseWriter, r *http.Request, userID string) {
	format := r.URL.Query().Get("format")
	start := r.URL.Query().Get("start")
	end := r.URL.Query().Get("end")

	if start == "" || end == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "start and end query params required (YYYY-MM-DD)"})
		return
	}

	meals, err := h.store.GetMealsInRange(r.Context(), userID, start, end)
	if err != nil {
		h.writeErr(w, err)
		return
	}

	switch format {
	case "csv":
		w.Header().Set("Content-Type", "text/csv")
		w.Header().Set("Content-Disposition", "attachment; filename=meals.csv")
		_ = exportfmt.WriteMealsCSV(w, meals)
	default:
		// JSON (default).
		w.Header().Set("Content-Disposition", "attachment; filename=meals.json")
		if meals == nil {
			meals = []types.Meal{}
		}
		_ = json.NewEncoder(w).Encode(meals)
	}
}

func (h *Handler) handleExportRollups(w http.ResponseWriter, r *http.Request, userID string) {
	format := r.URL.Query().Get("format")
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

	switch format {
	case "csv":
		w.Header().Set("Content-Type", "text/csv")
		w.Header().Set("Content-Disposition", "attachment; filename=rollups.csv")
		_ = exportfmt.WriteRollupsCSV(w, rollups)
	default:
		w.Header().Set("Content-Disposition", "attachment; filename=rollups.json")
		if rollups == nil {
			rollups = []types.DailyRollup{}
		}
		_ = json.NewEncoder(w).Encode(rollups)
	}
}
