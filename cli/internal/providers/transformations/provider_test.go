package transformations_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	transformationsClient "github.com/rudderlabs/rudder-iac/api/client/transformations"
	"github.com/rudderlabs/rudder-iac/cli/internal/project/specs"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/transformations"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/transformations/handlers/library"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/transformations/handlers/transformation"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/transformations/model"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources/state"
)

// mockTransformationStore implements the TransformationStore interface for testing
type mockTransformationStore struct {
	batchPublishCalled bool
	batchPublishFunc   func(ctx context.Context, req *transformationsClient.BatchPublishRequest) error
}

func newMockTransformationStore() *mockTransformationStore {
	return &mockTransformationStore{}
}

func (m *mockTransformationStore) BatchPublish(ctx context.Context, req *transformationsClient.BatchPublishRequest) error {
	m.batchPublishCalled = true
	if m.batchPublishFunc != nil {
		return m.batchPublishFunc(ctx, req)
	}
	return nil
}

func (m *mockTransformationStore) BatchTest(ctx context.Context, req *transformationsClient.BatchTestRequest) ([]*transformationsClient.TransformationTestResult, error) {
	return nil, fmt.Errorf("not implemented")
}

func (m *mockTransformationStore) CreateTransformation(ctx context.Context, req *transformationsClient.CreateTransformationRequest, publish bool) (*transformationsClient.Transformation, error) {
	return nil, fmt.Errorf("not implemented")
}

func (m *mockTransformationStore) UpdateTransformation(ctx context.Context, id string, req *transformationsClient.UpdateTransformationRequest, publish bool) (*transformationsClient.Transformation, error) {
	return nil, fmt.Errorf("not implemented")
}

func (m *mockTransformationStore) GetTransformation(ctx context.Context, id string) (*transformationsClient.Transformation, error) {
	return nil, fmt.Errorf("not implemented")
}

func (m *mockTransformationStore) ListTransformations(ctx context.Context) ([]*transformationsClient.Transformation, error) {
	return []*transformationsClient.Transformation{}, nil
}

func (m *mockTransformationStore) DeleteTransformation(ctx context.Context, id string) error {
	return fmt.Errorf("not implemented")
}

func (m *mockTransformationStore) SetTransformationExternalID(ctx context.Context, id string, externalID string) error {
	return fmt.Errorf("not implemented")
}

func (m *mockTransformationStore) CreateLibrary(ctx context.Context, req *transformationsClient.CreateLibraryRequest, publish bool) (*transformationsClient.TransformationLibrary, error) {
	return nil, fmt.Errorf("not implemented")
}

func (m *mockTransformationStore) UpdateLibrary(ctx context.Context, id string, req *transformationsClient.UpdateLibraryRequest, publish bool) (*transformationsClient.TransformationLibrary, error) {
	return nil, fmt.Errorf("not implemented")
}

func (m *mockTransformationStore) GetLibrary(ctx context.Context, id string) (*transformationsClient.TransformationLibrary, error) {
	return nil, fmt.Errorf("not implemented")
}

func (m *mockTransformationStore) ListLibraries(ctx context.Context) ([]*transformationsClient.TransformationLibrary, error) {
	return []*transformationsClient.TransformationLibrary{}, nil
}

func (m *mockTransformationStore) DeleteLibrary(ctx context.Context, id string) error {
	return fmt.Errorf("not implemented")
}

func (m *mockTransformationStore) SetLibraryExternalID(ctx context.Context, id string, externalID string) error {
	return fmt.Errorf("not implemented")
}

func TestProvider(t *testing.T) {
	t.Run("SupportedKinds", func(t *testing.T) {
		t.Parallel()

		mockStore := newMockTransformationStore()
		provider := transformations.NewProviderWithStore(mockStore)

		kinds := provider.SupportedKinds()

		assert.Len(t, kinds, 2)
		assert.Contains(t, kinds, "transformation-library")
		assert.Contains(t, kinds, "transformation")
	})

	t.Run("SupportedTypes", func(t *testing.T) {
		t.Parallel()

		mockStore := newMockTransformationStore()
		provider := transformations.NewProviderWithStore(mockStore)

		types := provider.SupportedTypes()

		assert.Len(t, types, 2)
		assert.Contains(t, types, "transformation-library")
		assert.Contains(t, types, "transformation")
	})
}

