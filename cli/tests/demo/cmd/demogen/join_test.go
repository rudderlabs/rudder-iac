package main

import (
	"testing"
	"time"

	"github.com/rudderlabs/rudder-iac/cli/tests/demo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func at(sec int) time.Time {
	return time.Date(2026, 9, 20, 10, 0, sec, 0, time.UTC)
}

// This is the test that would have caught R18's bug: go test -json batches
// every subtest's pass event at the instant the parent completes, so all
// three siblings here "pass" at the identical timestamp — mirroring the real
// TestAccountsApply trace that exposed the defect. A depth/span-based
// attribution treated all three as simultaneously "active" and collapsed
// every record onto whichever subtest happened to start first. Attribution
// must instead follow `run` order, which go test never batches.
func TestJoinAttributesRecordsBySubtestsMostRecentRun(t *testing.T) {
	events := []Event{
		{Time: at(0), Action: "run", Test: "TestAccountsApply"},
		{Time: at(1), Action: "run", Test: "TestAccountsApply/apply_create"},
		{Time: at(3), Action: "run", Test: "TestAccountsApply/apply_update"},
		{Time: at(5), Action: "run", Test: "TestAccountsApply/re-apply_leaves_state_unchanged"},
		{Time: at(9), Action: "pass", Test: "TestAccountsApply/apply_create"},
		{Time: at(9), Action: "pass", Test: "TestAccountsApply/apply_update"},
		{Time: at(9), Action: "pass", Test: "TestAccountsApply/re-apply_leaves_state_unchanged"},
		{Time: at(9), Action: "pass", Test: "TestAccountsApply"},
	}
	records := []demo.Record{
		{Kind: demo.KindExec, Start: at(2), Argv: []string{"rudder-cli", "apply", "-l", "accounts/create"}},
		{Kind: demo.KindExec, Start: at(4), Argv: []string{"rudder-cli", "apply", "-l", "accounts/update"}},
		{Kind: demo.KindExec, Start: at(6), Argv: []string{"rudder-cli", "apply", "-l", "accounts/reapply"}},
	}

	got, err := Join(events, records)
	require.NoError(t, err)

	assert.Equal(t, []Step{
		{
			Test:    "TestAccountsApply/apply_create",
			Records: []demo.Record{{Kind: demo.KindExec, Start: at(2), Argv: []string{"rudder-cli", "apply", "-l", "accounts/create"}}},
		},
		{
			Test:    "TestAccountsApply/apply_update",
			Records: []demo.Record{{Kind: demo.KindExec, Start: at(4), Argv: []string{"rudder-cli", "apply", "-l", "accounts/update"}}},
		},
		{
			Test:    "TestAccountsApply/re-apply_leaves_state_unchanged",
			Records: []demo.Record{{Kind: demo.KindExec, Start: at(6), Argv: []string{"rudder-cli", "apply", "-l", "accounts/reapply"}}},
		},
	}, got)
}

func TestJoinAttributesToParentWhenNoSubtestActive(t *testing.T) {
	events := []Event{
		{Time: at(0), Action: "run", Test: "TestAccountsApply"},
		{Time: at(5), Action: "pass", Test: "TestAccountsApply"},
	}
	records := []demo.Record{
		{Kind: demo.KindExec, Start: at(1), Argv: []string{"rudder-cli", "destroy", "--confirm=false"}},
	}

	got, err := Join(events, records)
	require.NoError(t, err)

	assert.Equal(t, []Step{{
		Test:    "TestAccountsApply",
		Records: []demo.Record{{Kind: demo.KindExec, Start: at(1), Argv: []string{"rudder-cli", "destroy", "--confirm=false"}}},
	}}, got)
}

// A record can also land on the parent while subtests exist, as long as it
// starts before the first subtest's `run` — e.g. setup the parent test does
// itself before delegating to t.Run.
func TestJoinAttributesToParentBeforeAnySubtestStarts(t *testing.T) {
	events := []Event{
		{Time: at(0), Action: "run", Test: "TestAccountsApply"},
		{Time: at(5), Action: "run", Test: "TestAccountsApply/apply_create"},
		{Time: at(9), Action: "pass", Test: "TestAccountsApply/apply_create"},
		{Time: at(9), Action: "pass", Test: "TestAccountsApply"},
	}
	records := []demo.Record{
		{Kind: demo.KindExec, Start: at(1), Argv: []string{"rudder-cli", "destroy", "--confirm=false"}},
	}

	got, err := Join(events, records)
	require.NoError(t, err)

	assert.Equal(t, []Step{{
		Test:    "TestAccountsApply",
		Records: []demo.Record{{Kind: demo.KindExec, Start: at(1), Argv: []string{"rudder-cli", "destroy", "--confirm=false"}}},
	}}, got)
}

func TestJoinKeepsNarrationInStreamOrder(t *testing.T) {
	events := []Event{
		{Time: at(0), Action: "run", Test: "TestA"},
		{Time: at(9), Action: "pass", Test: "TestA"},
	}
	records := []demo.Record{
		{Kind: demo.KindSay, Start: at(1), Text: "why this matters"},
		{Kind: demo.KindExec, Start: at(2), Argv: []string{"rudder-cli", "apply"}},
	}

	got, err := Join(events, records)
	require.NoError(t, err)

	require.Len(t, got, 1)
	require.Len(t, got[0].Records, 2)
	assert.Equal(t, demo.KindSay, got[0].Records[0].Kind)
	assert.Equal(t, demo.KindExec, got[0].Records[1].Kind)
}

func TestJoinDropsRecordsOutsideAnyTest(t *testing.T) {
	// TestMain builds the binary before the first `run` event; that go build
	// invocation is real but belongs to no test and must not open a demo.
	events := []Event{
		{Time: at(5), Action: "run", Test: "TestA"},
		{Time: at(9), Action: "pass", Test: "TestA"},
	}
	records := []demo.Record{
		{Kind: demo.KindExec, Start: at(1), Argv: []string{"go", "build", "-o", "/tmp/rudder-cli"}},
		{Kind: demo.KindExec, Start: at(6), Argv: []string{"rudder-cli", "apply"}},
	}

	got, err := Join(events, records)
	require.NoError(t, err)

	require.Len(t, got, 1)
	assert.Equal(t, []string{"rudder-cli", "apply"}, got[0].Records[0].Argv)
}

func TestJoinSkipsTestsThatRanNoCommands(t *testing.T) {
	events := []Event{
		{Time: at(0), Action: "run", Test: "TestA"},
		{Time: at(1), Action: "run", Test: "TestA/empty"},
		{Time: at(2), Action: "pass", Test: "TestA/empty"},
		{Time: at(3), Action: "run", Test: "TestA/busy"},
		{Time: at(5), Action: "pass", Test: "TestA/busy"},
	}
	records := []demo.Record{{Kind: demo.KindExec, Start: at(4), Argv: []string{"rudder-cli", "apply"}}}

	got, err := Join(events, records)
	require.NoError(t, err)

	require.Len(t, got, 1)
	assert.Equal(t, "TestA/busy", got[0].Test)
}
