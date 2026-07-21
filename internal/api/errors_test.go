package api

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestAPIErrorEnvelope(t *testing.T) {
	h := withAPIErrorEnvelope(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "database: secret detail", http.StatusInternalServerError)
	}))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/v1/test", nil))

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d", rec.Code)
	}
	if got, want := rec.Header().Get("Content-Type"), "application/json"; got != want {
		t.Fatalf("content type = %q, want %q", got, want)
	}
	if got := rec.Body.String(); got != "{\"error\":{\"code\":\"internal_error\",\"message\":\"Internal server error.\"}}\n" {
		t.Fatalf("body = %q", got)
	}
}

func TestAPIErrorEnvelopePreservesStreaming(t *testing.T) {
	h := withAPIErrorEnvelope(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("data: ready\n\n"))
		w.(http.Flusher).Flush()
	}))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/v1/test/stream", nil))

	if got := rec.Body.String(); !strings.Contains(got, "data: ready") {
		t.Fatalf("stream body = %q", got)
	}
}
