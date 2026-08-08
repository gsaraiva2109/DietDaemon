package auth

import (
	crand "crypto/rand"
	"errors"
	"strings"
	"testing"
)

// failingReader is an io.Reader that always errors, used to force
// crypto/rand.Int into its failure path.
type failingReader struct{}

func (failingReader) Read(p []byte) (int, error) {
	return 0, errors.New("simulated crypto/rand failure")
}

// TestCryptoRand5DigitsPanicsOnRandFailure pins the fix for #276 item 1:
// cryptoRand5Digits must panic (matching token.go/webauthn.go) rather than
// silently falling back to math/rand/v2 when crypto/rand fails.
func TestCryptoRand5DigitsPanicsOnRandFailure(t *testing.T) {
	orig := crand.Reader
	crand.Reader = failingReader{}
	defer func() { crand.Reader = orig }()

	defer func() {
		r := recover()
		if r == nil {
			t.Fatal("expected cryptoRand5Digits to panic on crypto/rand failure")
		}
		msg, ok := r.(string)
		if !ok || !strings.Contains(msg, "crypto/rand.Int failed") {
			t.Errorf("panic value = %v, want message containing %q", r, "crypto/rand.Int failed")
		}
	}()
	cryptoRand5Digits()
}

func TestGenerateRecoveryCodesCount(t *testing.T) {
	tests := []int{1, 5, 10, 20}
	for _, n := range tests {
		codes, err := GenerateRecoveryCodes(n)
		if err != nil {
			t.Fatalf("GenerateRecoveryCodes(%d): %v", n, err)
		}
		if len(codes) != n {
			t.Fatalf("GenerateRecoveryCodes(%d): got %d codes", n, len(codes))
		}
	}
}

func TestGenerateRecoveryCodesFormat(t *testing.T) {
	codes, err := GenerateRecoveryCodes(10)
	if err != nil {
		t.Fatalf("GenerateRecoveryCodes: %v", err)
	}

	for i, code := range codes {
		if len(code) != 11 {
			t.Fatalf("code[%d] %q: expected 11 chars (xxxxx-xxxxx)", i, code)
		}
		if code[5] != '-' {
			t.Fatalf("code[%d] %q: expected dash at position 5", i, code)
		}
	}
}

func TestGenerateRecoveryCodesUniqueness(t *testing.T) {
	codes, err := GenerateRecoveryCodes(100)
	if err != nil {
		t.Fatalf("GenerateRecoveryCodes: %v", err)
	}

	seen := make(map[string]bool)
	for _, code := range codes {
		if seen[code] {
			t.Fatalf("duplicate code: %q", code)
		}
		seen[code] = true
	}
}

func TestGenerateRecoveryCodesHashRoundtrip(t *testing.T) {
	codes, err := GenerateRecoveryCodes(10)
	if err != nil {
		t.Fatalf("GenerateRecoveryCodes: %v", err)
	}

	// Hash each code — the resulting hashes should be unique and stable.
	hashes := make(map[string]string)
	for _, code := range codes {
		h := HashToken(code)
		if h == "" {
			t.Fatalf("HashToken(%q) returned empty", code)
		}
		hashes[code] = h
	}

	// Hashing the same code should produce the same hash.
	for code, expected := range hashes {
		actual := HashToken(code)
		if actual != expected {
			t.Fatalf("HashToken(%q) not stable: %s vs %s", code, expected, actual)
		}
	}
}

func TestGenerateRecoveryCodesInvalidCount(t *testing.T) {
	tests := []int{0, -1, 101}
	for _, n := range tests {
		_, err := GenerateRecoveryCodes(n)
		if err == nil {
			t.Fatalf("GenerateRecoveryCodes(%d): expected error, got nil", n)
		}
	}
}
