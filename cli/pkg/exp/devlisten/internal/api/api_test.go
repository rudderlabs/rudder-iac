package api

import (
	"encoding/json"
	"io"
	"maps"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/rudderlabs/rudder-iac/cli/pkg/exp/devlisten/internal/store"
)

var testIdentity = Identity{
	APIVersion: "v1", ServerID: "9f3ac1d2b7e4c601", URL: "http://127.0.0.1:4321", Port: 4321,
	Bind: "127.0.0.1", PID: 42, StartedAt: time.Date(2026, 9, 29, 12, 0, 0, 0, time.UTC),
	WriteKey: "dev", WriteKeyPolicy: "any",
}

func sp(s string) *string { return &s }

func ingestion(events ...store.Event) store.Record {
	return store.Record{Kind: "ingestion", Route: "/v1/batch", Transport: "http", StatusCode: 200, Outcome: "accepted",
		WriteKey: "dev", Request: store.Request{Body: `{"batch":[],"sentAt":"2026-09-29T12:00:00.100Z"}`}, Events: events}
}

func track(idx int, name string) store.Event {
	msg := `{"type":"track","event":"` + name + `","userId":"u1","properties":{"n":1},"context":{"traits":{"plan":"pro"}}}`
	return store.Event{Idx: idx, Type: sp("track"), Event: sp(name), UserID: sp("u1"), MessageID: sp("m"),
		Message: json.RawMessage(msg), EnrichedMessage: json.RawMessage(`{"enriched":true}`)}
}

func newTestServer(t *testing.T) (*httptest.Server, *store.Store) {
	t.Helper()
	st := store.New(testIdentity.ServerID)
	srv := httptest.NewServer(New(st, testIdentity, Config{}))
	t.Cleanup(srv.Close)
	return srv, st
}

func get(t *testing.T, url string, header ...string) (int, map[string]any) {
	t.Helper()
	req, err := http.NewRequest(http.MethodGet, url, nil)
	require.NoError(t, err)
	for i := 0; i+1 < len(header); i += 2 {
		req.Header.Set(header[i], header[i+1])
	}
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	require.Equal(t, "application/json; charset=utf-8", resp.Header.Get("Content-Type"))
	require.Equal(t, "no-store", resp.Header.Get("Cache-Control"))
	require.Empty(t, resp.Header.Get("Access-Control-Allow-Origin"))
	var out map[string]any
	require.NoError(t, json.Unmarshal(body, &out), string(body))
	return resp.StatusCode, out
}

type fetched struct {
	status int
	body   map[string]any
	err    error
}

func fetch(url string) fetched {
	resp, err := http.Get(url) //nolint:gosec,noctx // test URL
	if err != nil {
		return fetched{err: err}
	}
	defer resp.Body.Close()
	var body map[string]any
	err = json.NewDecoder(resp.Body).Decode(&body)
	return fetched{status: resp.StatusCode, body: body, err: err}
}

func TestEventsEnvelope(t *testing.T) {
	t.Parallel()
	srv, st := newTestServer(t)
	st.Append(store.Record{Kind: "control", Route: "/sourceConfig", StatusCode: 200})
	st.Append(ingestion(track(0, "A"), track(1, "B")))

	status, page := get(t, srv.URL+"/_dev/v1/events?since=0&event=B&view=full")

	require.Equal(t, http.StatusOK, status)
	events := page["events"].([]any)
	require.Less(t, page["waitedMs"], float64(1000))
	require.Contains(t, page, "summary")
	delete(page, "events")
	delete(page, "waitedMs")
	require.Equal(t, map[string]any{
		"apiVersion": "v1", "serverId": "9f3ac1d2b7e4c601", "since": float64(0), "cursor": float64(2),
		"hasMore": false, "timedOut": false, "view": "full", "omitted": nil, "truncated": nil,
	}, withoutKey(page, "summary"))
	require.Len(t, events, 1)
	item := events[0].(map[string]any)
	delete(item, "receivedAt")
	require.Equal(t, map[string]any{
		"seq": float64(2), "idx": float64(1), "route": "/v1/batch", "transport": "http", "statusCode": float64(200),
		"outcome": "accepted", "writeKey": "dev", "type": "track", "event": "B", "userId": "u1", "anonymousId": nil,
		"messageId": "m", "sentAt": "2026-09-29T12:00:00.100Z", "originalTimestamp": nil,
		"properties": map[string]any{"n": float64(1)}, "traits": map[string]any{"plan": "pro"},
		"message": map[string]any{"type": "track", "event": "B", "userId": "u1", "properties": map[string]any{"n": float64(1)},
			"context": map[string]any{"traits": map[string]any{"plan": "pro"}}},
		"enrichedMessage": map[string]any{"enriched": true},
	}, item)
}

