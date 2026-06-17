// Package migrations embeds the SQL migration files so they are compiled into
// the binary. This keeps the Docker image self-contained (no extra file copies).
package migrations

import "embed"

// FS holds all .sql migration files from this directory. Consumers read
// entries in alphabetical order to apply schema changes.
//
//go:embed *.sql
var FS embed.FS
