// Package web serves the optional dashboard single-page app. The built Vite
// assets (web/dist at the repo root) are copied into ./dist by the build
// (see Makefile) and embedded here, so the whole dashboard ships inside the
// single Go binary — same origin as the API, no CORS, no second container.
//
// A committed placeholder dist/index.html keeps this package compiling even
// when the frontend hasn't been built (headless / CI). The real build
// overwrites it.
package web

import (
	"embed"
	"io/fs"
	"net/http"
	"strings"
)

//go:embed all:dist
var distFS embed.FS

// Handler returns an http.Handler that serves the SPA: real files are served
// as-is; any unknown path falls back to index.html so client-side routes
// (e.g. /history/123) load the app instead of 404ing. Mount it on "/" after
// the API routes — ServeMux's most-specific-match wins, so /api/v1/* still
// reaches the API.
func Handler() (http.Handler, error) {
	sub, err := fs.Sub(distFS, "dist")
	if err != nil {
		return nil, err
	}
	files := http.FileServer(http.FS(sub))

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		p := strings.TrimPrefix(r.URL.Path, "/")
		if p == "" {
			p = "index.html"
		}
		if _, statErr := fs.Stat(sub, p); statErr != nil {
			// Not a real asset — hand the SPA its entry point.
			r = r.Clone(r.Context())
			r.URL.Path = "/"
			p = "index.html"
		}

		// Cache policy: hashed assets (JS/CSS/fonts) are immutable —
		// Vite changes the filename on every build, so cache forever.
		// index.html must always revalidate so the browser sees new
		// asset URLs after a deploy.
		if p == "index.html" {
			w.Header().Set("Cache-Control", "no-cache, must-revalidate")
		} else {
			w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
		}

		files.ServeHTTP(w, r)
	}), nil
}
