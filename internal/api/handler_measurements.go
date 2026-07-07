package api

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/gsaraiva2109/dietdaemon/core/types"
)

// ---------------------------------------------------------------------------
// Body tracking — measurement handlers.
// ---------------------------------------------------------------------------

func (h *Handler) handleListMeasurements(w http.ResponseWriter, r *http.Request, userID string) {
	days := 30
	if s := r.URL.Query().Get("days"); s != "" {
		if n, err := strconv.Atoi(s); err == nil && n > 0 && n <= 365 {
			days = n
		}
	}
	entries, err := h.store.ListMeasurements(r.Context(), userID, days)
	if err != nil {
		h.writeErr(w, err)
		return
	}
	if entries == nil {
		entries = []types.MeasurementEntry{}
	}
	_ = json.NewEncoder(w).Encode(entries)
}

func (h *Handler) handleLogMeasurements(w http.ResponseWriter, r *http.Request, userID string) {
	var body types.MeasurementEntry
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid JSON body: " + err.Error()})
		return
	}
	body.ID = newHandlerID()
	body.UserID = userID
	body.CreatedAt = time.Now().UTC()
	if err := h.store.LogMeasurement(r.Context(), body); err != nil {
		h.writeErr(w, err)
		return
	}
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(body)
}

func (h *Handler) handleDeleteMeasurement(w http.ResponseWriter, r *http.Request, userID string) {
	entryID := r.PathValue("id")
	if err := h.store.DeleteMeasurement(r.Context(), userID, entryID); err != nil {
		h.writeErr(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
