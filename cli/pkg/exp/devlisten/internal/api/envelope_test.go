package api

import (
	"encoding/json"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/rudderlabs/rudder-iac/cli/pkg/exp/devlisten/internal/store"
)

// A CI job must see that the store evicted part of its window:
// evictedThrough above its cursor means the answer is not complete.
func TestEnvelopesCarryEvictedThrough(t *testing.T) {
	t.Parallel()
	st := store.New(testIdentity.ServerID, store.WithLimits(2, 1<<20))
	srv := httptest.NewServer(New(st, testIdentity, Config{}))
	t.Cleanup(srv.Close)
	for range 3 {
		st.Append(ingestion(track(0, "A")))
	}

	_, events := get(t, srv.URL+"/_dev/v1/events")
	_, requests := get(t, srv.URL+"/_dev/v1/requests")

	require.Equal(t, float64(1), events["evictedThrough"])
	require.Equal(t, float64(1), requests["evictedThrough"])
}

func TestEnvelopesNameTheNextCallAsCommandAndURL(t *testing.T) {
	t.Parallel()
	srv, st := newTestServer(t)
	st.Append(ingestion(track(0, "A")))
	st.Append(ingestion(track(0, "A")))

	resp, body := getRaw(t, srv.URL+"/_dev/v1/events?event=A&limit=1")
	var page map[string]any
	require.NoError(t, json.Unmarshal([]byte(body), &page))

	require.Equal(t, true, page["hasMore"])
	require.Equal(t, "rudder-cli dev events list --since 1 --event A --json", page["next"])
	require.Equal(t, map[string]any{"next": "events?event=A&limit=1&since=1"}, page["links"])
	require.Equal(t, `</_dev/v1/events?event=A&limit=1&since=1>; rel="next"`, resp.Header.Get("Link"))
}

// The error object names its HTTP status, so it still reads right after the
// CLI copied it to stderr.
func TestErrorObjectCarriesItsStatus(t *testing.T) {
	t.Parallel()
	srv, _ := newTestServer(t)

	status, body := get(t, srv.URL+"/_dev/v1/events?serverId=0000000000000000")

	require.Equal(t, 409, status)
	require.Equal(t, float64(409), body["error"].(map[string]any)["status"])
}

// A list read carries small totals, not the summary: bare dev events is the
// summary. That keeps a property read small.
func TestListViewsCarryTotalsNotTheSummary(t *testing.T) {
	t.Parallel()
	srv, st := newTestServer(t)
	st.Append(ingestion(track(0, "A"), track(1, "B")))
	st.Append(ingestion(track(0, "A")))

	_, list := get(t, srv.URL+"/_dev/v1/events?event=A&limit=1")
	_, counts := get(t, srv.URL+"/_dev/v1/events?event=A&view=counts")

	require.Nil(t, list["summary"])
	require.Equal(t, float64(2), list["total"])
	require.Equal(t, float64(1), list["returned"])
	require.NotNil(t, counts["summary"])
	require.Equal(t, float64(2), counts["total"])
	require.Equal(t, float64(0), counts["returned"])
}

func TestLimitZeroReadsTheCursorOnly(t *testing.T) {
	t.Parallel()
	srv, st := newTestServer(t)
	st.Append(ingestion(track(0, "A")))
	st.Append(ingestion(track(0, "A")))

	_, page := get(t, srv.URL+"/_dev/v1/events?limit=0")

	require.Equal(t, float64(2), page["cursor"])
	require.Equal(t, []any{}, page["events"])
	require.Nil(t, page["summary"])
	require.Equal(t, false, page["hasMore"])
}

// since is a cursor, a duration back from now, or an RFC 3339 time.
func TestSinceTakesATimeWindow(t *testing.T) {
	t.Parallel()
	srv, st := newTestServer(t)
	old := ingestion(track(0, "Old"))
	old.ReceivedAt = time.Now().Add(-time.Hour)
	st.Append(old)
	fresh := ingestion(track(0, "New"))
	fresh.ReceivedAt = time.Now()
	st.Append(fresh)

	_, byDuration := get(t, srv.URL+"/_dev/v1/events?since=5m")
	_, byTime := get(t, srv.URL+"/_dev/v1/events?since="+time.Now().Add(-30*time.Minute).UTC().Format(time.RFC3339))
	status, _ := get(t, srv.URL+"/_dev/v1/events?since=yesterday")

	require.Len(t, byDuration["events"], 1)
	require.Equal(t, float64(1), byDuration["since"], "since resolves to the cursor before the window")
	require.Len(t, byTime["events"], 1)
	require.Equal(t, 400, status)
}
