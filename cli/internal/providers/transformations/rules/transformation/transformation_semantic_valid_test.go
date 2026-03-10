package transformation

import (
	"testing"

	"github.com/rudderlabs/rudder-iac/cli/internal/project/specs"
	prules "github.com/rudderlabs/rudder-iac/cli/internal/provider/rules"
	libraryhandler "github.com/rudderlabs/rudder-iac/cli/internal/providers/transformations/handlers/library"
	transformationhandler "github.com/rudderlabs/rudder-iac/cli/internal/providers/transformations/handlers/transformation"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/transformations/model"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
	vrules "github.com/rudderlabs/rudder-iac/cli/internal/validation/rules"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewTransformationImportsSemanticValidRule_Metadata(t *testing.T) {
	t.Parallel()

	rule := NewTransformationImportsSemanticValidRule()

	assert.Equal(t, "transformations/transformation/imports-semantic-valid", rule.ID())
	assert.Equal(t, vrules.Error, rule.Severity())
	assert.Equal(t, "transformation imports must resolve to existing transformation libraries", rule.Description())
	assert.Equal(t, prules.V1VersionPatterns("transformation"), rule.AppliesTo())
}

func TestValidateTransformationImports(t *testing.T) {
	t.Parallel()

	t.Run("all imported libraries resolve", func(t *testing.T) {
		t.Parallel()

		spec := specs.TransformationSpec{
			ID:       "trans-1",
			Name:     "Transformation",
			Language: "javascript",
			Code:     "import mathLibrary from 'mathLibrary';",
		}

		graph := newTransformationGraph(
			&model.TransformationResource{
				ID:       "trans-1",
				Language: "javascript",
				Code:     "import mathLibrary from 'mathLibrary';\nexport function transformEvent(event, metadata) { return event; }",
			},
			&model.LibraryResource{ID: "lib-1", ImportName: "mathLibrary"},
		)

		results := validateTransformationImports("", "", nil, spec, graph)
		assert.Empty(t, results)
	})

	t.Run("missing imported library reports code reference", func(t *testing.T) {
		t.Parallel()

		spec := specs.TransformationSpec{
			ID:       "trans-1",
			Name:     "Transformation",
			Language: "javascript",
			Code:     "import missingLib from 'missingLib';",
		}

		graph := newTransformationGraph(
			&model.TransformationResource{
				ID:       "trans-1",
				Language: "javascript",
				Code:     "import missingLib from 'missingLib';\nexport function transformEvent(event, metadata) { return event; }",
			},
		)

		results := validateTransformationImports("", "", nil, spec, graph)
		require.Len(t, results, 1)
		assert.Equal(t, "/code", results[0].Reference)
		assert.Equal(t, "imported transformation library not found: missingLib", results[0].Message)
	})

	t.Run("file-backed spec reports file reference", func(t *testing.T) {
		t.Parallel()

		spec := specs.TransformationSpec{
			ID:       "trans-1",
			Name:     "Transformation",
			Language: "javascript",
			File:     "scripts/transform.js",
		}

		graph := newTransformationGraph(
			&model.TransformationResource{
				ID:       "trans-1",
				Language: "javascript",
				Code:     "import missingLib from 'missingLib';\nexport function transformEvent(event, metadata) { return event; }",
			},
		)

		results := validateTransformationImports("", "", nil, spec, graph)
		require.Len(t, results, 1)
		assert.Equal(t, "/file", results[0].Reference)
		assert.Equal(t, "imported transformation library not found: missingLib", results[0].Message)
	})

	t.Run("missing graph resource produces no diagnostics", func(t *testing.T) {
		t.Parallel()

		spec := specs.TransformationSpec{
			ID:       "trans-1",
			Name:     "Transformation",
			Language: "javascript",
			Code:     "import missingLib from 'missingLib';",
		}

		results := validateTransformationImports("", "", nil, spec, resources.NewGraph())
		assert.Empty(t, results)
	})
}

func newTransformationGraph(
	transformationResource *model.TransformationResource,
	libraries ...*model.LibraryResource,
) *resources.Graph {
	graph := resources.NewGraph()
	graph.AddResource(resources.NewResource(
		transformationResource.ID,
		transformationhandler.HandlerMetadata.ResourceType,
		resources.ResourceData{},
		nil,
		resources.WithRawData(transformationResource),
	))

	for _, libraryResource := range libraries {
		graph.AddResource(resources.NewResource(
			libraryResource.ID,
			libraryhandler.HandlerMetadata.ResourceType,
			resources.ResourceData{},
			nil,
			resources.WithRawData(libraryResource),
		))
	}

	return graph
}
