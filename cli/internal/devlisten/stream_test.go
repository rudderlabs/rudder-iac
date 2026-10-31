package devlisten

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/rudderlabs/rudder-iac/cli/internal/devlisten/ingest"
)

// syncBuffer is safe for the handler goroutines and the test to share.
type syncBuffer struct {
	mu sync.Mutex
	b  strings.Builder
}

func (s *syncBuffer) Write(p []byte) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.b.Write(p)
}

func (s *syncBuffer) String() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.b.String()
}

func post(t *testing.T, url, key, body string) int {
	t.Helper()
	req, err := http.NewRequest(http.MethodPost, url, strings.NewReader(body))
	require.NoError(t, err)
	req.SetBasicAuth(key, "")
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	_, _ = io.Copy(io.Discard, resp.Body)
	_ = resp.Body.Close()
	return resp.StatusCode
}

func TestWithoutStoreEventsGoToOutAndNothingIsKept(t *testing.T) {
	t.Parallel()
	var out, log syncBuffer
	s, err := Start(Config{WriteKeys: []string{"web"}, Out: &out, Log: &log})
	require.NoError(t, err)
	t.Cleanup(func() { _ = s.Close(context.Background()) })
	url := s.Ready().URL

	// A pretty-printed batch still gives one line per event.
	require.Equal(t, http.StatusOK, post(t, url+"/v1/batch", "web",
		"{\"batch\": [\n {\"type\": \"track\", \"event\": \"A\", \"userId\": \"u1\"},\n {\"type\": \"track\", \"event\": \"B\", \"userId\": \"u1\"}\n]}"))
	require.Equal(t, http.StatusUnauthorized, post(t, url+"/v1/track", "other", `{"event":"C","userId":"u1"}`))

	require.Equal(t,
		`{"type":"track","event":"A","userId":"u1"}`+"\n"+`{"type":"track","event":"B","userId":"u1"}`+"\n",
		out.String())
	require.Contains(t, log.String(), "refused: 401 POST /v1/track: ")
	require.Equal(t, 1, strings.Count(log.String(), "\n"))

	ready := s.Ready()
	require.Nil(t, ready.Cursor)
	require.Empty(t, ready.UI)

	resp, err := http.Get(url + "/_dev/v1/info")
	require.NoError(t, err)
	body, _ := io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	require.Equal(t, http.StatusNotFound, resp.StatusCode)
	require.Contains(t, string(body), `"code":"store_disabled"`)
}

func TestWithStoreEventsGoToOutAndTheStore(t *testing.T) {
	t.Parallel()
	var out syncBuffer
	s := start(t, Config{Out: &out})
	url := s.Ready().URL

	require.Equal(t, http.StatusOK, post(t, url+"/v1/track", "k", `{"event":"A","userId":"u1"}`))

	require.Equal(t, `{"event":"A","userId":"u1"}`+"\n", out.String())
	var info map[string]any
	getJSON(t, url+"/_dev/v1/info", &info)
	require.Equal(t, float64(1), info["store"].(map[string]any)["requests"])
}

func accepted(events ...string) *ingest.Capture {
	c := &ingest.Capture{
		Kind: "ingestion", StatusCode: http.StatusOK,
		ReceivedAt: time.Date(2026, 10, 16, 12, 0, 0, 5000, time.UTC),
	}
	for _, e := range events {
		c.Events = append(c.Events, ingest.Event{Message: json.RawMessage(e)})
	}
	return c
}

func TestStampedLinesKeepTheEventAfterTheDevObject(t *testing.T) {
	t.Parallel()
	var out strings.Builder
	s := newStreamSink(&out, io.Discard, "9f3ac1d2b7e4c601")

	s.Capture(accepted(`{ "event": "A",
 "userId": "u1" }`, `{}`))
	s.Capture(accepted(`{"event":"B"}`))

	const dev = `{"_dev":{"serverId":"9f3ac1d2b7e4c601","seq":%s,"receivedAt":"2026-10-16T12:00:00.000005Z"}`
	require.Equal(t,
		strings.Replace(dev, "%s", "1", 1)+`,"event":"A","userId":"u1"}`+"\n"+
			strings.Replace(dev, "%s", "2", 1)+`}`+"\n"+
			strings.Replace(dev, "%s", "3", 1)+`,"event":"B"}`+"\n",
		out.String())
}

