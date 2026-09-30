package dev

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/rudderlabs/rudder-iac/cli/internal/cmd/cmderrors"
	"github.com/rudderlabs/rudder-iac/cli/pkg/exp/devlisten"
	"github.com/rudderlabs/rudder-iac/cli/pkg/exp/devlisten/devlistentest"
)

func testDeps(t *testing.T) Deps {
	t.Helper()
	dir := t.TempDir()
	return Deps{
		Enabled:   func() bool { return true },
		DevURL:    func() string { return "" },
		ConfigDir: func() string { return dir },
	}
}

func runDev(t *testing.T, args ...string) (string, string, error) {
	t.Helper()
	return runDevWith(t, testDeps(t), args...)
}

func runDevWith(t *testing.T, deps Deps, args ...string) (string, string, error) {
	t.Helper()
	var stdout, stderr bytes.Buffer
	cmd := NewCmdDev(deps)
	cmd.SetArgs(args)
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	err := cmd.ExecuteContext(context.Background())
	return stdout.String(), stderr.String(), err
}

func track(t *testing.T, s *devlisten.Server, body string) {
	t.Helper()
	req, err := http.NewRequest(http.MethodPost, s.URL()+"/v1/track", strings.NewReader(body))
	require.NoError(t, err)
	req.SetBasicAuth("dev", "")
	resp, err := testHTTP.Do(req)
	require.NoError(t, err)
	_, _ = io.Copy(io.Discard, resp.Body)
	resp.Body.Close()
}

var testHTTP = &http.Client{Transport: &http.Transport{DisableKeepAlives: true}}

// errorObject decodes the one JSON error object a machine-mode failure
// prints on stderr.
func errorObject(t *testing.T, stderr string) map[string]any {
	t.Helper()
	var body map[string]map[string]any
	require.NoError(t, json.Unmarshal([]byte(stderr), &body), stderr)
	return body["error"]
}

func TestDevCommandIsHiddenAndGated(t *testing.T) {
	t.Parallel()
	deps := testDeps(t)
	deps.Enabled = func() bool { return false }

	cmd := NewCmdDev(deps)
	require.True(t, cmd.Hidden)
	_, _, err := runDevWith(t, deps, "events", "list", "--url", "http://127.0.0.1:1")

	require.Error(t, err)
	require.Contains(t, err.Error(), "RUDDERSTACK_CLI_EXPERIMENTAL=true")
	require.Contains(t, err.Error(), "RUDDERSTACK_X_DEV_LISTEN=true")
}

func TestEveryDevCommandHasHelp(t *testing.T) {
	t.Parallel()
	for _, c := range allCommands(NewCmdDev(testDeps(t))) {
		require.NotEmpty(t, c.Short, c.CommandPath())
		require.NotEmpty(t, c.Long, c.CommandPath())
		require.NotEmpty(t, c.Example, c.CommandPath())
		require.NotContains(t, c.Long+c.Example, "rudder-cli docs", "help points to --help until DEX-1011")
	}
}

// listenOnce runs dev listen in the background and returns its ready line.
func listenOnce(t *testing.T, deps Deps, opts listenOptions) (devlisten.Ready, *syncBuffer, func() error) {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	stdoutR, stdoutW := io.Pipe()
	stderr := &syncBuffer{}
	done := make(chan error, 1)
	go func() {
		done <- runListen(ctx, stdoutW, stderr, deps, opts)
		_ = stdoutW.Close()
	}()
	line, err := bufio.NewReader(stdoutR).ReadString('\n')
	require.NoError(t, err, stderr.String())
	var ready devlisten.Ready
	require.NoError(t, json.Unmarshal([]byte(line), &ready))
	stop := sync.OnceValue(func() error {
		cancel()
		select {
		case err := <-done:
			return err
		case <-time.After(5 * time.Second):
			t.Error("listen did not stop")
			return nil
		}
	})
	return ready, stderr, stop
}

