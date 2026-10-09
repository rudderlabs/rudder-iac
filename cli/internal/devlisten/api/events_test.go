package api

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/rudderlabs/rudder-iac/cli/internal/devlisten/ingest"
	"github.com/rudderlabs/rudder-iac/cli/internal/devlisten/store"
)

// sdk sends one request through the real ingestion handler, so the records
// under test are the records local event-stream serve keeps. An empty key sends no
// Authorization header.
func sdk(t *testing.T, st *store.Store, method, target, key, body string, allow ...string) int {
	t.Helper()
	r := httptest.NewRequest(method, target, strings.NewReader(body))
	if key != "" {
		r.SetBasicAuth(key, "")
	}
	w := httptest.NewRecorder()
	ingest.New(st, allow, "test").ServeHTTP(w, r)
	return w.Code
}

func track(t *testing.T, st *store.Store, body string) {
	t.Helper()
	require.Equal(t, http.StatusOK, sdk(t, st, http.MethodPost, "/v1/track", "dev", body))
}

func lines(body string) []string {
	if body == "" {
		return nil
	}
	return strings.Split(strings.TrimSuffix(body, "\n"), "\n")
}

func TestStreamPrintsEachAcceptedEventAsSent(t *testing.T) {
	t.Parallel()
	h, st := newTestHandler("127.0.0.1")
	batch := `{"batch":[
		{"type":"identify","userId":"u1"},
		{
			"type": "track",
			"userId": "u1",
			"event": "Order Completed",
			"properties": {"total": 1.0, "n": 1e2, "s": "é"}
		},
		{"type":"page","anonymousId":"a1","name":"Home"}
	]}`
	require.Equal(t, http.StatusOK, sdk(t, st, http.MethodPost, "/v1/batch", "dev", batch))
	require.Equal(t, http.StatusBadRequest, sdk(t, st, http.MethodPost, "/v1/track", "dev", `{"event":"No Identity"}`))
	track(t, st, `{"userId":"u2","event":"Signed Up"}`)
	require.Equal(t, http.StatusOK, sdk(t, st, http.MethodGet, "/sourceConfig", "dev", ""))

	w := get(h, "/_local/v1/events", nil)

	require.Equal(t, http.StatusOK, w.Code)
	require.Equal(t, "application/x-ndjson", w.Header().Get("Content-Type"))
	require.Equal(t, []string{
		`{"type":"identify","userId":"u1"}`,
		`{"type":"track","userId":"u1","event":"Order Completed","properties":{"total":1.0,"n":1e2,"s":"é"}}`,
		`{"type":"page","anonymousId":"a1","name":"Home"}`,
		`{"userId":"u2","event":"Signed Up"}`,
	}, lines(w.Body.String()), "a track sent without type prints without type")
	require.Equal(t, "4", w.Header().Get("X-Local-Cursor"), "control requests move the cursor too")
	require.Equal(t, "false", w.Header().Get("X-Local-Has-More"))
	require.Equal(t, "9f3ac1d2b7e4c601", w.Header().Get("X-Local-Server-Id"))
	require.Empty(t, w.Header().Get("Link"))
	require.Equal(t, "no-store", w.Header().Get("Cache-Control"))
}

func TestStreamOfAnEmptyStoreIsAnEmptyBody(t *testing.T) {
	t.Parallel()
	h, _ := newTestHandler("127.0.0.1")

	w := get(h, "/_local/v1/events?since=7", nil)

	require.Equal(t, http.StatusOK, w.Code)
	require.Empty(t, w.Body.String())
	require.Equal(t, "0", w.Header().Get("X-Local-Cursor"))
	require.Equal(t, "false", w.Header().Get("X-Local-Has-More"))
}

