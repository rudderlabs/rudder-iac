package api

import (
	"maps"
	"net/http"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/rudderlabs/rudder-iac/cli/pkg/exp/devlisten/internal/store"
)

const (
	timeoutShort = 2 * time.Second
	tick         = 10 * time.Millisecond
)

func control(route, method string, status int) store.Record {
	return store.Record{Kind: "control", Route: route, StatusCode: status, Failed: status > 299,
		Request: store.Request{Method: method}}
}

func rejected(stage string) store.Record {
	rec := ingestion()
	rec.StatusCode, rec.Failed, rec.Outcome = 400, true, "rejected"
	rec.Rejection = &store.Rejection{Stage: stage}
	return rec
}

// summaryOf reads the summary block of one /events?view=counts answer.
func summaryOf(t *testing.T, url string) map[string]any {
	t.Helper()
	sep := "?"
	if strings.Contains(url, "?") {
		sep = "&"
	}
	status, page := get(t, url+sep+"view=counts")
	require.Equal(t, http.StatusOK, status, page)
	return page["summary"].(map[string]any)
}

func diagnosisCodes(t *testing.T, summary map[string]any) []string {
	t.Helper()
	var codes []string
	for _, d := range summary["diagnosis"].([]any) {
		item := d.(map[string]any)
		require.NotEmpty(t, item["next"])
		codes = append(codes, item["code"].(string))
	}
	return codes
}

func TestListIsTheDefaultViewAndCountsHasAnEmptyEventList(t *testing.T) {
	t.Parallel()
	srv, st := newTestServer(t)
	st.Append(ingestion(track(0, "A")))

	_, list := get(t, srv.URL+"/_dev/v1/events")
	_, page := get(t, srv.URL+"/_dev/v1/events?view=counts")

	require.Equal(t, "list", list["view"])
	require.Len(t, list["events"], 1)
	require.Nil(t, list["summary"], "the list leaves the summary to view=counts")
	require.NotNil(t, page["summary"])
	require.Equal(t, "counts", page["view"])
	require.Equal(t, []any{}, page["events"])
	require.Equal(t, float64(1), page["cursor"])
	require.Equal(t, map[string]any{"fields": []any{"events"}, "context": []any{},
		"next": "rudder-cli dev events list --since 0 --json"}, page["omitted"])
	require.NotContains(t, page, "unfiltered", "summary replaces unfiltered")
}

func TestEveryViewCarriesTheSameEnvelope(t *testing.T) {
	t.Parallel()
	srv, st := newTestServer(t)
	st.Append(ingestion(track(0, "A")))

	var keys [][]string
	for _, query := range []string{"", "?view=list", "?view=compact", "?view=full", "?fields=properties"} {
		_, page := get(t, srv.URL+"/_dev/v1/events"+query)
		keys = append(keys, sortedKeys(page))
	}

	for _, k := range keys[1:] {
		require.Equal(t, keys[0], k)
	}
}

func TestSummaryCounts(t *testing.T) {
	t.Parallel()
	srv, st := newTestServer(t)
	st.Append(control("/sourceConfig", http.MethodGet, 200))
	st.Append(control("/v1/batch", http.MethodOptions, 204))
	st.Append(ingestion(track(0, "A"), track(1, "B")))
	st.Append(rejected("auth"))

	summary := summaryOf(t, srv.URL+"/_dev/v1/events?since=0")

	require.Equal(t, map[string]any{
		"total": float64(2), "failed": float64(1),
		"byRoute":      map[string]any{"/v1/batch": float64(2)},
		"byStatusCode": map[string]any{"200": float64(1), "400": float64(1)},
		"byOutcome":    map[string]any{"accepted": float64(1), "rejected": float64(1)},
		"byStage":      map[string]any{"auth": float64(1)},
	}, summary["requests"])
	require.Equal(t, map[string]any{"total": float64(2), "byType": map[string]any{"track": float64(2)}},
		summary["events"])
	require.Equal(t, map[string]any{"A": float64(1), "B": float64(1)}, summary["byEvent"])
	require.Equal(t, map[string]any{
		"total": float64(2), "sourceConfig": float64(1), "sourceConfigFailed": float64(0),
		"preflight": float64(1), "pluginPath": float64(0), "other": float64(0),
	}, summary["control"])
	require.Equal(t, []any{map[string]any{
		"code": "auth_rejected", "count": float64(1),
		"message": "Requests were rejected at the auth stage. Compare their writeKey with the SDK configuration.",
		"next":    "rudder-cli dev requests list --since 0 --status-code 401 --json",
	}}, summary["diagnosis"])
}

func TestByEventShowsEveryRequestedNameAtZero(t *testing.T) {
	t.Parallel()
	srv, st := newTestServer(t)
	st.Append(ingestion(track(0, "A"), track(1, "B")))

	summary := summaryOf(t, srv.URL+"/_dev/v1/events?event=A&event=Missing")

	require.Equal(t, map[string]any{"A": float64(1), "Missing": float64(0)}, summary["byEvent"])
	require.Equal(t, float64(1), summary["events"].(map[string]any)["total"], "event filters narrow the counts")
	require.Equal(t, float64(1), summary["requests"].(map[string]any)["total"])

	other := summaryOf(t, srv.URL+"/_dev/v1/events?event=Missing")
	require.Equal(t, float64(0), other["requests"].(map[string]any)["total"], "every count follows the filters")
}

