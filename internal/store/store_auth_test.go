package store

import (
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/gsaraiva2109/dietdaemon/core/types"
	"github.com/gsaraiva2109/dietdaemon/internal/auth"
)

func TestPurgeAuthRecords(t *testing.T) {
	s, cleanup := tempDB(t)
	defer cleanup()

	now := time.Now().UTC()
	if err := s.RecordLoginAttempt(ctx(), "old@example.com", false); err != nil {
		t.Fatalf("RecordLoginAttempt old: %v", err)
	}
	if err := s.RecordLoginAttempt(ctx(), "new@example.com", false); err != nil {
		t.Fatalf("RecordLoginAttempt new: %v", err)
	}
	if _, err := s.db.Exec(`UPDATE login_attempts SET created_at = ? WHERE identifier = ?`, utcStr(now.Add(-48*time.Hour)), "old@example.com"); err != nil {
		t.Fatalf("backdate login attempt: %v", err)
	}
	if n, err := s.PurgeLoginAttempts(ctx(), now.Add(-24*time.Hour)); err != nil || n != 1 {
		t.Fatalf("PurgeLoginAttempts = %d, %v; want 1, nil", n, err)
	}

	if err := s.WriteAuditEvent(ctx(), types.AuditEvent{ID: "audit-old", Event: "login.fail", CreatedAt: now.Add(-120 * 24 * time.Hour)}); err != nil {
		t.Fatalf("WriteAuditEvent old: %v", err)
	}
	if err := s.WriteAuditEvent(ctx(), types.AuditEvent{ID: "audit-new", Event: "login.success", CreatedAt: now}); err != nil {
		t.Fatalf("WriteAuditEvent new: %v", err)
	}
	if n, err := s.PurgeAuthAuditEvents(ctx(), now.Add(-90*24*time.Hour)); err != nil || n != 1 {
		t.Fatalf("PurgeAuthAuditEvents = %d, %v; want 1, nil", n, err)
	}
}

// ---------------------------------------------------------------------------
// OIDC state create / consume — single-use + expiry
// ---------------------------------------------------------------------------

func TestOIDCStateCreateConsume(t *testing.T) {
	s, cleanup := tempDB(t)
	defer cleanup()

	id := "abc123"
	nonce := "nonce-1"
	verifier := "pkce-verifier-1"
	next := "/dashboard"
	expiresAt := time.Now().UTC().Add(10 * time.Minute).Format(time.RFC3339)

	// Create state.
	if err := s.CreateOIDCState(ctx(), id, nonce, verifier, "", next, expiresAt); err != nil {
		t.Fatalf("CreateOIDCState: %v", err)
	}

	// Consume — should succeed.
	gotNonce, gotVerifier, gotLinkID, gotNext, err := s.ConsumeOIDCState(ctx(), id)
	if err != nil {
		t.Fatalf("ConsumeOIDCState: %v", err)
	}
	if gotNonce != nonce {
		t.Fatalf("nonce: expected %q, got %q", nonce, gotNonce)
	}
	if gotVerifier != verifier {
		t.Fatalf("verifier: expected %q, got %q", verifier, gotVerifier)
	}
	if gotNext != next {
		t.Fatalf("next: expected %q, got %q", next, gotNext)
	}
	if gotLinkID != "" {
		t.Fatalf("linkUserID: expected empty, got %q", gotLinkID)
	}

	// Second consume → ErrNotFound (single-use).
	_, _, _, _, err = s.ConsumeOIDCState(ctx(), id)
	if !errors.Is(err, types.ErrNotFound) {
		t.Fatalf("expected ErrNotFound on second consume, got %v", err)
	}
}

func TestOIDCStateLinkFlow(t *testing.T) {
	s, cleanup := tempDB(t)
	defer cleanup()

	linkUserID := "user-42"
	mustUser(t, s, types.User{ID: linkUserID})

	id := "link-state"
	expiresAt := time.Now().UTC().Add(10 * time.Minute).Format(time.RFC3339)

	if err := s.CreateOIDCState(ctx(), id, "nonce", "verifier", linkUserID, "", expiresAt); err != nil {
		t.Fatalf("CreateOIDCState: %v", err)
	}

	_, _, gotLinkID, _, err := s.ConsumeOIDCState(ctx(), id)
	if err != nil {
		t.Fatalf("ConsumeOIDCState: %v", err)
	}
	if gotLinkID != linkUserID {
		t.Fatalf("linkUserID: expected %q, got %q", linkUserID, gotLinkID)
	}
}

func TestOIDCStateExpired(t *testing.T) {
	s, cleanup := tempDB(t)
	defer cleanup()

	id := "expired-state"
	expiresAt := time.Now().UTC().Add(-1 * time.Minute).Format(time.RFC3339) // already expired

	if err := s.CreateOIDCState(ctx(), id, "nonce", "verifier", "", "", expiresAt); err != nil {
		t.Fatalf("CreateOIDCState: %v", err)
	}

	_, _, _, _, err := s.ConsumeOIDCState(ctx(), id)
	if !errors.Is(err, types.ErrNotFound) {
		t.Fatalf("expected ErrNotFound for expired state, got %v", err)
	}
}

// ---------------------------------------------------------------------------
// LinkOIDCIdentity — uniqueness conflict
// ---------------------------------------------------------------------------

func TestLinkOIDCIdentityUniqueness(t *testing.T) {
	s, cleanup := tempDB(t)
	defer cleanup()

	// Need a user with an account (FK constraint).
	if err := s.CreateAccount(ctx(), "acct-1"); err != nil {
		t.Fatalf("CreateAccount: %v", err)
	}
	u := types.User{
		ID: "user-1", AccountID: "acct-1", Email: "a@b.com",
		Status: "active", Timezone: "UTC",
		CreatedAt: time.Date(2026, 6, 20, 12, 0, 0, 0, time.UTC),
	}
	mustUser(t, s, u)

	// Link identity.
	if err := s.LinkOIDCIdentity(ctx(), "id-1", "user-1", "google", "sub-1", "a@b.com"); err != nil {
		t.Fatalf("LinkOIDCIdentity #1: %v", err)
	}

	// Same provider+subject again → ErrIdentityLinked.
	err := s.LinkOIDCIdentity(ctx(), "id-2", "user-1", "google", "sub-1", "a@b.com")
	if !errors.Is(err, types.ErrIdentityLinked) {
		t.Fatalf("expected ErrIdentityLinked, got %v", err)
	}

	// Same provider+subject, different user → also ErrIdentityLinked.
	if err := s.CreateAccount(ctx(), "acct-2"); err != nil {
		t.Fatalf("CreateAccount acct-2: %v", err)
	}
	u2 := types.User{
		ID: "user-2", AccountID: "acct-2", Email: "b@c.com",
		Status: "active", Timezone: "UTC",
		CreatedAt: time.Date(2026, 6, 20, 12, 0, 0, 0, time.UTC),
	}
	mustUser(t, s, u2)
	err = s.LinkOIDCIdentity(ctx(), "id-3", "user-2", "google", "sub-1", "b@c.com")
	if !errors.Is(err, types.ErrIdentityLinked) {
		t.Fatalf("expected ErrIdentityLinked for different user, got %v", err)
	}
}