// A page never splits a request, and the cursor of each page continues the
// read with no event skipped or repeated.
func TestStreamPages(t *testing.T) {
	t.Parallel()
	h, st := newTestHandler("127.0.0.1")
	track(t, st, `{"userId":"u","event":"e1"}`)                                             // seq 1
	require.Equal(t, http.StatusOK, sdk(t, st, http.MethodGet, "/sourceConfig", "dev", "")) // seq 2
	require.Equal(t, http.StatusOK, sdk(t, st, http.MethodPost, "/v1/batch", "dev",         // seq 3
		`{"batch":[{"userId":"u","event":"e2"},{"userId":"u","event":"e3"}]}`))
	track(t, st, `{"userId":"u","event":"e4"}`)                                     // seq 4
	require.Equal(t, http.StatusOK, sdk(t, st, http.MethodPost, "/v1/batch", "dev", // seq 5
		`{"batch":[{"userId":"u","event":"e5"},{"userId":"u","event":"e6"},{"userId":"u","event":"e7"}]}`))

	for _, tc := range []struct {
		target  string
		events  []string
		cursor  string
		hasMore string
		link    string
	}{
		{
			target: "/_local/v1/events?limit=2&event=e*", events: []string{"e1"}, cursor: "2", hasMore: "true",
			link: `</_local/v1/events?event=e%2A&limit=2&since=2>; rel="next"`,
		},
		{
			target: "/_local/v1/events?event=e*&limit=2&since=2", events: []string{"e2", "e3"}, cursor: "3", hasMore: "true",
			link: `</_local/v1/events?event=e%2A&limit=2&since=3>; rel="next"`,
		},
		{
			target: "/_local/v1/events?event=e*&limit=2&since=3", events: []string{"e4"}, cursor: "4", hasMore: "true",
			link: `</_local/v1/events?event=e%2A&limit=2&since=4>; rel="next"`,
		},
		{
			target: "/_local/v1/events?event=e*&limit=2&since=4", events: []string{"e5", "e6", "e7"}, cursor: "5", hasMore: "false",
		},
		{target: "/_local/v1/events?limit=1000&since=5", cursor: "5", hasMore: "false"},
	} {
		w := get(h, tc.target, nil)
		require.Equal(t, http.StatusOK, w.Code, tc.target)

		var got []string
		for _, line := range lines(w.Body.String()) {
			var ev struct{ Event string }
			require.NoError(t, json.Unmarshal([]byte(line), &ev))
			got = append(got, ev.Event)
		}
		require.Equal(t, tc.events, got, tc.target)
		require.Equal(t, tc.cursor, w.Header().Get("X-Local-Cursor"), tc.target)
		require.Equal(t, tc.hasMore, w.Header().Get("X-Local-Has-More"), tc.target)
		require.Equal(t, tc.link, w.Header().Get("Link"), tc.target)
	}
}

func TestStreamHeadSendsThePagingHeaders(t *testing.T) {
	t.Parallel()
	h, st := newTestHandler("127.0.0.1")
	track(t, st, `{"userId":"u","event":"e1"}`)
	track(t, st, `{"userId":"u","event":"e2"}`)
	srv := httptest.NewServer(h)
	t.Cleanup(srv.Close)

	resp, err := http.Head(srv.URL + "/_local/v1/events?limit=1")
	require.NoError(t, err)
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	require.NoError(t, resp.Body.Close())

	require.Equal(t, http.StatusOK, resp.StatusCode)
	require.Empty(t, body)
	require.Equal(t, "1", resp.Header.Get("X-Local-Cursor"))
	require.Equal(t, "true", resp.Header.Get("X-Local-Has-More"))
	require.Equal(t, `</_local/v1/events?limit=1&since=1>; rel="next"`, resp.Header.Get("Link"))
	require.Equal(t, "9f3ac1d2b7e4c601", resp.Header.Get("X-Local-Server-Id"))
}

func TestStreamViewsOnlySelectKeys(t *testing.T) {
	t.Parallel()
	h, st := newTestHandler("127.0.0.1")
	track(t, st, `{"type":"track","event":"Order Completed","userId":"u1","messageId":"m1",`+
		`"context":{"library":{"name":"js"},"page":{"path":"/cart"},"traits":{"plan":"pro"},"sessionId":1,"custom":"kept"},`+
		`"properties":{"total":"2","big":1.50e2}}`)
	track(t, st, `{"event":"Bare","userId":"u2","context":{"os":{"name":"mac"}}}`)

	for _, tc := range []struct {
		query string
		want  []string
	}{
		{
			query: "view=compact",
			want: []string{
				`{"type":"track","event":"Order Completed","userId":"u1","messageId":"m1","context":{"custom":"kept"},"properties":{"total":"2","big":1.50e2}}`,
				`{"event":"Bare","userId":"u2"}`,
			},
		},
		{
			query: "fields=properties",
			want:  []string{`{"properties":{"total":"2","big":1.50e2}}`, `{}`},
		},
		{
			query: "fields=context.page.path&fields=event&fields=context.library",
			want: []string{
				`{"context":{"page":{"path":"/cart"},"library":{"name":"js"}},"event":"Order Completed"}`,
				`{"event":"Bare"}`,
			},
		},
		{
			query: "fields=properties.total&fields=properties",
			want:  []string{`{"properties":{"total":"2","big":1.50e2}}`, `{}`},
		},
	} {
		w := get(h, "/_local/v1/events?"+tc.query, nil)
		require.Equal(t, http.StatusOK, w.Code, w.Body.String())
		require.Equal(t, tc.want, lines(w.Body.String()), tc.query)
	}
}

