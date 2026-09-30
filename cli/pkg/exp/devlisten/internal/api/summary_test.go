package api

import (
	"net/http"
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

func TestSummaryCounts(t *testing.T) {
	t.Parallel()
	srv, st := newTestServer(t)
	st.Append(control("/sourceConfig", http.MethodGet, 200))
	st.Append(control("/v1/batch", http.MethodOptions, 204))
	st.Append(ingestion(track(0, "A"), track(1, "B")))
	st.Append(rejected("auth"))

	status, summary := get(t, srv.URL+"/_dev/v1/summary?since=0")

	require.Equal(t, http.StatusOK, status)
	require.Equal(t, map[string]any{
		"total": float64(2), "failed": float64(1), "probes": float64(0),
		"byRoute":      map[string]any{"/v1/batch": float64(2)},
		"byStatusCode": map[string]any{"200": float64(1), "400": float64(1)},
		"byOutcome":    map[string]any{"accepted": float64(1), "rejected": float64(1)},
		"byStage":      map[string]any{"auth": float64(1)},
	}, summary["requests"])
	require.Equal(t, map[string]any{
		"total": float64(2), "byType": map[string]any{"track": float64(2)},
		"byEvent": map[string]any{"A": float64(1), "B": float64(1)},
	}, summary["events"])
	require.Equal(t, map[string]any{
		"total": float64(2), "sourceConfig": float64(1), "sourceConfigFailed": float64(0),
		"preflight": float64(1), "pluginPath": float64(0), "other": float64(0),
	}, summary["control"])
	require.Equal(t, []any{map[string]any{
		"code": "auth_rejected", "count": float64(1),
		"message": "Requests were rejected at the auth stage. Compare their writeKey with the SDK configuration.",
		"next":    "rudder-cli dev requests list --failed --stage auth --json",
	}}, summary["diagnosis"])
}

func TestSummaryDiagnosis(t *testing.T) {
	t.Parallel()

	probe := ingestion(track(0, "probe"))
	probe.Probe = true

	for name, tc := range map[string]struct {
		records []store.Record
		want    []string
	}{
		"empty":           {nil, []string{"nothing_received"}},
		"probe only":      {[]store.Record{probe}, []string{"nothing_received"}},
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
			_, summary := get(t, srv.URL+"/_dev/v1/summary")
			require.Equal(t, tc.want, diagnosisCodes(t, summary))
		})
	}
}

func TestSummaryAllAcceptedNamesTheListCall(t *testing.T) {
	t.Parallel()
	srv, st := newTestServer(t)
	st.Append(ingestion(track(0, "A")))
	st.Append(ingestion(track(0, "B")))

	_, summary := get(t, srv.URL+"/_dev/v1/summary?since=1")

	require.Equal(t, float64(1), summary["since"])
	require.Equal(t, float64(2), summary["cursor"])
	diagnosis := summary["diagnosis"].([]any)[0].(map[string]any)
	require.Equal(t, "rudder-cli dev events list --since 1 --view summary --json", diagnosis["next"])
}
