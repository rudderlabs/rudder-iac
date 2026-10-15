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

const testURL = "http://127.0.0.1:4321"

func testConfig(bind string, allowHosts ...string) Config {
	return Config{
		Identity: Identity{
			APIVersion:     "v1",
			ServerID:       "9f3ac1d2b7e4c601",
			URL:            testURL,
			Port:           4321,
			Bind:           bind,
			PID:            4242,
			StartedAt:      time.Date(2026, 9, 30, 12, 0, 0, 0, time.UTC),
			WriteKey:       "dev",
			WriteKeyPolicy: "any",
		},
		WriteKeys:  []string{},
		AllowHosts: allowHosts,
		Version:    "1.2.3",
		UI: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_, _ = io.WriteString(w, "page "+r.URL.Path)
		}),
	}
}

func newTestHandler(bind string, allowHosts ...string) (*Handler, *store.Store) {
	st := store.New()
	return New(st, testConfig(bind, allowHosts...)), st
}

func get(h http.Handler, target string, header http.Header) *httptest.ResponseRecorder {
	r := httptest.NewRequest(http.MethodGet, target, nil)
	r.Host = "127.0.0.1:4321"
	for k, v := range header {
		r.Header[k] = v
	}
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	return w
}

type errorBody struct {
	Error struct {
		Status  int     `json:"status"`
		Code    string  `json:"code"`
		Message string  `json:"message"`
		Param   *string `json:"param"`
		Details any     `json:"details"`
		Next    string  `json:"next"`
	} `json:"error"`
}

func decodeError(t *testing.T, w *httptest.ResponseRecorder) errorBody {
	t.Helper()
	var e errorBody
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &e), w.Body.String())
	require.Equal(t, w.Code, e.Error.Status)
	require.NotEmpty(t, e.Error.Next)
	return e
}

// The Host check blocks DNS rebinding: a page on attacker.example that
// resolves to this host is same-origin to the browser, so only the Host
// header tells it apart.
func TestHostCheck(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		bind    string
		allow   []string
		host    string
		allowed bool
	}{
		{bind: "127.0.0.1", host: "127.0.0.1:4321", allowed: true},
		{bind: "127.0.0.1", host: "127.0.0.1", allowed: true},
		{bind: "127.0.0.1", host: "127.9.8.7:1", allowed: true},
		{bind: "127.0.0.1", host: "localhost:4321", allowed: true},
		{bind: "127.0.0.1", host: "LocalHost", allowed: true},
		{bind: "127.0.0.1", host: "[::1]:4321", allowed: true},
		{bind: "127.0.0.1", host: "[::1]", allowed: true},
		{bind: "127.0.0.1", host: "evil.example:4321"},
		{bind: "127.0.0.1", host: "localhost.evil.example"},
		{bind: "127.0.0.1", host: ""},
		{bind: "0.0.0.0", host: "evil.example"},
		{bind: "0.0.0.0", host: "localhost:4321", allowed: true},
		{bind: "0.0.0.0", host: "172.17.0.2:4321"},
		{bind: "0.0.0.0", allow: []string{"dev-listen"}, host: "Dev-Listen:4321", allowed: true},
		{bind: "0.0.0.0", allow: []string{"172.17.0.2"}, host: "172.17.0.2:4321", allowed: true},
		{bind: "192.168.1.5", host: "192.168.1.5:4321", allowed: true},
		{bind: "192.168.1.5", host: "192.168.1.6:4321"},
		{bind: "::", allow: []string{"[FD00::2]"}, host: "[fd00::2]:4321", allowed: true},
	} {
		t.Run(tc.bind+" "+tc.host, func(t *testing.T) {
			t.Parallel()
			h, _ := newTestHandler(tc.bind, tc.allow...)
			r := httptest.NewRequest(http.MethodGet, "/_dev/v1/info", nil)
			r.Host = tc.host
			w := httptest.NewRecorder()

			h.ServeHTTP(w, r)

			if tc.allowed {
				require.Equal(t, http.StatusOK, w.Code, w.Body.String())
				return
			}
			require.Equal(t, http.StatusForbidden, w.Code)
			require.Equal(t, "host_not_allowed", decodeError(t, w).Error.Code)
		})
	}
}

