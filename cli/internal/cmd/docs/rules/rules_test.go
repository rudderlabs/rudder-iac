package rules

import (
	"bytes"
	"testing"

	"github.com/rudderlabs/rudder-iac/cli/internal/validation/docs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRules_HelpListsFlags(t *testing.T) {
	cmd := NewCmdRules()
	out := &bytes.Buffer{}
	cmd.SetOut(out)
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"--help"})

	require.NoError(t, cmd.Execute())

	help := out.String()
	assert.Contains(t, help, "--output-dir")
	assert.Contains(t, help, "--format")
	// The by-version layout left the CLI; grouping is a downstream concern.
	assert.NotContains(t, help, "--layout")
	assert.NotContains(t, help, "--version")
	// --strict-verify is intentionally absent until the verifier exists (DEX-372).
	assert.NotContains(t, help, "--strict-verify")
	assert.NotContains(t, help, "--fragments-dir")
}

func TestValidateFormat(t *testing.T) {
	tests := []struct {
		name    string
		format  string
		wantErr bool
	}{
		{name: "yaml", format: docs.FormatYAML, wantErr: false},
		{name: "json", format: docs.FormatJSON, wantErr: false},
		{name: "both", format: docs.FormatBoth, wantErr: false},
		{name: "bogus", format: "bogus", wantErr: true},
		{name: "empty", format: "", wantErr: true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := validateFormat(tc.format)
			if tc.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
		})
	}
}
