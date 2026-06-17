// Package gotify implements ports.Notifier for Gotify (https://gotify.net),
// a self-hosted push notification server.
package gotify

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/gsaraiva2109/dietdaemon/core/ports"
	"github.com/gsaraiva2109/dietdaemon/core/types"
)

// Compile-time interface check.
var _ ports.Notifier = (*Notifier)(nil)

// Notifier delivers push notifications via a Gotify server.
type Notifier struct {
	url    string // base URL, e.g. "https://gotify.example.com"
	token  string // app token
	client *http.Client
}

// New returns a ready Notifier. url is the Gotify server base URL (no trailing
// slash), token is an app token created in the Gotify UI.
func New(url, token string) *Notifier {
	return &Notifier{
		url:    strings.TrimRight(url, "/"),
		token:  token,
		client: &http.Client{},
	}
}

// Name returns "gotify".
func (n *Notifier) Name() string { return "gotify" }

// message is the JSON body expected by the Gotify /message endpoint.
type message struct {
	Title    string `json:"title"`
	Message  string `json:"message"`
	Priority int    `json:"priority"`
}

// Notify POSTs a JSON message to {url}/message?token=<token> with the
// canonical notification fields mapped to the Gotify schema.
func (n *Notifier) Notify(ctx context.Context, msg types.Notification) error {
	body := message{
		Title:    msg.Title,
		Message:  msg.Body,
		Priority: priorityInt(msg.Priority),
	}

	payload, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("gotify: marshal: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		n.url+"/message?token="+n.token,
		bytes.NewReader(payload),
	)
	if err != nil {
		return fmt.Errorf("gotify: build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := n.client.Do(req)
	if err != nil {
		return fmt.Errorf("gotify: post: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("gotify: unexpected status %d", resp.StatusCode)
	}
	return nil
}

// priorityInt maps a canonical priority to a Gotify integer (0–10).
// Gotify convention: 0=pssst, 2=low, 5=normal, 8=high, 10=urgent.
func priorityInt(p types.NotificationPriority) int {
	switch p {
	case types.PriorityLow:
		return 2
	case types.PriorityHigh:
		return 8
	default:
		return 5
	}
}
