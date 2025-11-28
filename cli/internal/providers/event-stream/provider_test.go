package eventstream_test

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	sourceClient "github.com/rudderlabs/rudder-iac/api/client/event-stream/source"
	"github.com/rudderlabs/rudder-iac/cli/internal/namer"
	"github.com/rudderlabs/rudder-iac/cli/internal/project/specs"
	eventstream "github.com/rudderlabs/rudder-iac/cli/internal/providers/event-stream"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/event-stream/source"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources/state"
)

func TestProvider(t *testing.T) {
	t.Run("SupportedKinds", func(t *testing.T) {
		provider := eventstream.New(source.NewMockSourceClient())
		kinds := provider.SupportedKinds()
		assert.Contains(t, kinds, "event-stream-source")
		assert.Len(t, kinds, 1)
	})

	t.Run("SupportedTypes", func(t *testing.T) {
		provider := eventstream.New(source.NewMockSourceClient())
		types := provider.SupportedTypes()
		assert.Contains(t, types, source.ResourceType)
		assert.Len(t, types, 1)
	})

	t.Run("LoadSpec", func(t *testing.T) {
		t.Run("UnsupportedKind", func(t *testing.T) {
			provider := eventstream.New(source.NewMockSourceClient())
			err := provider.LoadSpec("", &specs.Spec{Kind: "unsupported"})
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "unsupported kind")
		})

		t.Run("ValidKind", func(t *testing.T) {
			provider := eventstream.New(source.NewMockSourceClient())
			err := provider.LoadSpec("test.yaml", &specs.Spec{
				Kind: "event-stream-source",
				Spec: map[string]interface{}{
					"id":      "test-source",
					"name":    "Test Source",
					"type":    "javascript",
					"enabled": true,
				},
			})
			assert.NoError(t, err)
		})

		t.Run("InvalidSpec", func(t *testing.T) {
			provider := eventstream.New(source.NewMockSourceClient())
			err := provider.LoadSpec("test.yaml", &specs.Spec{
				Kind: "event-stream-source",
				Spec: map[string]interface{}{
					"id":      123, // should be string
					"enabled": "invalid",
				},
			})
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "'id' expected type 'string'")
		})
	})

	t.Run("Validate", func(t *testing.T) {
		testCases := []struct {
			externalGraphResources []*resources.Resource
			name                   string
			specs                  []*specs.Spec
			expectedError          bool
			errorMessage           string
		}{
			{
				name: "Valid sources",
				specs: []*specs.Spec{
					{
						Version: "rudder/v0.1",
						Kind:    "event-stream-source",
						Spec: map[string]interface{}{
							"id":      "test-source-1",
							"name":    "Test Source 1",
							"type":    "javascript",
							"enabled": true,
						},
					},
				},
				expectedError: false,
			},
			{
				name: "Invalid source - missing required fields",
				specs: []*specs.Spec{
					{
						Version: "rudder/v0.1",
						Kind:    "event-stream-source",
						Spec: map[string]interface{}{
							"id":      "test-source",
							"enabled": true,
						},
					},
				},
				expectedError: true,
				errorMessage:  "name is required",
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				provider := eventstream.New(source.NewMockSourceClient())

				// Load all specs
				for _, spec := range tc.specs {
					err := provider.LoadSpec("", spec)
					require.NoError(t, err, "LoadSpec should not fail")
				}

				err := provider.Validate(resources.NewGraph())

				if tc.expectedError {
					assert.Error(t, err)
					if tc.errorMessage != "" {
						assert.Contains(t, err.Error(), tc.errorMessage)
					}
				} else {
					assert.NoError(t, err)
				}
			})
		}
	})

	t.Run("GetResourceGraph", func(t *testing.T) {
		provider := eventstream.New(source.NewMockSourceClient())

		err := provider.LoadSpec("test1.yaml", &specs.Spec{
			Kind: "event-stream-source",
			Spec: map[string]interface{}{
				"id":      "test-source-1",
				"name":    "Test Source 1",
				"type":    "javascript",
				"enabled": true,
			},
		})
		require.NoError(t, err)

		err = provider.LoadSpec("test2.yaml", &specs.Spec{
			Kind: "event-stream-source",
			Spec: map[string]interface{}{
				"id":      "test-source-2",
				"name":    "Test Source 2",
				"type":    "python",
				"enabled": false,
			},
		})
		require.NoError(t, err)

		graph, err := provider.ResourceGraph()
		require.NoError(t, err)

		// Verify both resources are in the graph
		resources := graph.Resources()
		assert.Len(t, resources, 2)

		// Verify resource IDs
		resourceIDs := make([]string, 0, len(resources))
		for _, r := range resources {
			resourceIDs = append(resourceIDs, r.ID())
		}
		assert.Contains(t, resourceIDs, "test-source-1")
		assert.Contains(t, resourceIDs, "test-source-2")
	})

	t.Run("CRUD Operations", func(t *testing.T) {
		t.Run("Create", func(t *testing.T) {
			provider := eventstream.New(source.NewMockSourceClient())
			ctx := context.Background()

			createData := resources.ResourceData{
				"name":    "Test Source",
				"enabled": true,
				"type":    "javascript",
			}

			result, err := provider.Create(ctx, "test-source", source.ResourceType, createData)
			require.NoError(t, err)
			require.Equal(t, &resources.ResourceData{
				"id": "",
			}, result)
		})

		t.Run("Update", func(t *testing.T) {
			provider := eventstream.New(source.NewMockSourceClient())
			ctx := context.Background()

			updateData := resources.ResourceData{
				"name":    "Updated Source",
				"enabled": false,
			}

			stateData := resources.ResourceData{
				"id": "test-source-id",
			}

			result, err := provider.Update(ctx, "test-source", source.ResourceType, updateData, stateData)
			require.NoError(t, err)
			assert.Equal(t, &resources.ResourceData{
				"id": "test-source-id",
			}, result)
		})

		t.Run("Delete", func(t *testing.T) {
			provider := eventstream.New(source.NewMockSourceClient())
			ctx := context.Background()
			stateData := resources.ResourceData{
				"id": "test-source-id",
			}
			err := provider.Delete(ctx, "test-source", source.ResourceType, stateData)
			require.NoError(t, err)
		})
	})

	t.Run("Import", func(t *testing.T) {
		mockClient := source.NewMockSourceClient()
		mockClient.SetGetSourcesFunc(func(ctx context.Context) ([]sourceClient.EventStreamSource, error) {
			return []sourceClient.EventStreamSource{
				{
					ID:         "remote-123",
					ExternalID: "",
					Name:       "Existing Source",
					Type:       "javascript",
					Enabled:    true,
				},
			}, nil
		})
		provider := eventstream.New(mockClient)
		ctx := context.Background()

		data := resources.ResourceData{
			"name":    "Updated Source",
			"enabled": false,
			"type":    "javascript",
		}

		result, err := provider.Import(ctx, "test-source", source.ResourceType, data, "remote-123")
		require.NoError(t, err)
		assert.Equal(t, &resources.ResourceData{
			"id": "remote-123",
		}, result)
		assert.True(t, mockClient.GetSourcesCalled())
		assert.True(t, mockClient.UpdateCalled())
		assert.True(t, mockClient.SetExternalIDCalled())
	})

	t.Run("LoadResourcesFromRemote", func(t *testing.T) {
		mockClient := source.NewMockSourceClient()
		provider := eventstream.New(mockClient)

		ctx := context.Background()
		mockClient.SetGetSourcesFunc(func(ctx context.Context) ([]sourceClient.EventStreamSource, error) {
			return []sourceClient.EventStreamSource{
				{
					ID:         "remote123",
					ExternalID: "external-123",
					Name:       "Test Source 1",
					Type:       "javascript",
					Enabled:    true,
				},
				{
					ID:         "remote456",
					ExternalID: "external-456",
					Name:       "Test Source 2",
					Type:       "Python",
					Enabled:    false,
				},
			}, nil
		})

		collection, err := provider.LoadResourcesFromRemote(ctx)
		require.NoError(t, err)

		esResources := collection.GetAll(source.ResourceType)
		require.Len(t, esResources, 2)

		assert.Equal(t, map[string]*resources.RemoteResource{
			"remote123": {
				ID:         "remote123",
				ExternalID: "external-123",
				Data: sourceClient.EventStreamSource{
					ID:         "remote123",
					ExternalID: "external-123",
					Name:       "Test Source 1",
					Type:       "javascript",
					Enabled:    true,
				},
			},
			"remote456": {
				ID:         "remote456",
				ExternalID: "external-456",
				Data: sourceClient.EventStreamSource{
					ID:         "remote456",
					ExternalID: "external-456",
					Name:       "Test Source 2",
					Type:       "Python",
					Enabled:    false,
				},
			},
		}, esResources)
	})

	t.Run("MapRemoteToState", func(t *testing.T) {
		mockClient := source.NewMockSourceClient()
		provider := eventstream.New(mockClient)

		// Create a ResourceCollection with test data
		collection := resources.NewResourceCollection()
		resourceMap := map[string]*resources.RemoteResource{
			"remote123": {
				ID:         "remote123",
				ExternalID: "external-123",
				Data: sourceClient.EventStreamSource{
					ID:         "remote123",
					ExternalID: "external-123",
					Name:       "Test Source 1",
					Type:       "javascript",
					Enabled:    true,
				},
			},
			"remote456": {
				ID:         "remote456",
				ExternalID: "external-456",
				Data: sourceClient.EventStreamSource{
					ID:         "remote456",
					ExternalID: "external-456",
					Name:       "Test Source 2",
					Type:       "python",
					Enabled:    false,
				},
			},
		}
		collection.Set(source.ResourceType, resourceMap)

		loadedState, err := provider.MapRemoteToState(collection)
		require.NoError(t, err)

		assert.Len(t, loadedState.Resources, 2)

		// Check first resource
		assert.Equal(t, map[string]*state.ResourceState{
			"event-stream-source:external-123": {
				ID:   "external-123",
				Type: "event-stream-source",
				Input: resources.ResourceData{
					"name":    "Test Source 1",
					"enabled": true,
					"type":    "javascript",
				},
				Output: resources.ResourceData{
					"id": "remote123",
				},
			},
			"event-stream-source:external-456": {
				ID:   "external-456",
				Type: "event-stream-source",
				Input: resources.ResourceData{
					"name":    "Test Source 2",
					"enabled": false,
					"type":    "python",
				},
				Output: resources.ResourceData{
					"id": "remote456",
				},
			},
		}, loadedState.Resources)
	})

	t.Run("LoadImportable", func(t *testing.T) {
		mockClient := source.NewMockSourceClient()
		provider := eventstream.New(mockClient)
		ctx := context.Background()

		mockClient.SetGetSourcesFunc(func(ctx context.Context) ([]sourceClient.EventStreamSource, error) {
			return []sourceClient.EventStreamSource{
				{
					ID:      "remote456",
					Name:    "Test Source 2",
					Type:    "python",
					Enabled: false,
				},
				{
					ID:      "remote789",
					Name:    "Test Source 3",
					Type:    "javascript",
					Enabled: true,
				},
			}, nil
		})

		idNamer := &mockNamer{}

		collection, err := provider.LoadImportable(ctx, idNamer)
		require.NoError(t, err)

		esResources := collection.GetAll(source.ResourceType)
		assert.Len(t, esResources, 2)

		// Verify the returned resources
		assert.Equal(t, &resources.RemoteResource{
			ID:         "remote456",
			ExternalID: "test-source-2",
			Reference:  "#/event-stream-source/event-stream-source/test-source-2",
			Data: &sourceClient.EventStreamSource{
				ID:      "remote456",
				Name:    "Test Source 2",
				Type:    "python",
				Enabled: false,
			},
		}, esResources["remote456"])

		assert.Equal(t, &resources.RemoteResource{
			ID:         "remote789",
			ExternalID: "test-source-3",
			Reference:  "#/event-stream-source/event-stream-source/test-source-3",
			Data: &sourceClient.EventStreamSource{
				ID:      "remote789",
				Name:    "Test Source 3",
				Type:    "javascript",
				Enabled: true,
			},
		}, esResources["remote789"])
	})

	t.Run("FormatForExport", func(t *testing.T) {
		mockClient := source.NewMockSourceClient()
		provider := eventstream.New(mockClient)
		ctx := context.Background()

		collection := resources.NewResourceCollection()
		resourceMap := map[string]*resources.RemoteResource{
			"remote123": {
				ID:         "remote123",
				ExternalID: "test-source-1",
				Data: &sourceClient.EventStreamSource{
					ID:          "remote123",
					ExternalID:  "test-source-1",
					Name:        "Test Source 1",
					Type:        "javascript",
					Enabled:     true,
					WorkspaceID: "workspace-123",
				},
			},
			"remote456": {
				ID:         "remote456",
				ExternalID: "test-source-2",
				Data: &sourceClient.EventStreamSource{
					ID:          "remote456",
					ExternalID:  "test-source-2",
					Name:        "Test Source 2",
					Type:        "python",
					Enabled:     false,
					WorkspaceID: "workspace-123",
				},
			},
		}
		collection.Set(source.ResourceType, resourceMap)

		idNamer := &mockNamer{}
		resolver := &mockResolver{}

		entities, err := provider.FormatForExport(ctx, collection, idNamer, resolver)
		require.NoError(t, err)
		assert.Len(t, entities, 2)

		// Verify entities (order is not guaranteed in map iteration)
		entityMap := make(map[string]*specs.Spec)
		for _, entity := range entities {
			spec, ok := entity.Content.(*specs.Spec)
			require.True(t, ok)
			externalID := spec.Spec["id"].(string)
			entityMap[externalID] = spec
		}

		spec1 := entityMap["test-source-1"]
		require.NotNil(t, spec1)
		assert.Equal(t, "event-stream-source", spec1.Kind)
		assert.Equal(t, map[string]interface{}{
			"id":      "test-source-1",
			"name":    "Test Source 1",
			"enabled": true,
			"type":    "javascript",
		}, spec1.Spec)

		spec2 := entityMap["test-source-2"]
		require.NotNil(t, spec2)
		assert.Equal(t, "event-stream-source", spec2.Kind)
		assert.Equal(t, map[string]interface{}{
			"id":      "test-source-2",
			"name":    "Test Source 2",
			"enabled": false,
			"type":    "python",
		}, spec2.Spec)
	})
}

// mockNamer is a simple mock implementation of namer.Namer for testing
type mockNamer struct{}

func (m *mockNamer) Name(input namer.ScopeName) (string, error) {
	return strings.ToLower(strings.ReplaceAll(input.Name, " ", "-")), nil
}

func (m *mockNamer) Load(names []namer.ScopeName) error {
	return nil
}

// mockResolver is a simple mock implementation of resolver.ReferenceResolver for testing
type mockResolver struct{}

func (m *mockResolver) ResolveToReference(entityType string, remoteID string) (string, error) {
	return remoteID, nil
}
