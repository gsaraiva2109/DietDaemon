package store

import (
	"database/sql"
	"fmt"
	"strings"
	"sync/atomic"
	"time"

	"github.com/gsaraiva2109/dietdaemon/core/types"
)

// scanUser scans a single *sql.Row with the same column order as userRow.
// Kept for store_auth.go call sites (CreateUserWithPassword, GetUserByEmail,
// GetUserByAPIKey, GetSession-adjacent lookups) that build the row via a
// custom SELECT elsewhere in that file.
func scanUser(row *sql.Row) (types.User, error) {
	var r userRow
	if err := row.Scan(&r.ID, &r.AccountID, &r.Email, &r.EmailVerifiedAt, &r.Status, &r.DisplayName, &r.Timezone, &r.Locale, &r.CreatedAt, &r.WebAuthnHandle); err != nil {
		return types.User{}, err
	}
	return r.toUser(), nil
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func utcStr(t time.Time) string { return t.UTC().Format(time.RFC3339) }

func utcNow() string { return time.Now().UTC().Format(time.RFC3339) }

// nullStr returns nil for an empty string, otherwise returns the string.
// Used to store nullable TEXT columns as SQL NULL instead of "".
func nullStr(s string) any {
	if s == "" {
		return nil
	}
	return s
}

func parseUTC(s string) time.Time {
	if s == "" {
		return time.Time{}
	}
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		return time.Time{}
	}
	return t.UTC()
}

// ptrTime returns a pointer to t.
func ptrTime(t time.Time) *time.Time {
	p := new(time.Time)
	*p = t
	return p
}

// isUniqueViolation reports whether err is a SQL UNIQUE constraint violation.
// Works with modernc.org/sqlite; kept simple and portable.
func isUniqueViolation(err error) bool {
	if err == nil {
		return false
	}
	// modernc.org/sqlite surfaces this in the error string.
	return strings.Contains(err.Error(), "UNIQUE constraint failed")
}

// newID returns a short pseudo-unique ID using a monotonic counter + timestamp
// fallback. Simple identifiers keep the embedded DB readable.
var idCounter int64

func newID() string {
	n := atomic.AddInt64(&idCounter, 1)
	return fmt.Sprintf("%d%x", time.Now().UnixNano(), n)
}
