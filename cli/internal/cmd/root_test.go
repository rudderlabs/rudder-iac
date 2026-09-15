package cmd

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rudderlabs/rudder-iac/cli/internal/cmd/telemetry"
	"github.com/rudderlabs/rudder-iac/cli/internal/cmd/telemetry/telemetrytest"
	"github.com/rudderlabs/rudder-iac/cli/internal/testutils"
)

// Keys are the names each command already reports from RunE, so pre-run and
// run failures of one command land in the same funnel step.
func TestPreRunFailuresAreTrackedUnderRunCommandName(t *testing.T) {
	// Every PreRunE below fails: either on flag validation or on the missing token.
	testutils.SetConfig(t, "auth.accessToken", "")

	for command, path := range map[string][]string{
		"apply":                {"apply"},
		"validate":             {"validate"},
		"destroy":              {"destroy"},
		"migrate":              {"migrate"},
		"import workspace":     {"import", "workspace"},
		"transformations test": {"transformations", "test"},
		"data-graphs validate": {"data-graphs", "validate"},
	} {
		t.Run(command, func(t *testing.T) {
			calls := telemetrytest.Record(t)
			c, _, err := rootCmd.Find(path)
			require.NoError(t, err)

			require.Error(t, c.PreRunE(c, nil))
			assert.Equal(t, []telemetrytest.Call{{
				Command: command,
				Errored: true,
				Extras:  []telemetry.KV{{K: "stage", V: "pre_run"}},
			}}, *calls)
		})
	}
}
