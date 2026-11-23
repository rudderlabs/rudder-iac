package telemetry

import (
	"errors"
	"testing"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
)

func TestTrackInvalidInput(t *testing.T) {
	root := &cobra.Command{Use: "rudder-cli"}
	root.AddCommand(&cobra.Command{Use: "apply"})
	root.AddCommand(&cobra.Command{Use: "typer"})
	failure := errors.New("boom")
	tracked := []string{"apply"}

	run := func(t *testing.T, ready, alreadyReported bool, args []string, err error) []string {
		t.Helper()

		var calls []string
		origTrack, origReady := TrackCommand, telemetryReady
		TrackCommand = func(command string, _ error, extras ...KV) {
			calls = append(calls, command)
			assert.Equal(t, []KV{{K: "stage", V: InvalidInputStage}}, extras)
		}
		telemetryReady = func() bool { return ready }
		reported.Store(alreadyReported)
		t.Cleanup(func() {
			TrackCommand, telemetryReady = origTrack, origReady
			reported.Store(false)
		})

		TrackInvalidInput(root, args, err, tracked)
		return calls
	}

	tests := []struct {
		name            string
		ready, reported bool
		args            []string
		err             error
		want            []string
	}{
		{name: "tracks a command that reports its own result", ready: true, args: []string{"apply"}, err: failure, want: []string{"apply"}},
		{name: "skips a command that does not report its own result", ready: true, args: []string{"typer"}, err: failure},
		{name: "skips when a hook already reported", ready: true, reported: true, args: []string{"apply"}, err: failure},
		{name: "skips a nil error", ready: true, args: []string{"apply"}},
		{name: "skips when telemetry is not ready", args: []string{"apply"}, err: failure},
		{name: "skips an unknown command", ready: true, args: []string{"nope"}, err: failure},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, run(t, tc.ready, tc.reported, tc.args, tc.err))
		})
	}
}

// The real trackCommand is what sets reported. Tests that swap TrackCommand
// would pass without that line, and a failure would then count twice.
func TestTrackCommandMarksTheRunAsReported(t *testing.T) {
	reported.Store(false)
	viper.Set("telemetry.disabled", true)
	t.Cleanup(func() {
		reported.Store(false)
		viper.Set("telemetry.disabled", nil)
	})

	// Disabled telemetry sends nothing but still goes through the real path.
	trackCommand("apply", nil)

	assert.True(t, reported.Load())
}
