package api

import (
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestRequestsListIsCompactByDefault(t *testing.T) {
	t.Parallel()
	srv, st := newTestServer(t)
	rec := ingestion(track(0, "A"))
	rec.ReceivedAt = time.Date(2026, 9, 30, 12, 0, 0, 0, time.UTC)
	rec.Request.Method = http.MethodPost
	st.Append(rec)

	_, page := get(t, srv.URL+"/_dev/v1/requests")

	require.Equal(t, "compact", page["view"])
	require.Equal(t, []any{map[string]any{
		"seq": float64(1), "receivedAt": "2026-09-30T12:00:00Z", "method": "POST", "route": "/v1/batch",
		"statusCode": float64(200), "outcome": "accepted", "kind": "ingestion",
		"events": []any{map[string]any{"idx": float64(0), "type": "track", "event": "A"}},
	}}, page["requests"])
}

func TestRequestsListSummaryAndFullViews(t *testing.T) {
	t.Parallel()
	srv, st := newTestServer(t)
	rec := ingestion(track(0, "A"))
	rec.Request.Method = http.MethodPost
	st.Append(rec)

	_, page := get(t, srv.URL+"/_dev/v1/requests?view=summary")
	require.Equal(t, []any{map[string]any{
		"seq": float64(1), "kind": "ingestion", "method": "POST", "route": "/v1/batch",
		"statusCode": float64(200), "outcome": "accepted", "eventCount": float64(1),
	}}, page["requests"])
	require.Equal(t, "rudder-cli dev requests list --since 0 --json", page["omitted"].(map[string]any)["next"])

	_, page = get(t, srv.URL+"/_dev/v1/requests?view=full")
	full := page["requests"].([]any)[0].(map[string]any)
	require.Contains(t, full["events"].([]any)[0], "enrichedMessage")
	require.Nil(t, page["omitted"])
}

// A projection keeps method, so a preflight never looks like the request it
// precedes.
func TestRequestsProjectionKeepsMethod(t *testing.T) {
	t.Parallel()
	srv, st := newTestServer(t)
	st.Append(control("/v1/track", http.MethodOptions, 204))

	_, page := get(t, srv.URL+"/_dev/v1/requests?kind=control&fields=statusCode")

	require.Equal(t, []any{map[string]any{
		"seq": float64(1), "statusCode": float64(204), "request": map[string]any{"method": "OPTIONS"},
	}}, page["requests"])
}

func TestRequestsFieldsWithViewNamesTheCorrectedCommand(t *testing.T) {
	t.Parallel()
	srv, _ := newTestServer(t)

	status, body := get(t, srv.URL+"/_dev/v1/requests?view=summary&fields=route&kind=control")

	require.Equal(t, http.StatusBadRequest, status)
	require.Equal(t, "rudder-cli dev requests list --since 0 --kind 'control' --fields 'route' --json",
		body["error"].(map[string]any)["next"])
}

// The body already holds each message, so show leaves out the parsed and
// the enriched copies unless view=full.
func TestRequestShowLeavesOutTheMessageCopies(t *testing.T) {
	t.Parallel()
	srv, st := newTestServer(t)
	rec := ingestion(track(0, "A"))
	rec.Request.Body = `{"batch":[{"event":"A"}]}`
	st.Append(rec)

	_, got := get(t, srv.URL+"/_dev/v1/requests/1")
	require.Equal(t, `{"batch":[{"event":"A"}]}`, got["request"].(map[string]any)["body"])
	require.Equal(t, []any{map[string]any{
		"idx": float64(0), "type": "track", "event": "A", "userId": "u1", "anonymousId": nil, "messageId": "m",
	}}, got["events"])
	require.Equal(t, map[string]any{
		"fields": []any{"events.message", "events.enrichedMessage"}, "context": []any{},
		"next": "rudder-cli dev requests show 1 --fields request.body --json",
	}, got["omitted"])

	_, full := get(t, srv.URL+"/_dev/v1/requests/1?view=full")
	require.Contains(t, full["events"].([]any)[0], "enrichedMessage")
	require.NotContains(t, full, "omitted")
}
