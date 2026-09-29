package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"time"
)

// e2ePackage is the only package that drives the CLI binary, and so the only
// one whose events can own a journal record.
const e2ePackage = "github.com/rudderlabs/rudder-iac/cli/tests"

// ErrParallelSubtest reports a parallel subtest in the package that drives the
// CLI. Journal records are attributed to subtests by time, which is exact only
// while that package runs serially; a parallel subtest would silently scramble
// a demo rather than fail it, so it is refused instead.
var ErrParallelSubtest = errors.New("parallel subtest in the demo package")

// Event is one `go test -json` record, narrowed to the fields attribution needs.
type Event struct {
	Time    time.Time `json:"Time"`
	Action  string    `json:"Action"`
	Package string    `json:"Package"`
	Test    string    `json:"Test"`
}

// ParseEvents reads a `go test -json` stream and returns the test-scoped events
// belonging to pkg, in file order.
//
// Events from other packages are dropped rather than refused: cli/tests/helpers
// is parallel throughout, but holds unit tests that never touch the CLI and so
// contribute no journal records.
func ParseEvents(r io.Reader, pkg string) ([]Event, error) {
	var (
		dec = json.NewDecoder(r)
		out []Event
	)

	for dec.More() {
		var e Event
		if err := dec.Decode(&e); err != nil {
			return nil, fmt.Errorf("decoding go test event: %w", err)
		}

		if e.Package != pkg || e.Test == "" {
			continue
		}

		if e.Action == "pause" || e.Action == "cont" {
			return nil, fmt.Errorf("%w: %s", ErrParallelSubtest, e.Test)
		}

		out = append(out, e)
	}

	return out, nil
}
