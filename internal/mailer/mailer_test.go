package mailer

import (
	"testing"
)

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

func TestTemplatesNotEmpty(t *testing.T) {
	v := VerificationEmail("http://example.com/verify?t=abc")
	if v.Subject == "" || v.HTMLBody == "" || v.TextBody == "" {
		t.Error("verification email template should have subject, html, and text")
	}

	r := PasswordResetEmail("http://example.com/reset?t=abc")
	if r.Subject == "" || r.HTMLBody == "" || r.TextBody == "" {
		t.Error("password reset email template should have subject, html, and text")
	}
}
