package workspace

import (
	"testing"

	"github.com/spf13/cobra"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewCmdRetlConnections(t *testing.T) {
	t.Parallel()

	cmd := NewCmdRetlConnections()
	require.NotNil(t, cmd)

	assert.Equal(t, "retl-connections", cmd.Use)
	assert.Equal(t, "Manage RETL connections in the workspace", cmd.Short)
	subs := map[string]*cobra.Command{}
	for _, c := range cmd.Commands() {
		subs[c.Name()] = c
	}
	require.Contains(t, subs, "list")
	require.Contains(t, subs, "view")

	for name, sub := range subs {
		assert.NotNil(t, sub.RunE, name)
		jsonFlag := sub.Flags().Lookup("json")
		require.NotNil(t, jsonFlag, name+" needs --json")
		assert.Equal(t, "false", jsonFlag.DefValue)
	}
	assert.Equal(t, "view <external-id>", subs["view"].Use)
}

func TestNewCmdWorkspace_RegistersRetlConnections(t *testing.T) {
	t.Parallel()

	names := make([]string, 0)
	for _, subCmd := range NewCmdWorkspace().Commands() {
		names = append(names, subCmd.Name())
	}

	assert.Contains(t, names, "retl-connections")
	assert.Contains(t, names, "retl-sources")
}
