// Command healthcheck is a minimal liveness probe for distroless containers.
// It checks whether /data/healthy was touched recently by the main process.
// Exits 0 if fresh (< 15s old), 1 if stale, missing, or unparseable.
package main

import (
	"os"
	"time"
)

func main() {
	path := os.Getenv("HEALTH_CHECK_PATH")
	if path == "" {
		path = "/data/healthy"
	}
	data, err := os.ReadFile(path) // #nosec G304 G703 -- path provided by operator via env var, intentional file read
	if err != nil {
		os.Exit(1)
	}
	t, err := time.Parse(time.RFC3339, string(data))
	if err != nil {
		os.Exit(1)
	}
	if time.Since(t) > 15*time.Second {
		os.Exit(1)
	}
}
