package web

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHandlerServesSPA(t *testing.T) {
	h, err := Handler()
	if err != nil {
		t.Fatalf("Handler: %v", err)
	}

	cases := []struct {
		name string
		path string
	}{
		{"root", "/"},
		{"client route falls back to index", "/history/abc123"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, tc.path, nil))
			if rec.Code != http.StatusOK {
				t.Fatalf("status = %d, want 200", rec.Code)
			}
			if !strings.Contains(rec.Body.String(), "<div id=\"root\">") &&
				!strings.Contains(rec.Body.String(), "DietDaemon") {
				t.Fatalf("body does not look like the SPA index: %q", rec.Body.String())
			}
		})
	}
}
