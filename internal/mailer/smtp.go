package mailer

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/smtp"
	"strings"
)

type smtpMailer struct {
	from     string
	host     string
	port     string
	username string
	password string
	tls      bool
}

func newSMTP(from, host string, port int, username, password string, useTLS bool) *smtpMailer {
	return &smtpMailer{
		from:     from,
		host:     host,
		port:     smtpPortOrDefault(port),
		username: username,
		password: password,
		tls:      useTLS,
	}
}

func (m *smtpMailer) Send(ctx context.Context, to string, msg Message) error {
	addr := net.JoinHostPort(m.host, m.port)

	// Build the message with headers.
	var sb strings.Builder
	fmt.Fprintf(&sb, "From: %s\r\n", m.from)
	fmt.Fprintf(&sb, "To: %s\r\n", to)
	fmt.Fprintf(&sb, "Subject: %s\r\n", msg.Subject)
	sb.WriteString("MIME-Version: 1.0\r\n")
	sb.WriteString("Content-Type: text/plain; charset=\"UTF-8\"\r\n")
	sb.WriteString("\r\n")
	sb.WriteString(msg.TextBody)

	// Dial with context-aware timeout via the stdlib.
	conn, err := tls.Dial("tcp", addr, &tls.Config{ServerName: m.host})
	if err != nil {
		return fmt.Errorf("mailer/smtp: dial: %w", err)
	}
	defer func() { _ = conn.Close() }()

	client, err := smtp.NewClient(conn, m.host)
	if err != nil {
		return fmt.Errorf("mailer/smtp: new client: %w", err)
	}
	defer func() { _ = client.Quit() }()

	// Auth when credentials are provided.
	if m.username != "" || m.password != "" {
		auth := smtp.PlainAuth("", m.username, m.password, m.host)
		if err := client.Auth(auth); err != nil {
			return fmt.Errorf("mailer/smtp: auth: %w", err)
		}
	}

	if err := client.Mail(m.from); err != nil {
		return fmt.Errorf("mailer/smtp: mail from: %w", err)
	}
	if err := client.Rcpt(to); err != nil {
		return fmt.Errorf("mailer/smtp: rcpt to: %w", err)
	}

	wc, err := client.Data()
	if err != nil {
		return fmt.Errorf("mailer/smtp: data: %w", err)
	}
	defer wc.Close()

	if _, err := fmt.Fprint(wc, sb.String()); err != nil {
		return fmt.Errorf("mailer/smtp: write: %w", err)
	}

	return nil
}
