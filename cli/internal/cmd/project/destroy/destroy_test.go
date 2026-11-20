package destroy

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rudderlabs/rudder-iac/cli/internal/cmd/telemetry"
	"github.com/rudderlabs/rudder-iac/cli/internal/cmd/telemetry/telemetrytest"
	"github.com/rudderlabs/rudder-iac/cli/internal/testutils"
)

func TestDestroyTracksRunFailureAsErrored(t *testing.T) {
	testutils.UseFakeAPI(t)
	calls := telemetrytest.Record(t)

	err := telemetrytest.Execute(NewCmdDestroy(), nil, []string{"--confirm=false"})

	require.Error(t, err)
	assert.Equal(t, []telemetrytest.Call{{
		Command: "destroy",
		Errored: true,
		Extras: []telemetry.KV{
			{K: "dryRun", V: false},
			{K: "confirm", V: false},
		},
	}}, *calls)
}
