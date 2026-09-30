package devlisten_test

import (
	"context"
	"io"
	"net"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/rudderlabs/rudder-iac/cli/pkg/exp/devlisten"
)

func postTrack(t *testing.T, url, body string) {
	t.Helper()
	req, err := http.NewRequest(http.MethodPost, url+"/v1/track", strings.NewReader(body))
	require.NoError(t, err)
	req.SetBasicAuth("dev", "")
	resp, err := testClient.Do(req)
	require.NoError(t, err)
	_, _ = io.Copy(io.Discard, resp.Body)
	resp.Body.Close()
}

// testClient does not share the default transport, whose speculative dials
// leave connections a server Shutdown waits on.
var testClient = &http.Client{Transport: &http.Transport{DisableKeepAlives: true}}

func TestParametersListEveryQueryRoute(t *testing.T) {
	t.Parallel()

	params := devlisten.Parameters()

	require.Contains(t, params, "events")
	require.Contains(t, params, "requests")
	require.Contains(t, params["events"], devlisten.Parameter{Name: "limit", Default: "100"})
	require.Contains(t, params["events"], devlisten.Parameter{Name: "statusCode", Repeatable: true})
}

func TestRequestsRequestSummaryAndReset(t *testing.T) {
	t.Parallel()
	s := startServer(t)
	client := s.Client()
	ctx := context.Background()
	postTrack(t, s.URL(), `{"event":"A","userId":"u1"}`)

	failed := false
	page, err := client.Requests(ctx, devlisten.RequestQuery{Failed: &failed})
	require.NoError(t, err)
	require.Len(t, page.Requests, 1)
	require.Equal(t, uint64(1), page.Requests[0].Seq)
	require.Equal(t, "/v1/track", page.Requests[0].Route)
	require.NotNil(t, page.Omitted)
	require.Contains(t, string(page.Raw), `"requests":[`)

	rec, err := client.Request(ctx, 1, devlisten.RecordQuery{})
	require.NoError(t, err)
	require.Equal(t, "accepted", rec.Outcome)
	require.Contains(t, string(rec.Raw), `"body":"{\"event\":\"A\",\"userId\":\"u1\"}"`)

	summary, err := client.Summary(ctx, devlisten.SummaryQuery{})
	require.NoError(t, err)
	require.Len(t, summary.Diagnosis, 2, "a Go client with no browser traffic adds no_browser_traffic")
	require.Equal(t, "no_browser_traffic", summary.Diagnosis[0].Code)
	require.Equal(t, devlisten.Diagnosis{Code: "all_accepted", Count: 1,
		Message: "Every request was accepted. List the events to check names and properties.",
		Next:    "rudder-cli dev events list --since 0 --view summary --json"}, summary.Diagnosis[1])
	require.Equal(t, devlisten.SDKCounts{Requests: 1, Events: 1}, summary.BySource.BySdk["go"])

	reset, err := client.Reset(ctx)
	require.NoError(t, err)
	require.Equal(t, uint64(1), reset.Cursor)
	require.Equal(t, 1, reset.Removed.Requests)
	_, err = client.Request(ctx, 1, devlisten.RecordQuery{})
	require.ErrorIs(t, err, devlisten.ErrNotFound)
}

func TestSendPostsAProbeAndFindsItsSeq(t *testing.T) {
	t.Parallel()
	s := startServer(t)
	postTrack(t, s.URL(), `{"event":"A","userId":"u1"}`)

	res, err := s.Client().Send(context.Background(), devlisten.Probe{WriteKey: devlisten.DefaultWriteKey, UserAgent: "rudder-cli dev send/test"})

	require.NoError(t, err)
	require.Equal(t, devlisten.SendResult{StatusCode: 200, Body: "ok", Seq: 2, Route: "/v1/track"}, res)
	summary, err := s.Client().Summary(context.Background(), devlisten.SummaryQuery{Since: 1})
	require.NoError(t, err)
	require.Equal(t, 1, summary.Requests.Probes)
}

