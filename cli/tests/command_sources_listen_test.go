package tests

import (
	"bufio"
	"bytes"
	"encoding/json"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// lockedBuffer lets a test read stderr while exec still copies into it.
type lockedBuffer struct {
	mu  sync.Mutex
	buf strings.Builder
}

func (b *lockedBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.Write(p)
}

func (b *lockedBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.String()
}

type listenProcess struct {
	cmd   *exec.Cmd
	ready map[string]any
	// stdout holds the events; stderr holds every stderr line but the ready line.
	stdout, stderr *lockedBuffer
	exited         chan error
}

func (p *listenProcess) url() string { return p.ready["url"].(string) }

func sourcesListenCommand(t *testing.T, args ...string) *exec.Cmd {
	cmd := exec.Command(cliBinPath, append([]string{"sources", "listen"}, args...)...)
	cmd.Env = append(os.Environ(),
		"HOME="+t.TempDir(),
		"RUDDERSTACK_CLI_EXPERIMENTAL=true",
		"RUDDERSTACK_X_SOURCES_LISTEN=true",
		"RUDDERSTACK_CLI_TELEMETRY_DISABLED=true",
	)
	return cmd
}

// startListen runs the binary with --store, as the read commands need, and
// waits for the ready line.
func startListen(t *testing.T, args ...string) *listenProcess {
	t.Helper()
	return startStream(t, append([]string{"--store"}, args...)...)
}

// startStream runs the binary as a script would and waits for the ready line
// on stderr.
func startStream(t *testing.T, args ...string) *listenProcess {
	t.Helper()
	return startStreamTo(t, nil, args...)
}

// startStreamTo is startStream with stdout going to out, unless out is nil.
func startStreamTo(t *testing.T, out io.Writer, args ...string) *listenProcess {
	t.Helper()
	cmd := sourcesListenCommand(t, args...)
	stderr, err := cmd.StderrPipe()
	require.NoError(t, err)
	p := &listenProcess{cmd: cmd, stdout: &lockedBuffer{}, stderr: &lockedBuffer{}, exited: make(chan error, 1)}
	cmd.Stdout = p.stdout
	if out != nil {
		cmd.Stdout = out
	}
	require.NoError(t, cmd.Start())
	t.Cleanup(func() {
		_ = cmd.Process.Kill()
		<-p.exited
	})

	lines := make(chan []byte, 1)
	go func() {
		r := bufio.NewReader(stderr)
		for {
			line, err := r.ReadBytes('\n')
			if bytes.HasPrefix(line, []byte(`{"ready":true`)) {
				lines <- line
				break
			}
			_, _ = p.stderr.Write(line)
			if err != nil {
				lines <- nil
				break
			}
		}
		_, _ = io.Copy(p.stderr, r)
		p.exited <- cmd.Wait()
	}()
	select {
	case line := <-lines:
		require.NoError(t, json.Unmarshal(line, &p.ready), "ready %q, stderr %q", line, p.stderr.String())
	case <-time.After(10 * time.Second):
		t.Fatal("no ready line")
	}
	return p
}

// stop signals the process and returns its exit code and how long it took.
func (p *listenProcess) stop(t *testing.T, sig os.Signal) (int, time.Duration) {
	t.Helper()
	began := time.Now()
	require.NoError(t, p.cmd.Process.Signal(sig))
	select {
	case <-p.exited:
		p.exited <- nil
		return p.cmd.ProcessState.ExitCode(), time.Since(began)
	case <-time.After(10 * time.Second):
		t.Fatal("sources listen did not stop")
		return 0, 0
	}
}

func TestSourcesListenServesUntilASignal(t *testing.T) {
	t.Parallel()
	for _, sig := range []os.Signal{os.Interrupt, syscall.SIGTERM} {
		t.Run(sig.String(), func(t *testing.T) {
			t.Parallel()
			p := startListen(t)
			require.Equal(t, float64(p.cmd.Process.Pid), p.ready["pid"])
			require.Equal(t, float64(0), p.ready["cursor"])

			req, err := http.NewRequest(http.MethodPost, p.url()+"/v1/track",
				strings.NewReader(`{"event":"Order Completed","userId":"u1"}`))
			require.NoError(t, err)
			req.SetBasicAuth("dev", "")
			resp, err := http.DefaultClient.Do(req)
			require.NoError(t, err)
			body, _ := io.ReadAll(resp.Body)
			require.NoError(t, resp.Body.Close())
			require.Equal(t, "ok", string(body))

			resp, err = http.Get(p.url() + "/_dev/v1/info")
			require.NoError(t, err)
			var info map[string]any
			require.NoError(t, json.NewDecoder(resp.Body).Decode(&info))
			require.NoError(t, resp.Body.Close())
			for key, value := range p.ready {
				if key != "cursor" {
					require.Equal(t, value, info[key], key)
				}
			}
			require.Equal(t, float64(1), info["cursor"])
			require.Equal(t, float64(1), info["store"].(map[string]any)["requests"])
			require.NotEmpty(t, info["version"])

			code, _ := p.stop(t, sig)
			require.Equal(t, 0, code, p.stderr.String())
			require.Empty(t, p.stderr.String())
		})
	}
}

func TestSourcesListenStopsWithAStalledUpload(t *testing.T) {
	t.Parallel()
	p := startListen(t)
	conn, err := net.Dial("tcp", strings.TrimPrefix(p.url(), "http://"))
	require.NoError(t, err)
	t.Cleanup(func() { _ = conn.Close() })
	_, err = io.WriteString(conn, "POST /v1/track HTTP/1.1\r\nHost: 127.0.0.1\r\nContent-Length: 100\r\n\r\n{")
	require.NoError(t, err)
	time.Sleep(100 * time.Millisecond)

	code, took := p.stop(t, syscall.SIGTERM)

	require.Equal(t, 0, code, p.stderr.String())
	require.Less(t, took, 2*time.Second)
}

func TestSourcesListenInstancesShareNothing(t *testing.T) {
	t.Parallel()
	a, b := startListen(t), startListen(t)

	require.NotEqual(t, a.ready["port"], b.ready["port"])
	require.NotEqual(t, a.ready["serverId"], b.ready["serverId"])

	port := strconv.Itoa(int(a.ready["port"].(float64)))
	out, err := sourcesListenCommand(t, "--port", port).Output()
	var exit *exec.ExitError
	require.ErrorAs(t, err, &exit)
	require.Equal(t, 1, exit.ExitCode())
	require.Empty(t, out)
	lines := strings.Split(strings.TrimSpace(string(exit.Stderr)), "\n")
	var failure struct {
		Error struct {
			Code string `json:"code"`
			Next string `json:"next"`
		} `json:"error"`
	}
	require.NoError(t, json.Unmarshal([]byte(lines[len(lines)-1]), &failure), string(exit.Stderr))
	require.Equal(t, "port_in_use", failure.Error.Code)
	require.Equal(t, "rudder-cli sources listen --port 0", failure.Error.Next)
}

func TestSourcesListenPrintsEventsAndKeepsNothingByDefault(t *testing.T) {
	t.Parallel()
	p := startStream(t, "--write-key", "web")
	require.NotContains(t, p.ready, "cursor")
	require.NotContains(t, p.ready, "ui")

	for key, body := range map[string]string{
		"web":   `{"batch":[{"type":"track","event":"A","userId":"u1"},{"type":"track","event":"B","userId":"u1"}]}`,
		"other": `{"event":"C","userId":"u1"}`,
	} {
		req, err := http.NewRequest(http.MethodPost, p.url()+"/v1/batch", strings.NewReader(body))
		require.NoError(t, err)
		req.SetBasicAuth(key, "")
		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		require.NoError(t, resp.Body.Close())
	}
	resp, err := http.Get(p.url() + "/_dev/v1/info")
	require.NoError(t, err)
	require.NoError(t, resp.Body.Close())
	require.Equal(t, http.StatusNotFound, resp.StatusCode)

	code, _ := p.stop(t, syscall.SIGTERM)
	require.Equal(t, 0, code, p.stderr.String())
	require.Equal(t, `{"type":"track","event":"A","userId":"u1"}`+"\n"+`{"type":"track","event":"B","userId":"u1"}`+"\n",
		p.stdout.String())
	require.Contains(t, p.stderr.String(), "refused: 401 POST /v1/batch: ")
}

// A DNS-rebinding page reaches a container listener under its own name.
func TestSourcesListenChecksTheHostOnAWildcardBind(t *testing.T) {
	t.Parallel()
	p := startListen(t, "--bind", "0.0.0.0", "--allow-host", "dev-listen")
	require.Eventually(t, func() bool {
		return strings.Contains(p.stderr.String(), "warning: listening on 0.0.0.0:")
	}, 5*time.Second, 10*time.Millisecond)

	for host, want := range map[string]int{
		"evil.example":   http.StatusForbidden,
		"dev-listen":     http.StatusOK,
		"localhost:4321": http.StatusOK,
	} {
		req, err := http.NewRequest(http.MethodGet, p.url()+"/_dev/v1/info", nil)
		require.NoError(t, err)
		req.Host = host
		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		require.NoError(t, resp.Body.Close())
		require.Equal(t, want, resp.StatusCode, host)
	}
}

// A reader that closes the pipe must not leave the SDK getting answers for
// events that go nowhere: the listener says so and exits 1.
func TestSourcesListenStopsWhenStdoutCloses(t *testing.T) {
	t.Parallel()
	r, w, err := os.Pipe()
	require.NoError(t, err)
	p := startStreamTo(t, w)
	require.NoError(t, w.Close())
	require.NoError(t, r.Close())

	resp, err := http.Post(p.url()+"/v1/track", "application/json", strings.NewReader(`{"event":"A","userId":"u1"}`))
	if err == nil {
		require.NoError(t, resp.Body.Close())
	}

	select {
	case <-p.exited:
		p.exited <- nil
	case <-time.After(10 * time.Second):
		t.Fatal("sources listen did not stop")
	}
	require.Equal(t, 1, p.cmd.ProcessState.ExitCode(), p.stderr.String())
	require.Contains(t, p.stderr.String(), `"code":"output_failed"`)
	require.Contains(t, p.stderr.String(), "1 accepted event was never written")
}

func TestSourcesListenOutRefusesAFileThatExists(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "events")
	require.NoError(t, os.WriteFile(path, nil, 0o600))

	out, err := sourcesListenCommand(t, "--port", "0", "--out", path).CombinedOutput()

	var exit *exec.ExitError
	require.ErrorAs(t, err, &exit)
	require.Equal(t, 1, exit.ExitCode())
	require.Contains(t, string(out), `"code":"out_exists"`)
}