func TestListenWritesTheStateFileThenTheReadyLine(t *testing.T) {
	t.Parallel()
	deps := testDeps(t)
	path := filepath.Join(deps.ConfigDir(), "dev-listen.json")

	ready, stderr, stop := listenOnce(t, deps, listenOptions{bind: "127.0.0.1"})

	require.NotNil(t, ready.StateFile)
	require.Equal(t, path, *ready.StateFile)
	st, err := os.Stat(path)
	require.NoError(t, err)
	require.Equal(t, os.FileMode(0o600), st.Mode().Perm())
	stored, err := os.ReadFile(path)
	require.NoError(t, err)
	var fromFile devlisten.Ready
	require.NoError(t, json.Unmarshal(stored, &fromFile))
	require.Equal(t, ready, fromFile)

	require.NoError(t, stop())
	_, err = os.Stat(path)
	require.True(t, os.IsNotExist(err), "the server removes its own state file")
	require.Empty(t, stderr.String(), "no progress without a terminal")
}

func TestListenRefusesASecondServerOnALiveStateFile(t *testing.T) {
	t.Parallel()
	deps := testDeps(t)
	ready, _, stop := listenOnce(t, deps, listenOptions{bind: "127.0.0.1"})
	t.Cleanup(func() { _ = stop() })

	var stdout, stderr bytes.Buffer
	err := runListen(context.Background(), &stdout, &stderr, deps, listenOptions{bind: "127.0.0.1"})

	require.Error(t, err)
	require.Empty(t, stdout.String())
	errObj := errorObject(t, stderr.String())
	require.Equal(t, "already_running", errObj["code"])
	require.Contains(t, errObj["message"], ready.URL)
	require.Equal(t, "rudder-cli dev stop", errObj["next"])
}

func TestListenReplacesAStaleStateFile(t *testing.T) {
	t.Parallel()
	deps := testDeps(t)
	stale := `{"ready":true,"serverId":"0000000000000000","url":"http://127.0.0.1:1"}`
	require.NoError(t, os.WriteFile(filepath.Join(deps.ConfigDir(), "dev-listen.json"), []byte(stale), 0o600))

	ready, _, stop := listenOnce(t, deps, listenOptions{bind: "127.0.0.1"})
	t.Cleanup(func() { _ = stop() })

	require.NotEqual(t, "0000000000000000", ready.ServerID)
}

func TestListenProgressOnATerminal(t *testing.T) {
	t.Parallel()
	ready, stderr, stop := listenOnce(t, testDeps(t), listenOptions{bind: "127.0.0.1", noStateFile: true, progress: true})
	s := devlisten.NewClient(ready.URL)

	req, err := http.NewRequest(http.MethodPost, ready.URL+"/v1/track", strings.NewReader(`{"event":"A\u001b[31m","userId":"u1"}`))
	require.NoError(t, err)
	req.SetBasicAuth("dev", "")
	resp, err := testHTTP.Do(req)
	require.NoError(t, err)
	resp.Body.Close()
	_, err = s.Info(context.Background())
	require.NoError(t, err)
	require.NoError(t, stop())

	out := stderr.String()
	require.Contains(t, out, "dev listen: capturing at "+ready.URL)
	require.Contains(t, out, "SDK setup: dataPlaneUrl="+ready.URL+" configUrl="+ready.URL+" writeKey=dev")
	require.Contains(t, out, "Read events from another shell: rudder-cli dev events list --since 0")
	require.Contains(t, out, `seq=1 route=/v1/track type=track event="A\x1b[31m" statusCode=200 | rudder-cli dev requests show 1`)
	require.NotContains(t, out, "\x1b[31m", "control characters are escaped")
	require.Contains(t, out, "dev listen stopped (signal, 1 requests, 0 failed, up ")
}

func TestListenWarnsOnNonLoopbackBind(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	var stdout, stderr syncBuffer

	require.NoError(t, runListen(ctx, &stdout, &stderr, testDeps(t), listenOptions{bind: "0.0.0.0", noStateFile: true}))

	require.Contains(t, stderr.String(), "warning: listening on 0.0.0.0:")
	require.Contains(t, stderr.String(), "The query API has no authentication.")
	require.Contains(t, stdout.String(), `"bind":"0.0.0.0"`)
}

