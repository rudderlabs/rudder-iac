package graph

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewCmdGraph_Defaults(t *testing.T) {
	t.Parallel()

	cmd := NewCmdGraph()
	assert.Equal(t, "graph [path]", cmd.Use)

	format, err := cmd.Flags().GetString("format")
	assert.NoError(t, err)
	assert.Equal(t, "dot", format)

	location, err := cmd.Flags().GetString("location")
	assert.NoError(t, err)
	assert.Equal(t, ".", location)

	typeFilter, err := cmd.Flags().GetString("type")
	assert.NoError(t, err)
	assert.Empty(t, typeFilter)
}

func TestNewCmdGraph_AcceptsAtMostOnePositionalArg(t *testing.T) {
	t.Parallel()

	cmd := NewCmdGraph()
	assert.NoError(t, cmd.Args(cmd, []string{"./project"}))
	assert.Error(t, cmd.Args(cmd, []string{"a", "b"}))
}
