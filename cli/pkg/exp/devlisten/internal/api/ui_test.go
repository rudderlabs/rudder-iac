package api

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/rudderlabs/rudder-iac/cli/pkg/exp/devlisten/internal/store"
	"github.com/rudderlabs/rudder-iac/cli/pkg/exp/devlisten/internal/ui"
)

func getRaw(t *testing.T, url string) (*http.Response, string) {
	t.Helper()
	resp, err := http.Get(url) //nolint:gosec,noctx // test URL
	require.NoError(t, err)
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	return resp, string(body)
}

func TestUIPageIsServedWithTheGuards(t *testing.T) {
	t.Parallel()
	srv, _ := newTestServer(t)

	resp, body := getRaw(t, srv.URL+"/_dev/ui/")

	require.Equal(t, http.StatusOK, resp.StatusCode)
	require.Equal(t, "text/html; charset=utf-8", resp.Header.Get("Content-Type"))
	require.Equal(t, "default-src 'self'; frame-ancestors 'none'", resp.Header.Get("Content-Security-Policy"))
	require.Equal(t, "nosniff", resp.Header.Get("X-Content-Type-Options"))
	require.Empty(t, resp.Header.Get("Access-Control-Allow-Origin"))
	require.Contains(t, body, `<script src="app.js"`)

	head, err := http.Head(srv.URL + "/_dev/ui/") //nolint:noctx // test URL
	require.NoError(t, err)
	head.Body.Close()
	require.Equal(t, http.StatusOK, head.StatusCode)
}

func TestUIRefusesAForeignHost(t *testing.T) {
	t.Parallel()
	st := store.New(testIdentity.ServerID)
	srv := httptest.NewServer(New(st, testIdentity, Config{CheckHost: true}))
	t.Cleanup(srv.Close)

	req, err := http.NewRequest(http.MethodGet, srv.URL+"/_dev/ui/", nil)
	require.NoError(t, err)
	req.Host = "rebind.example:4321"
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	resp.Body.Close()

	require.Equal(t, http.StatusForbidden, resp.StatusCode)
}

func TestUIServesOnlyItsOwnFiles(t *testing.T) {
	t.Parallel()
	srv, _ := newTestServer(t)

	for name, want := range map[string]string{
		"app.js":        "text/javascript; charset=utf-8",
		"app.css":       "text/css; charset=utf-8",
		"../v1/info":    "",
		"ui.go":         "",
		"assets/app.js": "",
	} {
		resp, _ := getRaw(t, srv.URL+"/_dev/ui/"+name)
		if want == "" {
			require.Equal(t, http.StatusNotFound, resp.StatusCode, name)
			continue
		}
		require.Equal(t, want, resp.Header.Get("Content-Type"), name)
	}
}

// Captured values come from any page that can post to the listener, so the
// page renders them as text only.
func TestUIScriptNeverParsesHTML(t *testing.T) {
	t.Parallel()
	srv, _ := newTestServer(t)

	_, script := getRaw(t, srv.URL+"/_dev/ui/app.js")

	for _, sink := range []string{"innerHTML", "outerHTML", "insertAdjacentHTML", "document.write", "eval("} {
		require.NotContains(t, script, sink)
	}
	require.Contains(t, script, "textContent")
}

// A human clicks the ui link in a PR or chat: a cross-site top-level
// navigation cannot read the response, so it is allowed; a fetch is not.
func TestUILinkFromAnotherSiteOpens(t *testing.T) {
	t.Parallel()
	srv, _ := newTestServer(t)

	req, err := http.NewRequest(http.MethodGet, srv.URL+"/_dev/ui/", nil)
	require.NoError(t, err)
	req.Header.Set("Sec-Fetch-Site", "cross-site")
	req.Header.Set("Sec-Fetch-Mode", "navigate")
	req.Header.Set("Sec-Fetch-Dest", "document")
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	resp.Body.Close()
	nav := resp.StatusCode
	fetch, _ := get(t, srv.URL+"/_dev/v1/events", "Sec-Fetch-Site", "cross-site", "Sec-Fetch-Mode", "cors",
		"Sec-Fetch-Dest", "empty")

	require.Equal(t, http.StatusOK, nav)
	require.Equal(t, http.StatusForbidden, fetch)
}

// Everything the command line can ask of /events and /requests has a
// control on the page.
func TestEveryQueryParameterHasAControl(t *testing.T) {
	t.Parallel()
	page, _, _ := ui.File("")

	for _, route := range []string{"events", "requests"} {
		for _, p := range Params[route] {
			id, ok := ui.Controls[route][p.Name]
			require.True(t, ok, "%s: parameter %s has no UI control", route, p.Name)
			require.Contains(t, string(page), `id="`+id+`"`, "%s: control %s of %s", route, id, p.Name)
		}
	}
}

// The page stands alone: it never tells a reader to run a command.
func TestUINeverShowsCommands(t *testing.T) {
	t.Parallel()
	for _, name := range []string{"", "app.js", "app.css"} {
		body, _, ok := ui.File(name)
		require.True(t, ok, name)
		require.NotContains(t, string(body), "rudder-cli", name)
	}
}
