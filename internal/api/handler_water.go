package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/gsaraiva2109/dietdaemon/core/types"
)

// ---------------------------------------------------------------------------
// Water tracking handlers.
// ---------------------------------------------------------------------------

// defaultWaterGoalMl is used when the user has no stored targets row yet, or
// the row predates the water_goal_ml column (zero value).
const defaultWaterGoalMl = 2000

func (h *Handler) handleLogWater(w http.ResponseWriter, r *http.Request, userID string) {
	var body struct {
		AmountML int    `json:"amount_ml"`
		Note     string `json:"note,omitempty"`
		LoggedAt string `json:"logged_at"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid JSON body: " + err.Error()})
		return
	}
	if body.AmountML <= 0 {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "amountMl must be positive"})
		return
	}
	if body.LoggedAt == "" {
		body.LoggedAt = time.Now().UTC().Format(time.RFC3339)
	}
	entry := types.WaterLog{
		ID:       newHandlerID(),
		UserID:   userID,
		AmountML: body.AmountML,
		Note:     body.Note,
		LoggedAt: body.LoggedAt,
	}
	if err := h.store.LogWater(r.Context(), entry); err != nil {
		h.writeErr(w, err)
		return
	}
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(entry)
}

func (h *Handler) handleGetWaterToday(w http.ResponseWriter, r *http.Request, userID string) {
	today := time.Now().In(h.loc).Format("2006-01-02")
	logs, total, err := h.store.GetWaterToday(r.Context(), userID, today)
	if err != nil {
		h.writeErr(w, err)
		return
	}
	if logs == nil {
		logs = []types.WaterLog{}
	}
	goalMl := defaultWaterGoalMl
	dt, err := h.store.GetTargets(r.Context(), userID)
	if err != nil && !errors.Is(err, types.ErrNotFound) {
		h.writeErr(w, err)
		return
	}
	if err == nil && dt.WaterGoalMl > 0 {
		goalMl = dt.WaterGoalMl
	}
	_ = json.NewEncoder(w).Encode(map[string]any{
		"logs":     logs,
		"today_ml": total,
		"goal_ml":  goalMl,
	})
}

func (h *Handler) handleDeleteWater(w http.ResponseWriter, r *http.Request, userID string) {
	id := r.PathValue("id")
	if err := h.store.DeleteWater(r.Context(), userID, id); err != nil {
		h.writeErr(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
