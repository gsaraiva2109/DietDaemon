package api

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/gsaraiva2109/dietdaemon/core/types"
)

// ---------------------------------------------------------------------------
// Body tracking — measurement handlers.
// ---------------------------------------------------------------------------

func (h *Handler) handleListMeasurements(w http.ResponseWriter, r *http.Request, userID string) {
	days, ok := boundedQueryInt(w, r, "days", 30, 1, 365)
	if !ok {
		return
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
	var body struct {
		Date         string   `json:"date"`
		WaistCm      *float64 `json:"waist_cm"`
		HipsCm       *float64 `json:"hips_cm"`
		ChestCm      *float64 `json:"chest_cm"`
		LeftArmCm    *float64 `json:"left_arm_cm"`
		RightArmCm   *float64 `json:"right_arm_cm"`
		LeftThighCm  *float64 `json:"left_thigh_cm"`
		RightThighCm *float64 `json:"right_thigh_cm"`
		Note         string   `json:"note"`
	}
	if err := decodeRequestJSON(r, &body); err != nil {
		writeValidationError(w, "invalid JSON body")
		return
	}
	if !validDate(body.Date, h.loc) {
		writeValidationError(w, "date must be a non-future YYYY-MM-DD date")
		return
	}
	values := []*float64{body.WaistCm, body.HipsCm, body.ChestCm, body.LeftArmCm, body.RightArmCm, body.LeftThighCm, body.RightThighCm}
	if !validMeasurements(values) {
		writeValidationError(w, "at least one finite non-negative measurement is required")
		return
	}
	entry := types.MeasurementEntry{
		ID: newHandlerID(), UserID: userID, Date: body.Date, Note: body.Note, CreatedAt: time.Now().UTC(),
		WaistCm: valueOrZero(body.WaistCm), HipsCm: valueOrZero(body.HipsCm), ChestCm: valueOrZero(body.ChestCm),
		LeftArmCm: valueOrZero(body.LeftArmCm), RightArmCm: valueOrZero(body.RightArmCm),
		LeftThighCm: valueOrZero(body.LeftThighCm), RightThighCm: valueOrZero(body.RightThighCm),
	}
	id, err := h.store.LogMeasurement(r.Context(), entry)
	if err != nil {
		h.writeErr(w, err)
		return
	}
	// LogMeasurement upserts by (user_id, date): logging twice the same day
	// overwrites the earlier entry, so the persisted ID may not be the one
	// just generated above — always report back what was actually stored.
	entry.ID = id
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(entry)
}

func validMeasurements(values []*float64) bool {
	found := false
	for _, value := range values {
		if value == nil {
			continue
		}
		found = true
		if !isFinite(*value) || *value < 0 {
			return false
		}
	}
	return found
}

func valueOrZero(value *float64) float64 {
	if value == nil {
		return 0
	}
	return *value
}

func (h *Handler) handleDeleteMeasurement(w http.ResponseWriter, r *http.Request, userID string) {
	entryID := r.PathValue("id")
	if err := h.store.DeleteMeasurement(r.Context(), userID, entryID); err != nil {
		h.writeErr(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
