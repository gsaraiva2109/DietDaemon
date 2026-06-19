package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
)

// NewToken returns 32 cryptographically random bytes encoded as base64url
// without padding. Suitable for session cookies and opaque tokens.
func NewToken() string {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		panic(fmt.Sprintf("auth: crypto/rand.Read failed: %v", err))
	}
	return base64.RawURLEncoding.EncodeToString(b)
}

// HashToken returns the SHA-256 hex digest of tok. Used for storing session
// IDs and API keys so a DB leak cannot replay either.
func HashToken(tok string) string {
	h := sha256.Sum256([]byte(tok))
	return hex.EncodeToString(h[:])
}

// NewAPIKey returns a raw key (prefixed "ddk_") and its SHA-256 hash for
// storage. The raw key is shown once to the user; only the hash is persisted.
func NewAPIKey() (raw, hashed string) {
	raw = "ddk_" + NewToken()
	hashed = HashToken(raw)
	return raw, hashed
}
