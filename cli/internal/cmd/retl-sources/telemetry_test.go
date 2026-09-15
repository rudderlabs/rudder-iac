package retlsource

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rudderlabs/rudder-iac/cli/internal/cmd/telemetry"
	"github.com/rudderlabs/rudder-iac/cli/internal/cmd/telemetry/telemetrytest"
	"github.com/rudderlabs/rudder-iac/cli/internal/testutils"
)

const sqlModelSpec = `version: rudder/0.1
kind: retl-source-sql-model
metadata:
  name: user-orders-model
spec:
  id: user-orders-model
  display_name: User Orders
  account_id: acc-123
  primary_key: id
  source_definition: postgres
  sql: SELECT id, name FROM orders
`

// projectDirs returns locations that make a RETL command fail at each stage
// after dependencies are set up.
func projectDirs(t *testing.T) (missing, empty, withModel string) {
	t.Helper()

	missing = filepath.Join(t.TempDir(), "missing")
	empty = t.TempDir()
	withModel = t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(withModel, "model.yaml"), []byte(sqlModelSpec), 0o644))

	return missing, empty, withModel
}

func executeRecorded(t *testing.T, cmd *cobra.Command, args []string) []telemetrytest.Call {
	t.Helper()

	calls := telemetrytest.Record(t)
	cmd.SetArgs(args)
	cmd.SilenceUsage = true
	cmd.SilenceErrors = true

	require.Error(t, cmd.Execute())
	return *calls
}

func TestValidateTracksRunFailuresAsErrored(t *testing.T) {
	testutils.UseFakeAPI(t)
	missing, empty, withModel := projectDirs(t)

	for name, args := range map[string][]string{
		"missing external id":  {},
		"project load fails":   {"user-orders-model", "--location", missing},
		"model not in project": {"user-orders-model", "--location", empty},
		"query fails remotely": {"user-orders-model", "--location", withModel},
	} {
		t.Run(name, func(t *testing.T) {
			calls := executeRecorded(t, newCmdValidate(), args)
			assert.Equal(t, []telemetrytest.Call{{
				Command: "retl-sources validate",
				Errored: true,
			}}, calls)
		})
	}
}

func TestPreviewTracksRunFailuresAsErrored(t *testing.T) {
	testutils.UseFakeAPI(t)
	missing, empty, withModel := projectDirs(t)

	for name, args := range map[string][]string{
		"missing external id":    {"--interactive=false"},
		"project load fails":     {"user-orders-model", "--interactive=false", "--location", missing},
		"model not in project":   {"user-orders-model", "--interactive=false", "--location", empty},
		"preview fails remotely": {"user-orders-model", "--interactive=false", "--location", withModel},
	} {
		t.Run(name, func(t *testing.T) {
			calls := executeRecorded(t, newCmdPreview(), args)
			assert.Equal(t, []telemetrytest.Call{{
				Command: "retl-sources preview",
				Errored: true,
				Extras: []telemetry.KV{
					{K: "json", V: false},
					{K: "interactive", V: false},
					{K: "limit", V: 10},
				},
			}}, calls)
		})
	}
}
