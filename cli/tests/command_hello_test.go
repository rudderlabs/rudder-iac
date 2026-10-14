package tests

import (
	"bytes"
	"os/exec"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHello(t *testing.T) {
	home := t.TempDir()
	command := exec.Command(cliBinPath, "hello")
	command.Env = []string{
		"HOME=" + home,
		"XDG_CONFIG_HOME=" + home,
		"RUDDERSTACK_CLI_TELEMETRY_DISABLED=true",
	}

	var (
		stdout bytes.Buffer
		stderr bytes.Buffer
	)
	command.Stdout = &stdout
	command.Stderr = &stderr

	err := command.Run()

	require.NoError(t, err)
	assert.Equal(t, "Hello, World!\n", stdout.String())
	assert.Empty(t, stderr.String())
}
