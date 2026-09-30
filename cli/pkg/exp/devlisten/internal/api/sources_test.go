package api

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/rudderlabs/rudder-iac/cli/pkg/exp/devlisten/internal/store"
)

// sdkEvent is an event whose context names the SDK library and channel.
func sdkEvent(name, library, channel string) store.Event {
	msg := `{"type":"track","event":"` + name + `","userId":"u1","channel":"` + channel +
		`","context":{"library":{"name":"` + library + `"}}}`
	return store.Event{Type: sp("track"), Event: sp(name), UserID: sp("u1"), Message: json.RawMessage(msg)}
}

func withUserAgent(rec store.Record, ua string) store.Record {
	rec.Request.Headers = http.Header{"User-Agent": {ua}}
	return rec
}

func TestSummaryCountsBySource(t *testing.T) {
	t.Parallel()
	srv, st := newTestServer(t)
	browserUA := "Mozilla/5.0 (Macintosh) Chrome/140"
	st.Append(withUserAgent(control("/sourceConfig", http.MethodGet, 200), browserUA))
	st.Append(withUserAgent(ingestion(sdkEvent("Shown", "RudderLabs JavaScript SDK", "web")), browserUA))
	st.Append(withUserAgent(ingestion(sdkEvent("Order", "analytics-node", "server"),
		sdkEvent("Id", "analytics-node", "server")), "axios/1.7"))

	summary := summaryOf(t, srv.URL+"/_dev/v1/events")

	require.Equal(t, map[string]any{
		"byChannel": map[string]any{"web": float64(1), "server": float64(2)},
		"bySdk": map[string]any{
			"browser": map[string]any{"requests": float64(1), "events": float64(1), "control": float64(1)},
			"node":    map[string]any{"requests": float64(1), "events": float64(2), "control": float64(0)},
		},
	}, summary["bySource"])
}

// Server events without any browser request, config load or preflight
// point at the browser SDK setup, not at the listener.
func TestSummaryDiagnosesNoBrowserTraffic(t *testing.T) {
	t.Parallel()
	srv, st := newTestServer(t)
	st.Append(withUserAgent(ingestion(sdkEvent("Order", "analytics-node", "server")), "axios/1.7"))

	summary := summaryOf(t, srv.URL+"/_dev/v1/events")

	require.Equal(t, []string{"no_browser_traffic", "all_accepted"}, diagnosisCodes(t, summary))
	d := summary["diagnosis"].([]any)[0].(map[string]any)
	require.Equal(t, "rudder-cli dev requests list --since 0 --kind control --json", d["next"])
	require.Contains(t, d["message"], "check in order: 1. its configUrl is the listener URL")
}
