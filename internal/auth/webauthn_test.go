package auth

import (
	"bytes"
	"encoding/base64"
	"testing"
	"time"

	gowa "github.com/go-webauthn/webauthn/webauthn"

	"github.com/gsaraiva2109/dietdaemon/core/types"
)

func TestNewWebAuthn(t *testing.T) {
	wa, err := NewWebAuthn(WebAuthnConfig{
		RPID:          "example.com",
		RPDisplayName: "DietDaemon",
		RPOrigins:     []string{"https://example.com"},
	})
	if err != nil {
		t.Fatalf("NewWebAuthn: %v", err)
	}
	if wa.Config.RPID != "example.com" {
		t.Errorf("RPID = %q, want example.com", wa.Config.RPID)
	}
	if wa.Config.RPDisplayName != "DietDaemon" {
		t.Errorf("RPDisplayName = %q, want DietDaemon", wa.Config.RPDisplayName)
	}

	if _, err := NewWebAuthn(WebAuthnConfig{RPID: "example.com"}); err == nil {
		t.Error("expected error without RP origins")
	}
}

func TestWebAuthnUser(t *testing.T) {
	handle := base64.RawStdEncoding.EncodeToString([]byte("stable user handle"))
	credJSON, err := MarshalCredential(&gowa.Credential{ID: []byte("credential-id")})
	if err != nil {
		t.Fatalf("MarshalCredential: %v", err)
	}

	u := WebAuthnUser{
		User: types.User{
			Email:          "ada@example.com",
			DisplayName:    "Ada",
			WebAuthnHandle: handle,
		},
		Credentials: []types.WebAuthnCredential{
			{CredentialJSON: credJSON},
			{CredentialJSON: "not json"},
		},
	}
	if got := u.WebAuthnID(); !bytes.Equal(got, []byte("stable user handle")) {
		t.Errorf("WebAuthnID = %q, want stable user handle", got)
	}
	if got := u.WebAuthnName(); got != "ada@example.com" {
		t.Errorf("WebAuthnName = %q, want ada@example.com", got)
	}
	if got := u.WebAuthnDisplayName(); got != "Ada" {
		t.Errorf("WebAuthnDisplayName = %q, want Ada", got)
	}
	if got := u.WebAuthnCredentials(); len(got) != 1 || !bytes.Equal(got[0].ID, []byte("credential-id")) {
		t.Errorf("WebAuthnCredentials = %#v, want one decoded credential", got)
	}
	if got := u.WebAuthnIcon(); got != "" {
		t.Errorf("WebAuthnIcon = %q, want empty", got)
	}

	if got := (WebAuthnUser{}).WebAuthnID(); got != nil {
		t.Errorf("empty WebAuthnID = %q, want nil", got)
	}
	if got := (WebAuthnUser{User: types.User{Email: "ada@example.com"}}).WebAuthnDisplayName(); got != "ada@example.com" {
		t.Errorf("fallback display name = %q, want email", got)
	}
	if got := (WebAuthnUser{User: types.User{WebAuthnHandle: "%%%"}}).WebAuthnID(); got != nil {
		t.Errorf("invalid WebAuthnID = %q, want nil", got)
	}
}

func TestNewWebAuthnHandle(t *testing.T) {
	handle := NewWebAuthnHandle()
	decoded, err := base64.RawStdEncoding.DecodeString(handle)
	if err != nil {
		t.Fatalf("DecodeString: %v", err)
	}
	if len(decoded) != 32 {
		t.Errorf("handle length = %d, want 32", len(decoded))
	}
}

func TestSessionDataSerialization(t *testing.T) {
	want := &gowa.SessionData{
		Challenge:            "challenge",
		RelyingPartyID:       "example.com",
		UserID:               []byte("user-id"),
		AllowedCredentialIDs: [][]byte{[]byte("credential-id")},
		Expires:              time.Date(2026, time.July, 16, 12, 0, 0, 0, time.UTC),
	}
	raw, err := MarshalSessionData(want)
	if err != nil {
		t.Fatalf("MarshalSessionData: %v", err)
	}
	got, err := UnmarshalSessionData(raw)
	if err != nil {
		t.Fatalf("UnmarshalSessionData: %v", err)
	}
	if got.Challenge != want.Challenge || got.RelyingPartyID != want.RelyingPartyID || !bytes.Equal(got.UserID, want.UserID) || !bytes.Equal(got.AllowedCredentialIDs[0], want.AllowedCredentialIDs[0]) || !got.Expires.Equal(want.Expires) {
		t.Errorf("session round trip = %#v, want %#v", got, want)
	}

	if _, err := UnmarshalSessionData("not json"); err == nil {
		t.Error("expected error for invalid session data")
	}
}

func TestCredentialSerializationAndIDBytes(t *testing.T) {
	want := &gowa.Credential{ID: []byte("credential-id")}
	raw, err := MarshalCredential(want)
	if err != nil {
		t.Fatalf("MarshalCredential: %v", err)
	}
	if got := CredentialIDBytes(raw); !bytes.Equal(got, want.ID) {
		t.Errorf("CredentialIDBytes = %q, want %q", got, want.ID)
	}
	if got := CredentialIDBytes("not json"); got != nil {
		t.Errorf("invalid CredentialIDBytes = %q, want nil", got)
	}
	if _, err := ParseCredentialCreationResponse([]byte("not json")); err == nil {
		t.Error("expected error for invalid credential creation response")
	}
	if _, err := ParseCredentialRequestResponse([]byte("not json")); err == nil {
		t.Error("expected error for invalid credential request response")
	}
}
