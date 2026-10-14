package hello

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewCmdHelloMetadata(t *testing.T) {
	cmd := NewCmdHello()

	assert.Equal(t, "hello", cmd.Use)
	assert.NotEmpty(t, cmd.Short)
}

func TestNewCmdHelloRejectsArguments(t *testing.T) {
	cmd := NewCmdHello()

	assert.Error(t, cmd.Args(cmd, []string{"unexpected"}))
}

func TestNewCmdHelloPrintsMessage(t *testing.T) {
	var output bytes.Buffer
	cmd := NewCmdHello()
	cmd.SetOut(&output)

	require.NoError(t, cmd.Execute())
	assert.Equal(t, "Hello, World!\n", output.String())
}
