package project_test

import (
	"context"
	"testing"

	"github.com/rudderlabs/rudder-iac/cli/pkg/exp/project"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProjectLoad(t *testing.T) {
	t.Parallel()

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
