package api

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"regexp"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"

	"github.com/rudderlabs/rudder-iac/cli/internal/devlisten/ingest"
	"github.com/rudderlabs/rudder-iac/cli/internal/devlisten/store"
)

const longKey = "2fakeWriteKeyForTests0000wxyz"

// seedRequests stores four requests: an accepted batch (seq 1), a track
// rejected for its identity (seq 2), a config request (seq 3) and a track
// with a long key and its own messageId (seq 4).
func seedRequests(t *testing.T, st *store.Store) {
	t.Helper()
	require.Equal(t, http.StatusOK, sdk(t, st, http.MethodPost, "/v1/batch", "dev",
		`{"batch":[{"type":"identify","userId":"u1"},{"type":"track","userId":"u1","event":"Order Completed","messageId":"m-1"}]}`))
	require.Equal(t, http.StatusBadRequest, sdk(t, st, http.MethodPost, "/v1/track", "dev", `{"event":"No Identity"}`))
	require.Equal(t, http.StatusOK, sdk(t, st, http.MethodGet, "/sourceConfig", "dev", ""))
	r := httptest.NewRequest(http.MethodPost, "/v1/track", strings.NewReader(`{"userId":"u2","event":"Signed Up","messageId":"m-2"}`))
	r.SetBasicAuth(longKey, "")
	r.Header.Set("Content-Type", "application/json")
	r.Header.Set("User-Agent", "analytics-node/3.0.9")
	r.Header.Set("Cookie", "session=secret")
	w := httptest.NewRecorder()
	ingest.New(st, nil, "test").ServeHTTP(w, r)
	require.Equal(t, http.StatusOK, w.Code)
}

type envelope struct {
	Since          uint64 `json:"since"`
	Cursor         uint64 `json:"cursor"`
	EvictedThrough uint64 `json:"evictedThrough"`
	HasMore        bool   `json:"hasMore"`
	Total          int    `json:"total"`
	Returned       int    `json:"returned"`
	View           string `json:"view"`
	Omitted        *struct{ Fields []string }
	Truncated      map[string]any    `json:"truncated"`
	Links          map[string]any    `json:"links"`
	Requests       []json.RawMessage `json:"requests"`
}

func requestsOf(t *testing.T, h http.Handler, query string) (envelope, *httptest.ResponseRecorder) {
	t.Helper()
	w := get(h, "/_local/v1/requests?"+query, nil)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	require.Equal(t, "application/json; charset=utf-8", w.Header().Get("Content-Type"))
	var env envelope
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &env))
	require.Len(t, env.Requests, env.Returned)
	return env, w
}

func seqs(env envelope) []uint64 {
	out := []uint64{}
	for _, item := range env.Requests {
		out = append(out, gjson.GetBytes(item, "seq").Uint())
	}
	return out
}

func keys(raw string) []string {
	var out []string
	gjson.Parse(raw).ForEach(func(key, _ gjson.Result) bool {
		out = append(out, key.Str)
		return true
	})
	return out
}

func TestRequestsListsEachRequest(t *testing.T) {
	t.Parallel()
	h, st := newTestHandler("127.0.0.1")
	seedRequests(t, st)

	env, w := requestsOf(t, h, "")

	require.Equal(t, []string{
		"apiVersion", "serverId", "since", "cursor", "evictedThrough", "hasMore", "total", "returned", "view",
		"omitted", "truncated", "links", "requests",
	}, keys(w.Body.String()))
	require.Equal(t, []uint64{1, 2, 4}, seqs(env), "control requests stay out by default")
	require.Equal(t, uint64(4), env.Cursor)
	require.False(t, env.HasMore)
	require.Equal(t, 3, env.Total)
	require.Equal(t, "list", env.View)
	require.Equal(t, []string{"request.body"}, env.Omitted.Fields)
	require.Nil(t, env.Truncated)
	require.Equal(t, map[string]any{"next": nil}, env.Links)
	require.JSONEq(t, `{"seq":2,"kind":"ingestion","method":"POST","route":"/v1/track","statusCode":400,"outcome":"rejected","eventCount":1}`,
		string(env.Requests[1]))
	require.Equal(t, []string{"seq", "kind", "method", "route", "statusCode", "outcome", "eventCount"}, keys(string(env.Requests[0])))
}

