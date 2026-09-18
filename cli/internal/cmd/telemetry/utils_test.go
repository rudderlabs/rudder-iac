package telemetry_test

import (
	"errors"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rudderlabs/rudder-iac/cli/internal/cmd/telemetry"
	"github.com/rudderlabs/rudder-iac/cli/internal/cmd/telemetry/telemetrytest"
)

// newTree builds root -> import -> workspace, where workspace tracks itself
// from RunE the way real commands do.
func newTree(preRunErr error) (*cobra.Command, *bool) {
	ran := false
	leaf := &cobra.Command{
		Use: "workspace",
		PreRunE: func(*cobra.Command, []string) error {
			return preRunErr
		},
		RunE: func(*cobra.Command, []string) error {
			ran = true
			err := errors.New("run failed")
			telemetry.TrackCommand("import workspace", err)
			return err
		},
	}

	group := &cobra.Command{Use: "import <command>"}
	group.AddCommand(leaf)

	root := &cobra.Command{Use: "rudder-cli", SilenceUsage: true, SilenceErrors: true}
	root.AddCommand(group)

	return root, &ran
}

func TestTrackPreRunFailuresTracksFailedPreRun(t *testing.T) {
	calls := telemetrytest.Record(t)
	root, ranRunE := newTree(errors.New("access token is required"))
	telemetry.TrackPreRunFailures(root)

	root.SetArgs([]string{"import", "workspace"})
	require.Error(t, root.Execute())

	assert.False(t, *ranRunE)
	assert.Equal(t, []telemetrytest.Call{{
		Command: "import workspace",
		Errored: true,
		Extras:  []telemetry.KV{{K: "stage", V: "pre_run"}},
	}}, *calls)
}

func TestTrackPreRunFailuresLeavesRunTrackingAlone(t *testing.T) {
	calls := telemetrytest.Record(t)
	root, ranRunE := newTree(nil)
	telemetry.TrackPreRunFailures(root)

	root.SetArgs([]string{"import", "workspace"})
	require.Error(t, root.Execute())

	assert.True(t, *ranRunE)
	assert.Equal(t, []telemetrytest.Call{{
		Command: "import workspace",
		Errored: true,
	}}, *calls)
}

func TestTrackPreRunFailuresSkipsCommandsWithoutPreRun(t *testing.T) {
	root, _ := newTree(nil)
	group := root.Commands()[0]

	telemetry.TrackPreRunFailures(root)

	assert.Nil(t, root.PreRunE)
	assert.Nil(t, group.PreRunE)
}