func TestStreamFilters(t *testing.T) {
	t.Parallel()
	const longKey = "fake-write-key-for-tests"
	h, st := newTestHandler("127.0.0.1")
	require.Equal(t, http.StatusOK, sdk(t, st, http.MethodPost, "/v1/track", "web", `{"userId":"u1","event":"Order Completed"}`))
	require.Equal(t, http.StatusOK, sdk(t, st, http.MethodPost, "/v1/page", longKey, `{"anonymousId":"a1","name":"Home"}`))
	require.Equal(t, http.StatusOK, sdk(t, st, http.MethodPost, "/v1/batch", "", `{"batch":[`+
		`{"type":"track","userId":"u2","event":"Order Refunded"},{"type":"identify","userId":"u1"}]}`))

	for _, tc := range []struct {
		query string
		want  []string
	}{
		{query: "event=Order+Completed", want: []string{"u1 Order Completed"}},
		{query: "event=Order*", want: []string{"u1 Order Completed", "u2 Order Refunded"}},
		{query: "event=order*"},
		{query: "event=Home&event=Order+Refunded", want: []string{"u2 Order Refunded"}},
		{query: "type=page", want: []string{"a1 "}},
		{query: "type=track", want: []string{"u1 Order Completed", "u2 Order Refunded"}},
		{query: "userId=u1", want: []string{"u1 Order Completed", "u1 "}},
		{query: "anonymousId=a1", want: []string{"a1 "}},
		{query: "writeKey=web", want: []string{"u1 Order Completed"}},
		{query: "writeKey=" + longKey, want: []string{"a1 "}},
		{query: "writeKey=fake...ests"},
		{query: "writeKey=", want: []string{"u2 Order Refunded", "u1 "}},
		{query: "writeKey=web&writeKey=", want: []string{"u1 Order Completed", "u2 Order Refunded", "u1 "}},
		{query: "userId=u1&type=identify", want: []string{"u1 "}},
	} {
		w := get(h, "/_local/v1/events?"+tc.query, nil)
		require.Equal(t, http.StatusOK, w.Code, w.Body.String())
		var got []string
		for _, line := range lines(w.Body.String()) {
			var ev struct{ UserID, AnonymousID, Event string }
			require.NoError(t, json.Unmarshal([]byte(line), &ev))
			got = append(got, ev.UserID+ev.AnonymousID+" "+ev.Event)
		}
		require.Equal(t, tc.want, got, tc.query)
	}
}

// A slow upload that started before the window but completed inside it has
// a seq in the window and a receivedAt before it.
func TestSinceAsATime(t *testing.T) {
	t.Parallel()
	h, st := newTestHandler("127.0.0.1")
	now := time.Date(2026, 9, 30, 12, 10, 0, 0, time.UTC)
	h.now = func() time.Time { return now }
	at := func(ago time.Duration, event string) {
		r := httptest.NewRequest(http.MethodPost, "/v1/track", nil)
		st.Capture(&ingest.Capture{
			ReceivedAt: now.Add(-ago), Kind: "ingestion", Route: "/v1/track", Request: r, StatusCode: 200,
			Events: []ingest.Event{{Message: []byte(`{"userId":"u","event":"` + event + `"}`), Enrichment: &ingest.Enrichment{}}},
		})
	}
	at(20*time.Minute, "old")        // seq 1
	at(3*time.Minute, "new")         // seq 2
	at(9*time.Minute, "slow upload") // seq 3
	at(time.Minute, "newest")        // seq 4

	for _, since := range []string{"5m", "300s", "2026-09-30T12:05:00Z", "2026-09-30T14:05:00%2B02:00"} {
		w := get(h, "/_local/v1/events?since="+since, nil)
		require.Equal(t, http.StatusOK, w.Code, w.Body.String())
		require.Equal(t, []string{`{"userId":"u","event":"new"}`, `{"userId":"u","event":"newest"}`},
			lines(w.Body.String()), since)

		var summary struct {
			Since   uint64 `json:"since"`
			Summary struct {
				Events struct{ Total int } `json:"events"`
			} `json:"summary"`
		}
		require.NoError(t, json.Unmarshal(get(h, "/_local/v1/events?view=counts&since="+since, nil).Body.Bytes(), &summary))
		require.Equal(t, uint64(1), summary.Since, "the cursor before the first request in the window")
		require.Equal(t, 2, summary.Summary.Events.Total)
	}

	w := get(h, "/_local/v1/events?since=30s", nil)
	require.Empty(t, w.Body.String())
	require.Equal(t, "4", w.Header().Get("X-Local-Cursor"))
}