func TestBrowserOriginCheck(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name   string
		path   string
		header http.Header
		code   string
	}{
		{name: "no fetch metadata", path: "/_dev/v1/info"},
		{name: "typed in the address bar", path: "/_dev/v1/info", header: http.Header{"Sec-Fetch-Site": {"none"}}},
		{name: "same origin", path: "/_dev/v1/info", header: http.Header{"Sec-Fetch-Site": {"same-origin"}}},
		{name: "cross site", path: "/_dev/v1/info", header: http.Header{"Sec-Fetch-Site": {"cross-site"}}, code: "browser_origin"},
		{name: "same site", path: "/_dev/v1/info", header: http.Header{"Sec-Fetch-Site": {"same-site"}}, code: "browser_origin"},
		{
			name: "cross-site navigation to the api",
			path: "/_dev/v1/info",
			header: http.Header{
				"Sec-Fetch-Site": {"cross-site"}, "Sec-Fetch-Mode": {"navigate"}, "Sec-Fetch-Dest": {"document"},
			},
			code: "browser_origin",
		},
		{
			name: "cross-site navigation to the page",
			path: "/_dev/ui/",
			header: http.Header{
				"Sec-Fetch-Site": {"cross-site"}, "Sec-Fetch-Mode": {"navigate"}, "Sec-Fetch-Dest": {"document"},
			},
		},
		{name: "same-origin script of the page", path: "/_dev/ui/app.js", header: http.Header{"Sec-Fetch-Site": {"same-origin"}}},
		{
			name: "cross-site script of the page",
			path: "/_dev/ui/app.js",
			header: http.Header{
				"Sec-Fetch-Site": {"cross-site"}, "Sec-Fetch-Mode": {"no-cors"}, "Sec-Fetch-Dest": {"script"},
			},
			code: "browser_origin",
		},
		{
			name:   "cross-site fetch of the page",
			path:   "/_dev/ui/",
			header: http.Header{"Sec-Fetch-Site": {"cross-site"}, "Sec-Fetch-Mode": {"cors"}, "Sec-Fetch-Dest": {"empty"}},
			code:   "browser_origin",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			h, _ := newTestHandler("127.0.0.1")

			w := get(h, tc.path, tc.header)

			if tc.code == "" {
				require.Equal(t, http.StatusOK, w.Code, w.Body.String())
				return
			}
			require.Equal(t, tc.code, decodeError(t, w).Error.Code)
		})
	}
}

