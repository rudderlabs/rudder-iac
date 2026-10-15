// Package ui holds the review page that dev listen serves at /_dev/ui/. The
// page is plain HTML, CSS and JavaScript with no build step. It reads the
// capture through the query API only.
package ui

import (
	"bytes"
	"embed"
	"net/http"
	"strings"
	"time"
)

//go:embed index.html app.js app.css icon.svg
var files embed.FS

// policy lets the page load its own files and nothing else, and stops other
// sites from framing it.
const policy = "default-src 'self'; base-uri 'none'; form-action 'none'; frame-ancestors 'none'"

var contentTypes = map[string]string{
	"index.html": "text/html; charset=utf-8",
	"app.js":     "text/javascript; charset=utf-8",
	"app.css":    "text/css; charset=utf-8",
	"icon.svg":   "image/svg+xml",
}

// Handler serves the page files under prefix. The caller runs the Host and
// browser-origin guards and the method check first.
func Handler(prefix string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Security-Policy", policy)
		w.Header().Set("Referrer-Policy", "no-referrer")

		name := strings.TrimPrefix(r.URL.Path, prefix)
		if name == "" {
			name = "index.html"
		}
		contentType, ok := contentTypes[name]
		if !ok {
			w.Header().Set("Content-Type", "text/plain; charset=utf-8")
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte("not found\n"))
			return
		}
		data, err := files.ReadFile(name)
		if err != nil {
			http.Error(w, "the page files are incomplete", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", contentType)
		// A zero time sends no Last-Modified, so a reload always reads the
		// files of the running binary.
		http.ServeContent(w, r, name, time.Time{}, bytes.NewReader(data))
	})
}
