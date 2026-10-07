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

// execRecord builds a KindExec record with a realistic command duration
// (long relative to test2json's sub-millisecond run-event lag, short
// relative to the multi-second gaps between subtests), so the eligibility
// filter in currentTest has the same order-of-magnitude margin a real run
// gives it.
func execRecord(start time.Time, argv ...string) demo.Record {
	return demo.Record{Kind: demo.KindExec, Start: start, End: start.Add(1500 * time.Millisecond), Argv: argv}
}

// This is the regression fixture: it reproduces the real TestAccountsApply
// trace that exposed R19's bug (see the coordinator's measured deltas).
// test2json writes `run` asynchronously, so it lags the test body by a
// fraction of a millisecond — for apply_update and the re-apply subtest, the
// command's own record starts *before* that subtest's `run` event exists.
// "Most recent run at or before the record" (R18) puts both of those records
// on the wrong, earlier subtest. Only nearest-to-start survives this.
func TestJoinAttributesRecordsByNearestRun(t *testing.T) {
	base := at(0)
	events := []Event{
		{Time: base, Action: "run", Test: "TestAccountsApply"},
		{Time: base.Add(2 * time.Second), Action: "run", Test: "TestAccountsApply/apply_create"},
		{Time: base.Add(4 * time.Second), Action: "run", Test: "TestAccountsApply/apply_update"},
		{Time: base.Add(6 * time.Second), Action: "run", Test: "TestAccountsApply/re-apply_leaves_state_unchanged"},
		{Time: base.Add(9 * time.Second), Action: "pass", Test: "TestAccountsApply/apply_create"},
		{Time: base.Add(9 * time.Second), Action: "pass", Test: "TestAccountsApply/apply_update"},
		{Time: base.Add(9 * time.Second), Action: "pass", Test: "TestAccountsApply/re-apply_leaves_state_unchanged"},
		{Time: base.Add(9 * time.Second), Action: "pass", Test: "TestAccountsApply"},
	}

	// TestMain's `go build` finishes well before TestAccountsApply's own
	// `run` — no test was active yet, so it must be dropped.
	goBuild := execRecord(base.Add(-1*time.Second), "go", "build", "-o", "/tmp/rudder-cli")
	goBuild.End = base.Add(-500 * time.Millisecond)

	destroy := execRecord(base.Add(405*time.Microsecond), "rudder-cli", "destroy", "--confirm=false")
	applyCreate := execRecord(base.Add(2*time.Second+247*time.Microsecond), "rudder-cli", "apply", "-l", "accounts/create")
	// Starts 260µs BEFORE its own subtest's `run` event.
	applyUpdate := execRecord(base.Add(4*time.Second-260*time.Microsecond), "rudder-cli", "apply", "-l", "accounts/update")
	// Starts 69µs BEFORE its own subtest's `run` event.
	applyReapply := execRecord(base.Add(6*time.Second-69*time.Microsecond), "rudder-cli", "apply", "-l", "accounts/update")

	records := []demo.Record{goBuild, destroy, applyCreate, applyUpdate, applyReapply}

	got, err := Join(events, records)
	require.NoError(t, err)

	assert.Equal(t, []Step{
		{Test: "TestAccountsApply", Records: []demo.Record{destroy}},
		{Test: "TestAccountsApply/apply_create", Records: []demo.Record{applyCreate}},
		{Test: "TestAccountsApply/apply_update", Records: []demo.Record{applyUpdate}},
		{Test: "TestAccountsApply/re-apply_leaves_state_unchanged", Records: []demo.Record{applyReapply}},
	}, got)
}

// Isolates the sub-millisecond-early case from the larger regression fixture:
// a record that starts a fraction of a millisecond before its own subtest's
// `run` must still attribute to that subtest, not the previous one, because
// it is far nearer to the later run than to the earlier one.
func TestJoinAttributesRecordJustBeforeItsOwnRunToThatTest(t *testing.T) {
	events := []Event{
		{Time: at(0), Action: "run", Test: "TestA/first"},
		{Time: at(5), Action: "run", Test: "TestA/second"},
		{Time: at(9), Action: "pass", Test: "TestA/first"},
		{Time: at(9), Action: "pass", Test: "TestA/second"},
	}
	rec := execRecord(at(5).Add(-100*time.Microsecond), "rudder-cli", "apply")
	records := []demo.Record{rec}

	got, err := Join(events, records)
	require.NoError(t, err)

	require.Len(t, got, 1)
	assert.Equal(t, "TestA/second", got[0].Test)
}

