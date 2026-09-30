package api

import (
	"io"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
)

func getRaw(t *testing.T, url string) (*http.Response, string) {
	t.Helper()
	resp, err := http.Get(url)
	require.NoError(t, err)
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	return resp, string(body)
}

func TestProbePageLoadsTheSDKFromTheListener(t *testing.T) {
	t.Parallel()
	srv, _ := newTestServer(t)

	resp, page := getRaw(t, srv.URL+"/_dev/v1/probe.html")

	require.Equal(t, http.StatusOK, resp.StatusCode)
	require.Equal(t, "text/html; charset=utf-8", resp.Header.Get("Content-Type"))
	require.Contains(t, page, `import { RudderAnalytics } from './probe/analytics-js-3.31.4.mjs'`)
	require.Contains(t, page, `track('dev probe', { probe: true }`)
	require.Contains(t, page, `configUrl: origin`)
}

func TestProbeServesThePinnedBundleAndItsLicense(t *testing.T) {
	t.Parallel()
	srv, _ := newTestServer(t)

	resp, sdk := getRaw(t, srv.URL+"/_dev/v1/probe/analytics-js-3.31.4.mjs")
	require.Equal(t, http.StatusOK, resp.StatusCode)
	require.Equal(t, "text/javascript; charset=utf-8", resp.Header.Get("Content-Type"))
	require.Contains(t, sdk, "export { RudderAnalytics };")

	resp, license := getRaw(t, srv.URL+"/_dev/v1/probe/LICENSE-analytics-js.md")
	require.Equal(t, http.StatusOK, resp.StatusCode)
	require.Contains(t, license, "Elastic License 2.0")
}
