package datagraph

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewCmdDataGraphVisibleByDefault(t *testing.T) {
	t.Parallel()

	cmd := NewCmdDataGraph()

	assert.False(t, cmd.Hidden)
}