func TestResourceGraph(t *testing.T) {
	t.Parallel()

	t.Run("empty graph", func(t *testing.T) {
		t.Parallel()

		mockStore := newMockTransformationStore()
		provider := transformations.NewProviderWithStore(mockStore)

		graph, err := provider.ResourceGraph()

		require.NoError(t, err)
		require.NotNil(t, graph)
		assert.Len(t, graph.Resources(), 0)
	})

	t.Run("libraries without transformations", func(t *testing.T) {
		t.Parallel()

		mockStore := newMockTransformationStore()
		provider := transformations.NewProviderWithStore(mockStore)

		// Load a library spec
		err := provider.LoadSpec("lib.yaml", &specs.Spec{
			Kind: "transformation-library",
			Spec: map[string]interface{}{
				"id":          "lib-1",
				"name":        "Math Library",
				"language":    "javascript",
				"code":        "export function add(a, b) { return a + b; }",
				"import_name": "mathLibrary",
			},
		})
		require.NoError(t, err)

		graph, err := provider.ResourceGraph()

		require.NoError(t, err)
		require.NotNil(t, graph)
		assert.Len(t, graph.Resources(), 1)

		// Verify the library resource
		libURN := "transformation-library:lib-1"
		lib, exists := graph.GetResource(libURN)
		require.True(t, exists)
		assert.Equal(t, "lib-1", lib.ID())
		assert.Equal(t, "transformation-library", lib.Type())
	})

	t.Run("transformation without library dependencies", func(t *testing.T) {
		t.Parallel()

		mockStore := newMockTransformationStore()
		provider := transformations.NewProviderWithStore(mockStore)

		// Load a transformation spec without imports
		err := provider.LoadSpec("trans.yaml", &specs.Spec{
			Kind: "transformation",
			Spec: map[string]interface{}{
				"id":       "trans-1",
				"name":     "Simple Transformation",
				"language": "javascript",
				"code":     "export function transformEvent(event, metadata) { return event; }",
			},
		})
		require.NoError(t, err)

		graph, err := provider.ResourceGraph()

		require.NoError(t, err)
		require.NotNil(t, graph)
		assert.Len(t, graph.Resources(), 1)

		// Verify the transformation resource
		transURN := "transformation:trans-1"
		trans, exists := graph.GetResource(transURN)
		require.True(t, exists)
		assert.Equal(t, "trans-1", trans.ID())
		assert.Equal(t, "transformation", trans.Type())
	})

	t.Run("transformation with library dependency", func(t *testing.T) {
		t.Parallel()

		mockStore := newMockTransformationStore()
		provider := transformations.NewProviderWithStore(mockStore)

		// Load a library spec
		err := provider.LoadSpec("lib.yaml", &specs.Spec{
			Kind: "transformation-library",
			Spec: map[string]interface{}{
				"id":          "lib-1",
				"name":        "Math Library",
				"language":    "javascript",
				"code":        "export function add(a, b) { return a + b; }",
				"import_name": "mathLibrary",
			},
		})
		require.NoError(t, err)

		// Load a transformation spec that imports the library
		err = provider.LoadSpec("trans.yaml", &specs.Spec{
			Kind: "transformation",
			Spec: map[string]interface{}{
				"id":       "trans-1",
				"name":     "Math Transformation",
				"language": "javascript",
				"code":     "import mathLibrary from 'mathLibrary';\nexport function transformEvent(event, metadata) { return event; }",
			},
		})
		require.NoError(t, err)

		graph, err := provider.ResourceGraph()

		require.NoError(t, err)
		require.NotNil(t, graph)
		assert.Len(t, graph.Resources(), 2)

		// Verify dependency was added
		transURN := "transformation:trans-1"
		libURN := "transformation-library:lib-1"

		deps := graph.GetDependencies(transURN)
		require.Len(t, deps, 1)
		assert.Equal(t, libURN, deps[0])
	})

	t.Run("transformation with multiple library dependencies", func(t *testing.T) {
		t.Parallel()

		mockStore := newMockTransformationStore()
		provider := transformations.NewProviderWithStore(mockStore)

		// Load first library
		err := provider.LoadSpec("lib1.yaml", &specs.Spec{
			Kind: "transformation-library",
			Spec: map[string]interface{}{
				"id":          "lib-1",
				"name":        "Math Library",
				"language":    "javascript",
				"code":        "export function add(a, b) { return a + b; }",
				"import_name": "mathLibrary",
			},
		})
		require.NoError(t, err)

		// Load second library
		err = provider.LoadSpec("lib2.yaml", &specs.Spec{
			Kind: "transformation-library",
			Spec: map[string]interface{}{
				"id":          "lib-2",
				"name":        "String Library",
				"language":    "javascript",
				"code":        "export function upper(s) { return s.toUpperCase(); }",
				"import_name": "stringLibrary",
			},
		})
		require.NoError(t, err)

		// Load transformation that imports both libraries
		err = provider.LoadSpec("trans.yaml", &specs.Spec{
			Kind: "transformation",
			Spec: map[string]interface{}{
				"id":       "trans-1",
				"name":     "Complex Transformation",
				"language": "javascript",
				"code":     "import mathLibrary from 'mathLibrary';\nimport stringLibrary from 'stringLibrary';\nexport function transformEvent(event, metadata) { return event; }",
			},
		})
		require.NoError(t, err)

		graph, err := provider.ResourceGraph()

		require.NoError(t, err)
		require.NotNil(t, graph)
		assert.Len(t, graph.Resources(), 3)

		// Verify dependencies were added
		transURN := "transformation:trans-1"
		lib1URN := "transformation-library:lib-1"
		lib2URN := "transformation-library:lib-2"

		deps := graph.GetDependencies(transURN)
		require.Len(t, deps, 2)
		assert.Contains(t, deps, lib1URN)
		assert.Contains(t, deps, lib2URN)
	})

	t.Run("transformation with missing library dependency", func(t *testing.T) {
		t.Parallel()

		mockStore := newMockTransformationStore()
		provider := transformations.NewProviderWithStore(mockStore)

		// Load transformation that imports a non-existent library
		err := provider.LoadSpec("trans.yaml", &specs.Spec{
			Kind: "transformation",
			Spec: map[string]interface{}{
				"id":       "trans-1",
				"name":     "Broken Transformation",
				"language": "javascript",
				"code":     "import missingLib from 'missingLib';\nexport function transformEvent(event, metadata) { return event; }",
			},
		})
		require.NoError(t, err)

		graph, err := provider.ResourceGraph()

		require.Error(t, err)
		assert.Contains(t, err.Error(), "transformation trans-1 importing library with import_name 'missingLib' not found")
		assert.Nil(t, graph)
	})

	// TODO: Implement this test once we have a python parser
	t.Run("python transformation - no imports extracted", func(t *testing.T) {
		t.Parallel()

		mockStore := newMockTransformationStore()
		provider := transformations.NewProviderWithStore(mockStore)

		// Load a Python transformation (parser returns empty imports)
		err := provider.LoadSpec("trans.yaml", &specs.Spec{
			Kind: "transformation",
			Spec: map[string]interface{}{
				"id":       "trans-1",
				"name":     "Python Transformation",
				"language": "python",
				"code":     "def transform(event):\n    return event",
			},
		})
		require.NoError(t, err)

		graph, err := provider.ResourceGraph()

		require.NoError(t, err)
		require.NotNil(t, graph)
		assert.Len(t, graph.Resources(), 1)

		// Verify no dependencies (Python parser is skeleton)
		transURN := "transformation:trans-1"
		deps := graph.GetDependencies(transURN)
		assert.Len(t, deps, 0)
	})
}

