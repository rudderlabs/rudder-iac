package typer

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rudderlabs/rudder-iac/cli/internal/cmd/telemetry"
	"github.com/rudderlabs/rudder-iac/cli/internal/cmd/telemetry/telemetrytest"
)

func TestGenerateTracksRunFailuresAsErrored(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "missing")

	for name, tc := range map[string]struct {
		args     []string
		platform string
		local    bool
	}{
		"unsupported platform": {
			args:     []string{"--platform", "cobol"},
			platform: "cobol",
		},
		"local plan fails to load": {
			args:     []string{"--platform", platformKotlin, "--local", "--location", missing, "--tracking-plan-id", "tp"},
			platform: platformKotlin,
			local:    true,
		},
	} {
		t.Run(name, func(t *testing.T) {
			calls := telemetrytest.Record(t)

			cmd := newCmdGenerate()
			cmd.SetArgs(tc.args)
			cmd.SilenceUsage = true
			cmd.SilenceErrors = true

			require.Error(t, cmd.Execute())
			assert.Equal(t, []telemetrytest.Call{{
				Command: "typer",
				Errored: true,
				Extras: []telemetry.KV{
					{K: "platform", V: tc.platform},
					{K: "local", V: tc.local},
				},
			}}, *calls)
		})
	}
}
