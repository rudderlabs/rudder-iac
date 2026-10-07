package local

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net"
	"net/http"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/rudderlabs/rudder-iac/cli/internal/cmd/cmderrors"
)

func TestListenRejectsBadFlags(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		args []string
		want string
	}{
		{
			args: []string{"--port", "70000"},
			want: "Error: --port must be 0 to 65535, got 70000\nNext: rudder-cli local event-stream serve --port 0\n",
		},
		{
			args: []string{"--port", "-1"},
			want: "Error: --port must be 0 to 65535, got -1\nNext: rudder-cli local event-stream serve --port 0\n",
		},
		{
			args: []string{"--bind", "localhost"},
			want: "Error: --bind must be an IP address such as 127.0.0.1, got \"localhost\"\n" +
				"Next: rudder-cli local event-stream serve --bind 127.0.0.1\n",
		},
		{
			args: []string{"--write-key", ""},
			want: "Error: --write-key needs a key\nNext: rudder-cli local event-stream serve --help\n",
		},
		{
			args: []string{"--allow-host", ""},
			want: "Error: --allow-host needs a host name\nNext: rudder-cli local event-stream serve --help\n",
		},
		{
			args: []string{"--allow-host", "dev-listen:4321"},
			want: "Error: --allow-host takes a host name without a port, got \"dev-listen:4321\"\n" +
				"Next: rudder-cli local event-stream serve --allow-host dev-listen\n",
		},
		{
			args: []string{"now"},
			want: "Error: unknown argument \"now\" for \"rudder-cli local event-stream serve\"\nNext: rudder-cli local event-stream serve --help\n",
		},
		{
			args: []string{"--json"},
			want: "Error: unknown flag: --json\nNext: rudder-cli local event-stream serve --help\n",
		},
	} {
		t.Run(strings.Join(tc.args, " "), func(t *testing.T) {
			t.Parallel()

			stdout, stderr, err := execute(append([]string{"local", "event-stream", "serve"}, tc.args...)...)

			var silent *cmderrors.SilentError
			require.ErrorAs(t, err, &silent)
			require.Empty(t, stdout)
			require.Equal(t, tc.want, stderr)
		})
	}
}

// A script that starts the listener in the background reads the failure
// from the last line of its log.
func TestListenReportsATakenPort(t *testing.T) {
	t.Parallel()
	taken, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	t.Cleanup(func() { _ = taken.Close() })
	port := strconv.Itoa(taken.Addr().(*net.TCPAddr).Port)

	stdout, stderr, err := execute("local", "event-stream", "serve", "--port", port)

	var silent *cmderrors.SilentError
	require.ErrorAs(t, err, &silent)
	require.Empty(t, stdout)
	lines := strings.Split(strings.TrimSuffix(stderr, "\n"), "\n")
	var got struct {
		Error struct {
			Status  *int    `json:"status"`
			Code    string  `json:"code"`
			Message string  `json:"message"`
			Param   *string `json:"param"`
			Details any     `json:"details"`
			Next    string  `json:"next"`
		} `json:"error"`
	}
	require.NoError(t, json.Unmarshal([]byte(lines[len(lines)-1]), &got), stderr)
	require.Nil(t, got.Error.Status)
	require.Equal(t, "port_in_use", got.Error.Code)
	require.Contains(t, got.Error.Message, port)
	require.Equal(t, "rudder-cli local event-stream serve --port 0", got.Error.Next)
}

type listening struct {
	ready  map[string]any
	stderr *bytes.Buffer
	cancel context.CancelFunc
	// done is closed when runListen returns err.
	done chan struct{}
	err  error
}

// listen runs the command body until the test cancels it, and returns once
// the ready line is out.
func listen(t *testing.T, opts listenOptions) *listening {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	stdoutR, stdoutW := io.Pipe()
	l := &listening{stderr: &bytes.Buffer{}, cancel: cancel, done: make(chan struct{})}
	go func() {
		l.err = runListen(ctx, stdoutW, l.stderr, opts, "1.2.3")
		_ = stdoutW.Close()
		close(l.done)
	}()
	t.Cleanup(func() {
		cancel()
		<-l.done
	})

	line, err := bufio.NewReader(stdoutR).ReadBytes('\n')
	require.NoError(t, err)
	require.NoError(t, json.Unmarshal(line, &l.ready))
	go func() { _, _ = io.Copy(io.Discard, stdoutR) }()
	return l
}

func TestListenServesUntilItsContextEnds(t *testing.T) {
	t.Parallel()
	l := listen(t, listenOptions{
		bind:       "127.0.0.1",
		writeKeys:  []string{"fake-write-key-for-tests"},
		allowHosts: []string{"dev-listen"},
	})
	url := l.ready["url"].(string)

	require.Equal(t, true, l.ready["ready"])
	require.Equal(t, "127.0.0.1", l.ready["bind"])
	require.Equal(t, "fake...ests", l.ready["writeKey"])
	require.Equal(t, "allowlist", l.ready["writeKeyPolicy"])
	require.Equal(t, float64(0), l.ready["cursor"])
	require.Equal(t, url+"/_local/ui/", l.ready["ui"])

	req, err := http.NewRequest(http.MethodPost, url+"/v1/track", strings.NewReader(`{"userId":"u1","event":"e"}`))
	require.NoError(t, err)
	req.SetBasicAuth("fake-write-key-for-tests", "")
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	require.NoError(t, resp.Body.Close())
	require.Equal(t, http.StatusOK, resp.StatusCode)

	req, err = http.NewRequest(http.MethodGet, url+"/_local/v1/info", nil)
	require.NoError(t, err)
	req.Host = "dev-listen"
	resp, err = http.DefaultClient.Do(req)
	require.NoError(t, err)
	var info struct {
		Version string `json:"version"`
		Cursor  int    `json:"cursor"`
	}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&info))
	require.NoError(t, resp.Body.Close())
	require.Equal(t, "1.2.3", info.Version)
	require.Equal(t, 1, info.Cursor)

	l.cancel()
	select {
	case <-l.done:
		require.NoError(t, l.err)
	case <-time.After(2 * time.Second):
		t.Fatal("local event-stream serve did not stop")
	}
	require.Empty(t, l.stderr.String(), "stderr stays silent when it is not a terminal")
}

func TestListenWarnsOnAnExposedBind(t *testing.T) {
	t.Parallel()
	l := listen(t, listenOptions{bind: "0.0.0.0"})
	port := strconv.Itoa(int(l.ready["port"].(float64)))

	require.Equal(t, "http://127.0.0.1:"+port, l.ready["url"])
	require.Equal(t, "warning: listening on 0.0.0.0:"+port+". The query API has no authentication. "+
		"Any host that reaches this port can read it.\n", l.stderr.String())
}
