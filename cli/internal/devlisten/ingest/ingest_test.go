package ingest

import (
	"bufio"
	"bytes"
	"cmp"
	"compress/gzip"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"runtime"
	"runtime/debug"
	"runtime/metrics"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"testing/iotest"
	"time"

	"github.com/stretchr/testify/require"
)

type recordingSink struct {
	mu       sync.Mutex
	captures []*Capture
}

func (s *recordingSink) Capture(c *Capture) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.captures = append(s.captures, c)
}

func (s *recordingSink) only(t *testing.T) *Capture {
	t.Helper()
	s.mu.Lock()
	defer s.mu.Unlock()
	require.Len(t, s.captures, 1)
	return s.captures[0]
}

type discardSink struct{}

func (discardSink) Capture(*Capture) {}

func newTestHandler(writeKeys ...string) (*Handler, *recordingSink) {
	sink := &recordingSink{}
	return New(sink, writeKeys), sink
}

// fillSlots takes every in-flight slot, so the next request gets 503.
func fillSlots(h *Handler) {
	for range MaxInFlight {
		h.slots <- struct{}{}
	}
}

func post(path, body string) *http.Request {
	r := httptest.NewRequest(http.MethodPost, path, strings.NewReader(body))
	r.SetBasicAuth("dev", "")
	return r
}

func serve(h *Handler, r *http.Request) *httptest.ResponseRecorder {
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, r)
	return rec
}

func gz(t testing.TB, s string) []byte {
	t.Helper()
	var buf bytes.Buffer
	zw := gzip.NewWriter(&buf)
	_, err := zw.Write([]byte(s))
	require.NoError(t, err)
	require.NoError(t, zw.Close())
	return buf.Bytes()
}

func TestAcceptedBatchIsCapturedWithDecodedEvents(t *testing.T) {
	t.Parallel()
	h, sink := newTestHandler()
	body := `{"batch":[{"type":"identify","userId":"u1"},{"type":"track","event":"A","anonymousId":"a1","properties":{"n":1}}]}`
	compressed := gz(t, body)
	r := httptest.NewRequest(http.MethodPost, "/v1/batch", bytes.NewReader(compressed))
	r.SetBasicAuth("dev", "")
	r.Header.Set("Content-Encoding", "gzip")

	rec := serve(h, r)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, "ok", rec.Body.String())
	c := sink.only(t)
	require.Equal(t, "ingestion", c.Kind)
	require.Equal(t, "/v1/batch", c.Route)
	require.Equal(t, "http", c.Transport)
	require.Equal(t, "dev", c.WriteKey)
	require.Equal(t, compressed, c.Body)
	require.Equal(t, body, string(c.Decoded))
	require.True(t, c.BodyComplete)
	require.Nil(t, c.Rejection)
	require.Equal(t, []Event{
		{Message: []byte(`{"type":"identify","userId":"u1"}`)},
		{Message: []byte(`{"type":"track","event":"A","anonymousId":"a1","properties":{"n":1}}`)},
	}, c.Events)
	require.Equal(t, rec.Result().Header, c.Header, "the capture holds the headers sent")
}

func TestIdentityFailureNamesTheEvent(t *testing.T) {
	t.Parallel()
	h, sink := newTestHandler()

	rec := serve(h, post("/v1/batch", `{"batch":[{"userId":"u1"},{"userId":" ​ "}]}`))

	require.Equal(t, http.StatusBadRequest, rec.Code)
	c := sink.only(t)
	idx := 1
	require.Equal(t, &Rejection{Stage: "identity", Reason: "request neither has anonymousId nor userId", Idx: &idx}, c.Rejection)
	require.Len(t, c.Events, 2, "a rejected request keeps its parsed events")
}

