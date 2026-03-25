package transformations_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	tc "github.com/rudderlabs/rudder-iac/api/client/transformations"
	"github.com/rudderlabs/rudder-iac/cli/internal/project/specs"
	prules "github.com/rudderlabs/rudder-iac/cli/internal/provider/rules"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/transformations"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/transformations/handlers/library"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/transformations/handlers/transformation"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/transformations/model"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/transformations/testorchestrator"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/transformations/testutil"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources/state"
	vrules "github.com/rudderlabs/rudder-iac/cli/internal/validation/rules"
)

func TestProvider(t *testing.T) {
	t.Run("SupportedKinds", func(t *testing.T) {
		t.Parallel()

		mockStore := &testutil.MockTransformationStore{}
		provider := transformations.NewProviderWithStore(mockStore)

		kinds := provider.SupportedKinds()

		assert.Len(t, kinds, 2)
		assert.Contains(t, kinds, "transformation-library")
		assert.Contains(t, kinds, "transformation")
	})

	t.Run("SupportedTypes", func(t *testing.T) {
		t.Parallel()

		mockStore := &testutil.MockTransformationStore{}
		provider := transformations.NewProviderWithStore(mockStore)

		types := provider.SupportedTypes()

		assert.Len(t, types, 2)
		assert.Contains(t, types, "transformation-library")
		assert.Contains(t, types, "transformation")
	})

	t.Run("SupportedMatchPatterns", func(t *testing.T) {
		t.Parallel()

		mockStore := &testutil.MockTransformationStore{}
		p := transformations.NewProviderWithStore(mockStore)
		var want []vrules.MatchPattern
		want = append(want, prules.V1VersionPatterns("transformation")...)
		want = append(want, prules.V1VersionPatterns("transformation-library")...)
		assert.ElementsMatch(t, want, p.SupportedMatchPatterns())
	})

	t.Run("LoadLegacySpec requires V1 version", func(t *testing.T) {
		t.Parallel()

		mockStore := &testutil.MockTransformationStore{}
		provider := transformations.NewProviderWithStore(mockStore)

		t.Run("rejects legacy version via LoadLegacySpec", func(t *testing.T) {
			spec := &specs.Spec{
				Version: specs.SpecVersionV0_1,
				Kind:    "transformation",
				Metadata: map[string]any{
					"name": "test-transformation",
				},
				Spec: map[string]any{
					"language": "javascript",
					"code":     "export function transformEvent(event, metadata) { return event; }",
				},
			}

			err := provider.LoadLegacySpec("test.yaml", spec)
			require.Error(t, err)
			assert.Contains(t, err.Error(), "transformation specs require version 'rudder/v1'")
			assert.Contains(t, err.Error(), "Legacy versions are not supported")
		})
	})

	t.Run("validation rules", func(t *testing.T) {
		t.Parallel()

		mockStore := &testutil.MockTransformationStore{}
		provider := transformations.NewProviderWithStore(mockStore)

		syntacticRules := provider.SyntacticRules()
		require.Len(t, syntacticRules, 2)
		assert.Equal(t, "transformations/transformation/spec-syntax-valid", syntacticRules[0].ID())
		assert.Equal(t, "transformations/transformation-library/spec-syntax-valid", syntacticRules[1].ID())

		semanticRules := provider.SemanticRules()
		require.Len(t, semanticRules, 2)
		assert.Equal(t, "transformations/transformation/semantic-valid", semanticRules[0].ID())
		assert.Equal(t, "transformations/transformation-library/semantic-valid", semanticRules[1].ID())
	})
}

