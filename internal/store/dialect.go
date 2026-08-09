// Package store implements ports.Store, with SQLite and Postgres dialect
// implementations.
package store

import (
	"fmt"
	"strings"
	"unicode"

	_ "github.com/lib/pq" // register "postgres" driver for database/sql
)

// Dialect abstracts SQL dialect differences between SQLite and Postgres.
type Dialect interface {
	// Placeholder returns the parameter placeholder for the n-th argument
	// (1-based): "?" for SQLite, "$n" for Postgres.
	Placeholder(n int) string

	// RewritePlaceholders replaces every "?" in sql with the correct positional
	// placeholder for this dialect. SQLite is a no-op; Postgres replaces each ?
	// with $1, $2, ... in order.
	//
	// Assumption: no "?" characters appear inside SQL string literals or comments
	// in this project's queries. All ? are positional parameters.
	RewritePlaceholders(sql string) string

	// Now returns the SQL expression for the current timestamp:
	// "datetime('now')" for SQLite, "NOW()" for Postgres.
	Now() string

	// ColumnExists returns a query that checks whether a column exists in a
	// table. The query should return a single row with COUNT(*) (0 or 1).
	// table and column are the SQL identifiers (already validated, not user
	// input).
	ColumnExists(table, column string) string

	// SearchQuery converts a raw user search string into a dialect-specific
	// full-text query parameter. SQLite returns FTS5 prefix syntax (token*);
	// Postgres returns tsquery prefix syntax (token:* & token2:*).
	SearchQuery(raw string) string
}

// ---------------------------------------------------------------------------
// SQLite dialect
// ---------------------------------------------------------------------------

type sqliteDialect struct{}

func (d sqliteDialect) Placeholder(int) string { return "?" }

func (d sqliteDialect) RewritePlaceholders(sql string) string {
	return sql // no-op: SQLite uses ? natively
}

func (d sqliteDialect) Now() string { return "datetime('now')" }

func (d sqliteDialect) ColumnExists(table, column string) string {
	return fmt.Sprintf("SELECT COUNT(*) FROM pragma_table_info('%s') WHERE name = '%s'", table, column)
}

func (d sqliteDialect) SearchQuery(raw string) string {
	tokens := sanitizeSearchTokens(raw)
	if len(tokens) == 0 {
		return ""
	}
	return strings.Join(tokens, "* ") + "*"
}

// ---------------------------------------------------------------------------
// Postgres dialect
// ---------------------------------------------------------------------------

type postgresDialect struct{}

func (d postgresDialect) Placeholder(n int) string {
	return fmt.Sprintf("$%d", n)
}

func (d postgresDialect) RewritePlaceholders(sql string) string {
	// Replace ? with $1, $2, ... in order.
	// As documented: no ? appear inside string literals in our queries.
	var b strings.Builder
	n := 1
	for _, r := range sql {
		if r == '?' {
			fmt.Fprintf(&b, "$%d", n)
			n++
		} else {
			b.WriteRune(r)
		}
	}
	return b.String()
}

func (d postgresDialect) Now() string { return "NOW()" }

func (d postgresDialect) ColumnExists(table, column string) string {
	return fmt.Sprintf("SELECT COUNT(*) FROM information_schema.columns WHERE table_name = '%s' AND column_name = '%s'", table, column)
}

func (d postgresDialect) SearchQuery(raw string) string {
	tokens := sanitizeSearchTokens(raw)
	if len(tokens) == 0 {
		return ""
	}
	return strings.Join(tokens, ":* & ") + ":*"
}

// sanitizeSearchTokens splits raw on whitespace and strips every character
// from each token that isn't a letter or digit, dropping tokens that end up
// empty. FTS5 (SQLite) and tsquery (Postgres) both give special meaning to
// characters a user can easily type into a search box — quotes, parens,
// colons (column filters), the NEAR operator, etc. — and an unbalanced or
// stray one turns the whole generated query into a syntax error instead of
// just "no results". Reducing every token to plain alphanumerics before it's
// assembled into a query string keeps the request always well-formed; it's
// not a query-syntax parser, just enough to keep the *-suffixed prefix
// queries below from ever containing dialect-special characters.
func sanitizeSearchTokens(raw string) []string {
	fields := strings.Fields(raw)
	tokens := make([]string, 0, len(fields))
	for _, f := range fields {
		t := strings.Map(func(r rune) rune {
			if unicode.IsLetter(r) || unicode.IsDigit(r) {
				return r
			}
			return -1
		}, f)
		if t != "" {
			tokens = append(tokens, t)
		}
	}
	return tokens
}

// ---------------------------------------------------------------------------
// Factory
// ---------------------------------------------------------------------------

// SQLiteDialect returns a ready-to-use SQLite dialect. Convenience function
// for callers that always use SQLite (tests, tune CLI).
func SQLiteDialect() Dialect { return sqliteDialect{} }

// NewDialect returns the dialect for the given driver name.
func NewDialect(driver string) (Dialect, error) {
	switch driver {
	case "sqlite":
		return sqliteDialect{}, nil
	case "postgres":
		return postgresDialect{}, nil
	default:
		return nil, fmt.Errorf("DB_DRIVER=%s not supported; valid drivers: sqlite, postgres", driver)
	}
}
