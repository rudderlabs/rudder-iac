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

// A well-formed serial trace never has two active spans at the same depth: an
// active span's ancestors are also active, so concurrently active depths form
// a strictly increasing chain. The one way to get a same-depth tie is a span
// that never closed — e.g. a subtest whose pass/fail event was lost — left
// open when an unrelated, later span at the same depth starts. deepestActive
// must not let that later span silently steal a timestamp that arrived while
// the earlier span was still the best (and only) match; ties keep whichever
// span claimed the depth first.
func TestDeepestActiveKeepsEarlierSpanOnDepthTie(t *testing.T) {
	spans := []span{
		{test: "TestA", start: 0, stop: maxInt64},
		{test: "TestB", start: 5, stop: 15},
	}

	assert.Equal(t, "TestA", deepestActive(spans, 7))
}
