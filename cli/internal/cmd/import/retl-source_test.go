package importcmd

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rudderlabs/rudder-iac/cli/internal/cmd/telemetry"
	"github.com/rudderlabs/rudder-iac/cli/internal/cmd/telemetry/telemetrytest"
	"github.com/rudderlabs/rudder-iac/cli/internal/testutils"
)

func TestRetlSourceImportTracksRunFailureAsErrored(t *testing.T) {
	// No access token, so dependency setup inside RunE fails.
	testutils.SetConfig(t, "auth.accessToken", "")
	calls := telemetrytest.Record(t)

	cmd := NewCmdRetlSource()
	cmd.SetArgs([]string{"--local-id", "my-model", "--remote-id", "remote-1", "--location", "models"})
	cmd.SilenceUsage = true
	cmd.SilenceErrors = true

	require.Error(t, cmd.Execute())
	assert.Equal(t, []telemetrytest.Call{{
		Command: "import retl-sources",
		Errored: true,
		Extras: []telemetry.KV{
			{K: "localID", V: "my-model"},
			{K: "remoteID", V: "remote-1"},
			{K: "location", V: "models"},
			{K: "sqlLocation", V: ""},
		},
	}}, *calls)
}
