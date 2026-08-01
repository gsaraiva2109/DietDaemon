package mailer

import (
	"context"
	"fmt"

	"github.com/resend/resend-go/v3"
)

type resendMailer struct {
	client *resend.Client
	from   string
}

func newResend(from, apiKey string) *resendMailer {
	return &resendMailer{
		client: resend.NewClient(apiKey),
		from:   from,
	}
}

func (m *resendMailer) Send(ctx context.Context, to string, msg Message) error {
	params := &resend.SendEmailRequest{
		From:    m.from,
		To:      []string{to},
		Subject: msg.Subject,
		Html:    msg.HTMLBody,
		Text:    msg.TextBody,
	}

	_, err := m.client.Emails.SendWithContext(ctx, params)
	if err != nil {
		return fmt.Errorf("mailer/resend: %w", err)
	}
	return nil
}
