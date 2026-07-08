// Package store implements ports.Store. Supports SQLite (modernc.org/sqlite, pure
// Go, CGO-free) and Postgres (lib/pq) via a Dialect abstraction.
package store

import (
	"database/sql"
	"fmt"
	"strings"

	"github.com/jmoiron/sqlx"
	_ "modernc.org/sqlite"

	"github.com/gsaraiva2109/dietdaemon/core/ports"
	"github.com/gsaraiva2109/dietdaemon/internal/assistant"
	"github.com/gsaraiva2109/dietdaemon/internal/backup"
	"github.com/gsaraiva2109/dietdaemon/internal/scheduler"
	"github.com/gsaraiva2109/dietdaemon/migrations"
)

// Store implements ports.Store backed by SQLite or Postgres.
type Store struct {
	db      *sqlx.DB
	dialect Dialect
	driver  string // "sqlite" or "postgres"
}

// Compile-time guarantees that Store satisfies every interface boundary it must.
var (
	_ ports.Store                 = (*Store)(nil)
	_ scheduler.Store             = (*Store)(nil)
	_ scheduler.NudgeStore        = (*Store)(nil)
	_ scheduler.RuleConfigStore   = (*Store)(nil)
	_ scheduler.DigestStore       = (*Store)(nil)
	_ scheduler.ChatRouteStore    = (*Store)(nil)
	_ scheduler.SentNudgeStore    = (*Store)(nil)
	_ scheduler.WeeklyBudgetStore = (*Store)(nil)
	_ backup.Store                = (*Store)(nil)
	_ assistant.Store             = (*Store)(nil)
)

// New opens a database, applies driver-specific setup, runs migrations, and
// returns a ready Store.
//
// driver is "sqlite" or "postgres". dsn is the file path for SQLite or a
// connection URL for Postgres (e.g. "postgres://user:pass@host/db?sslmode=disable").
func New(driver, dsn string, dialect Dialect) (*Store, error) {
	db, err := sql.Open(driver, dsn)
	if err != nil {
		return nil, fmt.Errorf("store: open db: %w", err)
	}

	switch driver {
	case "sqlite":
		// SQLite is single-writer; a pool of 1 guarantees every PRAGMA and all
		// subsequent queries share the same connection, avoiding SQLITE_CANTOPEN
		// when a second connection races the EXCLUSIVE lock below.
		db.SetMaxOpenConns(1)

		// EXCLUSIVE locking before WAL avoids shared-memory (-shm) entirely.
		// SQLite keeps the wal-index in heap memory instead of mmap'ing a file.
		// Required for Docker Swarm / some overlay filesystems where the VFS
		// shared-memory primitives (xShmMap, xShmLock, etc.) don't work.
		if _, err := db.Exec("PRAGMA locking_mode = EXCLUSIVE"); err != nil {
			_ = db.Close()
			return nil, fmt.Errorf("store: exclusive lock: %w", err)
		}
		if _, err := db.Exec("PRAGMA journal_mode = WAL"); err != nil {
			_ = db.Close()
			return nil, fmt.Errorf("store: enable WAL: %w", err)
		}
		if _, err := db.Exec("PRAGMA foreign_keys = ON"); err != nil {
			_ = db.Close()
			return nil, fmt.Errorf("store: enable foreign keys: %w", err)
		}
	case "postgres":
		// Postgres manages its own connection pool; 25 is a sensible default
		// for modest deployments. The operator can tune via PG* env vars or
		// connection URL parameters.
		db.SetMaxOpenConns(25)
	}

	s := &Store{db: sqlx.NewDb(db, driver), dialect: dialect, driver: driver}
	if err := s.runMigrations(); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("store: migrate: %w", err)
	}

	return s, nil
}

// rewrite applies dialect-specific placeholder conversion to a SQL query.
// For SQLite this is a no-op; for Postgres it replaces ? with $1, $2, ...
func (s *Store) rewrite(sql string) string {
	return s.dialect.RewritePlaceholders(sql)
}

func (s *Store) runMigrations() error {
	if _, err := s.db.Exec(s.rewrite(`CREATE TABLE IF NOT EXISTS schema_migrations (name TEXT PRIMARY KEY, applied_at TEXT NOT NULL DEFAULT (` + s.dialect.Now() + `))`)); err != nil {
		return fmt.Errorf("init migration tracking: %w", err)
	}

	entries, err := migrations.FS(s.driver).ReadDir(s.driver)
	if err != nil {
		return fmt.Errorf("read migrations dir: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".sql") {
			continue
		}

		var already int
		if err := s.db.Get(&already, s.rewrite(`SELECT COUNT(*) FROM schema_migrations WHERE name = ?`), entry.Name()); err != nil {
			return fmt.Errorf("check migration %s: %w", entry.Name(), err)
		}
		if already > 0 {
			continue
		}

		content, err := migrations.FS(s.driver).ReadFile(s.driver + "/" + entry.Name())
		if err != nil {
			return fmt.Errorf("read %s: %w", entry.Name(), err)
		}
		if _, err := s.db.Exec(s.rewrite(string(content))); err != nil {
			// Idempotency: databases that predate migration tracking may
			// already have tables/columns/indexes from manual or older
			// migration paths. Treat "already exists" errors as success
			// so the migration is tracked and skipped on next start.
			if isBenignMigrationErr(err) {
				if _, recErr := s.db.Exec(s.rewrite(`INSERT INTO schema_migrations (name) VALUES (?)`), entry.Name()); recErr != nil {
					return fmt.Errorf("record migration %s after benign error: %w", entry.Name(), recErr)
				}
				continue
			}
			return fmt.Errorf("exec %s: %w", entry.Name(), err)
		}
		if _, err := s.db.Exec(s.rewrite(`INSERT INTO schema_migrations (name) VALUES (?)`), entry.Name()); err != nil {
			return fmt.Errorf("record migration %s: %w", entry.Name(), err)
		}
	}

	return nil
}

// isBenignMigrationErr returns true when err indicates the DDL/DML operation
// was already applied — a duplicate column, table, index, or constraint.
// This lets the migration runner treat pre-existing schema as success instead
// of aborting startup.
func isBenignMigrationErr(err error) bool {
	msg := err.Error()
	// SQLite error patterns for "already exists":
	//   - "duplicate column name: X"
	//   - "table X already exists"
	//   - "index X already exists"
	//   - "trigger X already exists"
	//   - "UNIQUE constraint failed: schema_migrations.name" (harmless)
	return strings.Contains(msg, "duplicate column name") ||
		strings.Contains(msg, "already exists") ||
		strings.Contains(msg, "UNIQUE constraint failed")
}

// DB returns the underlying *sql.DB so that callers (e.g. the embedding index)
// can operate on the same database connection without opening a second one.
func (s *Store) DB() *sql.DB { return s.db.DB }

// Close closes the database connection.
func (s *Store) Close() error {
	return s.db.Close()
}