// RudderStack checks numbers, then identity, one event at a time.
func TestNumberOutOfRangeIsInvalidJSON(t *testing.T) {
	t.Parallel()
	idx := func(i int) *int { return &i }

	for _, tc := range []struct {
		name, path, body string
		status           int
		rejection        *Rejection
	}{
		{
			name: "track", path: "/v1/track", body: `{"userId":"u1","n":1e400}`,
			status: 400, rejection: &Rejection{Stage: "parse", Reason: "invalid json", Idx: idx(0)},
		},
		{
			name: "batch member", path: "/v1/batch", body: `{"batch":[{"userId":"u1"},{"userId":"u2","p":{"n":[-1e400]}}]}`,
			status: 400, rejection: &Rejection{Stage: "parse", Reason: "invalid json", Idx: idx(1)},
		},
		{
			name: "number before identity", path: "/v1/batch", body: `{"batch":[{"n":1e400}]}`,
			status: 400, rejection: &Rejection{Stage: "parse", Reason: "invalid json", Idx: idx(0)},
		},
		{
			name: "identity of an earlier event first", path: "/v1/batch", body: `{"batch":[{"n":1},{"userId":"u1","n":1e400}]}`,
			status: 400, rejection: &Rejection{Stage: "identity", Reason: "request neither has anonymousId nor userId", Idx: idx(0)},
		},
		{
			name: "underflow and strings fit", path: "/v1/track", body: `{"userId":"u1","n":1e-400,"s":"1e400","e":"\"1e400"}`,
			status: 200,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			h, sink := newTestHandler()

			rec := serve(h, post(tc.path, tc.body))

			require.Equal(t, tc.status, rec.Code, rec.Body.String())
			c := sink.only(t)
			require.Equal(t, tc.rejection, c.Rejection)
			require.NotEmpty(t, c.Events, "a rejected request keeps its events")
		})
	}
}

// The Sink gets each event as the bytes the SDK sent.
func TestEventsKeepTheSentBytes(t *testing.T) {
	t.Parallel()
	member := `{ "userId" : "u1", "n" : 1.0, "k":1, "k":2 }`
	track := "{\"userId\":\"u1\",\"n\":1.0,\"k\":\"a\",\"k\":\"b\",\"s\":\"\xff\"}"
	batch := " {\"batch\": [ " + member + " ,{\"anonymousId\":\"a1\",\"s\":\"\\u00e9\xff\"}\n] } "

	for _, tc := range []struct {
		name   string
		r      *http.Request
		status int
		events []string
	}{
		{
			name:   "track",
			r:      post("/v1/track", track),
			status: 200, events: []string{track},
		},
		{
			name:   "rejected track",
			r:      post("/v1/track", `{"userId":"u1","n":1e400}`),
			status: 400, events: []string{`{"userId":"u1","n":1e400}`},
		},
		{
			name:   "batch members",
			r:      post("/v1/batch", batch),
			status: 200, events: []string{member, "{\"anonymousId\":\"a1\",\"s\":\"\\u00e9\xff\"}"},
		},
		{
			name:   "beacon",
			r:      httptest.NewRequest(http.MethodPost, "/beacon/v1/batch?writeKey=dev", strings.NewReader(`{"batch":[`+member+`]}`)),
			status: 200, events: []string{member},
		},
		{
			name: "gzip",
			r: func() *http.Request {
				r := post("/v1/batch", string(gz(t, batch)))
				r.Header.Set("Content-Encoding", "gzip")
				return r
			}(),
			status: 200, events: []string{member, "{\"anonymousId\":\"a1\",\"s\":\"\\u00e9\xff\"}"},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			h, sink := newTestHandler()

			rec := serve(h, tc.r)

			require.Equal(t, tc.status, rec.Code, rec.Body.String())
			var want []Event
			for _, e := range tc.events {
				want = append(want, Event{Message: []byte(e)})
			}
			require.Equal(t, want, sink.only(t).Events)
		})
	}
}

