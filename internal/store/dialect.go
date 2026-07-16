// Package store implements ports.Store with SQLite via a pure-Go driver.
package store

import (
	"fmt"
	"strings"

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

	// DateTrunc returns a SQL expression that truncates the named TEXT column
	// (storing timestamps as "YYYY-MM-DDTHH:MM:SSZ" or "YYYY-MM-DD HH:MM:SS")
	// down to its "YYYY-MM-DD" date portion, as text comparable to, and
	// groupable/orderable with, a "YYYY-MM-DD" parameter. SQLite's date()
	// accepts a text argument directly; Postgres's date() has no text
	// overload, so we take the same first-10-characters substring the
	// Postgres migration already uses for the generated logged_date columns
	// (see migrations/postgres/001_init.sql, water_logs.logged_date).
	DateTrunc(col string) string
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
	tokens := strings.Fields(raw)
	return strings.Join(tokens, "* ") + "*"
}

func (d sqliteDialect) DateTrunc(col string) string {
	return fmt.Sprintf("date(%s)", col)
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
	tokens := strings.Fields(raw)
	return strings.Join(tokens, ":* & ") + ":*"
}

func (d postgresDialect) DateTrunc(col string) string {
	// date(text) has no Postgres overload; substring matches the format the
	// migration's generated logged_date columns already produce.
	return fmt.Sprintf("substring(%s from 1 for 10)", col)
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
		return nil, &ErrUnsupportedDriver{Driver: driver}
	}
}

// ErrUnsupportedDriver is returned when DB_DRIVER is not "sqlite" or "postgres".
type ErrUnsupportedDriver struct{ Driver string }

func (e *ErrUnsupportedDriver) Error() string {
	return "DB_DRIVER=" + e.Driver + " not supported; valid drivers: sqlite, postgres"
}
