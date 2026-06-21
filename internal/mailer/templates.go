package mailer

import "fmt"

// VerificationEmail builds the "verify your email" message for a given link.
func VerificationEmail(link string) Message {
	return Message{
		Subject:  "Verify your email — DietDaemon",
		HTMLBody: fmt.Sprintf(`<p>Welcome to DietDaemon!</p><p><a href="%s">Click here to verify your email address</a>.</p><p>Or copy this link: %s</p><p>This link expires in 24 hours.</p>`, link, link),
		TextBody: fmt.Sprintf("Welcome to DietDaemon!\n\nVerify your email: %s\n\nOr copy this link: %s\n\nThis link expires in 24 hours.", link, link),
	}
}

// MagicSigninEmail builds the "sign-in code + link" message for passwordless sign-in.
func MagicSigninEmail(link, code string) Message {
	return Message{
		Subject:  "Your sign-in code — DietDaemon",
		HTMLBody: fmt.Sprintf(`<p>Here is your sign-in code:</p><p style="font-size:32px;font-weight:bold;letter-spacing:4px;margin:16px 0">%s</p><p>Or <a href="%s">click here to sign in instantly</a>.</p><p>This code and link expire in 15 minutes. If you didn't request this, you can ignore it.</p>`, code, link),
		TextBody: fmt.Sprintf("Your sign-in code: %s\n\nOr use this link: %s\n\nThis code and link expire in 15 minutes. If you didn't request this, you can ignore it.", code, link),
	}
}

// PasswordResetEmail builds the "reset your password" message for a given link.
func PasswordResetEmail(link string) Message {
	return Message{
		Subject:  "Reset your password — DietDaemon",
		HTMLBody: fmt.Sprintf(`<p>A password reset was requested for your account.</p><p><a href="%s">Click here to reset your password</a>.</p><p>Or copy this link: %s</p><p>This link expires in 1 hour. If you didn't request this, you can ignore it — your account is still secure.</p>`, link, link),
		TextBody: fmt.Sprintf("A password reset was requested for your account.\n\nReset your password: %s\n\nOr copy this link: %s\n\nThis link expires in 1 hour. If you didn't request this, you can ignore it.", link, link),
	}
}
