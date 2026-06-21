// Package auth provides authentication primitives. This file wraps the
// go-webauthn library so handler code stays thin and the rest of the codebase
// has no direct dependency on go-webauthn types beyond option/response JSON.
package auth

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"

	"github.com/go-webauthn/webauthn/protocol"
	gowa "github.com/go-webauthn/webauthn/webauthn"

	"github.com/gsaraiva2109/dietdaemon/core/types"
)

// WebAuthnConfig is the RP configuration for WebAuthn operations.
type WebAuthnConfig struct {
	RPID          string
	RPDisplayName string
	RPOrigins     []string
}

// NewWebAuthn creates a go-webauthn instance from the RP config.
func NewWebAuthn(cfg WebAuthnConfig) (*gowa.WebAuthn, error) {
	return gowa.New(&gowa.Config{
		RPID:          cfg.RPID,
		RPDisplayName: cfg.RPDisplayName,
		RPOrigins:     cfg.RPOrigins,
	})
}

// WebAuthnUser wraps a types.User and its stored credentials to satisfy the
// go-webauthn User interface.
type WebAuthnUser struct {
	User        types.User
	Credentials []types.WebAuthnCredential
}

// WebAuthnID returns the decoded webauthn_handle. Must be set (non-empty)
// before calling any go-webauthn method that requires a user.
func (u WebAuthnUser) WebAuthnID() []byte {
	if u.User.WebAuthnHandle == "" {
		return nil
	}
	raw, err := base64.RawStdEncoding.DecodeString(u.User.WebAuthnHandle)
	if err != nil {
		return nil
	}
	return raw
}

// WebAuthnName returns the user's email as the account name.
func (u WebAuthnUser) WebAuthnName() string {
	return u.User.Email
}

// WebAuthnDisplayName returns the user's display name or email as fallback.
func (u WebAuthnUser) WebAuthnDisplayName() string {
	if u.User.DisplayName != "" {
		return u.User.DisplayName
	}
	return u.User.Email
}

// WebAuthnCredentials returns the stored credentials decoded into go-webauthn
// Credential structs.
func (u WebAuthnUser) WebAuthnCredentials() []gowa.Credential {
	var out []gowa.Credential
	for _, c := range u.Credentials {
		var cred gowa.Credential
		if err := json.Unmarshal([]byte(c.CredentialJSON), &cred); err != nil {
			continue
		}
		out = append(out, cred)
	}
	return out
}

// WebAuthnIcon returns "" — no user icon.
func (u WebAuthnUser) WebAuthnIcon() string {
	return ""
}

// NewWebAuthnHandle generates 32 random bytes encoded as base64 (raw std). The
// handle is stable per-user and generated lazily on first passkey registration.
func NewWebAuthnHandle() string {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		panic(fmt.Sprintf("auth: crypto/rand.Read failed: %v", err))
	}
	return base64.RawStdEncoding.EncodeToString(b)
}

// --- Serialization helpers for store layer ---

// MarshalSessionData serializes a go-webauthn SessionData to JSON.
func MarshalSessionData(sd *gowa.SessionData) (string, error) {
	b, err := json.Marshal(sd)
	if err != nil {
		return "", fmt.Errorf("webauthn: marshal session data: %w", err)
	}
	return string(b), nil
}

// UnmarshalSessionData deserializes a go-webauthn SessionData from JSON.
func UnmarshalSessionData(raw string) (*gowa.SessionData, error) {
	var sd gowa.SessionData
	if err := json.Unmarshal([]byte(raw), &sd); err != nil {
		return nil, fmt.Errorf("webauthn: unmarshal session data: %w", err)
	}
	return &sd, nil
}

// MarshalCredential serializes a go-webauthn Credential to JSON for storage.
func MarshalCredential(cred *gowa.Credential) (string, error) {
	b, err := json.Marshal(cred)
	if err != nil {
		return "", fmt.Errorf("webauthn: marshal credential: %w", err)
	}
	return string(b), nil
}

// CredentialIDBytes returns the base64-decoded credential ID for a stored
// credential, or nil on error.
func CredentialIDBytes(credJSON string) []byte {
	var cred gowa.Credential
	if err := json.Unmarshal([]byte(credJSON), &cred); err != nil {
		return nil
	}
	return cred.ID
}

// ParseCredentialCreationResponse parses a raw RegistrationResponseJSON body
// into the go-webauthn parsed form. The body is the unwrapped credential JSON
// (NOT {label, credential} — callers must extract the inner credential first).
func ParseCredentialCreationResponse(body []byte) (*protocol.ParsedCredentialCreationData, error) {
	return protocol.ParseCredentialCreationResponseBytes(body)
}

// ParseCredentialRequestResponse parses a raw AuthenticationResponseJSON body
// into the go-webauthn parsed form.
func ParseCredentialRequestResponse(body []byte) (*protocol.ParsedCredentialAssertionData, error) {
	return protocol.ParseCredentialRequestResponseBytes(body)
}