func TestStatusCodeNarrowsRequestCounts(t *testing.T) {
	t.Parallel()
	srv, st := newTestServer(t)
	st.Append(ingestion(track(0, "A")))
	st.Append(rejected("auth"))

	summary := summaryOf(t, srv.URL+"/_dev/v1/events?statusCode=400")

	require.Equal(t, float64(1), summary["requests"].(map[string]any)["total"])
	require.Equal(t, map[string]any{}, summary["byEvent"])
}

func TestFieldsFillEventsWithoutAView(t *testing.T) {
	t.Parallel()
	srv, st := newTestServer(t)
	st.Append(ingestion(track(0, "A")))

	_, page := get(t, srv.URL+"/_dev/v1/events?event=A&fields=properties")

	require.Equal(t, "fields", page["view"])
	require.Equal(t, []any{map[string]any{"seq": float64(1), "idx": float64(0), "type": "track", "event": "A",
		"properties": map[string]any{"n": float64(1)}}}, page["events"])
}

func TestSummaryDiagnosis(t *testing.T) {
	t.Parallel()

	for name, tc := range map[string]struct {
		records []store.Record
		want    []string
	}{
		"empty":           {nil, []string{"nothing_received"}},
		"preflight only":  {[]store.Record{control("/v1/track", http.MethodOptions, 204)}, []string{"preflight_only"}},
		"config rejected": {[]store.Record{control("/sourceConfig", http.MethodGet, 401)}, []string{"sdk_config_rejected"}},
		"config, no events": {[]store.Record{control("/sourceConfig", http.MethodGet, 200)},
			[]string{"sdk_loaded_no_events"}},
		"body rejected": {[]store.Record{rejected("identity")}, []string{"body_rejected"}},
		"auth and body": {[]store.Record{rejected("auth"), rejected("parse")}, []string{"auth_rejected", "body_rejected"}},
		"all accepted":  {[]store.Record{control("/sourceConfig", http.MethodGet, 200), ingestion(track(0, "A"))}, []string{"all_accepted"}},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			srv, st := newTestServer(t)
			for _, rec := range tc.records {
				st.Append(rec)
			}
			require.Equal(t, tc.want, diagnosisCodes(t, summaryOf(t, srv.URL+"/_dev/v1/events")))
		})
	}
}

func TestAllAcceptedNamesTheListCallAndItsCaveat(t *testing.T) {
	t.Parallel()
	srv, st := newTestServer(t)
	st.Append(ingestion(track(0, "A")))
	st.Append(ingestion(track(0, "B")))

	_, page := get(t, srv.URL+"/_dev/v1/events?since=1&event=B&view=counts")

	require.Equal(t, float64(1), page["since"])
	require.Equal(t, float64(2), page["cursor"])
	d := page["summary"].(map[string]any)["diagnosis"].([]any)[0].(map[string]any)
	require.Equal(t, "rudder-cli dev events list --since 1 --event B --json", d["next"])
	require.Contains(t, d["message"], "says nothing about events that never arrived")
}

func TestNothingReceivedNamesACurlTrack(t *testing.T) {
	t.Parallel()
	srv, _ := newTestServer(t)

	d := summaryOf(t, srv.URL+"/_dev/v1/events")["diagnosis"].([]any)[0].(map[string]any)

	require.Equal(t, "curl -fsS -u 'dev:' -H 'Content-Type: application/json' "+
		`-d '{"event":"dev check","userId":"dev"}' 'http://127.0.0.1:4321/v1/track'`, d["next"])
}

func sortedKeys(m map[string]any) []string {
	return slices.Sorted(maps.Keys(m))
}

// Each rejection diagnosis names the requests behind it, not one shared list.
func TestRejectionDiagnosesNameTheirOwnRequests(t *testing.T) {
	t.Parallel()
	srv, st := newTestServer(t)
	st.Append(ingestion(track(0, "A")))
	auth := rejected("auth")
	auth.StatusCode = 401
	st.Append(auth)
	st.Append(rejected("parse"))

	nexts := map[string]any{}
	for _, d := range summaryOf(t, srv.URL+"/_dev/v1/events?since=1")["diagnosis"].([]any) {
		nexts[d.(map[string]any)["code"].(string)] = d.(map[string]any)["next"]
	}

	require.Equal(t, "rudder-cli dev requests list --since 1 --status-code 401 --json", nexts["auth_rejected"])
	require.Equal(t, "rudder-cli dev requests list --since 1 --failed --status-code 400 --status-code 413 --json",
		nexts["body_rejected"])
}

func TestStatusCodeClassAndEventPrefix(t *testing.T) {
	t.Parallel()
	srv, st := newTestServer(t)
	st.Append(ingestion(track(0, "Order Completed"), track(1, "Order Refunded"), track(2, "Page Viewed")))
	bad := rejected("parse")
	bad.Events = []store.Event{track(0, "Order Failed")}
	st.Append(bad)

	_, byClass := get(t, srv.URL+"/_dev/v1/events?statusCode=4xx")
	_, byPrefix := get(t, srv.URL+"/_dev/v1/events?event=Order*")
	_, prefixCounts := get(t, srv.URL+"/_dev/v1/events?event=Order*&view=counts")
	status, _ := get(t, srv.URL+"/_dev/v1/events?statusCode=9xx")

	require.Len(t, byClass["events"], 1)
	require.Len(t, byPrefix["events"], 3)
	require.Equal(t, map[string]any{"Order Completed": float64(1), "Order Refunded": float64(1),
		"Order Failed": float64(1)}, prefixCounts["summary"].(map[string]any)["byEvent"], "a pattern adds no zero entry")
	require.Equal(t, 400, status)
}
