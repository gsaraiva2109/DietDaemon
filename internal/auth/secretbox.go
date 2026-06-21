package auth

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"fmt"
)

// Encrypt encrypts plaintext using AES-256-GCM with a random nonce. The returned
// ciphertext is nonce || ciphertext. key must be exactly 32 bytes.
func Encrypt(plaintext, key []byte) ([]byte, error) {
	if len(key) != 32 {
		return nil, fmt.Errorf("auth: secretbox key must be 32 bytes, got %d", len(key))
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("auth: aes new cipher: %w", err)
	}

	aesgcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("auth: aes-gcm: %w", err)
	}

	nonce := make([]byte, aesgcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return nil, fmt.Errorf("auth: nonce: %w", err)
	}

	ct := aesgcm.Seal(nonce, nonce, plaintext, nil)
	return ct, nil
}

// Decrypt decrypts ciphertext produced by Encrypt (nonce || ciphertext). key must
// be exactly 32 bytes. Returns an error on authentication failure or wrong key.
func Decrypt(ciphertext, key []byte) ([]byte, error) {
	if len(key) != 32 {
		return nil, fmt.Errorf("auth: secretbox key must be 32 bytes, got %d", len(key))
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("auth: aes new cipher: %w", err)
	}

	aesgcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("auth: aes-gcm: %w", err)
	}

	nonceSize := aesgcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return nil, fmt.Errorf("auth: ciphertext too short")
	}

	nonce, ct := ciphertext[:nonceSize], ciphertext[nonceSize:]
	plaintext, err := aesgcm.Open(nil, nonce, ct, nil)
	if err != nil {
		return nil, fmt.Errorf("auth: decrypt: %w", err)
	}

	return plaintext, nil
}
