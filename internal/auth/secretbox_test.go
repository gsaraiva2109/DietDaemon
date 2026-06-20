package auth

import (
	"bytes"
	"testing"
)

func TestEncryptDecryptRoundtrip(t *testing.T) {
	key := make([]byte, 32)
	for i := range key {
		key[i] = byte(i)
	}

	plain := []byte("TOTP secret: JBSWY3DPEHPK3PXP")
	ct, err := Encrypt(plain, key)
	if err != nil {
		t.Fatalf("Encrypt: %v", err)
	}

	got, err := Decrypt(ct, key)
	if err != nil {
		t.Fatalf("Decrypt: %v", err)
	}

	if !bytes.Equal(got, plain) {
		t.Fatalf("roundtrip failed: got %q, want %q", got, plain)
	}

	// Verify ciphertext is not the plaintext.
	if bytes.Equal(ct, plain) {
		t.Fatal("ciphertext should not equal plaintext")
	}
}

func TestDecryptWrongKey(t *testing.T) {
	key := make([]byte, 32)
	plain := []byte("sensitive data")
	ct, err := Encrypt(plain, key)
	if err != nil {
		t.Fatalf("Encrypt: %v", err)
	}

	wrongKey := make([]byte, 32)
	wrongKey[0] = 0xFF

	_, err = Decrypt(ct, wrongKey)
	if err == nil {
		t.Fatal("expected error with wrong key, got nil")
	}
}

func TestDecryptTamperedCiphertext(t *testing.T) {
	key := make([]byte, 32)
	plain := []byte("sensitive data")
	ct, err := Encrypt(plain, key)
	if err != nil {
		t.Fatalf("Encrypt: %v", err)
	}

	// Tamper with a byte in the ciphertext portion.
	ct[len(ct)-1] ^= 0x01

	_, err = Decrypt(ct, key)
	if err == nil {
		t.Fatal("expected error with tampered ciphertext, got nil")
	}
}

func TestEncryptBadKeySize(t *testing.T) {
	_, err := Encrypt([]byte("x"), make([]byte, 16))
	if err == nil {
		t.Fatal("expected error with 16-byte key, got nil")
	}
}

func TestDecryptBadKeySize(t *testing.T) {
	_, err := Decrypt([]byte("x"), make([]byte, 16))
	if err == nil {
		t.Fatal("expected error with 16-byte key, got nil")
	}
}

func TestDecryptTooShort(t *testing.T) {
	key := make([]byte, 32)
	_, err := Decrypt([]byte("short"), key)
	if err == nil {
		t.Fatal("expected error with short ciphertext, got nil")
	}
}
