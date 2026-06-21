package mailer

import (
	"context"
	"fmt"

	"github.com/resendlabs/resend-go"
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

	_, err := m.client.Emails.Send(params)
	if err != nil {
		return fmt.Errorf("mailer/resend: %w", err)
	}
	return nil
}