func TestRequestsFilters(t *testing.T) {
	t.Parallel()
	h, st := newTestHandler("127.0.0.1")
	seedRequests(t, st)

	for query, want := range map[string][]uint64{
		"kind=all":                         {1, 2, 3, 4},
		"kind=control":                     {3},
		"failed=true":                      {2},
		"failed=false":                     {1, 4},
		"kind=all&failed=false":            {1, 3, 4},
		"statusCode=4xx":                   {2},
		"statusCode=200":                   {1, 4},
		"statusCode=400&statusCode=200":    {1, 2, 4},
		"statusCode=5xx":                   {},
		"writeKey=dev":                     {1, 2},
		"writeKey=" + longKey:              {4},
		"writeKey=":                        {},
		"writeKey=dev&writeKey=" + longKey: {1, 2, 4},
		"messageId=m-1":                    {1},
		"messageId=m-2&writeKey=dev":       {},
		"since=1":                          {2, 4},
		"since=1&failed=false":             {4},
	} {
		env, _ := requestsOf(t, h, query)
		require.Equal(t, want, seqs(env), query)
		require.Equal(t, len(want), env.Total, query)
	}
}

func TestRequestsCompactView(t *testing.T) {
	t.Parallel()
	h, st := newTestHandler("127.0.0.1")
	seedRequests(t, st)

	env, _ := requestsOf(t, h, "view=compact&failed=true")

	require.Equal(t, "compact", env.View)
	require.Len(t, env.Requests, 1)
	item := string(env.Requests[0])
	require.Equal(t, []string{"seq", "receivedAt", "method", "route", "statusCode", "outcome", "kind", "rejection", "events"}, keys(item))
	require.JSONEq(t, `{"stage":"identity","reason":"request neither has anonymousId nor userId","idx":0}`,
		gjson.Get(item, "rejection").Raw)
	require.JSONEq(t, `[{"idx":0,"messageId":null,"type":null,"event":"No Identity"}]`, gjson.Get(item, "events").Raw)

	env, _ = requestsOf(t, h, "view=compact&messageId=m-1")
	require.JSONEq(t, `[{"idx":0,"messageId":null,"type":"identify","event":null},{"idx":1,"messageId":"m-1","type":"track","event":"Order Completed"}]`,
		gjson.GetBytes(env.Requests[0], "events").Raw)
}

func TestRequestRecord(t *testing.T) {
	t.Parallel()
	h, st := newTestHandler("127.0.0.1")
	seedRequests(t, st)
	sum := sha256.Sum256([]byte(longKey))

	w := get(h, "/_local/v1/requests/4", nil)

	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	rec := w.Body.String()
	require.Equal(t, []string{
		"recordVersion", "serverId", "seq", "kind", "receivedAt", "route", "transport", "statusCode", "failed", "outcome",
		"writeKey", "writeKeySha256", "sourceId", "rejection", "hint", "request", "response", "events",
	}, keys(rec))
	for path, want := range map[string]any{
		"recordVersion":            float64(1),
		"serverId":                 "9f3ac1d2b7e4c601",
		"seq":                      float64(4),
		"kind":                     "ingestion",
		"route":                    "/v1/track",
		"transport":                "http",
		"statusCode":               float64(200),
		"failed":                   false,
		"outcome":                  "accepted",
		"writeKey":                 "2fak...wxyz",
		"writeKeySha256":           hex.EncodeToString(sum[:]),
		"sourceId":                 "dev-" + hex.EncodeToString(sum[:6]),
		"request.method":           "POST",
		"request.target":           "/v1/track",
		"request.droppedHeaders":   float64(2),
		"request.body":             `{"userId":"u2","event":"Signed Up","messageId":"m-2"}`,
		"request.bodyComplete":     true,
		"response.statusCode":      float64(200),
		"response.body":            "ok",
		"events.0.idx":             float64(0),
		"events.0.messageId":       "m-2",
		"events.0.event":           "Signed Up",
		"events.0.userId":          "u2",
		"events.0.enrichment.type": "track",
	} {
		require.Equal(t, want, gjson.Get(rec, path).Value(), path)
	}
	for _, path := range []string{"rejection", "hint", "events.0.type", "events.0.anonymousId"} {
		require.Equal(t, gjson.Null, gjson.Get(rec, path).Type, path)
	}
	require.JSONEq(t, `{"Content-Type":["application/json"],"User-Agent":["analytics-node/3.0.9"]}`, gjson.Get(rec, "request.headers").Raw)
	require.NotContains(t, rec, "secret", "a cookie is never kept")
	require.NotContains(t, rec, longKey)
	require.False(t, gjson.Get(rec, "events.0.message").Exists(), "compact leaves the message out: the body holds it")

	full := get(h, "/_local/v1/requests/4?view=full", nil).Body.String()
	require.Equal(t, `{"userId":"u2","event":"Signed Up","messageId":"m-2"}`, gjson.Get(full, "events.0.message").Raw)

	short := get(h, "/_local/v1/requests/1", nil).Body.String()
	shortSum := sha256.Sum256([]byte("dev"))
	require.Equal(t, "dev", gjson.Get(short, "writeKey").Value())
	require.Equal(t, gjson.Null, gjson.Get(short, "writeKeySha256").Type)
	require.Equal(t, "dev-"+hex.EncodeToString(shortSum[:6]), gjson.Get(short, "sourceId").Value())

	rejected := get(h, "/_local/v1/requests/2", nil).Body.String()
	require.Equal(t, true, gjson.Get(rejected, "failed").Value())
	require.Equal(t, gjson.Null, gjson.Get(rejected, "events.0.enrichment").Type, "a rejected request has no enrichment")
}

