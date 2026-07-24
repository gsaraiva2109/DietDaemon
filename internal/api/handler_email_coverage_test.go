package api

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gsaraiva2109/dietdaemon/core/types"
	"github.com/gsaraiva2109/dietdaemon/internal/auth"
)

// ---------------------------------------------------------------------------
// handleResendVerify
// ---------------------------------------------------------------------------

func TestHandleResendVerifyGetUserError(t *testing.T) {
	authStore := newEmailTestAuthStore()
	fm := &fakeMailer{}
	store := newFakeMealStore()
	store.getUserErr = errors.New("db down")

	// buildAuthTestHandler always builds a fresh fakeMealStore internally, so
	// construct directly to inject getUserErr.
	h := New(store, &fakeMealLogger{}, time.UTC, nil, nil,
		WithAuth(authStore, AuthRepos{Sessions: authStore, LoginAttempts: authStore, TOTP: authStore, MFAChallenges: authStore, RecoveryCodes: authStore}, nil, "DietDaemon", testAuthConfig()),
		WithMailer(fm, "none"),
		WithPublicBaseURL("http://localhost:8080"),
	)

	rec := doRequest(h, http.MethodPost, "/api/v1/auth/email/verify/resend", nil, nil)
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("resend status = %d, want 500: %s", rec.Code, rec.Body.String())
	}
}

func TestHandleResendVerifyAlreadyVerifiedIsNoOp(t *testing.T) {
	authStore := newEmailTestAuthStore()
	fm := &fakeMailer{}
	verifiedAt := time.Now().UTC()
	user := types.User{ID: "test-user", Email: "test@example.com", Status: "active", CreatedAt: verifiedAt, EmailVerifiedAt: &verifiedAt}
	h := buildAuthTestHandler(authStore, user, fm)

	rec := doRequest(h, http.MethodPost, "/api/v1/auth/email/verify/resend", nil, nil)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("resend status = %d, want 204: %s", rec.Code, rec.Body.String())
	}
	if len(fm.sent) != 0 {
		t.Error("must not send a verification email when already verified")
	}
}

func TestHandleResendVerifyLockout(t *testing.T) {
	authStore := newEmailTestAuthStore()
	fm := &fakeMailer{}
	user := types.User{ID: "test-user", Email: "test@example.com", Status: "active", CreatedAt: time.Now().UTC()}
	h := buildAuthTestHandler(authStore, user, fm)

	// The resend lockout uses MaxAttempts: 3.
	for range 3 {
		_ = authStore.RecordLoginAttempt(t.Context(), "resend:test-user", false)
	}

	rec := doRequest(h, http.MethodPost, "/api/v1/auth/email/verify/resend", nil, nil)
	if rec.Code != http.StatusTooManyRequests {
		t.Fatalf("resend status = %d, want 429: %s", rec.Code, rec.Body.String())
	}
	if rec.Header().Get("Retry-After") == "" {
		t.Error("expected Retry-After header")
	}
}

func TestHandleResendVerifyLockoutStoreError(t *testing.T) {
	authStore := newEmailTestAuthStore()
	authStore.recentFailedAttemptsErr = errors.New("store unavailable")
	fm := &fakeMailer{}
	user := types.User{ID: "test-user", Email: "test@example.com", Status: "active", CreatedAt: time.Now().UTC()}
	h := buildAuthTestHandler(authStore, user, fm)

	rec := doRequest(h, http.MethodPost, "/api/v1/auth/email/verify/resend", nil, nil)
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("resend status = %d, want 500: %s", rec.Code, rec.Body.String())
	}
}

func TestHandleResendVerifyCreateEmailTokenError(t *testing.T) {
	authStore := newEmailTestAuthStore()
	authStore.createEmailTokenErr = errors.New("store unavailable")
	fm := &fakeMailer{}
	user := types.User{ID: "test-user", Email: "test@example.com", Status: "active", CreatedAt: time.Now().UTC()}
	h := buildAuthTestHandler(authStore, user, fm)

	rec := doRequest(h, http.MethodPost, "/api/v1/auth/email/verify/resend", nil, nil)
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("resend status = %d, want 500: %s", rec.Code, rec.Body.String())
	}
}

