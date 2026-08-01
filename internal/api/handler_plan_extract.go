package api

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/gsaraiva2109/dietdaemon/adapters/model/planextract"
	"github.com/gsaraiva2109/dietdaemon/core/types"
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

// ---------------------------------------------------------------------------
// Diet-plan import, photo/PDF path (issue #194) — extracts a draft from a
// photographed or scanned plan page for the user to review and correct
// before anything saves via the existing plan-creation endpoints. The
// uploaded image is never persisted or passed to h.store: it is read into
// memory, handed to the vision adapter, and discarded once this handler
// returns.
// ---------------------------------------------------------------------------

// maxPlanPages and maxPlanTotalBytes bound a multi-page plan photo upload:
// at most 10 pages, 25 MB combined, checked before any page reaches the
// vision adapter.
const (
	maxPlanPages      = 10
	maxPlanTotalBytes = 25 << 20
)

func (h *Handler) handleExtractPlanFromImage(w http.ResponseWriter, r *http.Request, userID string) {
	if h.visionAdapter == nil {
		w.WriteHeader(http.StatusNotImplemented)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "plan photo extraction is not configured on this server"})
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, maxPlanTotalBytes)
	// #nosec G120 — MaxBytesReader above bounds the body before ParseMultipartForm.
	if err := r.ParseMultipartForm(maxPlanTotalBytes); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "upload too large (max 25 MB combined)"})
		return
	}

	fileHeaders := r.MultipartForm.File["file"]
	if len(fileHeaders) == 0 {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "file field required"})
		return
	}
	if len(fileHeaders) > maxPlanPages {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": fmt.Sprintf("too many pages (max %d)", maxPlanPages)})
		return
	}

	pages := make([]types.PlanImagePage, 0, len(fileHeaders))
	totalBytes := 0
	for i, fh := range fileHeaders {
		file, err := fh.Open()
		if err != nil {
			h.writeErr(w, err)
			return
		}
		data, err := io.ReadAll(io.LimitReader(file, maxPlanTotalBytes+1))
		_ = file.Close()
		if err != nil {
			h.writeErr(w, err)
			return
		}

		totalBytes += len(data)
		if totalBytes > maxPlanTotalBytes {
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "upload too large (max 25 MB combined)"})
			return
		}

		mimeType := http.DetectContentType(data)
		if len(mimeType) < 6 || mimeType[:6] != "image/" {
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": fmt.Sprintf("page %d is not an image", i+1)})
			return
		}

		pages = append(pages, types.PlanImagePage{Data: data, MimeType: mimeType})
	}

	draft, err := h.visionAdapter.ExtractPlan(r.Context(), pages)
	if err != nil {
		h.writeErr(w, err)
		return
	}
	_ = json.NewEncoder(w).Encode(draft)
}
