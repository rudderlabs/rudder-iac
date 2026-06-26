package helloworld

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewCmdHelloWorld(t *testing.T) {
	t.Parallel()

	cmd := NewCmdHelloWorld()

	require.NotNil(t, cmd)
	assert.Equal(t, "hello-world", cmd.Use)
	assert.Equal(t, "Print a hello world message", cmd.Short)
	assert.NotEmpty(t, cmd.Long)
	assert.NotEmpty(t, cmd.Example)
	assert.NotNil(t, cmd.Args)
	assert.NotNil(t, cmd.RunE)
}

func TestCmdHelloWorldPrintsMessage(t *testing.T) {
	t.Parallel()

	cmd := NewCmdHelloWorld()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{})

	require.NoError(t, cmd.Execute())
	assert.Equal(t, "Hello, world!\n", out.String())
}

func TestCmdHelloWorldRejectsUnexpectedArgs(t *testing.T) {
	t.Parallel()

	cmd := NewCmdHelloWorld()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"unexpected"})

	require.Error(t, cmd.Execute())
	assert.NotContains(t, out.String(), "Hello, world!")
}
