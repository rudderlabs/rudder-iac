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

	err := telemetrytest.Execute(NewCmdApply(), nil, []string{"--location", location, "--confirm=false"})

	require.Error(t, err)
	assert.Equal(t, []telemetrytest.Call{{
		Command: "apply",
		Errored: true,
		Extras: []telemetry.KV{
			{K: "location", V: location},
			{K: "dryRun", V: false},
			{K: "confirm", V: false},
			{K: "noDelete", V: false},
			{K: "providers", V: ""},
		},
	}}, *calls)
}

// An unknown --provider fails inside RunE before the syncer runs. Telemetry
// must still record the run as errored.
func TestApplyTracksUnknownProviderAsErrored(t *testing.T) {
	testutils.UseFakeAPI(t)
	testutils.SetConfig(t, "experimental", true)
	testutils.SetConfig(t, "flags.applyOptions", true)
	calls := telemetrytest.Record(t)
	location := t.TempDir()

	err := telemetrytest.Execute(NewCmdApply(), nil, []string{"--location", location, "--confirm=false", "--provider", "nope"})

	require.ErrorContains(t, err, "unknown provider: nope")
	require.Len(t, *calls, 1)
	assert.True(t, (*calls)[0].Errored)
	assert.Contains(t, (*calls)[0].Extras, telemetry.KV{K: "providers", V: "nope"})
}
