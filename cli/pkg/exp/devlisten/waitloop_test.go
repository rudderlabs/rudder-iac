package devlisten_test

import (
	"context"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/rudderlabs/rudder-iac/cli/pkg/exp/devlisten"
)

// countingTransport counts the HTTP requests a client sends.
type countingTransport struct{ n atomic.Int64 }

func (c *countingTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	c.n.Add(1)
	return http.DefaultTransport.RoundTrip(r)
}

func bigTrack(t *testing.T, s *devlisten.Server, n int) {
	t.Helper()
	pad := strings.Repeat("p", 1000)
	for range n {
		postTrack(t, s.URL(), `{"event":"Big","userId":"u1","properties":{"pad":"`+pad+`"}}`)
	}
}

// A page cut by MaxBytes below Min must move the cursor forward: the client
// collects the events across pages instead of asking the same page again.
func TestWaitForEventsAccumulatesAcrossTruncatedPages(t *testing.T) {
	t.Parallel()
	s := startServer(t)
	bigTrack(t, s, 3)
	transport := &countingTransport{}
	client := devlisten.NewClient(s.URL(), devlisten.WithHTTPClient(&http.Client{Transport: transport}))

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	page, err := client.WaitForEvents(ctx, devlisten.Query{Event: []string{"Big"}, Min: 3, MaxBytes: 3800})

	require.NoError(t, err)
	require.Len(t, page.Events, 3)
	require.Equal(t, uint64(3), page.Cursor)
	require.LessOrEqual(t, transport.n.Load(), int64(4))
}

// One request above MaxBytes never fits a page: the client stops at once
// instead of asking for it again.
func TestWaitForEventsStopsOnAnOversizedRequest(t *testing.T) {
	t.Parallel()
	s := startServer(t)
	bigTrack(t, s, 1)
	transport := &countingTransport{}
	client := devlisten.NewClient(s.URL(), devlisten.WithHTTPClient(&http.Client{Transport: transport}))

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	_, err := client.WaitForEvents(ctx, devlisten.Query{Event: []string{"Big"}, MaxBytes: 1800})

	require.ErrorIs(t, err, devlisten.ErrOutputLimit)
	require.Equal(t, int64(1), transport.n.Load())
}

// The deadline error reports the counts of the last page that arrived.
func TestWaitForEventsDeadlineReportsTheLastPage(t *testing.T) {
	t.Parallel()
	s := startServer(t)
	bigTrack(t, s, 2)
	transport := &countingTransport{}
	client := devlisten.NewClient(s.URL(), devlisten.WithHTTPClient(&http.Client{Transport: transport}))

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	page, err := client.WaitForEvents(ctx, devlisten.Query{Event: []string{"Big"}, Min: 3, MaxBytes: 3800})

	require.ErrorIs(t, err, context.DeadlineExceeded)
	require.Contains(t, err.Error(), "collected 2 of 3 events up to seq 2")
	require.Len(t, page.Events, 2)
	require.LessOrEqual(t, transport.n.Load(), int64(8))
}
