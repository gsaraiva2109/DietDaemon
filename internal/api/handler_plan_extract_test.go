package api

import (
	"bytes"
	"context"
	"fmt"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gsaraiva2109/dietdaemon/core/types"
)

// fakeCompletionAdapter returns a pre-programmed completion response/error,
// mirroring fakeVisionAdapter in handler_food_ocr_test.go.
type fakeCompletionAdapter struct {
	response string
	err      error

	calledPrompt string
}

func (f *fakeCompletionAdapter) Complete(_ context.Context, prompt string) (string, error) {
	f.calledPrompt = prompt
	if f.err != nil {
		return "", f.err
	}
	return f.response, nil
}

func (f *fakeCompletionAdapter) Embed(_ context.Context, _ string) ([]float32, error) {
	return nil, nil
}

const validPlanDraftResponse = `{"plan_name":"Plano","day_types":[{"name":"Dia único","targets":{"Calories":2000,"Protein":150,"Carbs":200,"Fat":60,"Fiber":25},"water_goal_ml":2500,"slots":[],"low_confidence_fields":[]}],"unreadable":false,"notes":null}`

func doExtractPlanFromText(h *Handler, text string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodPost, "/api/v1/plans/extract/text", strings.NewReader(text))
	req.Header.Set("Content-Type", "text/plain")
	req.Header.Set("Authorization", "Bearer test-api-key")

	rec := httptest.NewRecorder()
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)
	mux.ServeHTTP(rec, req)
	return rec
}

func TestHandleExtractPlanFromText(t *testing.T) {
	adapter := &fakeCompletionAdapter{response: validPlanDraftResponse}
	h := newHandler(&fakeMealStore{}, &fakeMealLogger{})
	h.completionAdapter = adapter

	rec := doExtractPlanFromText(h, "Dia único: 2000 kcal, 150g proteína...")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", rec.Code, rec.Body.String())
	}
	got := decodeJSON[types.PlanDraft](t, rec)
	if got.PlanName == nil || *got.PlanName != "Plano" {
		t.Errorf("PlanName = %v, want Plano", got.PlanName)
	}
	if len(got.DayTypes) != 1 || got.DayTypes[0].Name != "Dia único" {
		t.Errorf("DayTypes = %+v, want one day type named Dia único", got.DayTypes)
	}
	if !strings.Contains(adapter.calledPrompt, "Dia único: 2000 kcal") {
		t.Errorf("Complete prompt does not contain the pasted text: %q", adapter.calledPrompt)
	}
}

func TestHandleExtractPlanFromTextEmptyBody(t *testing.T) {
	h := newHandler(&fakeMealStore{}, &fakeMealLogger{})
	h.completionAdapter = &fakeCompletionAdapter{}

	rec := doExtractPlanFromText(h, "")
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400; body=%s", rec.Code, rec.Body.String())
	}
}

func TestHandleExtractPlanFromTextOversized(t *testing.T) {
	h := newHandler(&fakeMealStore{}, &fakeMealLogger{})
	h.completionAdapter = &fakeCompletionAdapter{response: validPlanDraftResponse}

	rec := doExtractPlanFromText(h, strings.Repeat("a", maxPlanTextChars+1))
	if rec.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("status = %d, want 413; body=%s", rec.Code, rec.Body.String())
	}
}

func TestHandleExtractPlanFromTextAdapterError(t *testing.T) {
	h := newHandler(&fakeMealStore{}, &fakeMealLogger{})
	h.completionAdapter = &fakeCompletionAdapter{err: context.DeadlineExceeded}

	rec := doExtractPlanFromText(h, "some plan text")
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500; body=%s", rec.Code, rec.Body.String())
	}
}

func TestHandleExtractPlanFromTextUnparseableResponse(t *testing.T) {
	h := newHandler(&fakeMealStore{}, &fakeMealLogger{})
	h.completionAdapter = &fakeCompletionAdapter{response: "this is not json"}

	rec := doExtractPlanFromText(h, "some plan text")
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500; body=%s", rec.Code, rec.Body.String())
	}
}

// ---------------------------------------------------------------------------
// handleExtractPlanFromImage (issue #194)
// ---------------------------------------------------------------------------

