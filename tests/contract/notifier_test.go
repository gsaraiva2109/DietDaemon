package contract

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gsaraiva2109/dietdaemon/adapters/notifier/gotify"
	"github.com/gsaraiva2109/dietdaemon/adapters/notifier/ntfy"
	"github.com/gsaraiva2109/dietdaemon/core/ports"
	"github.com/gsaraiva2109/dietdaemon/core/types"
)

// notifierFactory creates a ready Notifier pointing at a test server.
type notifierFactory func(t *testing.T, srv *httptest.Server) ports.Notifier

var notifiers = map[string]notifierFactory{
	"ntfy": func(t *testing.T, srv *httptest.Server) ports.Notifier {
		t.Helper()
		return ntfy.New(srv.URL, "test-topic", "")
	},
	"gotify": func(t *testing.T, srv *httptest.Server) ports.Notifier {
		t.Helper()
		return gotify.New(srv.URL, "test-token")
	},
}

// TestNotifierContract verifies that every Notifier sends valid HTTP requests
// for a notification.
func TestNotifierContract(t *testing.T) {
	for name, factory := range notifiers {
		t.Run(name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodPost {
					t.Errorf("%s: method = %s, want POST", name, r.Method)
				}
				w.WriteHeader(http.StatusOK)
			}))
			defer srv.Close()

			n := factory(t, srv)

			if got := n.Name(); got != name {
				t.Errorf("Name() = %q, want %q", got, name)
			}

			err := n.Notify(context.Background(), types.Notification{
				UserID:   "u1",
				Title:    "Test Title",
				Body:     "Test body",
				Priority: types.PriorityHigh,
			})
			if err != nil {
				t.Errorf("Notify: %v", err)
			}
		})
	}
}
