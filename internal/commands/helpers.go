package commands

import (
	"crypto/rand"
	"encoding/hex"
)

// randomID returns a hex-encoded 128-bit random identifier suitable for entity
// IDs that do not need to be human-readable or sortable.
func randomID() string {
	var b [16]byte
	_, _ = rand.Read(b[:])
	return hex.EncodeToString(b[:])
}
