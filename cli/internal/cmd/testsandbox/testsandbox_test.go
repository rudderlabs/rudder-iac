package testsandbox

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewCmdTestSandbox(t *testing.T) {
	t.Run("prints hello message", func(t *testing.T) {
		cmd := NewCmdTestSandbox()
		var out bytes.Buffer

		cmd.SetOut(&out)
		cmd.SetArgs([]string{})

		require.NoError(t, cmd.Execute())
		assert.Equal(t, "Hello from the sandbox!\n", out.String())
	})

	t.Run("rejects extra arguments", func(t *testing.T) {
		cmd := NewCmdTestSandbox()
		cmd.SetArgs([]string{"extra"})

		err := cmd.Execute()
		require.Error(t, err)
		assert.NotEmpty(t, err.Error())
	})
}