// ---------------------------------------------------------------------------
// GetUserByOIDCIdentity + CreateUserWithOIDC
// ---------------------------------------------------------------------------

func TestGetUserByOIDCIdentity(t *testing.T) {
	s, cleanup := tempDB(t)
	defer cleanup()

	// Not found when no identity exists.
	_, err := s.GetUserByOIDCIdentity(ctx(), "google", "sub-nonexistent")
	if !errors.Is(err, types.ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}

	// Create user with OIDC.
	u, err := s.CreateUserWithOIDC(ctx(), "acct-oidc", "user-oidc", "oidc@example.com", "OIDC User", types.OIDCIdentityInput{
		ID:       "id-oidc-1",
		Provider: "google",
		Subject:  "sub-123",
	})
	if err != nil {
		t.Fatalf("CreateUserWithOIDC: %v", err)
	}
	if u.ID != "user-oidc" {
		t.Fatalf("user id: expected user-oidc, got %s", u.ID)
	}
	if u.EmailVerifiedAt == nil {
		t.Fatal("expected EmailVerifiedAt to be set for OIDC user")
	}

	// Lookup by identity.
	u2, err := s.GetUserByOIDCIdentity(ctx(), "google", "sub-123")
	if err != nil {
		t.Fatalf("GetUserByOIDCIdentity: %v", err)
	}
	if u2.ID != "user-oidc" || u2.Email != "oidc@example.com" {
		t.Fatalf("user mismatch: %+v", u2)
	}
}

// ---------------------------------------------------------------------------
// ListOIDCIdentities + DeleteOIDCIdentity
// ---------------------------------------------------------------------------

