package mailer

import (
	"context"
	"log/slog"
)

// noneMailer logs email content to stdout — the dev/homelab stand-in.
type noneMailer struct {
	publicBaseURL string
}

func newNone(publicBaseURL string) *noneMailer {
	return &noneMailer{publicBaseURL: publicBaseURL}
}

func (m *noneMailer) Send(ctx context.Context, to string, msg Message) error {
	slog.Info("mailer (none): would send email",
		"to", to,
		"subject", msg.Subject,
		"html_body", msg.HTMLBody,
		"text_body", msg.TextBody,
		"public_base_url", m.publicBaseURL,
	)
	return nil
}
