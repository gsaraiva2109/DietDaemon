package api

import (
	"encoding/json"
	"io"
	"net/http"
	"time"

	"github.com/gsaraiva2109/dietdaemon/core/types"
)

// ---------------------------------------------------------------------------
// Body tracking — progress photo handlers.
// ---------------------------------------------------------------------------

func (h *Handler) handleListPhotos(w http.ResponseWriter, r *http.Request, userID string) {
	photos, err := h.store.ListPhotoMetadata(r.Context(), userID)
	if err != nil {
		h.writeErr(w, err)
		return
	}
	if photos == nil {
		photos = []types.ProgressPhoto{}
	}
	_ = json.NewEncoder(w).Encode(photos)
}

func (h *Handler) handlePhotoData(w http.ResponseWriter, r *http.Request, userID string) {
	photoID := r.PathValue("id")
	photo, err := h.store.GetPhotoData(r.Context(), photoID)
	if err != nil {
		h.writeErr(w, err)
		return
	}
	if photo.UserID != userID {
		h.writeErr(w, types.ErrNotFound)
		return
	}
	w.Header().Set("Content-Type", photo.MimeType)
	w.Header().Set("Cache-Control", "private, max-age=86400")
	_, _ = w.Write(photo.Data)
}

func (h *Handler) handleUploadPhoto(w http.ResponseWriter, r *http.Request, userID string) {
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

	view := r.FormValue("view")
	if view == "" {
		view = "front"
	}
	date := r.FormValue("date")
	if date == "" {
		date = time.Now().In(h.loc).Format("2006-01-02")
	}

	// Detect mime type from first 512 bytes.
	mimeType := http.DetectContentType(data)

	photo := types.ProgressPhoto{
		ID:        newHandlerID(),
		UserID:    userID,
		Date:      date,
		View:      view,
		MimeType:  mimeType,
		Data:      data,
		CreatedAt: time.Now().UTC(),
	}
	if err := h.store.UploadPhoto(r.Context(), photo); err != nil {
		h.writeErr(w, err)
		return
	}
	// Clear data before JSON response.
	photo.Data = nil
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(photo)
}

func (h *Handler) handleDeletePhoto(w http.ResponseWriter, r *http.Request, userID string) {
	photoID := r.PathValue("id")
	if err := h.store.DeletePhoto(r.Context(), userID, photoID); err != nil {
		h.writeErr(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