func TestEventsParameterErrors(t *testing.T) {
	t.Parallel()
	const (
		listHelp   = "rudder-cli local event-stream events list --help"
		eventsHelp = "rudder-cli local event-stream events summary --help"
	)
	for _, tc := range []struct {
		query string
		code  string
		param string
		next  string
	}{
		{query: "bogus=1", code: "unknown_parameter", param: "bogus", next: listHelp},
		{query: "order=asc", code: "unknown_parameter", param: "order", next: listHelp},
		{query: "since=1&since=2", code: "invalid_parameter", param: "since", next: listHelp},
		{query: "userId=a&userId=b", code: "invalid_parameter", param: "userId", next: listHelp},
		{query: "since=-1", code: "invalid_parameter", param: "since", next: listHelp},
		{query: "since=-5m", code: "invalid_parameter", param: "since", next: listHelp},
		{query: "since=yesterday", code: "invalid_parameter", param: "since", next: listHelp},
		{query: "view=table", code: "invalid_parameter", param: "view", next: listHelp},
		{query: "view=compact&fields=properties", code: "invalid_parameter", param: "fields", next: listHelp},
		{query: "view=counts&fields=properties", code: "invalid_parameter", param: "fields", next: eventsHelp},
		{query: "fields=properties,event", code: "invalid_parameter", param: "fields", next: listHelp},
		{query: "fields=context..page", code: "invalid_parameter", param: "fields", next: listHelp},
		{query: "limit=0", code: "invalid_parameter", param: "limit", next: listHelp},
		{query: "limit=1001", code: "invalid_parameter", param: "limit", next: listHelp},
		{query: "limit=ten", code: "invalid_parameter", param: "limit", next: listHelp},
		{query: "wait=5s", code: "invalid_parameter", param: "wait", next: listHelp},
		{query: "min=2", code: "invalid_parameter", param: "min", next: listHelp},
		{query: "view=counts&limit=5", code: "invalid_parameter", param: "limit", next: eventsHelp},
		{query: "view=counts&wait=111s", code: "invalid_parameter", param: "wait", next: eventsHelp},
		{query: "view=counts&wait=-1s", code: "invalid_parameter", param: "wait", next: eventsHelp},
		{query: "view=counts&wait=soon", code: "invalid_parameter", param: "wait", next: eventsHelp},
		{query: "view=counts&min=0", code: "invalid_parameter", param: "min", next: eventsHelp},
		{query: "event=Order%ZZ", code: "invalid_parameter", next: listHelp},
	} {
		t.Run(tc.query, func(t *testing.T) {
			t.Parallel()
			h, _ := newTestHandler("127.0.0.1")

			w := get(h, "/_local/v1/events?"+tc.query, nil)

			require.Equal(t, http.StatusBadRequest, w.Code)
			require.Equal(t, "application/json; charset=utf-8", w.Header().Get("Content-Type"))
			e := decodeError(t, w)
			require.Equal(t, tc.code, e.Error.Code)
			require.Equal(t, tc.next, e.Error.Next)
			if tc.param == "" {
				require.Nil(t, e.Error.Param)
				return
			}
			require.Equal(t, tc.param, *e.Error.Param)
		})
	}
}

func TestEventsMessages(t *testing.T) {
	t.Parallel()
	h, _ := newTestHandler("127.0.0.1")
	for query, want := range map[string]string{
		"view=counts&wait=5m":        "wait: 5m exceeds the maximum of 110s",
		"fields=a,b":                 "fields: repeat the parameter for each path, as fields=a&fields=b",
		"since=1&since=2":            "since: repeated; give it once",
		"view=compact&fields=a":      "fields: cannot combine with view",
		"since=yesterday":            `since: "yesterday" is not a cursor, a duration such as 5m or an RFC 3339 time`,
		"view=counts&limit=1":        "limit: applies to the stream, not to view=counts",
		"wait=1s":                    "wait: applies to view=counts only",
		"serverId=a&serverId=b&x=%z": "The query string is malformed: invalid URL escape \"%z\".",
	} {
		require.Equal(t, want, decodeError(t, get(h, "/_local/v1/events?"+query, nil)).Error.Message, query)
	}
}

