package api

import (
	"net/http"
	"net/url"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSummaryExpectProvesPresenceAndAbsence(t *testing.T) {
	t.Parallel()
	srv, st := newTestServer(t)
	st.Append(ingestion(track(0, "Shown"), track(1, "Clicked")))
	st.Append(ingestion(track(0, "Clicked")))

	query := url.Values{"expect": {"Shown", "Clicked=1", "Sent", "a=b=2"}}
	_, summary := get(t, srv.URL+"/_dev/v1/summary?"+query.Encode())

	require.Equal(t, []any{
		map[string]any{"event": "Shown", "want": nil, "got": float64(1), "status": "present"},
		map[string]any{"event": "Clicked", "want": float64(1), "got": float64(2), "status": "count_mismatch"},
		map[string]any{"event": "Sent", "want": nil, "got": float64(0), "status": "missing"},
		map[string]any{"event": "a=b", "want": float64(2), "got": float64(0), "status": "missing"},
	}, summary["expected"])
	require.Equal(t, []string{"expected_missing", "expected_count_mismatch", "all_accepted"}, diagnosisCodes(t, summary))
	missing := summary["diagnosis"].([]any)[0].(map[string]any)
	require.Equal(t, []any{"Sent", "a=b"}, missing["events"])
	require.Equal(t, float64(2), missing["count"])
}

func TestSummaryWithoutExpectHasNoExpectedKey(t *testing.T) {
	t.Parallel()
	srv, _ := newTestServer(t)

	_, summary := get(t, srv.URL+"/_dev/v1/summary")
	require.NotContains(t, summary, "expected")

	status, body := get(t, srv.URL+"/_dev/v1/summary?expect=")
	require.Equal(t, http.StatusBadRequest, status)
	require.Equal(t, "expect", body["error"].(map[string]any)["param"])
}