func doExtractPlanFromImage(h *Handler, fileContent []byte, fileName string) *httptest.ResponseRecorder {
	var files map[string][]byte
	if fileName != "" {
		files = map[string][]byte{fileName: fileContent}
	}
	return doExtractPlanFromImages(h, files)
}

// doExtractPlanFromImages posts a multipart request with one "file" part per
// entry in files, in map iteration order isn't guaranteed — callers that
// care about page order should use doExtractPlanFromImagesOrdered.
func doExtractPlanFromImages(h *Handler, files map[string][]byte) *httptest.ResponseRecorder {
	names := make([]string, 0, len(files))
	for name := range files {
		names = append(names, name)
	}
	contents := make([][]byte, len(names))
	for i, name := range names {
		contents[i] = files[name]
	}
	return doExtractPlanFromImagesOrdered(h, names, contents)
}

// doExtractPlanFromImagesOrdered posts a multipart request with one "file"
// field per (names[i], contents[i]) pair, in the given order — the order the
// handler must preserve into the PlanImagePage slice it builds.
func doExtractPlanFromImagesOrdered(h *Handler, names []string, contents [][]byte) *httptest.ResponseRecorder {
	var body bytes.Buffer
	w := multipart.NewWriter(&body)
	for i, name := range names {
		part, _ := w.CreateFormFile("file", name)
		_, _ = part.Write(contents[i])
	}
	_ = w.Close()

	req := httptest.NewRequest(http.MethodPost, "/api/v1/plans/extract/image", &body)
	req.Header.Set("Content-Type", w.FormDataContentType())
	req.Header.Set("Authorization", "Bearer test-api-key")

	rec := httptest.NewRecorder()
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)
	mux.ServeHTTP(rec, req)
	return rec
}

func TestHandleExtractPlanFromImageDisabled(t *testing.T) {
	h := newHandler(&fakeMealStore{}, &fakeMealLogger{})
	// h.visionAdapter left nil: plan photo extraction not configured.

	rec := doExtractPlanFromImage(h, testPNGBytes, "plan.png")
	if rec.Code != http.StatusNotImplemented {
		t.Fatalf("status = %d, want 501; body=%s", rec.Code, rec.Body.String())
	}
}

func TestHandleExtractPlanFromImageMissingFile(t *testing.T) {
	h := newHandler(&fakeMealStore{}, &fakeMealLogger{})
	h.visionAdapter = &fakeVisionAdapter{}

	rec := doExtractPlanFromImage(h, nil, "")
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400; body=%s", rec.Code, rec.Body.String())
	}
}

func TestHandleExtractPlanFromImageNotAnImage(t *testing.T) {
	h := newHandler(&fakeMealStore{}, &fakeMealLogger{})
	h.visionAdapter = &fakeVisionAdapter{}

	rec := doExtractPlanFromImage(h, []byte("plain text, not an image"), "plan.txt")
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400; body=%s", rec.Code, rec.Body.String())
	}
}

func TestHandleExtractPlanFromImageOversized(t *testing.T) {
	h := newHandler(&fakeMealStore{}, &fakeMealLogger{})
	h.visionAdapter = &fakeVisionAdapter{}

	rec := doExtractPlanFromImage(h, bytes.Repeat([]byte("a"), maxPlanTotalBytes+1), "plan.png")
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400; body=%s", rec.Code, rec.Body.String())
	}
}

func TestHandleExtractPlanFromImageTotalSizeOverLimit(t *testing.T) {
	h := newHandler(&fakeMealStore{}, &fakeMealLogger{})
	h.visionAdapter = &fakeVisionAdapter{}

	// Two pages that individually fit but together exceed maxPlanTotalBytes.
	half := maxPlanTotalBytes/2 + 1
	names := []string{"page1.png", "page2.png"}
	contents := [][]byte{bytes.Repeat([]byte("a"), half), bytes.Repeat([]byte("a"), half)}

	rec := doExtractPlanFromImagesOrdered(h, names, contents)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400; body=%s", rec.Code, rec.Body.String())
	}
}

