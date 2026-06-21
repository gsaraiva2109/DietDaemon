package auth

import (
	"strings"
	"testing"
)

func TestNewToken(t *testing.T) {
	tok := NewToken()
	if len(tok) == 0 {
		t.Error("token is empty")
	}

	// 32 bytes → 43 base64url chars (no padding).
	if len(tok) != 43 {
		t.Errorf("token length = %d, want 43", len(tok))
	}

	// Uniqueness: two tokens should not collide.
	tok2 := NewToken()
	if tok == tok2 {
		t.Error("consecutive tokens collided")
	}
}

func TestHashToken(t *testing.T) {
	h := HashToken("test-token")
	if len(h) != 64 {
		t.Errorf("SHA-256 hex length = %d, want 64", len(h))
	}

	// Deterministic.
	if HashToken("test-token") != h {
		t.Error("HashToken is not deterministic")
	}

	// Different input → different output.
	if HashToken("other-token") == h {
		t.Error("different tokens should produce different hashes")
	}
}

func TestNewAPIKey(t *testing.T) {
	raw, hashed := NewAPIKey()

	if !strings.HasPrefix(raw, "ddk_") {
		t.Errorf("API key should start with ddk_, got: %s", raw)
	}

	if len(raw) != 4+43 {
		t.Errorf("API key length = %d, want %d", len(raw), 4+43)
	}

	if len(hashed) != 64 {
		t.Errorf("hashed key length = %d, want 64", len(hashed))
	}

	// Hash should match.
	if HashToken(raw) != hashed {
		t.Error("hashed key does not match HashToken(raw)")
	}
}
