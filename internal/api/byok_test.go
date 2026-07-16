package api

import (
	"encoding/base64"
	"testing"
	"time"

	"github.com/gsaraiva2109/dietdaemon/internal/auth"
)

func TestDecryptAIKey(t *testing.T) {
	key := make([]byte, 32)
	ct, err := auth.Encrypt([]byte("secret-api-key"), key)
	if err != nil {
		t.Fatalf("Encrypt: %v", err)
	}

	got, err := decryptAIKey(base64.RawStdEncoding.EncodeToString(ct), key)
	if err != nil || string(got) != "secret-api-key" {
		t.Fatalf("decryptAIKey = %q, %v", got, err)
	}
	if _, err := decryptAIKey("not-base64!", key); err == nil {
		t.Error("decryptAIKey accepted invalid base64")
	}
}

func TestBuildBYOKAdaptersRejectUnsupportedProvider(t *testing.T) {
	if _, err := buildAdapterForProvider("ollama", "key", "", "", "", time.Second); err == nil {
		t.Error("buildAdapterForProvider accepted unsupported provider")
	}
	if _, err := buildChatAdapterForProvider("ollama", "key", "", "", "", time.Second); err == nil {
		t.Error("buildChatAdapterForProvider accepted unsupported provider")
	}
}