func TestEventsServerChanged(t *testing.T) {
	t.Parallel()
	h, _ := newTestHandler("127.0.0.1")

	for _, view := range []string{"", "&view=counts"} {
		w := get(h, "/_local/v1/events?serverId=0000000000000000"+view, nil)

		require.Equal(t, http.StatusConflict, w.Code)
		e := decodeError(t, w)
		require.Equal(t, "server_changed", e.Error.Code)
		require.Equal(t, "rudder-cli local event-stream events summary --json", e.Error.Next)
		require.Equal(t, map[string]any{"serverId": "9f3ac1d2b7e4c601", "startedAt": "2026-09-30T12:00:00Z"}, e.Error.Details)
	}
	require.Equal(t, http.StatusOK, get(h, "/_local/v1/events?serverId=9f3ac1d2b7e4c601", nil).Code)
}

func TestSummary(t *testing.T) {
	t.Parallel()
	h, st := newTestHandler("127.0.0.1")
	require.Equal(t, http.StatusOK, sdk(t, st, http.MethodGet, "/sourceConfig", "web", ""))
	require.Equal(t, http.StatusOK, sdk(t, st, http.MethodPost, "/v1/batch", "web", `{"batch":[`+
		`{"type":"track","userId":"u1","event":"chatOpened","channel":"web","context":{"library":{"name":"RudderLabs JavaScript SDK"}}},`+
		`{"type":"track","userId":"u1","event":"chatSuggestionClicked","channel":"web"}]}`))
	require.Equal(t, http.StatusBadRequest, sdk(t, st, http.MethodPost, "/v1/track", "api",
		`{"event":"Order Completed","channel":"server"}`))

	w := get(h, "/_local/v1/events?view=counts&event=chatOpened&event=Order+Completed&event=chat*", nil)

	require.Equal(t, http.StatusOK, w.Code)
	require.Equal(t, "application/json; charset=utf-8", w.Header().Get("Content-Type"))
	require.Equal(t, `{"apiVersion":"v1","serverId":"9f3ac1d2b7e4c601","since":0,"cursor":3,"evictedThrough":0,`+
		`"timedOut":false,"waitedMs":0,"summary":{`+
		`"requests":{"total":2,"failed":1,"byRoute":{"/v1/batch":1,"/v1/track":1},"byStatusCode":{"200":1,"400":1},`+
		`"byOutcome":{"accepted":1,"rejected":1},"byStage":{"identity":1}},`+
		`"events":{"total":2,"byType":{"track":2}},`+
		`"byEvent":{"Order Completed":0,"chatOpened":1,"chatSuggestionClicked":1},`+
		`"rejected":{"events":1,"byEvent":{"Order Completed":1}},`+
		`"byWriteKey":{"api":{"requests":1,"events":0},"web":{"requests":1,"events":2}},`+
		`"control":{"total":0,"sourceConfig":0,"sourceConfigFailed":0,"preflight":0,"pluginPath":0,"other":0},`+
		`"bySource":{"byChannel":{"server":1,"web":1},"bySdk":{"browser":{"requests":1,"control":0},"other":{"requests":1,"control":0}}},`+
		`"diagnosis":[{"code":"body_rejected","count":1,`+
		`"message":"1 request was rejected while the body was read: bad gzip, empty body, invalid JSON, wrong batch shape, no identity, or too large. Read rejection.reason.",`+
		`"next":"curl -fsS 'http://127.0.0.1:4321/_local/v1/requests?since=0&failed=true&view=compact'"}]},`+
		`"next":"rudder-cli local event-stream events summary --since 3 --event chatOpened --event 'Order Completed' --event 'chat*' --json"}`+"\n",
		w.Body.String())

	unfiltered := get(h, "/_local/v1/events?view=counts&since=0", nil).Body.String()
	require.Contains(t, unfiltered, `"control":{"total":1,"sourceConfig":1,"sourceConfigFailed":0,"preflight":0,"pluginPath":0,"other":0}`)
	require.Contains(t, unfiltered, `"bySdk":{"browser":{"requests":1,"control":0},"other":{"requests":1,"control":1}}`)
	require.Contains(t, unfiltered, `"byWriteKey":{"api":{"requests":1,"events":0},"web":{"requests":1,"events":2}}`)
}

