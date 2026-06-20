// Package mailer abstracts email sending behind a single interface, with four
// backends selected by EMAIL_PROVIDER: resend, ses, smtp, and none (dev/homelab).
package mailer

import (
	"context"
	"fmt"
	"strconv"
	"strings"
)

// Message is a rendered email ready to send.
type Message struct {
	Subject  string
	HTMLBody string
	TextBody string
}

// Mailer sends email messages.
type Mailer interface {
	Send(ctx context.Context, to string, m Message) error
}

// Config selects and configures the mailer backend.
type Config struct {
	Provider      string // "resend" | "ses" | "smtp" | "none"
	From          string
	ResendAPIKey  string
	SESRegion     string
	SMTPHost      string
	SMTPPort      int
	SMTPUsername  string
	SMTPPassword  string
	SMTPTLS       bool
	PublicBaseURL string // for logging links when provider=none
}

// New returns a Mailer for the configured provider, or an error if the
// configuration is invalid.
func New(cfg Config) (Mailer, error) {
	switch strings.ToLower(cfg.Provider) {
	case "resend":
		if cfg.ResendAPIKey == "" {
			return nil, fmt.Errorf("mailer: RESEND_API_KEY is required for provider=resend")
		}
		if cfg.From == "" {
			return nil, fmt.Errorf("mailer: EMAIL_FROM is required")
		}
		return newResend(cfg.From, cfg.ResendAPIKey), nil

	case "ses":
		if cfg.From == "" {
			return nil, fmt.Errorf("mailer: EMAIL_FROM is required")
		}
		return newSES(cfg.From, cfg.SESRegion), nil

	case "smtp":
		if cfg.SMTPHost == "" {
			return nil, fmt.Errorf("mailer: SMTP_HOST is required for provider=smtp")
		}
		if cfg.From == "" {
			return nil, fmt.Errorf("mailer: EMAIL_FROM is required")
		}
		port := cfg.SMTPPort
		if port == 0 {
			port = 587
		}
		return newSMTP(cfg.From, cfg.SMTPHost, port, cfg.SMTPUsername, cfg.SMTPPassword, cfg.SMTPTLS), nil

	case "none", "":
		return newNone(cfg.PublicBaseURL), nil

	default:
		return nil, fmt.Errorf("mailer: unknown EMAIL_PROVIDER %q — valid: resend, ses, smtp, none", cfg.Provider)
	}
}

// smtpPortOrDefault returns the port as a string for net/smtp.
func smtpPortOrDefault(port int) string {
	if port <= 0 {
		return "587"
	}
	return strconv.Itoa(port)
}
