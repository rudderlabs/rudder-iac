package devlisten

import (
	"context"
	"io"
	"net/http"
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
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
