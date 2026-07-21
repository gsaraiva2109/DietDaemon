package api

import (
	"encoding/json"
	"net/http"
)

// adminFoodImportRequest is the shared request body for the run/repair admin
// food-import endpoints. max_rows is optional and only meaningful for run
// (0 or omitted means "use the source's configured default").
type adminFoodImportRequest struct {
	Source  string `json:"source"`
	MaxRows int    `json:"max_rows"`
}

// handleAdminFoodImportRun triggers a bulk import of one nutrition source
// into the global food catalog, reusing the same BuildSource/FetchBulk/
// BulkUpsertFoods path cmd/import-foods uses, without needing shell/volume
// access to the running daemon's DB (issue #136).
func (h *Handler) handleAdminFoodImportRun(w http.ResponseWriter, r *http.Request) {
	if h.foodImportRunner == nil {
		w.WriteHeader(http.StatusServiceUnavailable)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "food import is not enabled on this server"})
		return
	}
	var req adminFoodImportRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, http.StatusBadRequest, ErrorValidation, "Invalid request body.")
		return
	}
	if req.Source == "" {
		WriteError(w, http.StatusBadRequest, ErrorValidation, "source is required.")
		return
	}

	rows, err := h.foodImportRunner.ImportSource(r.Context(), req.Source, req.MaxRows)
	if err != nil {
		h.writeErr(w, err)
		return
	}
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]int{"rows": rows})
}

// handleAdminFoodImportRepair re-fetches one nutrition source and overwrites
// macros on existing catalog rows matched by (source, name), fixing rows an
// older/different importer wrote under a different food_id scheme (see
// issue #111 for the underlying repair logic this exposes over HTTP).
func (h *Handler) handleAdminFoodImportRepair(w http.ResponseWriter, r *http.Request) {
	if h.foodImportRunner == nil {
		w.WriteHeader(http.StatusServiceUnavailable)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "food import is not enabled on this server"})
		return
	}
	var req adminFoodImportRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, http.StatusBadRequest, ErrorValidation, "Invalid request body.")
		return
	}
	if req.Source == "" {
		WriteError(w, http.StatusBadRequest, ErrorValidation, "source is required.")
		return
	}

	checked, fixed, err := h.foodImportRunner.RepairSource(r.Context(), req.Source)
	if err != nil {
		h.writeErr(w, err)
		return
	}
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]int{"checked": checked, "fixed": fixed})
}

// handleAdminFoodImportBackfillEmbeddings embeds every catalog food missing a
// vector, against a live Ollama endpoint. Takes no source: it's a standalone
// maintenance pass over whatever the DB already holds.
func (h *Handler) handleAdminFoodImportBackfillEmbeddings(w http.ResponseWriter, r *http.Request) {
	if h.foodImportRunner == nil {
		w.WriteHeader(http.StatusServiceUnavailable)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "food import is not enabled on this server"})
		return
	}

	embedded, failed, err := h.foodImportRunner.BackfillEmbeddings(r.Context())
	if err != nil {
		h.writeErr(w, err)
		return
	}
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]int{"embedded": embedded, "failed": failed})
}