func TestErrors(t *testing.T) {
	t.Parallel()
	crossSite := http.Header{"Sec-Fetch-Site": {"cross-site"}}
	for _, tc := range []struct {
		name    string
		method  string
		target  string
		host    string
		header  http.Header
		stopped bool
		status  int
		code    string
		param   string
		next    string
		allow   string
	}{
		{
			name: "host before browser origin", method: "GET", target: "/_dev/v1/info", host: "evil.example", header: crossSite,
			status: 403, code: "host_not_allowed", next: "rudder-cli dev events --json",
		},
		{
			name: "browser origin before shutdown", method: "GET", target: "/_dev/v1/info", header: crossSite, stopped: true,
			status: 403, code: "browser_origin", next: "rudder-cli dev events --json",
		},
		{
			name: "shutdown before route", method: "GET", target: "/_dev/v1/nope", stopped: true,
			status: 503, code: "shutting_down", next: "rudder-cli dev listen --help",
		},
		{
			name: "unknown route", method: "GET", target: "/_dev/v1/nope",
			status: 404, code: "not_found", next: "curl -fsS '" + testURL + "/_dev/v1/requests'",
		},
		{
			name: "removed summary route", method: "GET", target: "/_dev/v1/summary",
			status: 404, code: "not_found", next: "curl -fsS '" + testURL + "/_dev/v1/requests'",
		},
		{
			name: "route before method", method: "POST", target: "/_dev/v1/nope",
			status: 404, code: "not_found", next: "curl -fsS '" + testURL + "/_dev/v1/requests'",
		},
		{
			name: "write method on the page", method: "POST", target: "/_dev/ui/",
			status: 405, code: "method_not_allowed", allow: "GET, HEAD", next: "curl -fsS '" + testURL + "/_dev/v1/'",
		},
		{
			name: "page host check", method: "GET", target: "/_dev/ui/", host: "evil.example",
			status: 403, code: "host_not_allowed", next: "rudder-cli dev events --json",
		},
		{
			name: "page shutdown", method: "GET", target: "/_dev/ui/", stopped: true,
			status: 503, code: "shutting_down", next: "rudder-cli dev listen --help",
		},
		{
			name: "write method", method: "POST", target: "/_dev/v1/info",
			status: 405, code: "method_not_allowed", allow: "GET, HEAD", next: "curl -fsS '" + testURL + "/_dev/v1/'",
		},
		{
			name: "no CORS preflight", method: "OPTIONS", target: "/_dev/v1/info",
			status: 405, code: "method_not_allowed", allow: "GET, HEAD", next: "curl -fsS '" + testURL + "/_dev/v1/'",
		},
		{
			name: "method before parameters", method: "DELETE", target: "/_dev/v1/info?x=1",
			status: 405, code: "method_not_allowed", allow: "GET, HEAD", next: "curl -fsS '" + testURL + "/_dev/v1/'",
		},
		{
			name: "unknown parameter", method: "GET", target: "/_dev/v1/info?serverId=x",
			status: 400, code: "unknown_parameter", param: "serverId", next: "curl -fsS '" + testURL + "/_dev/v1/'",
		},
		{
			name: "unknown parameter on the index", method: "GET", target: "/_dev/v1/?view=counts",
			status: 400, code: "unknown_parameter", param: "view", next: "curl -fsS '" + testURL + "/_dev/v1/'",
		},
		{
			name: "malformed escape in a value", method: "GET", target: "/_dev/v1/info?x=%zz",
			status: 400, code: "invalid_parameter", next: "curl -fsS '" + testURL + "/_dev/v1/'",
		},
		{
			name: "malformed escape in a name", method: "GET", target: "/_dev/v1/info?%zz",
			status: 400, code: "invalid_parameter", next: "curl -fsS '" + testURL + "/_dev/v1/'",
		},
		{
			name: "semicolon separator", method: "GET", target: "/_dev/v1/info?a;b=1",
			status: 400, code: "invalid_parameter", next: "curl -fsS '" + testURL + "/_dev/v1/'",
		},
		{
			name: "malformed escape on the index", method: "GET", target: "/_dev/v1/?evnt=%ZZ",
			status: 400, code: "invalid_parameter", next: "curl -fsS '" + testURL + "/_dev/v1/'",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			h, _ := newTestHandler("127.0.0.1")
			if tc.stopped {
				h.Stop()
			}
			r := httptest.NewRequest(tc.method, tc.target, nil)
			r.Host = "localhost:4321"
			if tc.host != "" {
				r.Host = tc.host
			}
			for k, v := range tc.header {
				r.Header[k] = v
			}
			w := httptest.NewRecorder()

			h.ServeHTTP(w, r)

			require.Equal(t, tc.status, w.Code)
			require.Equal(t, "application/json; charset=utf-8", w.Header().Get("Content-Type"))
			require.Equal(t, tc.allow, w.Header().Get("Allow"))
			e := decodeError(t, w)
			require.Equal(t, tc.code, e.Error.Code)
			require.Equal(t, tc.next, e.Error.Next)
			require.Nil(t, e.Error.Details)
			if tc.param == "" {
				require.Nil(t, e.Error.Param)
			} else {
				require.Equal(t, tc.param, *e.Error.Param)
			}
		})
	}
}

// No /_dev/ answer may be cached or read by another origin's script.
func TestEveryAnswerIsPrivate(t *testing.T) {
	t.Parallel()
	h, _ := newTestHandler("127.0.0.1")
	for _, target := range []string{"/_dev/v1/", "/_dev/v1/info", "/_dev/v1/nope", "/_dev/ui/"} {
		w := get(h, target, http.Header{"Origin": {"http://evil.example"}})

		require.Equal(t, "no-store", w.Header().Get("Cache-Control"), target)
		require.Equal(t, "nosniff", w.Header().Get("X-Content-Type-Options"), target)
		require.Empty(t, w.Header().Get("Access-Control-Allow-Origin"), target)
	}
}

// The CLI tells a listener from any other server that answers 200 by this
// header, so every answer carries it, an error included.
func TestEveryAnswerNamesTheServer(t *testing.T) {
	t.Parallel()
	h, _ := newTestHandler("127.0.0.1")
	for _, target := range []string{
		"/_dev/v1/", "/_dev/v1/info", "/_dev/v1/events", "/_dev/v1/events?view=counts", "/_dev/v1/nope", "/_dev/v1/events?bogus=1",
		"/_dev/v1/requests", "/_dev/v1/requests/1", "/_dev/v1/guide",
	} {
		w := get(h, target, nil)

		require.Equal(t, "9f3ac1d2b7e4c601", w.Header().Get("X-Dev-Server-Id"), target)
	}
}

