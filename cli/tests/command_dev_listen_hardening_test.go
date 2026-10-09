package tests

import (
	"bytes"
	"compress/gzip"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// These tests drive the binary over HTTP the way a hostile page or a broken
// SDK would. They need no network, no account and no secrets, so they run in
// the same job as the other TestDevListen tests.

type devResponse struct {
	status int
	header http.Header
	body   string
}

// tryDo sends one request and returns the error instead of failing the test,
// so goroutines can call it. It never follows redirects and never reuses a
// connection, so one test cannot hold a listener slot for another.
func tryDo(method, url string, body []byte, mutate func(*http.Request)) (devResponse, error) {
	req, err := http.NewRequest(method, url, bytes.NewReader(body))
	if err != nil {
		return devResponse{}, err
	}
	req.Close = true
	if mutate != nil {
		mutate(req)
	}
	client := &http.Client{
		Timeout:       30 * time.Second,
		CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
	}
	resp, err := client.Do(req)
	if err != nil {
		return devResponse{}, err
	}
	defer resp.Body.Close()
	out, err := io.ReadAll(resp.Body)
	if err != nil {
		return devResponse{}, err
	}
	return devResponse{resp.StatusCode, resp.Header, string(out)}, nil
}

func devDo(t *testing.T, method, url string, body []byte, mutate func(*http.Request)) devResponse {
	t.Helper()
	resp, err := tryDo(method, url, body, mutate)
	require.NoError(t, err)
	return resp
}

// postJSON marks a request as JSON from the "dev" source.
func postJSON(r *http.Request) {
	r.Header.Set("Content-Type", "application/json")
	r.SetBasicAuth("dev", "")
}

// Security: a page that rebinds its DNS name to 127.0.0.1 must not read captures.
func TestDevListenRefusesForeignHostsAndCrossSiteReads(t *testing.T) {
	t.Parallel()
	p := startListen(t)
	port := strings.TrimPrefix(p.url(), "http://127.0.0.1:")

	hosts := map[string]int{
		"evil.example":           http.StatusForbidden,
		"evil.example:" + port:   http.StatusForbidden,
		"localhost.evil.example": http.StatusForbidden,
		"127.0.0.1.:" + port:     http.StatusForbidden,
		"2130706433:" + port:     http.StatusForbidden,
		"0x7f.1:" + port:         http.StatusForbidden,
		"127.1:" + port:          http.StatusForbidden,
		"0.0.0.0:" + port:        http.StatusForbidden,
		// '@' is not a valid Host byte, so Go's client sends an empty Host
		// and this row tests an empty Host, not the userinfo form.
		"127.0.0.1@evil.example:80": http.StatusForbidden,
		"127.0.0.1:" + port:         http.StatusOK,
		"localhost:" + port:         http.StatusOK,
		"LOCALHOST:" + port:         http.StatusOK,
		"[::1]:" + port:             http.StatusOK,
	}
	for _, path := range []string{"/_dev/v1/events", "/_dev/v1/requests", "/_dev/v1/info", "/_dev/ui/"} {
		for host, want := range hosts {
			resp := devDo(t, http.MethodGet, p.url()+path, nil, func(r *http.Request) { r.Host = host })
			assert.Equal(t, want, resp.status, "%s Host=%s", path, host)
		}
	}

	for header, want := range map[string]int{
		"cross-site":  http.StatusForbidden,
		"same-site":   http.StatusForbidden,
		"same-origin": http.StatusOK,
		"none":        http.StatusOK,
	} {
		resp := devDo(t, http.MethodGet, p.url()+"/_dev/v1/events", nil, func(r *http.Request) {
			r.Header.Set("Sec-Fetch-Site", header)
		})
		assert.Equal(t, want, resp.status, "Sec-Fetch-Site: %s", header)
	}

	// Ingestion stays open to any page and any Host: that is how a browser SDK
	// on another origin reaches the listener.
	resp := devDo(t, http.MethodPost, p.url()+"/v1/track", []byte(`{"userId":"u"}`), func(r *http.Request) {
		postJSON(r)
		r.Host = "evil.example"
		r.Header.Set("Origin", "https://evil.example")
	})
	assert.Equal(t, http.StatusOK, resp.status)
}

// CORS headers belong to ingestion only. A foreign page must not get
// permission to read /_dev/v1.
func TestDevListenCORSCoversIngestionOnly(t *testing.T) {
	t.Parallel()
	p := startListen(t)

	ingest := devDo(t, http.MethodPost, p.url()+"/v1/track", []byte(`{"userId":"u"}`), func(r *http.Request) {
		postJSON(r)
		r.Header.Set("Origin", "https://app.example")
	})
	assert.Equal(t, "https://app.example", ingest.header.Get("Access-Control-Allow-Origin"))

	for _, path := range []string{"/_dev/v1/info", "/_dev/v1/events", "/_dev/v1/requests", "/_dev/ui/"} {
		resp := devDo(t, http.MethodGet, p.url()+path, nil, func(r *http.Request) {
			r.Header.Set("Origin", "https://evil.example")
		})
		assert.Empty(t, resp.header.Get("Access-Control-Allow-Origin"), path)
	}

	preflight := devDo(t, http.MethodOptions, p.url()+"/_dev/v1/events", nil, func(r *http.Request) {
		r.Header.Set("Origin", "https://evil.example")
		r.Header.Set("Access-Control-Request-Method", "GET")
	})
	assert.Equal(t, http.StatusMethodNotAllowed, preflight.status)
	assert.Empty(t, preflight.header.Get("Access-Control-Allow-Origin"))
}

// The review page renders attacker-controlled events, so its headers are the
// last line of defense. The traversal cases guard the embedded file server.
func TestDevListenReviewPageIsLockedDown(t *testing.T) {
	t.Parallel()
	p := startListen(t)

	page := devDo(t, http.MethodGet, p.url()+"/_dev/ui/", nil, nil)
	require.Equal(t, http.StatusOK, page.status)
	csp := page.header.Get("Content-Security-Policy")
	assert.Contains(t, csp, "default-src 'self'")
	assert.NotContains(t, csp, "unsafe-inline")
	assert.NotContains(t, csp, "unsafe-eval")
	assert.Contains(t, csp, "frame-ancestors 'none'")
	assert.Equal(t, "nosniff", page.header.Get("X-Content-Type-Options"))
	assert.Equal(t, "no-store", page.header.Get("Cache-Control"))
	assert.NotContains(t, page.body, "<script>", "the page must not ship inline script")

	for _, path := range []string{
		"/_dev/ui/../../etc/passwd",
		"/_dev/ui/%2e%2e/%2e%2e/etc/passwd",
		"/_dev/ui/..%2f..%2fetc/passwd",
		"/_dev/ui/%2e%2e%2f",
		"/_dev/ui/;/",
	} {
		conn, err := net.Dial("tcp", strings.TrimPrefix(p.url(), "http://"))
		require.NoError(t, err)
		_, err = fmt.Fprintf(conn, "GET %s HTTP/1.1\r\nHost: 127.0.0.1\r\nConnection: close\r\n\r\n", path)
		require.NoError(t, err)
		raw, _ := io.ReadAll(conn)
		_ = conn.Close()
		first := strings.SplitN(string(raw), "\r\n", 2)[0]
		assert.Contains(t, first, "404", path)
		assert.NotContains(t, string(raw), "root:", path)
	}
}

// The write key is a credential for the user's real source in some setups.
// It must not leave the process in any record the review page or an agent reads.
func TestDevListenMasksTheWriteKey(t *testing.T) {
	t.Parallel()
	p := startListen(t)
	key := "test-" + strings.Repeat("k", 20)
	const body = `{"userId":"k1","event":"masked"}`

	send := func(path string, mutate func(*http.Request)) {
		devDo(t, http.MethodPost, p.url()+path, []byte(body), func(r *http.Request) {
			r.Header.Set("Content-Type", "application/json")
			mutate(r)
		})
	}
	send("/v1/track", func(r *http.Request) { r.SetBasicAuth(key, "") })
	send("/v1/track?writeKey="+key, func(*http.Request) {})
	send("/v1/identify", func(r *http.Request) {
		r.SetBasicAuth(key, "")
		r.Header.Set("AnonymousId", "anon")
		r.Header.Set("Cookie", "session="+key)
	})
	devDo(t, http.MethodGet, p.url()+"/pixel/v1/track?writeKey="+key+"&anonymousId=a&event=px", nil, nil)

	// A stored Authorization header holds the key as base64, not as plain text.
	encoded := base64.StdEncoding.EncodeToString([]byte(key + ":"))
	const fullPath = "/_dev/v1/requests?kind=all&view=full&limit=1000"
	var full string
	for _, path := range []string{fullPath, "/_dev/v1/info", "/_dev/v1/guide"} {
		got := curl(t, p.url()+path)
		if path == fullPath {
			full = got
		}
		assert.NotContains(t, got, key, path)
		assert.NotContains(t, got, encoded, "%s: the key as a Basic auth value", path)
	}
	assert.NotContains(t, full, "session="+key, "cookie headers must not be stored")
	assert.NotContains(t, strings.ToLower(full), "cookie")
	assert.NotContains(t, strings.ToLower(full), "authorization", "credential headers must not be stored")
	assert.Contains(t, full, "test...kkkk", "the masked form stays recognizable")
}

// Bytes the SDK sent are the contract. Numbers, key order, duplicate keys and
// escapes must come back untouched.
func TestDevListenKeepsTheBytesTheSDKSent(t *testing.T) {
	t.Parallel()
	p := startListen(t)
	payloads := []string{
		`{"userId":"u","event":"bytes","properties":{"a":12345678901234567890,"b":0.1000000000000000055511151231257827,"c":1e308,"d":-0}}`,
		`{"event":"bytes","userId":"u","properties":{"z":1,"a":2}}`,
		`{"userId":"u","event":"bytes","properties":{"s":"\u0000é😀 \\ \/"}}`,
	}
	for _, payload := range payloads {
		resp := devDo(t, http.MethodPost, p.url()+"/v1/track", []byte(payload), postJSON)
		require.Equal(t, http.StatusOK, resp.status, payload)
	}

	got := curl(t, p.url()+"/_dev/v1/events?event=bytes")
	lines := strings.Split(strings.TrimSpace(got), "\n")
	assert.Equal(t, payloads, lines)
}

// Limits that protect the developer's machine from any web page that can post
// to the loopback port.
func TestDevListenEnforcesBodyAndEventLimits(t *testing.T) {
	t.Parallel()
	p := startListen(t)
	const bodyCap = 2048000

	padded := func(n int) []byte {
		prefix, suffix := `{"userId":"u","event":"big","properties":{"p":"`, `"}}`
		return []byte(prefix + strings.Repeat("a", n-len(prefix)-len(suffix)) + suffix)
	}
	gz := func(b []byte) []byte {
		var buf bytes.Buffer
		w := gzip.NewWriter(&buf)
		_, _ = w.Write(b)
		require.NoError(t, w.Close())
		return buf.Bytes()
	}
	batch := func(n int) []byte {
		return []byte(`{"batch":[` + strings.TrimSuffix(strings.Repeat(`{"type":"track","event":"e","userId":"u"},`, n), ",") + `]}`)
	}
	gzipped := func(r *http.Request) { postJSON(r); r.Header.Set("Content-Encoding", "gzip") }

	cases := []struct {
		name   string
		path   string
		body   []byte
		mutate func(*http.Request)
		status int
	}{
		{"body at the cap", "/v1/track", padded(bodyCap), postJSON, http.StatusOK},
		{"body over the cap", "/v1/track", padded(bodyCap + 1), postJSON, http.StatusRequestEntityTooLarge},
		{"gzip that expands past the cap", "/v1/track", gz(bytes.Repeat([]byte("0"), 50<<20)), gzipped, http.StatusRequestEntityTooLarge},
		{"gzip of 200k empty objects", "/v1/batch", gz([]byte(`{"batch":[` + strings.TrimSuffix(strings.Repeat("{},", 200000), ",") + `]}`)), gzipped, http.StatusRequestEntityTooLarge},
		{"batch of 10000 events", "/v1/batch", batch(10000), postJSON, http.StatusOK},
		{"batch of 10001 events", "/v1/batch", batch(10001), postJSON, http.StatusRequestEntityTooLarge},
		{"corrupt gzip", "/v1/track", []byte("not gzip"), gzipped, http.StatusBadRequest},
		{"truncated JSON", "/v1/track", []byte(`{"userId":`), postJSON, http.StatusBadRequest},
		{"empty body", "/v1/track", nil, postJSON, http.StatusBadRequest},
		{"no identity", "/v1/track", []byte(`{"event":"e"}`), postJSON, http.StatusBadRequest},
	}
	for _, tc := range cases {
		resp := devDo(t, http.MethodPost, p.url()+tc.path, tc.body, tc.mutate)
		assert.Equal(t, tc.status, resp.status, "%s: %s", tc.name, resp.body)
	}

	resp := devDo(t, http.MethodPost, p.url()+"/v1/track", []byte(`{"userId":"u"}`), func(r *http.Request) {
		postJSON(r)
		r.Header.Set("X-Big", strings.Repeat("a", 600<<10))
	})
	assert.Equal(t, http.StatusRequestHeaderFieldsTooLarge, resp.status)
}

// Nine slow clients must not take the listener down, and the query API must
// stay reachable so an agent can still read what arrived.
func TestDevListenCapsInFlightRequests(t *testing.T) {
	t.Parallel()
	p := startListen(t)
	addr := strings.TrimPrefix(p.url(), "http://")

	// All nine are stalled uploads, so no probe can take a slot. The server
	// refuses the one that finds the eight slots full.
	var stalled []net.Conn
	t.Cleanup(func() {
		for _, c := range stalled {
			_ = c.Close()
		}
	})
	for range 9 {
		conn, err := net.Dial("tcp", addr)
		require.NoError(t, err)
		_, err = io.WriteString(conn, "POST /v1/track HTTP/1.1\r\nHost: 127.0.0.1\r\nAuthorization: Basic ZGV2Og==\r\nContent-Type: application/json\r\nContent-Length: 100\r\n\r\n{\"userId\"")
		require.NoError(t, err)
		stalled = append(stalled, conn)
	}

	// The refused handler answers at once while the others wait for their
	// body, so poll until one connection carries a 503.
	require.Eventually(t, func() bool {
		buf := make([]byte, 64)
		for _, c := range stalled {
			if err := c.SetReadDeadline(time.Now().Add(20 * time.Millisecond)); err != nil {
				return false
			}
			n, _ := c.Read(buf)
			if strings.HasPrefix(string(buf[:n]), "HTTP/1.1 503") {
				return true
			}
		}
		return false
	}, 8*time.Second, 50*time.Millisecond, "one of nine stalled uploads should be refused")
	assert.Equal(t, http.StatusOK, devDo(t, http.MethodGet, p.url()+"/_dev/v1/info", nil, nil).status,
		"the query API must not share the ingestion cap")
	assert.Equal(t, http.StatusOK, devDo(t, http.MethodGet, p.url()+"/_dev/ui/", nil, nil).status)

	// The body must arrive within 10 s, so the slots come back on their own.
	require.Eventually(t, func() bool {
		resp, err := tryDo(http.MethodPost, p.url()+"/v1/track", []byte(`{"userId":"x"}`), postJSON)
		return err == nil && resp.status == http.StatusOK
	}, 20*time.Second, 500*time.Millisecond)
}

// A header that never ends must not pin a connection.
func TestDevListenDropsASlowHeader(t *testing.T) {
	t.Parallel()
	p := startListen(t)
	conn, err := net.Dial("tcp", strings.TrimPrefix(p.url(), "http://"))
	require.NoError(t, err)
	t.Cleanup(func() { _ = conn.Close() })
	_, err = io.WriteString(conn, "POST /v1/track HTTP/1.1\r\nHost: 127.0.0.1\r\n")
	require.NoError(t, err)

	require.NoError(t, conn.SetReadDeadline(time.Now().Add(20*time.Second)))
	start := time.Now()
	_, err = io.ReadAll(conn)
	require.NoError(t, err, "the server should close the connection, not let it time out here")
	assert.Less(t, time.Since(start), 15*time.Second)
}

// Many writers at once. Every request lands once and cursors only grow.
func TestDevListenCapturesParallelWritersExactly(t *testing.T) {
	t.Parallel()
	p := startListen(t)
	const writers, each = 6, 25 // stays under the 8 in-flight cap

	var wg sync.WaitGroup
	for w := range writers {
		wg.Add(1)
		go func(w int) {
			defer wg.Done()
			for i := range each {
				payload := fmt.Sprintf(`{"userId":"w%d","event":"parallel","properties":{"i":%d}}`, w, i)
				resp, err := tryDo(http.MethodPost, p.url()+"/v1/track", []byte(payload), postJSON)
				if !assert.NoError(t, err) {
					return
				}
				assert.Equal(t, http.StatusOK, resp.status)
			}
		}(w)
	}
	wg.Wait()

	body := curl(t, p.url()+"/_dev/v1/requests?limit=1000&view=compact")
	var list struct {
		Total    int `json:"total"`
		Requests []struct {
			Seq int `json:"seq"`
		} `json:"requests"`
	}
	require.NoError(t, json.Unmarshal([]byte(body), &list))
	assert.Equal(t, writers*each, list.Total)
	for i := 1; i < len(list.Requests); i++ {
		require.Greater(t, list.Requests[i].Seq, list.Requests[i-1].Seq)
	}
	events := curl(t, p.url()+"/_dev/v1/events?event=parallel&limit=1000")
	assert.Equal(t, writers*each, len(strings.Split(strings.TrimSpace(events), "\n")))
}

// The store is bounded. Past 10,000 requests the oldest go, and the listener
// says so instead of returning a silently short list.
func TestDevListenEvictsOldestRequestsAndSaysSo(t *testing.T) {
	t.Parallel()
	p := startListen(t)
	const total = 10050

	// One request at a time stays far below the in-flight cap, so none is
	// refused. post drains each body, so the default client reuses one
	// connection and 10,000 posts do not exhaust ports.
	t.Cleanup(http.DefaultClient.CloseIdleConnections)
	for range total {
		require.Equal(t, http.StatusOK, post(t, p.url()+"/v1/track", `{"userId":"u","event":"flood"}`))
	}

	var info struct {
		Cursor int `json:"cursor"`
		Store  struct {
			Requests       int `json:"requests"`
			Evicted        int `json:"evicted"`
			EvictedThrough int `json:"evictedThrough"`
			MaxRequests    int `json:"maxRequests"`
		} `json:"store"`
	}
	require.NoError(t, json.Unmarshal([]byte(curl(t, p.url()+"/_dev/v1/info")), &info))
	assert.Equal(t, total, info.Cursor)
	assert.Equal(t, info.Store.MaxRequests, info.Store.Requests, "a full store holds exactly its cap")
	assert.Equal(t, total-info.Store.Requests, info.Store.Evicted)
	assert.Equal(t, info.Store.Evicted, info.Store.EvictedThrough)

	list := curl(t, p.url()+"/_dev/v1/requests?since=0&limit=1&view=compact")
	assert.Contains(t, list, fmt.Sprintf(`"evictedThrough":%d`, info.Store.EvictedThrough))
}

// Any page open in the developer's browser can post to the listener. A page
// that sends oversized bodies must not push out the captures the developer is
// waiting for. A refused request needs no stored body, so it costs little.
func TestDevListenKeepsCapturesWhenOversizedPostsArrive(t *testing.T) {
	t.Parallel()
	p := startListen(t)
	for _, name := range []string{"first", "second", "third"} {
		resp := devDo(t, http.MethodPost, p.url()+"/v1/track", []byte(`{"userId":"u","event":"`+name+`"}`), postJSON)
		require.Equal(t, http.StatusOK, resp.status)
	}

	oversized := []byte(`{"userId":"` + strings.Repeat("a", 2_100_000) + `"}`)
	for range 40 {
		resp := devDo(t, http.MethodPost, p.url()+"/v1/track", oversized, func(r *http.Request) {
			r.Header.Set("Content-Type", "text/plain") // a cross-origin simple request
			r.Header.Set("Origin", "https://evil.example")
		})
		require.Equal(t, http.StatusRequestEntityTooLarge, resp.status)
	}

	events := devDo(t, http.MethodGet, p.url()+"/_dev/v1/events?since=0", nil, nil)
	assert.Equal(t, 3, len(strings.Split(strings.TrimSpace(events.body), "\n")),
		"40 refused 2 MB posts evicted the earlier captures: %s", events.body)
}