func TestRequestRecordFields(t *testing.T) {
	t.Parallel()
	h, st := newTestHandler("127.0.0.1")
	seedRequests(t, st)

	for query, want := range map[string]string{
		"fields=request.body":                   `{"seq":4,"request":{"method":"POST","body":"{\"userId\":\"u2\",\"event\":\"Signed Up\",\"messageId\":\"m-2\"}"}}`,
		"fields=statusCode&fields=events.event": `{"seq":4,"request":{"method":"POST"},"statusCode":200,"events":[{"event":"Signed Up"}]}`,
		"fields=events.message.userId":          `{"seq":4,"request":{"method":"POST"},"events":[{"message":{"userId":"u2"}}]}`,
	} {
		w := get(h, "/_local/v1/requests/4?"+query, nil)
		require.Equal(t, http.StatusOK, w.Code, query)
		require.Equal(t, want+"\n", w.Body.String(), query)
	}

	env, _ := requestsOf(t, h, "fields=rejection.reason&failed=true")
	require.Equal(t, "fields", env.View)
	require.Nil(t, env.Omitted)
	require.Equal(t, `{"seq":2,"request":{"method":"POST"},"rejection":{"reason":"request neither has anonymousId nor userId"}}`,
		string(env.Requests[0]))
}

func TestRequestRecordNotFound(t *testing.T) {
	t.Parallel()
	h, st := newTestHandler("127.0.0.1")
	seedRequests(t, st)

	for _, target := range []string{
		"/_local/v1/requests/abc", "/_local/v1/requests/99", "/_local/v1/requests/0", "/_local/v1/requests/1/x",
		"/_local/v1/requests/", "/_local/v1/requests/-1", "/_local/v1/requests/+1",
	} {
		w := get(h, target, nil)
		require.Equal(t, http.StatusNotFound, w.Code, target)
		e := decodeError(t, w)
		require.Equal(t, "not_found", e.Error.Code, target)
		require.Equal(t, "curl -fsS '"+testURL+"/_local/v1/requests'", e.Error.Next)
	}
}

func TestRequestsPages(t *testing.T) {
	t.Parallel()
	h, st := newTestHandler("127.0.0.1")
	for i := range 5 {
		track(t, st, `{"userId":"u","event":"e`+strconv.Itoa(i)+`"}`)
	}

	env, w := requestsOf(t, h, "limit=2")
	require.Equal(t, []uint64{1, 2}, seqs(env))
	require.True(t, env.HasMore)
	require.Equal(t, uint64(2), env.Cursor)
	require.Equal(t, 5, env.Total)
	require.Equal(t, 2, env.Returned)
	require.Equal(t, map[string]any{"next": "requests?limit=2&since=2"}, env.Links)
	require.Equal(t, `</_local/v1/requests?limit=2&since=2>; rel="next"`, w.Header().Get("Link"))

	env, _ = requestsOf(t, h, "limit=2&since=2")
	require.Equal(t, []uint64{3, 4}, seqs(env))
	require.Equal(t, 3, env.Total, "total counts the matches after since")

	env, w = requestsOf(t, h, "limit=2&since=4")
	require.Equal(t, []uint64{5}, seqs(env))
	require.False(t, env.HasMore)
	require.Equal(t, uint64(5), env.Cursor)
	require.Empty(t, w.Header().Get("Link"))

	env, _ = requestsOf(t, h, "limit=0")
	require.Empty(t, env.Requests)
	require.Equal(t, uint64(5), env.Cursor, "limit=0 answers a fresh cursor")
	require.Equal(t, 5, env.Total)
	require.False(t, env.HasMore)
}

