package auth

import (
	"testing"
)

func TestGenerateSecret(t *testing.T) {
	secret, url, err := GenerateSecret("DietDaemon", "test@example.com")
	if err != nil {
		t.Fatalf("GenerateSecret: %v", err)
	}
	if secret == "" {
		t.Fatal("expected non-empty secret")
	}
	if url == "" {
		t.Fatal("expected non-empty otpauth URL")
	}

	// otpauth:// URL should include the issuer and account.
	if !contains(url, "DietDaemon") {
		t.Fatalf("expected URL to contain issuer, got %q", url)
	}
	if !contains(url, "test@example.com") {
		t.Fatalf("expected URL to contain account, got %q", url)
	}
}

func TestValidateCode(t *testing.T) {
	secret, _, err := GenerateSecret("DietDaemon", "test@example.com")
	if err != nil {
		t.Fatalf("GenerateSecret: %v", err)
	}

	// A random 6-digit code almost certainly won't validate.
	if ValidateCode(secret, "000000") {
		// Statistically possible but astronomically unlikely with a real TOTP.
		t.Log("unexpected: 000000 validated (1 in 1,000,000 chance)")
	}

	// ValidateCode should return false for non-numeric input.
	if ValidateCode(secret, "abcdef") {
		t.Fatal("non-numeric input should not validate")
	}
}

func TestValidateCodeEmptySecret(t *testing.T) {
	if ValidateCode("", "123456") {
		t.Fatal("empty secret should not validate any code")
	}
}

func TestGenerateSecretEmptyIssuer(t *testing.T) {
	_, _, err := GenerateSecret("", "test@example.com")
	if err == nil {
		t.Fatal("expected error for empty issuer")
	}
}

func TestGenerateSecretEmptyAccount(t *testing.T) {
	_, _, err := GenerateSecret("DietDaemon", "")
	if err == nil {
		t.Fatal("expected error for empty account name")
	}
}

func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
