package devlisten

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/rudderlabs/rudder-iac/cli/internal/devlisten/api"
)

func start(t *testing.T, cfg Config) *Server {
	t.Helper()
	s, err := Start(cfg)
	require.NoError(t, err)
	t.Cleanup(func() { _ = s.Close(context.Background()) })
	return s
}

func getJSON(t *testing.T, url string, v any) {
	t.Helper()
	req, err := http.NewRequest(http.MethodGet, url, nil)
	require.NoError(t, err)
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()
	require.Equal(t, http.StatusOK, resp.StatusCode)
	require.NoError(t, json.NewDecoder(resp.Body).Decode(v))
}

func TestStartReportsItsIdentity(t *testing.T) {
	t.Parallel()
	s := start(t, Config{Version: "1.2.3"})
	ready := s.Ready()

	require.Regexp(t, regexp.MustCompile(`^[0-9a-f]{16}$`), ready.ServerID)
	require.Positive(t, ready.Port)
	url := "http://127.0.0.1:" + strconv.Itoa(ready.Port)
	require.Equal(t, Ready{
		Ready: true,
		Identity: api.Identity{
			APIVersion:     "v1",
			ServerID:       ready.ServerID,
			URL:            url,
			Port:           ready.Port,
			Bind:           "127.0.0.1",
			PID:            os.Getpid(),
			StartedAt:      ready.StartedAt,
			WriteKey:       "dev",
			WriteKeyPolicy: "any",
		},
		Cursor: 0,
		UI:     url + "/_dev/ui/",
	}, ready)
	require.Equal(t, time.UTC, ready.StartedAt.Location())
	require.Zero(t, ready.StartedAt.Nanosecond())

	var info map[string]any
	getJSON(t, url+"/_dev/v1/info", &info)
	line, err := json.Marshal(ready)
	require.NoError(t, err)
	var readyLine map[string]any
	require.NoError(t, json.Unmarshal(line, &readyLine))
	for key, value := range readyLine {
		require.Equal(t, value, info[key], key)
	}
	require.Equal(t, "1.2.3", info["version"])
	var version map[string]string
	getJSON(t, url+"/version", &version)
	require.Equal(t, map[string]string{"Version": "1.2.3"}, version)
	require.Equal(t, []any{}, info["writeKeys"])
	require.Equal(t, float64(0), info["store"].(map[string]any)["requests"])
}

func TestReadyShowsAReachableURLAndTheMaskedKey(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name       string
		cfg        Config
		wantHost   string
		wantKey    string
		wantPolicy string
		wantKeys   []any
	}{
		{name: "wildcard bind", cfg: Config{Bind: "0.0.0.0"}, wantHost: "127.0.0.1", wantKey: "dev", wantPolicy: "any", wantKeys: []any{}},
		{
			name:     "allowlist",
			cfg:      Config{WriteKeys: []string{"fake-write-key-for-tests", "web"}},
			wantHost: "127.0.0.1", wantKey: "fake...ests", wantPolicy: "allowlist", wantKeys: []any{"fake...ests", "web"},
		},
		{name: "IPv6 loopback", cfg: Config{Bind: "::1"}, wantHost: "[::1]", wantKey: "dev", wantPolicy: "any", wantKeys: []any{}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			s := start(t, tc.cfg)
			ready := s.Ready()

			require.Equal(t, "http://"+tc.wantHost+":"+strconv.Itoa(ready.Port), ready.URL)
			require.Equal(t, tc.wantKey, ready.WriteKey)
			require.Equal(t, tc.wantPolicy, ready.WriteKeyPolicy)
			var info map[string]any
			getJSON(t, ready.URL+"/_dev/v1/info", &info)
			require.Equal(t, tc.wantKeys, info["writeKeys"])
		})
	}
}

func TestTwoListenersShareNothing(t *testing.T) {
	t.Parallel()
	a, b := start(t, Config{}), start(t, Config{})

	require.NotEqual(t, a.Ready().Port, b.Ready().Port)
	require.NotEqual(t, a.Ready().ServerID, b.Ready().ServerID)
}

func TestTakenPortIsErrPortInUse(t *testing.T) {
	t.Parallel()
	a := start(t, Config{})

	_, err := Start(Config{Port: a.Ready().Port})

	require.ErrorIs(t, err, ErrPortInUse)
}

func TestServerRoutesIngestionAndTheQueryAPI(t *testing.T) {
	t.Parallel()
	s := start(t, Config{})
	url := s.Ready().URL

	req, err := http.NewRequest(http.MethodPost, url+"/v1/track", strings.NewReader(`{"userId":"u1","event":"e"}`))
	require.NoError(t, err)
	req.SetBasicAuth("dev", "")
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	require.NoError(t, resp.Body.Close())
	require.Equal(t, "ok", string(body))

	var info struct {
		Cursor uint64         `json:"cursor"`
		Store  map[string]int `json:"store"`
	}
	getJSON(t, url+"/_dev/v1/info", &info)
	require.Equal(t, uint64(1), info.Cursor)
	require.Equal(t, 1, info.Store["requests"])
	require.Equal(t, 1, info.Store["events"])
	require.Equal(t, uint64(1), s.Ready().Cursor)
}

