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
	// Debug, not Info: the body carries raw secrets (password-reset tokens,
	// magic-signin links/codes). Fine to print in a local homelab's
	// `docker logs`, but Info is commonly shipped to a shared/centralized
	// log sink where that's a real exposure. Debug is off by default there.
	slog.Debug("mailer (none): would send email",
		"to", to,
		"subject", msg.Subject,
		"html_body", msg.HTMLBody,
		"text_body", msg.TextBody,
		"public_base_url", m.publicBaseURL,
	)
	return nil
}