func TestRequestsMaxBytes(t *testing.T) {
	t.Parallel()
	h, st := newTestHandler("127.0.0.1")
	big := strings.Repeat("x", 1000)
	for range 5 {
		track(t, st, `{"userId":"u","event":"e","properties":{"pad":"`+big+`"}}`)
	}

	env, w := requestsOf(t, h, "view=full&maxBytes=5000")
	require.LessOrEqual(t, w.Body.Len(), 5000)
	require.NotEmpty(t, env.Requests)
	kept := len(env.Requests)
	require.Less(t, kept, 5)
	require.True(t, env.HasMore)
	require.Equal(t, uint64(kept), env.Cursor, "the cursor is the last kept request")
	require.Equal(t, map[string]any{
		"by": "maxBytes", "kept": float64(kept), "matchedAfter": float64(5 - kept),
		"next": "curl -fsS '" + testURL + "/_local/v1/requests?maxBytes=5000&since=" + strconv.Itoa(kept) + "&view=full'",
	}, env.Truncated)

	env, _ = requestsOf(t, h, "view=full&maxBytes=100")
	require.Empty(t, env.Requests, "the first request alone does not fit")
	require.True(t, env.HasMore)
	require.Equal(t, uint64(1), env.Cursor, "the cursor passes the request that does not fit")
	require.Equal(t, "maxBytes", env.Truncated["by"])
	require.Greater(t, env.Truncated["requestBytes"], float64(1000))
	require.Equal(t, "curl -fsS '"+testURL+"/_local/v1/requests/1?fields=request.body&maxBytes=0'", env.Truncated["next"])

	env, _ = requestsOf(t, h, "view=full&maxBytes=10&since=5")
	require.Empty(t, env.Requests, "an empty page stays whole")
	require.Nil(t, env.Truncated)

	env, _ = requestsOf(t, h, "view=full&maxBytes=0")
	require.Len(t, env.Requests, 5)
	require.Nil(t, env.Truncated)

	env, w = requestsOf(t, h, "view=full")
	require.LessOrEqual(t, w.Body.Len(), 24000, "the default cap")
	require.Len(t, env.Requests, 5)

	w = get(h, "/_local/v1/requests/1?view=full&maxBytes=100", nil)
	require.Equal(t, http.StatusOK, w.Code)
	got := w.Body.String()
	require.Equal(t, []string{"seq", "truncated"}, keys(got))
	require.Equal(t, []string{"by", "requestBytes", "next"}, keys(gjson.Get(got, "truncated").Raw))
	require.Equal(t, "curl -fsS '"+testURL+"/_local/v1/requests/1?fields=request.body&maxBytes=0'", gjson.Get(got, "truncated.next").Str)
}

// A pager that follows links.next must reach the end, also when one record
// alone is larger than maxBytes.
func TestRequestsPagingEndsPastAnOversizeRecord(t *testing.T) {
	t.Parallel()
	h, st := newTestHandler("127.0.0.1")
	track(t, st, `{"userId":"u","event":"big","properties":{"pad":"`+strings.Repeat("x", 13_000)+`"}}`)
	track(t, st, `{"userId":"u","event":"small"}`)

	env, _ := requestsOf(t, h, "view=full")
	require.Empty(t, env.Requests)
	require.Equal(t, "curl -fsS '"+testURL+"/_local/v1/requests/1?fields=request.body&maxBytes=0'", env.Truncated["next"])
	var got []uint64
	for hops := 0; env.HasMore; hops++ {
		require.Less(t, hops, 3, "links.next must advance the cursor")
		next, _ := env.Links["next"].(string)
		require.Equal(t, "requests?since="+strconv.FormatUint(env.Cursor, 10)+"&view=full", next)
		env, _ = requestsOf(t, h, strings.TrimPrefix(next, "requests?"))
		got = append(got, seqs(env)...)
	}
	require.Equal(t, []uint64{2}, got)
	require.Equal(t, uint64(2), env.Cursor)
}

// maxBytes bounds the bytes sent, returned and links.next included.
func TestRequestsMaxBytesBoundsTheWholeAnswer(t *testing.T) {
	t.Parallel()
	h, st := newTestHandler("127.0.0.1")
	for range 3 {
		track(t, st, `{"userId":"u","event":"e"}`)
	}

	for _, query := range []string{"", "limit=1&", "limit=2&view=full&"} {
		for maxBytes := 200; maxBytes <= 3000; maxBytes += 7 {
			env, w := requestsOf(t, h, query+"maxBytes="+strconv.Itoa(maxBytes))
			if len(env.Requests) > 0 {
				require.LessOrEqual(t, w.Body.Len(), maxBytes, "%smaxBytes=%d", query, maxBytes)
			}
		}
	}
}