func TestHandleResendVerifyMailerSendFailureStillNoContent(t *testing.T) {
	authStore := newEmailTestAuthStore()
	fm := &fakeMailer{sendErr: errors.New("smtp down")}
	user := types.User{ID: "test-user", AccountID: "acct-1", Email: "test@example.com", Status: "active", CreatedAt: time.Now().UTC()}
	h := buildAuthTestHandler(authStore, user, fm)

	rec := doRequest(h, http.MethodPost, "/api/v1/auth/email/verify/resend", nil, nil)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("resend status = %d, want 204 (best-effort send): %s", rec.Code, rec.Body.String())
	}
	foundFailAudit := false
	for _, ev := range authStore.auditEvents {
		if ev.Event == "email.verification_send_failed" {
			foundFailAudit = true
		}
	}
	if !foundFailAudit {
		t.Error("expected email.verification_send_failed audit event")
	}
}

func TestHandleResendVerifySuccess(t *testing.T) {
	authStore := newEmailTestAuthStore()
	fm := &fakeMailer{}
	user := types.User{ID: "test-user", AccountID: "acct-1", Email: "test@example.com", Status: "active", CreatedAt: time.Now().UTC()}
	h := buildAuthTestHandler(authStore, user, fm)

	rec := doRequest(h, http.MethodPost, "/api/v1/auth/email/verify/resend", nil, nil)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("resend status = %d, want 204: %s", rec.Code, rec.Body.String())
	}
	if len(fm.sent) != 1 || fm.sent[0].to != "test@example.com" {
		t.Errorf("expected verification email sent, got %#v", fm.sent)
	}
	if len(authStore.emailTokens) != 1 {
		t.Errorf("expected one email token issued, got %d", len(authStore.emailTokens))
	}
	foundAudit := false
	for _, ev := range authStore.auditEvents {
		if ev.Event == "email.verification_sent" {
			foundAudit = true
		}
	}
	if !foundAudit {
		t.Error("expected email.verification_sent audit event")
	}
}

// ---------------------------------------------------------------------------
// handleEmailChange
// ---------------------------------------------------------------------------

func TestHandleEmailChangeInvalidEmail(t *testing.T) {
	authStore := newEmailTestAuthStore()
	h := buildEmailHandler(authStore, &fakeMailer{})

	rec := doRequest(h, http.MethodPost, "/api/v1/auth/email/change", map[string]string{
		"email": "not-an-email", "current_password": "whatever",
	}, nil)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("email change status = %d, want 400: %s", rec.Code, rec.Body.String())
	}
}

func TestHandleEmailChangeMissingPassword(t *testing.T) {
	authStore := newEmailTestAuthStore()
	h := buildEmailHandler(authStore, &fakeMailer{})

	rec := doRequest(h, http.MethodPost, "/api/v1/auth/email/change", map[string]string{
		"email": "new@example.com", "current_password": "",
	}, nil)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("email change status = %d, want 400: %s", rec.Code, rec.Body.String())
	}
}

func TestHandleEmailChangeGetPasswordHashError(t *testing.T) {
	authStore := newEmailTestAuthStore()
	h := buildEmailHandler(authStore, &fakeMailer{})
	// No phcHash entry for test-user.

	rec := doRequest(h, http.MethodPost, "/api/v1/auth/email/change", map[string]string{
		"email": "new@example.com", "current_password": "whatever",
	}, nil)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("email change status = %d, want 404: %s", rec.Code, rec.Body.String())
	}
}

func TestHandleEmailChangeWrongPassword(t *testing.T) {
	authStore := newEmailTestAuthStore()
	hash, err := auth.Hash("correct horse battery staple")
	if err != nil {
		t.Fatal(err)
	}
	authStore.phcHash["test-user"] = hash
	h := buildEmailHandler(authStore, &fakeMailer{})

	rec := doRequest(h, http.MethodPost, "/api/v1/auth/email/change", map[string]string{
		"email": "new@example.com", "current_password": "wrong password",
	}, nil)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("email change status = %d, want 401: %s", rec.Code, rec.Body.String())
	}
}

func TestHandleEmailChangeConflict(t *testing.T) {
	authStore := newEmailTestAuthStore()
	hash, err := auth.Hash("correct horse battery staple")
	if err != nil {
		t.Fatal(err)
	}
	authStore.phcHash["test-user"] = hash
	authStore.userByEmail["taken@example.com"] = types.User{ID: "other-user", Email: "taken@example.com"}
	h := buildEmailHandler(authStore, &fakeMailer{})

	rec := doRequest(h, http.MethodPost, "/api/v1/auth/email/change", map[string]string{
		"email": "taken@example.com", "current_password": "correct horse battery staple",
	}, nil)
	if rec.Code != http.StatusConflict {
		t.Fatalf("email change status = %d, want 409: %s", rec.Code, rec.Body.String())
	}
}

