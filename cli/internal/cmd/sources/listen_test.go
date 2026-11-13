package sources

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"syscall"
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
			want: "Error: --port must be 0 to 65535, got 70000\nNext: rudder-cli sources listen --port 0\n",
		},
		{
			args: []string{"--port", "-1"},
			want: "Error: --port must be 0 to 65535, got -1\nNext: rudder-cli sources listen --port 0\n",
		},
		{
			args: []string{"--bind", "localhost"},
			want: "Error: --bind must be an IP address such as 127.0.0.1, got \"localhost\"\n" +
				"Next: rudder-cli sources listen --bind 127.0.0.1\n",
		},
		{
			args: []string{"--write-key", ""},
			want: "Error: --write-key needs a key\nNext: rudder-cli sources listen --help\n",
		},
		{
			args: []string{"--allow-host", ""},
			want: "Error: --allow-host needs a host name\nNext: rudder-cli sources listen --help\n",
		},
		{
			args: []string{"--allow-host", "dev-listen:4321"},
			want: "Error: --allow-host takes a host name without a port, got \"dev-listen:4321\"\n" +
				"Next: rudder-cli sources listen --allow-host dev-listen\n",
		},
		{
			args: []string{"--out", ""},
			want: "Error: --out needs a file name\nNext: rudder-cli sources listen --out \"$(mktemp -d)/events\"\n",
		},
		{
			args: []string{"now"},
			want: "Error: unknown argument \"now\" for \"rudder-cli sources listen\"\nNext: rudder-cli sources listen --help\n",
		},
		{
			args: []string{"--json"},
			want: "Error: unknown flag: --json\nNext: rudder-cli sources listen --help\n",
		},
	} {
		t.Run(strings.Join(tc.args, " "), func(t *testing.T) {
			t.Parallel()

			stdout, stderr, err := execute(append([]string{"sources", "listen"}, tc.args...)...)

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

	stdout, stderr, err := execute("sources", "listen", "--port", port)

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
	require.Equal(t, "rudder-cli sources listen --port 0", got.Error.Next)
}

type listening struct {
	ready map[string]any
	// stdout holds the events; stderr holds every stderr line but the ready line.
	stdout, stderr *lockedBuffer
	cancel         context.CancelFunc
	// done is closed when runListen returns err; copied when stderr is read.
	done, copied chan struct{}
	err          error
}

// lockedBuffer is safe for the listener goroutines and the test to share.
type lockedBuffer struct {
	mu sync.Mutex
	b  bytes.Buffer
}

func (b *lockedBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.b.Write(p)
}

func (b *lockedBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.b.String()
}

// listen runs the command body until the test cancels it, and returns once
// the ready line is out on stderr.
func listen(t *testing.T, opts listenOptions) *listening {
	t.Helper()
	return listenTo(t, opts, nil)
}

// listenTo is listen with stdout replaced by out, unless out is nil.
func listenTo(t *testing.T, opts listenOptions, out io.Writer) *listening {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	stderrR, stderrW := io.Pipe()
	l := &listening{stdout: &lockedBuffer{}, stderr: &lockedBuffer{}, cancel: cancel, done: make(chan struct{}), copied: make(chan struct{})}
	if out == nil {
		out = l.stdout
	}
	go func() {
		l.err = runListen(ctx, out, stderrW, opts, "1.2.3")
		_ = stderrW.Close()
		close(l.done)
	}()
	t.Cleanup(func() {
		cancel()
		<-l.done
	})

	r := bufio.NewReader(stderrR)
	for {
		line, err := r.ReadBytes('\n')
		require.NoError(t, err)
		if bytes.HasPrefix(line, []byte(`{"ready":true`)) {
			require.NoError(t, json.Unmarshal(line, &l.ready))
			break
		}
		_, _ = l.stderr.Write(line)
	}
	go func() {
		_, _ = io.Copy(l.stderr, r)
		close(l.copied)
	}()
	return l
}

func TestListenServesUntilItsContextEnds(t *testing.T) {
	t.Parallel()
	l := listen(t, listenOptions{
		bind:       "127.0.0.1",
		writeKeys:  []string{"fake-write-key-for-tests"},
		allowHosts: []string{"dev-listen"},
		store:      true,
	})
	url := l.ready["url"].(string)

	require.Equal(t, true, l.ready["ready"])
	require.Equal(t, "127.0.0.1", l.ready["bind"])
	require.Equal(t, "fake...ests", l.ready["writeKey"])
	require.Equal(t, "allowlist", l.ready["writeKeyPolicy"])
	require.Equal(t, float64(0), l.ready["cursor"])
	require.Equal(t, url+"/_dev/ui/", l.ready["ui"])

	req, err := http.NewRequest(http.MethodPost, url+"/v1/track", strings.NewReader(`{"userId":"u1","event":"e"}`))
	require.NoError(t, err)
	req.SetBasicAuth("fake-write-key-for-tests", "")
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	require.NoError(t, resp.Body.Close())
	require.Equal(t, http.StatusOK, resp.StatusCode)

	req, err = http.NewRequest(http.MethodGet, url+"/_dev/v1/info", nil)
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
	require.Equal(t, `{"userId":"u1","event":"e"}`+"\n", l.stdout.String())

	l.cancel()
	select {
	case <-l.done:
		require.NoError(t, l.err)
	case <-time.After(2 * time.Second):
		t.Fatal("sources listen did not stop")
	}
	require.Empty(t, l.stderr.String(), "stderr has the ready line only when it is not a terminal")
}

func TestListenWithoutStoreKeepsNothing(t *testing.T) {
	t.Parallel()
	l := listen(t, listenOptions{bind: "127.0.0.1"})
	url := l.ready["url"].(string)

	require.NotContains(t, l.ready, "cursor")
	require.NotContains(t, l.ready, "ui")
	resp, err := http.Get(url + "/_dev/v1/info")
	require.NoError(t, err)
	require.NoError(t, resp.Body.Close())
	require.Equal(t, http.StatusNotFound, resp.StatusCode)
}

func TestListenWarnsOnAnExposedBind(t *testing.T) {
	t.Parallel()
	l := listen(t, listenOptions{bind: "0.0.0.0"})
	port := strconv.Itoa(int(l.ready["port"].(float64)))

	require.Equal(t, "http://127.0.0.1:"+port, l.ready["url"])
	require.Equal(t, "warning: listening on 0.0.0.0:"+port+". The query API has no authentication. "+
		"Any host that reaches this port can read it.\n", l.stderr.String())
}

func sendTrack(t *testing.T, url, body string) int {
	t.Helper()
	req, err := http.NewRequest(http.MethodPost, url+"/v1/track", strings.NewReader(body))
	require.NoError(t, err)
	req.SetBasicAuth("k", "")
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	require.NoError(t, resp.Body.Close())
	return resp.StatusCode
}

func stopped(t *testing.T, l *listening) error {
	t.Helper()
	select {
	case <-l.done:
		<-l.copied
		return l.err
	case <-time.After(5 * time.Second):
		t.Fatal("sources listen did not stop")
		return nil
	}
}

// lastError is the error object on the last stderr line.
func lastError(t *testing.T, stderr string) cliError {
	t.Helper()
	lines := strings.Split(strings.TrimSuffix(stderr, "\n"), "\n")
	var got struct{ Error cliError }
	require.NoError(t, json.Unmarshal([]byte(lines[len(lines)-1]), &got), stderr)
	return got.Error
}

func TestListenOutWritesStampedLinesToANewFile(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "events")
	l := listen(t, listenOptions{bind: "127.0.0.1", out: path})
	url := l.ready["url"].(string)

	before := time.Now().UTC()
	require.Equal(t, http.StatusOK, sendTrack(t, url, `{"userId":"u1","event":"A"}`))
	require.Equal(t, http.StatusOK, sendTrack(t, url, `{ "userId": "u1", "event": "B" }`))
	l.cancel()
	require.NoError(t, stopped(t, l))

	require.Empty(t, l.stdout.String(), "the events go to the file, not stdout")
	data, err := os.ReadFile(path)
	require.NoError(t, err)
	lines := strings.Split(strings.TrimSuffix(string(data), "\n"), "\n")
	require.Len(t, lines, 2)
	for i, event := range []string{"A", "B"} {
		var got struct {
			Dev struct {
				ServerID   string    `json:"serverId"`
				Seq        int       `json:"seq"`
				ReceivedAt time.Time `json:"receivedAt"`
			} `json:"_dev"`
		}
		require.NoError(t, json.Unmarshal([]byte(lines[i]), &got), lines[i])
		require.Equal(t, l.ready["serverId"], got.Dev.ServerID)
		require.Equal(t, i+1, got.Dev.Seq)
		require.False(t, got.Dev.ReceivedAt.Before(before.Truncate(time.Second)))
		// After the _dev object the line is the event as stdout prints it.
		_, rest, ok := strings.Cut(lines[i], `"},`)
		require.True(t, ok, lines[i])
		require.Equal(t, `"userId":"u1","event":"`+event+`"}`, rest)
	}
	info, err := os.Stat(path)
	require.NoError(t, err)
	require.Equal(t, os.FileMode(0o600), info.Mode().Perm())
}

func TestListenOutRefusesAFileThatExists(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "events")
	require.NoError(t, os.WriteFile(path, []byte("kept\n"), 0o600))
	var stderr bytes.Buffer

	err := runListen(context.Background(), io.Discard, &stderr, listenOptions{bind: "127.0.0.1", out: path}, "1.2.3")

	var silent *cmderrors.SilentError
	require.ErrorAs(t, err, &silent)
	got := lastError(t, stderr.String())
	require.Equal(t, "out_exists", got.Code)
	require.Equal(t, "--out "+path+" exists; give a new file", got.Message)
	require.Equal(t, `rudder-cli sources listen --out "$(mktemp -d)/events"`, got.Next)
	data, err := os.ReadFile(path)
	require.NoError(t, err)
	require.Equal(t, "kept\n", string(data))
}

