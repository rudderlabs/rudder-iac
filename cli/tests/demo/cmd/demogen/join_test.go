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

func TestJoinAttributesRecordsToDeepestActiveSubtest(t *testing.T) {
	events := []Event{
		{Time: at(0), Action: "run", Test: "TestProjectApply"},
		{Time: at(1), Action: "run", Test: "TestProjectApply/rudder_specs"},
		{Time: at(2), Action: "run", Test: "TestProjectApply/rudder_specs/should_create_entities"},
		{Time: at(6), Action: "pass", Test: "TestProjectApply/rudder_specs/should_create_entities"},
		{Time: at(7), Action: "run", Test: "TestProjectApply/rudder_specs/should_update_entities"},
		{Time: at(9), Action: "pass", Test: "TestProjectApply/rudder_specs/should_update_entities"},
	}
	records := []demo.Record{
		{Kind: demo.KindExec, Start: at(3), Argv: []string{"rudder-cli", "apply", "-l", "create"}},
		{Kind: demo.KindExec, Start: at(5), Argv: []string{"rudder-cli", "apply", "-l", "create"}},
		{Kind: demo.KindExec, Start: at(8), Argv: []string{"rudder-cli", "apply", "-l", "update"}},
	}

	got, err := Join(events, records)
	require.NoError(t, err)

	assert.Equal(t, []Step{
		{
			Test: "TestProjectApply/rudder_specs/should_create_entities",
			Records: []demo.Record{
				{Kind: demo.KindExec, Start: at(3), Argv: []string{"rudder-cli", "apply", "-l", "create"}},
				{Kind: demo.KindExec, Start: at(5), Argv: []string{"rudder-cli", "apply", "-l", "create"}},
			},
		},
		{
			Test: "TestProjectApply/rudder_specs/should_update_entities",
			Records: []demo.Record{
				{Kind: demo.KindExec, Start: at(8), Argv: []string{"rudder-cli", "apply", "-l", "update"}},
			},
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

// A well-formed, complete serial trace never has two active spans at the same
// depth: an active span's ancestors are also active, so concurrently active
// depths form a strictly increasing chain, and go test emits a pass or fail
// for every test — panics included — so a span always closes. A same-depth
// tie can therefore only arise from a truncated or malformed stream, input
// this package should not be trusting in the first place.
//
// "Earlier wins" is not claimed to be the more correct answer for that
// degenerate case — "later wins" is arguably just as defensible. This test
// exists only to pin whichever choice depth > bestDepth already makes, so the
// tie-break can't drift silently if someone touches deepestActive later.
func TestDeepestActiveKeepsEarlierSpanOnDepthTie(t *testing.T) {
	spans := []span{
		{test: "TestA", start: 0, stop: maxInt64},
		{test: "TestB", start: 5, stop: 15},
	}

	assert.Equal(t, "TestA", deepestActive(spans, 7))
}