func TestEventsListPrintsTheServerBytes(t *testing.T) {
	t.Parallel()
	s := devlistentest.Start(t)
	track(t, s, `{"event":"A","userId":"u1"}`)

	stdout, stderr, err := runDev(t, "events", "list", "--url", s.URL(), "--since", "0", "--event", "A", "--wait", "1s",
		"--json")

	require.NoError(t, err)
	require.Empty(t, stderr)
	var page map[string]any
	require.NoError(t, json.Unmarshal([]byte(stdout), &page))
	require.Equal(t, "compact", page["view"])
	events := page["events"].([]any)
	require.Len(t, events, 1)
	require.Nil(t, events[0].(map[string]any)["anonymousId"], "nulls survive the client round trip")
	require.True(t, strings.HasSuffix(stdout, "}\n"))
}

func TestMachineModeErrorGoesToStderrOnly(t *testing.T) {
	t.Parallel()
	s := devlistentest.Start(t)

	stdout, stderr, err := runDev(t, "events", "list", "--url", s.URL(), "--server-id", "0000000000000000", "--json")

	var silent *cmderrors.SilentError
	require.ErrorAs(t, err, &silent)
	require.Empty(t, stdout)
	errObj := errorObject(t, stderr)
	require.Equal(t, "server_changed", errObj["code"])
	require.Nil(t, errObj["param"], "an unused param is null, as on the wire")
	require.Equal(t, "rudder-cli dev info --json", errObj["next"])
}

func TestHumanModeErrorHasNextLine(t *testing.T) {
	t.Parallel()
	s := devlistentest.Start(t)

	stdout, stderr, err := runDev(t, "events", "list", "--url", s.URL(), "--view", "raw")

	require.Error(t, err)
	require.Empty(t, stdout)
	require.Contains(t, stderr, "Error:")
	require.Contains(t, stderr, "invalid_parameter: view:")
	require.Contains(t, stderr, "Next: rudder-cli dev events list --help")
}

func TestHumanTables(t *testing.T) {
	t.Parallel()
	s := devlistentest.Start(t)
	track(t, s, `{"event":"Order Completed","userId":"u1"}`)

	stdout, _, err := runDev(t, "events", "list", "--url", s.URL())
	require.NoError(t, err)
	require.Contains(t, stdout, "SEQ")
	require.Contains(t, stdout, "Order Completed")

	stdout, _, err = runDev(t, "requests", "list", "--url", s.URL())
	require.NoError(t, err)
	require.Contains(t, stdout, "OUTCOME")
	require.Contains(t, stdout, "accepted")

	stdout, _, err = runDev(t, "summary", "--url", s.URL())
	require.NoError(t, err)
	require.Contains(t, stdout, "all_accepted")
	require.Contains(t, stdout, "Next: rudder-cli dev events list --since 0 --view summary --json")
}

func TestJQFiltersASuccessfulPage(t *testing.T) {
	t.Parallel()
	s := devlistentest.Start(t)
	track(t, s, `{"event":"A","userId":"u1","properties":{"n":1}}`)
	track(t, s, `{"event":"B","userId":"u1","properties":{"n":2}}`)

	stdout, stderr, err := runDev(t, "events", "list", "--url", s.URL(), "--fields", "properties", "--json",
		"--jq", ".events[].properties")
	require.NoError(t, err)
	require.Equal(t, 1, strings.Count(stderr, "\n"), "stderr holds the cursor trailer only")
	require.Equal(t, "{\"n\":1}\n{\"n\":2}\n", stdout)

	stdout, _, err = runDev(t, "events", "list", "--url", s.URL(), "--json", "--jq", ".events[].event")
	require.NoError(t, err)
	require.Equal(t, "A\nB\n", stdout, "strings print raw")

	_, stderr, err = runDev(t, "events", "list", "--url", s.URL(), "--event", "none", "--wait", "100ms", "--json",
		"--jq", ".events")
	require.NoError(t, err, "timedOut exits 0")
	require.Contains(t, stderr, "note: timedOut")
}

func TestJQNeedsJSONAndAValidExpression(t *testing.T) {
	t.Parallel()
	s := devlistentest.Start(t)

	_, stderr, err := runDev(t, "events", "list", "--url", s.URL(), "--jq", ".events")
	require.Error(t, err)
	require.Contains(t, stderr, "--jq needs --json")

	_, stderr, err = runDev(t, "summary", "--url", s.URL(), "--json", "--jq", ".[")
	require.Error(t, err)
	require.Equal(t, "usage", errorObject(t, stderr)["code"])

	stdout, _, err := runDev(t, "info", "--url", s.URL(), "--json", "--jq", "$ENV | length")
	require.NoError(t, err)
	require.Equal(t, "0\n", stdout, "jq sees no environment")
}

