package retlsource

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

// Under --json stdout holds only the rows, so the warnings the project load
// renders go to stderr. Without --json they stay on stdout.
func TestPreviewKeepsLoadDiagnosticsOffJSONOutput(t *testing.T) {
	testutils.UseFakeAPIWith(t, map[string]string{
		"/v2/retl-sources/preview":           `{"id": "preview-1"}`,
		"/v2/retl-sources/preview/preview-1": `{"status": "completed", "rows": [{"id": 1}]}`,
	})
	telemetrytest.Record(t)

	_, _, dir := projectDirs(t)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "destination.yaml"), []byte(literalSecretDestination), 0o600))

	t.Run("json", func(t *testing.T) {
		stdout, stderr := executePreview(t, "user-orders-model", "--json", "--interactive=false", "--location", dir)

		assert.JSONEq(t, `[{"id": 1}]`, stdout)
		assert.Contains(t, stderr, secretWarning)
		assert.Contains(t, stderr, "Found 0 error(s), 1 warning(s)")
		assert.NotContains(t, stdout+stderr, "fake-literal-amplitude-key")
	})

	t.Run("text", func(t *testing.T) {
		stdout, stderr := executePreview(t, "user-orders-model", "--interactive=false", "--location", dir)

		assert.Contains(t, stdout, secretWarning)
		assert.Contains(t, stdout, "Found 0 error(s), 1 warning(s)")
		assert.Empty(t, stderr)
		assert.NotContains(t, stdout, "fake-literal-amplitude-key")
	})
}

func executePreview(t *testing.T, args ...string) (stdout, stderr string) {
	t.Helper()

	var errOut bytes.Buffer
	stdout = testutils.CaptureStdout(t, func() {
		cmd := newCmdPreview()
		cmd.SetArgs(args)
		cmd.SetErr(&errOut)
		require.NoError(t, cmd.Execute())
	})
	return stdout, errOut.String()
}
