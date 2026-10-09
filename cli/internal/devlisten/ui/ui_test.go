package ui

import (
	"io/fs"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

const prefix = "/_local/ui/"

func serve(method, target string) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	Handler(prefix).ServeHTTP(w, httptest.NewRequest(method, target, nil))
	return w
}

func TestHandlerServesThePageFiles(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		target      string
		status      int
		contentType string
		contains    string
	}{
		{target: "/_local/ui/", status: 200, contentType: "text/html; charset=utf-8", contains: `<script src="app.js"`},
		{target: "/_local/ui/?event=Order+Completed&tab=requests", status: 200, contentType: "text/html; charset=utf-8", contains: "<title>"},
		{target: "/_local/ui/index.html", status: 200, contentType: "text/html; charset=utf-8", contains: "<title>"},
		{target: "/_local/ui/app.js", status: 200, contentType: "text/javascript; charset=utf-8", contains: "use strict"},
		{target: "/_local/ui/app.css", status: 200, contentType: "text/css; charset=utf-8", contains: "prefers-color-scheme: dark"},
		{target: "/_local/ui/icon.svg", status: 200, contentType: "image/svg+xml", contains: "<svg"},
		{target: "/_local/ui/nope.js", status: 404, contentType: "text/plain; charset=utf-8", contains: "not found"},
		{target: "/_local/ui/app.js/", status: 404, contentType: "text/plain; charset=utf-8", contains: "not found"},
		{target: "/_local/ui/ui.go", status: 404, contentType: "text/plain; charset=utf-8", contains: "not found"},
		{target: "/_local/ui/../v1/info", status: 404, contentType: "text/plain; charset=utf-8", contains: "not found"},
	} {
		t.Run(tc.target, func(t *testing.T) {
			t.Parallel()

			w := serve(http.MethodGet, tc.target)

			require.Equal(t, tc.status, w.Code)
			require.Equal(t, tc.contentType, w.Header().Get("Content-Type"))
			require.Contains(t, w.Body.String(), tc.contains)
			require.Equal(t, policy, w.Header().Get("Content-Security-Policy"))
			require.Equal(t, "no-referrer", w.Header().Get("Referrer-Policy"))
			require.Empty(t, w.Header().Get("Last-Modified"))
		})
	}
}

func TestHandlerPolicyAllowsOnlyThisOrigin(t *testing.T) {
	t.Parallel()

	w := serve(http.MethodGet, "/_local/ui/")

	require.Equal(t, "default-src 'self'; base-uri 'none'; form-action 'none'; frame-ancestors 'none'",
		w.Header().Get("Content-Security-Policy"))
}

func TestHandlerAnswersHeadWithoutABody(t *testing.T) {
	t.Parallel()

	w := serve(http.MethodHead, "/_local/ui/app.js")

	require.Equal(t, http.StatusOK, w.Code)
	require.Equal(t, "text/javascript; charset=utf-8", w.Header().Get("Content-Type"))
	require.Empty(t, w.Body.String())
}

func pageFiles(t *testing.T) map[string]string {
	t.Helper()
	out := map[string]string{}
	require.NoError(t, fs.WalkDir(files, ".", func(path string, d fs.DirEntry, err error) error {
		require.NoError(t, err)
		if d.IsDir() {
			return nil
		}
		data, err := fs.ReadFile(files, path)
		require.NoError(t, err)
		out[path] = string(data)
		return nil
	}))
	require.Len(t, out, 4)
	return out
}

// The page stands alone: a reviewer in a browser has no shell, so it
// explains in its own words and never prints a command.
func TestPageNamesNoCommand(t *testing.T) {
	t.Parallel()
	for name, text := range pageFiles(t) {
		require.NotContains(t, strings.ToLower(text), "rudder-cli", name)
		require.NotContains(t, text, "curl ", name)
		require.NotContains(t, text, "--write-key", name)
	}
}

// The page works offline and loads nothing from another origin, which the
// policy would block anyway.
func TestPageLoadsNothingFromElsewhere(t *testing.T) {
	t.Parallel()
	for name, text := range pageFiles(t) {
		for _, ref := range []string{"http:", "https:", "//cdn", "@import", "url("} {
			if name == "icon.svg" && ref == "http:" {
				continue
			}
			require.NotContains(t, text, ref, name)
		}
	}
}

// Captured values are untrusted. The page writes them as text only, so an
// event name such as <img src=x onerror=alert(1)> stays text.
func TestPageWritesCapturedValuesAsText(t *testing.T) {
	t.Parallel()
	files := pageFiles(t)
	for _, sink := range []string{"innerHTML", "outerHTML", "insertAdjacentHTML", "document.write", "eval(", "new Function", "setTimeout(\"", "javascript:"} {
		require.NotContains(t, files["app.js"], sink)
	}
	for name, text := range files {
		require.NotContains(t, text, " style=\"", name)
		// The policy blocks inline handlers and inline style attributes.
		require.NotRegexp(t, `\son[a-z]+=`, text, name)
	}
}