// A record that finishes before any test has started — TestMain's `go
// build`, notably — has no eligible run event and must be dropped, not
// attributed to whichever test starts next.
func TestJoinDropsRecordThatFinishesBeforeAnyTestStarts(t *testing.T) {
	events := []Event{
		{Time: at(5), Action: "run", Test: "TestA"},
		{Time: at(9), Action: "pass", Test: "TestA"},
	}
	goBuild := execRecord(at(1), "go", "build", "-o", "/tmp/rudder-cli")
	goBuild.End = at(1).Add(500 * time.Millisecond) // still finishes well before TestA's run at at(5)
	applyRecord := execRecord(at(6), "rudder-cli", "apply")
	records := []demo.Record{goBuild, applyRecord}

	got, err := Join(events, records)
	require.NoError(t, err)

	require.Len(t, got, 1)
	assert.Equal(t, []string{"rudder-cli", "apply"}, got[0].Records[0].Argv)
}

// A record can land on the parent even though the parent has subtests, as
// long as it starts before the first subtest's `run` — e.g. setup work the
// parent test does itself before delegating to t.Run.
func TestJoinAttributesToParentBeforeAnySubtestStarts(t *testing.T) {
	events := []Event{
		{Time: at(0), Action: "run", Test: "TestAccountsApply"},
		{Time: at(5), Action: "run", Test: "TestAccountsApply/apply_create"},
		{Time: at(9), Action: "pass", Test: "TestAccountsApply/apply_create"},
		{Time: at(9), Action: "pass", Test: "TestAccountsApply"},
	}
	destroy := execRecord(at(1), "rudder-cli", "destroy", "--confirm=false")
	records := []demo.Record{destroy}

	got, err := Join(events, records)
	require.NoError(t, err)

	assert.Equal(t, []Step{{Test: "TestAccountsApply", Records: []demo.Record{destroy}}}, got)
}

func TestJoinKeepsNarrationInStreamOrder(t *testing.T) {
	events := []Event{
		{Time: at(0), Action: "run", Test: "TestA"},
		{Time: at(9), Action: "pass", Test: "TestA"},
	}
	say := demo.Record{Kind: demo.KindSay, Start: at(1), End: at(1), Text: "why this matters"}
	exec := execRecord(at(2), "rudder-cli", "apply")
	records := []demo.Record{say, exec}

	got, err := Join(events, records)
	require.NoError(t, err)

	require.Len(t, got, 1)
	require.Len(t, got[0].Records, 2)
	assert.Equal(t, demo.KindSay, got[0].Records[0].Kind)
	assert.Equal(t, demo.KindExec, got[0].Records[1].Kind)
}

