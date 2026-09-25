package main

import (
	"time"

	"github.com/rudderlabs/rudder-iac/cli/tests/demo"
)

// Step is one subtest and the records it ran, in order.
type Step struct {
	Test    string
	Records []demo.Record
}

// Join attributes each journal record to the subtest that was running when it
// started.
//
// A record's owner is the `run` event nearest to the record's start, among
// run events that began before the record finished. Both halves matter:
//
//   - The eligibility filter (a run event must be at or before the record's
//     End) is a hard causal constraint — a command cannot belong to a test
//     that started after the command had already ended. It is also what
//     drops TestMain's `go build`: no test has started before that build
//     finishes, so it has no eligible run and is dropped.
//   - Nearest-to-start, not most-recent-at-or-before, is required because
//     test2json writes `run` asynchronously: it lags the test body by a
//     fraction of a millisecond, so roughly half of a subtest's own first
//     command starts *before* that subtest's `run` event exists on the
//     stream. An earlier version of this function used "most recent run at
//     or before the record" (see git history for R18) and silently
//     misattributed exactly those records to the previous, still-open
//     subtest. Nearest-to-start survives that inversion without needing a
//     tolerance constant, because inter-test gaps in a real suite are
//     seconds while the event lag is sub-millisecond — four orders of
//     magnitude of margin. This assumption is load-bearing: a suite whose
//     commands complete in under a millisecond would need a different rule,
//     because nearest-to-start would stop being decisive.
//
// go test -json also batches every subtest's pass/fail event at the instant
// the parent completes, so pass/fail carries no reliable timing and cannot
// be used to bound a subtest's window at all — only eligibility (via each
// record's own End) and nearest-to-start (via run events) are used here.
// One ambiguity survives regardless: a record the *parent* emits after its
// last subtest's `run` but logically after that subtest's own work has
// finished is indistinguishable, on the wire, from a record the subtest
// itself emitted — there is no event marking that boundary. Such a record is
// attributed to the subtest.
//
// This is R19's rule, and it applies to demo.KindExec records only. A
// demo.KindSay record has Start == End — a single instant, no duration — so
// R19's own nearest-run comparison has no margin to work with: test2json's
// sub-millisecond lag on flushing a subtest's `run` event is exactly as
// likely to land on either side of that instant. Measured on a real
// TestAccountsApply run, a demo.Say placed as a subtest's first statement
// (the natural place to put narration that introduces a step) landed on the
// *previous* subtest's step three times out of four.
//
// Ruling R23: a say is bound to the command it introduces, not to its own
// timestamp. A demo.KindSay record's owner is the owner of the next
// demo.KindExec record in journal order — narration precedes and explains
// the command it sits in front of, so that is what it means to "introduce"
// a step. A trailing say with no following exec falls back to the owner of
// the preceding exec (there is nothing left to introduce, so it is treated
// as commentary on what just ran). A say with no exec on either side — the
// whole recording has no commands at all — attaches to nothing and is
// dropped, the same as any other record with no eligible owner.
func Join(events []Event, records []demo.Record) ([]Step, error) {
	var runs []Event
	for _, e := range events {
		if e.Action == "run" {
			runs = append(runs, e)
		}
	}

	owners := make([]string, len(records))
	for i, r := range records {
		if r.Kind == demo.KindExec {
			owners[i] = currentTest(runs, r.Start, r.End)
		}
	}

	for i, r := range records {
		if r.Kind != demo.KindSay {
			continue
		}
		owners[i] = nextExecOwner(records, owners, i)
		if owners[i] == "" {
			owners[i] = prevExecOwner(records, owners, i)
		}
	}

	byTest := map[string][]demo.Record{}
	var order []string

	for i, r := range records {
		owner := owners[i]
		if owner == "" {
			continue
		}

		if _, seen := byTest[owner]; !seen {
			order = append(order, owner)
		}
		byTest[owner] = append(byTest[owner], r)
	}

	// Emit in the order the tests first claimed a record, which for a serial
	// suite is the order a viewer will watch them in.
	out := make([]Step, 0, len(order))
	for _, test := range order {
		out = append(out, Step{Test: test, Records: byTest[test]})
	}

	return out, nil
}

// nextExecOwner returns the owner already computed for the nearest
// demo.KindExec record after i in journal order, or "" if none follows.
func nextExecOwner(records []demo.Record, owners []string, i int) string {
	for j := i + 1; j < len(records); j++ {
		if records[j].Kind == demo.KindExec {
			return owners[j]
		}
	}

	return ""
}

// prevExecOwner returns the owner already computed for the nearest
// demo.KindExec record before i in journal order, or "" if none precedes.
func prevExecOwner(records []demo.Record, owners []string, i int) string {
	for j := i - 1; j >= 0; j-- {
		if records[j].Kind == demo.KindExec {
			return owners[j]
		}
	}

	return ""
}

// currentTest returns the Test of whichever run event is nearest to start,
// among run events at or before end, or "" if none is eligible — start and
// end arrived before any test began (e.g. TestMain's `go build`).
func currentTest(runs []Event, start, end time.Time) string {
	var (
		owner    string
		bestDiff time.Duration
		found    bool
	)

	for _, r := range runs {
		if r.Time.After(end) {
			continue
		}

		diff := r.Time.Sub(start)
		if diff < 0 {
			diff = -diff
		}

		if !found || diff < bestDiff {
			owner, bestDiff, found = r.Test, diff, true
		}
	}

	return owner
}