// Duplicate keys and \u0000 escapes get RudderStack's answer, while
// the Sink still gets the bytes the SDK sent.
func TestSanitizingMatchesRudderStack(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		path, body string
		status     int
		answer     string
	}{
		{"/v1/batch", `{"batch":[{"userId":"u1"}],"batch":[]}`, 200, "ok"},
		{"/v1/batch", `{"batch":[],"batch":[{"userId":"u1"}]}`, 400, "empty batch payload\n"},
		{"/v1/track", `{"userId":"u1","userId":""}`, 400, "request neither has anonymousId nor userId\n"},
		{"/v1/track", `{"userId":"","userId":"u1"}`, 200, "ok"},
		{"/v1/batch", `{"batch":[{"type":"extract","type":"track"}]}`, 400, "request neither has anonymousId nor userId\n"},
		// RudderStack makes \xff a "?", a valid identity.
		{"/v1/track", "{\"userId\":\"\xff\"}", 200, "ok"},
		{"/v1/track", `{"userId":"\\u0000"}`, 400, "invalid json\n"},
		{"/v1/batch", `{"batch":[{"userId":"u1"},{"userId":"\\u0000"}]}`, 400, "invalid json\n"},
		{"/v1/track", `{"type":"ex\u0000tract"}`, 200, "ok"},
		{"/v1/track", `{"user\u0000Id":"u1"}`, 200, "ok"},
		{"/v1/track", `{"userId":"\u0000"}`, 400, "request neither has anonymousId nor userId\n"},
		{"/v1/track", `{"userId":"u\u0000","n":1e400}`, 400, "invalid json\n"},
	} {
		t.Run(tc.body, func(t *testing.T) {
			t.Parallel()
			h, sink := newTestHandler()

			rec := serve(h, post(tc.path, tc.body))

			require.Equal(t, tc.status, rec.Code)
			require.Equal(t, tc.answer, rec.Body.String())
			c := sink.only(t)
			require.Equal(t, tc.body, string(c.Decoded))
			for _, e := range c.Events {
				require.Contains(t, tc.body, string(e.Message), "each event keeps the sent bytes")
			}
		})
	}
}

// A request past the cap is refused before its body is read, so a burst
// cannot hold more than MaxInFlight bodies.
func TestRequestPastTheCapGets503(t *testing.T) {
	t.Parallel()
	h, sink := newTestHandler()
	fillSlots(h)

	rec := serve(h, post("/v1/track", `{"userId":"u1"}`))

	require.Equal(t, http.StatusServiceUnavailable, rec.Code)
	require.Equal(t, "Service Unavailable\n", rec.Body.String())
	require.Equal(t, "nosniff", rec.Header().Get("X-Content-Type-Options"))
	c := sink.only(t)
	require.Equal(t, &Rejection{Stage: "overload", Reason: "Service Unavailable"}, c.Rejection)
	require.Nil(t, c.Body, "the body is not read")

	<-h.slots
	require.Equal(t, http.StatusOK, serve(h, post("/v1/track", `{"userId":"u1"}`)).Code)
}

// sendStalledBody posts a body shorter than its Content-Length, so the
// server waits for bytes that never come.
func sendStalledBody(t *testing.T, h http.Handler, wait time.Duration) *http.Response {
	t.Helper()
	srv := httptest.NewServer(h)
	t.Cleanup(srv.Close)

	conn, err := net.Dial("tcp", srv.Listener.Addr().String())
	require.NoError(t, err)
	t.Cleanup(func() { _ = conn.Close() })
	require.NoError(t, conn.SetDeadline(time.Now().Add(wait)))
	_, err = fmt.Fprint(conn, "POST /v1/track HTTP/1.1\r\nHost: x\r\nContent-Length: 100\r\n\r\n{\"userId\":\"u1\"}")
	require.NoError(t, err)

	resp, err := http.ReadResponse(bufio.NewReader(conn), nil)
	require.NoError(t, err)
	return resp
}

// A stalled body must not hold the 503 back.
func TestRequestPastTheCapGets503WhileItsBodyStalls(t *testing.T) {
	t.Parallel()
	h, _ := newTestHandler()
	fillSlots(h)

	resp := sendStalledBody(t, h, 2*time.Second)

	require.Equal(t, http.StatusServiceUnavailable, resp.StatusCode)
	require.True(t, resp.Close, "the connection closes instead of reading the body")
}

// Every request frees its slot, accepted or rejected.
func TestRequestsFreeTheirSlots(t *testing.T) {
	t.Parallel()
	h, _ := newTestHandler()

	for i := range 2 * MaxInFlight {
		require.Equal(t, http.StatusBadRequest, serve(h, post("/v1/track", "{")).Code, "rejected request %d", i)
		require.Equal(t, http.StatusOK, serve(h, post("/v1/track", `{"userId":"u1"}`)).Code, "accepted request %d", i)
	}
}

