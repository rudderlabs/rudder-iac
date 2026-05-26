package rules

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCmdRules_StrictVerifyFlagReturnsErrorInSpike(t *testing.T) {
	cmd := NewCmdRules()
	cmd.SetArgs([]string{"--strict-verify"})
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	err := cmd.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "strict-verify mode is not implemented")
}

func TestCmdRules_DefaultsAreDocumentedInHelp(t *testing.T) {
	cmd := NewCmdRules()
	flag := cmd.Flags().Lookup("output-dir")
	require.NotNil(t, flag)
	assert.Equal(t, "./docs/generated/", flag.DefValue)

	strict := cmd.Flags().Lookup("strict-verify")
	require.NotNil(t, strict)
	assert.Contains(t, strict.Usage, "not implemented")
}