func TestShutdownRouteStopsTheServer(t *testing.T) {
	t.Parallel()
	s, err := devlisten.Start(context.Background())
	require.NoError(t, err)

	res, err := s.Client().Shutdown(context.Background())
	require.NoError(t, err)
	require.True(t, res.Stopping)

	select {
	case <-s.Done():
	case <-time.After(5 * time.Second):
		t.Fatal("server did not stop after /shutdown")
	}
	require.Equal(t, devlisten.StopReasonStop, s.StopReason())
}

func TestIdleExitStopsAQuietServer(t *testing.T) {
	t.Parallel()
	s, err := devlisten.Start(context.Background(), devlisten.WithIdleExit(300*time.Millisecond))
	require.NoError(t, err)

	start := time.Now()
	for time.Since(start) < 500*time.Millisecond {
		_, err := s.Client().Info(context.Background())
		require.NoError(t, err, "info does not reset the idle timer, but the server is still up")
		if time.Since(start) > 250*time.Millisecond {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}

	select {
	case <-s.Done():
	case <-time.After(5 * time.Second):
		t.Fatal("idle server did not stop")
	}
	require.Equal(t, devlisten.StopReasonIdle, s.StopReason())
	require.Less(t, time.Since(start), 2*time.Second)
}

func TestIdleExitWaitsForCaptures(t *testing.T) {
	t.Parallel()
	s, err := devlisten.Start(context.Background(), devlisten.WithIdleExit(400*time.Millisecond))
	require.NoError(t, err)
	t.Cleanup(func() { _ = s.Close(context.Background()) })

	for range 6 {
		postTrack(t, s.URL(), `{"event":"A","userId":"u1"}`)
		time.Sleep(100 * time.Millisecond)
	}

	select {
	case <-s.Done():
		t.Fatal("a server that keeps capturing is not idle")
	default:
	}
}

func TestCaptureHookSeesEveryRecord(t *testing.T) {
	t.Parallel()
	var (
		mu   sync.Mutex
		seen []devlisten.Capture
	)
	s := startServer(t, devlisten.WithCaptureHook(func(c devlisten.Capture) {
		mu.Lock()
		defer mu.Unlock()
		seen = append(seen, c)
	}))

	postTrack(t, s.URL(), `{"event":"A","userId":"u1"}`)
	resp, err := testClient.Get(s.URL() + "/nope") //nolint:noctx // test
	require.NoError(t, err)
	_, _ = io.Copy(io.Discard, resp.Body)
	resp.Body.Close()

	mu.Lock()
	defer mu.Unlock()
	require.Equal(t, []devlisten.Capture{
		{Seq: 1, Kind: "ingestion", Route: "/v1/track", Method: "POST", StatusCode: 200, Outcome: "accepted",
			Type: "track", Event: "A", Events: 1},
		{Seq: 2, Kind: "control", Route: "/nope", Method: "GET", StatusCode: 404, Outcome: "rejected"},
	}, seen)
}

func TestLoopbackServerRefusesAForeignHost(t *testing.T) {
	t.Parallel()
	s := startServer(t)

	req, err := http.NewRequest(http.MethodGet, s.URL()+"/_dev/v1/info", nil)
	require.NoError(t, err)
	req.Host = "rebind.example:" + strings.TrimPrefix(s.URL(), "http://127.0.0.1:")
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	resp.Body.Close()

	require.Equal(t, http.StatusForbidden, resp.StatusCode)
}

func TestCloseDoesNotWaitForConnectionsWithoutARequest(t *testing.T) {
	t.Parallel()
	s, err := devlisten.Start(context.Background())
	require.NoError(t, err)
	conn, err := net.Dial("tcp", strings.TrimPrefix(s.URL(), "http://"))
	require.NoError(t, err)
	t.Cleanup(func() { conn.Close() })
	time.Sleep(50 * time.Millisecond)

	start := time.Now()
	require.NoError(t, s.Close(context.Background()))
	require.Less(t, time.Since(start), 2*time.Second)
}
