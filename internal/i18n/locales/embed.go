// Package locales embeds the JSON translation bundles so they are compiled into
// the binary. This keeps the Docker image self-contained.
package locales

import "embed"

// FS holds all .json locale files from this directory.
//
//go:embed *.json
var FS embed.FS
