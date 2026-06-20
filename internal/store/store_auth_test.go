package store

import (
	"testing"
	"time"

	"github.com/gsaraiva2109/dietdaemon/core/types"
)

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
	if err != types.ErrNotFound {
		t.Fatalf("expected ErrNotFound on second consume, got %v", err)
	}
}

func TestOIDCStateLinkFlow(t *testing.T) {
	s, cleanup := tempDB(t)
	defer cleanup()

	id := "link-state"
	linkUserID := "user-42"
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
	if err != types.ErrNotFound {
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
	if err != types.ErrIdentityLinked {
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
	if err != types.ErrIdentityLinked {
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
	if err != types.ErrNotFound {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}

	// Create user with OIDC.
	u, err := s.CreateUserWithOIDC(ctx(), "acct-oidc", "user-oidc", "oidc@example.com", "OIDC User", "id-oidc-1", "google", "sub-123")
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
	if err != types.ErrNotFound {
		t.Fatalf("expected ErrNotFound for wrong user, got %v", err)
	}

	// Delete nonexistent → ErrNotFound.
	err = s.DeleteOIDCIdentity(ctx(), "ud-1", "nonexistent")
	if err != types.ErrNotFound {
		t.Fatalf("expected ErrNotFound for nonexistent, got %v", err)
	}
}