func TestHandleExtractPlanFromImagePageCountOverLimit(t *testing.T) {
	h := newHandler(&fakeMealStore{}, &fakeMealLogger{})
	h.visionAdapter = &fakeVisionAdapter{}

	names := make([]string, maxPlanPages+1)
	contents := make([][]byte, maxPlanPages+1)
	for i := range names {
		names[i] = fmt.Sprintf("page%d.png", i+1)
		contents[i] = testPNGBytes
	}

	rec := doExtractPlanFromImagesOrdered(h, names, contents)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400; body=%s", rec.Code, rec.Body.String())
	}
}

func TestHandleExtractPlanFromImageInvalidPageAmongValid(t *testing.T) {
	h := newHandler(&fakeMealStore{}, &fakeMealLogger{})
	h.visionAdapter = &fakeVisionAdapter{}

	names := []string{"page1.png", "page2.txt"}
	contents := [][]byte{testPNGBytes, []byte("plain text, not an image")}

	rec := doExtractPlanFromImagesOrdered(h, names, contents)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400; body=%s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "page 2") {
		t.Errorf("error body = %s, want it to name page 2", rec.Body.String())
	}
}

func TestHandleExtractPlanFromImageMultiplePages(t *testing.T) {
	planName := "Plano"
	adapter := &fakeVisionAdapter{planDraft: types.PlanDraft{
		PlanName: &planName,
		DayTypes: []types.PlanDraftDayType{{Name: "Dia único"}},
	}}
	h := newHandler(&fakeMealStore{}, &fakeMealLogger{})
	h.visionAdapter = adapter

	names := []string{"page1.png", "page2.png", "page3.png"}
	contents := [][]byte{testPNGBytes, testPNGBytes, testPNGBytes}

	rec := doExtractPlanFromImagesOrdered(h, names, contents)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", rec.Code, rec.Body.String())
	}
	if len(adapter.calledPages) != 3 {
		t.Fatalf("calledPages len = %d, want 3", len(adapter.calledPages))
	}
	for i, p := range adapter.calledPages {
		if p.MimeType != "image/png" {
			t.Errorf("calledPages[%d].MimeType = %q, want image/png", i, p.MimeType)
		}
		if len(p.Data) != len(testPNGBytes) {
			t.Errorf("calledPages[%d].Data len = %d, want %d", i, len(p.Data), len(testPNGBytes))
		}
	}
}

func TestHandleExtractPlanFromImageAdapterError(t *testing.T) {
	h := newHandler(&fakeMealStore{}, &fakeMealLogger{})
	h.visionAdapter = &fakeVisionAdapter{err: context.DeadlineExceeded}

	rec := doExtractPlanFromImage(h, testPNGBytes, "plan.png")
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500; body=%s", rec.Code, rec.Body.String())
	}
}

func TestHandleExtractPlanFromImage(t *testing.T) {
	planName := "Plano"
	adapter := &fakeVisionAdapter{planDraft: types.PlanDraft{
		PlanName: &planName,
		DayTypes: []types.PlanDraftDayType{{Name: "Dia único"}},
	}}
	h := newHandler(&fakeMealStore{}, &fakeMealLogger{})
	h.visionAdapter = adapter

	rec := doExtractPlanFromImage(h, testPNGBytes, "plan.png")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", rec.Code, rec.Body.String())
	}
	got := decodeJSON[types.PlanDraft](t, rec)
	if got.PlanName == nil || *got.PlanName != "Plano" {
		t.Errorf("PlanName = %v, want Plano", got.PlanName)
	}
	if len(got.DayTypes) != 1 || got.DayTypes[0].Name != "Dia único" {
		t.Errorf("DayTypes = %+v, want one day type named Dia único", got.DayTypes)
	}
	if len(adapter.calledPages) != 1 {
		t.Fatalf("calledPages len = %d, want 1", len(adapter.calledPages))
	}
	if adapter.calledPages[0].MimeType != "image/png" {
		t.Errorf("ExtractPlan mimeType = %q, want image/png", adapter.calledPages[0].MimeType)
	}
	if len(adapter.calledPages[0].Data) != len(testPNGBytes) {
		t.Errorf("ExtractPlan image len = %d, want %d", len(adapter.calledPages[0].Data), len(testPNGBytes))
	}
}
