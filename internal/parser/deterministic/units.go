package deterministic

import textnormalize "github.com/gsaraiva2109/dietdaemon/internal/normalize"

// normalize lowercases, trims, and strips accents so unit lookups and food
// phrases are matched consistently (the store normalizes the same way).
func normalize(s string) string {
	return textnormalize.Normalize(s)
}