func TestSummaryWriteKeysAppearWithZero(t *testing.T) {
	t.Parallel()
	h, st := newTestHandler("127.0.0.1")
	track(t, st, `{"userId":"u","event":"e"}`)

	w := get(h, "/_local/v1/events?view=counts&writeKey=worker&writeKey=fake-write-key-for-tests&writeKey=dev", nil)

	require.Contains(t, w.Body.String(),
		`"byWriteKey":{"dev":{"requests":1,"events":1},"fake...ests":{"requests":0,"events":0},"worker":{"requests":0,"events":0}}`)
}

type diagnosis struct {
	Code    string `json:"code"`
	Count   int    `json:"count"`
	Message string `json:"message"`
	Next    string `json:"next"`
}

func summaryOf(t *testing.T, h http.Handler, query string) (out struct {
	Since    uint64 `json:"since"`
	Cursor   uint64 `json:"cursor"`
	TimedOut bool   `json:"timedOut"`
	WaitedMs int64  `json:"waitedMs"`
	Summary  struct {
		Events    struct{ Total int } `json:"events"`
		Diagnosis []diagnosis         `json:"diagnosis"`
	} `json:"summary"`
	Next string `json:"next"`
},
) {
	t.Helper()
	w := get(h, "/_local/v1/events?view=counts&"+query, nil)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &out))
	return out
}