func TestHandleEmailChangeSameUserNoConflict(t *testing.T) {
	authStore := newEmailTestAuthStore()
	hash, err := auth.Hash("correct horse battery staple")
	if err != nil {
		t.Fatal(err)
	}
	authStore.phcHash["test-user"] = hash
	// "own" email already resolves to the same user — not a conflict.
	authStore.userByEmail["self@example.com"] = types.User{ID: "test-user", Email: "self@example.com"}
	h := buildEmailHandler(authStore, &fakeMailer{})

	rec := doRequest(h, http.MethodPost, "/api/v1/auth/email/change", map[string]string{
		"email": "self@example.com", "current_password": "correct horse battery staple",
	}, nil)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("email change status = %d, want 204: %s", rec.Code, rec.Body.String())
	}
}

func TestHandleEmailChangeGetUserError(t *testing.T) {
	authStore := newEmailTestAuthStore()
	hash, err := auth.Hash("correct horse battery staple")
	if err != nil {
		t.Fatal(err)
	}
	authStore.phcHash["test-user"] = hash
	store := newFakeMealStore()
	store.getUserErr = errors.New("db down")
	h := New(store, &fakeMealLogger{}, time.UTC, nil, nil,
		WithAuth(authStore, AuthRepos{Sessions: authStore, LoginAttempts: authStore, TOTP: authStore, MFAChallenges: authStore, RecoveryCodes: authStore}, nil, "DietDaemon", testAuthConfig()),
		WithMailer(&fakeMailer{}, "none"),
		WithPublicBaseURL("http://localhost:8080"),
	)

	rec := doRequest(h, http.MethodPost, "/api/v1/auth/email/change", map[string]string{
		"email": "new@example.com", "current_password": "correct horse battery staple",
	}, nil)
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("email change status = %d, want 500: %s", rec.Code, rec.Body.String())
	}
}

func TestHandleEmailChangeUpdateEmailError(t *testing.T) {
	authStore := newEmailTestAuthStore()
	hash, err := auth.Hash("correct horse battery staple")
	if err != nil {
		t.Fatal(err)
	}
	authStore.phcHash["test-user"] = hash
	authStore.updateEmailFail = true
	h := buildEmailHandler(authStore, &fakeMailer{})

	rec := doRequest(h, http.MethodPost, "/api/v1/auth/email/change", map[string]string{
		"email": "new@example.com", "current_password": "correct horse battery staple",
	}, nil)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("email change status = %d, want 404: %s", rec.Code, rec.Body.String())
	}
}

func TestHandleEmailChangeCreateEmailTokenError(t *testing.T) {
	authStore := newEmailTestAuthStore()
	hash, err := auth.Hash("correct horse battery staple")
	if err != nil {
		t.Fatal(err)
	}
	authStore.phcHash["test-user"] = hash
	authStore.createEmailTokenErr = errors.New("store unavailable")
	h := buildEmailHandler(authStore, &fakeMailer{})

	rec := doRequest(h, http.MethodPost, "/api/v1/auth/email/change", map[string]string{
		"email": "new@example.com", "current_password": "correct horse battery staple",
	}, nil)
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("email change status = %d, want 500: %s", rec.Code, rec.Body.String())
	}
}

func TestHandleEmailChangeSuccess(t *testing.T) {
	authStore := newEmailTestAuthStore()
	hash, err := auth.Hash("correct horse battery staple")
	if err != nil {
		t.Fatal(err)
	}
	authStore.phcHash["test-user"] = hash
	fm := &fakeMailer{}
	h := buildEmailHandler(authStore, fm)

	rec := doRequest(h, http.MethodPost, "/api/v1/auth/email/change", map[string]string{
		"email": "new@example.com", "current_password": "correct horse battery staple",
	}, nil)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("email change status = %d, want 204: %s", rec.Code, rec.Body.String())
	}
	if authStore.userEmails["test-user"] != "new@example.com" {
		t.Errorf("expected email updated to new@example.com, got %q", authStore.userEmails["test-user"])
	}
	if len(fm.sent) != 1 || fm.sent[0].to != "new@example.com" {
		t.Errorf("expected verification email sent to new address, got %#v", fm.sent)
	}
	var events []string
	for _, ev := range authStore.auditEvents {
		events = append(events, ev.Event)
	}
	if !containsStr(events, "email.changed") || !containsStr(events, "email.verification_sent") {
		t.Errorf("expected email.changed and email.verification_sent audits, got %v", events)
	}
}