func TestEventsPagesAtRequestBoundaries(t *testing.T) {
	t.Parallel()
	srv, st := newTestServer(t)
	st.Append(ingestion(track(0, "A"), track(1, "A")))
	st.Append(ingestion(track(0, "A")))
	st.Append(ingestion(track(0, "X")))

	_, first := get(t, srv.URL+"/_dev/v1/events?view=list&limit=1&event=A")
	require.Len(t, first["events"], 2, "a request is never split")
	require.Equal(t, true, first["hasMore"])
	require.Equal(t, float64(1), first["cursor"])

	_, second := get(t, srv.URL+"/_dev/v1/events?view=list&limit=1&event=A&since=1")
	require.Len(t, second["events"], 1)
	require.Equal(t, false, second["hasMore"])
	require.Equal(t, float64(3), second["cursor"], "the cursor covers unmatched requests scanned")
}

func TestUnknownParameterIsRejected(t *testing.T) {
	t.Parallel()
	srv, _ := newTestServer(t)

	status, body := get(t, srv.URL+"/_dev/v1/events?sinc=3")

	require.Equal(t, http.StatusBadRequest, status)
	require.Equal(t, map[string]any{"error": map[string]any{
		"code": "unknown_parameter", "message": "unknown parameter: sinc", "param": "sinc", "details": nil,
		"next": "rudder-cli dev events list --help",
	}}, body)
}

func TestParameterBounds(t *testing.T) {
	t.Parallel()
	srv, _ := newTestServer(t)

	for query, param := range map[string]string{
		"wait=5m":         "wait",
		"wait=x":          "wait",
		"since=-1":        "since",
		"limit=0":         "limit",
		"limit=5&min=6":   "min",
		"since=1&since=2": "since",
	} {
		status, body := get(t, srv.URL+"/_dev/v1/events?"+query)
		require.Equal(t, http.StatusBadRequest, status, query)
		errObj := body["error"].(map[string]any)
		require.Equal(t, "invalid_parameter", errObj["code"], query)
		require.Equal(t, param, errObj["param"], query)
	}
}

func TestServerIDMismatchIs409(t *testing.T) {
	t.Parallel()
	srv, _ := newTestServer(t)

	status, body := get(t, srv.URL+"/_dev/v1/events?serverId=other")

	require.Equal(t, http.StatusConflict, status)
	require.Equal(t, "server_changed", body["error"].(map[string]any)["code"])
}

// newWaitingServer reports on waiting each time a long-poll starts to wait,
// so a test appends only after the poll is registered.
func newWaitingServer(t *testing.T) (*httptest.Server, *store.Store, <-chan struct{}) {
	t.Helper()
	st := store.New(testIdentity.ServerID)
	h := New(st, testIdentity, Config{})
	waiting := make(chan struct{}, 8)
	h.waiting = func() { waiting <- struct{}{} }
	srv := httptest.NewServer(h)
	t.Cleanup(srv.Close)
	return srv, st, waiting
}

func awaitSignal(t *testing.T, ch <-chan struct{}) {
	t.Helper()
	select {
	case <-ch:
	case <-time.After(5 * time.Second):
		t.Fatal("the long-poll never started to wait")
	}
}

func TestLongPollWakesOnCapture(t *testing.T) {
	t.Parallel()
	srv, st, waiting := newWaitingServer(t)

	done := make(chan fetched, 1)
	go func() { done <- fetch(srv.URL + "/_dev/v1/events?view=list&event=B&wait=10s") }()

	awaitSignal(t, waiting)
	st.Append(ingestion(track(0, "A")))
	awaitSignal(t, waiting) // A woke the poll, did not match, and it waits again
	st.Append(ingestion(track(0, "B")))

	select {
	case res := <-done:
		require.NoError(t, res.err)
		require.Equal(t, http.StatusOK, res.status)
		require.Equal(t, false, res.body["timedOut"])
		require.Len(t, res.body["events"], 1)
		require.Equal(t, float64(2), res.body["cursor"])
	case <-time.After(5 * time.Second):
		t.Fatal("long-poll did not wake on capture")
	}
}

func TestLongPollTimesOutWith200(t *testing.T) {
	t.Parallel()
	srv, st := newTestServer(t)
	st.Append(ingestion(track(0, "A")))

	status, page := get(t, srv.URL+"/_dev/v1/events?event=B&wait=200ms")

	require.Equal(t, http.StatusOK, status)
	require.Equal(t, true, page["timedOut"])
	require.Empty(t, page["events"])
	require.GreaterOrEqual(t, page["waitedMs"], float64(200))
	require.Equal(t, float64(1), page["cursor"])
	require.Equal(t, map[string]any{"B": float64(0)}, page["summary"].(map[string]any)["byEvent"])
}

