package workspace

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewCmdEventStreamSources(t *testing.T) {
	t.Parallel()

	cmd := NewCmdEventStreamSources()
	require.NotNil(t, cmd)

	assert.Equal(t, "event-stream-sources", cmd.Use)
	assert.Equal(t, "Manage event stream sources in the workspace", cmd.Short)
	assert.Len(t, cmd.Commands(), 1)

	listCmd := cmd.Commands()[0]
	assert.Equal(t, "list", listCmd.Use)
	assert.Equal(t, "List event stream sources in the workspace", listCmd.Short)
	assert.NotNil(t, listCmd.RunE)

	jsonFlag := listCmd.Flags().Lookup("json")
	require.NotNil(t, jsonFlag)
	assert.Equal(t, "false", jsonFlag.DefValue)
}

func TestNewCmdWorkspace_RegistersEventStreamSources(t *testing.T) {
	t.Parallel()

	cmd := NewCmdWorkspace()
	require.NotNil(t, cmd)

	found := false
	for _, subCmd := range cmd.Commands() {
		if subCmd.Name() == "event-stream-sources" {
			found = true
			break
		}
	}

	assert.True(t, found, "event-stream-sources command should be registered")
}