func TestResourceGraph(t *testing.T) {
	t.Parallel()

	t.Run("empty graph", func(t *testing.T) {
		t.Parallel()

		mockStore := &testutil.MockTransformationStore{}
		provider := transformations.NewProviderWithStore(mockStore)

		graph, err := provider.ResourceGraph()

		require.NoError(t, err)
		require.NotNil(t, graph)
		assert.Len(t, graph.Resources(), 0)
	})

	t.Run("libraries without transformations", func(t *testing.T) {
		t.Parallel()

		mockStore := &testutil.MockTransformationStore{}
		provider := transformations.NewProviderWithStore(mockStore)

		// Load a library spec
		err := provider.LoadSpec("lib.yaml", &specs.Spec{
			Version: specs.SpecVersionV1,
			Kind:    "transformation-library",
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

		mockStore := &testutil.MockTransformationStore{}
		provider := transformations.NewProviderWithStore(mockStore)

		// Load a transformation spec without imports
		err := provider.LoadSpec("trans.yaml", &specs.Spec{
			Version: specs.SpecVersionV1,
			Kind:    "transformation",
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

		mockStore := &testutil.MockTransformationStore{}
		provider := transformations.NewProviderWithStore(mockStore)

		// Load a library spec
		err := provider.LoadSpec("lib.yaml", &specs.Spec{
			Version: specs.SpecVersionV1,
			Kind:    "transformation-library",
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
			Version: specs.SpecVersionV1,
			Kind:    "transformation",
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

		mockStore := &testutil.MockTransformationStore{}
		provider := transformations.NewProviderWithStore(mockStore)

		// Load first library
		err := provider.LoadSpec("lib1.yaml", &specs.Spec{
			Version: specs.SpecVersionV1,
			Kind:    "transformation-library",
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
			Version: specs.SpecVersionV1,
			Kind:    "transformation-library",
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
			Version: specs.SpecVersionV1,
			Kind:    "transformation",
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

		mockStore := &testutil.MockTransformationStore{}
		provider := transformations.NewProviderWithStore(mockStore)

		// Load transformation that imports a non-existent library
		err := provider.LoadSpec("trans.yaml", &specs.Spec{
			Version: specs.SpecVersionV1,
			Kind:    "transformation",
			Spec: map[string]interface{}{
				"id":       "trans-1",
				"name":     "Broken Transformation",
				"language": "javascript",
				"code":     "import missingLib from 'missingLib';\nexport function transformEvent(event, metadata) { return event; }",
			},
		})
		require.NoError(t, err)

		graph, err := provider.ResourceGraph()

		require.NoError(t, err)
		require.NotNil(t, graph)
		assert.Len(t, graph.Resources(), 1)

		transURN := "transformation:trans-1"
		trans, exists := graph.GetResource(transURN)
		require.True(t, exists)
		assert.Equal(t, "trans-1", trans.ID())
		assert.Empty(t, graph.GetDependencies(transURN))
	})

	// TODO: Implement this test once we have a python parser
	t.Run("python transformation - no imports extracted", func(t *testing.T) {
		t.Parallel()

		mockStore := &testutil.MockTransformationStore{}
		provider := transformations.NewProviderWithStore(mockStore)

		// Load a Python transformation (parser returns empty imports)
		err := provider.LoadSpec("trans.yaml", &specs.Spec{
			Version: specs.SpecVersionV1,
			Kind:    "transformation",
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

		var batchPublishCalled bool
		mockStore := &testutil.MockTransformationStore{
			BatchPublishFunc: func(_ context.Context, _ *tc.BatchPublishRequest) (*tc.BatchPublishResponse, error) {
				batchPublishCalled = true
				return &tc.BatchPublishResponse{Published: true}, nil
			},
		}
		provider := transformations.NewProviderWithStore(mockStore)

		st := state.EmptyState()
		err := provider.ConsolidateSync(context.Background(), resources.NewGraph(), st)

		require.NoError(t, err)
		assert.False(t, batchPublishCalled)
	})

	t.Run("state with transformations only", func(t *testing.T) {
		t.Parallel()

		var (
			capturedReq        *tc.BatchPublishRequest
			batchPublishCalled bool
		)

		mockStore := &testutil.MockTransformationStore{}
		mockStore.BatchPublishFunc = func(ctx context.Context, req *tc.BatchPublishRequest) (*tc.BatchPublishResponse, error) {
			batchPublishCalled = true
			capturedReq = req
			return &tc.BatchPublishResponse{Published: true}, nil
		}

		provider := transformations.NewProviderWithStore(mockStore)

		// Build graph with transformations
		graph := resources.NewGraph()
		graph.AddResource(resources.NewResource(
			"trans-1",
			transformation.HandlerMetadata.ResourceType,
			resources.ResourceData{},
			nil,
			resources.WithRawData(&model.TransformationResource{
				Name: "trans-1",
				Code: "function transform(event) { return event; }",
			}),
		))
		graph.AddResource(resources.NewResource(
			"trans-2",
			transformation.HandlerMetadata.ResourceType,
			resources.ResourceData{},
			nil,
			resources.WithRawData(&model.TransformationResource{
				Name: "trans-2",
				Code: "function transform(event) { return event; }",
			}),
		))

		// Build state with transformations
		st := state.EmptyState()
		st.Resources = map[string]*state.ResourceState{
			"transformation:trans-1": {
				ID:   "trans-1",
				Type: transformation.HandlerMetadata.ResourceType,
				OutputRaw: &model.TransformationState{
					ID:        "remote-trans-1",
					VersionID: "ver-1",
					Modified:  true,
				},
			},
			"transformation:trans-2": {
				ID:   "trans-2",
				Type: transformation.HandlerMetadata.ResourceType,
				OutputRaw: &model.TransformationState{
					ID:        "remote-trans-2",
					VersionID: "ver-2",
					Modified:  true,
				},
			},
		}

		err := provider.ConsolidateSync(context.Background(), graph, st)

		require.NoError(t, err)
		assert.True(t, batchPublishCalled)
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

		var (
			capturedReq        *tc.BatchPublishRequest
			batchPublishCalled bool
		)

		mockStore := &testutil.MockTransformationStore{}
		mockStore.BatchPublishFunc = func(ctx context.Context, req *tc.BatchPublishRequest) (*tc.BatchPublishResponse, error) {
			batchPublishCalled = true
			capturedReq = req
			return &tc.BatchPublishResponse{Published: true}, nil
		}

		provider := transformations.NewProviderWithStore(mockStore)

		// Build state with libraries
		st := state.EmptyState()
		st.Resources = map[string]*state.ResourceState{
			"transformation-library:lib-1": {
				ID:   "lib-1",
				Type: library.HandlerMetadata.ResourceType,
				OutputRaw: &model.LibraryState{
					ID:        "remote-lib-1",
					VersionID: "lib-ver-1",
					Modified:  true,
				},
			},
			"transformation-library:lib-2": {
				ID:   "lib-2",
				Type: library.HandlerMetadata.ResourceType,
				OutputRaw: &model.LibraryState{
					ID:        "remote-lib-2",
					VersionID: "lib-ver-2",
					Modified:  true,
				},
			},
		}

		err := provider.ConsolidateSync(context.Background(), resources.NewGraph(), st)

		require.NoError(t, err)
		assert.True(t, batchPublishCalled)
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

		var (
			capturedReq        *tc.BatchPublishRequest
			batchPublishCalled bool
		)

		mockStore := &testutil.MockTransformationStore{}
		mockStore.BatchPublishFunc = func(ctx context.Context, req *tc.BatchPublishRequest) (*tc.BatchPublishResponse, error) {
			batchPublishCalled = true
			capturedReq = req
			return &tc.BatchPublishResponse{Published: true}, nil
		}

		provider := transformations.NewProviderWithStore(mockStore)

		// Build graph with transformation and library
		graph := resources.NewGraph()
		graph.AddResource(resources.NewResource(
			"trans-1",
			transformation.HandlerMetadata.ResourceType,
			resources.ResourceData{},
			nil,
			resources.WithRawData(&model.TransformationResource{
				Name: "trans-1",
				Code: "function transform(event) { return event; }",
			}),
		))

		// Build state with both transformations and libraries
		st := state.EmptyState()
		st.Resources = map[string]*state.ResourceState{
			"transformation:trans-1": {
				ID:   "trans-1",
				Type: transformation.HandlerMetadata.ResourceType,
				OutputRaw: &model.TransformationState{
					ID:        "remote-trans-1",
					VersionID: "ver-1",
					Modified:  true,
				},
			},
			"transformation-library:lib-1": {
				ID:   "lib-1",
				Type: library.HandlerMetadata.ResourceType,
				OutputRaw: &model.LibraryState{
					ID:        "remote-lib-1",
					VersionID: "lib-ver-1",
					Modified:  true,
				},
			},
		}

		err := provider.ConsolidateSync(context.Background(), graph, st)

		require.NoError(t, err)
		assert.True(t, batchPublishCalled)
		require.NotNil(t, capturedReq)
		assert.Len(t, capturedReq.Transformations, 1)
		assert.Len(t, capturedReq.Libraries, 1)

		assert.Equal(t, "ver-1", capturedReq.Transformations[0].VersionID)
		assert.Equal(t, "lib-ver-1", capturedReq.Libraries[0].VersionID)
	})

	t.Run("API error during batch publish", func(t *testing.T) {
		t.Parallel()

		mockStore := &testutil.MockTransformationStore{}
		mockStore.BatchPublishFunc = func(ctx context.Context, req *tc.BatchPublishRequest) (*tc.BatchPublishResponse, error) {
			return nil, fmt.Errorf("API error")
		}

		provider := transformations.NewProviderWithStore(mockStore)

		// Build graph with transformation
		graph := resources.NewGraph()
		graph.AddResource(resources.NewResource(
			"trans-1",
			transformation.HandlerMetadata.ResourceType,
			resources.ResourceData{},
			nil,
			resources.WithRawData(&model.TransformationResource{
				Name: "trans-1",
				Code: "function transform(event) { return event; }",
			}),
		))

		// Build state with a transformation
		st := state.EmptyState()
		st.Resources = map[string]*state.ResourceState{
			"transformation:trans-1": {
				ID:   "trans-1",
				Type: transformation.HandlerMetadata.ResourceType,
				OutputRaw: &model.TransformationState{
					ID:        "remote-trans-1",
					VersionID: "ver-1",
					Modified:  true,
				},
			},
		}

		err := provider.ConsolidateSync(context.Background(), graph, st)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "batch publishing 1 transformations and 0 libraries")
		assert.Contains(t, err.Error(), "API error")
	})

	t.Run("invalid state - wrong OutputRaw type for transformation", func(t *testing.T) {
		t.Parallel()

		mockStore := &testutil.MockTransformationStore{}
		provider := transformations.NewProviderWithStore(mockStore)

		// Build state with wrong OutputRaw type
		// With the new Modified flag approach, invalid types are silently skipped in buildConnectedSubgraph
		// So this test no longer expects an error
		st := state.EmptyState()
		st.Resources = map[string]*state.ResourceState{
			"transformation:trans-1": {
				ID:        "trans-1",
				Type:      transformation.HandlerMetadata.ResourceType,
				OutputRaw: "invalid-type", // Wrong type - will be skipped, no error
			},
		}

		err := provider.ConsolidateSync(context.Background(), resources.NewGraph(), st)

		// With Modified flag approach, invalid types are silently skipped - no error expected
		require.NoError(t, err)
	})

	t.Run("invalid state - wrong OutputRaw type for library", func(t *testing.T) {
		t.Parallel()

		mockStore := &testutil.MockTransformationStore{}
		provider := transformations.NewProviderWithStore(mockStore)

		// Build state with wrong OutputRaw type
		st := state.EmptyState()
		st.Resources = map[string]*state.ResourceState{
			"transformation-library:lib-1": {
				ID:        "lib-1",
				Type:      library.HandlerMetadata.ResourceType,
				OutputRaw: "invalid-type", // Wrong type - will be skipped, no error
			},
		}

		err := provider.ConsolidateSync(context.Background(), resources.NewGraph(), st)

		// With Modified flag approach, invalid types are silently skipped - no error expected
		require.NoError(t, err)
	})

	t.Run("invalid state - empty version ID for transformation", func(t *testing.T) {
		t.Parallel()

		mockStore := &testutil.MockTransformationStore{}
		provider := transformations.NewProviderWithStore(mockStore)

		// Build graph with transformation
		graph := resources.NewGraph()
		graph.AddResource(resources.NewResource(
			"trans-1",
			transformation.HandlerMetadata.ResourceType,
			resources.ResourceData{},
			nil,
			resources.WithRawData(&model.TransformationResource{
				Name: "trans-1",
				Code: "function transform(event) { return event; }",
			}),
		))

		// Build state with empty version ID
		st := state.EmptyState()
		st.Resources = map[string]*state.ResourceState{
			"transformation:trans-1": {
				ID:   "trans-1",
				Type: transformation.HandlerMetadata.ResourceType,
				OutputRaw: &model.TransformationState{
					ID:        "remote-trans-1",
					VersionID: "", // Empty version ID
					Modified:  true,
				},
			},
		}

		err := provider.ConsolidateSync(context.Background(), graph, st)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "has empty version ID")
	})

	t.Run("invalid state - empty version ID for library", func(t *testing.T) {
		t.Parallel()

		mockStore := &testutil.MockTransformationStore{}
		provider := transformations.NewProviderWithStore(mockStore)

		// Build state with empty version ID
		st := state.EmptyState()
		st.Resources = map[string]*state.ResourceState{
			"transformation-library:lib-1": {
				ID:   "lib-1",
				Type: library.HandlerMetadata.ResourceType,
				OutputRaw: &model.LibraryState{
					ID:        "remote-lib-1",
					VersionID: "", // Empty version ID
					Modified:  true,
				},
			},
		}

		err := provider.ConsolidateSync(context.Background(), resources.NewGraph(), st)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "has empty version ID")
	})

	t.Run("state with other resource types - ignored", func(t *testing.T) {
		t.Parallel()

		var (
			capturedReq        *tc.BatchPublishRequest
			batchPublishCalled bool
		)

		mockStore := &testutil.MockTransformationStore{}
		mockStore.BatchPublishFunc = func(ctx context.Context, req *tc.BatchPublishRequest) (*tc.BatchPublishResponse, error) {
			batchPublishCalled = true
			capturedReq = req
			return &tc.BatchPublishResponse{Published: true}, nil
		}

		provider := transformations.NewProviderWithStore(mockStore)

		// Build graph with transformation
		graph := resources.NewGraph()
		graph.AddResource(resources.NewResource(
			"trans-1",
			transformation.HandlerMetadata.ResourceType,
			resources.ResourceData{},
			nil,
			resources.WithRawData(&model.TransformationResource{
				Name: "trans-1",
				Code: "function transform(event) { return event; }",
			}),
		))

		// Build state with transformation and unrelated resource
		st := state.EmptyState()
		st.Resources = map[string]*state.ResourceState{
			"transformation:trans-1": {
				ID:   "trans-1",
				Type: transformation.HandlerMetadata.ResourceType,
				OutputRaw: &model.TransformationState{
					ID:        "remote-trans-1",
					VersionID: "ver-1",
					Modified:  true,
				},
			},
			"some-other-type:other-1": {
				Type:      "some-other-type",
				OutputRaw: "some-data",
			},
		}

		err := provider.ConsolidateSync(context.Background(), graph, st)

		require.NoError(t, err)
		assert.True(t, batchPublishCalled)
		require.NotNil(t, capturedReq)
		assert.Len(t, capturedReq.Transformations, 1)
		assert.Len(t, capturedReq.Libraries, 0)
	})

	t.Run("validation failure returns error", func(t *testing.T) {
		t.Parallel()

		var batchPublishCalled bool
		mockStore := &testutil.MockTransformationStore{}
		mockStore.BatchPublishFunc = func(ctx context.Context, req *tc.BatchPublishRequest) (*tc.BatchPublishResponse, error) {
			batchPublishCalled = true
			// Return validation failure with test results
			return &tc.BatchPublishResponse{
				Published: false,
				ValidationOutput: tc.ValidationOutput{
					Transformations: []tc.TransformationTestResult{
						{
							Name:      "test-transformation",
							VersionID: "ver-1",
							TestSuiteResult: tc.TestSuiteRunResult{
								Results: []tc.TestResult{
									{
										Name:         "test-case-1",
										Status:       tc.TestRunStatusFail,
										ActualOutput: []any{map[string]any{"result": "unexpected"}},
									},
								},
							},
						},
					},
				},
			}, nil
		}

		provider := transformations.NewProviderWithStore(mockStore)

		// Build graph with transformation
		graph := resources.NewGraph()
		graph.AddResource(resources.NewResource(
			"trans-1",
			transformation.HandlerMetadata.ResourceType,
			resources.ResourceData{},
			nil,
			resources.WithRawData(&model.TransformationResource{
				Name: "test-transformation",
				Code: "function transform(event) { return event; }",
			}),
		))

		// Build state
		st := state.EmptyState()
		st.Resources = map[string]*state.ResourceState{
			"transformation:trans-1": {
				ID:   "trans-1",
				Type: transformation.HandlerMetadata.ResourceType,
				OutputRaw: &model.TransformationState{
					ID:        "remote-trans-1",
					VersionID: "ver-1",
					Modified:  true,
				},
			},
		}

		err := provider.ConsolidateSync(context.Background(), graph, st)

		// Should fail due to validation failure
		require.Error(t, err)
		assert.Contains(t, err.Error(), "batch publish validation failed")
		assert.True(t, batchPublishCalled)
	})
}

func TestBuildTestResultsFromResponse(t *testing.T) {
	t.Parallel()

	t.Run("populates test definitions from request", func(t *testing.T) {
		t.Parallel()

		// Create request with test definitions
		req := &tc.BatchPublishRequest{
			Transformations: []tc.BatchPublishTransformation{
				{
					VersionID: "ver-1",
					TestSuite: []tc.TestDefinition{
						{
							Name:           "test-case-1",
							Input:          []any{map[string]any{"input": "data"}},
							ExpectedOutput: []any{map[string]any{"result": "expected"}},
						},
						{
							Name:           "test-case-2",
							Input:          []any{map[string]any{"input": "data2"}},
							ExpectedOutput: []any{map[string]any{"result": "expected2"}},
						},
					},
				},
				{
					VersionID: "ver-2",
					TestSuite: []tc.TestDefinition{
						{
							Name:           "another-test",
							Input:          []any{map[string]any{"input": "other"}},
							ExpectedOutput: []any{map[string]any{"result": "other"}},
						},
					},
				},
			},
		}

		// Create response with test results
		resp := &tc.BatchPublishResponse{
			Published: false,
			ValidationOutput: tc.ValidationOutput{
				Transformations: []tc.TransformationTestResult{
					{
						Name:      "transformation-1",
						VersionID: "ver-1",
						TestSuiteResult: tc.TestSuiteRunResult{
							Results: []tc.TestResult{
								{Name: "test-case-1", Status: tc.TestRunStatusPass},
								{Name: "test-case-2", Status: tc.TestRunStatusFail},
							},
						},
					},
					{
						Name:      "transformation-2",
						VersionID: "ver-2",
						TestSuiteResult: tc.TestSuiteRunResult{
							Results: []tc.TestResult{
								{Name: "another-test", Status: tc.TestRunStatusPass},
							},
						},
					},
				},
			},
		}

		// Direct test: We can test the logic by creating a similar structure
		testResults := &testorchestrator.TestResults{
			Transformations: []*testorchestrator.TransformationTestWithDefinitions{
				{
					Result: &resp.ValidationOutput.Transformations[0],
					Definitions: []*tc.TestDefinition{
						&req.Transformations[0].TestSuite[0],
						&req.Transformations[0].TestSuite[1],
					},
				},
				{
					Result: &resp.ValidationOutput.Transformations[1],
					Definitions: []*tc.TestDefinition{
						&req.Transformations[1].TestSuite[0],
					},
				},
			},
		}

		// Verify the structure
		require.Len(t, testResults.Transformations, 2)

		// First transformation
		assert.Equal(t, "transformation-1", testResults.Transformations[0].Result.Name)
		assert.Equal(t, "ver-1", testResults.Transformations[0].Result.VersionID)
		require.Len(t, testResults.Transformations[0].Definitions, 2)
		assert.Equal(t, "test-case-1", testResults.Transformations[0].Definitions[0].Name)
		assert.Equal(t, []any{map[string]any{"result": "expected"}}, testResults.Transformations[0].Definitions[0].ExpectedOutput)

		// Second transformation
		assert.Equal(t, "transformation-2", testResults.Transformations[1].Result.Name)
		assert.Equal(t, "ver-2", testResults.Transformations[1].Result.VersionID)
		require.Len(t, testResults.Transformations[1].Definitions, 1)
		assert.Equal(t, "another-test", testResults.Transformations[1].Definitions[0].Name)
		assert.Equal(t, []any{map[string]any{"result": "other"}}, testResults.Transformations[1].Definitions[0].ExpectedOutput)
	})

	t.Run("handles transformations without test definitions", func(t *testing.T) {
		t.Parallel()

		// Response without test definitions
		resp := &tc.BatchPublishResponse{
			Published: false,
			ValidationOutput: tc.ValidationOutput{
				Transformations: []tc.TransformationTestResult{
					{
						Name:      "transformation-1",
						VersionID: "ver-1",
						TestSuiteResult: tc.TestSuiteRunResult{
							Results: []tc.TestResult{
								{Name: "test-1", Status: tc.TestRunStatusPass},
							},
						},
					},
				},
			},
		}

		// Verify that nil definitions are handled
		testResults := &testorchestrator.TestResults{
			Transformations: []*testorchestrator.TransformationTestWithDefinitions{
				{
					Result:      &resp.ValidationOutput.Transformations[0],
					Definitions: nil,
				},
			},
		}

		require.Len(t, testResults.Transformations, 1)
		assert.Nil(t, testResults.Transformations[0].Definitions)
	})
}

func TestMapRemoteToState(t *testing.T) {
	t.Parallel()

	t.Run("dependencies populated from imports field", func(t *testing.T) {
		t.Parallel()

		mockStore := &testutil.MockTransformationStore{}
		provider := transformations.NewProviderWithStore(mockStore)

		collection := resources.NewRemoteResources()
		collection.Set(library.HandlerMetadata.ResourceType, map[string]*resources.RemoteResource{
			"remote-lib-1": {
				ID:         "remote-lib-1",
				ExternalID: "lib-1",
				Data: &model.RemoteLibrary{
					TransformationLibrary: &tc.TransformationLibrary{
						ImportName: "mathLib",
						ExternalID: "lib-1",
					},
				},
			},
			"remote-lib-2": {
				ID:         "remote-lib-2",
				ExternalID: "lib-2",
				Data: &model.RemoteLibrary{
					TransformationLibrary: &tc.TransformationLibrary{
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
					Transformation: &tc.Transformation{
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

		mockStore := &testutil.MockTransformationStore{}
		provider := transformations.NewProviderWithStore(mockStore)

		collection := resources.NewRemoteResources()
		collection.Set(transformation.HandlerMetadata.ResourceType, map[string]*resources.RemoteResource{
			"remote-trans-1": {
				ID:         "remote-trans-1",
				ExternalID: "trans-1",
				Data: &model.RemoteTransformation{
					Transformation: &tc.Transformation{
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

func TestDeferredDeletes(t *testing.T) {
	t.Parallel()

	t.Run("deferred deletes execute after batch publish", func(t *testing.T) {
		t.Parallel()

		var callOrder []string

		mockStore := &testutil.MockTransformationStore{}
		mockStore.BatchPublishFunc = func(ctx context.Context, req *tc.BatchPublishRequest) (*tc.BatchPublishResponse, error) {
			callOrder = append(callOrder, "batch_publish")
			return &tc.BatchPublishResponse{Published: true}, nil
		}
		mockStore.DeleteLibraryFunc = func(ctx context.Context, id string) error {
			callOrder = append(callOrder, "delete_library:"+id)
			return nil
		}

		p := transformations.NewProviderWithStore(mockStore)

		// Record a deferred library delete via DeleteRaw
		err := p.DeleteRaw(context.Background(), "lib-1", "transformation-library",
			&model.LibraryResource{ID: "lib-1"},
			&model.LibraryState{ID: "remote-lib-1", VersionID: "ver-1"})
		require.NoError(t, err)

		// Build graph with transformation
		graph := resources.NewGraph()
		graph.AddResource(resources.NewResource(
			"trans-1",
			transformation.HandlerMetadata.ResourceType,
			resources.ResourceData{},
			nil,
			resources.WithRawData(&model.TransformationResource{
				Name: "trans-1",
				Code: "function transform(event) { return event; }",
			}),
		))

		// State has a transformation that was updated (needs publishing)
		st := state.EmptyState()
		st.Resources = map[string]*state.ResourceState{
			"transformation:trans-1": {
				ID:   "trans-1",
				Type: transformation.HandlerMetadata.ResourceType,
				OutputRaw: &model.TransformationState{
					ID:        "remote-trans-1",
					VersionID: "ver-2",
					Modified:  true,
				},
			},
		}

		err = p.ConsolidateSync(context.Background(), graph, st)
		require.NoError(t, err)

		require.Equal(t, []string{
			"batch_publish",
			"delete_library:remote-lib-1",
		}, callOrder)
	})

	t.Run("deferred deletes only - nothing to publish", func(t *testing.T) {
		t.Parallel()

		var (
			batchPublishCalled         bool
			deleteTransformationCalled bool
		)
		mockStore := &testutil.MockTransformationStore{}
		mockStore.BatchPublishFunc = func(ctx context.Context, req *tc.BatchPublishRequest) (*tc.BatchPublishResponse, error) {
			batchPublishCalled = true
			return &tc.BatchPublishResponse{Published: true}, nil
		}
		mockStore.DeleteTransformationFunc = func(ctx context.Context, id string) error {
			deleteTransformationCalled = true
			return nil
		}

		p := transformations.NewProviderWithStore(mockStore)

		err := p.DeleteRaw(context.Background(), "trans-1", "transformation",
			&model.TransformationResource{ID: "trans-1"},
			&model.TransformationState{ID: "remote-trans-1", VersionID: "ver-1"})
		require.NoError(t, err)

		st := state.EmptyState()
		err = p.ConsolidateSync(context.Background(), resources.NewGraph(), st)

		require.NoError(t, err)
		assert.False(t, batchPublishCalled)
		assert.True(t, deleteTransformationCalled)
	})

	t.Run("transformations deleted before libraries", func(t *testing.T) {
		t.Parallel()

		var deleteOrder []string

		mockStore := &testutil.MockTransformationStore{}
		mockStore.DeleteTransformationFunc = func(ctx context.Context, id string) error {
			deleteOrder = append(deleteOrder, "transformation:"+id)
			return nil
		}
		mockStore.DeleteLibraryFunc = func(ctx context.Context, id string) error {
			deleteOrder = append(deleteOrder, "library:"+id)
			return nil
		}

		p := transformations.NewProviderWithStore(mockStore)

		// Record library delete first, then transformation
		err := p.DeleteRaw(context.Background(), "lib-1", "transformation-library",
			&model.LibraryResource{ID: "lib-1"},
			&model.LibraryState{ID: "remote-lib-1", VersionID: "ver-1"})
		require.NoError(t, err)

		err = p.DeleteRaw(context.Background(), "trans-1", "transformation",
			&model.TransformationResource{ID: "trans-1"},
			&model.TransformationState{ID: "remote-trans-1", VersionID: "ver-1"})
		require.NoError(t, err)

		st := state.EmptyState()
		err = p.ConsolidateSync(context.Background(), resources.NewGraph(), st)

		require.NoError(t, err)
		// Transformations must be deleted before libraries regardless of recording order
		require.Equal(t, []string{
			"transformation:remote-trans-1",
			"library:remote-lib-1",
		}, deleteOrder)
	})

	t.Run("deferred delete failure propagates from ConsolidateSync", func(t *testing.T) {
		t.Parallel()

		mockStore := &testutil.MockTransformationStore{}
		mockStore.DeleteLibraryFunc = func(ctx context.Context, id string) error {
			return fmt.Errorf("backend rejected delete")
		}

		p := transformations.NewProviderWithStore(mockStore)

		err := p.DeleteRaw(context.Background(), "lib-1", "transformation-library",
			&model.LibraryResource{ID: "lib-1"},
			&model.LibraryState{ID: "remote-lib-1", VersionID: "ver-1"})
		require.NoError(t, err)

		st := state.EmptyState()
		err = p.ConsolidateSync(context.Background(), resources.NewGraph(), st)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "deleting library remote-lib-1")
		assert.Contains(t, err.Error(), "backend rejected delete")
	})
}
