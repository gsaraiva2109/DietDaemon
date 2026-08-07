package store

import (
	"testing"
	"time"

	"github.com/gsaraiva2109/dietdaemon/core/types"
)

// TestListUsersWithCap verifies ListUsers returns every user (id-ordered),
// which stays correct now that the query carries a LIMIT %d safety cap
// (maxListRows, see store_helpers.go). The cap is a defense-in-depth
// ceiling — see Fix 3 in the DB query audit — not exercised at full scale
// here since this project runs at personal-tracker scale.
func TestListUsersWithCap(t *testing.T) {
	s, cleanup := tempDB(t)
	defer cleanup()

	users, err := s.ListUsers(ctx())
	if err != nil {
		t.Fatalf("ListUsers (empty): %v", err)
	}
	if len(users) != 0 {
		t.Fatalf("ListUsers (empty) = %d users, want 0", len(users))
	}

	want := []types.User{
		{ID: "lu-1", Email: "lu1@example.com", DisplayName: "User One", Timezone: "UTC", CreatedAt: time.Now().UTC()},
		{ID: "lu-2", Email: "lu2@example.com", DisplayName: "User Two", Timezone: "UTC", CreatedAt: time.Now().UTC()},
		{ID: "lu-3", Email: "lu3@example.com", DisplayName: "User Three", Timezone: "UTC", CreatedAt: time.Now().UTC()},
	}
	for _, u := range want {
		mustUser(t, s, u)
	}

	users, err = s.ListUsers(ctx())
	if err != nil {
		t.Fatalf("ListUsers: %v", err)
	}
	if len(users) != len(want) {
		t.Fatalf("ListUsers = %d users, want %d", len(users), len(want))
	}

	// ORDER BY id, so the result should come back lu-1, lu-2, lu-3.
	for i, u := range users {
		if u.ID != want[i].ID || u.Email != want[i].Email {
			t.Fatalf("user[%d] = %+v, want id=%q email=%q", i, u, want[i].ID, want[i].Email)
		}
	}
}

// TestListUsersExcludesPendingDeletionAccounts pins down #277's store-level
// fix: a user whose account has been soft-deleted (RequestAccountDeletion)
// must not appear in ListUsers, since the scheduler and backup both drive
// off this query with no filter of their own.
func TestListUsersExcludesPendingDeletionAccounts(t *testing.T) {
	s, cleanup := tempDB(t)
	defer cleanup()

	active := types.User{ID: "ld-active", Email: "active@example.com", DisplayName: "Active", Timezone: "UTC", CreatedAt: time.Now().UTC()}
	deleted := types.User{ID: "ld-deleted", Email: "deleted@example.com", DisplayName: "Deleted", Timezone: "UTC", CreatedAt: time.Now().UTC()}
	mustUser(t, s, active)
	mustUser(t, s, deleted)

	if err := s.RequestAccountDeletion(ctx(), deleted.ID); err != nil {
		t.Fatalf("RequestAccountDeletion: %v", err)
	}

	users, err := s.ListUsers(ctx())
	if err != nil {
		t.Fatalf("ListUsers: %v", err)
	}
	if len(users) != 1 || users[0].ID != active.ID {
		t.Fatalf("ListUsers = %+v, want only %q", users, active.ID)
	}
}
