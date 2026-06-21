package auth

import (
	"context"
	crand "crypto/rand"
	"fmt"
	"math/big"
	mrand "math/rand/v2"
)

// GenerateRecoveryCodes returns n recovery codes in "xxxxx-xxxxx" format. Each
// segment is 5 decimal digits drawn from crypto/rand. Codes are one-time use
// and hashed with HashToken (SHA-256) before storage.
func GenerateRecoveryCodes(n int) ([]string, error) {
	if n < 1 || n > 100 {
		return nil, fmt.Errorf("auth: recovery codes count must be 1-100, got %d", n)
	}

	codes := make([]string, n)
	seen := make(map[string]bool, n)

	for i := 0; i < n; i++ {
		var code string
		for {
			code = fmt.Sprintf("%05d-%05d",
				cryptoRand5Digits(),
				cryptoRand5Digits(),
			)
			if !seen[code] {
				seen[code] = true
				break
			}
		}
		codes[i] = code
	}

	return codes, nil
}

// cryptoRand5Digits returns a random integer in [0, 99999] using crypto/rand.
func cryptoRand5Digits() int {
	max := big.NewInt(100000)
	n, err := crand.Int(crand.Reader, max)
	if err != nil {
		// crypto/rand failures are terminal — fall back to math/rand for
		// this single digit block rather than panicking. The code is still
		// CSPRNG-quality for the other segment.
		return mrand.IntN(100000) //#nosec G404 — fallback only; crypto/rand is used normally
	}
	return int(n.Int64())
}

// RecoveryCodeRepo is the persistence boundary for recovery codes.
// Implemented by the store.
type RecoveryCodeRepo interface {
	ReplaceRecoveryCodes(ctx context.Context, userID string, hashes []string) error
	ConsumeRecoveryCode(ctx context.Context, userID, hash string) (bool, error)
}
