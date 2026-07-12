package ntfy

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gsaraiva2109/dietdaemon/core/types"
)

func TestNotify(t *testing.T) {
	var (
		gotPath   string
		gotBody   string
		gotTitle  string
		gotPrio   string
		gotAuth   string
		gotMethod string
	)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotMethod = r.Method
		gotTitle = r.Header.Get("Title")
		gotPrio = r.Header.Get("Priority")
		gotAuth = r.Header.Get("Authorization")
		b, _ := io.ReadAll(r.Body)
		gotBody = string(b)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	n := New(srv.URL, "mytopic", "")

	msg := types.Notification{
		UserID:   "u1",
		Title:    "Protein check",
		Body:     "You're at 42% of daily protein target.",
		Priority: types.PriorityHigh,
	}

	if err := n.Notify(context.Background(), msg); err != nil {
		t.Fatalf("Notify: %v", err)
	}

	if gotMethod != http.MethodPost {
		t.Errorf("method = %q, want POST", gotMethod)
	}
	if gotPath != "/mytopic" {
		t.Errorf("path = %q, want /mytopic", gotPath)
	}
	if gotTitle != "Protein check" {
		t.Errorf("Title header = %q, want %q", gotTitle, "Protein check")
	}
	if gotPrio != "high" {
		t.Errorf("Priority header = %q, want high", gotPrio)
	}
	if gotBody != msg.Body {
		t.Errorf("body = %q, want %q", gotBody, msg.Body)
	}
	if gotAuth != "" {
		t.Errorf("Authorization header = %q, want empty (no token set)", gotAuth)
	}
}

func TestNotifyWithAuthToken(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		if auth != "Bearer secret-token" {
			t.Errorf("Authorization = %q, want Bearer secret-token", auth)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	n := New(srv.URL, "mytopic", "secret-token")

	err := n.Notify(context.Background(), types.Notification{Title: "x", Body: "x"})
	if err != nil {
		t.Fatalf("Notify with token: %v", err)
	}
}

func TestNotifyWithoutAuthToken(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		if auth != "" {
			t.Errorf("Authorization header present (%q) but no token was set", auth)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	n := New(srv.URL, "t", "")
	_ = n.Notify(context.Background(), types.Notification{Title: "x", Body: "x"})
}

func TestPriorityMapping(t *testing.T) {
	tests := []struct {
		p    types.NotificationPriority
		want string
	}{
		{types.PriorityLow, "low"},
		{types.PriorityDefault, "default"},
		{types.PriorityHigh, "high"},
	}
	for _, tc := range tests {
		got := priorityString(tc.p)
		if got != tc.want {
			t.Errorf("priorityString(%d) = %q, want %q", tc.p, got, tc.want)
		}
	}
}

func TestName(t *testing.T) {
	n := New("https://ntfy.sh", "test", "")
	if n.Name() != "ntfy" {
		t.Errorf("Name() = %q, want ntfy", n.Name())
	}
}

func TestContextCancellation(t *testing.T) {
	n := New("https://ntfy.sh", "test", "")
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := n.Notify(ctx, types.Notification{Title: "x", Body: "x"})
	if err == nil {
		t.Error("expected error from cancelled context")
	}
}

func TestHTTPErrorStatus(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	n := New(srv.URL, "t", "")
	err := n.Notify(context.Background(), types.Notification{Title: "x", Body: "x"})
	if err == nil {
		t.Error("expected error on 500 status")
	}
}

func TestTrailingSlashTrim(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/topic" {
			t.Errorf("path = %q, want /topic", r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	n := New(srv.URL+"/", "topic", "")
	_ = n.Notify(context.Background(), types.Notification{Title: "x", Body: "x"})
}

// ---------------------------------------------------------------------------
// Compile-time guard — re-asserted via the var block in the main file.
// This test just confirms the concrete type satisfies the interface at
// the type-checker level by calling the interface method through the var.
// ---------------------------------------------------------------------------

func TestInterfaceGuard(t *testing.T) {
	n := New("http://localhost", "t", "")
	_ = n.Name()
}
