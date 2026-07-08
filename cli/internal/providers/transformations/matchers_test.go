package transformations

import (
	"testing"

	transformations "github.com/rudderlabs/rudder-iac/api/client/transformations"
	"github.com/rudderlabs/rudder-iac/cli/internal/provider/resolve"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/transformations/model"
	ttypes "github.com/rudderlabs/rudder-iac/cli/internal/providers/transformations/types"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func localLibrary(id, importName string) *resources.Resource {
	return resources.NewResource(id, ttypes.LibraryResourceType, resources.ResourceData{}, []string{},
		resources.WithRawData(&model.LibraryResource{ID: id, ImportName: importName}))
}

func localTransformation(id, name string) *resources.Resource {
	return resources.NewResource(id, ttypes.TransformationResourceType, resources.ResourceData{}, []string{},
		resources.WithRawData(&model.TransformationResource{ID: id, Name: name}))
}

func remoteLibrary(remoteID, importName string) *resources.RemoteResource {
	return &resources.RemoteResource{
		ID: remoteID,
		Data: &model.RemoteLibrary{
			TransformationLibrary: &transformations.TransformationLibrary{ID: remoteID, ImportName: importName},
		},
	}
}

func remoteTransformation(remoteID, name string) *resources.RemoteResource {
	return &resources.RemoteResource{
		ID: remoteID,
		Data: &model.RemoteTransformation{
			Transformation: &transformations.Transformation{ID: remoteID, Name: name},
		},
	}
}

func matchContext(rs ...*resources.Resource) resolve.MatchContext {
	g := resources.NewGraph()
	for _, r := range rs {
		g.AddResource(r)
	}
	return resolve.MatchContext{LocalGraph: g}
}

func TestResourceMatchers_RegistersLibraryThenTransformation(t *testing.T) {
	t.Parallel()

	p := NewProviderWithStore(nil)

	matchers := p.ResourceMatchers()

	require.Len(t, matchers, 2)
	assert.Equal(t, ttypes.LibraryResourceType, matchers[0].ResourceType)
	assert.Equal(t, ttypes.TransformationResourceType, matchers[1].ResourceType)
}

func TestLibraryMatcher(t *testing.T) {
	t.Parallel()

	p := NewProviderWithStore(nil)
	matcher := p.ResourceMatchers()[0]

	t.Run("matches on import name", func(t *testing.T) {
		t.Parallel()

		ctx := matchContext(localLibrary("lodash-lib", "lodash"))

		local := matcher.Match(ctx, remoteLibrary("rem-1", "lodash"))

		require.NotNil(t, local)
		assert.Equal(t, "lodash-lib", local.ID())
	})

	t.Run("no match for different import name", func(t *testing.T) {
		t.Parallel()

		ctx := matchContext(localLibrary("lodash-lib", "lodash"))

		assert.Nil(t, matcher.Match(ctx, remoteLibrary("rem-1", "underscore")))
	})

	t.Run("empty import name never matches", func(t *testing.T) {
		t.Parallel()

		ctx := matchContext(localLibrary("broken-lib", ""))

		assert.Nil(t, matcher.Match(ctx, remoteLibrary("rem-1", "")))
	})
}

func TestTransformationMatcher(t *testing.T) {
	t.Parallel()

	p := NewProviderWithStore(nil)
	matcher := p.ResourceMatchers()[1]

	t.Run("matches on name", func(t *testing.T) {
		t.Parallel()

		ctx := matchContext(localTransformation("enrich-events", "Enrich Events"))

		local := matcher.Match(ctx, remoteTransformation("rem-1", "Enrich Events"))

		require.NotNil(t, local)
		assert.Equal(t, "enrich-events", local.ID())
	})

	t.Run("no match for different name", func(t *testing.T) {
		t.Parallel()

		ctx := matchContext(localTransformation("enrich-events", "Enrich Events"))

		assert.Nil(t, matcher.Match(ctx, remoteTransformation("rem-1", "Drop PII")))
	})

	t.Run("empty name never matches", func(t *testing.T) {
		t.Parallel()

		ctx := matchContext(localTransformation("broken", ""))

		assert.Nil(t, matcher.Match(ctx, remoteTransformation("rem-1", "")))
	})
}
