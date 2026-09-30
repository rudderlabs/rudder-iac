// Package ui holds the read-only review page served at /_dev/ui/. It is
// plain HTML, CSS and JavaScript with no build step; the page reads the
// same query API as the CLI.
package ui

import (
	"embed"
	"io/fs"
	"path"
)

//go:embed assets
var assets embed.FS

// contentTypes are the only files the page serves.
var contentTypes = map[string]string{
	"index.html": "text/html; charset=utf-8",
	"app.css":    "text/css; charset=utf-8",
	"app.js":     "text/javascript; charset=utf-8",
}

// File returns the bytes and content type of name, a path below /_dev/ui/.
// "" is the page itself.
func File(name string) ([]byte, string, bool) {
	if name == "" {
		name = "index.html"
	}
	contentType, ok := contentTypes[name]
	if !ok {
		return nil, "", false
	}
	b, err := fs.ReadFile(assets, path.Join("assets", name))
	return b, contentType, err == nil
}