func TestOutputLimitKeepsThePageAndFails(t *testing.T) {
	t.Parallel()
	s := devlistentest.Start(t)
	track(t, s, `{"event":"A","userId":"u1","properties":{"text":"`+strings.Repeat("x", 500)+`"}}`)

	stdout, stderr, err := runDev(t, "events", "list", "--url", s.URL(), "--max-bytes", "400", "--json")

	require.Error(t, err)
	var page devlisten.Page
	require.NoError(t, json.Unmarshal([]byte(stdout), &page))
	require.Empty(t, page.Events)
	require.NotNil(t, page.Truncated)
	errObj := errorObject(t, stderr)
	require.Equal(t, "output_limit", errObj["code"])
	require.Equal(t, page.Truncated.Next, errObj["next"])
}

func TestRequestsShow(t *testing.T) {
	t.Parallel()
	s := devlistentest.Start(t)
	track(t, s, `{"event":"A","userId":"u1"}`)

	stdout, _, err := runDev(t, "requests", "show", "1", "--url", s.URL(), "--json")
	require.NoError(t, err)
	require.Contains(t, stdout, `"body":"{\"event\":\"A\",\"userId\":\"u1\"}"`)

	_, stderr, err := runDev(t, "requests", "show", "9", "--url", s.URL(), "--json")
	require.Error(t, err)
	require.Equal(t, "not_found", errorObject(t, stderr)["code"])

	_, stderr, err = runDev(t, "requests", "show", "--url", s.URL(), "--json")
	require.Error(t, err)
	require.Equal(t, "usage", errorObject(t, stderr)["code"])
}

func TestDiscovery(t *testing.T) {
	t.Parallel()

	t.Run("no source", func(t *testing.T) {
		t.Parallel()
		_, stderr, err := runDev(t, "summary", "--json")
		require.Error(t, err)
		errObj := errorObject(t, stderr)
		require.Equal(t, "server_unreachable", errObj["code"])
		require.Equal(t, "rudder-cli dev listen --detach", errObj["next"])
	})

	t.Run("env url", func(t *testing.T) {
		t.Parallel()
		s := devlistentest.Start(t)
		deps := testDeps(t)
		deps.DevURL = func() string { return s.URL() }
		stdout, _, err := runDevWith(t, deps, "info", "--json")
		require.NoError(t, err)
		require.Contains(t, stdout, s.Ready().ServerID)
	})

	t.Run("state file verified by serverId", func(t *testing.T) {
		t.Parallel()
		s := devlistentest.Start(t)
		deps := testDeps(t)
		ready := s.Ready()
		require.NoError(t, writeState(stateFilePath(deps.ConfigDir()), ready))
		stdout, _, err := runDevWith(t, deps, "summary", "--json")
		require.NoError(t, err)
		require.Contains(t, stdout, `"serverId":"`+ready.ServerID+`"`)
	})

	t.Run("stale state file", func(t *testing.T) {
		t.Parallel()
		s := devlistentest.Start(t)
		deps := testDeps(t)
		ready := s.Ready()
		ready.ServerID = "0000000000000000"
		path := stateFilePath(deps.ConfigDir())
		require.NoError(t, writeState(path, ready))
		_, stderr, err := runDevWith(t, deps, "events", "list", "--json")
		require.Error(t, err)
		errObj := errorObject(t, stderr)
		require.Equal(t, "stale_state", errObj["code"])
		require.Contains(t, errObj["message"], path)
		_, statErr := os.Stat(path)
		require.NoError(t, statErr, "a client never deletes the state file")
	})
}

