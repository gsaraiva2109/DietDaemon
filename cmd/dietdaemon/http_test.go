package main

import (
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gsaraiva2109/dietdaemon/internal/config"
)

func TestHTTPServerTimeouts(t *testing.T) {
	srv := newHTTPServer(":8080", http.NotFoundHandler())
	if srv.ReadHeaderTimeout != 3*time.Second || srv.WriteTimeout != 30*time.Second || srv.IdleTimeout != 120*time.Second {
		t.Fatalf("timeouts = read %v, write %v, idle %v", srv.ReadHeaderTimeout, srv.WriteTimeout, srv.IdleTimeout)
	}
}

func TestHTTPHandlerRecoversAndKeepsServing(t *testing.T) {
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/panic" {
			panic("boom")
		}
		w.WriteHeader(http.StatusNoContent)
	})
	h := newHTTPHandler(next, nil)

	panicRec := httptest.NewRecorder()
	h.ServeHTTP(panicRec, httptest.NewRequest(http.MethodGet, "/panic", nil))
	if panicRec.Code != http.StatusInternalServerError || panicRec.Header().Get("Content-Type") != "application/json" || !strings.Contains(panicRec.Body.String(), "internal server error") {
		t.Fatalf("panic response = %d, headers %v, body %q", panicRec.Code, panicRec.Header(), panicRec.Body.String())
	}
	okRec := httptest.NewRecorder()
	h.ServeHTTP(okRec, httptest.NewRequest(http.MethodGet, "/ok", nil))
	if okRec.Code != http.StatusNoContent {
		t.Fatalf("subsequent response = %d, want %d", okRec.Code, http.StatusNoContent)
	}
}

func TestHTTPHandlerSecurityHeadersAndHSTS(t *testing.T) {
	for _, tt := range []struct {
		name string
		cfg  *config.Config
		hsts string
	}{
		{name: "disabled"},
		{name: "enabled", cfg: &config.Config{HSTSEnabled: true}, hsts: "max-age=31536000"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			newHTTPHandler(http.NotFoundHandler(), tt.cfg).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/missing", nil))
			if rec.Header().Get("X-Content-Type-Options") != "nosniff" || rec.Header().Get("X-Frame-Options") != "DENY" || rec.Header().Get("Content-Security-Policy") != contentSecurityPolicy || rec.Header().Get("Strict-Transport-Security") != tt.hsts {
				t.Fatalf("headers = %v", rec.Header())
			}
		})
	}
}

func TestHTTPCORS(t *testing.T) {
	allowed := "https://app.example.com"
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusAccepted) })
	allowedHandler := newHTTPHandler(next, &config.Config{CORSAllowedOrigins: []string{allowed}})
	noCORSHandler := newHTTPHandler(next, &config.Config{})

	for _, tt := range []struct {
		name       string
		handler    http.Handler
		method     string
		origin     string
		wantStatus int
		wantCORS   bool
	}{
		{name: "allowed request", handler: allowedHandler, method: http.MethodGet, origin: allowed, wantStatus: http.StatusAccepted, wantCORS: true},
		{name: "allowed preflight", handler: allowedHandler, method: http.MethodOptions, origin: allowed, wantStatus: http.StatusNoContent, wantCORS: true},
		{name: "disallowed", handler: allowedHandler, method: http.MethodGet, origin: "https://other.example.com", wantStatus: http.StatusAccepted},
		{name: "default no cors", handler: noCORSHandler, method: http.MethodGet, origin: allowed, wantStatus: http.StatusAccepted},
	} {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, "/api/v1/healthz", nil)
			if tt.origin != "" {
				req.Header.Set("Origin", tt.origin)
			}
			rec := httptest.NewRecorder()
			tt.handler.ServeHTTP(rec, req)
			if rec.Code != tt.wantStatus || (rec.Header().Get("Access-Control-Allow-Origin") == allowed) != tt.wantCORS {
				t.Fatalf("response = %d, headers %v", rec.Code, rec.Header())
			}
			if tt.wantCORS && (rec.Header().Get("Access-Control-Allow-Credentials") != "true" || !strings.Contains(rec.Header().Get("Vary"), "Origin")) {
				t.Fatalf("missing credentialed CORS headers: %v", rec.Header())
			}
			if tt.method == http.MethodOptions && (rec.Header().Get("Access-Control-Allow-Methods") == "" || rec.Header().Get("Access-Control-Allow-Headers") == "") {
				t.Fatalf("missing preflight headers: %v", rec.Header())
			}
		})
	}
}

func TestHTTPBodyLimits(t *testing.T) {
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, err := io.Copy(io.Discard, r.Body)
		if err != nil {
			var maxErr *http.MaxBytesError
			if errors.As(err, &maxErr) {
				w.WriteHeader(http.StatusRequestEntityTooLarge)
				return
			}
			t.Fatal(err)
		}
		w.WriteHeader(http.StatusNoContent)
	})
	h := newHTTPHandler(next, nil)

	for _, tt := range []struct {
		name string
		path string
		size int
		want int
	}{
		{name: "default cap", path: "/api/v1/meals", size: defaultRequestBodyLimit + 1, want: http.StatusRequestEntityTooLarge},
		{name: "ocr exception", path: "/api/v1/foods/custom/ocr", size: defaultRequestBodyLimit + 1, want: http.StatusNoContent},
		{name: "photo exception", path: "/api/v1/body/photos", size: defaultRequestBodyLimit + 1, want: http.StatusNoContent},
		{name: "upload cap", path: "/api/v1/body/photos", size: uploadRequestBodyLimit + 1, want: http.StatusRequestEntityTooLarge},
	} {
		t.Run(tt.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			h.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, tt.path, strings.NewReader(strings.Repeat("x", tt.size))))
			if rec.Code != tt.want {
				t.Fatalf("status = %d, want %d", rec.Code, tt.want)
			}
		})
	}
}
