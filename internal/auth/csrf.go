package auth

// CSRF implementation uses the double-submit cookie pattern implemented in
// session.go (VerifyCSRF). This file exists as the canonical location for
// future CSRF enhancements (HMAC-based tokens, Origin/Referer validation,
// per-form nonce). Phase 1 uses the simplest correct approach: the session's
// csrf_token is set as a readable (non-HttpOnly) dd_csrf cookie, echoed by
// the client in X-CSRF-Token, and compared with constant-time equality.