func TestListDeleteOIDCIdentities(t *testing.T) {
	s, cleanup := tempDB(t)
	defer cleanup()

	if err := s.CreateAccount(ctx(), "acct-ld"); err != nil {
		t.Fatalf("CreateAccount: %v", err)
	}
	u := types.User{
		ID: "ud-1", AccountID: "acct-ld", Email: "ld@test.com",
		Status: "active", Timezone: "UTC",
		CreatedAt: time.Date(2026, 6, 20, 12, 0, 0, 0, time.UTC),
	}
	mustUser(t, s, u)

	// No identities yet.
	list, err := s.ListOIDCIdentities(ctx(), "ud-1")
	if err != nil {
		t.Fatalf("ListOIDCIdentities: %v", err)
	}
	if len(list) != 0 {
		t.Fatalf("expected empty list, got %d", len(list))
	}

	// Add two identities.
	if err := s.LinkOIDCIdentity(ctx(), "li-1", "ud-1", "google", "g-sub", "ld@gmail.com"); err != nil {
		t.Fatalf("LinkOIDCIdentity google: %v", err)
	}
	if err := s.LinkOIDCIdentity(ctx(), "li-2", "ud-1", "github", "gh-sub", "ld@github.com"); err != nil {
		t.Fatalf("LinkOIDCIdentity github: %v", err)
	}

	list, err = s.ListOIDCIdentities(ctx(), "ud-1")
	if err != nil {
		t.Fatalf("ListOIDCIdentities: %v", err)
	}
	if len(list) != 2 {
		t.Fatalf("expected 2 identities, got %d", len(list))
	}

	// Delete one.
	if err := s.DeleteOIDCIdentity(ctx(), "ud-1", "li-1"); err != nil {
		t.Fatalf("DeleteOIDCIdentity: %v", err)
	}

	list, err = s.ListOIDCIdentities(ctx(), "ud-1")
	if err != nil {
		t.Fatalf("ListOIDCIdentities after delete: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("expected 1 identity after delete, got %d", len(list))
	}

	// Delete scoped to wrong user → ErrNotFound.
	err = s.DeleteOIDCIdentity(ctx(), "wrong-user", "li-2")
	if !errors.Is(err, types.ErrNotFound) {
		t.Fatalf("expected ErrNotFound for wrong user, got %v", err)
	}

	// Delete nonexistent → ErrNotFound.
	err = s.DeleteOIDCIdentity(ctx(), "ud-1", "nonexistent")
	if !errors.Is(err, types.ErrNotFound) {
		t.Fatalf("expected ErrNotFound for nonexistent, got %v", err)
	}
}

// ---------------------------------------------------------------------------
// Magic code upsert / get / increment / delete
// ---------------------------------------------------------------------------

func TestMagicCodeUpsertGet(t *testing.T) {
	s, cleanup := tempDB(t)
	defer cleanup()

	// Create a user first (FK constraint).
	u, err := s.CreateUserWithPassword(ctx(), "acct-mc", "user-mc", "mc@example.com", "MC User", "$argon2id$dummy")
	if err != nil {
		t.Fatalf("CreateUserWithPassword: %v", err)
	}

	codeHash := "abc123hash"
	expiresAt := time.Now().UTC().Add(15 * time.Minute).Format(time.RFC3339)

	// Upsert (insert).
	if err := s.UpsertMagicCode(ctx(), u.ID, codeHash, expiresAt); err != nil {
		t.Fatalf("UpsertMagicCode: %v", err)
	}

	// Get.
	gotHash, gotExpiry, attempts, err := s.GetMagicCode(ctx(), u.ID)
	if err != nil {
		t.Fatalf("GetMagicCode: %v", err)
	}
	if gotHash != codeHash {
		t.Fatalf("codeHash: expected %q, got %q", codeHash, gotHash)
	}
	if gotExpiry != expiresAt {
		t.Fatalf("expiresAt: expected %q, got %q", expiresAt, gotExpiry)
	}
	if attempts != 0 {
		t.Fatalf("attempts: expected 0, got %d", attempts)
	}
}

func TestMagicCodeUpsertOverwrite(t *testing.T) {
	s, cleanup := tempDB(t)
	defer cleanup()

	u, err := s.CreateUserWithPassword(ctx(), "acct-mc2", "user-mc2", "mc2@example.com", "MC2 User", "$argon2id$dummy")
	if err != nil {
		t.Fatalf("CreateUserWithPassword: %v", err)
	}

	expiresAt := time.Now().UTC().Add(15 * time.Minute).Format(time.RFC3339)

	// First upsert.
	if err := s.UpsertMagicCode(ctx(), u.ID, "hash1", expiresAt); err != nil {
		t.Fatalf("first UpsertMagicCode: %v", err)
	}

	// Second upsert (overwrite).
	if err := s.UpsertMagicCode(ctx(), u.ID, "hash2", expiresAt); err != nil {
		t.Fatalf("second UpsertMagicCode: %v", err)
	}

	gotHash, _, attempts, err := s.GetMagicCode(ctx(), u.ID)
	if err != nil {
		t.Fatalf("GetMagicCode: %v", err)
	}
	if gotHash != "hash2" {
		t.Fatalf("expected hash2 after overwrite, got %q", gotHash)
	}
	// Attempts should be reset to 0 on overwrite.
	if attempts != 0 {
		t.Fatalf("attempts should reset to 0 on overwrite, got %d", attempts)
	}
}

func TestMagicCodeNotFound(t *testing.T) {
	s, cleanup := tempDB(t)
	defer cleanup()

	_, _, _, err := s.GetMagicCode(ctx(), "nonexistent")
	if !errors.Is(err, types.ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestMagicCodeIncrementAttempts(t *testing.T) {
	s, cleanup := tempDB(t)
	defer cleanup()

	u, err := s.CreateUserWithPassword(ctx(), "acct-mc3", "user-mc3", "mc3@example.com", "MC3 User", "$argon2id$dummy")
	if err != nil {
		t.Fatalf("CreateUserWithPassword: %v", err)
	}

	expiresAt := time.Now().UTC().Add(15 * time.Minute).Format(time.RFC3339)
	if err := s.UpsertMagicCode(ctx(), u.ID, "somehash", expiresAt); err != nil {
		t.Fatalf("UpsertMagicCode: %v", err)
	}

	if err := s.IncrementMagicCodeAttempts(ctx(), u.ID); err != nil {
		t.Fatalf("IncrementMagicCodeAttempts: %v", err)
	}

	_, _, attempts, err := s.GetMagicCode(ctx(), u.ID)
	if err != nil {
		t.Fatalf("GetMagicCode: %v", err)
	}
	if attempts != 1 {
		t.Fatalf("attempts should be 1, got %d", attempts)
	}
}

func TestMagicCodeDelete(t *testing.T) {
	s, cleanup := tempDB(t)
	defer cleanup()

	u, err := s.CreateUserWithPassword(ctx(), "acct-mc4", "user-mc4", "mc4@example.com", "MC4 User", "$argon2id$dummy")
	if err != nil {
		t.Fatalf("CreateUserWithPassword: %v", err)
	}

	expiresAt := time.Now().UTC().Add(15 * time.Minute).Format(time.RFC3339)
	if err := s.UpsertMagicCode(ctx(), u.ID, "somehash", expiresAt); err != nil {
		t.Fatalf("UpsertMagicCode: %v", err)
	}

	if err := s.DeleteMagicCode(ctx(), u.ID); err != nil {
		t.Fatalf("DeleteMagicCode: %v", err)
	}

	_, _, _, err = s.GetMagicCode(ctx(), u.ID)
	if !errors.Is(err, types.ErrNotFound) {
		t.Fatalf("expected ErrNotFound after delete, got %v", err)
	}
}

// ---------------------------------------------------------------------------
// Email / password credential lookups
// ---------------------------------------------------------------------------

func TestGetUserByEmail(t *testing.T) {
	s, cleanup := tempDB(t)
	defer cleanup()

	if _, err := s.GetUserByEmail(ctx(), "nobody@example.com"); !errors.Is(err, types.ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}

	u, err := s.CreateUserWithPassword(ctx(), "acct-em", "user-em", "em@example.com", "Em User", "$argon2id$dummy")
	if err != nil {
		t.Fatalf("CreateUserWithPassword: %v", err)
	}

	got, err := s.GetUserByEmail(ctx(), "em@example.com")
	if err != nil {
		t.Fatalf("GetUserByEmail: %v", err)
	}
	if got.ID != u.ID || got.Email != "em@example.com" {
		t.Fatalf("user mismatch: %+v", got)
	}
}

func TestPasswordHashSetGet(t *testing.T) {
	s, cleanup := tempDB(t)
	defer cleanup()

	if _, err := s.GetPasswordHash(ctx(), "nonexistent"); !errors.Is(err, types.ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}

	u, err := s.CreateUserWithPassword(ctx(), "acct-ph", "user-ph", "ph@example.com", "PH User", "$argon2id$initial")
	if err != nil {
		t.Fatalf("CreateUserWithPassword: %v", err)
	}

	hash, err := s.GetPasswordHash(ctx(), u.ID)
	if err != nil {
		t.Fatalf("GetPasswordHash: %v", err)
	}
	if hash != "$argon2id$initial" {
		t.Fatalf("hash = %q, want $argon2id$initial", hash)
	}

	if err := s.SetPasswordHash(ctx(), u.ID, "$argon2id$updated"); err != nil {
		t.Fatalf("SetPasswordHash (update): %v", err)
	}
	hash, err = s.GetPasswordHash(ctx(), u.ID)
	if err != nil {
		t.Fatalf("GetPasswordHash after update: %v", err)
	}
	if hash != "$argon2id$updated" {
		t.Fatalf("hash after update = %q, want $argon2id$updated", hash)
	}

	// SetPasswordHash also covers the insert (no prior row) path via ON CONFLICT.
	mustUser(t, s, types.User{ID: "ph-user2", CreatedAt: time.Now().UTC()})
	if err := s.SetPasswordHash(ctx(), "ph-user2", "$argon2id$new"); err != nil {
		t.Fatalf("SetPasswordHash (insert): %v", err)
	}
	hash, err = s.GetPasswordHash(ctx(), "ph-user2")
	if err != nil {
		t.Fatalf("GetPasswordHash for ph-user2: %v", err)
	}
	if hash != "$argon2id$new" {
		t.Fatalf("hash for ph-user2 = %q, want $argon2id$new", hash)
	}
}

func TestCountUsers(t *testing.T) {
	s, cleanup := tempDB(t)
	defer cleanup()

	n, err := s.CountUsers(ctx())
	if err != nil {
		t.Fatalf("CountUsers (empty): %v", err)
	}
	if n != 0 {
		t.Fatalf("CountUsers (empty) = %d, want 0", n)
	}

	mustUser(t, s, types.User{ID: "cu-1", CreatedAt: time.Now().UTC()})
	mustUser(t, s, types.User{ID: "cu-2", CreatedAt: time.Now().UTC()})

	n, err = s.CountUsers(ctx())
	if err != nil {
		t.Fatalf("CountUsers: %v", err)
	}
	if n != 2 {
		t.Fatalf("CountUsers = %d, want 2", n)
	}
}

// ---------------------------------------------------------------------------
// Sessions
// ---------------------------------------------------------------------------

func TestSessionLifecycle(t *testing.T) {
	s, cleanup := tempDB(t)
	defer cleanup()

	mustUser(t, s, types.User{ID: "sess-user", CreatedAt: time.Now().UTC()})

	now := time.Now().UTC().Truncate(time.Second)
	sess := auth.Session{
		ID:                "sess-1",
		UserID:            "sess-user",
		CSRFToken:         "csrf-abc",
		CreatedAt:         now,
		LastSeenAt:        now,
		IdleExpiresAt:     now.Add(30 * time.Minute),
		AbsoluteExpiresAt: now.Add(24 * time.Hour),
		Remember:          true,
		IP:                "203.0.113.5",
		UserAgent:         "Mozilla/5.0 test",
	}
	if err := s.CreateSession(ctx(), sess); err != nil {
		t.Fatalf("CreateSession: %v", err)
	}

	got, err := s.GetSession(ctx(), "sess-1")
	if err != nil {
		t.Fatalf("GetSession: %v", err)
	}
	if got.ID != sess.ID || got.UserID != sess.UserID || got.CSRFToken != sess.CSRFToken ||
		!got.CreatedAt.Equal(sess.CreatedAt) || !got.Remember || got.IP != sess.IP || got.UserAgent != sess.UserAgent {
		t.Fatalf("session mismatch: got %+v, want %+v", got, sess)
	}

	newLastSeen := now.Add(5 * time.Minute)
	newIdleExp := now.Add(35 * time.Minute)
	if err := s.TouchSession(ctx(), "sess-1", newLastSeen, newIdleExp); err != nil {
		t.Fatalf("TouchSession: %v", err)
	}
	touched, err := s.GetSession(ctx(), "sess-1")
	if err != nil {
		t.Fatalf("GetSession after touch: %v", err)
	}
	if !touched.LastSeenAt.Equal(newLastSeen) || !touched.IdleExpiresAt.Equal(newIdleExp) {
		t.Fatalf("touch did not update: %+v", touched)
	}

	if err := s.DeleteSession(ctx(), "sess-1"); err != nil {
		t.Fatalf("DeleteSession: %v", err)
	}
	if _, err := s.GetSession(ctx(), "sess-1"); !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("expected wrapped sql.ErrNoRows after delete, got %v", err)
	}
}

func TestDeleteUserSessions(t *testing.T) {
	s, cleanup := tempDB(t)
	defer cleanup()

	mustUser(t, s, types.User{ID: "du-user", CreatedAt: time.Now().UTC()})
	now := time.Now().UTC().Truncate(time.Second)
	for _, id := range []string{"du-sess-1", "du-sess-2"} {
		sess := auth.Session{
			ID: id, UserID: "du-user", CSRFToken: "csrf",
			CreatedAt: now, LastSeenAt: now,
			IdleExpiresAt: now.Add(time.Hour), AbsoluteExpiresAt: now.Add(24 * time.Hour),
			IP: "127.0.0.1", UserAgent: "ua",
		}
		if err := s.CreateSession(ctx(), sess); err != nil {
			t.Fatalf("CreateSession %s: %v", id, err)
		}
	}

	if err := s.DeleteUserSessions(ctx(), "du-user"); err != nil {
		t.Fatalf("DeleteUserSessions: %v", err)
	}
	for _, id := range []string{"du-sess-1", "du-sess-2"} {
		if _, err := s.GetSession(ctx(), id); err == nil {
			t.Fatalf("session %s still exists after DeleteUserSessions", id)
		}
	}
}

// ---------------------------------------------------------------------------
// API keys
// ---------------------------------------------------------------------------

func TestAPIKeyLifecycle(t *testing.T) {
	s, cleanup := tempDB(t)
	defer cleanup()

	mustUser(t, s, types.User{ID: "ak-user", CreatedAt: time.Now().UTC()})
	mustUser(t, s, types.User{ID: "ak-other", CreatedAt: time.Now().UTC()})

	if err := s.CreateAPIKey(ctx(), "ak-1", "ak-user", "hashed-key-1", "laptop"); err != nil {
		t.Fatalf("CreateAPIKey: %v", err)
	}
	if err := s.CreateAPIKey(ctx(), "ak-2", "ak-user", "hashed-key-2", "phone"); err != nil {
		t.Fatalf("CreateAPIKey #2: %v", err)
	}

	keys, err := s.ListAPIKeys(ctx(), "ak-user")
	if err != nil {
		t.Fatalf("ListAPIKeys: %v", err)
	}
	if len(keys) != 2 {
		t.Fatalf("expected 2 keys, got %d", len(keys))
	}
	for _, k := range keys {
		if k.LastUsedAt != nil || k.RevokedAt != nil {
			t.Fatalf("fresh key should have nil LastUsedAt/RevokedAt: %+v", k)
		}
	}

	u, err := s.GetUserByAPIKey(ctx(), "hashed-key-1")
	if err != nil {
		t.Fatalf("GetUserByAPIKey: %v", err)
	}
	if u.ID != "ak-user" {
		t.Fatalf("user id = %q, want ak-user", u.ID)
	}

	var lastUsed sql.NullString
	if err := s.db.Get(&lastUsed, `SELECT last_used_at FROM api_keys WHERE id = ?`, "ak-1"); err != nil {
		t.Fatalf("query last_used_at: %v", err)
	}
	if !lastUsed.Valid || lastUsed.String == "" {
		t.Fatal("expected last_used_at to be set after GetUserByAPIKey")
	}

	if _, err := s.GetUserByAPIKey(ctx(), "no-such-key"); !errors.Is(err, types.ErrNotFound) {
		t.Fatalf("expected ErrNotFound for unknown key, got %v", err)
	}

	if err := s.RevokeAPIKey(ctx(), "ak-other", "ak-1"); !errors.Is(err, types.ErrNotFound) {
		t.Fatalf("expected ErrNotFound revoking as wrong user, got %v", err)
	}

	if err := s.RevokeAPIKey(ctx(), "ak-user", "ak-1"); err != nil {
		t.Fatalf("RevokeAPIKey: %v", err)
	}
	if err := s.RevokeAPIKey(ctx(), "ak-user", "ak-1"); !errors.Is(err, types.ErrNotFound) {
		t.Fatalf("expected ErrNotFound on double revoke, got %v", err)
	}
	if err := s.RevokeAPIKey(ctx(), "ak-user", "no-such-id"); !errors.Is(err, types.ErrNotFound) {
		t.Fatalf("expected ErrNotFound revoking nonexistent, got %v", err)
	}

	if _, err := s.GetUserByAPIKey(ctx(), "hashed-key-1"); !errors.Is(err, types.ErrNotFound) {
		t.Fatalf("expected ErrNotFound for revoked key, got %v", err)
	}
	keys, err = s.ListAPIKeys(ctx(), "ak-user")
	if err != nil {
		t.Fatalf("ListAPIKeys after revoke: %v", err)
	}
	if len(keys) != 1 || keys[0].ID != "ak-2" {
		t.Fatalf("expected only ak-2 left, got %+v", keys)
	}
}

// ---------------------------------------------------------------------------
// Share tokens
// ---------------------------------------------------------------------------

func TestShareTokenLifecycle(t *testing.T) {
	s, cleanup := tempDB(t)
	defer cleanup()

	mustUser(t, s, types.User{ID: "st-user", CreatedAt: time.Now().UTC()})
	mustUser(t, s, types.User{ID: "st-other", CreatedAt: time.Now().UTC()})

	if err := s.CreateShareToken(ctx(), "st-1", "st-user", "hashed-tok-1", "dashboard"); err != nil {
		t.Fatalf("CreateShareToken: %v", err)
	}
	if err := s.CreateShareToken(ctx(), "st-2", "st-user", "hashed-tok-2", "tv"); err != nil {
		t.Fatalf("CreateShareToken #2: %v", err)
	}

	tokens, err := s.ListShareTokens(ctx(), "st-user")
	if err != nil {
		t.Fatalf("ListShareTokens: %v", err)
	}
	if len(tokens) != 2 {
		t.Fatalf("expected 2 tokens, got %d", len(tokens))
	}

	u, err := s.GetUserByShareToken(ctx(), "hashed-tok-1")
	if err != nil {
		t.Fatalf("GetUserByShareToken: %v", err)
	}
	if u.ID != "st-user" {
		t.Fatalf("user id = %q, want st-user", u.ID)
	}

	var lastUsed sql.NullString
	if err := s.db.Get(&lastUsed, `SELECT last_used_at FROM share_tokens WHERE id = ?`, "st-1"); err != nil {
		t.Fatalf("query last_used_at: %v", err)
	}
	if !lastUsed.Valid || lastUsed.String == "" {
		t.Fatal("expected last_used_at to be set after GetUserByShareToken")
	}

	if _, err := s.GetUserByShareToken(ctx(), "no-such-token"); !errors.Is(err, types.ErrNotFound) {
		t.Fatalf("expected ErrNotFound for unknown token, got %v", err)
	}

	if err := s.RevokeShareToken(ctx(), "st-other", "st-1"); !errors.Is(err, types.ErrNotFound) {
		t.Fatalf("expected ErrNotFound revoking as wrong user, got %v", err)
	}
	if err := s.RevokeShareToken(ctx(), "st-user", "st-1"); err != nil {
		t.Fatalf("RevokeShareToken: %v", err)
	}
	if err := s.RevokeShareToken(ctx(), "st-user", "st-1"); !errors.Is(err, types.ErrNotFound) {
		t.Fatalf("expected ErrNotFound on double revoke, got %v", err)
	}

	if _, err := s.GetUserByShareToken(ctx(), "hashed-tok-1"); !errors.Is(err, types.ErrNotFound) {
		t.Fatalf("expected ErrNotFound for revoked token, got %v", err)
	}
	tokens, err = s.ListShareTokens(ctx(), "st-user")
	if err != nil {
		t.Fatalf("ListShareTokens after revoke: %v", err)
	}
	if len(tokens) != 1 || tokens[0].ID != "st-2" {
		t.Fatalf("expected only st-2 left, got %+v", tokens)
	}
}

// ---------------------------------------------------------------------------
// Login attempt lockout window
// ---------------------------------------------------------------------------

func TestRecentFailedAttempts(t *testing.T) {
	s, cleanup := tempDB(t)
	defer cleanup()

	ident := "flood@example.com"
	if err := s.RecordLoginAttempt(ctx(), ident, false); err != nil {
		t.Fatalf("RecordLoginAttempt #1: %v", err)
	}
	if err := s.RecordLoginAttempt(ctx(), ident, false); err != nil {
		t.Fatalf("RecordLoginAttempt #2: %v", err)
	}
	if err := s.RecordLoginAttempt(ctx(), ident, true); err != nil {
		t.Fatalf("RecordLoginAttempt (success): %v", err)
	}

	n, err := s.RecentFailedAttempts(ctx(), ident, time.Now().UTC().Add(-1*time.Hour))
	if err != nil {
		t.Fatalf("RecentFailedAttempts: %v", err)
	}
	if n != 2 {
		t.Fatalf("RecentFailedAttempts = %d, want 2 (success excluded)", n)
	}

	n, err = s.RecentFailedAttempts(ctx(), "other@example.com", time.Now().UTC().Add(-1*time.Hour))
	if err != nil {
		t.Fatalf("RecentFailedAttempts (unrelated identifier): %v", err)
	}
	if n != 0 {
		t.Fatalf("RecentFailedAttempts (unrelated) = %d, want 0", n)
	}
}

// ---------------------------------------------------------------------------
// TOTP secrets
// ---------------------------------------------------------------------------

func TestTOTPLifecycle(t *testing.T) {
	s, cleanup := tempDB(t)
	defer cleanup()

	mustUser(t, s, types.User{ID: "totp-user", CreatedAt: time.Now().UTC()})

	if _, _, err := s.GetTOTPSecret(ctx(), "totp-user"); !errors.Is(err, types.ErrNotFound) {
		t.Fatalf("expected ErrNotFound before upsert, got %v", err)
	}

	if err := s.UpsertTOTPSecret(ctx(), "totp-user", "enc-secret-1"); err != nil {
		t.Fatalf("UpsertTOTPSecret: %v", err)
	}
	secret, confirmed, err := s.GetTOTPSecret(ctx(), "totp-user")
	if err != nil {
		t.Fatalf("GetTOTPSecret: %v", err)
	}
	if secret != "enc-secret-1" || confirmed {
		t.Fatalf("secret=%q confirmed=%v, want enc-secret-1/false", secret, confirmed)
	}

	if has, err := s.HasConfirmedTOTP(ctx(), "totp-user"); err != nil || has {
		t.Fatalf("HasConfirmedTOTP before confirm = %v, %v; want false, nil", has, err)
	}

	if err := s.ConfirmTOTP(ctx(), "totp-user"); err != nil {
		t.Fatalf("ConfirmTOTP: %v", err)
	}
	_, confirmed, err = s.GetTOTPSecret(ctx(), "totp-user")
	if err != nil {
		t.Fatalf("GetTOTPSecret after confirm: %v", err)
	}
	if !confirmed {
		t.Fatal("expected confirmed=true after ConfirmTOTP")
	}
	if has, err := s.HasConfirmedTOTP(ctx(), "totp-user"); err != nil || !has {
		t.Fatalf("HasConfirmedTOTP after confirm = %v, %v; want true, nil", has, err)
	}

	// Re-upsert replaces the secret without needing to reconfirm state to be readable.
	if err := s.UpsertTOTPSecret(ctx(), "totp-user", "enc-secret-2"); err != nil {
		t.Fatalf("UpsertTOTPSecret (overwrite): %v", err)
	}
	secret, _, err = s.GetTOTPSecret(ctx(), "totp-user")
	if err != nil {
		t.Fatalf("GetTOTPSecret after overwrite: %v", err)
	}
	if secret != "enc-secret-2" {
		t.Fatalf("secret after overwrite = %q, want enc-secret-2", secret)
	}

	if err := s.DeleteTOTP(ctx(), "totp-user"); err != nil {
		t.Fatalf("DeleteTOTP: %v", err)
	}
	if _, _, err := s.GetTOTPSecret(ctx(), "totp-user"); !errors.Is(err, types.ErrNotFound) {
		t.Fatalf("expected ErrNotFound after delete, got %v", err)
	}
}

// ---------------------------------------------------------------------------
// Recovery codes
// ---------------------------------------------------------------------------

func TestRecoveryCodesLifecycle(t *testing.T) {
	s, cleanup := tempDB(t)
	defer cleanup()

	mustUser(t, s, types.User{ID: "rc-user", CreatedAt: time.Now().UTC()})

	if err := s.ReplaceRecoveryCodes(ctx(), "rc-user", []string{"hash1", "hash2", "hash3"}); err != nil {
		t.Fatalf("ReplaceRecoveryCodes: %v", err)
	}

	ok, err := s.ConsumeRecoveryCode(ctx(), "rc-user", "hash2")
	if err != nil {
		t.Fatalf("ConsumeRecoveryCode: %v", err)
	}
	if !ok {
		t.Fatal("expected hash2 to be consumable")
	}

	ok, err = s.ConsumeRecoveryCode(ctx(), "rc-user", "hash2")
	if err != nil {
		t.Fatalf("ConsumeRecoveryCode (reuse): %v", err)
	}
	if ok {
		t.Fatal("expected hash2 to be already-used")
	}

	ok, err = s.ConsumeRecoveryCode(ctx(), "rc-user", "nonexistent")
	if err != nil {
		t.Fatalf("ConsumeRecoveryCode (nonexistent): %v", err)
	}
	if ok {
		t.Fatal("expected nonexistent code to fail")
	}

	// Replacing wipes the old set entirely, including still-unused codes.
	if err := s.ReplaceRecoveryCodes(ctx(), "rc-user", []string{"hashA"}); err != nil {
		t.Fatalf("ReplaceRecoveryCodes (2nd): %v", err)
	}
	ok, err = s.ConsumeRecoveryCode(ctx(), "rc-user", "hash1")
	if err != nil {
		t.Fatalf("ConsumeRecoveryCode (hash1 after replace): %v", err)
	}
	if ok {
		t.Fatal("expected hash1 to be gone after replace")
	}
	ok, err = s.ConsumeRecoveryCode(ctx(), "rc-user", "hashA")
	if err != nil {
		t.Fatalf("ConsumeRecoveryCode (hashA): %v", err)
	}
	if !ok {
		t.Fatal("expected hashA to be consumable")
	}
}

// ---------------------------------------------------------------------------
// MFA challenges
// ---------------------------------------------------------------------------

func TestMFAChallengeLifecycle(t *testing.T) {
	s, cleanup := tempDB(t)
	defer cleanup()

	mustUser(t, s, types.User{ID: "mfa-user", CreatedAt: time.Now().UTC()})
	expiresAt := time.Now().UTC().Add(5 * time.Minute).Format(time.RFC3339)

	if err := s.CreateMFAChallenge(ctx(), "mfa-1", "mfa-user", true, expiresAt); err != nil {
		t.Fatalf("CreateMFAChallenge: %v", err)
	}
	gotUserID, remember, gotExpiry, err := s.GetMFAChallenge(ctx(), "mfa-1")
	if err != nil {
		t.Fatalf("GetMFAChallenge: %v", err)
	}
	if gotUserID != "mfa-user" || !remember || gotExpiry != expiresAt {
		t.Fatalf("challenge mismatch: userID=%q remember=%v expiry=%q", gotUserID, remember, gotExpiry)
	}

	if err := s.CreateMFAChallenge(ctx(), "mfa-2", "mfa-user", false, expiresAt); err != nil {
		t.Fatalf("CreateMFAChallenge (remember=false): %v", err)
	}
	_, remember, _, err = s.GetMFAChallenge(ctx(), "mfa-2")
	if err != nil {
		t.Fatalf("GetMFAChallenge (remember=false): %v", err)
	}
	if remember {
		t.Fatal("expected remember=false")
	}

	if err := s.DeleteMFAChallenge(ctx(), "mfa-1"); err != nil {
		t.Fatalf("DeleteMFAChallenge: %v", err)
	}
	if _, _, _, err := s.GetMFAChallenge(ctx(), "mfa-1"); !errors.Is(err, types.ErrNotFound) {
		t.Fatalf("expected ErrNotFound after delete, got %v", err)
	}
	if _, _, _, err := s.GetMFAChallenge(ctx(), "nonexistent"); !errors.Is(err, types.ErrNotFound) {
		t.Fatalf("expected ErrNotFound for nonexistent, got %v", err)
	}
}

// ---------------------------------------------------------------------------
// OIDC state deletion + null-email identity
// ---------------------------------------------------------------------------

func TestDeleteOIDCState(t *testing.T) {
	s, cleanup := tempDB(t)
	defer cleanup()

	id := "state-del"
	expiresAt := time.Now().UTC().Add(10 * time.Minute).Format(time.RFC3339)
	if err := s.CreateOIDCState(ctx(), id, "nonce", "verifier", "", "", expiresAt); err != nil {
		t.Fatalf("CreateOIDCState: %v", err)
	}
	if err := s.DeleteOIDCState(ctx(), id); err != nil {
		t.Fatalf("DeleteOIDCState: %v", err)
	}
	if _, _, _, _, err := s.ConsumeOIDCState(ctx(), id); !errors.Is(err, types.ErrNotFound) {
		t.Fatalf("expected ErrNotFound after delete, got %v", err)
	}

	// Deleting a nonexistent state is a best-effort no-op.
	if err := s.DeleteOIDCState(ctx(), "nonexistent"); err != nil {
		t.Fatalf("DeleteOIDCState (nonexistent): %v", err)
	}
}

func TestListOIDCIdentitiesNullEmail(t *testing.T) {
	s, cleanup := tempDB(t)
	defer cleanup()

	mustUser(t, s, types.User{ID: "oid-null", CreatedAt: time.Now().UTC()})
	if err := s.LinkOIDCIdentity(ctx(), "li-null", "oid-null", "github", "sub-null", ""); err != nil {
		t.Fatalf("LinkOIDCIdentity: %v", err)
	}

	list, err := s.ListOIDCIdentities(ctx(), "oid-null")
	if err != nil {
		t.Fatalf("ListOIDCIdentities: %v", err)
	}
	if len(list) != 1 || list[0].Email != "" {
		t.Fatalf("expected one identity with empty email, got %+v", list)
	}
}

// ---------------------------------------------------------------------------
// Email verification / change
// ---------------------------------------------------------------------------

func TestMarkEmailVerifiedAndUpdateEmail(t *testing.T) {
	s, cleanup := tempDB(t)
	defer cleanup()

	mustUser(t, s, types.User{ID: "ev-user", Email: "old@example.com", CreatedAt: time.Now().UTC()})

	if err := s.MarkEmailVerified(ctx(), "ev-user"); err != nil {
		t.Fatalf("MarkEmailVerified: %v", err)
	}
	got, err := s.GetUser(ctx(), "ev-user")
	if err != nil {
		t.Fatalf("GetUser: %v", err)
	}
	if got.EmailVerifiedAt == nil {
		t.Fatal("expected EmailVerifiedAt to be set")
	}

	if err := s.UpdateUserEmail(ctx(), "ev-user", "new@example.com"); err != nil {
		t.Fatalf("UpdateUserEmail: %v", err)
	}
	got, err = s.GetUser(ctx(), "ev-user")
	if err != nil {
		t.Fatalf("GetUser after email change: %v", err)
	}
	if got.Email != "new@example.com" {
		t.Fatalf("email = %q, want new@example.com", got.Email)
	}
	if got.EmailVerifiedAt != nil {
		t.Fatal("expected EmailVerifiedAt to be cleared after email change")
	}
}

// ---------------------------------------------------------------------------
// Email tokens (verification / password reset)
// ---------------------------------------------------------------------------

func TestEmailTokenLifecycle(t *testing.T) {
	s, cleanup := tempDB(t)
	defer cleanup()

	mustUser(t, s, types.User{ID: "et-user", CreatedAt: time.Now().UTC()})
	expiresAt := time.Now().UTC().Add(15 * time.Minute).Format(time.RFC3339)

	if err := s.CreateEmailToken(ctx(), "tok-1", "et-user", "email_verify", expiresAt); err != nil {
		t.Fatalf("CreateEmailToken: %v", err)
	}
	gotUserID, err := s.ConsumeEmailToken(ctx(), "tok-1", "email_verify")
	if err != nil {
		t.Fatalf("ConsumeEmailToken: %v", err)
	}
	if gotUserID != "et-user" {
		t.Fatalf("userID = %q, want et-user", gotUserID)
	}
	if _, err := s.ConsumeEmailToken(ctx(), "tok-1", "email_verify"); !errors.Is(err, types.ErrNotFound) {
		t.Fatalf("expected ErrNotFound on re-consume, got %v", err)
	}

	// Purpose mismatch does not delete the row; it can still be consumed
	// with the correct purpose afterward.
	if err := s.CreateEmailToken(ctx(), "tok-2", "et-user", "password_reset", expiresAt); err != nil {
		t.Fatalf("CreateEmailToken #2: %v", err)
	}
	if _, err := s.ConsumeEmailToken(ctx(), "tok-2", "email_verify"); !errors.Is(err, types.ErrNotFound) {
		t.Fatalf("expected ErrNotFound for purpose mismatch, got %v", err)
	}
	if _, err := s.ConsumeEmailToken(ctx(), "tok-2", "password_reset"); err != nil {
		t.Fatalf("ConsumeEmailToken (correct purpose): %v", err)
	}

	// Expired tokens are deleted and reported as not found.
	pastExpiry := time.Now().UTC().Add(-1 * time.Minute).Format(time.RFC3339)
	if err := s.CreateEmailToken(ctx(), "tok-3", "et-user", "magic_signin", pastExpiry); err != nil {
		t.Fatalf("CreateEmailToken #3: %v", err)
	}
	if _, err := s.ConsumeEmailToken(ctx(), "tok-3", "magic_signin"); !errors.Is(err, types.ErrNotFound) {
		t.Fatalf("expected ErrNotFound for expired token, got %v", err)
	}

	if _, err := s.ConsumeEmailToken(ctx(), "nonexistent", "email_verify"); !errors.Is(err, types.ErrNotFound) {
		t.Fatalf("expected ErrNotFound for nonexistent token, got %v", err)
	}
}

func TestDeleteEmailTokensByUserAndPurpose(t *testing.T) {
	s, cleanup := tempDB(t)
	defer cleanup()

	mustUser(t, s, types.User{ID: "det-user", CreatedAt: time.Now().UTC()})
	expiresAt := time.Now().UTC().Add(15 * time.Minute).Format(time.RFC3339)

	if err := s.CreateEmailToken(ctx(), "tok-4", "det-user", "email_verify", expiresAt); err != nil {
		t.Fatalf("CreateEmailToken: %v", err)
	}
	if err := s.DeleteEmailTokensByUserAndPurpose(ctx(), "det-user", "email_verify"); err != nil {
		t.Fatalf("DeleteEmailTokensByUserAndPurpose: %v", err)
	}
	if _, err := s.ConsumeEmailToken(ctx(), "tok-4", "email_verify"); !errors.Is(err, types.ErrNotFound) {
		t.Fatalf("expected ErrNotFound after purge, got %v", err)
	}
}

// ---------------------------------------------------------------------------
// WebAuthn handle + user lookup
// ---------------------------------------------------------------------------

func TestWebAuthnHandleLifecycle(t *testing.T) {
	s, cleanup := tempDB(t)
	defer cleanup()

	mustUser(t, s, types.User{ID: "wa-user", CreatedAt: time.Now().UTC()})

	handle, err := s.GetOrCreateWebAuthnHandle(ctx(), "wa-user")
	if err != nil {
		t.Fatalf("GetOrCreateWebAuthnHandle: %v", err)
	}
	if handle == "" {
		t.Fatal("expected non-empty handle")
	}

	again, err := s.GetOrCreateWebAuthnHandle(ctx(), "wa-user")
	if err != nil {
		t.Fatalf("GetOrCreateWebAuthnHandle (idempotent): %v", err)
	}
	if again != handle {
		t.Fatalf("handle changed on second call: %q != %q", again, handle)
	}

	if _, err := s.GetOrCreateWebAuthnHandle(ctx(), "no-such-user"); !errors.Is(err, types.ErrNotFound) {
		t.Fatalf("expected ErrNotFound for unknown user, got %v", err)
	}

	u, err := s.GetUserByWebAuthnHandle(ctx(), handle)
	if err != nil {
		t.Fatalf("GetUserByWebAuthnHandle: %v", err)
	}
	if u.ID != "wa-user" {
		t.Fatalf("user id = %q, want wa-user", u.ID)
	}

	if _, err := s.GetUserByWebAuthnHandle(ctx(), "no-such-handle"); !errors.Is(err, types.ErrNotFound) {
		t.Fatalf("expected ErrNotFound for unknown handle, got %v", err)
	}
}

// ---------------------------------------------------------------------------
// WebAuthn credentials (passkeys)
// ---------------------------------------------------------------------------

func TestWebAuthnCredentialLifecycle(t *testing.T) {
	s, cleanup := tempDB(t)
	defer cleanup()

	mustUser(t, s, types.User{ID: "pk-user", CreatedAt: time.Now().UTC()})

	createdAt := utcNow()
	if err := s.CreateWebAuthnCredential(ctx(), "cred-1", "pk-user", "My Phone", `{"id":"cred-1"}`, 0, createdAt); err != nil {
		t.Fatalf("CreateWebAuthnCredential: %v", err)
	}

	list, err := s.ListWebAuthnCredentials(ctx(), "pk-user")
	if err != nil {
		t.Fatalf("ListWebAuthnCredentials: %v", err)
	}
	if len(list) != 1 || list[0].ID != "cred-1" || list[0].Label != "My Phone" || list[0].LastUsedAt != "" {
		t.Fatalf("unexpected list: %+v", list)
	}

	raw, err := s.GetWebAuthnCredentialsRaw(ctx(), "pk-user")
	if err != nil {
		t.Fatalf("GetWebAuthnCredentialsRaw: %v", err)
	}
	if len(raw) != 1 || raw[0].CredentialJSON != `{"id":"cred-1"}` {
		t.Fatalf("unexpected raw credentials: %+v", raw)
	}

	if err := s.UpdateWebAuthnCredentialOnAuth(ctx(), "cred-1", `{"id":"cred-1","sc":1}`, 1, utcNow()); err != nil {
		t.Fatalf("UpdateWebAuthnCredentialOnAuth: %v", err)
	}
	raw, err = s.GetWebAuthnCredentialsRaw(ctx(), "pk-user")
	if err != nil {
		t.Fatalf("GetWebAuthnCredentialsRaw after update: %v", err)
	}
	if raw[0].CredentialJSON != `{"id":"cred-1","sc":1}` {
		t.Fatalf("credential JSON not updated: %+v", raw)
	}
	list, err = s.ListWebAuthnCredentials(ctx(), "pk-user")
	if err != nil {
		t.Fatalf("ListWebAuthnCredentials after update: %v", err)
	}
	if list[0].LastUsedAt == "" {
		t.Fatal("expected LastUsedAt to be set after auth update")
	}

	if err := s.RenameWebAuthnCredential(ctx(), "wrong-user", "cred-1", "x"); !errors.Is(err, types.ErrNotFound) {
		t.Fatalf("expected ErrNotFound renaming as wrong user, got %v", err)
	}
	if err := s.RenameWebAuthnCredential(ctx(), "pk-user", "cred-1", "Renamed"); err != nil {
		t.Fatalf("RenameWebAuthnCredential: %v", err)
	}
	list, err = s.ListWebAuthnCredentials(ctx(), "pk-user")
	if err != nil {
		t.Fatalf("ListWebAuthnCredentials after rename: %v", err)
	}
	if list[0].Label != "Renamed" {
		t.Fatalf("label = %q, want Renamed", list[0].Label)
	}

	if err := s.DeleteWebAuthnCredential(ctx(), "wrong-user", "cred-1"); !errors.Is(err, types.ErrNotFound) {
		t.Fatalf("expected ErrNotFound deleting as wrong user, got %v", err)
	}
	if err := s.DeleteWebAuthnCredential(ctx(), "pk-user", "cred-1"); err != nil {
		t.Fatalf("DeleteWebAuthnCredential: %v", err)
	}
	list, err = s.ListWebAuthnCredentials(ctx(), "pk-user")
	if err != nil {
		t.Fatalf("ListWebAuthnCredentials after delete: %v", err)
	}
	if len(list) != 0 {
		t.Fatalf("expected no credentials left, got %+v", list)
	}
}

// ---------------------------------------------------------------------------
// WebAuthn ceremony sessions
// ---------------------------------------------------------------------------

func TestWebAuthnSessionLifecycle(t *testing.T) {
	s, cleanup := tempDB(t)
	defer cleanup()

	expiresAt := time.Now().UTC().Add(5 * time.Minute).Format(time.RFC3339)

	// Discoverable login: no user known yet.
	if err := s.CreateWebAuthnSession(ctx(), "wsess-1", "", `{"challenge":"abc"}`, expiresAt); err != nil {
		t.Fatalf("CreateWebAuthnSession (discoverable): %v", err)
	}
	gotUserID, gotJSON, err := s.ConsumeWebAuthnSession(ctx(), "wsess-1")
	if err != nil {
		t.Fatalf("ConsumeWebAuthnSession: %v", err)
	}
	if gotUserID != "" || gotJSON != `{"challenge":"abc"}` {
		t.Fatalf("unexpected consume result: userID=%q json=%q", gotUserID, gotJSON)
	}
	if _, _, err := s.ConsumeWebAuthnSession(ctx(), "wsess-1"); !errors.Is(err, types.ErrNotFound) {
		t.Fatalf("expected ErrNotFound on re-consume, got %v", err)
	}

	mustUser(t, s, types.User{ID: "wac-user", CreatedAt: time.Now().UTC()})
	if err := s.CreateWebAuthnSession(ctx(), "wsess-2", "wac-user", `{"x":1}`, expiresAt); err != nil {
		t.Fatalf("CreateWebAuthnSession (known user): %v", err)
	}
	gotUserID, _, err = s.ConsumeWebAuthnSession(ctx(), "wsess-2")
	if err != nil {
		t.Fatalf("ConsumeWebAuthnSession (known user): %v", err)
	}
	if gotUserID != "wac-user" {
		t.Fatalf("userID = %q, want wac-user", gotUserID)
	}

	pastExpiry := time.Now().UTC().Add(-1 * time.Minute).Format(time.RFC3339)
	if err := s.CreateWebAuthnSession(ctx(), "wsess-3", "", `{}`, pastExpiry); err != nil {
		t.Fatalf("CreateWebAuthnSession (expired): %v", err)
	}
	if _, _, err := s.ConsumeWebAuthnSession(ctx(), "wsess-3"); !errors.Is(err, types.ErrNotFound) {
		t.Fatalf("expected ErrNotFound for expired session, got %v", err)
	}

	if _, _, err := s.ConsumeWebAuthnSession(ctx(), "nonexistent"); !errors.Is(err, types.ErrNotFound) {
		t.Fatalf("expected ErrNotFound for nonexistent session, got %v", err)
	}
}

// ---------------------------------------------------------------------------
// MFA email codes
// ---------------------------------------------------------------------------

func TestMFAEmailCodeLifecycle(t *testing.T) {
	s, cleanup := tempDB(t)
	defer cleanup()

	mustUser(t, s, types.User{ID: "mfae-user", CreatedAt: time.Now().UTC()})
	expiresAt := time.Now().UTC().Add(10 * time.Minute).Format(time.RFC3339)

	if err := s.UpsertMFAEmailCode(ctx(), "mfae-user", "codehash1", expiresAt); err != nil {
		t.Fatalf("UpsertMFAEmailCode: %v", err)
	}
	gotHash, gotExpiry, attempts, err := s.GetMFAEmailCode(ctx(), "mfae-user")
	if err != nil {
		t.Fatalf("GetMFAEmailCode: %v", err)
	}
	if gotHash != "codehash1" || gotExpiry != expiresAt || attempts != 0 {
		t.Fatalf("unexpected code: hash=%q expiry=%q attempts=%d", gotHash, gotExpiry, attempts)
	}

	if err := s.IncrementMFAEmailCodeAttempts(ctx(), "mfae-user"); err != nil {
		t.Fatalf("IncrementMFAEmailCodeAttempts: %v", err)
	}
	_, _, attempts, err = s.GetMFAEmailCode(ctx(), "mfae-user")
	if err != nil {
		t.Fatalf("GetMFAEmailCode after increment: %v", err)
	}
	if attempts != 1 {
		t.Fatalf("attempts = %d, want 1", attempts)
	}

	// Overwrite resets attempts.
	if err := s.UpsertMFAEmailCode(ctx(), "mfae-user", "codehash2", expiresAt); err != nil {
		t.Fatalf("UpsertMFAEmailCode (overwrite): %v", err)
	}
	gotHash, _, attempts, err = s.GetMFAEmailCode(ctx(), "mfae-user")
	if err != nil {
		t.Fatalf("GetMFAEmailCode after overwrite: %v", err)
	}
	if gotHash != "codehash2" || attempts != 0 {
		t.Fatalf("hash=%q attempts=%d after overwrite, want codehash2/0", gotHash, attempts)
	}

	if err := s.DeleteMFAEmailCode(ctx(), "mfae-user"); err != nil {
		t.Fatalf("DeleteMFAEmailCode: %v", err)
	}
	if _, _, _, err := s.GetMFAEmailCode(ctx(), "mfae-user"); !errors.Is(err, types.ErrNotFound) {
		t.Fatalf("expected ErrNotFound after delete, got %v", err)
	}
	if _, _, _, err := s.GetMFAEmailCode(ctx(), "nonexistent"); !errors.Is(err, types.ErrNotFound) {
		t.Fatalf("expected ErrNotFound for nonexistent user, got %v", err)
	}
}
