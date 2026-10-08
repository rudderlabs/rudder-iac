package telemetry

import (
	"errors"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
)

func TestTrackUnreportedFailure(t *testing.T) {
	root := &cobra.Command{Use: "rudder-cli"}
	root.AddCommand(&cobra.Command{Use: "apply"})
	failure := errors.New("boom")

	run := func(t *testing.T, ready, alreadyReported bool, args []string, err error) []string {
		t.Helper()

		var tracked []string
		origTrack, origReady := TrackCommand, telemetryReady
		TrackCommand = func(command string, _ error, extras ...KV) {
			tracked = append(tracked, command)
			assert.Equal(t, []KV{{K: "stage", V: "unreported"}}, extras)
		}
		telemetryReady = func() bool { return ready }
		reported.Store(alreadyReported)
		t.Cleanup(func() {
			TrackCommand, telemetryReady = origTrack, origReady
			reported.Store(false)
		})

		TrackUnreportedFailure(root, args, err)
		return tracked
	}

	tests := []struct {
		name            string
		ready, reported bool
		args            []string
		err             error
		want            []string
	}{
		{name: "tracks a resolved command", ready: true, args: []string{"apply"}, err: failure, want: []string{"apply"}},
		{name: "skips when a hook already reported", ready: true, reported: true, args: []string{"apply"}, err: failure},
		{name: "skips a nil error", ready: true, args: []string{"apply"}},
		{name: "skips when telemetry is not ready", args: []string{"apply"}, err: failure},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, run(t, tc.ready, tc.reported, tc.args, tc.err))
		})
	}
}
