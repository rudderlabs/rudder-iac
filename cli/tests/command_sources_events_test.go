package tests

import (
	"bytes"
	"errors"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// devRead runs a read command of the binary and returns stdout, stderr and
// the exit code.
func devRead(t *testing.T, args ...string) (string, string, int) {
	t.Helper()
	cmd := exec.Command(cliBinPath, append([]string{"sources", "events"}, args...)...)
	cmd.Env = append(os.Environ(),
		"HOME="+t.TempDir(),
		"RUDDERSTACK_CLI_EXPERIMENTAL=true",
		"RUDDERSTACK_X_SOURCES_LISTEN=true",
		"RUDDERSTACK_CLI_TELEMETRY_DISABLED=true",
		// A URL exported in the developer's shell would change the nexts.
		"RUDDERSTACK_DEV_URL=",
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
	// Draining the body lets the client reuse the connection.
	_, err = io.Copy(io.Discard, resp.Body)
	require.NoError(t, err)
	require.NoError(t, resp.Body.Close())
	return resp.StatusCode
}

// sources events prints what curl of the same URL prints.
func TestSourcesListenEventsMatchCurl(t *testing.T) {
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
		{args: []string{"--json"}, query: ""},
		{args: []string{"--json", "--event", "Order Completed", "--fields", "properties"}, query: "?event=Order+Completed&fields=properties"},
		{args: []string{"--json", "--view", "compact", "--limit", "1"}, query: "?limit=1&view=compact"},
	} {
		stdout, _, code := devRead(t, append(tc.args, "--url", url)...)
		require.Equal(t, 0, code)
		require.Equal(t, curl(t, url+"/_dev/v1/events"+tc.query), stdout, tc.args)
	}
}

// --json is read wherever it appears, also after the flag that failed.
func TestSourcesListenEventsErrorsInJSON(t *testing.T) {
	t.Parallel()
	stdout, stderr, code := devRead(t, "--limit", "0", "--json")

	require.Equal(t, 1, code)
	require.Empty(t, stdout)
	require.Equal(t, `{"error":{"status":null,"code":"usage","message":"no listener URL: pass --url or set `+
		`RUDDERSTACK_DEV_URL to the url of the ready line","param":null,"details":null,`+
		`"next":"rudder-cli sources events --url http://127.0.0.1:4321"}}`+"\n", stderr)
}

// SIGTERM with a blocked upload and a long-poll exits 0 in under 2 s.
func TestSourcesListenStopsWithALongPollAndAStalledUpload(t *testing.T) {
	t.Parallel()
	p := startListen(t)
	upload, err := net.Dial("tcp", strings.TrimPrefix(p.url(), "http://"))
	require.NoError(t, err)
	t.Cleanup(func() { _ = upload.Close() })
	_, err = io.WriteString(upload, "POST /v1/track HTTP/1.1\r\nHost: 127.0.0.1\r\nContent-Length: 100\r\n\r\n{")
	require.NoError(t, err)
	answered := make(chan int, 1)
	go func() {
		resp, err := http.Get(p.url() + "/_dev/v1/events?view=counts&wait=60s")
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
