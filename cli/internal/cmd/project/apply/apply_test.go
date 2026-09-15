package apply

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rudderlabs/rudder-iac/cli/internal/cmd/telemetry"
	"github.com/rudderlabs/rudder-iac/cli/internal/cmd/telemetry/telemetrytest"
	"github.com/rudderlabs/rudder-iac/cli/internal/testutils"
)

func TestApplyTracksRunFailureAsErrored(t *testing.T) {
	testutils.UseFakeAPI(t)
	calls := telemetrytest.Record(t)
	location := t.TempDir()

	cmd := NewCmdApply()
	cmd.SetArgs([]string{"--location", location, "--confirm=false"})
	cmd.SilenceUsage = true
	cmd.SilenceErrors = true

	require.Error(t, cmd.Execute())
	assert.Equal(t, []telemetrytest.Call{{
		Command: "apply",
		Errored: true,
		Extras: []telemetry.KV{
			{K: "location", V: location},
			{K: "dryRun", V: false},
			{K: "confirm", V: false},
		},
	}}, *calls)
}
