package validate

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rudderlabs/rudder-iac/cli/internal/cmd/telemetry/telemetrytest"
	"github.com/rudderlabs/rudder-iac/cli/internal/testutils"
)

func TestNewCmdValidateAcceptsVarFile(t *testing.T) {
	t.Parallel()

	flag := NewCmdValidate().Flags().Lookup("var-file")
	require.NotNil(t, flag, "data-graphs validate must accept --var-file")
	assert.Equal(t, "stringArray", flag.Value.Type())
}

func TestNewCmdValidatePreRunValidatesModeBeforeDependencies(t *testing.T) {
	t.Parallel()

	cmd := NewCmdValidate()

	err := cmd.PreRunE(cmd, []string{})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "must specify either")
	assert.NotContains(t, err.Error(), "experimental flag")
}

func TestValidateFlags(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		args          []string
		all           bool
		modified      bool
		expectedError bool
		errorContains string
	}{
		// Valid cases
		{
			name:          "valid model ID",
			args:          []string{"model", "my-model"},
			all:           false,
			modified:      false,
			expectedError: false,
		},
		{
			name:          "valid relationship ID",
			args:          []string{"relationship", "my-relationship"},
			all:           false,
			modified:      false,
			expectedError: false,
		},
		{
			name:          "valid --all flag",
			args:          []string{},
			all:           true,
			modified:      false,
			expectedError: false,
		},
		{
			name:          "valid --modified flag",
			args:          []string{},
			all:           false,
			modified:      true,
			expectedError: false,
		},

		// Invalid cases
		{
			name:          "no mode specified",
			args:          []string{},
			all:           false,
			modified:      false,
			expectedError: true,
			errorContains: "must specify either",
		},
		{
			name:          "args + --all",
			args:          []string{"model", "my-model"},
			all:           true,
			modified:      false,
			expectedError: true,
			errorContains: "cannot combine validation modes",
		},
		{
			name:          "args + --modified",
			args:          []string{"model", "my-model"},
			all:           false,
			modified:      true,
			expectedError: true,
			errorContains: "cannot combine validation modes",
		},
		{
			name:          "--all + --modified",
			args:          []string{},
			all:           true,
			modified:      true,
			expectedError: true,
			errorContains: "cannot combine validation modes",
		},
		{
			name:          "all three modes",
			args:          []string{"model", "my-model"},
			all:           true,
			modified:      true,
			expectedError: true,
			errorContains: "cannot combine validation modes",
		},
		{
			name:          "only type without ID",
			args:          []string{"model"},
			all:           false,
			modified:      false,
			expectedError: true,
			errorContains: "expected exactly 2 arguments",
		},
		{
			name:          "too many arguments",
			args:          []string{"model", "id1", "id2"},
			all:           false,
			modified:      false,
			expectedError: true,
			errorContains: "expected exactly 2 arguments",
		},
		{
			name:          "invalid resource type",
			args:          []string{"invalid", "my-id"},
			all:           false,
			modified:      false,
			expectedError: true,
			errorContains: "invalid resource type",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := validateFlags(tt.args, tt.all, tt.modified)

			if tt.expectedError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorContains)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

// literalSecretDestination holds a literal in a secret key, so loading it
// renders a destination/secret-reference warning.
const literalSecretDestination = `version: rudder/v1
kind: destination
metadata:
  name: shop-amplitude
spec:
  id: shop-amplitude
  display_name: Shop Amplitude
  type: am
  definition_version: 1
  enabled: true
  config:
    api_key: fake-literal-amplitude-key
    residency_server: standard
`

const secretWarning = `warning[destination/secret-reference]: secret key "api_key" must hold only a variable reference with no default`

// Under --json stdout holds only the report, so the warnings the project load
// renders go to stderr. Without --json they stay on stdout.
func TestValidateKeepsLoadDiagnosticsOffJSONOutput(t *testing.T) {
	testutils.UseFakeAPI(t)
	telemetrytest.Record(t)

	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "destination.yaml"), []byte(literalSecretDestination), 0o600))

	t.Run("json", func(t *testing.T) {
		stdout, stderr := executeValidate(t, "--all", "--json", "--location", dir)

		assert.JSONEq(t, `{"resources": []}`, stdout)
		assert.Contains(t, stderr, secretWarning)
		assert.Contains(t, stderr, "Found 0 error(s), 1 warning(s)")
		assert.NotContains(t, stdout+stderr, "fake-literal-amplitude-key")
	})

	t.Run("text", func(t *testing.T) {
		stdout, stderr := executeValidate(t, "--all", "--location", dir)

		assert.Contains(t, stdout, secretWarning)
		assert.Contains(t, stdout, "Found 0 error(s), 1 warning(s)\nNo resources to validate\n")
		assert.Empty(t, stderr)
		assert.NotContains(t, stdout, "fake-literal-amplitude-key")
	})
}

func executeValidate(t *testing.T, args ...string) (stdout, stderr string) {
	t.Helper()

	var errOut bytes.Buffer
	stdout = testutils.CaptureStdout(t, func() {
		cmd := NewCmdValidate()
		cmd.SetArgs(args)
		cmd.SetErr(&errOut)
		require.NoError(t, cmd.Execute())
	})
	return stdout, errOut.String()
}
