package hello

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewCmdHelloWritesGreeting(t *testing.T) {
	t.Parallel()

	var output bytes.Buffer
	cmd := NewCmdHello()
	cmd.SetOut(&output)
	cmd.SetArgs([]string{})

	require.NoError(t, cmd.Execute())
	assert.Equal(t, "Hello, World!\n", output.String())
}

func TestNewCmdHelloMetadata(t *testing.T) {
	t.Parallel()

	cmd := NewCmdHello()

	assert.Equal(t, "hello", cmd.Use)
	assert.Equal(t, "Print a hello world greeting", cmd.Short)
}

func TestNewCmdHelloRejectsArguments(t *testing.T) {
	t.Parallel()

	cmd := NewCmdHello()
	cmd.SetArgs([]string{"unexpected"})

	require.Error(t, cmd.Execute())
}
