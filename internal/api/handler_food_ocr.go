package api

import (
	"encoding/json"
	"io"
	"net/http"
)

// ---------------------------------------------------------------------------
// OCR-assisted nutrition-label capture (issue #87) — extracts a draft from a
// photographed label for the user to review and explicitly save via the
// existing custom-food endpoints. The uploaded image is never persisted or
// passed to h.store: it is read into memory, handed to the vision adapter,
// and discarded once this handler returns.
// ---------------------------------------------------------------------------

func (h *Handler) handleOCRExtractCustomFood(w http.ResponseWriter, r *http.Request, userID string) {
	if h.visionAdapter == nil {
		w.WriteHeader(http.StatusNotImplemented)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "OCR label scanning is not configured on this server"})
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, 5<<20)
	// #nosec G120 — MaxBytesReader above bounds the body before ParseMultipartForm.
	if err := r.ParseMultipartForm(5 << 20); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "file too large (max 5 MB)"})
		return
	}

	file, _, err := r.FormFile("file")
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "file field required"})
		return
	}
	defer func() { _ = file.Close() }()

	data, err := io.ReadAll(io.LimitReader(file, 5<<20))
	if err != nil {
		h.writeErr(w, err)
		return
	}

	mimeType := http.DetectContentType(data)
	if len(mimeType) < 6 || mimeType[:6] != "image/" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "uploaded file is not an image"})
		return
	}

	draft, err := h.visionAdapter.ExtractLabel(r.Context(), data, mimeType)
	if err != nil {
		h.writeErr(w, err)
		return
	}
	_ = json.NewEncoder(w).Encode(draft)
}
