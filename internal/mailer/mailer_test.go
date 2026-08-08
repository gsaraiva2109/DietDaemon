package mailer

import (
	"context"
	"log/slog"
	"strings"
	"testing"
)

// captureHandler is a minimal slog.Handler that records every record passed
// to it, honoring a configurable minimum level like a real handler would.
type captureHandler struct {
	level   slog.Level
	records []slog.Record
}

func (h *captureHandler) Enabled(_ context.Context, level slog.Level) bool {
	return level >= h.level
}

func (h *captureHandler) Handle(_ context.Context, r slog.Record) error {
	h.records = append(h.records, r)
	return nil
}

func (h *captureHandler) WithAttrs(_ []slog.Attr) slog.Handler { return h }
func (h *captureHandler) WithGroup(_ string) slog.Handler      { return h }

func TestNew(t *testing.T) {
	tests := []struct {
		name    string
		cfg     Config
		wantErr bool
	}{
		{
			name:    "none provider",
			cfg:     Config{Provider: "none"},
			wantErr: false,
		},
		{
			name:    "empty provider defaults to none",
			cfg:     Config{Provider: ""},
			wantErr: false,
		},
		{
			name:    "resend without api key",
			cfg:     Config{Provider: "resend", From: "test@example.com"},
			wantErr: true,
		},
		{
			name:    "resend without from",
			cfg:     Config{Provider: "resend", ResendAPIKey: "re_xxx"},
			wantErr: true,
		},
		{
			name: "resend valid",
			cfg:  Config{Provider: "resend", From: "test@example.com", ResendAPIKey: "re_xxx"},
		},
		{
			name:    "ses without from",
			cfg:     Config{Provider: "ses"},
			wantErr: true,
		},
		{
			name: "ses valid",
			cfg:  Config{Provider: "ses", From: "test@example.com"},
		},
		{
			name:    "smtp without host",
			cfg:     Config{Provider: "smtp", From: "test@example.com"},
			wantErr: true,
		},
		{
			name:    "smtp without from",
			cfg:     Config{Provider: "smtp", SMTPHost: "smtp.example.com"},
			wantErr: true,
		},
		{
			name: "smtp valid",
			cfg:  Config{Provider: "smtp", From: "test@example.com", SMTPHost: "smtp.example.com"},
		},
		{
			name:    "unknown provider",
			cfg:     Config{Provider: "sendgrid"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m, err := New(tt.cfg)
			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if m == nil {
				t.Error("expected non-nil Mailer")
			}
		})
	}
}

func TestNoneMailerSend(t *testing.T) {
	m := newNone("http://localhost:8080")
	err := m.Send(t.Context(), "test@example.com", VerificationEmail("http://localhost:8080/verify?token=abc"))
	if err != nil {
		t.Errorf("none mailer should never error: %v", err)
	}
}

// TestNoneMailerSendLogsAtDebug pins the #276 item 4 fix: the none mailer
// must log at Debug, not Info, since the body carries raw secrets
// (password-reset tokens, magic-signin links). A handler configured at the
// common production minimum (Info) must see nothing; one at Debug must see
// the record.
func TestNoneMailerSendLogsAtDebug(t *testing.T) {
	prev := slog.Default()
	defer slog.SetDefault(prev)

	m := newNone("http://localhost:8080")
	msg := PasswordResetEmail("http://localhost:8080/reset?token=SECRET123")

	infoHandler := &captureHandler{level: slog.LevelInfo}
	slog.SetDefault(slog.New(infoHandler))
	if err := m.Send(t.Context(), "test@example.com", msg); err != nil {
		t.Fatalf("Send: %v", err)
	}
	if len(infoHandler.records) != 0 {
		t.Errorf("Info-level handler captured %d record(s); the secret-bearing body must not log at Info", len(infoHandler.records))
	}

	debugHandler := &captureHandler{level: slog.LevelDebug}
	slog.SetDefault(slog.New(debugHandler))
	if err := m.Send(t.Context(), "test@example.com", msg); err != nil {
		t.Fatalf("Send: %v", err)
	}
	if len(debugHandler.records) != 1 {
		t.Fatalf("Debug-level handler captured %d record(s), want 1", len(debugHandler.records))
	}
	if got := debugHandler.records[0].Level; got != slog.LevelDebug {
		t.Errorf("record level = %v, want Debug", got)
	}
}

func TestTemplatesNotEmpty(t *testing.T) {
	v := VerificationEmail("http://example.com/verify?t=abc")
	if v.Subject == "" || v.HTMLBody == "" || v.TextBody == "" {
		t.Error("verification email template should have subject, html, and text")
	}

	r := PasswordResetEmail("http://example.com/reset?t=abc")
	if r.Subject == "" || r.HTMLBody == "" || r.TextBody == "" {
		t.Error("password reset email template should have subject, html, and text")
	}

	link := "http://example.com/reactivate?t=abc"
	d := AccountDeletionRequestedEmail(link)
	if d.Subject == "" || d.HTMLBody == "" || d.TextBody == "" {
		t.Error("account deletion requested email template should have subject, html, and text")
	}
	if !strings.Contains(d.HTMLBody, link) || !strings.Contains(d.TextBody, link) {
		t.Error("account deletion requested email should include the reactivation link")
	}

	a := AccountReactivatedEmail()
	if a.Subject == "" || a.HTMLBody == "" || a.TextBody == "" {
		t.Error("account reactivated email template should have subject, html, and text")
	}
}
