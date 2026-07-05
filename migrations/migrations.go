// Package migrations embeds the SQL migration files so they are compiled into
// the binary. This keeps the Docker image self-contained (no extra file copies).
//
// Each driver has its own subdirectory with dialect-specific SQL:
//   - sqlite/  — SQLite migrations (original, unchanged)
//   - postgres/ — Postgres-ported equivalents
//
// Call FS(driver) to get the correct embed.FS for the active driver.
package migrations

import "embed"

//go:embed sqlite/*.sql
var SQLiteFS embed.FS

//go:embed postgres/*.sql
var PostgresFS embed.FS

// FS returns the embed.FS for the given driver name.
func FS(driver string) embed.FS {
	switch driver {
	case "postgres":
		return PostgresFS
	default:
		return SQLiteFS
	}
}
