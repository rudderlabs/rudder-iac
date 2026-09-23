package retlconnection

import (
	"testing"
	"time"

	retlClient "github.com/rudderlabs/rudder-iac/api/client/retl"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRenderSync(t *testing.T) {
	t.Parallel()

	start := time.Date(2026, 9, 20, 10, 0, 0, 0, time.UTC)
	now := start.Add(10 * time.Minute)

	// The API calls this run a success. It dropped three rows, and the webapp
	// shows it as failed; printing the raw status would tell an author their
	// sync worked.
	t.Run("an api success with failed rows renders as failed", func(t *testing.T) {
		t.Parallel()

		out := renderSync(retlClient.Sync{
			ID:         "run-1",
			Status:     retlClient.SyncSucceeded,
			StartedAt:  start,
			FinishedAt: start.Add(90 * time.Second),
			Metrics: retlClient.SyncMetrics{
				Changed:   retlClient.SyncRowMetrics{Total: 10},
				Succeeded: retlClient.SyncRowMetrics{Total: 7},
				Failed:    retlClient.SyncRowMetrics{Total: 3},
			},
		}, now)

		assert.Contains(t, out, "sync run-1  failed")
		assert.NotContains(t, out, "succeeded\n")
		assert.Contains(t, out, "10 changed, 7 delivered, 3 failed")
		assert.Contains(t, out, "duration  1m 30s")
	})

	t.Run("a clean run renders as succeeded", func(t *testing.T) {
		t.Parallel()

		out := renderSync(retlClient.Sync{
			ID:         "run-2",
			Status:     retlClient.SyncSucceeded,
			StartedAt:  start,
			FinishedAt: start.Add(5 * time.Second),
			Metrics:    retlClient.SyncMetrics{Changed: retlClient.SyncRowMetrics{Total: 5}, Succeeded: retlClient.SyncRowMetrics{Total: 5}},
		}, now)

		assert.Contains(t, out, "sync run-2  succeeded")
	})

	// The reason a run died is why anyone looks at it, so it survives whole.
	t.Run("an aborted run keeps its reason intact", func(t *testing.T) {
		t.Parallel()

		reason := "could not create database client of type 'postgres': parsing postgres credentials: json: cannot unmarshal string into Go struct field Config.port of type int"
		out := renderSync(retlClient.Sync{
			ID:        "run-3",
			Status:    retlClient.SyncFailed,
			StartedAt: start,
			Error:     reason,
		}, now)

		assert.Contains(t, out, "sync run-3  aborted")
		assert.Contains(t, out, reason)
	})
}

func TestCompactDuration(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		in   time.Duration
		want string
	}{
		{0, "0s"},
		{45 * time.Second, "45s"},
		{90 * time.Second, "1m 30s"},
		{2 * time.Hour, "2h 0m"},
		{-5 * time.Second, "0s"},
	} {
		assert.Equal(t, tc.want, compactDuration(tc.in), tc.in.String())
	}
}

func TestCommandWiring(t *testing.T) {
	t.Parallel()

	cmd := NewCmdRetlConnections()
	require.NotNil(t, cmd)
	assert.Equal(t, "retl-connections", cmd.Use)

	names := map[string]*struct{}{}
	for _, c := range cmd.Commands() {
		names[c.Name()] = nil
	}
	assert.Contains(t, names, "sync")
	assert.Contains(t, names, "stop", "a sync that cannot be cancelled is half a feature")

	var sync = cmd.Commands()[0]
	for _, c := range cmd.Commands() {
		if c.Name() == "sync" {
			sync = c
		}
	}
	require.NotNil(t, sync.Flags().Lookup("wait"))
	assert.Equal(t, "true", sync.Flags().Lookup("wait").DefValue, "following the run to a verdict is the useful default")
	require.NotNil(t, sync.Flags().Lookup("type"))
	assert.Equal(t, "full", sync.Flags().Lookup("type").DefValue)
	require.NotNil(t, sync.Flags().Lookup("timeout"))
	require.NotNil(t, sync.Flags().Lookup("json"))
}