func TestStop(t *testing.T) {
	t.Parallel()

	t.Run("running server from the state file", func(t *testing.T) {
		t.Parallel()
		deps := testDeps(t)
		ready, _, stop := listenOnce(t, deps, listenOptions{bind: "127.0.0.1"})
		t.Cleanup(func() { _ = stop() })

		stdout, _, err := runDevWith(t, deps, "stop", "--json")

		require.NoError(t, err)
		require.JSONEq(t, `{"stopped":true,"serverId":"`+ready.ServerID+`"}`, stdout)
		require.NoError(t, stop(), "listen returns once the server stopped itself")
	})

	t.Run("nothing running", func(t *testing.T) {
		t.Parallel()
		stdout, _, err := runDev(t, "stop", "--json")
		require.NoError(t, err)
		require.JSONEq(t, `{"stopped":false,"reason":"not_running"}`, stdout)
	})

	t.Run("stale state file is removed", func(t *testing.T) {
		t.Parallel()
		deps := testDeps(t)
		path := stateFilePath(deps.ConfigDir())
		require.NoError(t, writeState(path, devlisten.Ready{ServerID: "0000000000000000", URL: "http://127.0.0.1:1"}))
		stdout, _, err := runDevWith(t, deps, "stop", "--json")
		require.NoError(t, err)
		require.JSONEq(t, `{"stopped":false,"reason":"not_running"}`, stdout)
		_, statErr := os.Stat(path)
		require.True(t, os.IsNotExist(statErr))
	})
}

func TestSendResetInfo(t *testing.T) {
	t.Parallel()
	s := devlistentest.Start(t)

	stdout, _, err := runDev(t, "send", "--url", s.URL(), "--json")
	require.NoError(t, err)
	require.JSONEq(t, `{"statusCode":200,"body":"ok","seq":1,"route":"/v1/track"}`, stdout)

	stdout, _, err = runDev(t, "info", "--url", s.URL(), "--json", "--jq", ".cursor")
	require.NoError(t, err)
	require.Equal(t, "1\n", stdout)

	stdout, _, err = runDev(t, "reset", "--url", s.URL(), "--json")
	require.NoError(t, err)
	require.JSONEq(t, `{"cursor":1,"removed":{"requests":1,"events":1,"control":0}}`, stdout)

	_, stderr, err := runDev(t, "send", "--url", s.URL(), "--write-key", "", "--json")
	require.Error(t, err)
	require.Equal(t, "send_failed", errorObject(t, stderr)["code"])
}

// syncBuffer lets the test read stderr while runListen writes to it.
type syncBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (b *syncBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.Write(p)
}

func (b *syncBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.String()
}

func TestListenWarnsWhenTheStateFileCannotBeWritten(t *testing.T) {
	t.Parallel()
	deps := testDeps(t)
	blocker := filepath.Join(deps.ConfigDir(), "file")
	require.NoError(t, os.WriteFile(blocker, nil, 0o600))
	deps.ConfigDir = func() string { return blocker }

	ready, stderr, stop := listenOnce(t, deps, listenOptions{bind: "127.0.0.1"})
	require.NoError(t, stop())

	require.Nil(t, ready.StateFile)
	require.Contains(t, stderr.String(), "warning: discovery file unavailable")
}

func TestRequestsShowUsage(t *testing.T) {
	t.Parallel()
	s := devlistentest.Start(t)
	track(t, s, `{"event":"A","userId":"u1","properties":{"text":"`+strings.Repeat("x", 500)+`"}}`)

	for args, code := range map[string]string{
		"requests show x":                 "usage",
		"requests show 1 --jq .seq":       "usage",
		"requests show 1 --max-bytes 200": "output_limit",
		"events list --status-code x":     "usage",
		"requests list --status-code x":   "usage",
		"requests list --jq .":            "usage",
		"summary --jq .":                  "usage",
		"info --jq .":                     "usage",
		"reset --jq .":                    "usage",
		"stop --jq .":                     "usage",
		"send --jq .":                     "usage",
		"requests list --max-bytes 100":   "output_limit",
	} {
		argv := append(strings.Fields(args), "--url", s.URL(), "--json")
		if strings.Contains(args, "--jq") {
			argv = append(strings.Fields(args), "--url", s.URL())
		}
		_, stderr, err := runDev(t, argv...)
		require.Error(t, err, args)
		if strings.Contains(args, "--jq") {
			require.Contains(t, stderr, "--jq needs --json", args)
			continue
		}
		require.Equal(t, code, errorObject(t, lastLine(stderr))["code"], args)
	}

	stdout, _, err := runDev(t, "requests", "show", "1", "--url", s.URL())
	require.NoError(t, err)
	require.Contains(t, stdout, "  \"seq\": 1,", "human mode indents the record")
}

func lastLine(s string) string {
	lines := strings.Split(strings.TrimSpace(s), "\n")
	return lines[len(lines)-1]
}
