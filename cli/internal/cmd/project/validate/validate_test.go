package validate

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rudderlabs/rudder-iac/cli/internal/cmd/telemetry"
	"github.com/rudderlabs/rudder-iac/cli/internal/cmd/telemetry/telemetrytest"
	"github.com/rudderlabs/rudder-iac/cli/internal/testutils"
)

func TestValidateTracksRunFailureAsErrored(t *testing.T) {
	testutils.UseFakeAPI(t)
	calls := telemetrytest.Record(t)
	location := filepath.Join(t.TempDir(), "missing")

	cmd := NewCmdValidate()
	cmd.SetArgs([]string{"--location", location})
	cmd.SilenceUsage = true
	cmd.SilenceErrors = true

	require.Error(t, cmd.Execute())
	assert.Equal(t, []telemetrytest.Call{{
		Command: "validate",
		Errored: true,
		Extras:  []telemetry.KV{{K: "location", V: location}},
	}}, *calls)
}
