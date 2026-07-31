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

	u, sess := seedAccountForDeletion(t, s)

	// Not-found case first: deleting a nonexistent user must error.
	if err := s.DeleteAccount(ctx(), "no-such-user"); !errors.Is(err, types.ErrNotFound) {
		t.Fatalf("DeleteAccount(missing user) = %v; want types.ErrNotFound", err)
	}

	if err := s.DeleteAccount(ctx(), u.ID); err != nil {
		t.Fatalf("DeleteAccount: %v", err)
	}

	assertAccountDataDeleted(t, s, u, sess)
	assertAuditRowSurvivesDeletion(t, s, "audit1")
}

// seedAccountForDeletion creates a user with data across several per-user
// tables (weight, meals, sessions, photos, audit log), used by
// TestDeleteAccount to verify DeleteAccount cascades across all of them.
func seedAccountForDeletion(t *testing.T, s *Store) (types.User, auth.Session) {
	t.Helper()

	u, err := s.CreateUserWithPassword(ctx(), "acct-del", "user-del", "del@example.com", "Del User", "$argon2id$dummy")
	if err != nil {
		t.Fatalf("CreateUserWithPassword: %v", err)
	}

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

	return u, sess
}

// assertAccountDataDeleted checks that DeleteAccount cascaded across every
// per-user table seeded by seedAccountForDeletion.
func assertAccountDataDeleted(t *testing.T, s *Store, u types.User, sess auth.Session) {
	t.Helper()

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
}

