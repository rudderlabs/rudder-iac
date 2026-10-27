package tests

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestHTTPV2Apply runs only on an explicitly disposable, HTTP-v2-enabled
// workspace. Its separate fixture set keeps the shared v1 destination suite
// runnable while the v2 backend and catalogue gate roll out.
func TestHTTPV2Apply(t *testing.T) {
	if os.Getenv("RUN_HTTP_V2_E2E") != "1" {
		t.Skip("set RUN_HTTP_V2_E2E=1 only with a disposable HTTP-v2-enabled workspace")
	}

	executor, err := NewCmdExecutor("")
	require.NoError(t, err)
	out, err := executor.Execute(cliBinPath, "destroy", "--confirm=false")
	require.NoError(t, err, "initial destroy failed: %s", out)
	t.Cleanup(func() {
		out, err := executor.Execute(cliBinPath, "destroy", "--confirm=false")
		assert.NoError(t, err, "cleanup destroy failed: %s", out)
	})

	for _, phase := range []string{"create", "update"} {
		t.Run(phase, func(t *testing.T) {
			fixtureDir := filepath.Join("testdata", "destinations", "http-v2", phase)
			out, err := executor.Execute(cliBinPath, "apply", "-l", fixtureDir, "--confirm=false")
			require.NoError(t, err, "%s apply failed: %s", phase, out)

			verifyDestinationStateAt(t, filepath.Join(
				"testdata", "expected", "upstream", "destinations", "http-v2", phase))
		})
	}
}
