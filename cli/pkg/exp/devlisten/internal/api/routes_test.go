package api

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/rudderlabs/rudder-iac/cli/pkg/exp/devlisten/internal/store"
)

func do(t *testing.T, method, url, contentType string, mutate ...func(*http.Request)) (int, map[string]any) {
	t.Helper()
	req, err := http.NewRequest(method, url, strings.NewReader("{}"))
	require.NoError(t, err)
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	for _, m := range mutate {
		m(req)
	}
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	var out map[string]any
	require.NoError(t, json.Unmarshal(body, &out), string(body))
	return resp.StatusCode, out
}

func TestIndexNamesTheFirstCall(t *testing.T) {
	t.Parallel()
	srv, _ := newTestServer(t)

	status, index := get(t, srv.URL+"/_dev/v1/")

	require.Equal(t, http.StatusOK, status)
	require.Equal(t, "rudder-cli dev summary --since 0 --json", index["next"])
	require.Equal(t, "curl -fsS 'http://127.0.0.1:4321/_dev/v1/summary?serverId=9f3ac1d2b7e4c601'", index["curl"])
	require.Equal(t, "rudder-cli dev --help", index["help"])
	require.Contains(t, index["links"], "summary")
	raw, err := json.Marshal(index)
	require.NoError(t, err)
	require.Less(t, len(raw), 1024)
}

func TestHostCheckOnLoopbackBind(t *testing.T) {
	t.Parallel()
	st := store.New(testIdentity.ServerID)
	srv := httptest.NewServer(New(st, testIdentity, Config{CheckHost: true}))
	t.Cleanup(srv.Close)

	for host, want := range map[string]int{
		"127.0.0.1:4321":        http.StatusOK,
		"localhost:4321":        http.StatusOK,
		"[::1]:4321":            http.StatusOK,
		"attacker.example:4321": http.StatusForbidden,
		"127.0.0.1:9999":        http.StatusForbidden,
		"127.0.0.1":             http.StatusForbidden,
	} {
		status, body := do(t, http.MethodGet, srv.URL+"/_dev/v1/info", "", func(r *http.Request) { r.Host = host })
		require.Equal(t, want, status, host)
		if want == http.StatusForbidden {
			require.Equal(t, "host_not_allowed", body["error"].(map[string]any)["code"], host)
		}
	}
}

func TestRequestBySeq(t *testing.T) {
	t.Parallel()
	srv, st := newTestServer(t)
	rec := ingestion(track(0, "A"))
	rec.Request.Body = "hello"
	st.Append(rec)

	status, got := get(t, srv.URL+"/_dev/v1/requests/1")
	require.Equal(t, http.StatusOK, status)
	require.Equal(t, float64(1), got["seq"])
	require.Equal(t, "hello", got["request"].(map[string]any)["body"], "the single record is whole")
	require.Contains(t, got["events"].([]any)[0], "enrichedMessage")

	_, fields := get(t, srv.URL+"/_dev/v1/requests/1?fields=request.body&fields=events.idx")
	require.Equal(t, map[string]any{
		"seq":     float64(1),
		"request": map[string]any{"body": "hello"},
		"events":  []any{map[string]any{"idx": float64(0)}},
	}, fields)

	status, body := get(t, srv.URL+"/_dev/v1/requests/9")
	require.Equal(t, http.StatusNotFound, status)
	require.Equal(t, "rudder-cli dev info --json", body["error"].(map[string]any)["next"])

	status, _ = get(t, srv.URL+"/_dev/v1/requests/x")
	require.Equal(t, http.StatusBadRequest, status)

	_, small := get(t, srv.URL+"/_dev/v1/requests/1?maxBytes=100")
	require.Equal(t, float64(1), small["seq"])
	require.NotContains(t, small, "request")
	truncated := small["truncated"].(map[string]any)
	require.Greater(t, truncated["requestBytes"], float64(100))
	require.Equal(t, "rudder-cli dev requests show 1 --fields request.headers --fields response --json", truncated["next"])
}

func TestRequestsListFiltersAndOmitted(t *testing.T) {
	t.Parallel()
	srv, st := newTestServer(t)
	st.Append(ingestion(track(0, "A")))
	bad := ingestion()
	bad.StatusCode, bad.Failed, bad.Outcome = 401, true, "rejected"
	bad.Rejection = &store.Rejection{Stage: "auth", Reason: "no write key"}
	st.Append(bad)

	_, page := get(t, srv.URL+"/_dev/v1/requests")
	require.Equal(t, map[string]any{
		"fields":  []any{"request.body", "request.bodyBase64", "events.enrichedMessage"},
		"context": []any{},
		"next":    "rudder-cli dev requests show 1 --json",
	}, page["omitted"])

	for query, want := range map[string]int{
		"failed=true":            1,
		"failed=false":           1,
		"stage=auth":             1,
		"statusCode=401":         1,
		"route=/v1/batch":        2,
		"failed=true&stage=body": 0,
	} {
		_, page := get(t, srv.URL+"/_dev/v1/requests?"+query)
		require.Len(t, page["requests"], want, query)
	}
	status, _ := get(t, srv.URL+"/_dev/v1/requests?failed=maybe")
	require.Equal(t, http.StatusBadRequest, status)
}

