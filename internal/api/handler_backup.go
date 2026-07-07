package api

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/gsaraiva2109/dietdaemon/core/types"
)

// ---------------------------------------------------------------------------
// Scheduled backup settings handlers.
// ---------------------------------------------------------------------------

const defaultBackupIntervalHrs = 24

func (h *Handler) handleGetBackupConfig(w http.ResponseWriter, r *http.Request, userID string) {
	cfg, err := h.store.GetBackupConfig(r.Context(), userID)
	if errors.Is(err, types.ErrNotFound) {
		// No config yet: report sensible defaults, disabled.
		cfg = types.BackupConfig{UserID: userID, Destination: "local", IntervalHrs: defaultBackupIntervalHrs}
	} else if err != nil {
		h.writeErr(w, err)
		return
	}
	_ = json.NewEncoder(w).Encode(cfg)
}

func (h *Handler) handleSetBackupConfig(w http.ResponseWriter, r *http.Request, userID string) {
	var body types.BackupConfig
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid JSON body: " + err.Error()})
		return
	}
	body.UserID = userID
	if body.Destination == "" {
		body.Destination = "local"
	}
	if body.IntervalHrs <= 0 {
		body.IntervalHrs = defaultBackupIntervalHrs
	}
	if err := h.store.SetBackupConfig(r.Context(), body); err != nil {
		h.writeErr(w, err)
		return
	}
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(body)
}

// handleRunBackupNow triggers an immediate backup for the authenticated user,
// reusing the same export logic the scheduled ticker uses (backup.Runner.RunOnce).
func (h *Handler) handleRunBackupNow(w http.ResponseWriter, r *http.Request, userID string) {
	if h.backupRunner == nil {
		w.WriteHeader(http.StatusServiceUnavailable)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "backup is not enabled on this server"})
		return
	}
	if err := h.backupRunner.RunOnce(r.Context(), userID); err != nil {
		h.writeErr(w, err)
		return
	}
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}
