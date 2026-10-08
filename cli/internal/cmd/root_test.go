package cmd

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rudderlabs/rudder-iac/cli/internal/cmd/telemetry"
	"github.com/rudderlabs/rudder-iac/cli/internal/cmd/telemetry/telemetrytest"
	"github.com/rudderlabs/rudder-iac/cli/internal/testutils"
)

// A pre-run failure is reported under the command's path with stage "pre_run".
func TestPreRunFailuresAreTrackedUnderCommandPath(t *testing.T) {
	// Every PreRunE below fails: either on flag validation or on the missing token.
	testutils.SetConfig(t, "auth.accessToken", "")

	for _, path := range [][]string{
		{"apply"},
		{"validate"},
		{"destroy"},
		{"migrate"},
		{"import", "workspace"},
		{"transformations", "test"},
		{"data-graphs", "validate"},
	} {
		t.Run(strings.Join(path, " "), func(t *testing.T) {
			calls := telemetrytest.Record(t)
			c, _, err := rootCmd.Find(path)
			require.NoError(t, err)
			require.NotNil(t, c.PreRunE, "command has no PreRunE to fail")

			require.Error(t, c.PreRunE(c, nil))
			assert.Equal(t, []telemetrytest.Call{{
				Command: telemetry.CommandName(c),
				Errored: true,
				Extras:  []telemetry.KV{{K: "stage", V: "pre_run"}},
			}}, *calls)
		})
	}
}
