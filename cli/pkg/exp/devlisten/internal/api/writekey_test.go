package api

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// Each service of an app sends with its own write key; a foreign key is
// stored masked, so the filter matches the literal key by its sha256.
func TestWriteKeyFiltersAndCountsPerKey(t *testing.T) {
	t.Parallel()
	srv, st := newTestServer(t)
	st.Append(ingestion(track(0, "A")))
	st.Append(withWriteKey(ingestion(track(0, "B"), track(1, "B")), "backendKey123456"))

	summary := summaryOf(t, srv.URL+"/_dev/v1/events?writeKey=backendKey123456&writeKey=unusedKey9876543")

	require.Equal(t, map[string]any{
		"back...3456": map[string]any{"requests": float64(1), "events": float64(2)},
		"unus...6543": map[string]any{"requests": float64(0), "events": float64(0)},
	}, summary["byWriteKey"])
	require.Equal(t, map[string]any{"B": float64(2)}, summary["byEvent"])
}

func TestWriteKeyFilterNarrowsRequestsAndEvents(t *testing.T) {
	t.Parallel()
	srv, st := newTestServer(t)
	st.Append(ingestion(track(0, "A")))
	st.Append(withWriteKey(ingestion(track(0, "B")), "backendKey123456"))

	_, requests := get(t, srv.URL+"/_dev/v1/requests?view=list&writeKey=dev")
	_, events := get(t, srv.URL+"/_dev/v1/events?view=list&writeKey=backendKey123456")

	require.Len(t, requests["requests"], 1)
	require.Equal(t, float64(1), requests["requests"].([]any)[0].(map[string]any)["seq"])
	require.Len(t, events["events"], 1)
	require.Equal(t, "B", events["events"].([]any)[0].(map[string]any)["event"])
	require.Equal(t, "rudder-cli dev events list --since 0 --write-key 'backendKey123456' --view compact --json",
		events["omitted"].(map[string]any)["next"])
}

func TestMissingWriteKeyIsAWarning(t *testing.T) {
	t.Parallel()
	srv, st := newTestServer(t)
	st.Append(withWriteKey(ingestion(track(0, "A")), ""))

	summary := summaryOf(t, srv.URL+"/_dev/v1/events")

	require.Equal(t, map[string]any{"requests": float64(1), "events": float64(1)},
		summary["byWriteKey"].(map[string]any)[""])
	require.Equal(t, []string{"missing_write_key", "all_accepted"}, diagnosisCodes(t, summary))
	d := summary["diagnosis"].([]any)[0].(map[string]any)
	require.Equal(t, "1 requests had no write key; RudderStack rejects these with 401.", d["message"])
	require.Equal(t, "rudder-cli dev requests list --since 0 --write-key '' --json", d["next"])
}