// The diagnosis reads the whole window after since, whatever the filters.
func TestDiagnosis(t *testing.T) {
	t.Parallel()
	const url = "http://127.0.0.1:4321/_local/v1/requests?since="
	for _, tc := range []struct {
		name  string
		allow []string
		send  func(t *testing.T, st *store.Store, allow []string)
		query string
		want  []diagnosis
	}{
		{
			name:  "an empty listener",
			send:  func(*testing.T, *store.Store, []string) {},
			query: "event=Nope",
			want: []diagnosis{{
				Code: "nothing_received", Count: 0,
				Message: "The listener has received no event request since it started. Point the SDK at this URL, or send the check request.",
				Next:    `curl -fsS -u 'dev:' -H 'Content-Type: application/json' -d '{"event":"dev check","userId":"dev"}' 'http://127.0.0.1:4321/v1/track'`,
			}},
		},
		{
			name:  "an allowlist names KEY in the check request",
			allow: []string{"web"},
			send:  func(*testing.T, *store.Store, []string) {},
			want: []diagnosis{{
				Code: "nothing_received", Count: 0,
				Message: "The listener has received no event request since it started. Point the SDK at this URL, or send the check request.",
				Next:    `curl -fsS -u 'KEY:' -H 'Content-Type: application/json' -d '{"event":"dev check","userId":"dev"}' 'http://127.0.0.1:4321/v1/track'`,
			}},
		},
		{
			name: "control requests only are no event request",
			send: func(t *testing.T, st *store.Store, _ []string) {
				require.Equal(t, http.StatusOK, sdk(t, st, http.MethodGet, "/sourceConfig", "dev", ""))
			},
			want: []diagnosis{
				{
					Code: "nothing_received", Count: 0,
					Message: "The listener has received no event request since it started. Point the SDK at this URL, or send the check request.",
					Next:    `curl -fsS -u 'dev:' -H 'Content-Type: application/json' -d '{"event":"dev check","userId":"dev"}' 'http://127.0.0.1:4321/v1/track'`,
				},
				{
					Code: "sdk_loaded_no_events", Count: 1,
					Message: "The SDK loaded its config but sent no event: check the SDK plugins, or the action never ran.",
					Next:    "curl -fsS '" + url + "0&kind=control'",
				},
			},
		},
		{
			name: "nothing after the cursor",
			send: func(t *testing.T, st *store.Store, _ []string) {
				track(t, st, `{"userId":"u","event":"e"}`)
				track(t, st, `{"userId":"u","event":"e"}`)
			},
			query: "since=2&event=Nope",
			want: []diagnosis{{
				Code: "nothing_new", Count: 0,
				Message: "No request after cursor 2; the listener holds 2. The app sent before the cursor, or has not sent yet.",
				Next:    "rudder-cli local event-stream events summary --since 0 --json",
			}},
		},
		{
			name: "a filter that matches nothing",
			send: func(t *testing.T, st *store.Store, _ []string) {
				track(t, st, `{"userId":"u","event":"Order Completed"}`)
			},
			query: "event=order+completed&userId=u",
			want: []diagnosis{
				{
					Code: "no_browser_traffic", Count: 1,
					Message: "If the app also runs a browser SDK, the browser did not reach the listener: no browser request, config request or preflight arrived.",
					Next:    "rudder-cli local event-stream serve --help",
				},
				{
					Code: "all_accepted", Count: 1,
					Message: "Every request that arrived was accepted. This says nothing about events that never came: read byEvent.",
					Next:    "rudder-cli local event-stream events list --since 0 --event 'order completed' --user-id u --json",
				},
				{
					Code: "filtered_empty", Count: 1,
					Message: "Requests arrived, but the filters match none. Names are exact and case-sensitive.",
					Next:    "rudder-cli local event-stream events summary --since 0 --json",
				},
			},
		},
		{
			name: "preflights only",
			send: func(t *testing.T, st *store.Store, _ []string) {
				track(t, st, `{"userId":"u","event":"e"}`)
				r := httptest.NewRequest(http.MethodOptions, "/v1/batch", nil)
				r.Header.Set("Origin", "http://localhost:3000")
				r.Header.Set("Access-Control-Request-Method", "POST")
				r.Header.Set("User-Agent", "Mozilla/5.0")
				ingest.New(st, nil, "test").ServeHTTP(httptest.NewRecorder(), r)
			},
			query: "since=1&type=nope",
			want: []diagnosis{
				{
					Code: "nothing_new", Count: 0,
					Message: "No request after cursor 1; the listener holds 1. The app sent before the cursor, or has not sent yet.",
					Next:    "rudder-cli local event-stream events summary --since 0 --json",
				},
				{
					Code: "preflight_only", Count: 1,
					Message: "The browser sent CORS preflights only, so it never sent the requests that carry events.",
					Next:    "curl -fsS '" + url + "1&kind=control'",
				},
			},
		},
		{
			name:  "the config request refused",
			allow: []string{"web"},
			send: func(t *testing.T, st *store.Store, allow []string) {
				require.Equal(t, http.StatusBadRequest, sdk(t, st, http.MethodGet, "/sourceConfig", "nope", "", allow...))
				require.Equal(t, http.StatusUnauthorized, sdk(t, st, http.MethodPost, "/v1/track", "nope", `{"userId":"u"}`, allow...))
				require.Equal(t, http.StatusOK, sdk(t, st, http.MethodPost, "/v1/track", "web", `{"userId":"u"}`, allow...))
			},
			want: []diagnosis{
				{
					Code: "sdk_config_rejected", Count: 1,
					Message: "The SDK asked for its config and got 401 or 400: its write key is not on the --write-key allowlist.",
					Next:    "curl -fsS '" + url + "0&kind=control&failed=true'",
				},
				{
					Code: "auth_rejected", Count: 1,
					Message: "1 request had a missing write key or one that is not on the --write-key allowlist.",
					Next:    "curl -fsS '" + url + "0&statusCode=401&view=compact'",
				},
			},
		},
		{
			name: "a request with no write key",
			send: func(t *testing.T, st *store.Store, _ []string) {
				require.Equal(t, http.StatusOK, sdk(t, st, http.MethodPost, "/v1/track", "", `{"userId":"u"}`))
				require.Equal(t, http.StatusBadRequest, sdk(t, st, http.MethodPost, "/v1/track", "", `{}`))
			},
			want: []diagnosis{
				{
					Code: "no_browser_traffic", Count: 2,
					Message: "If the app also runs a browser SDK, the browser did not reach the listener: no browser request, config request or preflight arrived.",
					Next:    "rudder-cli local event-stream serve --help",
				},
				{
					Code: "body_rejected", Count: 1,
					Message: "1 request was rejected while the body was read: bad gzip, empty body, invalid JSON, wrong batch shape, no identity, or too large. Read rejection.reason.",
					Next:    "curl -fsS '" + url + "0&failed=true&view=compact'",
				},
				{
					Code: "missing_write_key", Count: 2,
					Message: "2 requests arrived with no write key. The listener accepts them; RudderStack answers 401.",
					Next:    "curl -fsS '" + url + "0&writeKey='",
				},
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			cfg := testConfig("127.0.0.1")
			if len(tc.allow) > 0 {
				cfg.WriteKeys = tc.allow
			}
			st := store.New()
			h := New(st, cfg)
			tc.send(t, st, tc.allow)

			got := summaryOf(t, h, tc.query).Summary.Diagnosis
			require.Equal(t, tc.want, got)
			// A curl next on the query API runs and finds what it names.
			for _, d := range got {
				target, ok := strings.CutPrefix(d.Next, "curl -fsS '"+testURL+"/_local/")
				if !ok {
					continue
				}
				env, _ := requestsOf(t, h, strings.TrimSuffix(strings.TrimPrefix(target, "v1/requests?"), "'"))
				require.Positive(t, env.Total, d.Next)
			}
		})
	}
}

