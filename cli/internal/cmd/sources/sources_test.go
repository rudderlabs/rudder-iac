package sources

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"

	"github.com/rudderlabs/rudder-iac/cli/internal/cmd/cmderrors"
	"github.com/rudderlabs/rudder-iac/cli/internal/config"
)

// TestMain turns sourcesListen on for the whole package; cli/tests covers the
// flag-off path through the binary.
func TestMain(m *testing.M) {
	dir, err := os.MkdirTemp("", "dev-test-*")
	if err != nil {
		panic(err)
	}
	_ = os.Setenv("RUDDERSTACK_CLI_EXPERIMENTAL", "true")
	_ = os.Setenv("RUDDERSTACK_X_SOURCES_LISTEN", "true")
	_ = os.Setenv("RUDDERSTACK_CLI_TELEMETRY_DISABLED", "true")
	config.InitConfig(filepath.Join(dir, "config.json"))

	code := m.Run()
	_ = os.RemoveAll(dir)
	os.Exit(code)
}

// execute runs args through a root that handles errors as the real root does.
func execute(args ...string) (stdout, stderr string, err error) {
	root := &cobra.Command{Use: "rudder-cli", SilenceErrors: true, SilenceUsage: true}
	root.AddCommand(NewCmdSources())

	var out, errOut bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&errOut)
	root.SetArgs(args)
	err = root.Execute()

	return out.String(), errOut.String(), err
}

func TestNewCmdDevIsHidden(t *testing.T) {
	t.Parallel()

	require.True(t, NewCmdSources().Hidden)
}

func TestDevPrintsHelp(t *testing.T) {
	t.Parallel()

	for _, args := range [][]string{{"sources"}, {"sources", "--help"}} {
		t.Run(strings.Join(args, " "), func(t *testing.T) {
			t.Parallel()

			stdout, stderr, err := execute(args...)

			require.NoError(t, err)
			require.Contains(t, stdout, "Usage:")
			require.Empty(t, stderr)
		})
	}
}

func TestDevUsageErrors(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		args []string
		want string
	}{
		{args: []string{"sources", "summary"}, want: `Error: unknown command "summary" for "rudder-cli sources"`},
		{args: []string{"sources", "exec"}, want: `Error: unknown command "exec" for "rudder-cli sources"`},
		{args: []string{"sources", "stop"}, want: `Error: unknown command "stop" for "rudder-cli sources"`},
		{args: []string{"sources", "cursor"}, want: `Error: unknown command "cursor" for "rudder-cli sources"`},
		{args: []string{"sources", "nope"}, want: `Error: unknown command "nope" for "rudder-cli sources"`},
		{args: []string{"sources", "--bogus"}, want: "Error: unknown flag: --bogus"},
	} {
		t.Run(strings.Join(tc.args, " "), func(t *testing.T) {
			t.Parallel()

			stdout, stderr, err := execute(tc.args...)

			var silent *cmderrors.SilentError
			require.ErrorAs(t, err, &silent)
			require.Empty(t, stdout)
			require.Equal(t, tc.want+"\nNext: rudder-cli sources --help\n", stderr)
		})
	}
}

// Cobra runs only the nearest PersistentPreRunE, so a child with its own
// would skip the experimental gate.
func TestNoChildReplacesTheGate(t *testing.T) {
	t.Parallel()

	var walk func(*cobra.Command)
	walk = func(c *cobra.Command) {
		for _, child := range c.Commands() {
			require.Nil(t, child.PersistentPreRunE, child.CommandPath())
			require.Nil(t, child.PersistentPreRun, child.CommandPath())
			walk(child)
		}
	}
	walk(NewCmdSources())
}
