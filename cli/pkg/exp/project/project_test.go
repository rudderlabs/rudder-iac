package project_test

import (
	"context"
	"testing"

	"github.com/rudderlabs/rudder-iac/cli/pkg/exp/project"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProjectLoad(t *testing.T) {
	t.Setenv("RUDDERSTACK_X_TRANSFORMATIONS", "true")

	// The shared create fixtures keep two api_tracking fields as {{ .VAR }}
	// placeholders. Enable substitution and supply their values via env so the
	// loader resolves them before validation, mirroring how apply/validate run.
	t.Setenv("RUDDERSTACK_CLI_EXPERIMENTAL", "true")
	t.Setenv("RUDDERSTACK_X_ENABLE_VAR_SUBSTITUTION", "true")
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
