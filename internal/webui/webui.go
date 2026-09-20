// Package webui serves the built single-page app from the binary. The frontend build writes into
// dist/ (see web/vite.config.ts); `go build` embeds whatever is there. A checkout without a
// frontend build still compiles and serves a page that says so, instead of a 404 that looks like
// a routing bug.
package webui

import (
	"embed"
	"io/fs"
	"net/http"
	"strings"
)

//go:embed all:dist
var dist embed.FS

// Handler serves the app. Any path that is not a built asset gets index.html, so the client-side
// router owns the URL space — every screen state is a URL, and a reload lands where you were.
func Handler() http.Handler {
	sub, _ := fs.Sub(dist, "dist")
	files := http.FS(sub)
	index, err := fs.ReadFile(sub, "index.html")
	built := err == nil
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !built {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.WriteHeader(http.StatusServiceUnavailable)
			_, _ = w.Write([]byte(notBuilt))
			return
		}
		p := strings.TrimPrefix(r.URL.Path, "/")
		if p != "" {
			if f, err := sub.Open(p); err == nil {
				_ = f.Close()
				if strings.HasPrefix(p, "assets/") {
					w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
				}
				http.StripPrefix("/", http.FileServer(files)).ServeHTTP(w, r)
				return
			}
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("Cache-Control", "no-store")
		_, _ = w.Write(index)
	})
}

// Built reports whether a frontend build is embedded.
func Built() bool {
	_, err := fs.ReadFile(dist, "dist/index.html")
	return err == nil
}

const notBuilt = `<!doctype html><meta charset="utf-8"><title>TightShip</title>
<body style="font:16px system-ui;padding:3rem;max-width:40rem">
<h1>TightShip</h1><p>The API is up, but this binary was built without the web app.
Run <code>make web</code> then rebuild, or install a release binary.</p>`
