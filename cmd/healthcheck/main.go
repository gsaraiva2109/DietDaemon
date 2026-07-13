// Command healthcheck is a minimal liveness probe for distroless containers.
// It checks whether /data/healthy was touched recently by the main process.
// Exits 0 if fresh (< 15s old), 1 if stale, missing, or unparseable.
package main

import (
	"os"
	"time"
)

func main() {
	data, err := os.ReadFile("/data/healthy")
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
