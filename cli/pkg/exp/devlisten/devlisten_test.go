package devlisten_test

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/rudderlabs/rudder-iac/cli/pkg/exp/devlisten"
)

func startServer(t *testing.T, opts ...devlisten.Option) *devlisten.Server {
	t.Helper()
	s, err := devlisten.Start(context.Background(), opts...)
	require.NoError(t, err)
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		require.NoError(t, s.Close(ctx))
	})
	return s
}

func postGzip(t *testing.T, url, body string) {
	t.Helper()
	var buf bytes.Buffer
	zw := gzip.NewWriter(&buf)
	_, err := zw.Write([]byte(body))
	require.NoError(t, err)
	require.NoError(t, zw.Close())

	req, err := http.NewRequest(http.MethodPost, url, &buf)
	require.NoError(t, err)
	req.SetBasicAuth("dev", "")
	req.Header.Set("Content-Encoding", "gzip")
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	got, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	require.Equal(t, "ok", string(got))
}

func TestReadyObject(t *testing.T) {
	t.Parallel()
	s := startServer(t)

	ready := s.Ready()

	require.True(t, ready.Ready)
	require.Equal(t, "v1", ready.APIVersion)
	require.Regexp(t, `^[0-9a-f]{16}$`, ready.ServerID)
	require.Equal(t, "127.0.0.1", ready.Bind)
	require.NotZero(t, ready.Port)
	require.Equal(t, s.URL(), ready.URL)
	require.Equal(t, "dev", ready.WriteKey)
	require.Equal(t, "any", ready.WriteKeyPolicy)
	require.Equal(t, uint64(0), ready.Cursor)
	require.Nil(t, ready.StateFile)

	line, err := json.Marshal(ready)
	require.NoError(t, err)
	require.Contains(t, string(line), `"stateFile":null`)
}

func TestPostBatchThenReadItBackWithSinceAndWait(t *testing.T) {
	t.Parallel()
	s := startServer(t)
	client := s.Client()
	ctx := context.Background()

	info, err := client.Info(ctx)
	require.NoError(t, err)
	require.Equal(t, uint64(0), info.Cursor)

	postGzip(t, s.URL()+"/v1/track", `{"event":"Before","userId":"u0"}`)
	info, err = client.Info(ctx)
	require.NoError(t, err)
	since := info.Cursor
	require.Equal(t, uint64(1), since)

	type result struct {
		page devlisten.Page
		err  error
	}
	done := make(chan result, 1)
	go func() {
		page, err := client.Events(ctx, devlisten.Query{Since: since, View: devlisten.ViewFull, Event: []string{"Order Completed"}, Wait: 10 * time.Second, Min: 1})
		done <- result{page, err}
	}()
	time.Sleep(100 * time.Millisecond)
	postGzip(t, s.URL()+"/v1/batch", `{"batch":[{"type":"identify","userId":"u1","traits":{"plan":"pro"}},{"type":"track","event":"Order Completed","userId":"u1","properties":{"total":42}}],"sentAt":"2026-09-29T12:00:00.100Z"}`)

	var res result
	select {
	case res = <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("wait did not return after the batch arrived")
	}
	require.NoError(t, res.err)
	page := res.page
	require.False(t, page.TimedOut)
	require.Equal(t, uint64(2), page.Cursor)
	require.Equal(t, devlisten.Unfiltered{Requests: 1, Events: 2}, page.Unfiltered)
	require.Len(t, page.Events, 1)

	ev := page.Events[0]
	require.Equal(t, uint64(2), ev.Seq)
	require.Equal(t, 1, ev.Idx)
	require.Equal(t, "track", ev.Type)
	require.Equal(t, "Order Completed", ev.Name)
	require.Equal(t, "u1", ev.UserID)
	require.Equal(t, "", ev.AnonymousID)
	require.Equal(t, "2026-09-29T12:00:00.100Z", ev.SentAt)
	require.Equal(t, map[string]any{"total": float64(42)}, ev.Properties)
	require.Equal(t, "127.0.0.1", ev.EnrichedMessage["request_ip"])
	require.Contains(t, string(ev.Raw), `"anonymousId":null`)
}

func TestWaitTimesOutWithoutError(t *testing.T) {
	t.Parallel()
	s := startServer(t)

	page, err := s.Client().Events(context.Background(), devlisten.Query{Event: []string{"never"}, Wait: 200 * time.Millisecond, Min: 1})

	require.NoError(t, err)
	require.True(t, page.TimedOut)
	require.Empty(t, page.Events)
}

func TestClientWaitLoopsUntilMin(t *testing.T) {
	t.Parallel()
	s := startServer(t)
	go func() {
		time.Sleep(200 * time.Millisecond)
		postGzip(t, s.URL()+"/v1/track", `{"event":"Late","userId":"u1"}`)
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	page, err := s.Client().WaitForEvents(ctx, devlisten.Query{Event: []string{"Late"}, Min: 1})

	require.NoError(t, err)
	require.Len(t, page.Events, 1)
}

func TestClientWaitReportsDeadline(t *testing.T) {
	t.Parallel()
	s := startServer(t)

	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Millisecond)
	defer cancel()
	_, err := s.Client().WaitForEvents(ctx, devlisten.Query{Event: []string{"never"}, Min: 1})

	require.ErrorIs(t, err, context.DeadlineExceeded)
	require.Contains(t, err.Error(), "unfiltered: 0 requests, 0 events, 0 control")
}

func TestClientPinsServerID(t *testing.T) {
	t.Parallel()
	s := startServer(t)

	_, err := devlisten.NewClient(s.URL(), devlisten.WithServerID("0000000000000000")).Events(context.Background(), devlisten.Query{})

	require.ErrorIs(t, err, devlisten.ErrServerChanged)
	var apiErr *devlisten.APIError
	require.True(t, errors.As(err, &apiErr))
	require.Equal(t, http.StatusConflict, apiErr.StatusCode)
	require.Equal(t, "server_changed", apiErr.Code)
}

func TestFixedPortInUse(t *testing.T) {
	t.Parallel()
	s := startServer(t)

	_, err := devlisten.Start(context.Background(), devlisten.WithPort(s.Port()))

	require.ErrorIs(t, err, devlisten.ErrPortInUse)
}

func TestCloseStopsServing(t *testing.T) {
	t.Parallel()
	s, err := devlisten.Start(context.Background())
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	require.NoError(t, s.Close(ctx))
	require.NoError(t, s.Wait())

	_, err = s.Client().Info(context.Background())
	require.Error(t, err)
}

func TestQueryValues(t *testing.T) {
	t.Parallel()

	q := devlisten.Query{
		Since: 57, Limit: 10, Order: devlisten.OrderAsc, View: devlisten.ViewSummary,
		Event: []string{"A", "B"}, Type: []string{"track"}, Route: []string{"/v1/track"}, StatusCode: []int{200, 400},
		UserID: "u1", AnonymousID: "a1", Include: []string{"context"}, Fields: []string{"properties"},
		MaxBytes: devlisten.MaxBytesOff, Min: 2, Wait: 30 * time.Second,
	}

	require.Equal(t, "anonymousId=a1&event=A&event=B&fields=properties&include=context&limit=10&maxBytes=0&min=2"+
		"&order=asc&route=%2Fv1%2Ftrack&since=57&statusCode=200&statusCode=400&type=track&userId=u1&view=summary&wait=30s",
		q.Values().Encode())
	require.Equal(t, "maxBytes=500", devlisten.Query{MaxBytes: 500}.Values().Encode())
	require.Equal(t, "", devlisten.Query{}.Values().Encode())
}
