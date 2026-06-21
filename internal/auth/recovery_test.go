package auth

import (
	"testing"
)

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
