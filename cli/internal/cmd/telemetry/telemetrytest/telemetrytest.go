// Package telemetrytest records CLI command telemetry in tests instead of
// sending it.
package telemetrytest

import (
	"testing"

	"github.com/rudderlabs/rudder-iac/cli/internal/cmd/telemetry"
)

// Call is what a command reported through telemetry.TrackCommand. Only the
// errored flag is kept from the error because that is all the event carries.
type Call struct {
	Command string
	Errored bool
	Extras  []telemetry.KV
}

// Record replaces telemetry.TrackCommand for the duration of the test. Tests
// using it must not run in parallel, since TrackCommand is package state.
func Record(t *testing.T) *[]Call {
	t.Helper()

	var (
		calls    []Call
		original = telemetry.TrackCommand
	)

	telemetry.TrackCommand = func(command string, err error, extras ...telemetry.KV) {
		calls = append(calls, Call{Command: command, Errored: err != nil, Extras: extras})
	}
	t.Cleanup(func() { telemetry.TrackCommand = original })

	return &calls
}
