package auth

import (
	"context"
	"fmt"

	"github.com/pquerna/otp/totp"
)

// GenerateSecret creates a new TOTP key using HMAC-SHA1 (Google Authenticator
// compatible). Returns the base32-encoded secret and the otpauth:// provisioning
// URL for QR code rendering.
func GenerateSecret(issuer, accountName string) (secret, otpauthURL string, err error) {
	key, err := totp.Generate(totp.GenerateOpts{
		Issuer:      issuer,
		AccountName: accountName,
	})
	if err != nil {
		return "", "", fmt.Errorf("auth: generate totp: %w", err)
	}
	return key.Secret(), key.URL(), nil
}

// ValidateCode checks whether code is a valid TOTP code for secret, allowing
// ±1 period skew (30s windows). Code comparison is against both the current
// period and the adjacent periods to tolerate minor clock drift.
func ValidateCode(secret, code string) bool {
	return totp.Validate(code, secret)
}

// --- Repo interfaces (satisfied by *store.Store) ---

// TOTPRepo is the persistence boundary for TOTP secrets. The store holds the
// AES-256-GCM encrypted secret; encryption/decryption happens above this layer.
type TOTPRepo interface {
	UpsertTOTPSecret(ctx context.Context, userID, encSecret string) error
	ConfirmTOTP(ctx context.Context, userID string) error
	GetTOTPSecret(ctx context.Context, userID string) (secret string, confirmed bool, err error)
	DeleteTOTP(ctx context.Context, userID string) error
	HasConfirmedTOTP(ctx context.Context, userID string) (bool, error)
}

// MFAChallengeRepo is the persistence boundary for MFA challenge tokens.
// Challenges are short-lived (5 min) and hashed before storage.
type MFAChallengeRepo interface {
	CreateMFAChallenge(ctx context.Context, id, userID string, remember bool, expiresAt string) error
	GetMFAChallenge(ctx context.Context, id string) (userID string, remember bool, expiresAt string, err error)
	DeleteMFAChallenge(ctx context.Context, id string) error
}
