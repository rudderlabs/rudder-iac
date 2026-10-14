package hello

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestHelloPrintsGreeting(t *testing.T) {
	cmd := NewCmdHello()
	output := &bytes.Buffer{}
	cmd.SetOut(output)

	require.NoError(t, cmd.Execute())
	require.Equal(t, "Hello Foo!\n", output.String())
}

func TestHelloRejectsPositionalArguments(t *testing.T) {
	cmd := NewCmdHello()
	cmd.SetArgs([]string{"unexpected"})

	require.Error(t, cmd.Execute())
}
