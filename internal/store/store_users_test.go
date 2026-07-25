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
