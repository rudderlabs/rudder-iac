package workspace

import (
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func findSubcommand(parent *cobra.Command, name string) *cobra.Command {
	for _, c := range parent.Commands() {
		if c.Name() == name {
			return c
		}
	}
	return nil
}

func TestEventStreamSourcesList_Deprecated(t *testing.T) {
	t.Parallel()

	cmd := findSubcommand(NewCmdEventStreamSources(), "list")
	require.NotNil(t, cmd)
	assert.NotEmpty(t, cmd.Deprecated)
	assert.Contains(t, cmd.Deprecated, "get event-stream-source")
}

// TestAccountsList_NotDeprecated asserts that accounts list is deliberately NOT
// deprecated: "get account" is not yet supported (the workspace provider is not
// registered in the composite), so pointing users there would break their workflow.
// Remove this test (and add a deprecation) once get account is fully wired up.
func TestAccountsList_NotDeprecated(t *testing.T) {
	t.Parallel()

	cmd := findSubcommand(NewCmdAccounts(), "list")
	require.NotNil(t, cmd)
	assert.Empty(t, cmd.Deprecated)
}

func TestRetlSourcesList_Deprecated(t *testing.T) {
	t.Parallel()

	cmd := findSubcommand(NewCmdRetlSource(), "list")
	require.NotNil(t, cmd)
	assert.NotEmpty(t, cmd.Deprecated)
	assert.Contains(t, cmd.Deprecated, "get retl-source-sql-model")
}