type brokenWriter struct{ writes int }

func (w *brokenWriter) Write([]byte) (int, error) {
	w.writes++
	return 0, errors.New("no space left on device")
}

// After the first failed write the sink writes nothing more, counts what it
// could not write, and reports the error once.
func TestAFailedWriteIsReportedOnceAndCountsTheLoss(t *testing.T) {
	t.Parallel()
	out := &brokenWriter{}
	s := newStreamSink(out, io.Discard, "")

	s.Capture(accepted(`{"event":"A"}`, `{"event":"B"}`))
	s.Capture(accepted(`{"event":"C"}`))

	require.EqualError(t, <-s.failed, "no space left on device")
	require.Empty(t, s.failed)
	require.Equal(t, 1, out.writes)
	require.Equal(t, 3, s.Lost())
}

// An event can carry a _dev member of its own. The listener's then goes last,
// where jq and encoding/json, which keep the last of two members with one
// name, read it. A _dev deeper in the event is no collision.
func TestStampedLinesKeepTheListenersDevWhenTheEventHasOne(t *testing.T) {
	t.Parallel()
	var out strings.Builder
	s := newStreamSink(&out, io.Discard, "9f3ac1d2b7e4c601")

	s.Capture(accepted(
		`{"_dev":{"serverId":"fake-spoofed","seq":99},"event":"A"}`,
		`{"\u005fdev":"fake-escaped","event":"B"}`,
		`{"properties":{"_dev":"nested"},"event":"C"}`,
	))

	const dev = `"_dev":{"serverId":"9f3ac1d2b7e4c601","seq":%s,"receivedAt":"2026-10-16T12:00:00.000005Z"}`
	lines := strings.Split(strings.TrimSuffix(out.String(), "\n"), "\n")
	require.Equal(t, []string{
		`{"_dev":{"serverId":"fake-spoofed","seq":99},"event":"A",` + strings.Replace(dev, "%s", "1", 1) + `}`,
		`{"\u005fdev":"fake-escaped","event":"B",` + strings.Replace(dev, "%s", "2", 1) + `}`,
		`{` + strings.Replace(dev, "%s", "3", 1) + `,"properties":{"_dev":"nested"},"event":"C"}`,
	}, lines)

	for i, line := range lines {
		var got map[string]any
		require.NoError(t, json.Unmarshal([]byte(line), &got))
		require.Equal(t, map[string]any{
			"serverId":   "9f3ac1d2b7e4c601",
			"seq":        float64(i + 1),
			"receivedAt": "2026-10-16T12:00:00.000005Z",
		}, got["_dev"], line)
	}
}

// partialWriter writes the first n bytes it is given, then fails, as a disk
// that fills in the middle of a write does.
type partialWriter struct{ n int }

func (w *partialWriter) Write(p []byte) (int, error) {
	return min(w.n, len(p)), errors.New("no space left on device")
}

// A write that fails part way still wrote its whole lines, so only the events
// after them are lost: the one cut short and the ones not reached.
func TestAPartialWriteCountsOnlyTheEventsItDidNotWrite(t *testing.T) {
	t.Parallel()
	first := `{"event":"A"}` + "\n"
	out := &partialWriter{n: len(first) + 3}
	s := newStreamSink(out, io.Discard, "")

	s.Capture(accepted(`{"event":"A"}`, `{"event":"B"}`, `{"event":"C"}`))

	require.EqualError(t, <-s.failed, "no space left on device")
	require.Equal(t, 2, s.Lost())
}
