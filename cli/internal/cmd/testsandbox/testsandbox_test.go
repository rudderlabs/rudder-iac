package testsandbox

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewCmdTestSandbox(t *testing.T) {
	t.Run("prints expected message", func(t *testing.T) {
		cmd := NewCmdTestSandbox()
		var buf bytes.Buffer
		cmd.SetOut(&buf)

		require.NoError(t, cmd.Execute())
		assert.Equal(t, "Hello from the sandbox using the new image!\n", buf.String())
	})

	t.Run("rejects unexpected args", func(t *testing.T) {
		cmd := NewCmdTestSandbox()
		err := cmd.ValidateArgs([]string{"extra"})
		require.Error(t, err)
		assert.ErrorContains(t, err, "unknown command \"extra\" for \"test-sandbox\"")
	})
}