func TestPreflightIsNotCapped(t *testing.T) {
	t.Parallel()
	h, sink := newTestHandler()
	fillSlots(h)
	r := httptest.NewRequest(http.MethodOptions, "/v1/track", nil)
	r.Header.Set("Origin", "https://app.example")
	r.Header.Set("Access-Control-Request-Method", "POST")

	require.Equal(t, http.StatusNoContent, serve(h, r).Code)
	require.Equal(t, "control", sink.only(t).Kind, "a preflight is control")
}

// Any web page can post to the listener, so a burst of large posts must stay
// bounded.
func TestBurstOfLargePostsStaysUnder128MiB(t *testing.T) {
	if testing.Short() {
		t.Skip("sends 970 MB through loopback")
	}
	srv := httptest.NewServer(New(discardSink{}, nil))
	t.Cleanup(srv.Close)
	// Collect often, so the peak follows live memory and not garbage.
	defer debug.SetGCPercent(debug.SetGCPercent(10))

	tiny := `{"userId":"u"},`
	padded := `{"userId":"u","p":"` + strings.Repeat("x", 180) + `"},`
	for _, tc := range []struct {
		name string
		path string
		body []byte
		gzip bool
		want int
	}{
		{name: "one long string", body: []byte(`{"userId":"u1","event":"Big","properties":{"blob":"` + strings.Repeat("x", 2_000_000) + `"}}`)},
		{name: "gzip of many empty objects", body: gz(t, `{"userId":"u1","a":[{}`+strings.Repeat(`,{}`, 680_000)+`]}`), gzip: true},
		{name: "many zeros", body: []byte(`{"userId":"u1","a":[0` + strings.Repeat(`,0`, 1_020_000) + `]}`)},
		{
			name: "batch of tiny events past the event cap", path: "/v1/batch", want: http.StatusRequestEntityTooLarge,
			body: []byte(`{"batch":[` + strings.Repeat(tiny, maxReqSize/len(tiny)-2) + `{"userId":"u"}]}`),
		},
		{
			name: "batch at the event cap", path: "/v1/batch",
			body: []byte(`{"batch":[` + strings.Repeat(padded, maxBatchEvents-1) + strings.TrimSuffix(padded, ",") + `]}`),
		},
	} {
		path, want := cmp.Or(tc.path, "/v1/track"), cmp.Or(tc.want, http.StatusOK)
		require.LessOrEqual(t, len(tc.body), maxReqSize, tc.name)
		statuses, growth := burst(t, 120, func() *http.Request {
			r, err := http.NewRequest(http.MethodPost, srv.URL+path, bytes.NewReader(tc.body))
			if err != nil {
				panic(err)
			}
			if tc.gzip {
				r.Header.Set("Content-Encoding", "gzip")
			}
			return r
		})
		t.Logf("%s: %d bytes sent, statuses %v, peak heap growth %d MiB", tc.name, len(tc.body), statuses, growth>>20)
		require.Positive(t, statuses[want], tc.name)
		require.Equal(t, 120, statuses[want]+statuses[http.StatusServiceUnavailable]+statuses[-1], tc.name)
		require.Less(t, growth, uint64(128<<20), tc.name)
	}
}

// A GET needs no preflight, so any page can send a long query from an img
// tag.
func TestBurstOfLongQueriesStaysUnder128MiB(t *testing.T) {
	if testing.Short() {
		t.Skip("sends 48 MB of request lines through loopback")
	}
	srv := httptest.NewServer(New(discardSink{}, nil))
	t.Cleanup(srv.Close)
	// Collect often, so the peak follows live memory and not garbage.
	defer debug.SetGCPercent(debug.SetGCPercent(10))

	var manyKeys strings.Builder
	manyKeys.WriteString("writeKey=dev")
	for i := 0; manyKeys.Len() < 1_000_000; i++ {
		fmt.Fprintf(&manyKeys, "&%d=", i)
	}

	for _, tc := range []struct{ name, target string }{
		{name: "beacon with many keys", target: "/beacon/v1/batch?" + manyKeys.String()},
		{name: "pixel with many keys", target: "/pixel/v1/track?event=e&" + manyKeys.String()},
		{name: "pixel with a deep key", target: "/pixel/v1/track?writeKey=dev&event=e&anonymousId=a&" + strings.Repeat("a.", 499_000) + "b=1"},
	} {
		statuses, growth := burst(t, 2*MaxInFlight, func() *http.Request {
			r, err := http.NewRequest(http.MethodGet, srv.URL+tc.target, nil)
			if err != nil {
				panic(err)
			}
			return r
		})
		t.Logf("%s: %d bytes of query, statuses %v, peak heap and stack growth %d MiB", tc.name, len(tc.target), statuses, growth>>20)
		require.Zero(t, statuses[-1], tc.name)
		require.Less(t, growth, uint64(128<<20), tc.name)
	}
}

