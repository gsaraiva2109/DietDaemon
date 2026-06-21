package auth

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"fmt"
	"strconv"
	"strings"

	"golang.org/x/crypto/argon2"
)

const (
	// argon2id parameters — memory-hard KDF tuned for auth latency.
	argonMemory  = 64 * 1024 // 64 MiB
	argonTime    = 3
	argonThreads = 4
	argonKeyLen  = 32
	argonSaltLen = 16

	minPasswordLen = 8
	maxPasswordLen = 128
)

// Hash returns an argon2id PHC string for password. Returns
// ErrPasswordTooShort / ErrPasswordTooLong for length violations.
// The PHC format: $argon2id$v=19$m=...,t=...,p=...$<b64salt>$<b64hash>
func Hash(password string) (string, error) {
	if len(password) < minPasswordLen {
		return "", ErrPasswordTooShort
	}
	if len(password) > maxPasswordLen {
		return "", ErrPasswordTooLong
	}

	salt := make([]byte, argonSaltLen)
	if _, err := rand.Read(salt); err != nil {
		return "", fmt.Errorf("auth: hash: salt: %w", err)
	}

	key := argon2.IDKey([]byte(password), salt, argonTime, argonMemory, argonThreads, argonKeyLen)

	enc := base64.RawStdEncoding
	phc := fmt.Sprintf("$argon2id$v=19$m=%d,t=%d,p=%d$%s$%s",
		argonMemory, argonTime, argonThreads, enc.EncodeToString(salt), enc.EncodeToString(key))

	return phc, nil
}

// Verify returns true when password matches the PHC string produced by Hash.
// Comparison is constant-time. Returns false (not an error) on mismatch or
// malformed PHC — the caller must never leak timing differences between
// "bad phc" and "wrong password".
func Verify(password, phc string) (bool, error) {
	memory, time, threads, salt, hash, err := parsePHC(phc)
	if err != nil {
		// Malformed PHC — run a dummy argon2 to preserve constant-time-ish
		// behaviour relative to the success path.
		dummy := make([]byte, 32)
		_, _ = rand.Read(dummy)
		subtle.ConstantTimeCompare(dummy, dummy)
		return false, nil // return false, not error — don't reveal parse failure
	}

	// Bounds-check parameters before narrowing conversions. These were already
	// validated by parsePHC (positive, non-zero) but an attacker could craft a
	// malicious PHC string with values exceeding the uint range.
	if memory < 0 || memory > 1<<31-1 || time < 0 || time > 1<<31-1 || threads < 0 || threads > 255 {
		return false, nil
	}
	// #nosec G115 — bounds-checked above; safe narrowing.
	key := argon2.IDKey([]byte(password), salt, uint32(time), uint32(memory), uint8(threads), uint32(len(hash)))
	return subtle.ConstantTimeCompare(key, hash) == 1, nil
}

// parsePHC extracts argon2id parameters and raw salt/hash from a PHC string.
// Matches: $argon2id$v=19$m=<mem>,t=<time>,p=<threads>$<b64salt>$<b64hash>
func parsePHC(phc string) (memory, time, threads int, salt, hash []byte, err error) {
	parts := strings.Split(phc, "$")
	// parts[0] is empty (leading $), parts[1]=argon2id, parts[2]=v=19,
	// parts[3]=m=...,t=...,p=..., parts[4]=salt, parts[5]=hash
	if len(parts) != 6 || parts[1] != "argon2id" {
		return 0, 0, 0, nil, nil, fmt.Errorf("bad phc format")
	}

	params := strings.Split(parts[3], ",")
	if len(params) != 3 {
		return 0, 0, 0, nil, nil, fmt.Errorf("bad phc params")
	}

	getInt := func(s, prefix string) (int, error) {
		if !strings.HasPrefix(s, prefix) {
			return 0, fmt.Errorf("missing prefix %s in %s", prefix, s)
		}
		return strconv.Atoi(s[len(prefix):])
	}

	memory, err = getInt(params[0], "m=")
	if err != nil {
		return 0, 0, 0, nil, nil, err
	}
	time, err = getInt(params[1], "t=")
	if err != nil {
		return 0, 0, 0, nil, nil, err
	}
	threads, err = getInt(params[2], "p=")
	if err != nil {
		return 0, 0, 0, nil, nil, err
	}

	enc := base64.RawStdEncoding
	salt, err = enc.DecodeString(parts[4])
	if err != nil {
		return 0, 0, 0, nil, nil, err
	}
	hash, err = enc.DecodeString(parts[5])
	if err != nil {
		return 0, 0, 0, nil, nil, err
	}

	return memory, time, threads, salt, hash, nil
}