func TestResetKeepsCounting(t *testing.T) {
	t.Parallel()
	srv, st := newTestServer(t)
	st.Append(ingestion(track(0, "A")))
	st.Append(store.Record{Kind: "control"})

	status, body := do(t, http.MethodPost, srv.URL+"/_dev/v1/reset", "application/json")
	require.Equal(t, http.StatusOK, status)
	require.Equal(t, map[string]any{
		"cursor":  float64(2),
		"removed": map[string]any{"requests": float64(1), "events": float64(1), "control": float64(1)},
	}, body)

	require.Equal(t, uint64(3), st.Append(ingestion()).Seq)
	status, _ = do(t, http.MethodPost, srv.URL+"/_dev/v1/reset", "text/plain")
	require.Equal(t, http.StatusUnsupportedMediaType, status)
}

func TestShutdownCallsTheHookOnLoopbackOnly(t *testing.T) {
	t.Parallel()
	var called atomic.Int32
	st := store.New(testIdentity.ServerID)
	srv := httptest.NewServer(New(st, testIdentity, Config{Shutdown: func() { called.Add(1) }}))
	t.Cleanup(srv.Close)

	status, body := do(t, http.MethodPost, srv.URL+"/_dev/v1/shutdown", "application/json")
	require.Equal(t, http.StatusAccepted, status)
	require.Equal(t, map[string]any{"stopping": true, "serverId": testIdentity.ServerID}, body)
	require.Eventually(t, func() bool { return called.Load() == 1 }, timeoutShort, tick)

	remote := testIdentity
	remote.Bind = "0.0.0.0"
	srv2 := httptest.NewServer(New(st, remote, Config{Shutdown: func() { called.Add(1) }}))
	t.Cleanup(srv2.Close)
	status, body = do(t, http.MethodPost, srv2.URL+"/_dev/v1/shutdown", "application/json")
	require.Equal(t, http.StatusForbidden, status)
	require.Equal(t, "stop_disabled", body["error"].(map[string]any)["code"])
}

func TestEveryErrorCarriesNext(t *testing.T) {
	t.Parallel()
	srv, _ := newTestServer(t)

	for _, tc := range []struct{ method, path, ct string }{
		{http.MethodGet, "/_dev/v1/events?sinc=1", ""},
		{http.MethodGet, "/_dev/v1/events?serverId=other", ""},
		{http.MethodGet, "/_dev/v1/nope", ""},
		{http.MethodPut, "/_dev/v1/events", ""},
		{http.MethodPost, "/_dev/v1/reset", "text/plain"},
		{http.MethodGet, "/_dev/v1/summary?limit=1", ""},
	} {
		_, body := do(t, tc.method, srv.URL+tc.path, tc.ct)
		errObj := body["error"].(map[string]any)
		next, _ := errObj["next"].(string)
		require.NotEmpty(t, next, tc.path)
	}
}

func TestRequestsTruncatedNextKeepsTheFilters(t *testing.T) {
	t.Parallel()
	srv, st := newTestServer(t)
	for range 4 {
		rec := ingestion(browserTrack(0, "A"))
		rec.StatusCode, rec.Failed, rec.Outcome = 400, true, "rejected"
		rec.Rejection = &store.Rejection{Stage: "body"}
		st.Append(rec)
	}

	_, page := get(t, srv.URL+"/_dev/v1/requests?serverId=9f3ac1d2b7e4c601&kind=all&route=/v1/batch&statusCode=400"+
		"&failed=true&stage=body&fields=events&maxBytes=1500")
	truncated := page["truncated"].(map[string]any)
	require.Equal(t, "rudder-cli dev requests list --since "+jsonNumber(page["cursor"])+" --server-id '9f3ac1d2b7e4c601'"+
		" --kind 'all' --route '/v1/batch' --status-code '400' --failed=true --stage 'body' --fields 'events'"+
		" --max-bytes '1500' --json", truncated["next"])

	_, page = get(t, srv.URL+"/_dev/v1/requests?maxBytes=100")
	require.Equal(t, "rudder-cli dev requests list --since 0 --fields request.headers --json",
		page["truncated"].(map[string]any)["next"])
}
