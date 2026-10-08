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
func newTree() (*cobra.Command, *bool) {
	ran := false
	leaf := &cobra.Command{
		Use: "workspace",
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

func TestTrackPreRunFailures(t *testing.T) {
	tests := []struct {
		name       string
		preRunErr  error
		persistent bool
		wantRunE   bool
		want       []telemetrytest.Call
	}{
		{
			name:      "tracks a failed PreRunE",
			preRunErr: errors.New("access token is required"),
			want:      []telemetrytest.Call{{Command: "import workspace", Errored: true, Extras: []telemetry.KV{{K: "stage", V: "pre_run"}}}},
		},
		{
			name:       "tracks a failed PersistentPreRunE under the running command",
			preRunErr:  errors.New("experimental flag is off"),
			persistent: true,
			want:       []telemetrytest.Call{{Command: "import workspace", Errored: true, Extras: []telemetry.KV{{K: "stage", V: "pre_run"}}}},
		},
		{
			name:     "leaves run tracking alone",
			wantRunE: true,
			want:     []telemetrytest.Call{{Command: "import workspace", Errored: true}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			calls := telemetrytest.Record(t)
			root, ranRunE := newTree()
			group := root.Commands()[0]
			if tt.persistent {
				group.PersistentPreRunE = func(*cobra.Command, []string) error { return tt.preRunErr }
			} else {
				group.Commands()[0].PreRunE = func(*cobra.Command, []string) error { return tt.preRunErr }
			}
			telemetry.TrackPreRunFailures(root)

			root.SetArgs([]string{"import", "workspace"})
			require.Error(t, root.Execute())

			assert.Equal(t, tt.wantRunE, *ranRunE)
			assert.Equal(t, tt.want, *calls)
		})
	}
}

func TestTrackPreRunFailuresSkipsCommandsWithoutPreRun(t *testing.T) {
	root, _ := newTree()
	group := root.Commands()[0]

	telemetry.TrackPreRunFailures(root)

	assert.Nil(t, root.PreRunE)
	assert.Nil(t, group.PreRunE)
	assert.Nil(t, group.PersistentPreRunE)
}
