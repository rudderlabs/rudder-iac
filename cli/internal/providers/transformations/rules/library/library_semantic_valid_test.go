package library

import (
	"testing"

	"github.com/rudderlabs/rudder-iac/cli/internal/project/specs"
	"github.com/rudderlabs/rudder-iac/cli/internal/provider/rules"
	libraryhandler "github.com/rudderlabs/rudder-iac/cli/internal/providers/transformations/handlers/library"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/transformations/model"
	internalresources "github.com/rudderlabs/rudder-iac/cli/internal/resources"
	vrules "github.com/rudderlabs/rudder-iac/cli/internal/validation/rules"
	"github.com/stretchr/testify/assert"
)

func TestNewLibrarySemanticValidRule_Metadata(t *testing.T) {
	t.Parallel()

	rule := NewLibrarySemanticValidRule()

	assert.Equal(t, "transformations/transformation-library/semantic-valid", rule.ID())
	assert.Equal(t, vrules.Error, rule.Severity())
	assert.Equal(t, "transformation library must be semantically valid", rule.Description())
	assert.Equal(t, rules.V1VersionPatterns("transformation-library"), rule.AppliesTo())
}

func TestValidateLibrarySemanticValid_ValidLibraries(t *testing.T) {
	t.Parallel()

	t.Run("single library with unique import_name", func(t *testing.T) {
		t.Parallel()

		spec := specs.TransformationLibrarySpec{
			ID:         "lib-1",
			Name:       "My Library",
			ImportName: "myLibrary",
			Language:   "javascript",
			Code:       "export function helper() { return 42; }",
		}

		graph := createGraphWithLibrary(&model.LibraryResource{
			ID:         "lib-1",
			Name:       "My Library",
			ImportName: "myLibrary",
			Language:   "javascript",
			Code:       "export function helper() { return 42; }",
		})

		results := validateLibrarySemanticValid("", "", nil, spec, graph)
		assert.Empty(t, results)
	})

	t.Run("multiple libraries with unique import_names", func(t *testing.T) {
		t.Parallel()

		spec := specs.TransformationLibrarySpec{
			ID:         "lib-1",
			Name:       "My Library",
			ImportName: "myLibrary",
			Language:   "javascript",
			Code:       "export function helper() {}",
		}

		graph := createGraphWithLibraries([]*model.LibraryResource{
			{
				ID:         "lib-1",
				Name:       "My Library",
				ImportName: "myLibrary",
				Language:   "javascript",
				Code:       "export function helper() {}",
			},
			{
				ID:         "lib-2",
				Name:       "Other Library",
				ImportName: "otherLibrary",
				Language:   "javascript",
				Code:       "export function other() {}",
			},
		})

		results := validateLibrarySemanticValid("", "", nil, spec, graph)
		assert.Empty(t, results)
	})

	t.Run("library with empty import_name is ignored in uniqueness check", func(t *testing.T) {
		t.Parallel()

		spec := specs.TransformationLibrarySpec{
			ID:         "lib-1",
			Name:       "My Library",
			ImportName: "myLibrary",
			Language:   "javascript",
			Code:       "export function helper() {}",
		}

		graph := createGraphWithLibraries([]*model.LibraryResource{
			{
				ID:         "lib-1",
				Name:       "My Library",
				ImportName: "myLibrary",
				Language:   "javascript",
				Code:       "export function helper() {}",
			},
			{
				ID:         "lib-2",
				Name:       "Other Library",
				ImportName: "",
				Language:   "javascript",
				Code:       "export function other() {}",
			},
		})

		results := validateLibrarySemanticValid("", "", nil, spec, graph)
		assert.Empty(t, results)
	})
}

func TestValidateLibrarySemanticValid_InvalidLibraries(t *testing.T) {
	t.Parallel()

	t.Run("library not found in graph", func(t *testing.T) {
		t.Parallel()

		spec := specs.TransformationLibrarySpec{
			ID:         "lib-1",
			Name:       "My Library",
			ImportName: "myLibrary",
			Language:   "javascript",
			Code:       "export function helper() {}",
		}

		graph := internalresources.NewGraph()

		results := validateLibrarySemanticValid("", "", nil, spec, graph)

		assert.Equal(t, []string{"/id"}, extractReferences(results))
		assert.Equal(t, []string{"'transformation-library' resource not found in graph"}, extractMessages(results))
	})

	t.Run("duplicate import_name reported for all affected libraries", func(t *testing.T) {
		t.Parallel()

		graph := createGraphWithLibraries([]*model.LibraryResource{
			{
				ID:         "lib-1",
				Name:       "First Library",
				ImportName: "myLibrary",
				Language:   "javascript",
				Code:       "export function helper() {}",
			},
			{
				ID:         "lib-2",
				Name:       "Second Library",
				ImportName: "myLibrary",
				Language:   "javascript",
				Code:       "export function other() {}",
			},
		})

		// Validate lib-1 - should report error since myLibrary is duplicated
		spec1 := specs.TransformationLibrarySpec{
			ID:         "lib-1",
			Name:       "First Library",
			ImportName: "myLibrary",
			Language:   "javascript",
			Code:       "export function helper() {}",
		}

		results1 := validateLibrarySemanticValid("", "", nil, spec1, graph)
		assert.Equal(t, []string{"/import_name"}, extractReferences(results1))
		assert.Equal(t, []string{"import_name 'myLibrary' is duplicate"}, extractMessages(results1))

		// Validate lib-2 - should also report error since myLibrary is duplicated
		spec2 := specs.TransformationLibrarySpec{
			ID:         "lib-2",
			Name:       "Second Library",
			ImportName: "myLibrary",
			Language:   "javascript",
			Code:       "export function other() {}",
		}

		results2 := validateLibrarySemanticValid("", "", nil, spec2, graph)
		assert.Equal(t, []string{"/import_name"}, extractReferences(results2))
		assert.Equal(t, []string{"import_name 'myLibrary' is duplicate"}, extractMessages(results2))
	})
}

func createGraphWithLibrary(lib *model.LibraryResource) *internalresources.Graph {
	return createGraphWithLibraries([]*model.LibraryResource{lib})
}

func createGraphWithLibraries(libs []*model.LibraryResource) *internalresources.Graph {
	graph := internalresources.NewGraph()

	for _, lib := range libs {
		resource := internalresources.NewResource(
			lib.ID,
			libraryhandler.HandlerMetadata.ResourceType,
			internalresources.ResourceData{},
			nil,
			internalresources.WithRawData(lib),
		)
		graph.AddResource(resource)
	}

	return graph
}