// assertAuditRowSurvivesDeletion checks that the audit row identified by
// auditID survives DeleteAccount (ON DELETE SET NULL, by design), with
// account_id/user_id cleared rather than the row being cascaded away.
func assertAuditRowSurvivesDeletion(t *testing.T, s *Store, auditID string) {
	t.Helper()

	var event string
	var accountID, userID sql.NullString
	err := s.db.QueryRow(`SELECT event, account_id, user_id FROM auth_audit_log WHERE id = ?`, auditID).
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

// TestReactivateAccountNeverDeletedIsNoOp pins down ReactivateAccount's
// documented behavior on an account that was never soft-deleted: it's a
// deliberate no-op on deleted_at (stays NULL), but it still unconditionally
// writes the account.reactivated audit event -- ReactivateAccount does not
// check whether the account was actually pending deletion before acting.
func TestReactivateAccountNeverDeletedIsNoOp(t *testing.T) {
	s, cleanup := tempDB(t)
	defer cleanup()

	u, err := s.CreateUserWithPassword(ctx(), "acct-never-deleted", "user-never-deleted", "neverdeleted@example.com", "User", "$argon2id$dummy")
	if err != nil {
		t.Fatalf("CreateUserWithPassword: %v", err)
	}

	if err := s.ReactivateAccount(ctx(), u.ID); err != nil {
		t.Fatalf("ReactivateAccount: %v", err)
	}

	status, err := s.AccountDeletionStatus(ctx(), u.ID)
	if err != nil {
		t.Fatalf("AccountDeletionStatus: %v", err)
	}
	if status.DeletedAt != nil {
		t.Fatalf("AccountDeletionStatus.DeletedAt = %v; want nil (was never deleted)", status.DeletedAt)
	}

	has, err := s.HasAuditEvent(ctx(), u.AccountID, "account.reactivated")
	if err != nil {
		t.Fatalf("HasAuditEvent: %v", err)
	}
	if !has {
		t.Fatal("HasAuditEvent(account.reactivated) = false; want true even on a no-op reactivation")
	}
}

// TestReactivateAccountDoubleCallIsIdempotent pins down that calling
// ReactivateAccount twice in a row succeeds both times with deleted_at left
// cleared -- the second call is a no-op like
// TestReactivateAccountNeverDeletedIsNoOp above, just reached via a real
// soft-delete instead of an account that was never deleted.
func TestReactivateAccountDoubleCallIsIdempotent(t *testing.T) {
	s, cleanup := tempDB(t)
	defer cleanup()

	u, err := s.CreateUserWithPassword(ctx(), "acct-double-reactivate", "user-double-reactivate", "doublereactivate@example.com", "User", "$argon2id$dummy")
	if err != nil {
		t.Fatalf("CreateUserWithPassword: %v", err)
	}

	if err := s.RequestAccountDeletion(ctx(), u.ID); err != nil {
		t.Fatalf("RequestAccountDeletion: %v", err)
	}

	if err := s.ReactivateAccount(ctx(), u.ID); err != nil {
		t.Fatalf("first ReactivateAccount: %v", err)
	}
	if err := s.ReactivateAccount(ctx(), u.ID); err != nil {
		t.Fatalf("second ReactivateAccount: %v", err)
	}

	status, err := s.AccountDeletionStatus(ctx(), u.ID)
	if err != nil {
		t.Fatalf("AccountDeletionStatus: %v", err)
	}
	if status.DeletedAt != nil {
		t.Fatalf("AccountDeletionStatus.DeletedAt = %v; want nil after two reactivations", status.DeletedAt)
	}
}

// TestReactivateAccountAfterHardPurgeReturnsNotFound is a regression test for
// the day-90 tier cascading away the users row: once PurgeAccount has run,
// the account owning userID no longer exists, so a later ReactivateAccount
// call must hit the same types.ErrNotFound path as any other missing user --
// no code change is needed for this to pass, it's here to pin the behavior
// down against a future change.
func TestReactivateAccountAfterHardPurgeReturnsNotFound(t *testing.T) {
	s, cleanup := tempDB(t)
	defer cleanup()

	u, err := s.CreateUserWithPassword(ctx(), "acct-reactivate-after-purge", "user-reactivate-after-purge", "reactivateafterpurge@example.com", "User", "$argon2id$dummy")
	if err != nil {
		t.Fatalf("CreateUserWithPassword: %v", err)
	}

	if err := s.RequestAccountDeletion(ctx(), u.ID); err != nil {
		t.Fatalf("RequestAccountDeletion: %v", err)
	}
	if err := s.PurgeAccount(ctx(), u.AccountID); err != nil {
		t.Fatalf("PurgeAccount: %v", err)
	}

	if err := s.ReactivateAccount(ctx(), u.ID); !errors.Is(err, types.ErrNotFound) {
		t.Fatalf("ReactivateAccount after hard purge = %v; want types.ErrNotFound", err)
	}
}

// backdateDeletedAt rewrites accounts.deleted_at directly, simulating an
// account soft-deleted longer ago than "now" (the purge job's cutoffs are
// relative to real time, so tests can't just wait 30/90 days).
func backdateDeletedAt(t *testing.T, s *Store, accountID string, when time.Time) {
	t.Helper()
	if _, err := s.db.Exec(`UPDATE accounts SET deleted_at = ? WHERE id = ?`, when.UTC().Format(time.RFC3339), accountID); err != nil {
		t.Fatalf("backdate deleted_at: %v", err)
	}
}

// TestPurgeAccountPhotos verifies the day-30 photo-purge tier: an account
// past the cutoff shows up in ListAccountsPendingPhotoPurge, PurgeAccountPhotos
// deletes its photos and sets photos_purged_at (idempotently dropping it from
// future listings), and the purge is audited.
func TestPurgeAccountPhotos(t *testing.T) {
	s, cleanup := tempDB(t)
	defer cleanup()

	u, err := s.CreateUserWithPassword(ctx(), "acct-photo-purge", "user-photo-purge", "photopurge@example.com", "User", "$argon2id$dummy")
	if err != nil {
		t.Fatalf("CreateUserWithPassword: %v", err)
	}
	if err := s.UploadPhoto(ctx(), types.ProgressPhoto{ID: "pp1", UserID: u.ID, Date: "2026-07-01", View: "front", MimeType: "image/png", Data: []byte("fake")}); err != nil {
		t.Fatalf("UploadPhoto: %v", err)
	}
	if err := s.RequestAccountDeletion(ctx(), u.ID); err != nil {
		t.Fatalf("RequestAccountDeletion: %v", err)
	}
	backdateDeletedAt(t, s, u.AccountID, time.Now().UTC().AddDate(0, 0, -31))

	ids, err := s.ListAccountsPendingPhotoPurge(ctx(), time.Now().UTC().AddDate(0, 0, -30))
	if err != nil {
		t.Fatalf("ListAccountsPendingPhotoPurge: %v", err)
	}
	if len(ids) != 1 || ids[0] != u.AccountID {
		t.Fatalf("ListAccountsPendingPhotoPurge = %v; want [%s]", ids, u.AccountID)
	}

	if err := s.PurgeAccountPhotos(ctx(), u.AccountID); err != nil {
		t.Fatalf("PurgeAccountPhotos: %v", err)
	}

	photos, err := s.ListPhotoMetadata(ctx(), u.ID)
	if err != nil {
		t.Fatalf("ListPhotoMetadata: %v", err)
	}
	if len(photos) != 0 {
		t.Fatalf("ListPhotoMetadata after purge = %d; want 0", len(photos))
	}

	status, err := s.AccountDeletionStatus(ctx(), u.ID)
	if err != nil {
		t.Fatalf("AccountDeletionStatus: %v", err)
	}
	if status.PhotosPurgedAt == nil {
		t.Fatalf("AccountDeletionStatus.PhotosPurgedAt = nil; want set after purge")
	}

	has, err := s.HasAuditEvent(ctx(), u.AccountID, "account.photos_purged")
	if err != nil {
		t.Fatalf("HasAuditEvent: %v", err)
	}
	if !has {
		t.Fatal("HasAuditEvent(account.photos_purged) = false; want true")
	}

	// Idempotent: photos_purged_at now set, so it drops out of the listing.
	ids, err = s.ListAccountsPendingPhotoPurge(ctx(), time.Now().UTC().AddDate(0, 0, -30))
	if err != nil {
		t.Fatalf("ListAccountsPendingPhotoPurge (after purge): %v", err)
	}
	if len(ids) != 0 {
		t.Fatalf("ListAccountsPendingPhotoPurge after purge = %v; want empty", ids)
	}
}

// TestPurgeAccountPhotosSkipsReactivatedAccount is a regression test for the
// TOCTOU race: if a job lists an account as pending photo purge, the owner
// reactivates before the job runs, and the job then calls PurgeAccountPhotos
// anyway, the guarded UPDATE (deleted_at IS NOT NULL) must match zero rows
// and the whole call must be a no-op -- photos untouched, no audit event.
func TestPurgeAccountPhotosSkipsReactivatedAccount(t *testing.T) {
	s, cleanup := tempDB(t)
	defer cleanup()

	u, err := s.CreateUserWithPassword(ctx(), "acct-photo-purge-race", "user-photo-purge-race", "photopurgerace@example.com", "User", "$argon2id$dummy")
	if err != nil {
		t.Fatalf("CreateUserWithPassword: %v", err)
	}
	if err := s.UploadPhoto(ctx(), types.ProgressPhoto{ID: "pp-race", UserID: u.ID, Date: "2026-07-01", View: "front", MimeType: "image/png", Data: []byte("fake")}); err != nil {
		t.Fatalf("UploadPhoto: %v", err)
	}
	if err := s.RequestAccountDeletion(ctx(), u.ID); err != nil {
		t.Fatalf("RequestAccountDeletion: %v", err)
	}
	backdateDeletedAt(t, s, u.AccountID, time.Now().UTC().AddDate(0, 0, -31))

	// Owner reactivates in the interim, clearing deleted_at, before the purge
	// job's call reaches the store.
	if err := s.ReactivateAccount(ctx(), u.ID); err != nil {
		t.Fatalf("ReactivateAccount: %v", err)
	}

	if err := s.PurgeAccountPhotos(ctx(), u.AccountID); err != nil {
		t.Fatalf("PurgeAccountPhotos: %v", err)
	}

	photos, err := s.ListPhotoMetadata(ctx(), u.ID)
	if err != nil {
		t.Fatalf("ListPhotoMetadata: %v", err)
	}
	if len(photos) != 1 {
		t.Fatalf("ListPhotoMetadata after skipped purge = %d; want 1 (untouched)", len(photos))
	}

	status, err := s.AccountDeletionStatus(ctx(), u.ID)
	if err != nil {
		t.Fatalf("AccountDeletionStatus: %v", err)
	}
	if status.PhotosPurgedAt != nil {
		t.Fatalf("AccountDeletionStatus.PhotosPurgedAt = %v; want nil (purge skipped)", status.PhotosPurgedAt)
	}

	has, err := s.HasAuditEvent(ctx(), u.AccountID, "account.photos_purged")
	if err != nil {
		t.Fatalf("HasAuditEvent: %v", err)
	}
	if has {
		t.Fatal("HasAuditEvent(account.photos_purged) = true; want false, purge was skipped")
	}
}

// TestPurgeAccount verifies the day-90 full-purge tier: an account past the
// cutoff shows up in ListAccountsPastDeletion, PurgeAccount hard-deletes it,
// and the purge event survives in auth_audit_log with account_id nulled.
func TestPurgeAccount(t *testing.T) {
	s, cleanup := tempDB(t)
	defer cleanup()

	u, err := s.CreateUserWithPassword(ctx(), "acct-full-purge", "user-full-purge", "fullpurge@example.com", "User", "$argon2id$dummy")
	if err != nil {
		t.Fatalf("CreateUserWithPassword: %v", err)
	}
	if err := s.RequestAccountDeletion(ctx(), u.ID); err != nil {
		t.Fatalf("RequestAccountDeletion: %v", err)
	}
	backdateDeletedAt(t, s, u.AccountID, time.Now().UTC().AddDate(0, 0, -91))

	ids, err := s.ListAccountsPastDeletion(ctx(), time.Now().UTC().AddDate(0, 0, -90))
	if err != nil {
		t.Fatalf("ListAccountsPastDeletion: %v", err)
	}
	if len(ids) != 1 || ids[0] != u.AccountID {
		t.Fatalf("ListAccountsPastDeletion = %v; want [%s]", ids, u.AccountID)
	}

	if err := s.PurgeAccount(ctx(), u.AccountID); err != nil {
		t.Fatalf("PurgeAccount: %v", err)
	}

	if _, err := s.GetUser(ctx(), u.ID); !errors.Is(err, types.ErrNotFound) {
		t.Fatalf("GetUser after purge = %v; want types.ErrNotFound", err)
	}

	// The audit row survives (ON DELETE SET NULL), account_id cleared —
	// mirrors the DeleteAccount audit-survival assertion above.
	var event string
	var accountID sql.NullString
	err = s.db.QueryRow(`SELECT event, account_id FROM auth_audit_log WHERE event = 'account.delete.purged'`).
		Scan(&event, &accountID)
	if err != nil {
		t.Fatalf("query purge audit row: %v", err)
	}
	if accountID.Valid {
		t.Fatalf("purge audit row account_id = %v; want NULL", accountID)
	}
}

// TestPurgeAccountSkipsReactivatedAccount is a regression test for the same
// TOCTOU race as TestPurgeAccountPhotosSkipsReactivatedAccount, at the
// day-90 hard-purge tier: if the owner reactivates before a queued
// PurgeAccount call reaches the store, the guarded claim (deleted_at IS NOT
// NULL) must match zero rows, leaving the account row (and its data) intact
// with no spurious account.delete.purged audit event.
func TestPurgeAccountSkipsReactivatedAccount(t *testing.T) {
	s, cleanup := tempDB(t)
	defer cleanup()

	u, err := s.CreateUserWithPassword(ctx(), "acct-full-purge-race", "user-full-purge-race", "fullpurgerace@example.com", "User", "$argon2id$dummy")
	if err != nil {
		t.Fatalf("CreateUserWithPassword: %v", err)
	}
	if err := s.RequestAccountDeletion(ctx(), u.ID); err != nil {
		t.Fatalf("RequestAccountDeletion: %v", err)
	}
	backdateDeletedAt(t, s, u.AccountID, time.Now().UTC().AddDate(0, 0, -91))

	// Owner reactivates in the interim, clearing deleted_at, before the purge
	// job's call reaches the store.
	if err := s.ReactivateAccount(ctx(), u.ID); err != nil {
		t.Fatalf("ReactivateAccount: %v", err)
	}

	if err := s.PurgeAccount(ctx(), u.AccountID); err != nil {
		t.Fatalf("PurgeAccount: %v", err)
	}

	if _, err := s.GetUser(ctx(), u.ID); err != nil {
		t.Fatalf("GetUser after skipped purge: %v; want account still present", err)
	}

	has, err := s.HasAuditEvent(ctx(), u.AccountID, "account.delete.purged")
	if err != nil {
		t.Fatalf("HasAuditEvent: %v", err)
	}
	if has {
		t.Fatal("HasAuditEvent(account.delete.purged) = true; want false, purge was skipped")
	}
}

// TestAccountEmails verifies AccountEmails collects every user's email under
// an account, and HasAuditEvent reports false for an event never written.
func TestAccountEmails(t *testing.T) {
	s, cleanup := tempDB(t)
	defer cleanup()

	u1, err := s.CreateUserWithPassword(ctx(), "acct-emails", "user-emails-1", "emails1@example.com", "User One", "$argon2id$dummy")
	if err != nil {
		t.Fatalf("CreateUserWithPassword u1: %v", err)
	}
	u2, err := s.CreateUserWithPassword(ctx(), "acct-emails", "user-emails-2", "emails2@example.com", "User Two", "$argon2id$dummy")
	if err != nil {
		t.Fatalf("CreateUserWithPassword u2: %v", err)
	}

	emails, err := s.AccountEmails(ctx(), u1.AccountID)
	if err != nil {
		t.Fatalf("AccountEmails: %v", err)
	}
	want := map[string]bool{u1.Email: true, u2.Email: true}
	if len(emails) != 2 || !want[emails[0]] || !want[emails[1]] {
		t.Fatalf("AccountEmails = %v; want %v", emails, want)
	}

	has, err := s.HasAuditEvent(ctx(), u1.AccountID, "account.delete.requested")
	if err != nil {
		t.Fatalf("HasAuditEvent: %v", err)
	}
	if has {
		t.Fatal("HasAuditEvent(account.delete.requested) = true; want false (never written)")
	}
}

// TestAccountDeletionStatusNotFound covers AccountDeletionStatus's
// types.ErrNotFound branch for a userID with no matching account/user row.
func TestAccountDeletionStatusNotFound(t *testing.T) {
	s, cleanup := tempDB(t)
	defer cleanup()

	if _, err := s.AccountDeletionStatus(ctx(), "no-such-user"); !errors.Is(err, types.ErrNotFound) {
		t.Fatalf("AccountDeletionStatus(missing user) = %v; want types.ErrNotFound", err)
	}
}

// TestAccountDeletedAt covers the narrow AccountDeletedAt view used by
// internal/backup: nil for an active account, set after a deletion request,
// and types.ErrNotFound passed through unchanged for a missing user.
func TestAccountDeletedAt(t *testing.T) {
	s, cleanup := tempDB(t)
	defer cleanup()

	u, err := s.CreateUserWithPassword(ctx(), "acct-deletedat", "user-deletedat", "deletedat@example.com", "User", "$argon2id$dummy")
	if err != nil {
		t.Fatalf("CreateUserWithPassword: %v", err)
	}

	deletedAt, err := s.AccountDeletedAt(ctx(), u.ID)
	if err != nil {
		t.Fatalf("AccountDeletedAt: %v", err)
	}
	if deletedAt != nil {
		t.Fatalf("AccountDeletedAt = %v; want nil for an active account", deletedAt)
	}

	if err := s.RequestAccountDeletion(ctx(), u.ID); err != nil {
		t.Fatalf("RequestAccountDeletion: %v", err)
	}

	deletedAt, err = s.AccountDeletedAt(ctx(), u.ID)
	if err != nil {
		t.Fatalf("AccountDeletedAt: %v", err)
	}
	if deletedAt == nil {
		t.Fatalf("AccountDeletedAt = nil; want set after RequestAccountDeletion")
	}

	if _, err := s.AccountDeletedAt(ctx(), "no-such-user"); !errors.Is(err, types.ErrNotFound) {
		t.Fatalf("AccountDeletedAt(missing user) = %v; want types.ErrNotFound", err)
	}
}

// TestAccountMethodsWrapDBErrorsWhenClosed exercises the "if err != nil {
// return fmt.Errorf(...) }" wrapper branch guarding every DB call in these
// methods, by closing the store's real DB connection first -- BeginTx/Query/
// QueryRow then fail honestly, without a mocking dependency.
func TestAccountMethodsWrapDBErrorsWhenClosed(t *testing.T) {
	s, cleanup := tempDB(t)
	defer cleanup()
	if err := s.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	if err := s.RequestAccountDeletion(ctx(), "any"); err == nil {
		t.Error("RequestAccountDeletion on closed db: want error")
	}
	if err := s.ReactivateAccount(ctx(), "any"); err == nil {
		t.Error("ReactivateAccount on closed db: want error")
	}
	if err := s.PurgeAccountPhotos(ctx(), "any"); err == nil {
		t.Error("PurgeAccountPhotos on closed db: want error")
	}
	if err := s.PurgeAccount(ctx(), "any"); err == nil {
		t.Error("PurgeAccount on closed db: want error")
	}
	if _, err := s.AccountDeletionStatus(ctx(), "any"); err == nil {
		t.Error("AccountDeletionStatus on closed db: want error")
	}
	if _, err := s.ListAccountsPendingPhotoPurge(ctx(), time.Now()); err == nil {
		t.Error("ListAccountsPendingPhotoPurge on closed db: want error")
	}
	if _, err := s.ListAccountsPastDeletion(ctx(), time.Now()); err == nil {
		t.Error("ListAccountsPastDeletion on closed db: want error")
	}
	if _, err := s.HasAuditEvent(ctx(), "any", "ev"); err == nil {
		t.Error("HasAuditEvent on closed db: want error")
	}
	if _, err := s.AccountEmails(ctx(), "any"); err == nil {
		t.Error("AccountEmails on closed db: want error")
	}
}

// TestAccountUserIDsErrors covers accountUserIDs' own two error-wrap
// branches directly (it's unexported but same-package tests can call it):
// a bad query and a column-count mismatch that fails the Scan.
func TestAccountUserIDsErrors(t *testing.T) {
	s, cleanup := tempDB(t)
	defer cleanup()
	if _, err := s.CreateUserWithPassword(ctx(), "acct-uid-err", "user-uid-err", "uiderr@example.com", "User", "$argon2id$dummy"); err != nil {
		t.Fatalf("CreateUserWithPassword: %v", err)
	}

	tx, err := s.db.BeginTx(ctx(), nil)
	if err != nil {
		t.Fatalf("BeginTx: %v", err)
	}
	defer func() { _ = tx.Rollback() }()

	badQuery := func(string) string { return "SELECT * FROM no_such_table_xyz" }
	if _, err := accountUserIDs(ctx(), tx, badQuery, "any"); err == nil {
		t.Error("accountUserIDs with a bad query: want error")
	}

	multiCol := func(string) string { return "SELECT id, account_id FROM users" }
	if _, err := accountUserIDs(ctx(), tx, multiCol, "any"); err == nil {
		t.Error("accountUserIDs with a multi-column query: want scan error")
	}
}

// TestScanAccountIDsScanError covers scanAccountIDs' own error-wrap branch
// directly: a multi-column result set fails its single-destination Scan.
func TestScanAccountIDsScanError(t *testing.T) {
	s, cleanup := tempDB(t)
	defer cleanup()
	if _, err := s.CreateUserWithPassword(ctx(), "acct-scan-err", "user-scan-err", "scanerr@example.com", "User", "$argon2id$dummy"); err != nil {
		t.Fatalf("CreateUserWithPassword: %v", err)
	}

	rows, err := s.db.QueryContext(ctx(), `SELECT id, account_id FROM users`)
	if err != nil {
		t.Fatalf("QueryContext: %v", err)
	}
	defer func() { _ = rows.Close() }()

	if _, err := scanAccountIDs(rows); err == nil {
		t.Error("scanAccountIDs with a multi-column result set: want scan error")
	}
}

// TestInsertAuditEventTxDuplicateID covers insertAuditEventTx's own
// error-wrap branch directly: inserting the same id twice violates the
// auth_audit_log primary key.
func TestInsertAuditEventTxDuplicateID(t *testing.T) {
	s, cleanup := tempDB(t)
	defer cleanup()

	tx, err := s.db.BeginTx(ctx(), nil)
	if err != nil {
		t.Fatalf("BeginTx: %v", err)
	}
	defer func() { _ = tx.Rollback() }()

	ev := types.AuditEvent{ID: "dup-audit-id", Event: "test.event", CreatedAt: time.Now()}
	if err := insertAuditEventTx(ctx(), tx, s.rewrite, ev); err != nil {
		t.Fatalf("first insertAuditEventTx: %v", err)
	}
	if err := insertAuditEventTx(ctx(), tx, s.rewrite, ev); err == nil {
		t.Error("insertAuditEventTx with a duplicate id: want primary-key violation error")
	}
}
