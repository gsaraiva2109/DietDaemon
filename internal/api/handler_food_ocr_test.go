package api

import (
	"bytes"
	"context"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gsaraiva2109/dietdaemon/core/types"
)

// fakeVisionAdapter returns a pre-programmed draft/error, mirroring
// fakeChatAdapter in internal/assistant/assistant_test.go.
type fakeVisionAdapter struct {
	draft types.NutritionLabelDraft
	err   error

	calledMime string
	calledLen  int
}

func (f *fakeVisionAdapter) ExtractLabel(_ context.Context, image []byte, mimeType string) (types.NutritionLabelDraft, error) {
	f.calledMime = mimeType
	f.calledLen = len(image)
	if f.err != nil {
		return types.NutritionLabelDraft{}, f.err
	}
	return f.draft, nil
}

// tiny 1x1 PNG, enough for http.DetectContentType to report image/png.
var testPNGBytes = []byte{
	0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A,
	0x00, 0x00, 0x00, 0x0D, 0x49, 0x48, 0x44, 0x52,
	0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x01,
	0x08, 0x06, 0x00, 0x00, 0x00, 0x1F, 0x15, 0xC4,
	0x89, 0x00, 0x00, 0x00, 0x0A, 0x49, 0x44, 0x41,
	0x54, 0x78, 0x9C, 0x63, 0x00, 0x01, 0x00, 0x00,
	0x05, 0x00, 0x01, 0x0D, 0x0A, 0x2D, 0xB4, 0x00,
	0x00, 0x00, 0x00, 0x49, 0x45, 0x4E, 0x44, 0xAE,
	0x42, 0x60, 0x82,
}

func doOCRUpload(h *Handler, fileContent []byte, fileName string) *httptest.ResponseRecorder {
	var body bytes.Buffer
	w := multipart.NewWriter(&body)
	if fileName != "" {
		part, _ := w.CreateFormFile("file", fileName)
		_, _ = part.Write(fileContent)
	}
	_ = w.Close()

	req := httptest.NewRequest(http.MethodPost, "/api/v1/foods/custom/ocr", &body)
	req.Header.Set("Content-Type", w.FormDataContentType())
	req.Header.Set("Authorization", "Bearer test-api-key")

	rec := httptest.NewRecorder()
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)
	mux.ServeHTTP(rec, req)
	return rec
}

func TestHandleOCRExtractCustomFood(t *testing.T) {
	store := &fakeMealStore{}
	name := "Whole Milk"
	calories := 61.0
	adapter := &fakeVisionAdapter{draft: types.NutritionLabelDraft{Name: &name, Calories: &calories}}
	h := newHandler(store, &fakeMealLogger{})
	h.visionAdapter = adapter

	rec := doOCRUpload(h, testPNGBytes, "label.png")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", rec.Code, rec.Body.String())
	}
	got := decodeJSON[types.NutritionLabelDraft](t, rec)
	if got.Name == nil || *got.Name != "Whole Milk" {
		t.Errorf("Name = %v, want Whole Milk", got.Name)
	}
	if adapter.calledMime != "image/png" {
		t.Errorf("ExtractLabel mimeType = %q, want image/png", adapter.calledMime)
	}
	if adapter.calledLen != len(testPNGBytes) {
		t.Errorf("ExtractLabel image len = %d, want %d", adapter.calledLen, len(testPNGBytes))
	}
}

func TestHandleOCRExtractCustomFoodDisabled(t *testing.T) {
	h := newHandler(&fakeMealStore{}, &fakeMealLogger{})
	// h.visionAdapter left nil: OCR not configured.

	rec := doOCRUpload(h, testPNGBytes, "label.png")
	if rec.Code != http.StatusNotImplemented {
		t.Fatalf("status = %d, want 501; body=%s", rec.Code, rec.Body.String())
	}
}

func TestHandleOCRExtractCustomFoodMissingFile(t *testing.T) {
	h := newHandler(&fakeMealStore{}, &fakeMealLogger{})
	h.visionAdapter = &fakeVisionAdapter{}

	rec := doOCRUpload(h, nil, "")
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400; body=%s", rec.Code, rec.Body.String())
	}
}

func TestHandleOCRExtractCustomFoodNonImage(t *testing.T) {
	h := newHandler(&fakeMealStore{}, &fakeMealLogger{})
	h.visionAdapter = &fakeVisionAdapter{}

	rec := doOCRUpload(h, []byte("not an image, just plain text bytes"), "label.txt")
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400; body=%s", rec.Code, rec.Body.String())
	}
}

func TestHandleOCRExtractCustomFoodAdapterError(t *testing.T) {
	h := newHandler(&fakeMealStore{}, &fakeMealLogger{})
	h.visionAdapter = &fakeVisionAdapter{err: context.DeadlineExceeded}

	rec := doOCRUpload(h, testPNGBytes, "label.png")
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500; body=%s", rec.Code, rec.Body.String())
	}
}
