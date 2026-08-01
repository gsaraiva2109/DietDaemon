package mailer

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
)

func TestResendMailerSend(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/emails" {
			t.Fatalf("request = %s %s, want POST /emails", r.Method, r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer re_test" {
			t.Fatalf("Authorization = %q, want Bearer re_test", got)
		}

		var got struct {
			From    string   `json:"from"`
			To      []string `json:"to"`
			Subject string   `json:"subject"`
			HTML    string   `json:"html"`
			Text    string   `json:"text"`
		}
		if err := json.NewDecoder(r.Body).Decode(&got); err != nil {
			t.Fatal(err)
		}
		if got.From != "from@example.com" || len(got.To) != 1 || got.To[0] != "to@example.com" || got.Subject != "Subject" || got.HTML != "<p>HTML</p>" || got.Text != "Text" {
			t.Fatalf("email request = %#v", got)
		}
		_, _ = w.Write([]byte(`{"id":"email-id"}`))
	}))
	defer server.Close()

	baseURL, err := url.Parse(server.URL + "/")
	if err != nil {
		t.Fatal(err)
	}
	m := newResend("from@example.com", "re_test")
	m.client.BaseURL = baseURL
	if err := m.Send(t.Context(), "to@example.com", Message{Subject: "Subject", HTMLBody: "<p>HTML</p>", TextBody: "Text"}); err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	if err := m.Send(ctx, "to@example.com", Message{}); err == nil {
		t.Fatal("expected cancelled context error")
	}
}
