package dev

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/rudderlabs/rudder-iac/cli/pkg/exp/devlisten"
)

func TestDevCommandIsHiddenAndGated(t *testing.T) {
	t.Parallel()

	cmd := NewCmdDev(func() bool { return false })
	require.True(t, cmd.Hidden)

	cmd.SetArgs([]string{"events", "--url", "http://127.0.0.1:1"})
	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)
	err := cmd.Execute()

	require.Error(t, err)
	require.Contains(t, err.Error(), "RUDDERSTACK_CLI_EXPERIMENTAL=true")
	require.Contains(t, err.Error(), "RUDDERSTACK_X_DEV_LISTEN=true")
}

func TestListenPrintsOneReadyLineAndStopsOnCancel(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithCancel(context.Background())
	stdoutR, stdoutW := io.Pipe()
	var stderr syncBuffer

	done := make(chan error, 1)
	go func() {
		done <- runListen(ctx, stdoutW, &stderr, listenOptions{port: 0, bind: "127.0.0.1"})
		_ = stdoutW.Close()
	}()

	line, err := bufio.NewReader(stdoutR).ReadString('\n')
	require.NoError(t, err)
	var ready devlisten.Ready
	require.NoError(t, json.Unmarshal([]byte(line), &ready))
	require.True(t, ready.Ready)
	require.True(t, strings.HasPrefix(ready.URL, "http://127.0.0.1:"))

	resp, err := http.Get(ready.URL + "/_dev/v1/info") //nolint:noctx // test
	require.NoError(t, err)
	resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)

	cancel()
	select {
	case err := <-done:
		require.NoError(t, err)
	case <-time.After(5 * time.Second):
		t.Fatal("listen did not stop on cancel")
	}
	rest, _ := io.ReadAll(stdoutR)
	require.Empty(t, rest, "stdout carries the ready line only")
	require.NotContains(t, stderr.String(), "warning")
	require.Contains(t, stderr.String(), "dev listen stopped")
}

func TestListenWarnsOnNonLoopbackBind(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	var stdout, stderr syncBuffer

	require.NoError(t, runListen(ctx, &stdout, &stderr, listenOptions{port: 0, bind: "0.0.0.0"}))

	require.Contains(t, stderr.String(), "warning: listening on 0.0.0.0:")
	require.Contains(t, stderr.String(), "The query API has no authentication.")
	require.Contains(t, stdout.String(), `"bind":"0.0.0.0"`)
	require.Contains(t, stdout.String(), `"url":"http://127.0.0.1:`)
}

func TestEventsPrintsTheEnvelope(t *testing.T) {
	t.Parallel()
	srv, err := devlisten.Start(context.Background())
	require.NoError(t, err)
	t.Cleanup(func() { _ = srv.Close(context.Background()) })

	req, err := http.NewRequest(http.MethodPost, srv.URL()+"/v1/track", strings.NewReader(`{"event":"A","userId":"u1"}`))
	require.NoError(t, err)
	req.SetBasicAuth("dev", "")
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	resp.Body.Close()

	var stdout bytes.Buffer
	cmd := NewCmdDev(func() bool { return true })
	cmd.SetArgs([]string{"events", "--url", srv.URL(), "--since", "0", "--event", "A", "--wait", "1s", "--min", "1", "--json"})
	cmd.SetOut(&stdout)
	require.NoError(t, cmd.Execute())

	var page map[string]any
	require.NoError(t, json.Unmarshal(stdout.Bytes(), &page))
	require.Equal(t, float64(1), page["cursor"])
	events := page["events"].([]any)
	require.Len(t, events, 1)
	require.Nil(t, events[0].(map[string]any)["anonymousId"], "nulls survive the client round trip")
}

func TestEventsPrintsAPIErrorAsJSON(t *testing.T) {
	t.Parallel()
	srv, err := devlisten.Start(context.Background())
	require.NoError(t, err)
	t.Cleanup(func() { _ = srv.Close(context.Background()) })

	var stdout bytes.Buffer
	cmd := NewCmdDev(func() bool { return true })
	cmd.SetArgs([]string{"events", "--url", srv.URL(), "--server-id", "0000000000000000", "--json"})
	cmd.SetOut(&stdout)
	cmd.SetErr(io.Discard)
	err = cmd.Execute()

	require.Error(t, err)
	var body map[string]map[string]any
	require.NoError(t, json.Unmarshal(stdout.Bytes(), &body))
	require.Equal(t, "server_changed", body["error"]["code"])
	require.Contains(t, body["error"], "param")
	require.Nil(t, body["error"]["param"], "an unused param is null, as on the wire")
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
