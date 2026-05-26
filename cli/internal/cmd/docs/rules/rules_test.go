package rules

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCmd_HelpRunsCleanly(t *testing.T) {
	cmd := NewCmdRules()
	cmd.SetArgs([]string{"--help"})
	buf := &bytes.Buffer{}
	cmd.SetOut(buf)
	cmd.SetErr(buf)

	require.NoError(t, cmd.Execute())
	out := buf.String()
	assert.Contains(t, out, "--fragments-dir")
	assert.Contains(t, out, "--output-dir")
	assert.Contains(t, out, "--strict-verify")
}

func TestCmd_StrictVerifyReturnsUnimplementedError(t *testing.T) {
	cmd := NewCmdRules()
	cmd.SetArgs([]string{"--strict-verify"})
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})

	err := cmd.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "strict-verify")
	assert.Contains(t, err.Error(), "not implemented")
}
