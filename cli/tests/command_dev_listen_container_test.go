package tests

import (
	"encoding/json"
	"io"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// devListenImageEnv names the image under test. The test builds no image:
// the CI job runs `make docker-build` and passes the tag.
const devListenImageEnv = "RUDDER_CLI_IMAGE"

func docker(t *testing.T, args ...string) string {
	t.Helper()
	cmd := exec.Command("docker", args...)
	var stderr strings.Builder
	cmd.Stderr = &stderr
	// stdout only: docker prints warnings, such as a platform mismatch, on stderr.
	out, err := cmd.Output()
	require.NoError(t, err, "docker %s: %s", strings.Join(args, " "), stderr.String())
	return strings.TrimSpace(string(out))
}

// The published image runs local event-stream serve with the gate set by env, a host
// client reaches it through the mapped port, and docker stop exits 0.
func TestDevListenInADockerContainer(t *testing.T) {
	image := os.Getenv(devListenImageEnv)
	if image == "" {
		t.Skipf("set %s to an image built with make docker-build", devListenImageEnv)
	}
	t.Parallel()

	id := docker(t, "run", "-d", "-p", "127.0.0.1::4321",
		"-e", "RUDDERSTACK_CLI_EXPERIMENTAL=true",
		"-e", "RUDDERSTACK_X_LOCAL_EVENT_STREAM=true",
		"-e", "RUDDERSTACK_CLI_TELEMETRY_DISABLED=true",
		image, "local", "event-stream", "serve", "--bind", "0.0.0.0", "--port", "4321")
	t.Cleanup(func() { _ = exec.Command("docker", "rm", "-fv", id).Run() })

	// "127.0.0.1:55001", the first line when docker maps IPv4 and IPv6.
	addr := strings.SplitN(docker(t, "port", id, "4321/tcp"), "\n", 2)[0]
	_, port, ok := strings.Cut(addr, ":")
	require.True(t, ok, addr)

	// A port that accepts and never answers must not outlive the deadline.
	client := &http.Client{Timeout: 2 * time.Second}
	get := func(host, path string) (int, []byte) {
		req, err := http.NewRequest(http.MethodGet, "http://"+addr+path, nil)
		require.NoError(t, err)
		req.Host = host
		resp, err := client.Do(req)
		if err != nil {
			return 0, nil
		}
		defer func() { _ = resp.Body.Close() }()
		body, _ := io.ReadAll(resp.Body)
		return resp.StatusCode, body
	}

	var info struct {
		ServerID string `json:"serverId"`
		Bind     string `json:"bind"`
		Port     int    `json:"port"`
	}
	// An image without local event-stream serve prints an error and exits: fail at once.
	for deadline := time.Now().Add(20 * time.Second); ; time.Sleep(100 * time.Millisecond) {
		code, body := get("localhost:"+port, "/_local/v1/info")
		if code == http.StatusOK && json.Unmarshal(body, &info) == nil {
			break
		}
		running := docker(t, "inspect", "-f", "{{.State.Running}}", id) == "true"
		if !running || time.Now().After(deadline) {
			require.FailNow(t, "no listener", "running=%t; logs: %s", running, dockerLogs(id))
		}
	}
	require.NotEmpty(t, info.ServerID)
	require.Equal(t, "0.0.0.0", info.Bind)
	require.Equal(t, 4321, info.Port)

	// A host client sends the mapped port in Host. The check compares names
	// only, so both loopback forms pass and a foreign name does not.
	for host, want := range map[string]int{
		"127.0.0.1:" + port: http.StatusOK,
		"localhost:" + port: http.StatusOK,
		"evil.example":      http.StatusForbidden,
	} {
		code, _ := get(host, "/_local/v1/info")
		require.Equal(t, want, code, host)
	}

	req, err := http.NewRequest(http.MethodPost, "http://"+addr+"/v1/track",
		strings.NewReader(`{"event":"Order Completed","userId":"u1"}`))
	require.NoError(t, err)
	req.SetBasicAuth("dev", "")
	resp, err := client.Do(req)
	require.NoError(t, err)
	body, _ := io.ReadAll(resp.Body)
	require.NoError(t, resp.Body.Close())
	require.Equal(t, "ok", string(body))

	code, body := get("localhost:"+port, "/_local/v1/events?view=counts&serverId="+info.ServerID)
	require.Equal(t, http.StatusOK, code, string(body))
	var counts struct {
		Summary struct {
			ByEvent map[string]int `json:"byEvent"`
		} `json:"summary"`
	}
	require.NoError(t, json.Unmarshal(body, &counts), string(body))
	require.Equal(t, 1, counts.Summary.ByEvent["Order Completed"])

	// rudder-cli runs as PID 1 and handles SIGTERM itself.
	docker(t, "stop", "-t", "10", id)
	logs := dockerLogs(id)
	require.Equal(t, "0", docker(t, "inspect", "-f", "{{.State.ExitCode}}", id), logs)
	require.Contains(t, logs, "warning: listening on 0.0.0.0:4321")
}

func dockerLogs(id string) string {
	out, _ := exec.Command("docker", "logs", id).CombinedOutput()
	return string(out)
}
