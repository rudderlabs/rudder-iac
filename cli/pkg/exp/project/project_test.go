package project_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/rudderlabs/rudder-iac/cli/internal/testutils"
	"github.com/rudderlabs/rudder-iac/cli/pkg/exp/project"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProjectLoad(t *testing.T) {
	// The create fixtures keep the two api_tracking fields as {{ .VAR }}
	// placeholders. project.Load() resolves them from RUDDER_* env vars, so
	// supply the values here; the e2e suite passes the same ones via the
	// --var-file at tests/testdata/project/substitution.vars.yaml.
	t.Setenv("RUDDER_API_TRACKING_NAME", "API Tracking")
	t.Setenv("RUDDER_API_TRACKING_DESCRIPTION", "This event is triggered every time a user views a product.")

	t.Run("Load project and verify resource graph", func(t *testing.T) {
		graph, err := project.Load(context.Background(), "../../../tests/testdata/project/create")
		require.NoError(t, err)
		require.NotNil(t, graph)

		all := graph.Resources()
		assert.Greater(t, len(all), 0, "expected non-empty resource graph")

		for urn := range all {
			res, found := graph.GetResource(urn)
			assert.True(t, found, "expected to find resource with URN: %s", urn)
			assert.NotNil(t, res, "expected non-nil resource for URN: %s", urn)
		}
	})

	t.Run("Load non-existent project", func(t *testing.T) {
		_, err := project.Load(context.Background(), "non/existent/path")
		require.Error(t, err, "expected error when loading non-existent project")
	})
}

// A program that loads a project may write its own output on stdout, so the
// warnings the load reports go to stderr.
func TestProjectLoadPrintsDiagnosticsOnStderr(t *testing.T) {
	// Load builds the API client, which needs a token, but makes no call.
	t.Setenv("RUDDERSTACK_ACCESS_TOKEN", "fake-token")

	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "destination.yaml"), []byte(`version: rudder/v1
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
`), 0o600))

	var stdout string
	stderr := testutils.CaptureStderr(t, func() {
		stdout = testutils.CaptureStdout(t, func() {
			graph, err := project.Load(context.Background(), dir)
			require.NoError(t, err)
			require.NotNil(t, graph)
		})
	})

	assert.Empty(t, stdout)
	assert.Contains(t, stderr, `warning[destination/secret-reference]: secret key "api_key" must hold only a variable reference with no default`)
	assert.NotContains(t, stderr, "fake-literal-amplitude-key")
}
