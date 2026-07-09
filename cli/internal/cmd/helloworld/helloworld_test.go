package helloworld

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewCmdHelloWorldPrintsGreeting(t *testing.T) {
	t.Parallel()

	cmd := NewCmdHelloWorld()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetArgs([]string{})

	require.NoError(t, cmd.Execute())

	assert.Equal(t, "Hello, World!\n", out.String())
}

func TestNewCmdHelloWorldRejectsArgs(t *testing.T) {
	t.Parallel()

	cmd := NewCmdHelloWorld()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"unexpected"})

	require.Error(t, cmd.Execute())
	assert.NotContains(t, out.String(), "Hello, World!")
}
