package api

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/gsaraiva2109/dietdaemon/internal/adherence"
)

// ---------------------------------------------------------------------------
// Adherence streak handler.
// ---------------------------------------------------------------------------

// handleStreak returns the user's current adherence streak (consecutive days
// within 90-110% of their calorie target, looking back 180 days).
// GET /api/v1/streak
func (h *Handler) handleStreak(w http.ResponseWriter, r *http.Request, userID string) {
	end := time.Now().AddDate(0, 0, -1).Format("2006-01-02")
	start := time.Now().AddDate(0, 0, -180).Format("2006-01-02")

	rollups, err := h.store.GetRollups(r.Context(), userID, start, end)
	if err != nil {
		h.writeErr(w, err)
		return
	}

	days := adherence.Streak(rollups, 0.90, 1.10)
	_ = json.NewEncoder(w).Encode(map[string]int{"current_days": days})
}