func containsStr(list []string, want string) bool {
	for _, v := range list {
		if v == want {
			return true
		}
	}
	return false
}

// ---------------------------------------------------------------------------
// handleEmailVerify
// ---------------------------------------------------------------------------

func TestHandleEmailVerifyInvalidJSON(t *testing.T) {
	authStore := newEmailTestAuthStore()
	h := buildEmailHandler(authStore, &fakeMailer{})

	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/email/verify", strings.NewReader("not json"))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("verify status = %d, want 400", rec.Code)
	}
}

func TestHandleEmailVerifyEmptyToken(t *testing.T) {
	authStore := newEmailTestAuthStore()
	h := buildEmailHandler(authStore, &fakeMailer{})

	rec := doRequest(h, http.MethodPost, "/api/v1/auth/email/verify", map[string]string{"token": ""}, nil)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("verify status = %d, want 400: %s", rec.Code, rec.Body.String())
	}
}

func TestHandleEmailVerifyMarkVerifiedError(t *testing.T) {
	authStore := newEmailTestAuthStore()
	authStore.markVerifiedFail = true
	h := buildEmailHandler(authStore, &fakeMailer{})

	tok := auth.NewToken()
	hashed := auth.HashToken(tok)
	_ = authStore.CreateEmailToken(t.Context(), hashed, "test-user", "verify", time.Now().UTC().Add(time.Hour).Format(time.RFC3339))

	rec := doRequest(h, http.MethodPost, "/api/v1/auth/email/verify", map[string]string{"token": tok}, nil)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("verify status = %d, want 404: %s", rec.Code, rec.Body.String())
	}
}

func TestHandleEmailVerifyAuditsAccountID(t *testing.T) {
	authStore := newEmailTestAuthStore()
	fm := &fakeMailer{}
	user := types.User{ID: "test-user", AccountID: "acct-1", Email: "test@example.com", Status: "active", CreatedAt: time.Now().UTC()}
	h := buildAuthTestHandler(authStore, user, fm)

	tok := auth.NewToken()
	hashed := auth.HashToken(tok)
	_ = authStore.CreateEmailToken(t.Context(), hashed, "test-user", "verify", time.Now().UTC().Add(time.Hour).Format(time.RFC3339))

	rec := doRequest(h, http.MethodPost, "/api/v1/auth/email/verify", map[string]string{"token": tok}, nil)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("verify status = %d, want 204: %s", rec.Code, rec.Body.String())
	}
	found := false
	for _, ev := range authStore.auditEvents {
		if ev.Event == "email.verified" && ev.AccountID == "acct-1" {
			found = true
		}
	}
	if !found {
		t.Error("expected email.verified audit event with account id")
	}
}

// ---------------------------------------------------------------------------
// handleEmailChange: invalid JSON, mailer send failure still succeeds
// ---------------------------------------------------------------------------

func TestHandleEmailChangeInvalidJSON(t *testing.T) {
	authStore := newEmailTestAuthStore()
	h := buildEmailHandler(authStore, &fakeMailer{})

	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/email/change", strings.NewReader("not json"))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer test-api-key")
	rec := httptest.NewRecorder()
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("email change status = %d, want 400", rec.Code)
	}
}

func TestHandleEmailChangeMailerSendErrorStillSucceeds(t *testing.T) {
	authStore := newEmailTestAuthStore()
	hash, err := auth.Hash("correct horse battery staple")
	if err != nil {
		t.Fatal(err)
	}
	authStore.phcHash["test-user"] = hash
	fm := &fakeMailer{sendErr: errors.New("smtp down")}
	h := buildEmailHandler(authStore, fm)

	rec := doRequest(h, http.MethodPost, "/api/v1/auth/email/change", map[string]string{
		"email": "new@example.com", "current_password": "correct horse battery staple",
	}, nil)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("email change status = %d, want 204 (best-effort send): %s", rec.Code, rec.Body.String())
	}
	if authStore.userEmails["test-user"] != "new@example.com" {
		t.Error("email should still be updated despite mailer failure")
	}
}

// ---------------------------------------------------------------------------
// handleForgotPassword
// ---------------------------------------------------------------------------

