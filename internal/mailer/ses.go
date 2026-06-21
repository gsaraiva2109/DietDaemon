package mailer

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/sesv2"
	"github.com/aws/aws-sdk-go-v2/service/sesv2/types"
)

type sesMailer struct {
	client *sesv2.Client
	from   string
}

func newSES(from, region string) *sesMailer {
	// Use the default AWS credential chain (env, ~/.aws, etc.).
	cfg, err := config.LoadDefaultConfig(context.Background())
	if err != nil {
		// Return a mailer that fails on Send — config loading is lazy here
		// because we don't want to fatal at boot when provider=ses but the
		// instance hasn't been configured yet (e.g. during local dev). The
		// first Send will surface the error.
		return &sesMailer{from: from}
	}

	if region != "" {
		cfg.Region = region
	}

	return &sesMailer{
		client: sesv2.NewFromConfig(cfg),
		from:   from,
	}
}

func (m *sesMailer) Send(ctx context.Context, to string, msg Message) error {
	if m.client == nil {
		return fmt.Errorf("mailer/ses: client not initialized — check AWS credentials")
	}

	input := &sesv2.SendEmailInput{
		FromEmailAddress: aws.String(m.from),
		Destination: &types.Destination{
			ToAddresses: []string{to},
		},
		Content: &types.EmailContent{
			Simple: &types.Message{
				Subject: &types.Content{
					Data:    aws.String(msg.Subject),
					Charset: aws.String("UTF-8"),
				},
				Body: &types.Body{
					Html: &types.Content{
						Data:    aws.String(msg.HTMLBody),
						Charset: aws.String("UTF-8"),
					},
					Text: &types.Content{
						Data:    aws.String(msg.TextBody),
						Charset: aws.String("UTF-8"),
					},
				},
			},
		},
	}

	_, err := m.client.SendEmail(ctx, input)
	if err != nil {
		return fmt.Errorf("mailer/ses: %w", err)
	}
	return nil
}