func TestConsolidateSync(t *testing.T) {
	t.Parallel()

	t.Run("empty state - no batch publish", func(t *testing.T) {
		t.Parallel()

		mockStore := newMockTransformationStore()
		provider := transformations.NewProviderWithStore(mockStore)

		st := state.EmptyState()
		err := provider.ConsolidateSync(context.Background(), st)

		require.NoError(t, err)
		assert.False(t, mockStore.batchPublishCalled)
	})

	t.Run("state with transformations only", func(t *testing.T) {
		t.Parallel()

		var capturedReq *transformationsClient.BatchPublishRequest

		mockStore := newMockTransformationStore()
		mockStore.batchPublishFunc = func(ctx context.Context, req *transformationsClient.BatchPublishRequest) error {
			capturedReq = req
			return nil
		}

		provider := transformations.NewProviderWithStore(mockStore)

		// Build state with transformations
		st := state.EmptyState()
		st.Resources = map[string]*state.ResourceState{
			"transformation:trans-1": {
				Type: transformation.HandlerMetadata.ResourceType,
				OutputRaw: &model.TransformationState{
					ID:        "remote-trans-1",
					VersionID: "ver-1",
				},
			},
			"transformation:trans-2": {
				Type: transformation.HandlerMetadata.ResourceType,
				OutputRaw: &model.TransformationState{
					ID:        "remote-trans-2",
					VersionID: "ver-2",
				},
			},
		}

		err := provider.ConsolidateSync(context.Background(), st)

		require.NoError(t, err)
		assert.True(t, mockStore.batchPublishCalled)
		require.NotNil(t, capturedReq)
		assert.Len(t, capturedReq.Transformations, 2)
		assert.Len(t, capturedReq.Libraries, 0)

		// Verify version IDs
		versionIDs := []string{
			capturedReq.Transformations[0].VersionID,
			capturedReq.Transformations[1].VersionID,
		}
		assert.Contains(t, versionIDs, "ver-1")
		assert.Contains(t, versionIDs, "ver-2")
	})

	t.Run("state with libraries only", func(t *testing.T) {
		t.Parallel()

		var capturedReq *transformationsClient.BatchPublishRequest

		mockStore := newMockTransformationStore()
		mockStore.batchPublishFunc = func(ctx context.Context, req *transformationsClient.BatchPublishRequest) error {
			capturedReq = req
			return nil
		}

		provider := transformations.NewProviderWithStore(mockStore)

		// Build state with libraries
		st := state.EmptyState()
		st.Resources = map[string]*state.ResourceState{
			"transformation-library:lib-1": {
				Type: library.HandlerMetadata.ResourceType,
				OutputRaw: &model.LibraryState{
					ID:        "remote-lib-1",
					VersionID: "lib-ver-1",
				},
			},
			"transformation-library:lib-2": {
				Type: library.HandlerMetadata.ResourceType,
				OutputRaw: &model.LibraryState{
					ID:        "remote-lib-2",
					VersionID: "lib-ver-2",
				},
			},
		}

		err := provider.ConsolidateSync(context.Background(), st)

		require.NoError(t, err)
		assert.True(t, mockStore.batchPublishCalled)
		require.NotNil(t, capturedReq)
		assert.Len(t, capturedReq.Transformations, 0)
		assert.Len(t, capturedReq.Libraries, 2)

		// Verify version IDs
		versionIDs := []string{
			capturedReq.Libraries[0].VersionID,
			capturedReq.Libraries[1].VersionID,
		}
		assert.Contains(t, versionIDs, "lib-ver-1")
		assert.Contains(t, versionIDs, "lib-ver-2")
	})

	t.Run("state with both transformations and libraries", func(t *testing.T) {
		t.Parallel()

		var capturedReq *transformationsClient.BatchPublishRequest

		mockStore := newMockTransformationStore()
		mockStore.batchPublishFunc = func(ctx context.Context, req *transformationsClient.BatchPublishRequest) error {
			capturedReq = req
			return nil
		}

		provider := transformations.NewProviderWithStore(mockStore)

		// Build state with both transformations and libraries
		st := state.EmptyState()
		st.Resources = map[string]*state.ResourceState{
			"transformation:trans-1": {
				Type: transformation.HandlerMetadata.ResourceType,
				OutputRaw: &model.TransformationState{
					ID:        "remote-trans-1",
					VersionID: "ver-1",
				},
			},
			"transformation-library:lib-1": {
				Type: library.HandlerMetadata.ResourceType,
				OutputRaw: &model.LibraryState{
					ID:        "remote-lib-1",
					VersionID: "lib-ver-1",
				},
			},
		}

		err := provider.ConsolidateSync(context.Background(), st)

		require.NoError(t, err)
		assert.True(t, mockStore.batchPublishCalled)
		require.NotNil(t, capturedReq)
		assert.Len(t, capturedReq.Transformations, 1)
		assert.Len(t, capturedReq.Libraries, 1)

		assert.Equal(t, "ver-1", capturedReq.Transformations[0].VersionID)
		assert.Equal(t, "lib-ver-1", capturedReq.Libraries[0].VersionID)
	})

	t.Run("API error during batch publish", func(t *testing.T) {
		t.Parallel()

		mockStore := newMockTransformationStore()
		mockStore.batchPublishFunc = func(ctx context.Context, req *transformationsClient.BatchPublishRequest) error {
			return fmt.Errorf("API error")
		}

		provider := transformations.NewProviderWithStore(mockStore)

		// Build state with a transformation
		st := state.EmptyState()
		st.Resources = map[string]*state.ResourceState{
			"transformation:trans-1": {
				Type: transformation.HandlerMetadata.ResourceType,
				OutputRaw: &model.TransformationState{
					ID:        "remote-trans-1",
					VersionID: "ver-1",
				},
			},
		}

		err := provider.ConsolidateSync(context.Background(), st)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "batch publishing 1 transformations and 0 libraries")
		assert.Contains(t, err.Error(), "API error")
	})

	t.Run("invalid state - wrong OutputRaw type for transformation", func(t *testing.T) {
		t.Parallel()

		mockStore := newMockTransformationStore()
		provider := transformations.NewProviderWithStore(mockStore)

		// Build state with wrong OutputRaw type
		st := state.EmptyState()
		st.Resources = map[string]*state.ResourceState{
			"transformation:trans-1": {
				Type:      transformation.HandlerMetadata.ResourceType,
				OutputRaw: "invalid-type", // Should be *model.TransformationState
			},
		}

		err := provider.ConsolidateSync(context.Background(), st)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid OutputRaw type for transformation")
	})

	t.Run("invalid state - wrong OutputRaw type for library", func(t *testing.T) {
		t.Parallel()

		mockStore := newMockTransformationStore()
		provider := transformations.NewProviderWithStore(mockStore)

		// Build state with wrong OutputRaw type
		st := state.EmptyState()
		st.Resources = map[string]*state.ResourceState{
			"transformation-library:lib-1": {
				Type:      library.HandlerMetadata.ResourceType,
				OutputRaw: "invalid-type", // Should be *model.LibraryState
			},
		}

		err := provider.ConsolidateSync(context.Background(), st)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid OutputRaw type for library")
	})

	t.Run("invalid state - empty version ID for transformation", func(t *testing.T) {
		t.Parallel()

		mockStore := newMockTransformationStore()
		provider := transformations.NewProviderWithStore(mockStore)

		// Build state with empty version ID
		st := state.EmptyState()
		st.Resources = map[string]*state.ResourceState{
			"transformation:trans-1": {
				Type: transformation.HandlerMetadata.ResourceType,
				OutputRaw: &model.TransformationState{
					ID:        "remote-trans-1",
					VersionID: "", // Empty version ID
				},
			},
		}

		err := provider.ConsolidateSync(context.Background(), st)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "has empty version ID")
	})

	t.Run("invalid state - empty version ID for library", func(t *testing.T) {
		t.Parallel()

		mockStore := newMockTransformationStore()
		provider := transformations.NewProviderWithStore(mockStore)

		// Build state with empty version ID
		st := state.EmptyState()
		st.Resources = map[string]*state.ResourceState{
			"transformation-library:lib-1": {
				Type: library.HandlerMetadata.ResourceType,
				OutputRaw: &model.LibraryState{
					ID:        "remote-lib-1",
					VersionID: "", // Empty version ID
				},
			},
		}

		err := provider.ConsolidateSync(context.Background(), st)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "has empty version ID")
	})

	t.Run("state with other resource types - ignored", func(t *testing.T) {
		t.Parallel()

		var capturedReq *transformationsClient.BatchPublishRequest

		mockStore := newMockTransformationStore()
		mockStore.batchPublishFunc = func(ctx context.Context, req *transformationsClient.BatchPublishRequest) error {
			capturedReq = req
			return nil
		}

		provider := transformations.NewProviderWithStore(mockStore)

		// Build state with transformation and unrelated resource
		st := state.EmptyState()
		st.Resources = map[string]*state.ResourceState{
			"transformation:trans-1": {
				Type: transformation.HandlerMetadata.ResourceType,
				OutputRaw: &model.TransformationState{
					ID:        "remote-trans-1",
					VersionID: "ver-1",
				},
			},
			"some-other-type:other-1": {
				Type:      "some-other-type",
				OutputRaw: "some-data",
			},
		}

		err := provider.ConsolidateSync(context.Background(), st)

		require.NoError(t, err)
		assert.True(t, mockStore.batchPublishCalled)
		require.NotNil(t, capturedReq)
		assert.Len(t, capturedReq.Transformations, 1)
		assert.Len(t, capturedReq.Libraries, 0)
	})
}

