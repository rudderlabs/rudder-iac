package tests

import (
	"bufio"
	"encoding/json"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
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
	cmd    *exec.Cmd
	ready  map[string]any
	stderr *lockedBuffer
	exited chan error
}

func (p *listenProcess) url() string { return p.ready["url"].(string) }

func devListenCommand(t *testing.T, args ...string) *exec.Cmd {
	cmd := exec.Command(cliBinPath, append([]string{"dev", "listen"}, args...)...)
	cmd.Env = append(os.Environ(),
		"HOME="+t.TempDir(),
		"RUDDERSTACK_CLI_EXPERIMENTAL=true",
		"RUDDERSTACK_X_DEV_LISTEN=true",
		"RUDDERSTACK_CLI_TELEMETRY_DISABLED=true",
	)
	return cmd
}

// startListen runs the binary as a script would and waits for the ready line.
func startListen(t *testing.T, args ...string) *listenProcess {
	t.Helper()
	cmd := devListenCommand(t, args...)
	stdout, err := cmd.StdoutPipe()
	require.NoError(t, err)
	p := &listenProcess{cmd: cmd, stderr: &lockedBuffer{}, exited: make(chan error, 1)}
	cmd.Stderr = p.stderr
	require.NoError(t, cmd.Start())
	t.Cleanup(func() {
		_ = cmd.Process.Kill()
		<-p.exited
	})

	lines := make(chan []byte, 1)
	go func() {
		line, _ := bufio.NewReader(stdout).ReadBytes('\n')
		lines <- line
		_, _ = io.Copy(io.Discard, stdout)
		p.exited <- cmd.Wait()
	}()
	select {
	case line := <-lines:
		require.NoError(t, json.Unmarshal(line, &p.ready), "stdout %q, stderr %q", line, p.stderr.String())
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
		t.Fatal("dev listen did not stop")
		return 0, 0
	}
}

func TestDevListenServesUntilASignal(t *testing.T) {
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

func TestDevListenStopsWithAStalledUpload(t *testing.T) {
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

func TestDevListenInstancesShareNothing(t *testing.T) {
	t.Parallel()
	a, b := startListen(t), startListen(t)

	require.NotEqual(t, a.ready["port"], b.ready["port"])
	require.NotEqual(t, a.ready["serverId"], b.ready["serverId"])

	port := strconv.Itoa(int(a.ready["port"].(float64)))
	out, err := devListenCommand(t, "--port", port).Output()
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
	require.Equal(t, "rudder-cli dev listen --port 0", failure.Error.Next)
}

// A DNS-rebinding page reaches a container listener under its own name.
func TestDevListenChecksTheHostOnAWildcardBind(t *testing.T) {
	t.Parallel()
	p := startListen(t, "--bind", "0.0.0.0", "--allow-host", "dev-listen")
	// stdout and stderr are copied by separate goroutines, so the warning can
	// land after the ready line even though the child writes it first.
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
