// Package ntfy implements ports.Notifier for ntfy (https://ntfy.sh).
// Notifications are delivered as HTTP POSTs with Title and Priority headers.
package ntfy

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/gsaraiva2109/dietdaemon/core/ports"
	"github.com/gsaraiva2109/dietdaemon/core/types"
)

// Compile-time interface check.
var _ ports.Notifier = (*Notifier)(nil)

// Notifier sends push notifications via an ntfy server.
type Notifier struct {
	url    string // base URL of the ntfy server, e.g. "https://ntfy.sh"
	topic  string
	client *http.Client
}

// New returns a ready Notifier. url is the ntfy server base (no trailing slash),
// topic is the target topic name.
func New(url, topic string) *Notifier {
	return &Notifier{
		url:    strings.TrimRight(url, "/"),
		topic:  topic,
		client: &http.Client{},
	}
}

// Name returns "ntfy".
func (n *Notifier) Name() string { return "ntfy" }

// Notify POSTs notification.Body to {url}/{topic} with Title and Priority
// headers mapped from the canonical notification type.
func (n *Notifier) Notify(ctx context.Context, msg types.Notification) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		n.url+"/"+n.topic,
		strings.NewReader(msg.Body),
	)
	if err != nil {
		return fmt.Errorf("ntfy: build request: %w", err)
	}

	req.Header.Set("Title", msg.Title)
	req.Header.Set("Priority", priorityString(msg.Priority))

	resp, err := n.client.Do(req)
	if err != nil {
		return fmt.Errorf("ntfy: post: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("ntfy: unexpected status %d", resp.StatusCode)
	}
	return nil
}

// priorityString maps a canonical priority to the ntfy header value.
// ntfy accepts: min(1), low(2), default(3), high(4), urgent(5).
func priorityString(p types.NotificationPriority) string {
	switch p {
	case types.PriorityLow:
		return "low"
	case types.PriorityHigh:
		return "high"
	default:
		return "default"
	}
}