// A request line and headers above the rudder-server gateway default get
// 431 before any handler sees them.
func TestServerLimitsHeaderBytes(t *testing.T) {
	t.Parallel()
	s := start(t, Config{})
	req, err := http.NewRequest(http.MethodGet, s.Ready().URL+"/health", nil)
	require.NoError(t, err)
	req.Header.Set("X-Big", strings.Repeat("a", maxHeaderBytes+4096))

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	require.NoError(t, resp.Body.Close())

	require.Equal(t, http.StatusRequestHeaderFieldsTooLarge, resp.StatusCode)
}

// net/http reads a body the handler left unread before it reuses the
// connection. Without a read limit, a peer that stops mid-body would hold the
// connection open for good.
func TestUnreadBodyDoesNotHoldTheConnection(t *testing.T) {
	t.Parallel()
	s := start(t, Config{readTimeout: 200 * time.Millisecond})
	for _, target := range []string{"/_dev/v1/info", "/health"} {
		t.Run(target, func(t *testing.T) {
			t.Parallel()
			conn, err := net.Dial("tcp", strings.TrimPrefix(s.Ready().URL, "http://"))
			require.NoError(t, err)
			t.Cleanup(func() { _ = conn.Close() })
			_, err = io.WriteString(conn, "POST "+target+" HTTP/1.1\r\nHost: 127.0.0.1\r\nContent-Length: 100\r\n\r\n{")
			require.NoError(t, err)

			require.True(t, readsUntilClosed(conn, 5*time.Second), "the connection stayed open")
		})
	}
}

// stalledUpload sends the headers of a POST and part of its body, then waits
// until the server holds the request.
func stalledUpload(t *testing.T, s *Server) net.Conn {
	t.Helper()
	conn, err := net.Dial("tcp", strings.TrimPrefix(s.Ready().URL, "http://"))
	require.NoError(t, err)
	t.Cleanup(func() { _ = conn.Close() })
	_, err = io.WriteString(conn, "POST /v1/track HTTP/1.1\r\nHost: 127.0.0.1\r\nAuthorization: Basic ZGV2Og==\r\n"+
		"Content-Length: 100\r\n\r\n{\"userId\":")
	require.NoError(t, err)
	require.Eventually(t, func() bool { return s.readingConns() > 0 }, time.Second, time.Millisecond)
	return conn
}

// readsUntilClosed reports whether the server closes conn before the deadline.
func readsUntilClosed(conn net.Conn, deadline time.Duration) bool {
	_ = conn.SetReadDeadline(time.Now().Add(deadline))
	_, err := io.Copy(io.Discard, bufio.NewReader(conn))
	var netErr net.Error
	if err != nil && errors.As(err, &netErr) && netErr.Timeout() {
		return false
	}
	return true
}

func TestCloseDrainsAStalledUploadAtOnce(t *testing.T) {
	t.Parallel()
	s := start(t, Config{})
	upload := stalledUpload(t, s)
	idle, err := net.Dial("tcp", strings.TrimPrefix(s.Ready().URL, "http://"))
	require.NoError(t, err)
	t.Cleanup(func() { _ = idle.Close() })

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	began := time.Now()
	require.NoError(t, s.Close(ctx))

	require.Less(t, time.Since(began), 2*time.Second)
	require.True(t, readsUntilClosed(upload, time.Second), "the upload connection stayed open")
	require.True(t, readsUntilClosed(idle, time.Second), "the new connection stayed open")
	select {
	case err := <-s.Done():
		require.NoError(t, err)
	case <-time.After(time.Second):
		t.Fatal("Done did not report the end of serving")
	}
}

// With no time left to drain, Close still frees every connection instead of
// leaving goroutines behind.
func TestCloseWithAnEndedContextFreesEveryConnection(t *testing.T) {
	t.Parallel()
	s := start(t, Config{})
	upload := stalledUpload(t, s)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := s.Close(ctx)

	if err != nil {
		require.ErrorIs(t, err, context.Canceled)
	}
	require.True(t, readsUntilClosed(upload, time.Second), "the upload connection stayed open")
	require.Eventually(t, func() bool { return s.activeConns() == 0 }, time.Second, time.Millisecond)
}

func TestCloseIsIdempotent(t *testing.T) {
	t.Parallel()
	s := start(t, Config{})

	require.NoError(t, s.Close(context.Background()))
	require.NoError(t, s.Close(context.Background()))
}
