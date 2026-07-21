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
		{"client route parameter may contain a dot", "/shared/token.example"},
		{"body tab route", "/body/measurements"},
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

func TestHandlerReturnsBrandedNotFoundForUnknownNavigation(t *testing.T) {
	h, err := Handler()
	if err != nil {
		t.Fatalf("Handler: %v", err)
	}
	req := httptest.NewRequest(http.MethodGet, "/not-a-route", nil)
	req.Header.Set("Accept", "text/html,application/xhtml+xml")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", rec.Code)
	}
	if got := rec.Header().Get("Content-Type"); got != "text/html; charset=utf-8" {
		t.Fatalf("content type = %q", got)
	}
	if !strings.Contains(rec.Body.String(), "DietDaemon") || !strings.Contains(rec.Body.String(), "Page not found") {
		t.Fatalf("body is not the branded 404: %q", rec.Body.String())
	}
}

func TestHandlerKeepsMissingAssetsAndNonHTMLRequestsAsNotFound(t *testing.T) {
	h, err := Handler()
	if err != nil {
		t.Fatalf("Handler: %v", err)
	}
	for _, tc := range []struct {
		name   string
		path   string
		accept string
	}{
		{name: "missing asset", path: "/assets/missing.js", accept: "text/html"},
		{name: "non HTML request", path: "/not-a-route", accept: "application/json"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tc.path, nil)
			req.Header.Set("Accept", tc.accept)
			rec := httptest.NewRecorder()
			h.ServeHTTP(rec, req)
			if rec.Code != http.StatusNotFound {
				t.Fatalf("status = %d, want 404", rec.Code)
			}
			if strings.Contains(rec.Body.String(), "Page not found") {
				t.Fatalf("body unexpectedly contains branded page: %q", rec.Body.String())
			}
		})
	}
}
