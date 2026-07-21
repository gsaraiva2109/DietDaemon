package api

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/gsaraiva2109/dietdaemon/core/types"
)

// ---------------------------------------------------------------------------
// Sleep tracking handlers.
// ---------------------------------------------------------------------------

// sleepDurationHours computes the duration of a sleep log in hours.
// If wakeAt is nil (active sleep), duration is from sleepAt to now.
func sleepDurationHours(sleepAt string, wakeAt *string) float64 {
	start, err := time.Parse(time.RFC3339, sleepAt)
	if err != nil {
		return 0
	}
	var end time.Time
	if wakeAt != nil {
		end, err = time.Parse(time.RFC3339, *wakeAt)
		if err != nil {
			return 0
		}
	} else {
		end = time.Now().UTC()
	}
	return end.Sub(start).Hours()
}

func (h *Handler) handleLogSleep(w http.ResponseWriter, r *http.Request, userID string) {
	var body struct {
		SleepAt string  `json:"sleep_at"`
		WakeAt  *string `json:"wake_at,omitempty"`
		Quality string  `json:"quality"`
		Note    string  `json:"note,omitempty"`
	}
	if err := decodeRequestJSON(r, &body); err != nil {
		writeValidationError(w, "invalid JSON body")
		return
	}
	if body.SleepAt == "" {
		body.SleepAt = time.Now().UTC().Format(time.RFC3339)
	}
	if body.Quality == "" {
		body.Quality = "ok"
	}
	if !validSleepQuality(body.Quality) {
		writeValidationError(w, "quality is invalid")
		return
	}
	entry := types.SleepLog{
		ID:            newHandlerID(),
		UserID:        userID,
		SleepAt:       body.SleepAt,
		WakeAt:        body.WakeAt,
		DurationHours: sleepDurationHours(body.SleepAt, body.WakeAt),
		Quality:       body.Quality,
		Note:          body.Note,
	}
	if err := h.store.LogSleep(r.Context(), entry); err != nil {
		h.writeErr(w, err)
		return
	}
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(entry)
}

func (h *Handler) handleListSleep(w http.ResponseWriter, r *http.Request, userID string) {
	limit, ok := boundedQueryInt(w, r, "limit", 10, 1, 100)
	if !ok {
		return
	}
	sleeps, err := h.store.ListSleep(r.Context(), userID, limit)
	if err != nil {
		h.writeErr(w, err)
		return
	}
	if sleeps == nil {
		sleeps = []types.SleepLog{}
	}
	for i := range sleeps {
		sleeps[i].DurationHours = sleepDurationHours(sleeps[i].SleepAt, sleeps[i].WakeAt)
	}
	_ = json.NewEncoder(w).Encode(sleeps)
}

func (h *Handler) handleGetActiveSleep(w http.ResponseWriter, r *http.Request, userID string) {
	sleep, err := h.store.GetActiveSleep(r.Context(), userID)
	if err != nil {
		h.writeErr(w, err)
		return
	}
	sleep.DurationHours = sleepDurationHours(sleep.SleepAt, sleep.WakeAt)
	_ = json.NewEncoder(w).Encode(sleep)
}

func (h *Handler) handleEndSleep(w http.ResponseWriter, r *http.Request, userID string) {
	id := r.PathValue("id")
	var body struct {
		WakeAt  string `json:"wake_at"`
		Quality string `json:"quality"`
	}
	if err := decodeRequestJSON(r, &body); err != nil {
		writeValidationError(w, "invalid JSON body")
		return
	}
	if body.WakeAt == "" {
		body.WakeAt = time.Now().UTC().Format(time.RFC3339)
	}
	if body.Quality == "" {
		body.Quality = "ok"
	}
	if !validSleepQuality(body.Quality) {
		writeValidationError(w, "quality is invalid")
		return
	}
	if err := h.store.EndSleep(r.Context(), userID, id, body.WakeAt, body.Quality); err != nil {
		h.writeErr(w, err)
		return
	}
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "ended"})
}

func (h *Handler) handleDeleteSleep(w http.ResponseWriter, r *http.Request, userID string) {
	id := r.PathValue("id")
	if err := h.store.DeleteSleep(r.Context(), userID, id); err != nil {
		h.writeErr(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
