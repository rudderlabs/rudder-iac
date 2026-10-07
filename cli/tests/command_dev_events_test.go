package tests

import (
	"bytes"
	"errors"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/http/httputil"
	neturl "net/url"
	"os"
	"os/exec"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
)

// devRead runs a read command of the binary and returns stdout, stderr and
// the exit code.
func devRead(t *testing.T, args ...string) (string, string, int) {
	t.Helper()
	cmd := exec.Command(cliBinPath, append([]string{"local", "event-stream", "events"}, args...)...)
	cmd.Env = append(os.Environ(),
		"HOME="+t.TempDir(),
		"RUDDERSTACK_CLI_EXPERIMENTAL=true",
		"RUDDERSTACK_X_LOCAL_EVENT_STREAM=true",
		"RUDDERSTACK_CLI_TELEMETRY_DISABLED=true",
		// A URL exported in the developer's shell would change the nexts.
		"RUDDERSTACK_LOCAL_EVENT_STREAM_URL=",
	)
	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	err := cmd.Run()
	var exit *exec.ExitError
	if errors.As(err, &exit) {
		return stdout.String(), stderr.String(), exit.ExitCode()
	}
	require.NoError(t, err)
	return stdout.String(), stderr.String(), 0
}

func curl(t *testing.T, url string) string {
	t.Helper()
	req, err := http.NewRequest(http.MethodGet, url, nil)
	require.NoError(t, err)
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	require.NoError(t, resp.Body.Close())
	require.Equal(t, http.StatusOK, resp.StatusCode, string(body))
	return string(body)
}

func post(t *testing.T, url, body string) int {
	t.Helper()
	req, err := http.NewRequest(http.MethodPost, url, strings.NewReader(body))
	require.NoError(t, err)
	req.SetBasicAuth("dev", "")
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	require.NoError(t, resp.Body.Close())
	return resp.StatusCode
}

// A read command prints what curl of the same URL prints. The
// summary differs only in the --url the CLI adds to each next.
func TestDevListenEventsMatchCurl(t *testing.T) {
	t.Parallel()
	p := startListen(t)
	url := p.url()
	require.Equal(t, http.StatusOK, post(t, url+"/v1/batch", `{"batch":[
		{"type":"track","userId":"u1","event":"Order Completed","properties":{"total":42}},
		{"type":"track","userId":"u1","event":"Order Completed","properties":{"total":"42"}},
		{"type":"identify","userId":"u1","traits":{"plan":"pro"}}]}`))
	require.Equal(t, http.StatusBadRequest, post(t, url+"/v1/track", `{"event":"Order Completed"}`))

	for _, tc := range []struct {
		args  []string
		query string
	}{
		{args: []string{"list", "--json"}, query: ""},
		{args: []string{"list", "--json", "--event", "Order Completed", "--fields", "properties"}, query: "?event=Order+Completed&fields=properties"},
		{args: []string{"list", "--json", "--view", "compact", "--limit", "1"}, query: "?limit=1&view=compact"},
	} {
		stdout, _, code := devRead(t, append(tc.args, "--url", url)...)
		require.Equal(t, 0, code)
		require.Equal(t, curl(t, url+"/_local/v1/events"+tc.query), stdout, tc.args)
	}

	stdout, stderr, code := devRead(t, "summary", "--url", url, "--event", "Order Completed", "--json")
	require.Equal(t, 0, code, stderr)
	want := strings.ReplaceAll(curl(t, url+"/_local/v1/events?event=Order+Completed&view=counts"),
		`"next":"rudder-cli local event-stream events summary `, `"next":"rudder-cli local event-stream events summary --url `+url+` `)
	require.Equal(t, want, stdout)
	require.Contains(t, stdout, `"byEvent":{"Order Completed":2}`)
	require.Contains(t, stdout, `"rejected":{"events":1,"byEvent":{"Order Completed":1}}`)
}

// A read that waits gets the event that arrives after it started.
func TestDevListenEventsWaitsForANewEvent(t *testing.T) {
	t.Parallel()
	p := startListen(t)
	listener := p.url()
	require.Equal(t, http.StatusOK, post(t, listener+"/v1/track", `{"userId":"u","event":"Before"}`))

	// The binary's start time varies under load, so a proxy reports when
	// the read reaches the listener, and the post follows that.
	target, err := neturl.Parse(listener)
	require.NoError(t, err)
	forward := httputil.NewSingleHostReverseProxy(target)
	reading := make(chan struct{})
	var once sync.Once
	proxy := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		once.Do(func() { close(reading) })
		forward.ServeHTTP(w, r)
	}))
	t.Cleanup(proxy.Close)
	go func() {
		<-reading
		time.Sleep(300 * time.Millisecond)
		req, _ := http.NewRequest(http.MethodPost, listener+"/v1/track", strings.NewReader(`{"userId":"u","event":"After"}`))
		req.SetBasicAuth("dev", "")
		// No require here: it may not run off the test goroutine. A lost
		// post fails the read below.
		resp, err := http.DefaultClient.Do(req)
		if err == nil {
			_ = resp.Body.Close()
		}
	}()

	stdout, stderr, code := devRead(t, "summary", "--url", proxy.URL, "--since", "1", "--wait", "10s", "--json")

	require.Equal(t, 0, code, stderr)
	require.Contains(t, stdout, `"timedOut":false`)
	require.Contains(t, stdout, `"byEvent":{"After":1}`)
	require.GreaterOrEqual(t, gjson.Get(stdout, "waitedMs").Int(), int64(250), "the read waited for the post")
}

// --json is read wherever it appears, also after the flag that failed.
func TestDevListenEventsErrorsInJSON(t *testing.T) {
	t.Parallel()
	stdout, stderr, code := devRead(t, "list", "--wait", "30s", "--json")

	require.Equal(t, 1, code)
	require.Empty(t, stdout)
	require.Equal(t, `{"error":{"status":null,"code":"usage","message":"--wait and --min belong to events summary: `+
		`events list reads what is there","param":null,"details":null,`+
		`"next":"rudder-cli local event-stream events summary --url http://127.0.0.1:4321 --wait 30s --json"}}`+"\n", stderr)
}

// SIGTERM with a blocked upload and a long-poll exits 0 in under 2 s.
func TestDevListenStopsWithALongPollAndAStalledUpload(t *testing.T) {
	t.Parallel()
	p := startListen(t)
	upload, err := net.Dial("tcp", strings.TrimPrefix(p.url(), "http://"))
	require.NoError(t, err)
	t.Cleanup(func() { _ = upload.Close() })
	_, err = io.WriteString(upload, "POST /v1/track HTTP/1.1\r\nHost: 127.0.0.1\r\nContent-Length: 100\r\n\r\n{")
	require.NoError(t, err)
	answered := make(chan int, 1)
	go func() {
		resp, err := http.Get(p.url() + "/_local/v1/events?view=counts&wait=60s")
		if err != nil {
			answered <- 0
			return
		}
		_ = resp.Body.Close()
		answered <- resp.StatusCode
	}()

	code, took := p.stop(t, syscall.SIGTERM)

	require.Equal(t, 0, code, p.stderr.String())
	require.Less(t, took, 2*time.Second)
	// The poll has no observable start. A poll that waited answers 503; one
	// the stop beat to the listener never connects.
	require.Contains(t, []int{http.StatusServiceUnavailable, 0}, <-answered)
}