// burst sends n parallel requests and returns the status counts and the
// peak growth of heap and goroutine stacks.
func burst(t *testing.T, n int, newRequest func() *http.Request) (map[int]int, uint64) {
	t.Helper()
	runtime.GC()
	base := heapBytes()

	var (
		peak    atomic.Uint64
		stop    = make(chan struct{})
		sampled = make(chan struct{})
	)
	go func() {
		defer close(sampled)
		for {
			select {
			case <-stop:
				return
			case <-time.After(time.Millisecond):
				peak.Store(max(peak.Load(), heapBytes()))
			}
		}
	}()

	var (
		wg       sync.WaitGroup
		mu       sync.Mutex
		statuses = map[int]int{}
	)
	for range n {
		wg.Go(func() {
			resp, err := http.DefaultClient.Do(newRequest())
			status := -1 // the server closed the connection while the client still sent the body
			if err == nil {
				_, _ = io.Copy(io.Discard, resp.Body)
				_ = resp.Body.Close()
				status = resp.StatusCode
			}
			mu.Lock()
			statuses[status]++
			mu.Unlock()
		})
	}
	wg.Wait()
	close(stop)
	<-sampled
	return statuses, max(peak.Load(), base) - base
}

func heapBytes() uint64 {
	s := []metrics.Sample{{Name: "/memory/classes/heap/objects:bytes"}, {Name: "/memory/classes/heap/stacks:bytes"}}
	metrics.Read(s)
	return s[0].Value.Uint64() + s[1].Value.Uint64()
}

// A request past the cap reads only its write key, so it costs no memory.
func TestRequestPastTheCapDoesNotParseItsQuery(t *testing.T) {
	h := New(discardSink{}, nil)
	fillSlots(h)
	var query strings.Builder
	query.WriteString("writeKey=dev")
	for i := range 60_000 {
		fmt.Fprintf(&query, "&k%d=", i)
	}
	r := httptest.NewRequest(http.MethodPost, "/beacon/v1/batch?"+query.String(), nil)

	allocs := testing.AllocsPerRun(5, func() {
		rec := serve(h, r)
		require.Equal(t, http.StatusServiceUnavailable, rec.Code)
	})

	require.Less(t, allocs, 100.0)
}

// A stalled upload fails at the read deadline as an interrupted read does.
func TestStalledBodyFailsAtTheReadDeadline(t *testing.T) {
	t.Parallel()
	h, sink := newTestHandler()
	h.readTimeout = 100 * time.Millisecond

	resp := sendStalledBody(t, h, 5*time.Second)
	got, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	require.Equal(t, http.StatusBadRequest, resp.StatusCode)
	require.Equal(t, "failed to read body from request\n", string(got))
	c := sink.only(t)
	require.False(t, c.BodyComplete)
	require.Equal(t, `{"userId":"u1"}`, string(c.Body))
	require.Empty(t, c.Events)
	require.Equal(t, "body", c.Rejection.Stage)
}

// A cut upload whose prefix is valid JSON is not accepted.
func TestValidJSONThenReadErrorIsRejected(t *testing.T) {
	t.Parallel()
	h, sink := newTestHandler()
	r := httptest.NewRequest(http.MethodPost, "/v1/track",
		io.MultiReader(strings.NewReader(`{"userId":"u1"}`), iotest.ErrReader(io.ErrUnexpectedEOF)))
	r.ContentLength = 100

	rec := serve(h, r)

	require.Equal(t, http.StatusBadRequest, rec.Code)
	require.Equal(t, "failed to read body from request\n", rec.Body.String())
	c := sink.only(t)
	require.False(t, c.BodyComplete)
	require.Equal(t, `{"userId":"u1"}`, string(c.Body))
	require.Empty(t, c.Events)
}

