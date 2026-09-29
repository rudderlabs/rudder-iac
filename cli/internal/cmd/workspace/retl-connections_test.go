package workspace

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewCmdRetlConnections(t *testing.T) {
	t.Parallel()

	cmd := NewCmdRetlConnections()
	require.NotNil(t, cmd)

	assert.Equal(t, "retl-connections", cmd.Use)
	assert.Equal(t, "Manage RETL connections in the workspace", cmd.Short)
	require.Len(t, cmd.Commands(), 1)

	listCmd := cmd.Commands()[0]
	assert.Equal(t, "list", listCmd.Use)
	assert.NotNil(t, listCmd.RunE)

	jsonFlag := listCmd.Flags().Lookup("json")
	require.NotNil(t, jsonFlag)
	assert.Equal(t, "false", jsonFlag.DefValue)
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
