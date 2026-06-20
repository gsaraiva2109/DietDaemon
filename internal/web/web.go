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
	"path"
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
			// A missing asset-like path (favicon.ico, vite.svg, …) must 404, not
			// return the HTML shell — otherwise browsers get index.html where they
			// expected an image and fall back to a stale/default favicon.
			if path.Ext(p) != "" {
				http.NotFound(w, r)
				return
			}
			// Otherwise it's a client-side route — hand the SPA its entry point.
			r = r.Clone(r.Context())
			r.URL.Path = "/"
		}
		files.ServeHTTP(w, r)
	}), nil
}
