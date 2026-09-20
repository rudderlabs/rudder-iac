package retl_test

import (
	"testing"
	"time"

	retl "github.com/rudderlabs/rudder-iac/api/client/retl"
	"github.com/stretchr/testify/assert"
)

// The API's status and the outcome a reader is shown invert for two of the
// four cases. This is the whole reason Outcome exists, so it is pinned by
// example rather than by restating the switch.
func TestSyncOutcome(t *testing.T) {
	t.Parallel()

	sync := func(status retl.SyncStatus, failed int) retl.Sync {
		return retl.Sync{
			Status:  status,
			Metrics: retl.SyncMetrics{Failed: retl.SyncRowMetrics{Total: failed}},
		}
	}

	tests := map[string]struct {
		sync      retl.Sync
		outcome   retl.SyncOutcome
		delivered bool
	}{
		"a run still going is syncing":                  {sync(retl.SyncRunning, 0), retl.OutcomeSyncing, false},
		"a clean run succeeded":                         {sync(retl.SyncSucceeded, 0), retl.OutcomeSucceeded, true},
		"an api success with failed rows is a failure":  {sync(retl.SyncSucceeded, 3), retl.OutcomeFailed, false},
		"an api failure is an aborted run":              {sync(retl.SyncFailed, 0), retl.OutcomeAborted, false},
		"an aborted run with failed rows is still that": {sync(retl.SyncFailed, 7), retl.OutcomeAborted, false},
		"an unknown status is reported as-is":           {sync("paused", 0), retl.SyncOutcome("paused"), false},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.outcome, tc.sync.Outcome())
			assert.Equal(t, tc.delivered, tc.sync.Delivered())
		})
	}
}

func TestSyncDuration(t *testing.T) {
	t.Parallel()

	start := time.Date(2026, 9, 20, 10, 0, 0, 0, time.UTC)
	now := start.Add(5 * time.Minute)

	t.Run("a finished run measures to its end", func(t *testing.T) {
		t.Parallel()

		s := retl.Sync{Status: retl.SyncSucceeded, StartedAt: start, FinishedAt: start.Add(90 * time.Second)}
		assert.Equal(t, 90*time.Second, s.Duration(now))
	})

	// finishedAt carries no meaning until the run ends, and upstream has sent a
	// zero time there in the past.
	t.Run("a running sync measures to now", func(t *testing.T) {
		t.Parallel()

		s := retl.Sync{Status: retl.SyncRunning, StartedAt: start}
		assert.Equal(t, 5*time.Minute, s.Duration(now))
	})

	t.Run("a finished time before the start is not believed", func(t *testing.T) {
		t.Parallel()

		s := retl.Sync{Status: retl.SyncSucceeded, StartedAt: start, FinishedAt: start.Add(-time.Hour)}
		assert.Equal(t, 5*time.Minute, s.Duration(now))
	})
}
