package store

import (
	"slices"
	"strings"
	"testing"
)

func TestSQLiteRewritePlaceholders(t *testing.T) {
	d := SQLiteDialect()
	tests := []struct{ name, in, want string }{
		{"no placeholders", "SELECT 1", "SELECT 1"},
		{"single placeholder", "SELECT * FROM t WHERE x = ?", "SELECT * FROM t WHERE x = ?"},
		{"multiple placeholders", "SELECT * FROM t WHERE x = ? AND y = ?", "SELECT * FROM t WHERE x = ? AND y = ?"},
		{"insert with placeholders", "INSERT INTO t (a, b) VALUES (?, ?)", "INSERT INTO t (a, b) VALUES (?, ?)"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := d.RewritePlaceholders(tt.in)
			if got != tt.want {
				t.Errorf("RewritePlaceholders(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestPostgresRewritePlaceholders(t *testing.T) {
	d, err := NewDialect("postgres")
	if err != nil {
		t.Fatalf("NewDialect(postgres): %v", err)
	}
	tests := []struct{ name, in, want string }{
		{"no placeholders", "SELECT 1", "SELECT 1"},
		{"single placeholder", "SELECT * FROM t WHERE x = ?", "SELECT * FROM t WHERE x = $1"},
		{"multiple placeholders", "SELECT * FROM t WHERE x = ? AND y = ?", "SELECT * FROM t WHERE x = $1 AND y = $2"},
		{"insert with placeholders", "INSERT INTO t (a, b) VALUES (?, ?)", "INSERT INTO t (a, b) VALUES ($1, $2)"},
		{"three placeholders", "?, ?, ?", "$1, $2, $3"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := d.RewritePlaceholders(tt.in)
			if got != tt.want {
				t.Errorf("RewritePlaceholders(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestPlaceholder(t *testing.T) {
	sqlite := SQLiteDialect()
	for i := 1; i <= 5; i++ {
		if got := sqlite.Placeholder(i); got != "?" {
			t.Errorf("SQLite Placeholder(%d) = %q, want ?", i, got)
		}
	}

	pg, err := NewDialect("postgres")
	if err != nil {
		t.Fatalf("NewDialect(postgres): %v", err)
	}
	for i, want := range []string{"$1", "$2", "$3", "$4", "$5"} {
		if got := pg.Placeholder(i + 1); got != want {
			t.Errorf("Postgres Placeholder(%d) = %q, want %q", i+1, got, want)
		}
	}
}

func TestNow(t *testing.T) {
	sqlite := SQLiteDialect()
	if got := sqlite.Now(); got != "datetime('now')" {
		t.Errorf("SQLite Now() = %q, want datetime('now')", got)
	}

	pg, err := NewDialect("postgres")
	if err != nil {
		t.Fatalf("NewDialect(postgres): %v", err)
	}
	if got := pg.Now(); got != "NOW()" {
		t.Errorf("Postgres Now() = %q, want NOW()", got)
	}
}

func TestColumnExists(t *testing.T) {
	sqlite := SQLiteDialect()
	sqliteQ := sqlite.ColumnExists("users", "email")
	if !strings.Contains(sqliteQ, "pragma_table_info") {
		t.Errorf("SQLite ColumnExists should use pragma_table_info, got: %s", sqliteQ)
	}
	if !strings.Contains(sqliteQ, "users") || !strings.Contains(sqliteQ, "email") {
		t.Errorf("SQLite ColumnExists should reference table and column, got: %s", sqliteQ)
	}

	pg, err := NewDialect("postgres")
	if err != nil {
		t.Fatalf("NewDialect(postgres): %v", err)
	}
	pgQ := pg.ColumnExists("users", "email")
	if !strings.Contains(pgQ, "information_schema.columns") {
		t.Errorf("Postgres ColumnExists should use information_schema, got: %s", pgQ)
	}
	if !strings.Contains(pgQ, "users") || !strings.Contains(pgQ, "email") {
		t.Errorf("Postgres ColumnExists should reference table and column, got: %s", pgQ)
	}
}

// TestSearchQueryCleanInput pins the pre-existing prefix-query shape for
// ordinary alphanumeric input, for both dialects, so the sanitization added
// below can't quietly change the common-case output.
func TestSearchQueryCleanInput(t *testing.T) {
	sqlite := SQLiteDialect()
	if got, want := sqlite.SearchQuery("chicken breast"), "chicken* breast*"; got != want {
		t.Errorf("SQLite SearchQuery(clean) = %q, want %q", got, want)
	}

	pg, err := NewDialect("postgres")
	if err != nil {
		t.Fatalf("NewDialect(postgres): %v", err)
	}
	if got, want := pg.SearchQuery("chicken breast"), "chicken:* & breast:*"; got != want {
		t.Errorf("Postgres SearchQuery(clean) = %q, want %q", got, want)
	}
}

// TestSearchQueryMalformedInput pins the #276 item 3 fix: unbalanced quotes,
// NEAR/column-filter syntax, parens, and other FTS5/tsquery-special
// characters must not survive into the generated query string, since any of
// them can turn the whole search request into a query-syntax error instead
// of just "no results".
func TestSearchQueryMalformedInput(t *testing.T) {
	dangerous := []string{`"`, "*", "(", ")", ":", ";", "'", "-", "+", "^", "~"}

	malformed := []string{
		`"unbalanced quote`,
		`col:value NEAR(term)`,
		`foo:bar OR 1=1); DROP TABLE x;--`,
		`(a & b) | c`,
		`???`,
	}

	sqlite := SQLiteDialect()
	pg, err := NewDialect("postgres")
	if err != nil {
		t.Fatalf("NewDialect(postgres): %v", err)
	}

	for _, raw := range malformed {
		// The generated "*" prefix suffix is expected for both dialects, and
		// postgres additionally expects a ":*" suffix; only flag characters
		// that leaked in from the raw input itself.
		assertNoDangerousChars(t, "SQLite", raw, sqlite.SearchQuery(raw), dangerous, "*")
		assertNoDangerousChars(t, "Postgres", raw, pg.SearchQuery(raw), dangerous, ":", "*")
	}
}

// assertNoDangerousChars fails t if got contains any of dangerous, except
// characters listed in allowed (expected dialect syntax like "*" or ":*").
func assertNoDangerousChars(t *testing.T, dialect, raw, got string, dangerous []string, allowed ...string) {
	t.Helper()
	for _, d := range dangerous {
		if slices.Contains(allowed, d) {
			continue
		}
		if strings.Contains(got, d) {
			t.Errorf("%s SearchQuery(%q) = %q, contains dangerous char %q", dialect, raw, got, d)
		}
	}
}

// TestSearchQueryAllPunctuationYieldsEmpty ensures a query that sanitizes
// down to nothing (e.g. all punctuation) produces an empty query string
// instead of a bare "*"/"​:*", which is itself invalid FTS5/tsquery syntax.
func TestSearchQueryAllPunctuationYieldsEmpty(t *testing.T) {
	sqlite := SQLiteDialect()
	if got := sqlite.SearchQuery("???"); got != "" {
		t.Errorf("SQLite SearchQuery(all-punctuation) = %q, want empty", got)
	}

	pg, err := NewDialect("postgres")
	if err != nil {
		t.Fatalf("NewDialect(postgres): %v", err)
	}
	if got := pg.SearchQuery("???"); got != "" {
		t.Errorf("Postgres SearchQuery(all-punctuation) = %q, want empty", got)
	}
}

func TestNewDialectInvalid(t *testing.T) {
	d, err := NewDialect("mysql")
	if err == nil {
		t.Errorf("NewDialect(mysql) should return error, got dialect %v", d)
	}
	if !strings.Contains(err.Error(), "mysql") {
		t.Errorf("error should mention driver name, got: %v", err)
	}
}
