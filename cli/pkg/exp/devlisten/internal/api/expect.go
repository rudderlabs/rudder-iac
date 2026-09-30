package api

import (
	"strconv"
	"strings"
)

// Statuses of an expected event.
const (
	expectPresent       = "present"
	expectMissing       = "missing"
	expectCountMismatch = "count_mismatch"
)

// expectation is one expect value: NAME, or NAME=COUNT for an exact count.
// COUNT=0 asserts absence.
type expectation struct {
	Event  string `json:"event"`
	Want   *int   `json:"want"`
	Got    int    `json:"got"`
	Status string `json:"status"`
}

func parseExpect(p *params) []expectation {
	var out []expectation
	for _, raw := range p.list("expect") {
		e := expectation{Event: raw}
		if i := strings.LastIndex(raw, "="); i > 0 {
			if n, err := strconv.Atoi(raw[i+1:]); err == nil && n >= 0 {
				e.Event, e.Want = raw[:i], &n
			}
		}
		if e.Event == "" {
			p.fail("expect", "%q is not NAME or NAME=COUNT", raw)
		}
		out = append(out, e)
	}
	return out
}

// check fills got and status from the event counts.
func checkExpected(expected []expectation, byEvent map[string]int) []expectation {
	for i := range expected {
		e := &expected[i]
		e.Got = byEvent[e.Event]
		switch {
		case e.Want == nil && e.Got > 0, e.Want != nil && e.Got == *e.Want:
			e.Status = expectPresent
		case e.Got == 0:
			e.Status = expectMissing
		default:
			e.Status = expectCountMismatch
		}
	}
	return expected
}

func withStatus(expected []expectation, status string) []string {
	var names []string
	for _, e := range expected {
		if e.Status == status {
			names = append(names, e.Event)
		}
	}
	return names
}

// expectations adds one diagnosis for missing events and one for counts
// that differ. The names are caller values; they go in events, never next.
func (d *diagnosis) expectations(expected []expectation, since uint64) {
	next := newCommand(routeCommands["events"]).num("since", since).bare("view", viewSummary).flag("json").String()
	missing := withStatus(expected, expectMissing)
	d.addEvents(len(missing) > 0, "expected_missing", missing,
		"Expected events did not arrive after the cursor: see events. all_accepted covers received requests only.",
		next)
	mismatched := withStatus(expected, expectCountMismatch)
	d.addEvents(len(mismatched) > 0, "expected_count_mismatch", mismatched,
		"Expected events arrived a different number of times: compare want and got in expected.", next)
}
