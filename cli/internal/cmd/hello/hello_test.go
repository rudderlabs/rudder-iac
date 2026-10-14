package hello

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHelloPrintsGreeting(t *testing.T) {
	cmd := NewCmdHello()
	out := &bytes.Buffer{}
	cmd.SetOut(out)

	err := cmd.Execute()

	require.NoError(t, err)
	assert.Equal(t, "Hello, World!\n", out.String())
}

func TestHelloMetadata(t *testing.T) {
	cmd := NewCmdHello()

	assert.Equal(t, "hello", cmd.Use)
	assert.Equal(t, "Print a hello world greeting", cmd.Short)
	assert.False(t, cmd.HasAvailableLocalFlags())
}

func TestHelloRejectsArguments(t *testing.T) {
	cmd := NewCmdHello()
	cmd.SetArgs([]string{"unexpected"})

	err := cmd.Execute()

	require.Error(t, err)
	assert.ErrorContains(t, err, "unknown command")
}