// BodyComplete is false when gzip decoding fails or hits the limit.
func TestGzipFailureLeavesTheBodyIncomplete(t *testing.T) {
	t.Parallel()

	for name, body := range map[string][]byte{
		"not gzip":           []byte("not gzip"),
		"cut, ISIZE too big": storedGzip("abcd"),
		"cut, ISIZE below":   storedGzip("\x00\x00\x00\x00"),
		"forged small ISIZE": forgedGzip(t),
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			h, sink := newTestHandler()
			r := post("/v1/track", string(body))
			r.Header.Set("Content-Encoding", "gzip")

			rec := serve(h, r)

			require.NotEqual(t, http.StatusOK, rec.Code)
			c := sink.only(t)
			require.False(t, c.BodyComplete)
			require.Equal(t, body, c.Body)
			require.Empty(t, c.Events)
		})
	}
}

// A request refused before auth keeps the write key it carried.
func TestEarlyRejectionKeepsTheWriteKey(t *testing.T) {
	t.Parallel()
	badGzip := func(r *http.Request) *http.Request {
		r.Header.Set("Content-Encoding", "gzip")
		return r
	}

	for name, tc := range map[string]struct {
		r    *http.Request
		full bool
	}{
		"overload":        {r: post("/v1/track", `{"userId":"u1"}`), full: true},
		"bad gzip":        {r: badGzip(post("/v1/track", "not gzip"))},
		"beacon bad gzip": {r: badGzip(httptest.NewRequest(http.MethodPost, "/beacon/v1/batch?writeKey=dev", strings.NewReader("not gzip")))},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			h, sink := newTestHandler("dev")
			if tc.full {
				fillSlots(h)
			}

			rec := serve(h, tc.r)

			require.NotEqual(t, http.StatusOK, rec.Code)
			require.Equal(t, "dev", sink.only(t).WriteKey)
		})
	}
}

func TestPixelBuildsItsEventFromTheQuery(t *testing.T) {
	t.Parallel()
	h, sink := newTestHandler()
	h.now = func() time.Time { return time.Date(2026, 9, 30, 12, 0, 0, 0, time.UTC) }
	r := httptest.NewRequest(http.MethodGet,
		"/pixel/v1/track?writeKey=dev&event=Click&anonymousId=%22a1%22&context.page.path=%2Fhome&a..b=x", nil)

	rec := serve(h, r)

	require.Equal(t, http.StatusOK, rec.Code)
	c := sink.only(t)
	require.Equal(t, "pixel", c.Transport)
	require.Equal(t, "dev", c.WriteKey)
	require.Nil(t, c.Rejection)
	require.Len(t, c.Events, 1)
	require.True(t, c.Events[0].FromQuery, "no JSON was sent")
	require.JSONEq(t, `{
		"anonymousId": "a1",
		"channel": "web",
		"context": {"page": {"path": "/home"}},
		"event": "Click",
		"integrations": {"All": true},
		"originalTimestamp": "2026-09-30T12:00:00Z",
		"sentAt": "2026-09-30T12:00:00Z",
		"type": "track"
	}`, string(c.Events[0].Message))
}

// A pixel answers the GIF whatever happens, so the capture carries the reason.
func TestPixelFailureIsCapturedBehindTheGIF(t *testing.T) {
	t.Parallel()
	h, sink := newTestHandler()

	rec := serve(h, httptest.NewRequest(http.MethodGet, "/pixel/v1/track?writeKey=dev&event=", nil))

	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, pixelGIF, rec.Body.String())
	require.Equal(t, &Rejection{Stage: "parse", Reason: "track: Mandatory field 'event' missing"}, sink.only(t).Rejection)
}

// An unknown path is control, with the transport of its prefix.
func TestUnknownPathIsControl(t *testing.T) {
	t.Parallel()
	for target, transport := range map[string]string{
		"/pixel/not-a-route?writeKey=dev":  "pixel",
		"/beacon/not-a-route?writeKey=dev": "beacon",
		"/v2/track":                        "http",
	} {
		t.Run(target, func(t *testing.T) {
			t.Parallel()
			h, sink := newTestHandler()

			rec := serve(h, httptest.NewRequest(http.MethodGet, target, nil))

			require.Equal(t, http.StatusNotFound, rec.Code)
			c := sink.only(t)
			require.Equal(t, "control", c.Kind)
			require.Equal(t, transport, c.Transport)
		})
	}
}
