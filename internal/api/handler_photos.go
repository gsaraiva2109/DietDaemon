package api

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/gsaraiva2109/dietdaemon/core/types"
	"github.com/gsaraiva2109/dietdaemon/internal/store"
)

// ---------------------------------------------------------------------------
// Body tracking — progress photo handlers.
// ---------------------------------------------------------------------------

// decodeImageUpload reads a single-file multipart upload from the "file"
// field, bounding the request body to maxBytes and sniffing the content
// type of the decoded bytes. On failure it writes the appropriate error
// response itself and returns ok=false; callers must return immediately.
func (h *Handler) decodeImageUpload(w http.ResponseWriter, r *http.Request, maxBytes int64) (data []byte, mimeType string, ok bool) {
	r.Body = http.MaxBytesReader(w, r.Body, maxBytes)
	// #nosec G120 — MaxBytesReader above bounds the body before ParseMultipartForm.
	if err := r.ParseMultipartForm(maxBytes); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": fmt.Sprintf("file too large (max %d MB)", maxBytes>>20)})
		return nil, "", false
	}

	file, _, err := r.FormFile("file")
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "file field required"})
		return nil, "", false
	}
	defer func() { _ = file.Close() }()

	data, err = io.ReadAll(io.LimitReader(file, maxBytes))
	if err != nil {
		h.writeErr(w, err)
		return nil, "", false
	}

	return data, http.DetectContentType(data), true
}

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
	photo, err := h.store.GetPhotoData(r.Context(), userID, photoID)
	if err != nil {
		h.writeErr(w, err)
		return
	}
	// Defense in depth: the store query is already scoped by user_id, so this
	// is redundant in the real store, but keeps the handler safe even if it's
	// ever called against a store implementation that isn't scoped correctly.
	if photo.UserID != userID {
		h.writeErr(w, types.ErrNotFound)
		return
	}
	w.Header().Set("Content-Type", photo.MimeType)
	w.Header().Set("Cache-Control", "private, max-age=86400")
	_, _ = w.Write(photo.Data)
}

func (h *Handler) handleUploadPhoto(w http.ResponseWriter, r *http.Request, userID string) {
	data, mimeType, ok := h.decodeImageUpload(w, r, 5<<20)
	if !ok {
		return
	}

	view := r.FormValue("view")
	if view == "" {
		view = "front"
	}
	date := r.FormValue("date")
	if date == "" {
		date = time.Now().In(h.userLoc(r.Context(), userID)).Format("2006-01-02")
	}

	if !validView(view) {
		writeValidationError(w, "view must be one of: front, side, back")
		return
	}
	if !validPhotoMimeType(mimeType) {
		writeValidationError(w, "unsupported image type; allowed: jpeg, png, webp")
		return
	}
	count, err := h.store.CountPhotos(r.Context(), userID)
	if err != nil {
		h.writeErr(w, err)
		return
	}
	if count >= store.MaxPhotosPerUser {
		h.writeErr(w, types.ErrQuotaExceeded)
		return
	}

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
	meta, _ := json.Marshal(map[string]string{"photo_id": photoID})
	h.writeAudit(r.Context(), "", userID, "photo.delete", h.clientIP(r), r.UserAgent(), string(meta))
	w.WriteHeader(http.StatusNoContent)
}
