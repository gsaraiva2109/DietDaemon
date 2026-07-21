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
// as-is and declared client routes fall back to index.html. Unknown HTML
// navigations get a branded 404; all other missing paths remain normal 404s.
// Mount it on "/" after the API routes — ServeMux's most-specific-match wins,
// so /api/v1/* still reaches the API.
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
			if IsSPARoute(r.URL.Path) {
				// Declared client route — hand the SPA its entry point.
				r = r.Clone(r.Context())
				r.URL.Path = "/"
				p = "index.html"
			} else if path.Ext(p) != "" {
				// A missing asset must never return an HTML navigation page.
				http.NotFound(w, r)
				return
			} else if isHTMLNavigation(r) {
				writeNotFoundPage(w)
				return
			} else {
				// Missing assets and non-HTML requests stay regular 404s.
				http.NotFound(w, r)
				return
			}
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

func isHTMLNavigation(r *http.Request) bool {
	return (r.Method == http.MethodGet || r.Method == http.MethodHead) &&
		strings.Contains(r.Header.Get("Accept"), "text/html")
}

func writeNotFoundPage(w http.ResponseWriter) {
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusNotFound)
	_, _ = w.Write([]byte(notFoundPage))
}

const notFoundPage = `<!doctype html>
<html lang="en">
<head><meta charset="utf-8"><meta name="viewport" content="width=device-width, initial-scale=1"><title>Page not found · DietDaemon</title></head>
<body style="margin:0;background:#f7f8f5;color:#182016;font:16px system-ui,-apple-system,sans-serif"><main style="display:grid;min-height:100vh;place-items:center;padding:24px"><section style="max-width:420px;text-align:center"><div style="margin:auto auto 24px;display:grid;width:56px;height:56px;place-items:center;border-radius:16px;background:#e2f3dc;color:#287348;font-size:28px">✦</div><p style="margin:0;color:#5d6959;font-size:12px;font-weight:700;letter-spacing:.16em;text-transform:uppercase">DietDaemon</p><h1 style="margin:12px 0 8px;font-size:32px;letter-spacing:-.04em">Page not found</h1><p style="margin:0;color:#5d6959;line-height:1.5">The page you requested does not exist or has moved.</p><a href="/" style="display:inline-block;margin-top:24px;border-radius:999px;background:#287348;color:#fff;padding:12px 20px;font-weight:700;text-decoration:none">Go to dashboard</a></section></main></body>
</html>`
