package project_test

import (
	"context"
	"testing"

	"github.com/rudderlabs/rudder-iac/cli/pkg/exp/project"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProjectLoad(t *testing.T) {
	// The umbrella flag has to come first: every per-flag setting below is
	// ignored unless it is set. The create fixtures are shared with the e2e
	// suites and carry a destination and an event-stream-connections spec, so
	// those kinds need their flags on to pass syntax validation.
	t.Setenv("RUDDERSTACK_CLI_EXPERIMENTAL", "true")
	t.Setenv("RUDDERSTACK_X_TRANSFORMATIONS", "true")
	t.Setenv("RUDDERSTACK_X_DESTINATION_SUPPORT", "true")
	t.Setenv("RUDDERSTACK_X_CONNECTION_SUPPORT", "true")
	t.Setenv("RUDDERSTACK_X_ENABLE_VAR_SUBSTITUTION", "true")

	// Those same fixtures keep several fields as {{ .VAR }} placeholders.
	// project.Load() resolves them from RUDDER_* env vars, so supply the values
	// here; the e2e suites pass the same ones via the --var-file at
	// tests/testdata/project/substitution.vars.yaml.
	t.Setenv("RUDDER_API_TRACKING_NAME", "API Tracking")
	t.Setenv("RUDDER_API_TRACKING_DESCRIPTION", "This event is triggered every time a user views a product.")
	t.Setenv("RUDDER_S3_ACCESS_KEY_ID", "AKIAXXXXXXXXXXXXXXXX")
	t.Setenv("RUDDER_S3_ACCESS_KEY", "wJalrXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX")

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
