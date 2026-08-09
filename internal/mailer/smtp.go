package mailer

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/smtp"
	"strconv"
	"strings"
	"time"
)

const smtpDialTimeout = 10 * time.Second

type smtpMailer struct {
	from     string
	host     string
	port     string
	username string
	password string
	tls      bool

	// insecureSkipVerify is only ever set by tests, to point Send at a
	// local server presenting a self-signed cert. Zero value (false) in
	// production, since newSMTP never sets it.
	insecureSkipVerify bool
}

func newSMTP(from, host string, port int, username, password string, useTLS bool) *smtpMailer {
	return &smtpMailer{
		from:     from,
		host:     host,
		port:     strconv.Itoa(port),
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

	// Dial with a context-aware timeout, then bound the whole conversation
	// (handshake + MAIL/RCPT/DATA/QUIT) with a conn-level deadline — an
	// unresponsive server must not hang the calling goroutine forever.
	rawConn, err := (&net.Dialer{Timeout: smtpDialTimeout}).DialContext(ctx, "tcp", addr)
	if err != nil {
		return fmt.Errorf("mailer/smtp: dial: %w", err)
	}

	deadline, ok := ctx.Deadline()
	if !ok {
		deadline = time.Now().Add(smtpDialTimeout)
	}
	if err := rawConn.SetDeadline(deadline); err != nil {
		_ = rawConn.Close()
		return fmt.Errorf("mailer/smtp: set deadline: %w", err)
	}

	conn := tls.Client(rawConn, &tls.Config{ServerName: m.host, InsecureSkipVerify: m.insecureSkipVerify}) // #nosec G402 -- only true in tests, see field doc
	defer func() { _ = conn.Close() }()
	if err := conn.HandshakeContext(ctx); err != nil {
		return fmt.Errorf("mailer/smtp: tls handshake: %w", err)
	}

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

	if _, err := fmt.Fprint(wc, sb.String()); err != nil {
		_ = wc.Close()
		return fmt.Errorf("mailer/smtp: write: %w", err)
	}

	// Close finalizes the DATA command (sends the terminating "." and reads
	// the server's response) — a failure here means the message was not
	// actually accepted, so it must not be swallowed like a plain resource
	// cleanup would be.
	if err := wc.Close(); err != nil {
		return fmt.Errorf("mailer/smtp: close: %w", err)
	}

	return nil
}
