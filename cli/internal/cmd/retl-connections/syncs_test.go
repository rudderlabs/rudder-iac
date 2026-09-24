package retlconnection

import (
	"testing"
	"time"

	retlClient "github.com/rudderlabs/rudder-iac/api/client/retl"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRenderSyncTable(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 9, 20, 12, 0, 0, 0, time.UTC)
	start := now.Add(-2 * time.Hour)

	// Both rows come back from the API as "succeeded". Only one of them
	// delivered everything, and a history that showed them identically would
	// hide the run that dropped records.
	out := renderSyncTable([]retlClient.Sync{
		{
			ID: "run-clean", Status: retlClient.SyncSucceeded,
			StartedAt: start, FinishedAt: start.Add(30 * time.Second),
			Metrics: retlClient.SyncMetrics{Changed: retlClient.SyncRowMetrics{Total: 5}},
		},
		{
			ID: "run-lossy", Status: retlClient.SyncSucceeded,
			StartedAt: start, FinishedAt: start.Add(45 * time.Second),
			Metrics: retlClient.SyncMetrics{
				Changed: retlClient.SyncRowMetrics{Total: 9},
				Failed:  retlClient.SyncRowMetrics{Total: 4},
			},
		},
	}, now)

	assert.Contains(t, out, "RUN ID")
	assert.Contains(t, out, "run-clean")
	assert.Contains(t, out, "succeeded")
	assert.Contains(t, out, "run-lossy")
	assert.Contains(t, out, "failed", "an api success that dropped rows must not read as a success")
	assert.Contains(t, out, "2h ago")
}

func TestRelativeTime(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 9, 20, 12, 0, 0, 0, time.UTC)

	for _, tc := range []struct {
		name string
		at   time.Time
		want string
	}{
		{"seconds ago reads as just now", now.Add(-20 * time.Second), "just now"},
		{"minutes", now.Add(-5 * time.Minute), "5m ago"},
		{"hours", now.Add(-3 * time.Hour), "3h ago"},
		{"days", now.Add(-50 * time.Hour), "2d ago"},
		{"a zero time has no answer", time.Time{}, "-"},
		{"clock skew does not print a negative", now.Add(time.Minute), "just now"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.want, relativeTime(tc.at, now))
		})
	}
}

func TestSyncsCommandWiring(t *testing.T) {
	t.Parallel()

	var syncs *struct{ found bool }
	for _, c := range NewCmdRetlConnections().Commands() {
		if c.Name() == "syncs" {
			syncs = &struct{ found bool }{true}

			names := map[string]bool{}
			for _, sub := range c.Commands() {
				names[sub.Name()] = true
			}
			assert.True(t, names["list"])
			assert.True(t, names["view"])

			for _, sub := range c.Commands() {
				require.NotNil(t, sub.Flags().Lookup("json"), sub.Name()+" needs --json")
				if sub.Name() == "list" {
					require.NotNil(t, sub.Flags().Lookup("limit"))
					require.NotNil(t, sub.Flags().Lookup("status"))
					require.NotNil(t, sub.Flags().Lookup("since"))
					assert.Equal(t, "20", sub.Flags().Lookup("limit").DefValue)
				}
			}
		}
	}
	require.NotNil(t, syncs, "syncs must be registered under retl-connections")
}
