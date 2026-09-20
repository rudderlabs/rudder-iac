package main

import (
	"strings"

	"github.com/rudderlabs/rudder-iac/cli/tests/demo"
)

// maxInt64 stands in for "has not finished yet" in a span's stop time.
const maxInt64 = int64(^uint64(0) >> 1)

// Step is one subtest and the records it ran, in order.
type Step struct {
	Test    string
	Records []demo.Record
}

// span is one test's start and finish, in unix nanoseconds.
type span struct {
	test        string
	start, stop int64
}

// Join attributes each journal record to the subtest that was running when it
// started.
//
// The suite is serial, so "which test was running" is a well-defined question:
// walk the events in time order maintaining the set of started-but-not-finished
// tests, and a record belongs to the deepest one active at its start. Records
// that precede the first test — TestMain's `go build`, notably — belong to no
// step and are dropped.
func Join(events []Event, records []demo.Record) ([]Step, error) {
	var (
		open  = map[string]int{} // test name -> index into spans
		spans []span
	)

	for _, e := range events {
		switch e.Action {
		case "run":
			open[e.Test] = len(spans)
			spans = append(spans, span{test: e.Test, start: e.Time.UnixNano(), stop: maxInt64})
		case "pass", "fail", "skip":
			idx, ok := open[e.Test]
			if !ok {
				continue
			}
			spans[idx].stop = e.Time.UnixNano()
			delete(open, e.Test)
		}
	}

	byTest := map[string][]demo.Record{}
	var order []string

	for _, r := range records {
		owner := deepestActive(spans, r.Start.UnixNano())
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

// deepestActive returns the most deeply nested test covering ts, or "" if none
// does. Depth is the number of "/" separators, so a subtest always wins over
// the parent whose span contains it.
func deepestActive(spans []span, ts int64) string {
	var (
		best      string
		bestDepth = -1
	)

	for _, s := range spans {
		if ts < s.start || ts > s.stop {
			continue
		}

		depth := strings.Count(s.test, "/")
		if depth > bestDepth {
			best, bestDepth = s.test, depth
		}
	}

	return best
}