func TestSummaryWait(t *testing.T) {
	t.Parallel()

	t.Run("returns once min accepted events arrive", func(t *testing.T) {
		t.Parallel()
		h, st := newTestHandler("127.0.0.1")
		track(t, st, `{"userId":"u","event":"e"}`)
		answered := make(chan *httptest.ResponseRecorder, 1)
		go func() { answered <- get(h, "/_local/v1/events?view=counts&event=e&min=2&wait=10s", nil) }()
		// Post only once the read waits, so the posts below are what end it.
		require.Eventually(t, func() bool { return h.waiters.Load() == 1 }, 5*time.Second, time.Millisecond)

		// A rejected copy never ends the wait.
		require.Equal(t, http.StatusBadRequest, sdk(t, st, http.MethodPost, "/v1/track", "dev", `{"event":"e"}`))
		track(t, st, `{"userId":"u","event":"other"}`)
		time.Sleep(20 * time.Millisecond)
		require.Len(t, answered, 0, "only the second accepted e ends the wait")
		track(t, st, `{"userId":"u","event":"e"}`)

		w := <-answered
		require.Equal(t, http.StatusOK, w.Code, w.Body.String())
		var got struct {
			Cursor   uint64 `json:"cursor"`
			TimedOut bool   `json:"timedOut"`
			WaitedMs int64  `json:"waitedMs"`
			Summary  struct {
				Events struct{ Total int } `json:"events"`
			} `json:"summary"`
		}
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &got))
		require.False(t, got.TimedOut)
		require.Positive(t, got.WaitedMs)
		require.Equal(t, 2, got.Summary.Events.Total)
		require.Equal(t, uint64(4), got.Cursor)
	})

	t.Run("times out with the counts so far", func(t *testing.T) {
		t.Parallel()
		h, st := newTestHandler("127.0.0.1")
		require.Equal(t, http.StatusBadRequest, sdk(t, st, http.MethodPost, "/v1/track", "dev", `{"event":"e"}`))

		got := summaryOf(t, h, "event=e&wait=50ms")

		require.True(t, got.TimedOut)
		require.GreaterOrEqual(t, got.WaitedMs, int64(50))
		require.Zero(t, got.Summary.Events.Total)
	})

	t.Run("stop answers the waiter 503", func(t *testing.T) {
		t.Parallel()
		h, _ := newTestHandler("127.0.0.1")
		srv := httptest.NewServer(h)
		t.Cleanup(srv.Close)
		ctx, cancel := context.WithCancel(context.Background())
		t.Cleanup(cancel)
		done := make(chan *http.Response)
		go func() {
			req, _ := http.NewRequestWithContext(ctx, http.MethodGet, srv.URL+"/_local/v1/events?view=counts&wait=60s", nil)
			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				close(done)
				return
			}
			done <- resp
		}()
		require.Eventually(t, func() bool { return h.waiters.Load() == 1 }, 5*time.Second, time.Millisecond)

		h.Stop()

		select {
		case resp := <-done:
			require.NotNil(t, resp)
			require.NoError(t, resp.Body.Close())
			require.Equal(t, http.StatusServiceUnavailable, resp.StatusCode)
		case <-time.After(2 * time.Second):
			t.Fatal("the long-poll did not end on Stop")
		}
	})
}

// A fields path through a list applies to each element, so
// fields=properties.products.sku lists every sku.
func TestStreamFieldsThroughAList(t *testing.T) {
	t.Parallel()
	h, st := newTestHandler("127.0.0.1")
	track(t, st, `{"userId":"u","event":"Cart","properties":{"products":[{"sku":"a","n":1},{"n":2},"x",null,3]}}`)

	w := get(h, "/_local/v1/events?fields=properties.products.sku", nil)

	require.Equal(t, `{"properties":{"products":[{"sku":"a"},{},"x",null,3]}}`+"\n", w.Body.String(),
		"an element that is not an object stays as sent")
}