// A record that ingestion accepts always renders, however deep its JSON.
func TestRequestsRenderADeepEvent(t *testing.T) {
	t.Parallel()
	// The record adds three levels and the envelope two, so the depths near
	// the encoder limit of 10,000 each fail in another place.
	for depth := 9_993; depth <= 9_998; depth++ {
		t.Run(strconv.Itoa(depth), func(t *testing.T) {
			t.Parallel()
			h, st := newTestHandler("127.0.0.1")
			track(t, st, `{"userId":"u","event":"e","p":`+strings.Repeat("[", depth)+"0"+strings.Repeat("]", depth)+`}`)

			for _, target := range []string{
				"/_local/v1/requests/1?view=full&maxBytes=0",
				"/_local/v1/requests?view=full&maxBytes=0",
				"/_local/v1/requests?fields=events&maxBytes=0",
			} {
				w := get(h, target, nil)
				require.Equal(t, http.StatusOK, w.Code, target)
				require.True(t, json.Valid(w.Body.Bytes()), target)
				require.Contains(t, w.Body.String(), `"seq":1`, target)
			}
		})
	}
}

// An identity key nested near the decoder limit renders as null, and the
// record names it, so every view still answers a body a client can decode.
func TestRequestsRenderADeepIdentityKey(t *testing.T) {
	t.Parallel()
	for depth := 9_995; depth <= 9_999; depth++ {
		t.Run(strconv.Itoa(depth), func(t *testing.T) {
			t.Parallel()
			h, st := newTestHandler("127.0.0.1")
			track(t, st, `{"type":"track","userId":"u","messageId":"deep","event":`+
				strings.Repeat("[", depth)+strings.Repeat("]", depth)+`}`)

			for _, target := range []string{
				"/_local/v1/requests?view=list",
				"/_local/v1/requests?view=compact",
				"/_local/v1/requests?view=full&maxBytes=0",
				"/_local/v1/requests?fields=events&maxBytes=0",
				"/_local/v1/requests/1",
				"/_local/v1/requests/1?view=full&maxBytes=0",
				"/_local/v1/requests/1?fields=events.event&maxBytes=0",
			} {
				w := get(h, target, nil)
				require.Equal(t, http.StatusOK, w.Code, target)
				var decoded map[string]any
				require.NoError(t, json.Unmarshal(w.Body.Bytes(), &decoded), target)
				require.Contains(t, w.Body.String(), `"seq":1`, target)
			}

			w := get(h, "/_local/v1/requests/1?view=full&maxBytes=0", nil)
			event := gjson.GetBytes(w.Body.Bytes(), "events.0.event")
			if depth <= 9_995 {
				require.True(t, event.IsArray())
				require.False(t, gjson.GetBytes(w.Body.Bytes(), "omitted").Exists())
				return
			}
			require.Equal(t, gjson.Null, event.Type)
			require.Equal(t, `{"fields":["events.0.event"]}`, gjson.GetBytes(w.Body.Bytes(), "omitted").Raw)
			require.Equal(t, `"deep"`, gjson.GetBytes(w.Body.Bytes(), "events.0.messageId").Raw)
		})
	}
}

// writeJSON never answers 200 with a body it could not encode.
func TestWriteJSONAnswers500WhenEncodingFails(t *testing.T) {
	t.Parallel()
	w := httptest.NewRecorder()
	writeJSON(w, http.StatusOK, json.RawMessage(`{"a":`))
	require.Equal(t, http.StatusInternalServerError, w.Code)
	require.Equal(t, "encode_failed", gjson.Get(w.Body.String(), "error.code").String())
}

