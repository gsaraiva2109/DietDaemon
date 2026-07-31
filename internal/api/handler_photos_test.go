package api

import (
	"bytes"
	"context"
	"errors"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gsaraiva2109/dietdaemon/core/types"
	photostore "github.com/gsaraiva2109/dietdaemon/internal/store"
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

// crossUserPhotoStore wraps fakeMealStore to apply the same per-user scoping
// the real SQL-backed store enforces at the query layer (WHERE user_id = ?,
// see internal/store/store_photos.go). The base fake ignores the userID
// argument entirely, which is fine for single-user tests but can't exercise
// cross-user denial, so this wrapper adds it back for the one fixture photo
// under test.
type crossUserPhotoStore struct {
	*fakeMealStore
	ownerID string
}

func (s *crossUserPhotoStore) ListPhotoMetadata(ctx context.Context, userID string) ([]types.ProgressPhoto, error) {
	if userID != s.ownerID {
		return []types.ProgressPhoto{}, nil
	}
	return s.fakeMealStore.ListPhotoMetadata(ctx, userID)
}

func (s *crossUserPhotoStore) GetPhotoData(ctx context.Context, userID, photoID string) (types.ProgressPhoto, error) {
	if userID != s.ownerID {
		return types.ProgressPhoto{}, types.ErrNotFound
	}
	return s.fakeMealStore.GetPhotoData(ctx, userID, photoID)
}

func (s *crossUserPhotoStore) DeletePhoto(ctx context.Context, userID, photoID string) error {
	if userID != s.ownerID {
		return types.ErrNotFound
	}
	return s.fakeMealStore.DeletePhoto(ctx, userID, photoID)
}

// TestPhotoEndpointsCrossUserDenied consolidates cross-user authorization
// checks across every registered photo endpoint (see RegisterRoutes in
// handler.go): list, get-data, and delete. A photo is "owned" by ownerID
// while every request in this test authenticates as the fixed test-user
// (see doRequest), so each case exercises the same wrong-user scenario as
// TestPhotoDataWrongUser, just against a different endpoint. Upload is
// excluded: it creates a new resource rather than accessing an existing
// one, so there's no ownership boundary to cross.
func TestPhotoEndpointsCrossUserDenied(t *testing.T) {
	const ownerID = "owner-a"
	const photoID = "p1"

	tests := []struct {
		name       string
		method     string
		path       string
		wantStatus int
		checkBody  func(t *testing.T, rec *httptest.ResponseRecorder)
	}{
		{
			name:       "list excludes another user's photos",
			method:     "GET",
			path:       "/api/v1/body/photos",
			wantStatus: http.StatusOK,
			checkBody: func(t *testing.T, rec *httptest.ResponseRecorder) {
				photos := decodeJSON[[]types.ProgressPhoto](t, rec)
				if len(photos) != 0 {
					t.Errorf("list leaked %d photo(s) belonging to another user", len(photos))
				}
			},
		},
		{
			name:       "get-data denies another user's photo",
			method:     "GET",
			path:       "/api/v1/body/photos/" + photoID + "/data",
			wantStatus: http.StatusNotFound,
		},
		{
			name:       "delete denies another user's photo",
			method:     "DELETE",
			path:       "/api/v1/body/photos/" + photoID,
			wantStatus: http.StatusNotFound,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			store := &crossUserPhotoStore{fakeMealStore: newFakeMealStore(), ownerID: ownerID}
			store.photoMetadata = []types.ProgressPhoto{{ID: photoID, UserID: ownerID, View: "front"}}
			store.photoData = types.ProgressPhoto{ID: photoID, UserID: ownerID, MimeType: "image/png", Data: []byte("x")}
			h := newHandler(store, &fakeMealLogger{})

			rec := doRequest(h, tc.method, tc.path, nil, nil)
			if rec.Code != tc.wantStatus {
				t.Fatalf("%s %s: got status %d, want %d (body=%s)", tc.method, tc.path, rec.Code, tc.wantStatus, rec.Body.String())
			}
			if tc.checkBody != nil {
				tc.checkBody(t, rec)
			}
		})
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

// TestHandleUploadPhotoInvalidView is a regression test proving an invalid
// view value now gets a clean 400 instead of falling through to the DB's
// `view IN ('front','side','back')` CHECK constraint as a raw 500.
func TestHandleUploadPhotoInvalidView(t *testing.T) {
	store := newFakeMealStore()
	h := newHandler(store, &fakeMealLogger{})

	req := multipartUploadRequest(map[string]string{"view": "diagonal"}, true, pngBytes)
	rec := httptest.NewRecorder()
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)
	mux.ServeHTTP(rec, req)

	assertValidationError(t, rec)
}

func TestHandleUploadPhotoInvalidMimeType(t *testing.T) {
	store := newFakeMealStore()
	h := newHandler(store, &fakeMealLogger{})

	req := multipartUploadRequest(nil, true, []byte("not an image, just plain text"))
	rec := httptest.NewRecorder()
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)
	mux.ServeHTTP(rec, req)

	assertValidationError(t, rec)
}

func TestHandleUploadPhotoQuotaExceeded(t *testing.T) {
	store := newFakeMealStore()
	store.photoCount = photostore.MaxPhotosPerUser
	h := newHandler(store, &fakeMealLogger{})

	req := multipartUploadRequest(nil, true, pngBytes)
	rec := httptest.NewRecorder()
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusConflict {
		t.Fatalf("expected 409, got %d: %s", rec.Code, rec.Body.String())
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

// TestDeletePhotoLogsAuditEvent covers the plan's requirement that a
// single-photo delete stays immediate but now also logs an auth_audit_log
// event, so deletions are traceable even outside the tiered account-deletion
// flow.
func TestDeletePhotoLogsAuditEvent(t *testing.T) {
	mealStore := newFakeMealStore()
	authStore := newFakeAuthStore()
	h := newHandlerWithAccountStore(mealStore, authStore)

	rec := doRequest(h, "DELETE", "/api/v1/body/photos/p1", nil, nil)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d: %s", rec.Code, rec.Body.String())
	}

	if len(authStore.auditEvents) != 1 {
		t.Fatalf("expected 1 audit event, got %d", len(authStore.auditEvents))
	}
	ev := authStore.auditEvents[0]
	if ev.Event != "photo.delete" {
		t.Errorf("event = %q, want photo.delete", ev.Event)
	}
	if ev.UserID != "test-user" {
		t.Errorf("userID = %q, want test-user", ev.UserID)
	}
	if !strings.Contains(ev.Meta, "p1") {
		t.Errorf("meta = %q, want it to contain photo id p1", ev.Meta)
	}
}
