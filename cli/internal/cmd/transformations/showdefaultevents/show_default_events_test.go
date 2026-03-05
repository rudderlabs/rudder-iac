package showdefaultevents

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewCmdShowDefaultEvents(t *testing.T) {
	t.Parallel()

	cmd := NewCmdShowDefaultEvents()

	require.NotNil(t, cmd)
	assert.Equal(t, "show-default-events", cmd.Use)
	assert.Equal(t, "Show default test events", cmd.Short)
	assert.NotEmpty(t, cmd.Long)
	assert.NotEmpty(t, cmd.Example)
	assert.NotNil(t, cmd.RunE)
}

func TestCmdShowDefaultEvents_NoFlags(t *testing.T) {
	t.Parallel()

	cmd := NewCmdShowDefaultEvents()

	// Verify no flags are defined
	assert.Empty(t, cmd.Flags().Args())
}

func TestCmdShowDefaultEvents_NoArgs(t *testing.T) {
	t.Parallel()

	cmd := NewCmdShowDefaultEvents()

	// Command should not require any arguments
	assert.False(t, cmd.Args != nil)
}