func TestHandleForgotPasswordInvalidJSON(t *testing.T) {
	authStore := newEmailTestAuthStore()
	h := buildEmailHandler(authStore, &fakeMailer{})

	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/password/forgot", strings.NewReader("not json"))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("forgot-password status = %d, want 200 (generic response): %s", rec.Code, rec.Body.String())
	}
}

func TestHandleForgotPasswordEmptyEmail(t *testing.T) {
	authStore := newEmailTestAuthStore()
	h := buildEmailHandler(authStore, &fakeMailer{})

	rec := doRequest(h, http.MethodPost, "/api/v1/auth/password/forgot", map[string]string{"email": ""}, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("forgot-password status = %d, want 200: %s", rec.Code, rec.Body.String())
	}
}

func TestHandleForgotPasswordSuccess(t *testing.T) {
	authStore := newEmailTestAuthStore()
	authStore.userByEmail["test@example.com"] = types.User{ID: "test-user", Email: "test@example.com"}
	authStore.phcHash["test-user"] = "password hash"
	fm := &fakeMailer{}
	h := buildEmailHandler(authStore, fm)

	rec := doRequest(h, http.MethodPost, "/api/v1/auth/password/forgot", map[string]string{"email": "test@example.com"}, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("forgot-password status = %d, want 200: %s", rec.Code, rec.Body.String())
	}
	if len(fm.sent) != 1 || fm.sent[0].to != "test@example.com" {
		t.Errorf("expected password reset email sent, got %#v", fm.sent)
	}
	if len(authStore.emailTokens) != 1 {
		t.Errorf("expected one reset token issued, got %d", len(authStore.emailTokens))
	}
}

// ---------------------------------------------------------------------------
// handleResetPassword
// ---------------------------------------------------------------------------

func TestHandleResetPasswordMissingFields(t *testing.T) {
	authStore := newEmailTestAuthStore()
	h := buildEmailHandler(authStore, &fakeMailer{})

	cases := []map[string]string{
		{"token": "", "password": "newSecurePassword123!"},
		{"token": "sometoken", "password": ""},
	}
	for _, body := range cases {
		rec := doRequest(h, http.MethodPost, "/api/v1/auth/password/reset", body, nil)
		if rec.Code != http.StatusBadRequest {
			t.Errorf("reset-password with %+v status = %d, want 400", body, rec.Code)
		}
	}
}

func TestHandleResetPasswordPasswordTooShort(t *testing.T) {
	authStore := newEmailTestAuthStore()
	h := buildEmailHandler(authStore, &fakeMailer{})

	rec := doRequest(h, http.MethodPost, "/api/v1/auth/password/reset", map[string]string{
		"token": "sometoken", "password": "short",
	}, nil)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("reset-password status = %d, want 400: %s", rec.Code, rec.Body.String())
	}
}

func TestHandleResetPasswordInvalidToken(t *testing.T) {
	authStore := newEmailTestAuthStore()
	h := buildEmailHandler(authStore, &fakeMailer{})

	rec := doRequest(h, http.MethodPost, "/api/v1/auth/password/reset", map[string]string{
		"token": "bogus-token", "password": "newSecurePassword123!",
	}, nil)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("reset-password status = %d, want 400: %s", rec.Code, rec.Body.String())
	}
}

func TestHandleResetPasswordAuditsAccountID(t *testing.T) {
	authStore := newEmailTestAuthStore()
	authStore.phcHash["test-user"] = "$argon2id$v=19$m=65536,t=3,p=4$c29tZXNhbHQ$c29tZWhhc2g"
	user := types.User{ID: "test-user", AccountID: "acct-1", Email: "test@example.com", Status: "active", CreatedAt: time.Now().UTC()}
	h := buildAuthTestHandler(authStore, user, &fakeMailer{})

	tok := auth.NewToken()
	hashed := auth.HashToken(tok)
	_ = authStore.CreateEmailToken(t.Context(), hashed, "test-user", "reset", time.Now().UTC().Add(time.Hour).Format(time.RFC3339))

	rec := doRequest(h, http.MethodPost, "/api/v1/auth/password/reset", map[string]string{
		"token": tok, "password": "newSecurePassword123!",
	}, nil)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("reset-password status = %d, want 204: %s", rec.Code, rec.Body.String())
	}
	found := false
	for _, ev := range authStore.auditEvents {
		if ev.Event == "password.reset" && ev.AccountID == "acct-1" {
			found = true
		}
	}
	if !found {
		t.Error("expected password.reset audit event with account id")
	}
}
