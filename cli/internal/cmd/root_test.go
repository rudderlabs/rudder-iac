package cmd

import (
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rudderlabs/rudder-iac/cli/internal/cmd/telemetry"
	"github.com/rudderlabs/rudder-iac/cli/internal/cmd/telemetry/telemetrytest"
	"github.com/rudderlabs/rudder-iac/cli/internal/testutils"
)

// A pre-run failure has to land in the same funnel step as a run failure of the
// same command, so the name a command reports from RunE must equal the path the
// PreRunE wrapper derives.
//
// The expected value is derived from the resolved command rather than taken from
// the case, so changing the derivation fails here instead of passing against a
// label that was written to match it.
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
				Command: commandPath(c),
				Errored: true,
				Extras:  []telemetry.KV{{K: "stage", V: "pre_run"}},
			}}, *calls)
		})
	}
}

// TestTrackedNamesMatchCommandPaths is the other half of the invariant: every
// name passed to TrackCommand from a RunE must be the command's own path.
//
// The literals cannot be read back out of the command tree, so they are listed
// here against the path each is expected to resolve to — which is what makes a
// rename of either side fail. Three were wrong when this check was written:
// "import retl-sources" against `import retl-sources`, "workspace retl-source
// list" against `workspace retl-sources list`, and "typer" against
// `typer generate`.
func TestTrackedNamesMatchCommandPaths(t *testing.T) {
	cases := []struct {
		literal string
		path    []string
	}{
		{"apply", []string{"apply"}},
		{"validate", []string{"validate"}},
		{"destroy", []string{"destroy"}},
		{"migrate", []string{"migrate"}},
		{"typer generate", []string{"typer", "generate"}},
		{"auth login", []string{"auth", "login"}},
		{"import workspace", []string{"import", "workspace"}},
		{"import retl-sources", []string{"import", "retl-sources"}},
		{"retl-sources validate", []string{"retl-sources", "validate"}},
		{"retl-sources preview", []string{"retl-sources", "preview"}},
		{"workspace info", []string{"workspace", "info"}},
		{"workspace accounts list", []string{"workspace", "accounts", "list"}},
		{"workspace retl-sources list", []string{"workspace", "retl-sources", "list"}},
		{"workspace tracking-plans list", []string{"workspace", "tracking-plans", "list"}},
		{"workspace event-stream-sources list", []string{"workspace", "event-stream-sources", "list"}},
		{"transformations test", []string{"transformations", "test"}},
		{"transformations show-default-events", []string{"transformations", "show-default-events"}},
		{"data-graphs validate", []string{"data-graphs", "validate"}},
	}

	for _, tc := range cases {
		t.Run(tc.literal, func(t *testing.T) {
			c, _, err := rootCmd.Find(tc.path)
			require.NoError(t, err)
			assert.Equal(t, tc.literal, commandPath(c),
				"this name is passed to TrackCommand but is not the command's path; the same command would report under two funnel names")
		})
	}
}

// commandPath is the name the PreRunE wrapper derives, kept here so the tests
// fail if that derivation changes rather than restating it.
func commandPath(c *cobra.Command) string {
	return strings.TrimPrefix(c.CommandPath(), c.Root().Name()+" ")
}
