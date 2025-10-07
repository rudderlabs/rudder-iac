package eventstream_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	sourceClient "github.com/rudderlabs/rudder-iac/api/client/event-stream/source"
	"github.com/rudderlabs/rudder-iac/cli/internal/project/specs"
	eventstream "github.com/rudderlabs/rudder-iac/cli/internal/providers/event-stream"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/event-stream/source"
	"github.com/rudderlabs/rudder-iac/cli/internal/syncer/resources"
	"github.com/rudderlabs/rudder-iac/cli/internal/syncer/state"
)

func TestProvider(t *testing.T) {
	t.Run("GetSupportedKinds", func(t *testing.T) {
		provider := eventstream.New(source.NewMockSourceClient())
		kinds := provider.GetSupportedKinds()
		assert.Contains(t, kinds, "event-stream-source")
		assert.Len(t, kinds, 1)
	})

	t.Run("GetSupportedTypes", func(t *testing.T) {
		provider := eventstream.New(source.NewMockSourceClient())
		types := provider.GetSupportedTypes()
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
					"id":                "test-source",
					"name":              "Test Source",
					"type": "javascript",
					"enabled":           true,
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
			name          string
			specs         []*specs.Spec
			expectedError bool
			errorMessage  string
		}{
			{
				name: "Valid sources",
				specs: []*specs.Spec{
					{
						Version: "rudder/v0.1",
						Kind:    "event-stream-source",
						Spec: map[string]interface{}{
							"id":                "test-source-1",
							"name":              "Test Source 1",
							"type": "javascript",
							"enabled":           true,
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

				// Validate all specs
				err := provider.Validate()
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
				"id":                "test-source-1",
				"name":              "Test Source 1",
				"type": "javascript",
				"enabled":           true,
			},
		})
		require.NoError(t, err)

		err = provider.LoadSpec("test2.yaml", &specs.Spec{
			Kind: "event-stream-source",
			Spec: map[string]interface{}{
				"id":                "test-source-2",
				"name":              "Test Source 2",
				"type": "python",
				"enabled":           false,
			},
		})
		require.NoError(t, err)

		graph, err := provider.GetResourceGraph()
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

	t.Run("LoadState", func(t *testing.T) {
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
					Type:       "python",
					Enabled:    false,
				},
			}, nil
		})

		s, err := provider.LoadState(ctx)
		require.NoError(t, err)

		// Check that both resources are loaded
		assert.Len(t, s.Resources, 2)

		// Check first resource
		urn1 := resources.URN("external-123", source.ResourceType)
		rs1 := s.GetResource(urn1)
		require.NotNil(t, rs1)
		assert.Equal(t, &state.ResourceState{
			ID:   "external-123",
			Type: source.ResourceType,
			Input: resources.ResourceData{
				"name":              "Test Source 1",
				"enabled":           true,
				"type": "javascript",
			},
			Output: resources.ResourceData{
				"id": "remote123",
			},
		}, rs1)

		rs2 := s.GetResource("event-stream-source:external-456")
		assert.Equal(t, &state.ResourceState{
			ID:   "external-456",
			Type: source.ResourceType,
			Input: resources.ResourceData{
				"name":              "Test Source 2",
				"enabled":           false,
				"type": "python",
			},
			Output: resources.ResourceData{
				"id": "remote456",
			},
		}, rs2)
	})

	t.Run("CRUD Operations", func(t *testing.T) {
		t.Run("Create", func(t *testing.T) {
			provider := eventstream.New(source.NewMockSourceClient())
			ctx := context.Background()

			createData := resources.ResourceData{
				"name":              "Test Source",
				"enabled":           true,
				"type": "javascript",
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
		provider := eventstream.New(source.NewMockSourceClient())
		ctx := context.Background()
		_, err := provider.Import(ctx, "test-source", source.ResourceType, nil, "workspace-123", "remote-123")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "importing event stream source is not supported")
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

	t.Run("LoadStateFromResources", func(t *testing.T) {
		mockClient := source.NewMockSourceClient()
		provider := eventstream.New(mockClient)

		ctx := context.Background()

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

		loadedState, err := provider.LoadStateFromResources(ctx, collection)
		require.NoError(t, err)

		assert.Len(t, loadedState.Resources, 2)

		// Check first resource
		assert.Equal(t, map[string]*state.ResourceState{
			"event-stream-source:external-123": {
				ID:   "external-123",
				Type: "event-stream-source",
				Input: resources.ResourceData{
					"name":              "Test Source 1",
					"enabled":           true,
					"type": "javascript",
				},
				Output: resources.ResourceData{
					"id": "remote123",
				},
			},
			"event-stream-source:external-456": {
				ID:   "external-456",
				Type: "event-stream-source",
				Input: resources.ResourceData{
					"name":              "Test Source 2",
					"enabled":           false,
					"type": "python",
				},
				Output: resources.ResourceData{
					"id": "remote456",
				},
			},
		}, loadedState.Resources)
	})
}
