package api

import (
	"bytes"
	"errors"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gsaraiva2109/dietdaemon/core/types"
)

func TestPhotosRoutesRequireAuth(t *testing.T) {
	store := newFakeMealStore()
	h := newHandler(store, &fakeMealLogger{})

	rec := doRequest(h, "GET", "/api/v1/body/photos", nil, map[string]string{"Authorization": ""})
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rec.Code)
	}
}

func TestListPhotosStoreError(t *testing.T) {
	store := newFakeMealStore()
	store.photoMetadataErr = errors.New("db unavailable")
	h := newHandler(store, &fakeMealLogger{})

	rec := doRequest(h, "GET", "/api/v1/body/photos", nil, nil)
	if rec.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", rec.Code)
	}
}

func TestPhotoDataHappyPath(t *testing.T) {
	store := newFakeMealStore()
	store.photoData = types.ProgressPhoto{
		ID: "p1", UserID: "test-user", MimeType: "image/png", Data: []byte("pngdata"),
	}
	h := newHandler(store, &fakeMealLogger{})

	rec := doRequest(h, "GET", "/api/v1/body/photos/p1/data", nil, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	if ct := rec.Header().Get("Content-Type"); ct != "image/png" {
		t.Errorf("content-type = %q, want image/png", ct)
	}
	if rec.Body.String() != "pngdata" {
		t.Errorf("body = %q, want pngdata", rec.Body.String())
	}
}

// TestPhotoDataWrongUser exercises the ownership check: the store returns a
// photo, but its UserID belongs to someone else, so the handler must 404
// rather than leak another user's photo bytes.
func TestPhotoDataWrongUser(t *testing.T) {
	store := newFakeMealStore()
	store.photoData = types.ProgressPhoto{ID: "p1", UserID: "other-user", MimeType: "image/png", Data: []byte("x")}
	h := newHandler(store, &fakeMealLogger{})

	rec := doRequest(h, "GET", "/api/v1/body/photos/p1/data", nil, nil)
	if rec.Code != http.StatusNotFound {
		t.Errorf("cross-user photo access expected 404, got %d", rec.Code)
	}
}

// multipartUploadRequest builds a POST request with a "file" form field, plus
// optional extra form fields, since doRequest only knows how to send JSON.
func multipartUploadRequest(fields map[string]string, includeFile bool, fileBytes []byte) *http.Request {
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	if includeFile {
		fw, _ := mw.CreateFormFile("file", "photo.png")
		_, _ = fw.Write(fileBytes)
	}
	for k, v := range fields {
		_ = mw.WriteField(k, v)
	}
	_ = mw.Close()

	req := httptest.NewRequest("POST", "/api/v1/body/photos", &buf)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	req.Header.Set("Authorization", "Bearer test-api-key")
	return req
}

// pngBytes is a minimal valid PNG signature so http.DetectContentType
// reports "image/png".
var pngBytes = []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A}

func TestUploadPhoto(t *testing.T) {
	store := newFakeMealStore()
	h := newHandler(store, &fakeMealLogger{})

	req := multipartUploadRequest(map[string]string{"view": "side", "date": "2026-06-17"}, true, pngBytes)
	rec := httptest.NewRecorder()
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rec.Code, rec.Body.String())
	}
	photo := decodeJSON[types.ProgressPhoto](t, rec)
	if photo.View != "side" || photo.Date != "2026-06-17" {
		t.Errorf("unexpected photo metadata: %+v", photo)
	}
	if photo.MimeType != "image/png" {
		t.Errorf("mime type = %q, want image/png", photo.MimeType)
	}
	if photo.Data != nil {
		t.Errorf("expected data cleared from response, got %d bytes", len(photo.Data))
	}
}

func TestUploadPhotoDefaultsViewAndDate(t *testing.T) {
	store := newFakeMealStore()
	h := newHandler(store, &fakeMealLogger{})

	req := multipartUploadRequest(nil, true, pngBytes)
	rec := httptest.NewRecorder()
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rec.Code, rec.Body.String())
	}
	photo := decodeJSON[types.ProgressPhoto](t, rec)
	if photo.View != "front" {
		t.Errorf("expected default view front, got %q", photo.View)
	}
	if photo.Date == "" {
		t.Errorf("expected default date to be populated")
	}
}

func TestUploadPhotoMissingFile(t *testing.T) {
	store := newFakeMealStore()
	h := newHandler(store, &fakeMealLogger{})

	req := multipartUploadRequest(map[string]string{"view": "front"}, false, nil)
	rec := httptest.NewRecorder()
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("missing file expected 400, got %d", rec.Code)
	}
}

func TestUploadPhotoStoreError(t *testing.T) {
	store := newFakeMealStore()
	store.uploadPhotoErr = errors.New("disk full")
	h := newHandler(store, &fakeMealLogger{})

	req := multipartUploadRequest(nil, true, pngBytes)
	rec := httptest.NewRecorder()
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", rec.Code)
	}
}

func TestDeletePhotoNotFound(t *testing.T) {
	store := newFakeMealStore()
	store.deletePhotoErr = types.ErrNotFound
	h := newHandler(store, &fakeMealLogger{})

	rec := doRequest(h, "DELETE", "/api/v1/body/photos/missing", nil, nil)
	if rec.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", rec.Code)
	}
}
