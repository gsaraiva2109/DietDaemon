package api

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gsaraiva2109/dietdaemon/core/types"
	"github.com/gsaraiva2109/dietdaemon/internal/auth"
	"github.com/gsaraiva2109/dietdaemon/internal/config"
)

func TestAuthenticatedRateLimitCategories(t *testing.T) {
	h := New(nil, nil, nil, nil, &config.Config{
		AuthenticatedReadRateLimitPerMinute:      1,
		AuthenticatedWriteRateLimitPerMinute:     1,
		AuthenticatedExpensiveRateLimitPerMinute: 1,
	})
	for _, tt := range []struct {
		name   string
		method string
		path   string
	}{
		{name: "read", method: http.MethodGet, path: "/api/v1/meals"},
		{name: "write", method: http.MethodPost, path: "/api/v1/meals"},
		{name: "expensive", method: http.MethodGet, path: "/api/v1/suggest"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			r := httptest.NewRequest(tt.method, tt.path, nil)
			limiter := h.authLimiter(r)
			if !limiter.Allow("user") || limiter.Allow("user") {
				t.Fatal("limiter did not enforce its category limit")
			}
		})
	}
}

func TestAuthenticatedRateLimitReturnsStructuredError(t *testing.T) {
	store := newFakeAuthStore()
	store.users["rate-user"] = types.User{ID: "rate-user"}
	store.keyUserID[auth.HashToken("rate-key")] = "rate-user"
	h := New(nil, nil, nil, nil, &config.Config{AuthenticatedReadRateLimitPerMinute: 1})
	h.authStore = store
	wrapped := h.wrap(func(w http.ResponseWriter, _ *http.Request, _ string) { w.WriteHeader(http.StatusNoContent) })

	for attempt, want := range []int{http.StatusNoContent, http.StatusTooManyRequests} {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/meals", nil)
		req.Header.Set("Authorization", "Bearer rate-key")
		rec := httptest.NewRecorder()
		wrapped.ServeHTTP(rec, req)
		if rec.Code != want {
			t.Fatalf("attempt %d status = %d, want %d", attempt+1, rec.Code, want)
		}
		if want == http.StatusTooManyRequests && rec.Body.String() != "{\"error\":{\"code\":\"rate_limited\",\"message\":\"Too many requests.\"}}\n" {
			t.Fatalf("error body = %q", rec.Body.String())
		}
	}
}

func TestExpensiveRequestRoutes(t *testing.T) {
	for _, path := range []string{
		"/api/v1/chat/sessions/a/messages", "/api/v1/suggest", "/api/v1/suggest/ingredients",
		"/api/v1/goals/suggestions", "/api/v1/foods/custom/ocr", "/api/v1/settings/backup/run",
	} {
		if !isExpensiveRequest(httptest.NewRequest(http.MethodGet, path, nil)) {
			t.Errorf("%s is not expensive", path)
		}
	}
}
