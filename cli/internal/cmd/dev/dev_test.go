package dev

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
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/rudderlabs/rudder-iac/cli/internal/cmd/cmderrors"
	"github.com/rudderlabs/rudder-iac/cli/internal/cmd/telemetry"
	"github.com/rudderlabs/rudder-iac/cli/pkg/exp/devlisten"
	"github.com/rudderlabs/rudder-iac/cli/pkg/exp/devlisten/devlistentest"
)

func testDeps(t *testing.T) Deps {
	t.Helper()
	return Deps{
		Enabled: func() bool { return true },
		DevURL:  func() string { return "" },
		Track:   func(string, error, ...telemetry.KV) {},
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
	// The root command silences cobra's own printing; each dev command prints
	// its errors.
	cmd.SilenceUsage, cmd.SilenceErrors = true, true
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

func decode(t *testing.T, stdout string) map[string]any {
	t.Helper()
	var page map[string]any
	require.NoError(t, json.Unmarshal([]byte(stdout), &page), stdout)
	return page
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
		if c.Runnable() {
			require.NotEmpty(t, c.Long, c.CommandPath())
			require.NotEmpty(t, c.Example, c.CommandPath())
		}
	}
}

func TestCommandTreeIsTheTrimmedSurface(t *testing.T) {
	t.Parallel()
	var paths []string
	for _, c := range allCommands(NewCmdDev(testDeps(t))) {
		paths = append(paths, c.CommandPath())
	}

	require.ElementsMatch(t, []string{"dev", "dev listen", "dev events", "dev events list", "dev requests",
		"dev requests list", "dev requests show"}, paths)
}

// listenOnce runs dev listen in the background and returns its ready line.
func listenOnce(t *testing.T, opts listenOptions) (devlisten.Ready, *syncBuffer, func() error) {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	stdoutR, stdoutW := io.Pipe()
	stderr := &syncBuffer{}
	done := make(chan error, 1)
	go func() {
		done <- runListen(ctx, stdoutW, stderr, opts)
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
	t.Cleanup(func() { _ = stop() })
	return ready, stderr, stop
}

func TestListenPrintsOneReadyLineAndNothingElseOffATerminal(t *testing.T) {
	t.Parallel()
	ready, stderr, stop := listenOnce(t, listenOptions{bind: "127.0.0.1"})

	require.True(t, ready.Ready)
	require.Equal(t, ready.URL+"/_dev/ui/", ready.UI)
	require.NotZero(t, ready.PID)
	require.NoError(t, stop(), "a signal is a clean stop")
	require.Empty(t, stderr.String(), "no startup note without a terminal")
}

func TestListenStartupNoteOnATerminal(t *testing.T) {
	t.Parallel()
	ready, stderr, _ := listenOnce(t, listenOptions{bind: "127.0.0.1", note: true})

	require.Eventually(t, func() bool { return strings.Contains(stderr.String(), "Read events with") },
		time.Second, 10*time.Millisecond)
	out := stderr.String()
	require.Contains(t, out, "dev listen: capturing at "+ready.URL)
	require.Contains(t, out, "Review in a browser: "+ready.UI)
	require.Contains(t, out, "Read events with: rudder-cli dev events --url "+ready.URL)
}

func TestListenWarnsOnNonLoopbackBind(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	var stdout, stderr syncBuffer

	require.NoError(t, runListen(ctx, &stdout, &stderr, listenOptions{bind: "0.0.0.0"}))

	require.Contains(t, stderr.String(), "warning: listening on 0.0.0.0:")
	require.Contains(t, stdout.String(), `"bind":"0.0.0.0"`)
}

func TestListenOnATakenPortFailsWithPortInUse(t *testing.T) {
	t.Parallel()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	t.Cleanup(func() { ln.Close() })
	port := ln.Addr().(*net.TCPAddr).Port
	var stdout, stderr syncBuffer

	err = runListen(context.Background(), &stdout, &stderr, listenOptions{bind: "127.0.0.1", port: port})

	var silent *cmderrors.SilentError
	require.ErrorAs(t, err, &silent)
	require.Empty(t, stdout.String())
	errObj := errorObject(t, stderr.String())
	require.Equal(t, "port_in_use", errObj["code"])
	require.Equal(t, "rudder-cli dev listen --port 0", errObj["next"])
}

func TestListenWriteKeyRejectsOtherKeys(t *testing.T) {
	t.Parallel()
	ready, _, _ := listenOnce(t, listenOptions{bind: "127.0.0.1", writeKeys: []string{"a-key-of-16-char"}})

	req, err := http.NewRequest(http.MethodPost, ready.URL+"/v1/track", strings.NewReader(`{"event":"A","userId":"u1"}`))
	require.NoError(t, err)
	req.SetBasicAuth("b", "")
	resp, err := testHTTP.Do(req)
	require.NoError(t, err)
	resp.Body.Close()

	require.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	require.Equal(t, "allowlist", ready.WriteKeyPolicy)
}

// Several listeners run side by side: each has its own port and serverId,
// and a client pinned to one gets server_changed from the other.
func TestTwoListenersShareNothing(t *testing.T) {
	t.Parallel()
	first, _, _ := listenOnce(t, listenOptions{bind: "127.0.0.1"})
	second, _, _ := listenOnce(t, listenOptions{bind: "127.0.0.1"})

	require.NotEqual(t, first.Port, second.Port)
	require.NotEqual(t, first.ServerID, second.ServerID)
	_, stderr, err := runDev(t, "events", "--url", second.URL, "--server-id", first.ServerID, "--json")
	require.Error(t, err)
	require.Equal(t, "server_changed", errorObject(t, stderr)["code"])
}

func TestEventsIsTheSummary(t *testing.T) {
	t.Parallel()
	s := devlistentest.Start(t)
	track(t, s, `{"event":"A","userId":"u1"}`)

	stdout, stderr, err := runDev(t, "events", "--url", s.URL(), "--event", "A", "--event", "Missing", "--json")

	require.NoError(t, err)
	require.Empty(t, stderr)
	page := decode(t, stdout)
	require.Equal(t, "counts", page["view"])
	require.Equal(t, []any{}, page["events"])
	require.Equal(t, map[string]any{"A": float64(1), "Missing": float64(0)}, page["summary"].(map[string]any)["byEvent"])
}

func TestEventsListFillsEventsAndTotals(t *testing.T) {
	t.Parallel()
	s := devlistentest.Start(t)
	track(t, s, `{"event":"A","userId":"u1"}`)

	stdout, _, err := runDev(t, "events", "list", "--url", s.URL(), "--json")

	require.NoError(t, err)
	page := decode(t, stdout)
	require.Equal(t, "list", page["view"])
	require.Len(t, page["events"], 1)
	require.Equal(t, float64(1), page["total"])
	require.Nil(t, page["summary"], "dev events is the summary")
	require.True(t, strings.HasSuffix(stdout, "}\n"))
}

func TestEventsListFieldsReturnsTheProperties(t *testing.T) {
	t.Parallel()
	s := devlistentest.Start(t)
	track(t, s, `{"event":"X","userId":"u1","properties":{"total":42}}`)

	stdout, _, err := runDev(t, "events", "list", "--url", s.URL(), "--event", "X", "--fields", "properties", "--json")

	require.NoError(t, err)
	events := decode(t, stdout)["events"].([]any)
	require.Equal(t, map[string]any{"total": float64(42)}, events[0].(map[string]any)["properties"])
}

func TestEventsListFieldsWithViewNamesTheCommandWithoutView(t *testing.T) {
	t.Parallel()
	s := devlistentest.Start(t)

	_, stderr, err := runDev(t, "events", "list", "--url", s.URL(), "--fields", "properties", "--view", "full", "--json")

	require.Error(t, err)
	require.Equal(t, "rudder-cli dev events list --since 0 --fields properties --json", errorObject(t, stderr)["next"])
}

// A --wait that runs out is an answer, not an error: exit 0, timedOut true.
func TestWaitThatRunsOutExitsZero(t *testing.T) {
	t.Parallel()
	s := devlistentest.Start(t)

	stdout, _, err := runDev(t, "events", "--url", s.URL(), "--event", "never", "--wait", "100ms", "--json")

	require.NoError(t, err)
	require.Equal(t, true, decode(t, stdout)["timedOut"])
}

func TestMissingURLIsAUsageErrorThatNamesURL(t *testing.T) {
	t.Parallel()

	_, stderr, err := runDev(t, "events", "list", "--since", "3", "--json")

	var silent *cmderrors.SilentError
	require.ErrorAs(t, err, &silent)
	errObj := errorObject(t, stderr)
	require.Equal(t, "usage", errObj["code"])
	require.Equal(t, "rudder-cli dev events list --json --since 3 --url URL", errObj["next"])
}

func TestEnvURLIsUsed(t *testing.T) {
	t.Parallel()
	s := devlistentest.Start(t)
	deps := testDeps(t)
	deps.DevURL = func() string { return s.URL() }

	stdout, _, err := runDevWith(t, deps, "events", "--json")

	require.NoError(t, err)
	require.Equal(t, s.Ready().ServerID, decode(t, stdout)["serverId"])
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
	require.Equal(t, "rudder-cli dev events list --json", errObj["next"])
}

func TestTableUnlessJSON(t *testing.T) {
	t.Parallel()
	s := devlistentest.Start(t)
	track(t, s, `{"event":"Order Completed","userId":"u1"}`)

	stdout, _, err := runDev(t, "events", "list", "--url", s.URL())
	require.NoError(t, err)
	require.Contains(t, stdout, "1 of 1 matching events, cursor 1")
	require.NotContains(t, stdout, "requests:")
	require.Contains(t, stdout, "SEQ")
	require.Contains(t, stdout, "WRITE KEY")
	require.Contains(t, stdout, "Order Completed")

	stdout, _, err = runDev(t, "events", "--url", s.URL())
	require.NoError(t, err)
	require.Contains(t, stdout, "all_accepted")
	require.NotContains(t, stdout, "SEQ")

	stdout, _, err = runDev(t, "requests", "list", "--url", s.URL())
	require.NoError(t, err)
	require.Contains(t, stdout, "OUTCOME")
}

func TestHumanModeErrorHasNextLine(t *testing.T) {
	t.Parallel()
	s := devlistentest.Start(t)

	stdout, stderr, err := runDev(t, "events", "list", "--url", s.URL(), "--view", "raw")

	require.Error(t, err)
	require.Empty(t, stdout)
	require.Contains(t, stderr, "invalid_parameter: view:")
	require.Contains(t, stderr, "Next: rudder-cli dev events list --help")
}

func TestOutputLimitKeepsThePageAndFails(t *testing.T) {
	t.Parallel()
	s := devlistentest.Start(t)
	track(t, s, `{"event":"A","userId":"u1","properties":{"text":"`+strings.Repeat("x", 500)+`"}}`)

	stdout, stderr, err := runDev(t, "events", "list", "--url", s.URL(), "--view", "compact", "--max-bytes", "1200",
		"--json")

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

	stdout, _, err = runDev(t, "requests", "show", "1", "--url", s.URL())
	require.NoError(t, err)
	require.Contains(t, stdout, "  \"seq\": 1,", "human mode indents the record")
}

func TestReadErrorsExitOne(t *testing.T) {
	t.Parallel()
	s := devlistentest.Start(t)
	track(t, s, `{"event":"A","userId":"u1","properties":{"text":"`+strings.Repeat("x", 500)+`"}}`)

	for args, code := range map[string]string{
		"requests show x":                      "usage",
		"requests show 1 --max-bytes 200":      "output_limit",
		"events list --status-code x":          "usage",
		"requests list --status-code x":        "usage",
		"requests list --max-bytes 100":        "output_limit",
		"events --server-id 0000000000000000":  "server_changed",
		"events --since " + strconv.Itoa(1<<9): "",
	} {
		_, stderr, err := runDev(t, append(strings.Fields(args), "--url", s.URL(), "--json")...)
		if code == "" {
			require.NoError(t, err, args)
			continue
		}
		require.Error(t, err, args)
		require.Equal(t, code, errorObject(t, lastLine(stderr))["code"], args)
	}
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

func lastLine(s string) string {
	lines := strings.Split(strings.TrimSpace(s), "\n")
	return lines[len(lines)-1]
}

func TestEventsListTableShowsIntegerSeqAndFieldColumns(t *testing.T) {
	t.Parallel()
	s := devlistentest.Start(t)
	track(t, s, `{"event":"A","userId":"u1","properties":{"total":42}}`)

	stdout, _, err := runDev(t, "events", "list", "--url", s.URL(), "--fields", "properties.total")

	require.NoError(t, err)
	require.Contains(t, stdout, "SEQ  TYPE   EVENT  properties.total")
	require.Contains(t, stdout, "1    track  A      42")
	require.NotContains(t, stdout, "1.0")
}

func TestSinceTakesADuration(t *testing.T) {
	t.Parallel()
	s := devlistentest.Start(t)
	track(t, s, `{"event":"A","userId":"u1"}`)

	stdout, _, err := runDev(t, "events", "list", "--url", s.URL(), "--since", "5m", "--json")

	require.NoError(t, err)
	require.Len(t, decode(t, stdout)["events"], 1)
}
