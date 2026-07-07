package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/gsaraiva2109/dietdaemon/core/types"
)

// ---------------------------------------------------------------------------
// Fasting handlers -- start, end, active, list.
// ---------------------------------------------------------------------------

func (h *Handler) handleStartFast(w http.ResponseWriter, r *http.Request, userID string) {
	var body struct {
		TargetHours float64 `json:"target_hours"`
	}
	// Body is optional; ignore decode errors on empty bodies.
	_ = json.NewDecoder(r.Body).Decode(&body)
	if body.TargetHours <= 0 {
		body.TargetHours = 16
	}

	// Reject if a fast is already in progress.
	if _, err := h.store.GetActiveFast(r.Context(), userID); err == nil {
		w.WriteHeader(http.StatusConflict)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "a fast is already in progress"})
		return
	} else if !errors.Is(err, types.ErrNotFound) {
		h.writeErr(w, err)
		return
	}

	now := time.Now().UTC()
	fast := types.Fast{
		ID:          newHandlerID(),
		UserID:      userID,
		StartAt:     now,
		TargetHours: body.TargetHours,
		CreatedAt:   now,
	}
	if err := h.store.StartFast(r.Context(), fast); err != nil {
		h.writeErr(w, err)
		return
	}
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(fast)
}

func (h *Handler) handleEndFast(w http.ResponseWriter, r *http.Request, userID string) {
	active, err := h.store.GetActiveFast(r.Context(), userID)
	if err != nil {
		h.writeErr(w, err) // ErrNotFound → 404 when no active fast.
		return
	}
	end := time.Now().UTC()
	completed := end.Sub(active.StartAt).Hours() >= active.TargetHours
	updated, err := h.store.EndFast(r.Context(), userID, active.ID, end, completed)
	if err != nil {
		h.writeErr(w, err)
		return
	}
	_ = json.NewEncoder(w).Encode(updated)
}

func (h *Handler) handleGetActiveFast(w http.ResponseWriter, r *http.Request, userID string) {
	active, err := h.store.GetActiveFast(r.Context(), userID)
	if err != nil {
		h.writeErr(w, err) // ErrNotFound → 404; frontend treats as "no active fast".
		return
	}
	_ = json.NewEncoder(w).Encode(active)
}

func (h *Handler) handleListFasts(w http.ResponseWriter, r *http.Request, userID string) {
	limit := 10
	if s := r.URL.Query().Get("limit"); s != "" {
		if n, err := strconv.Atoi(s); err == nil && n > 0 && n <= 100 {
			limit = n
		}
	}
	fasts, err := h.store.ListFasts(r.Context(), userID, limit)
	if err != nil {
		h.writeErr(w, err)
		return
	}
	if fasts == nil {
		fasts = []types.Fast{}
	}
	_ = json.NewEncoder(w).Encode(fasts)
}