// failingWriter fails every write, as a closed pipe or a full disk does.
type failingWriter struct{}

func (failingWriter) Write([]byte) (int, error) { return 0, syscall.EPIPE }

func TestListenStopsWhenAnEventCannotBeWritten(t *testing.T) {
	t.Parallel()
	l := listenTo(t, listenOptions{bind: "127.0.0.1"}, failingWriter{})
	url := l.ready["url"].(string)

	require.Equal(t, http.StatusOK, sendTrack(t, url, `{"userId":"u1","event":"A"}`))

	var silent *cmderrors.SilentError
	require.ErrorAs(t, stopped(t, l), &silent)
	got := lastError(t, l.stderr.String())
	require.Equal(t, "output_failed", got.Code)
	require.Equal(t, "writing events to stdout: broken pipe; the listener stopped, and 1 accepted event was never written",
		got.Message)
	_, err := http.Post(url+"/v1/track", "application/json", strings.NewReader(`{}`))
	require.Error(t, err, "a stopped listener takes no more events")
}

// heldWriter holds the first write until release closes, then fails it, as a
// write to a disk that fills while the listener is stopping.
type heldWriter struct {
	writing, release chan struct{}
	once             sync.Once
}

func (w *heldWriter) Write([]byte) (int, error) {
	w.once.Do(func() { close(w.writing) })
	<-w.release
	return 0, syscall.ENOSPC
}

