package dev

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/rudderlabs/rudder-iac/cli/pkg/exp/devlisten"
	"github.com/rudderlabs/rudder-iac/cli/pkg/exp/devlisten/devlistentest"
)

const childEnv = "DEV_LISTEN_TEST_CHILD_ARGS"

// TestMain lets the test binary act as the detached child: a test spawns
// os.Args[0] with the child's arguments in childEnv.
func TestMain(m *testing.M) {
	if args := os.Getenv(childEnv); args != "" {
		dir := os.Getenv("DEV_LISTEN_TEST_DIR")
		cmd := NewCmdDev(Deps{
			Enabled:   func() bool { return true },
			DevURL:    func() string { return "" },
			ConfigDir: func() string { return dir },
		})
		cmd.SetArgs(strings.Split(args, "\x1f"))
		if err := cmd.Execute(); err != nil {
			os.Exit(1)
		}
		os.Exit(0)
	}
	os.Exit(m.Run())
}

func testSpawner(t *testing.T, listenArgs ...string) spawnFunc {
	t.Helper()
	dir := t.TempDir()
	return func(extra []string) *exec.Cmd {
		args := append(append([]string{"listen"}, listenArgs...), extra...)
		c := exec.Command(os.Args[0])
		c.Env = append(os.Environ(), childEnv+"="+strings.Join(args, "\x1f"), "DEV_LISTEN_TEST_DIR="+dir)
		return c
	}
}

func TestDetachPrintsTheChildReadyLineAndReturns(t *testing.T) {
	t.Parallel()
	var stdout, stderr bytes.Buffer

	start := time.Now()
	err := runDetach(&stdout, &stderr, listenOptions{bind: "127.0.0.1"}, testSpawner(t, "--port", "0"))

	require.NoError(t, err, stderr.String())
	require.Less(t, time.Since(start), 10*time.Second)
	require.Equal(t, 1, strings.Count(stdout.String(), "\n"), "stdout is the ready line only")
	var ready devlisten.Ready
	require.NoError(t, json.Unmarshal(stdout.Bytes(), &ready))
	require.NotEqual(t, os.Getpid(), ready.PID, "the server runs in the child")

	client := devlisten.NewClient(ready.URL)
	info, err := client.Info(context.Background())
	require.NoError(t, err, "the child keeps serving after the parent returns")
	require.Equal(t, ready.ServerID, info.ServerID)
	_, err = client.Shutdown(context.Background())
	require.NoError(t, err)
}

func TestDetachReportsTheChildStartupError(t *testing.T) {
	t.Parallel()
	busy := devlistentest.Start(t)
	var stdout, stderr bytes.Buffer

	err := runDetach(&stdout, &stderr, listenOptions{bind: "127.0.0.1"},
		testSpawner(t, "--port", strconv.Itoa(busy.Port()), "--no-state-file"))

	require.Error(t, err)
	require.Empty(t, stdout.String())
	lines := strings.Split(strings.TrimSpace(stderr.String()), "\n")
	errObj := errorObject(t, lines[len(lines)-1])
	require.Equal(t, "port_in_use", errObj["code"])
	require.Equal(t, "rudder-cli dev listen --port 0", errObj["next"])
}

func TestChildArgsDropDetachAndAddDefaults(t *testing.T) {
	t.Parallel()

	require.Equal(t,
		[]string{"-c", "/tmp/cfg.json", "dev", "listen", "--port", "4321", "--detached-child", "--quiet", "--idle-exit", "30m"},
		childArgs([]string{"-c", "/tmp/cfg.json", "dev", "listen", "--port", "4321", "--detach"}, false))
	require.Equal(t,
		[]string{"dev", "listen", "--idle-exit", "0", "--detached-child", "--quiet"},
		childArgs([]string{"dev", "listen", "--detach=true", "--idle-exit", "0"}, true))
}

func TestDetachRepeatsTheBindWarningAndProgress(t *testing.T) {
	t.Parallel()
	var stdout, stderr bytes.Buffer

	err := runDetach(&stdout, &stderr, listenOptions{bind: "0.0.0.0", progress: true},
		testSpawner(t, "--bind", "0.0.0.0", "--no-state-file"))
	require.NoError(t, err, stderr.String())
	var ready devlisten.Ready
	require.NoError(t, json.Unmarshal(stdout.Bytes(), &ready))
	t.Cleanup(func() {
		// A non-loopback bind refuses /shutdown; stop the child by pid.
		if p, err := os.FindProcess(ready.PID); err == nil {
			_ = p.Kill()
		}
	})

	require.Contains(t, stderr.String(), "warning: listening on 0.0.0.0:")
	require.Contains(t, stderr.String(), "dev listen: running in background at "+ready.URL+"; idle exit 30m0s.")
	require.Contains(t, stderr.String(), "Stop: rudder-cli dev stop --url "+ready.URL)
}
