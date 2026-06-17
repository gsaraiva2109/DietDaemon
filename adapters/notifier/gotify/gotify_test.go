package gotify

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gsaraiva2109/dietdaemon/core/types"
)

func TestNotify(t *testing.T) {
	var (
		gotPath   string
		gotMethod string
		gotToken  string
		gotBody   message
	)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotMethod = r.Method
		gotToken = r.URL.Query().Get("token")

		b, _ := io.ReadAll(r.Body)
		var msg message
		if err := json.Unmarshal(b, &msg); err != nil {
			t.Errorf("decode body: %v", err)
		}
		gotBody = msg

		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	n := New(srv.URL, "app-token-123")

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
	if gotPath != "/message" {
		t.Errorf("path = %q, want /message", gotPath)
	}
	if gotToken != "app-token-123" {
		t.Errorf("token = %q, want app-token-123", gotToken)
	}
	if gotBody.Title != "Protein check" {
		t.Errorf("title = %q", gotBody.Title)
	}
	if gotBody.Message != msg.Body {
		t.Errorf("message = %q", gotBody.Message)
	}
	if gotBody.Priority != 8 {
		t.Errorf("priority = %d, want 8", gotBody.Priority)
	}
}

func TestPriorityMapping(t *testing.T) {
	tests := []struct {
		p    types.NotificationPriority
		want int
	}{
		{types.PriorityLow, 2},
		{types.PriorityDefault, 5},
		{types.PriorityHigh, 8},
	}
	for _, tc := range tests {
		got := priorityInt(tc.p)
		if got != tc.want {
			t.Errorf("priorityInt(%d) = %d, want %d", tc.p, got, tc.want)
		}
	}
}

func TestName(t *testing.T) {
	n := New("https://gotify.example.com", "t")
	if n.Name() != "gotify" {
		t.Errorf("Name() = %q, want gotify", n.Name())
	}
}

func TestContextCancellation(t *testing.T) {
	n := New("https://gotify.example.com", "t")
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := n.Notify(ctx, types.Notification{Title: "x", Body: "x"})
	if err == nil {
		t.Error("expected error from cancelled context")
	}
}

func TestHTTPErrorStatus(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer srv.Close()

	n := New(srv.URL, "bad-token")
	err := n.Notify(context.Background(), types.Notification{Title: "x", Body: "x"})
	if err == nil {
		t.Error("expected error on 401 status")
	}
}

func TestJSONBodyFields(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		var raw map[string]interface{}
		json.Unmarshal(b, &raw)

		// Only the three expected fields.
		if len(raw) != 3 {
			t.Errorf("expected 3 JSON fields, got %d: %v", len(raw), raw)
		}
		if _, ok := raw["title"]; !ok {
			t.Error("missing title field")
		}
		if _, ok := raw["message"]; !ok {
			t.Error("missing message field")
		}
		if _, ok := raw["priority"]; !ok {
			t.Error("missing priority field")
		}

		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	n := New(srv.URL, "t")
	n.Notify(context.Background(), types.Notification{Title: "T", Body: "B"})
}