func TestHeadAnswersLikeGet(t *testing.T) {
	t.Parallel()
	h, _ := newTestHandler("127.0.0.1")
	srv := httptest.NewServer(h)
	t.Cleanup(srv.Close)

	for _, path := range []string{"/_dev/v1/", "/_dev/v1/info", "/_dev/v1/requests", "/_dev/v1/guide"} {
		resp, err := http.Head(srv.URL + path)
		require.NoError(t, err)
		body, err := io.ReadAll(resp.Body)
		require.NoError(t, err)
		require.NoError(t, resp.Body.Close())

		require.Equal(t, http.StatusOK, resp.StatusCode, path)
		require.NotEmpty(t, resp.Header.Get("Content-Type"))
		require.Empty(t, body)
	}
}

// The page path without its slash opens the page with the same view.
func TestPagePathWithoutSlashRedirects(t *testing.T) {
	t.Parallel()
	h, _ := newTestHandler("127.0.0.1")
	for target, location := range map[string]string{
		"/_dev/ui":                    "/_dev/ui/",
		"/_dev/ui?event=Order*&tab=x": "/_dev/ui/?event=Order*&tab=x",
	} {
		w := get(h, target, nil)

		require.Equal(t, http.StatusFound, w.Code, target)
		require.Equal(t, location, w.Header().Get("Location"), target)
	}
	require.Equal(t, "page /_dev/ui/app.js", get(h, "/_dev/ui/app.js", nil).Body.String())
}

func TestInfo(t *testing.T) {
	t.Parallel()
	h, st := newTestHandler("127.0.0.1")

	w := get(h, "/_dev/v1/info", nil)
	require.Equal(t, http.StatusOK, w.Code)
	require.Equal(t, `{"ready":true,"apiVersion":"v1","serverId":"9f3ac1d2b7e4c601","url":"http://127.0.0.1:4321",`+
		`"port":4321,"bind":"127.0.0.1","pid":4242,"startedAt":"2026-09-30T12:00:00Z","writeKey":"dev",`+
		`"writeKeyPolicy":"any","writeKeys":[],"recordVersion":1,"cursor":0,"exposed":false,"version":"1.2.3",`+
		`"ui":"http://127.0.0.1:4321/_dev/ui/","store":{"requests":0,"events":0,"control":0,"bytes":0,"evicted":0,`+
		`"evictedThrough":0,"maxRequests":10000,"maxBytes":67108864}}`+"\n", w.Body.String())

	st.Capture(&ingest.Capture{
		Kind:    "ingestion",
		Request: httptest.NewRequest(http.MethodPost, "/v1/track", nil),
		Events:  []ingest.Event{{Message: []byte(`{"userId":"u1"}`)}},
	})
	var got struct {
		Cursor uint64      `json:"cursor"`
		Store  store.Stats `json:"store"`
	}
	require.NoError(t, json.Unmarshal(get(h, "/_dev/v1/info", nil).Body.Bytes(), &got))
	require.Equal(t, uint64(1), got.Cursor)
	require.Equal(t, st.Stats(), got.Store)
	require.Equal(t, 1, got.Store.Requests)
}

func TestInfoOnAnExposedBind(t *testing.T) {
	t.Parallel()
	cfg := testConfig("0.0.0.0")
	cfg.WriteKeys = []string{"web", "fake...ests"}
	cfg.Identity.WriteKey, cfg.Identity.WriteKeyPolicy = "web", "allowlist"
	h := New(store.New(), cfg)

	var got map[string]any
	require.NoError(t, json.Unmarshal(get(h, "/_dev/v1/info", nil).Body.Bytes(), &got))

	require.Equal(t, true, got["exposed"])
	require.Equal(t, "allowlist", got["writeKeyPolicy"])
	require.Equal(t, []any{"web", "fake...ests"}, got["writeKeys"])
}

