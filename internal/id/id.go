// Package id generates opaque identifiers.
package id

import (
	"crypto/rand"
	"encoding/hex"
)

// New returns a hex-encoded 128-bit random identifier.
func New() string {
	var b [16]byte
	_, _ = rand.Read(b[:])
	return hex.EncodeToString(b[:])
}