func TestShutdownWakesLongPollWith503(t *testing.T) {
	t.Parallel()
	srv, st, waiting := newWaitingServer(t)

	done := make(chan fetched, 1)
	go func() { done <- fetch(srv.URL + "/_dev/v1/events?wait=10s") }()
	awaitSignal(t, waiting)
	st.Close()

	select {
	case res := <-done:
		require.NoError(t, res.err)
		require.Equal(t, http.StatusServiceUnavailable, res.status)
		require.Equal(t, "shutting_down", res.body["error"].(map[string]any)["code"])
	case <-time.After(5 * time.Second):
		t.Fatal("long-poll did not wake on shutdown")
	}
}

func TestCrossSiteBrowserRequestIsRefused(t *testing.T) {
	t.Parallel()
	srv, _ := newTestServer(t)

	status, body := get(t, srv.URL+"/_dev/v1/info", "Sec-Fetch-Site", "cross-site")
	require.Equal(t, http.StatusForbidden, status)
	require.Equal(t, "browser_origin", body["error"].(map[string]any)["code"])

	status, _ = get(t, srv.URL+"/_dev/v1/info", "Sec-Fetch-Site", "none")
	require.Equal(t, http.StatusOK, status)
}

func TestWrongMethodAndUnknownRoute(t *testing.T) {
	t.Parallel()
	srv, _ := newTestServer(t)

	req, err := http.NewRequest(http.MethodOptions, srv.URL+"/_dev/v1/events", nil)
	require.NoError(t, err)
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	resp.Body.Close()
	require.Equal(t, http.StatusMethodNotAllowed, resp.StatusCode)
	require.Equal(t, "GET", resp.Header.Get("Allow"))

	status, body := get(t, srv.URL+"/_dev/v1/nope")
	require.Equal(t, http.StatusNotFound, status)
	require.Equal(t, "not_found", body["error"].(map[string]any)["code"])
}

func TestInfo(t *testing.T) {
	t.Parallel()
	srv, st := newTestServer(t)
	st.Append(ingestion(track(0, "A")))
	st.Append(store.Record{Kind: "control"})

	status, info := get(t, srv.URL+"/_dev/v1/info")

	require.Equal(t, http.StatusOK, status)
	require.Equal(t, map[string]any{
		"ready": true, "apiVersion": "v1", "serverId": "9f3ac1d2b7e4c601", "url": "http://127.0.0.1:4321",
		"port": float64(4321), "bind": "127.0.0.1", "pid": float64(42), "startedAt": "2026-09-29T12:00:00Z",
		"writeKey": "dev", "writeKeyPolicy": "any", "writeKeys": []any{}, "recordVersion": float64(1),
		"cursor": float64(2), "exposed": false, "store": map[string]any{"requests": float64(1), "events": float64(1), "control": float64(1),
			"bytes": float64(st.Stats().Bytes), "evicted": float64(0), "evictedThrough": float64(0),
			"maxRequests": float64(store.DefaultMaxRecords), "maxBytes": float64(store.DefaultMaxBytes)},
	}, info)
}

func TestRequestsFiltersByKindAndHidesBodies(t *testing.T) {
	t.Parallel()
	srv, st := newTestServer(t)
	st.Append(ingestion(track(0, "A")))
	st.Append(store.Record{Kind: "control", Route: "/sourceConfig"})

	_, page := get(t, srv.URL+"/_dev/v1/requests?view=compact")
	requests := page["requests"].([]any)
	require.Len(t, requests, 1)
	rec := requests[0].(map[string]any)
	require.NotContains(t, rec, "request")
	require.NotContains(t, rec["events"].([]any)[0], "message")
	require.NotContains(t, rec["events"].([]any)[0], "enrichedMessage")

	_, page = get(t, srv.URL+"/_dev/v1/requests?kind=control&since=0")
	require.Len(t, page["requests"], 1)
	require.Equal(t, float64(2), page["cursor"])

	_, page = get(t, srv.URL+"/_dev/v1/requests?kind=all")
	require.Len(t, page["requests"], 2)

	status, _ := get(t, srv.URL+"/_dev/v1/requests?kind=nope")
	require.Equal(t, http.StatusBadRequest, status)
}

func withoutKey(m map[string]any, key string) map[string]any {
	out := maps.Clone(m)
	delete(out, key)
	return out
}

// withWriteKey stores key the way the gateway does.
func withWriteKey(rec store.Record, key string) store.Record {
	store.SetWriteKey(&rec, key)
	return rec
}