// A signal stops the listener while an event is being written. The write
// fails as Close drains the request, and the listener must still report it.
func TestListenReportsAWriteThatFailsWhileItStops(t *testing.T) {
	t.Parallel()
	w := &heldWriter{writing: make(chan struct{}), release: make(chan struct{})}
	l := listenTo(t, listenOptions{bind: "127.0.0.1"}, w)
	url := l.ready["url"].(string)
	addr := strings.TrimPrefix(url, "http://")

	status := make(chan int, 1)
	go func() {
		resp, err := http.Post(url+"/v1/track", "application/json", strings.NewReader(`{"userId":"u1","event":"A"}`))
		if err != nil {
			status <- 0
			return
		}
		_ = resp.Body.Close()
		status <- resp.StatusCode
	}()
	<-w.writing

	// Stop, and wait until Close has shut the port, so the signal has won.
	l.cancel()
	require.Eventually(t, func() bool {
		conn, err := net.Dial("tcp", addr)
		if err != nil {
			return true
		}
		_ = conn.Close()
		return false
	}, 5*time.Second, 5*time.Millisecond)
	close(w.release)

	require.Equal(t, http.StatusOK, <-status, "the SDK still gets its answer")
	var silent *cmderrors.SilentError
	require.ErrorAs(t, stopped(t, l), &silent)
	got := lastError(t, l.stderr.String())
	require.Equal(t, "output_failed", got.Code)
	require.Equal(t, "writing events to stdout: no space left on device; the listener stopped, and 1 accepted event was never written",
		got.Message)
}
