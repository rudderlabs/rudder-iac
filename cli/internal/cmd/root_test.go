package cmd

import (
	"bytes"
	"os"
	"reflect"
	"runtime"
	"strings"
	"testing"

	"github.com/spf13/cobra"
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

// trackedCommands decides which commands report input that cobra rejects. A
// command that tracks itself but is missing from the list would lose that
// report; a listed command that does not track itself would send failures with
// no successes to compare them with.
func TestTrackedCommandsMatchRunEReporters(t *testing.T) {
	var reporters []string

	var walk func(c *cobra.Command)
	walk = func(c *cobra.Command) {
		for _, sub := range c.Commands() {
			walk(sub)
		}
		if c.RunE == nil {
			return
		}

		fn := runtime.FuncForPC(reflect.ValueOf(c.RunE).Pointer())
		file, _ := fn.FileLine(fn.Entry())
		source, err := os.ReadFile(file)
		require.NoError(t, err)
		if bytes.Contains(source, []byte("TrackCommand(")) {
			reporters = append(reporters, telemetry.CommandName(c))
		}
	}
	walk(rootCmd)

	assert.ElementsMatch(t, reporters, trackedCommands)
}
