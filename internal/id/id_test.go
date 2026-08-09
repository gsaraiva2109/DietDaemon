package id

import (
	"encoding/hex"
	"testing"
)

func TestNew(t *testing.T) {
	got := New()
	if len(got) != 32 {
		t.Fatalf("New() length = %d, want 32", len(got))
	}
	if _, err := hex.DecodeString(got); err != nil {
		t.Fatalf("New() = %q, want hexadecimal: %v", got, err)
	}
}
