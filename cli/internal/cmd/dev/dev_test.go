package dev

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

// TestMain turns devListen on for the whole package; cli/tests covers the
// flag-off path through the binary.
func TestMain(m *testing.M) {
	dir, err := os.MkdirTemp("", "dev-test-*")
	if err != nil {
		panic(err)
	}
	_ = os.Setenv("RUDDERSTACK_CLI_EXPERIMENTAL", "true")
	_ = os.Setenv("RUDDERSTACK_X_DEV_LISTEN", "true")
	config.InitConfig(filepath.Join(dir, "config.json"))

	code := m.Run()
	_ = os.RemoveAll(dir)
	os.Exit(code)
}

// execute runs args through a root that handles errors as the real root does.
func execute(args ...string) (stdout, stderr string, err error) {
	root := &cobra.Command{Use: "rudder-cli", SilenceErrors: true, SilenceUsage: true}
	root.AddCommand(NewCmdDev())

	var out, errOut bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&errOut)
	root.SetArgs(args)
	err = root.Execute()

	return out.String(), errOut.String(), err
}

func TestNewCmdDevIsHidden(t *testing.T) {
	t.Parallel()

	require.True(t, NewCmdDev().Hidden)
}

func TestDevPrintsHelp(t *testing.T) {
	t.Parallel()

	for _, args := range [][]string{{"dev"}, {"dev", "--help"}} {
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
		{args: []string{"dev", "summary"}, want: `Error: unknown command "summary" for "rudder-cli dev"`},
		{args: []string{"dev", "exec"}, want: `Error: unknown command "exec" for "rudder-cli dev"`},
		{args: []string{"dev", "stop"}, want: `Error: unknown command "stop" for "rudder-cli dev"`},
		{args: []string{"dev", "cursor"}, want: `Error: unknown command "cursor" for "rudder-cli dev"`},
		{args: []string{"dev", "nope"}, want: `Error: unknown command "nope" for "rudder-cli dev"`},
		{args: []string{"dev", "--bogus"}, want: "Error: unknown flag: --bogus"},
	} {
		t.Run(strings.Join(tc.args, " "), func(t *testing.T) {
			t.Parallel()

			stdout, stderr, err := execute(tc.args...)

			var silent *cmderrors.SilentError
			require.ErrorAs(t, err, &silent)
			require.Empty(t, stdout)
			require.Equal(t, tc.want+"\nNext: rudder-cli dev --help\n", stderr)
		})
	}
}