func TestIndexListsEveryRoute(t *testing.T) {
	t.Parallel()
	h, st := newTestHandler("127.0.0.1")
	track(t, st, `{"userId":"u","event":"e"}`)

	for _, target := range []string{"/_dev/v1/", "/_dev/v1"} {
		w := get(h, target, nil)
		require.Equal(t, http.StatusOK, w.Code, target)

		var idx struct {
			APIVersion string            `json:"apiVersion"`
			ServerID   string            `json:"serverId"`
			Next       string            `json:"next"`
			Curl       string            `json:"curl"`
			Links      map[string]string `json:"links"`
			Endpoints  []struct {
				Method  string   `json:"method"`
				Path    string   `json:"path"`
				Params  []string `json:"params"`
				Example string   `json:"example"`
			} `json:"endpoints"`
			Help string `json:"help"`
		}
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &idx))
		require.Equal(t, "v1", idx.APIVersion)
		require.Equal(t, "9f3ac1d2b7e4c601", idx.ServerID)
		require.Equal(t, "rudder-cli dev --help", idx.Help)
		require.Equal(t, "curl -fsS '"+testURL+"/_dev/v1/info'", idx.Next)
		require.Equal(t, idx.Next, idx.Curl)
		require.Equal(t, map[string]string{
			"events": "events", "counts": "events?view=counts", "requests": "requests", "request": "requests/{seq}",
			"info": "info", "guide": "guide", "ui": "/_dev/ui/",
		}, idx.Links)

		var paths []string
		for _, ep := range idx.Endpoints {
			require.Equal(t, http.MethodGet, ep.Method)
			require.NotNil(t, ep.Params)
			require.True(t, strings.HasPrefix(ep.Example, "curl -fsS '"+testURL), ep.Example)
			paths = append(paths, ep.Path)
		}
		for path := range h.routes {
			if path == strings.TrimSuffix(base, "/") {
				continue
			}
			require.Contains(t, paths, path)
		}
		require.Contains(t, paths, base+"requests/{seq}")
		// The index serves the curl user, so each example runs.
		for _, ep := range idx.Endpoints {
			target := strings.TrimSuffix(strings.TrimPrefix(ep.Example, "curl -fsS '"+testURL), "'")
			require.Equal(t, http.StatusOK, get(h, target, nil).Code, ep.Example)
		}
	}
}

// The first call of a new user hits an empty store, so each example runs there too.
func TestIndexExamplesRunOnAnEmptyStore(t *testing.T) {
	t.Parallel()
	h, _ := newTestHandler("127.0.0.1")

	var idx struct {
		Endpoints []struct {
			Example string `json:"example"`
		} `json:"endpoints"`
	}
	require.NoError(t, json.Unmarshal(get(h, "/_dev/v1/", nil).Body.Bytes(), &idx))
	require.NotEmpty(t, idx.Endpoints)
	for _, ep := range idx.Endpoints {
		target := strings.TrimSuffix(strings.TrimPrefix(ep.Example, "curl -fsS '"+testURL), "'")
		require.Equal(t, http.StatusOK, get(h, target, nil).Code, ep.Example)
	}
}

func TestWaitReturnsOnceEnoughRecordsArrive(t *testing.T) {
	t.Parallel()
	h, st := newTestHandler("127.0.0.1")
	go st.Capture(&ingest.Capture{Kind: "ingestion", Request: httptest.NewRequest(http.MethodPost, "/v1/track", nil)})

	found, err := h.wait(context.Background(), 0, 1, func(*store.Record) int { return 1 })

	require.NoError(t, err)
	require.Equal(t, 1, found)
}

// Shutdown must not wait for a long-poll to reach its deadline.
func TestStopWakesEveryWaiter(t *testing.T) {
	t.Parallel()
	h, _ := newTestHandler("127.0.0.1")
	errs := make(chan error)
	for range 3 {
		go func() {
			_, err := h.wait(context.Background(), 0, 1, func(*store.Record) int { return 1 })
			errs <- err
		}()
	}

	// Stop only after every waiter has registered for the wake, so the test
	// takes the wake path, not the stopped-before-start path.
	require.Eventually(t, func() bool { return h.waiters.Load() == 3 }, 5*time.Second, time.Millisecond)
	h.Stop()

	for range 3 {
		select {
		case err := <-errs:
			require.ErrorIs(t, err, ErrShuttingDown)
		case <-time.After(time.Second):
			t.Fatal("a waiter did not wake on Stop")
		}
	}
}

func TestWaitAfterStopReturnsAtOnce(t *testing.T) {
	t.Parallel()
	h, _ := newTestHandler("127.0.0.1")
	h.Stop()

	_, err := h.wait(context.Background(), 0, 1, func(*store.Record) int { return 1 })

	require.ErrorIs(t, err, ErrShuttingDown)
}
