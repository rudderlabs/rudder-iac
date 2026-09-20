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
// go test -json batches every subtest's pass/fail event at the instant the
// parent test itself completes, so sibling subtests' [start, finish) windows
// all appear to end at once and cannot be told apart by depth or by "which
// span is still open" — an earlier version of this function did exactly
// that, and silently merged every sibling's records onto whichever one
// happened to start first (see git history for R18). What batching does not
// disturb is `run`: each test emits exactly one, at the instant it starts,
// and — because ParseEvents already requires the run to be serial — those
// events arrive in the order the tests actually ran. So a record's owner is
// the test whose `run` event most recently precedes the record's start: the
// test that was current when the command began.
//
// This cannot resolve one case. Because pass/fail is batched and carries no
// reliable timing of the work it reports on, a record the *parent* emits
// after its last subtest's `run` — but logically after that subtest's own
// work has finished — is indistinguishable on the wire from a record the
// subtest emitted itself; there is no event marking that boundary. Such a
// record is attributed to the subtest. That is narrower than the bug this
// replaces, which could misattribute a record across unrelated siblings;
// here the only remaining ambiguity is between a subtest and its own parent.
func Join(events []Event, records []demo.Record) ([]Step, error) {
	var runs []Event
	for _, e := range events {
		if e.Action == "run" {
			runs = append(runs, e)
		}
	}

	byTest := map[string][]demo.Record{}
	var order []string

	for _, r := range records {
		owner := currentTest(runs, r.Start)
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

// currentTest returns the Test of whichever run event most recently precedes
// ts, or "" if none does — ts arrived before the first test started (e.g.
// TestMain's `go build`). runs must be in chronological order, which they
// are: they're a filtered subsequence of events, and this package never
// reorders events (see Join's own "no sorting anywhere" precedent).
func currentTest(runs []Event, ts time.Time) string {
	var owner string
	for _, r := range runs {
		if r.Time.After(ts) {
			break
		}
		owner = r.Test
	}
	return owner
}