// Ruling R23 regression fixture: the exact shape measured recording
// TestAccountsApply with demo.Say placed as the first statement of each
// subtest, before R23 existed. R19's nearest-run rule has no duration to
// work with on a demo.KindSay record (Start == End), so test2json's
// sub-millisecond lag on flushing a subtest's `run` event landed each of
// these on the wrong side of it:
//
//   - the top-level say ran 197µs *before* TestAccountsApply's own run event
//     had even been flushed — no eligible run at all under R19, dropped.
//   - apply_update's say ran 81µs before its own run flushed — R19's nearest
//     eligible run was apply_create's, two seconds earlier.
//   - the re-apply say ran 295µs before its own run flushed — R19's nearest
//     eligible run was apply_update's.
//
// Only apply_create's say happened to land correctly under R19 (its run
// flushed 32µs before the say) — one out of four. R23 fixes all four by
// binding each say to the step of the exec that follows it in journal order,
// which needs no timestamp comparison at all.
func TestJoinBindsSayToItsFollowingExecEvenWhenTimestampFallsInThePreviousWindow(t *testing.T) {
	base := at(0)
	events := []Event{
		{Time: base, Action: "run", Test: "TestAccountsApply"},
		{Time: base.Add(2 * time.Second), Action: "run", Test: "TestAccountsApply/apply_create"},
		{Time: base.Add(4 * time.Second), Action: "run", Test: "TestAccountsApply/apply_update"},
		{Time: base.Add(6 * time.Second), Action: "run", Test: "TestAccountsApply/re-apply"},
		{Time: base.Add(9 * time.Second), Action: "pass", Test: "TestAccountsApply/apply_create"},
		{Time: base.Add(9 * time.Second), Action: "pass", Test: "TestAccountsApply/apply_update"},
		{Time: base.Add(9 * time.Second), Action: "pass", Test: "TestAccountsApply/re-apply"},
		{Time: base.Add(9 * time.Second), Action: "pass", Test: "TestAccountsApply"},
	}

	say := func(start time.Time, text string) demo.Record {
		return demo.Record{Kind: demo.KindSay, Start: start, End: start, Text: text}
	}

	topSay := say(base.Add(-197*time.Microsecond), "clean workspace")
	destroy := execRecord(base.Add(405*time.Microsecond), "rudder-cli", "destroy", "--confirm=false")

	createSay := say(base.Add(2*time.Second+32*time.Microsecond), "accounts created")
	applyCreate := execRecord(base.Add(2*time.Second+400*time.Microsecond), "rudder-cli", "apply", "-l", "accounts/create")

	updateSay := say(base.Add(4*time.Second-81*time.Microsecond), "updates in place")
	applyUpdate := execRecord(base.Add(4*time.Second+400*time.Microsecond), "rudder-cli", "apply", "-l", "accounts/update")

	reapplySay := say(base.Add(6*time.Second-295*time.Microsecond), "no-op except the secret")
	applyReapply := execRecord(base.Add(6*time.Second+400*time.Microsecond), "rudder-cli", "apply", "-l", "accounts/update")

	records := []demo.Record{
		topSay, destroy,
		createSay, applyCreate,
		updateSay, applyUpdate,
		reapplySay, applyReapply,
	}

	got, err := Join(events, records)
	require.NoError(t, err)

	assert.Equal(t, []Step{
		{Test: "TestAccountsApply", Records: []demo.Record{topSay, destroy}},
		{Test: "TestAccountsApply/apply_create", Records: []demo.Record{createSay, applyCreate}},
		{Test: "TestAccountsApply/apply_update", Records: []demo.Record{updateSay, applyUpdate}},
		{Test: "TestAccountsApply/re-apply", Records: []demo.Record{reapplySay, applyReapply}},
	}, got, "every say must land with the exec that follows it, not with whichever subtest its own timestamp is nearest to")
}

// A trailing say with nothing after it falls back to the exec that precedes
// it — there is no step left to introduce, so it reads as commentary on what
// just ran.
func TestJoinBindsTrailingSayToPrecedingExec(t *testing.T) {
	events := []Event{
		{Time: at(0), Action: "run", Test: "TestA"},
		{Time: at(9), Action: "pass", Test: "TestA"},
	}
	exec := execRecord(at(1), "rudder-cli", "apply")
	trailing := demo.Record{Kind: demo.KindSay, Start: at(3), End: at(3), Text: "and that proves it"}
	records := []demo.Record{exec, trailing}

	got, err := Join(events, records)
	require.NoError(t, err)

	require.Len(t, got, 1)
	assert.Equal(t, []demo.Record{exec, trailing}, got[0].Records)
}

// A say with no exec anywhere in the recording — before or after — has
// nothing to introduce and nothing to comment on, so it is dropped like any
// other record with no eligible owner.
func TestJoinDropsSayWithNoExecOnEitherSide(t *testing.T) {
	events := []Event{
		{Time: at(0), Action: "run", Test: "TestA"},
		{Time: at(9), Action: "pass", Test: "TestA"},
	}
	records := []demo.Record{{Kind: demo.KindSay, Start: at(1), End: at(1), Text: "orphaned"}}

	got, err := Join(events, records)
	require.NoError(t, err)

	assert.Empty(t, got)
}

func TestJoinSkipsTestsThatRanNoCommands(t *testing.T) {
	events := []Event{
		{Time: at(0), Action: "run", Test: "TestA"},
		{Time: at(1), Action: "run", Test: "TestA/empty"},
		{Time: at(2), Action: "pass", Test: "TestA/empty"},
		{Time: at(3), Action: "run", Test: "TestA/busy"},
		{Time: at(5), Action: "pass", Test: "TestA/busy"},
	}
	records := []demo.Record{execRecord(at(4), "rudder-cli", "apply")}

	got, err := Join(events, records)
	require.NoError(t, err)

	require.Len(t, got, 1)
	assert.Equal(t, "TestA/busy", got[0].Test)
}
