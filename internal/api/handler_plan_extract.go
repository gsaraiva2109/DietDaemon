package api

import (
	"encoding/json"
	"io"
	"net/http"

	"github.com/gsaraiva2109/dietdaemon/adapters/model/planextract"
)

// ---------------------------------------------------------------------------
// Diet-plan import, paste-text path (issue #193) — extracts a draft from
// pasted plan text for the user to review and correct before anything saves
// via the existing plan-creation endpoints. The pasted text is never
// persisted or passed to h.store: it is read into memory, handed to the
// completion adapter, and discarded once this handler returns.
// ---------------------------------------------------------------------------

// maxPlanTextChars bounds the pasted-text body accepted by
// handleExtractPlanFromText, checked before the text goes into a prompt.
const maxPlanTextChars = 20_000

const planExtractSeparator = "\n\n---\nPasted plan text follows:\n\n"

func (h *Handler) handleExtractPlanFromText(w http.ResponseWriter, r *http.Request, userID string) {
	data, err := io.ReadAll(io.LimitReader(r.Body, maxPlanTextChars+1))
	if err != nil {
		h.writeErr(w, err)
		return
	}
	if len(data) == 0 {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "pasted text is required"})
		return
	}
	if len(data) > maxPlanTextChars {
		w.WriteHeader(http.StatusRequestEntityTooLarge)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "pasted text too large (max 20,000 characters)"})
		return
	}

	ctx, err := h.injectModelOverride(r.Context(), userID)
	if err != nil {
		h.writeErr(w, err)
		return
	}

	prompt := planextract.Prompt + planExtractSeparator + string(data)
	raw, err := h.completionAdapter.Complete(ctx, prompt)
	if err != nil {
		h.writeErr(w, err)
		return
	}

	draft, err := planextract.ParseResponse(raw)
	if err != nil {
		h.writeErr(w, err)
		return
	}
	_ = json.NewEncoder(w).Encode(draft)
}