func TestMapRemoteToState(t *testing.T) {
	t.Parallel()

	t.Run("dependencies populated from imports field", func(t *testing.T) {
		t.Parallel()

		mockStore := newMockTransformationStore()
		provider := transformations.NewProviderWithStore(mockStore)

		collection := resources.NewRemoteResources()
		collection.Set(library.HandlerMetadata.ResourceType, map[string]*resources.RemoteResource{
			"remote-lib-1": {
				ID:         "remote-lib-1",
				ExternalID: "lib-1",
				Data: &model.RemoteLibrary{
					TransformationLibrary: &transformationsClient.TransformationLibrary{
						ImportName: "mathLib",
						ExternalID: "lib-1",
					},
				},
			},
			"remote-lib-2": {
				ID:         "remote-lib-2",
				ExternalID: "lib-2",
				Data: &model.RemoteLibrary{
					TransformationLibrary: &transformationsClient.TransformationLibrary{
						ImportName: "stringLib",
						ExternalID: "lib-2",
					},
				},
			},
		})
		collection.Set(transformation.HandlerMetadata.ResourceType, map[string]*resources.RemoteResource{
			"remote-trans-1": {
				ID:         "remote-trans-1",
				ExternalID: "trans-1",
				Data: &model.RemoteTransformation{
					Transformation: &transformationsClient.Transformation{
						Imports:    []string{"mathLib", "stringLib"},
						ExternalID: "trans-1",
					},
				},
			},
		})

		st, err := provider.MapRemoteToState(collection)

		require.NoError(t, err)
		resourceState := st.GetResource("transformation:trans-1")
		require.NotNil(t, resourceState)
		require.Len(t, resourceState.Dependencies, 2)
		assert.Contains(t, resourceState.Dependencies, "transformation-library:lib-1")
		assert.Contains(t, resourceState.Dependencies, "transformation-library:lib-2")
	})

	t.Run("unresolved library import skipped", func(t *testing.T) {
		t.Parallel()

		mockStore := newMockTransformationStore()
		provider := transformations.NewProviderWithStore(mockStore)

		collection := resources.NewRemoteResources()
		collection.Set(transformation.HandlerMetadata.ResourceType, map[string]*resources.RemoteResource{
			"remote-trans-1": {
				ID:         "remote-trans-1",
				ExternalID: "trans-1",
				Data: &model.RemoteTransformation{
					Transformation: &transformationsClient.Transformation{
						Imports:    []string{"unmanagedLib"},
						ExternalID: "trans-1",
					},
				},
			},
		})

		st, err := provider.MapRemoteToState(collection)

		require.NoError(t, err)
		resourceState := st.GetResource("transformation:trans-1")
		require.NotNil(t, resourceState)
		assert.Len(t, resourceState.Dependencies, 0)
	})
}
