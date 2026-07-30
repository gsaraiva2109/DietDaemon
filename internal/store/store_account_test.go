package store

import (
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/gsaraiva2109/dietdaemon/core/types"
	"github.com/gsaraiva2109/dietdaemon/internal/auth"
)

func TestDeleteAccount(t *testing.T) {
	s, cleanup := tempDB(t)
	defer cleanup()

	u, err := s.CreateUserWithPassword(ctx(), "acct-del", "user-del", "del@example.com", "Del User", "$argon2id$dummy")
	if err != nil {
		t.Fatalf("CreateUserWithPassword: %v", err)
	}

	// Log data across several per-user tables.
	if _, err := s.LogWeight(ctx(), types.WeightEntry{ID: "w1", UserID: u.ID, Date: "2026-07-01", WeightKg: 80}); err != nil {
		t.Fatalf("LogWeight: %v", err)
	}

	meal := types.Meal{
		ID:      "meal1",
		UserID:  u.ID,
		At:      time.Now().UTC(),
		RawText: "ovos",
		Items: []types.ResolvedItem{
			{
				Parsed: types.ParsedItem{RawPhrase: "ovos", Quantity: 2, Unit: "un", NormalizedGrams: 100},
				Match:  types.FoodMatch{FoodID: "ovo-cozido", Name: "Ovo Cozido", Source: "taco", MatchScore: 1.0},
				Macros: types.Macros{Calories: 155, Protein: 13, Carbs: 1.1, Fat: 10.6},
			},
		},
	}
	if err := s.SaveMeal(ctx(), meal); err != nil {
		t.Fatalf("SaveMeal: %v", err)
	}

	sess := auth.Session{
		ID:                "sess1",
		UserID:            u.ID,
		CSRFToken:         "csrf",
		CreatedAt:         time.Now().UTC(),
		LastSeenAt:        time.Now().UTC(),
		IdleExpiresAt:     time.Now().UTC().Add(time.Hour),
		AbsoluteExpiresAt: time.Now().UTC().Add(24 * time.Hour),
	}
	if err := s.CreateSession(ctx(), sess); err != nil {
		t.Fatalf("CreateSession: %v", err)
	}

	if err := s.UploadPhoto(ctx(), types.ProgressPhoto{ID: "photo1", UserID: u.ID, Date: "2026-07-01", View: "front", MimeType: "image/png", Data: []byte("fake")}); err != nil {
		t.Fatalf("UploadPhoto: %v", err)
	}

	if err := s.WriteAuditEvent(ctx(), types.AuditEvent{ID: "audit1", AccountID: u.AccountID, UserID: u.ID, Event: "login.success", CreatedAt: time.Now().UTC()}); err != nil {
		t.Fatalf("WriteAuditEvent: %v", err)
	}

	// Not-found case first: deleting a nonexistent user must error.
	if err := s.DeleteAccount(ctx(), "no-such-user"); !errors.Is(err, types.ErrNotFound) {
		t.Fatalf("DeleteAccount(missing user) = %v; want types.ErrNotFound", err)
	}

	if err := s.DeleteAccount(ctx(), u.ID); err != nil {
		t.Fatalf("DeleteAccount: %v", err)
	}

	if _, err := s.GetUser(ctx(), u.ID); !errors.Is(err, types.ErrNotFound) {
		t.Fatalf("GetUser after delete = %v; want types.ErrNotFound", err)
	}

	weights, err := s.ListWeight(ctx(), u.ID, 365)
	if err != nil {
		t.Fatalf("ListWeight: %v", err)
	}
	if len(weights) != 0 {
		t.Fatalf("ListWeight after delete = %d entries; want 0", len(weights))
	}

	meals, err := s.RecentMeals(ctx(), u.ID, 10)
	if err != nil {
		t.Fatalf("RecentMeals: %v", err)
	}
	if len(meals) != 0 {
		t.Fatalf("RecentMeals after delete = %d; want 0", len(meals))
	}

	if _, err := s.GetSession(ctx(), sess.ID); !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("GetSession after delete = %v; want sql.ErrNoRows", err)
	}

	photos, err := s.ListPhotoMetadata(ctx(), u.ID)
	if err != nil {
		t.Fatalf("ListPhotoMetadata: %v", err)
	}
	if len(photos) != 0 {
		t.Fatalf("ListPhotoMetadata after delete = %d; want 0", len(photos))
	}

	// The audit row must survive (ON DELETE SET NULL, by design), with
	// account_id/user_id cleared rather than the row being cascaded away.
	var event string
	var accountID, userID sql.NullString
	err = s.db.QueryRow(`SELECT event, account_id, user_id FROM auth_audit_log WHERE id = ?`, "audit1").
		Scan(&event, &accountID, &userID)
	if err != nil {
		t.Fatalf("query audit row: %v", err)
	}
	if event != "login.success" {
		t.Fatalf("audit event = %q; want login.success", event)
	}
	if accountID.Valid || userID.Valid {
		t.Fatalf("audit row account_id/user_id = (%v, %v); want both NULL", accountID, userID)
	}
}

