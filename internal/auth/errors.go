// Package auth implements authentication primitives: password hashing,
// opaque token generation, server-side session lifecycle, CSRF protection,
// brute-force lockout, and an audit trail. It is pure (no database/sql)
// so every component is trivially unit-testable; persistence lives in
// the store layer.
package auth

import "errors"

// Sentinels returned by the auth package. Callers use errors.Is to branch.
var (
	// ErrInvalidCredentials is returned for any authentication failure.
	// It MUST be used for BOTH unknown-email and wrong-password — never
	// reveal which field was wrong.
	ErrInvalidCredentials = errors.New("invalid email or password")

	// ErrLocked is returned when the account or IP is locked due to
	// too many failed attempts.
	ErrLocked = errors.New("account temporarily locked")

	// ErrRegistrationClosed is returned by register when new sign-ups
	// are not allowed (invite mode after bootstrap, or oidc-only).
	ErrRegistrationClosed = errors.New("registration is closed")

	// ErrEmailTaken is returned when a registration email already exists
	// within the same account. Unlike login errors, this CAN be revealed
	// to the user (they already own the email).
	ErrEmailTaken = errors.New("email already taken")

	// ErrPasswordTooShort is returned by Hash when len(password) < 8.
	ErrPasswordTooShort = errors.New("password must be at least 8 characters")

	// ErrPasswordTooLong is returned by Hash when len(password) > 128.
	// This is a DoS guard: argon2 memory cost is proportional to input length.
	ErrPasswordTooLong = errors.New("password must be at most 128 characters")
)
