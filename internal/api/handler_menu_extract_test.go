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

// ---------------------------------------------------------------------------
// handleExtractMenuFromImage (issue #201)
// ---------------------------------------------------------------------------

func doExtractMenuFromImage(h *Handler, fileContent []byte, fileName string) *httptest.ResponseRecorder {
	var body bytes.Buffer
	w := multipart.NewWriter(&body)
	if fileName != "" {
		part, _ := w.CreateFormFile("file", fileName)
		_, _ = part.Write(fileContent)
	}
	_ = w.Close()

	req := httptest.NewRequest(http.MethodPost, "/api/v1/menu/extract/image", &body)
	req.Header.Set("Content-Type", w.FormDataContentType())
	req.Header.Set("Authorization", "Bearer test-api-key")

	rec := httptest.NewRecorder()
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)
	mux.ServeHTTP(rec, req)
	return rec
}

func TestHandleExtractMenuFromImageDisabled(t *testing.T) {
	h := newHandler(&fakeMealStore{}, &fakeMealLogger{})
	// h.visionAdapter left nil: menu photo extraction not configured.

	rec := doExtractMenuFromImage(h, testPNGBytes, "menu.png")
	if rec.Code != http.StatusNotImplemented {
		t.Fatalf("status = %d, want 501; body=%s", rec.Code, rec.Body.String())
	}
}

func TestHandleExtractMenuFromImageMissingFile(t *testing.T) {
	h := newHandler(&fakeMealStore{}, &fakeMealLogger{})
	h.visionAdapter = &fakeVisionAdapter{}

	rec := doExtractMenuFromImage(h, nil, "")
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400; body=%s", rec.Code, rec.Body.String())
	}
}

func TestHandleExtractMenuFromImageNotAnImage(t *testing.T) {
	h := newHandler(&fakeMealStore{}, &fakeMealLogger{})
	h.visionAdapter = &fakeVisionAdapter{}

	rec := doExtractMenuFromImage(h, []byte("plain text, not an image"), "menu.txt")
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400; body=%s", rec.Code, rec.Body.String())
	}
}

func TestHandleExtractMenuFromImageOversized(t *testing.T) {
	h := newHandler(&fakeMealStore{}, &fakeMealLogger{})
	h.visionAdapter = &fakeVisionAdapter{}

	rec := doExtractMenuFromImage(h, bytes.Repeat([]byte("a"), 6<<20), "menu.png")
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400; body=%s", rec.Code, rec.Body.String())
	}
}

func TestHandleExtractMenuFromImageAdapterError(t *testing.T) {
	h := newHandler(&fakeMealStore{}, &fakeMealLogger{})
	h.visionAdapter = &fakeVisionAdapter{err: context.DeadlineExceeded}

	rec := doExtractMenuFromImage(h, testPNGBytes, "menu.png")
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500; body=%s", rec.Code, rec.Body.String())
	}
}

func TestHandleExtractMenuFromImage(t *testing.T) {
	adapter := &fakeVisionAdapter{menuDraft: types.MenuDraft{
		Dishes: []types.MenuDishCandidate{{Name: "Frango à parmegiana", Description: "Peito empanado, molho de tomate"}},
	}}
	h := newHandler(&fakeMealStore{}, &fakeMealLogger{})
	h.visionAdapter = adapter

	rec := doExtractMenuFromImage(h, testPNGBytes, "menu.png")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", rec.Code, rec.Body.String())
	}
	got := decodeJSON[types.MenuDraft](t, rec)
	if len(got.Dishes) != 1 || got.Dishes[0].Name != "Frango à parmegiana" {
		t.Errorf("Dishes = %+v, want one dish named Frango à parmegiana", got.Dishes)
	}
	if adapter.calledMime != "image/png" {
		t.Errorf("ExtractMenu mimeType = %q, want image/png", adapter.calledMime)
	}
	if adapter.calledLen != len(testPNGBytes) {
		t.Errorf("ExtractMenu image len = %d, want %d", adapter.calledLen, len(testPNGBytes))
	}
}

func TestHandleExtractMenuFromImageUnreadable(t *testing.T) {
	adapter := &fakeVisionAdapter{menuDraft: types.MenuDraft{Unreadable: true}}
	h := newHandler(&fakeMealStore{}, &fakeMealLogger{})
	h.visionAdapter = adapter

	rec := doExtractMenuFromImage(h, testPNGBytes, "menu.png")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", rec.Code, rec.Body.String())
	}
	got := decodeJSON[types.MenuDraft](t, rec)
	if !got.Unreadable {
		t.Errorf("Unreadable = false, want true")
	}
	if len(got.Dishes) != 0 {
		t.Errorf("Dishes = %v, want empty", got.Dishes)
	}
}

// ---------------------------------------------------------------------------
// handleLogMenuDish (issue #201)
// ---------------------------------------------------------------------------

func doLogMenuDish(h *Handler, body map[string]string) *httptest.ResponseRecorder {
	return doRequest(h, http.MethodPost, "/api/v1/menu/log-dish", body, map[string]string{"Authorization": "Bearer test-api-key"})
}

func TestHandleLogMenuDishEmptyName(t *testing.T) {
	h := newHandler(&fakeMealStore{}, &fakeMealLogger{})

	rec := doLogMenuDish(h, map[string]string{"description": "some description"})
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400; body=%s", rec.Code, rec.Body.String())
	}
}

func TestHandleLogMenuDishNoResolvedItems(t *testing.T) {
	logger := &fakeMealLogger{parseItems: nil}
	h := newHandler(&fakeMealStore{}, logger)

	rec := doLogMenuDish(h, map[string]string{"name": "Mystery dish"})
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400; body=%s", rec.Code, rec.Body.String())
	}
}

func TestHandleLogMenuDishForcesLowConfidence(t *testing.T) {
	logger := &fakeMealLogger{
		parseItems: []types.ResolvedItem{
			{Match: types.FoodMatch{FoodID: "food-1", Name: "Frango à parmegiana"}},
		},
	}
	h := newHandler(&fakeMealStore{}, logger)

	rec := doLogMenuDish(h, map[string]string{"name": "Frango à parmegiana", "description": "Peito empanado"})
	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201; body=%s", rec.Code, rec.Body.String())
	}
	got := decodeJSON[types.Meal](t, rec)
	if got.Confidence != restaurantEstimateConfidence {
		t.Errorf("Confidence = %v, want forced %v regardless of resolved items", got.Confidence, restaurantEstimateConfidence)
	}
	if logger.loggedConfidence != restaurantEstimateConfidence {
		t.Errorf("LogMealFromItems called with confidence = %v, want %v", logger.loggedConfidence, restaurantEstimateConfidence)
	}
	if got.RawText != "Frango à parmegiana. Peito empanado" {
		t.Errorf("RawText = %q, want name + description joined", got.RawText)
	}
}

func TestHandleLogMenuDishParseError(t *testing.T) {
	logger := &fakeMealLogger{parseErr: context.DeadlineExceeded}
	h := newHandler(&fakeMealStore{}, logger)

	rec := doLogMenuDish(h, map[string]string{"name": "Frango à parmegiana"})
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500; body=%s", rec.Code, rec.Body.String())
	}
}