func TestRequestsParameterErrors(t *testing.T) {
	t.Parallel()
	h, st := newTestHandler("127.0.0.1")
	track(t, st, `{"userId":"u","event":"e"}`)
	index := "curl -fsS '" + testURL + "/_local/v1/'"
	for _, tc := range []struct {
		target string
		code   string
		param  string
	}{
		{target: "requests?bogus=1", code: "unknown_parameter", param: "bogus"},
		{target: "requests?wait=1s", code: "unknown_parameter", param: "wait"},
		{target: "requests?event=x", code: "unknown_parameter", param: "event"},
		{target: "requests?kind=nope", code: "invalid_parameter", param: "kind"},
		{target: "requests?statusCode=600", code: "invalid_parameter", param: "statusCode"},
		{target: "requests?statusCode=6xx", code: "invalid_parameter", param: "statusCode"},
		{target: "requests?statusCode=ok", code: "invalid_parameter", param: "statusCode"},
		{target: "requests?failed=maybe", code: "invalid_parameter", param: "failed"},
		{target: "requests?limit=1001", code: "invalid_parameter", param: "limit"},
		{target: "requests?limit=-1", code: "invalid_parameter", param: "limit"},
		{target: "requests?maxBytes=-1", code: "invalid_parameter", param: "maxBytes"},
		{target: "requests?view=counts", code: "invalid_parameter", param: "view"},
		{target: "requests?messageId=a&messageId=b", code: "invalid_parameter", param: "messageId"},
		{target: "requests?since=soon", code: "invalid_parameter", param: "since"},
		{target: "requests?view=full&fields=request.body", code: "invalid_parameter", param: "fields"},
		{target: "requests?fields=request,response", code: "invalid_parameter", param: "fields"},
		{target: "requests?fields=nope.x", code: "invalid_parameter", param: "fields"},
		{target: "requests?x=%zz", code: "invalid_parameter"},
		{target: "requests/1?since=1", code: "unknown_parameter", param: "since"},
		{target: "requests/1?view=list", code: "invalid_parameter", param: "view"},
	} {
		w := get(h, "/_local/v1/"+tc.target, nil)

		require.Equal(t, http.StatusBadRequest, w.Code, tc.target)
		e := decodeError(t, w)
		require.Equal(t, tc.code, e.Error.Code, tc.target)
		require.Equal(t, index, e.Error.Next, tc.target)
		if tc.param == "" {
			require.Nil(t, e.Error.Param, tc.target)
			continue
		}
		require.Equal(t, tc.param, *e.Error.Param, tc.target)
	}

	e := decodeError(t, get(h, "/_local/v1/requests?fields=nope.x", nil))
	require.Equal(t, map[string]any{"validRoots": []any{
		"recordVersion", "serverId", "seq", "kind", "receivedAt", "route", "transport", "statusCode", "failed", "outcome",
		"writeKey", "writeKeySha256", "sourceId", "rejection", "hint", "request", "response", "events",
	}}, e.Error.Details)
}

func TestRequestsServerChanged(t *testing.T) {
	t.Parallel()
	h, st := newTestHandler("127.0.0.1")
	seedRequests(t, st)

	for _, target := range []string{"/_local/v1/requests?serverId=0", "/_local/v1/requests/1?serverId=0"} {
		w := get(h, target, nil)
		require.Equal(t, http.StatusConflict, w.Code, target)
		require.Equal(t, "server_changed", decodeError(t, w).Error.Code)
	}
	require.Equal(t, http.StatusOK, get(h, "/_local/v1/requests/1?serverId=9f3ac1d2b7e4c601", nil).Code)
}

func TestGuide(t *testing.T) {
	t.Parallel()
	cfg := testConfig("127.0.0.1")
	cfg.Guide = "# The guide\n\nRun `rudder-cli local event-stream serve`.\n"
	h := New(store.New(), cfg)

	w := get(h, "/_local/v1/guide", nil)

	require.Equal(t, http.StatusOK, w.Code)
	require.Equal(t, "text/markdown; charset=utf-8", w.Header().Get("Content-Type"))
	require.Equal(t, cfg.Guide, w.Body.String())
	require.Equal(t, http.StatusBadRequest, get(h, "/_local/v1/guide?x=1", nil).Code)
}

// Each query API URL in the guide answers 200, so a reader can paste it.
func TestGuideURLsAnswer(t *testing.T) {
	t.Parallel()
	guide, err := os.ReadFile("../guide.md")
	require.NoError(t, err)
	h, st := newTestHandler("127.0.0.1")
	seedRequests(t, st)
	placeholder := strings.NewReplacer("$cur", "0", "${cur:-0}", "0", "SEQ", "1", "=N", "=0")

	found := regexp.MustCompile("(?:URL|\\$url|\\$\\{url:-\\})(/_local/v1/[^\"'`\\s]*)").FindAllStringSubmatch(string(guide), -1)
	require.Greater(t, len(found), 5)
	for _, m := range found {
		target := placeholder.Replace(strings.TrimRight(m[1], ".,"))
		require.Equal(t, http.StatusOK, get(h, target, nil).Code, m[0])
	}
}