// createSessionAndAPIKey seeds a session and API key for u, used by
// TestRequestAccountDeletion to set up revocation fixtures for each user.
func createSessionAndAPIKey(t *testing.T, s *Store, u types.User) {
	t.Helper()
	sess := auth.Session{
		ID:                "sess-" + u.ID,
		UserID:            u.ID,
		CSRFToken:         "csrf",
		CreatedAt:         time.Now().UTC(),
		LastSeenAt:        time.Now().UTC(),
		IdleExpiresAt:     time.Now().UTC().Add(time.Hour),
		AbsoluteExpiresAt: time.Now().UTC().Add(24 * time.Hour),
	}
	if err := s.CreateSession(ctx(), sess); err != nil {
		t.Fatalf("CreateSession(%s): %v", u.ID, err)
	}
	if err := s.CreateAPIKey(ctx(), "key-"+u.ID, u.ID, "hashed-"+u.ID, "label"); err != nil {
		t.Fatalf("CreateAPIKey(%s): %v", u.ID, err)
	}
}

// assertSessionAndKeysRevoked checks that u's session is gone and it has no
// non-revoked API keys, used by TestRequestAccountDeletion to verify
// revocation happened for every user under the account.
func assertSessionAndKeysRevoked(t *testing.T, s *Store, u types.User) {
	t.Helper()
	if _, err := s.GetSession(ctx(), "sess-"+u.ID); !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("GetSession(%s) after deletion request = %v; want sql.ErrNoRows", u.ID, err)
	}
	keys, err := s.ListAPIKeys(ctx(), u.ID)
	if err != nil {
		t.Fatalf("ListAPIKeys(%s): %v", u.ID, err)
	}
	if len(keys) != 0 {
		t.Fatalf("ListAPIKeys(%s) after deletion request = %d non-revoked keys; want 0", u.ID, len(keys))
	}
}

// TestRequestAccountDeletion verifies that requesting deletion revokes every
// session and API key for every user under the account (not just the one
// whose ID was passed in), and leaves the account row and its data intact.
func TestRequestAccountDeletion(t *testing.T) {
	s, cleanup := tempDB(t)
	defer cleanup()

	u1, err := s.CreateUserWithPassword(ctx(), "acct-req-del", "user-req-del-1", "reqdel1@example.com", "User One", "$argon2id$dummy")
	if err != nil {
		t.Fatalf("CreateUserWithPassword u1: %v", err)
	}
	u2, err := s.CreateUserWithPassword(ctx(), "acct-req-del", "user-req-del-2", "reqdel2@example.com", "User Two", "$argon2id$dummy")
	if err != nil {
		t.Fatalf("CreateUserWithPassword u2: %v", err)
	}

	for _, u := range []types.User{u1, u2} {
		createSessionAndAPIKey(t, s, u)
	}

	if err := s.RequestAccountDeletion(ctx(), "no-such-user"); !errors.Is(err, types.ErrNotFound) {
		t.Fatalf("RequestAccountDeletion(missing user) = %v; want types.ErrNotFound", err)
	}

	if err := s.RequestAccountDeletion(ctx(), u1.ID); err != nil {
		t.Fatalf("RequestAccountDeletion: %v", err)
	}

	for _, u := range []types.User{u1, u2} {
		assertSessionAndKeysRevoked(t, s, u)
	}

	// The account and its users still exist (soft delete, not a hard delete).
	if _, err := s.GetUser(ctx(), u1.ID); err != nil {
		t.Fatalf("GetUser(u1) after deletion request: %v", err)
	}

	status, err := s.AccountDeletionStatus(ctx(), u1.ID)
	if err != nil {
		t.Fatalf("AccountDeletionStatus: %v", err)
	}
	if status.DeletedAt == nil {
		t.Fatalf("AccountDeletionStatus.DeletedAt = nil; want set")
	}
	if status.PhotosPurgedAt != nil {
		t.Fatalf("AccountDeletionStatus.PhotosPurgedAt = %v; want nil", status.PhotosPurgedAt)
	}
}

// TestReactivateAccount verifies that reactivation clears deleted_at only,
// leaving a previously-set photos_purged_at untouched.
func TestReactivateAccount(t *testing.T) {
	s, cleanup := tempDB(t)
	defer cleanup()

	u, err := s.CreateUserWithPassword(ctx(), "acct-reactivate", "user-reactivate", "reactivate@example.com", "User", "$argon2id$dummy")
	if err != nil {
		t.Fatalf("CreateUserWithPassword: %v", err)
	}

	if err := s.RequestAccountDeletion(ctx(), u.ID); err != nil {
		t.Fatalf("RequestAccountDeletion: %v", err)
	}

	// Simulate the day-30 photo purge job having already run.
	if _, err := s.db.Exec(`UPDATE accounts SET photos_purged_at = ? WHERE id = ?`, utcNow(), u.AccountID); err != nil {
		t.Fatalf("simulate photo purge: %v", err)
	}

	if err := s.ReactivateAccount(ctx(), "no-such-user"); !errors.Is(err, types.ErrNotFound) {
		t.Fatalf("ReactivateAccount(missing user) = %v; want types.ErrNotFound", err)
	}

	if err := s.ReactivateAccount(ctx(), u.ID); err != nil {
		t.Fatalf("ReactivateAccount: %v", err)
	}

	status, err := s.AccountDeletionStatus(ctx(), u.ID)
	if err != nil {
		t.Fatalf("AccountDeletionStatus: %v", err)
	}
	if status.DeletedAt != nil {
		t.Fatalf("AccountDeletionStatus.DeletedAt = %v; want nil after reactivation", status.DeletedAt)
	}
	if status.PhotosPurgedAt == nil {
		t.Fatalf("AccountDeletionStatus.PhotosPurgedAt = nil; want untouched (still set) after reactivation")
	}
}
