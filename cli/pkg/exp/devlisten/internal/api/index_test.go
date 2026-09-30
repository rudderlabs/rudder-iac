package api

import (
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/rudderlabs/rudder-iac/cli/pkg/exp/devlisten/internal/store"
)

// A curl user finds every endpoint, its parameters and an example in the
// index, without the CLI.
func TestIndexListsEveryRoute(t *testing.T) {
	t.Parallel()
	srv, _ := newTestServer(t)
	h := New(store.New("x"), testIdentity, Config{})

	_, index := get(t, srv.URL+"/_dev/v1/")

	var paths []string
	for _, e := range index["endpoints"].([]any) {
		ep := e.(map[string]any)
		require.NotEmpty(t, ep["example"], ep["path"])
		require.True(t, strings.HasPrefix(ep["example"].(string), "http://127.0.0.1:4321/_dev/"), ep["example"])
		paths = append(paths, ep["path"].(string))
	}
	for path := range h.routes {
		if path == base || path == strings.TrimSuffix(base, "/") {
			continue
		}
		require.Contains(t, paths, path)
	}
	for _, path := range []string{base + "requests/{seq}", UIPath} {
		require.Contains(t, paths, path)
	}
	events := index["endpoints"].([]any)[slices.Index(paths, base+"events")].(map[string]any)
	require.Contains(t, events["params"], "writeKey")
}

func TestGuideIsServedAsMarkdown(t *testing.T) {
	t.Parallel()
	st := store.New(testIdentity.ServerID)
	srv := httptest.NewServer(New(st, testIdentity, Config{Guide: "# guide\n"}))
	t.Cleanup(srv.Close)

	resp, body := getRaw(t, srv.URL+"/_dev/v1/guide")

	require.Equal(t, http.StatusOK, resp.StatusCode)
	require.Equal(t, "text/markdown; charset=utf-8", resp.Header.Get("Content-Type"))
	require.Equal(t, "# guide\n", body)
}
