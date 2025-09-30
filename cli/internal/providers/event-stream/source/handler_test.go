package source_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	sourceClient "github.com/rudderlabs/rudder-iac/api/client/event-stream/source"
	"github.com/rudderlabs/rudder-iac/cli/internal/project/specs"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/event-stream/source"
	"github.com/rudderlabs/rudder-iac/cli/internal/syncer/resources"
	"github.com/rudderlabs/rudder-iac/cli/internal/syncer/state"
)

func TestEventStreamSourceHandler(t *testing.T) {
	t.Run("LoadSpec", func(t *testing.T) {
		t.Run("error with invalid type", func(t *testing.T) {
			mockClient := source.NewMockSourceClient()
			handler := source.NewHandler(mockClient)

			spec := &specs.Spec{
				Version: "rudder/v0.1",
				Kind:    "event-stream-source",
				Spec: map[string]interface{}{
					"id":                123,
					"name":              "Test Source",
					"type": "javascript",
					"enabled":           true,
				},
			}
			err := handler.LoadSpec("", spec)
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "'id' expected type 'string'")
		})

		t.Run("error with duplicate id", func(t *testing.T) {
			mockClient := source.NewMockSourceClient()
			handler := source.NewHandler(mockClient)

			spec := &specs.Spec{
				Version: "rudder/v0.1",
				Kind:    "event-stream-source",
				Spec: map[string]interface{}{
					"id":                "test-source",
					"name":              "Test Source",
					"type": "javascript",
					"enabled":           true,
				},
			}

			err := handler.LoadSpec("", spec)
			require.NoError(t, err)

			err = handler.LoadSpec("", spec)
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "event stream source with id test-source already exists")
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
					{
						Version: "rudder/v0.1",
						Kind:    "event-stream-source",
						Spec: map[string]interface{}{
							"id":                "test-source-2",
							"name":              "Test Source 2",
							"type": "python",
							"enabled":           false,
						},
					},
				},
				expectedError: false,
			},
			{
				name: "Missing id",
				specs: []*specs.Spec{
					{
						Version: "rudder/v0.1",
						Kind:    "event-stream-source",
						Spec: map[string]interface{}{
							"name":              "Test Source",
							"type": "javascript",
							"enabled":           true,
						},
					},
				},
				expectedError: true,
				errorMessage:  "id is required",
			},
			{
				name: "Missing name",
				specs: []*specs.Spec{
					{
						Version: "rudder/v0.1",
						Kind:    "event-stream-source",
						Spec: map[string]interface{}{
							"id":                "test-source",
							"type": "javascript",
							"enabled":           true,
						},
					},
				},
				expectedError: true,
				errorMessage:  "name is required",
			},
			{
				name: "Missing type",
				specs: []*specs.Spec{
					{
						Version: "rudder/v0.1",
						Kind:    "event-stream-source",
						Spec: map[string]interface{}{
							"id":      "test-source",
							"name":    "Test Source",
							"enabled": true,
						},
					},
				},
				expectedError: true,
				errorMessage:  "type is required",
			},
			{
				name: "Invalid type",
				specs: []*specs.Spec{
					{
						Version: "rudder/v0.1",
						Kind:    "event-stream-source",
						Spec: map[string]interface{}{
							"id":                "test-source",
							"name":              "Test Source",
							"type": "InvalidSDK",
							"enabled":           true,
						},
					},
				},
				expectedError: true,
				errorMessage:  "type 'InvalidSDK' is invalid",
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				mockClient := source.NewMockSourceClient()
				handler := source.NewHandler(mockClient)

				for _, spec := range tc.specs {
					err := handler.LoadSpec("", spec)
					require.NoError(t, err)
				}

				err := handler.Validate()

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

	t.Run("GetResources", func(t *testing.T) {
		mockClient := source.NewMockSourceClient()
		handler := source.NewHandler(mockClient)

		spec := &specs.Spec{
			Version: "rudder/v0.1",
			Kind:    "event-stream-source",
			Spec: map[string]interface{}{
				"id":                "test-source-1",
				"name":              "Test Source 1",
				"type": "javascript",
				"enabled":           true,
			},
		}

		err := handler.LoadSpec("", spec)
		require.NoError(t, err)

		res, err := handler.GetResources()
		assert.NoError(t, err)
		assert.Len(t, res, 1)

		assert.Equal(t, "test-source-1", res[0].ID())
		assert.Equal(t, source.ResourceType, res[0].Type())
		assert.Equal(t, resources.ResourceData{
			"name":              "Test Source 1",
			"enabled":           true,
			"type": "javascript",
		}, res[0].Data())
		assert.Empty(t, res[0].Dependencies())
	})

	t.Run("Create", func(t *testing.T) {
		mockClient := source.NewMockSourceClient()
		handler := source.NewHandler(mockClient)

		data := resources.ResourceData{
			"name":              "Test Source",
			"enabled":           true,
			"type": "javascript",
		}

		result, err := handler.Create(context.Background(), "test-source", data)

		assert.NoError(t, err)
		assert.True(t, mockClient.CreateCalled())
		assert.Equal(t, &resources.ResourceData{
			"name":              "Test Source",
			"enabled":           true,
			"type": "javascript",
		}, result)
	})

	t.Run("Update", func(t *testing.T) {
		t.Run("Success", func(t *testing.T) {
			mockClient := source.NewMockSourceClient()
			handler := source.NewHandler(mockClient)

			data := resources.ResourceData{
				"name":              "Updated Source",
				"enabled":           false,
				"type": "javascript",
			}
			state := resources.ResourceData{
				"id":                "remote123",
				"name":              "Original Source",
				"enabled":           true,
				"type": "javascript",
			}

			result, err := handler.Update(context.Background(), "test-source", data, state)

			assert.NoError(t, err)
			assert.True(t, mockClient.UpdateCalled())
			assert.Equal(t, &resources.ResourceData{
				"name":              "Updated Source",
				"enabled":           false,
				"type": "javascript",
			}, result)
		})

		t.Run("Source definition cannot be changed", func(t *testing.T) {
			mockClient := source.NewMockSourceClient()
			handler := source.NewHandler(mockClient)

			data := resources.ResourceData{
				"type": "python",
			}
			state := resources.ResourceData{
				"id":                "remote123",
				"name":              "Original Source",
				"enabled":           true,
				"type": "javascript",
			}

			_, err := handler.Update(context.Background(), "test-source", data, state)

			assert.Error(t, err)
			assert.Contains(t, err.Error(), "type cannot be changed")
			assert.False(t, mockClient.UpdateCalled())
		})
	})

	t.Run("Delete", func(t *testing.T) {
		mockClient := source.NewMockSourceClient()
		handler := source.NewHandler(mockClient)

		state := resources.ResourceData{
			"id": "remote123",
		}

		err := handler.Delete(context.Background(), "test-source", state)

		assert.NoError(t, err)
		assert.True(t, mockClient.DeleteCalled())
	})

	t.Run("LoadState", func(t *testing.T) {
		mockClient := source.NewMockSourceClient()
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
				{
					ID:         "remote789",
					ExternalID: "", // This should be skipped
					Name:       "Test Source 3",
					Type:       "go",
					Enabled:    true,
				},
			}, nil
		})
		handler := source.NewHandler(mockClient)

		st, err := handler.LoadState(context.Background())

		assert.NoError(t, err)
		assert.True(t, mockClient.GetSourcesCalled())

		// Should have 2 resources (one with empty ExternalID is skipped)
		assert.Len(t, st.Resources, 2)

		resource1 := st.GetResource("event-stream-source:external-123")
		assert.Equal(t, &state.ResourceState{
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
		}, resource1)

		resource2 := st.GetResource("event-stream-source:external-456")
		assert.Equal(t, &state.ResourceState{
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
		}, resource2)
	})

	t.Run("LoadResourcesFromRemote", func(t *testing.T) {
		mockClient := source.NewMockSourceClient()
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
		handler := source.NewHandler(mockClient)

		collection, err := handler.LoadResourcesFromRemote(context.Background())

		assert.NoError(t, err)
		assert.True(t, mockClient.GetSourcesCalled())

		esResources := collection.GetAll(source.ResourceType)
		assert.Len(t, esResources, 2)

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
					Type:       "python",
					Enabled:    false,
				},
			},
		}, esResources)
	})

	t.Run("LoadStateFromResources", func(t *testing.T) {
		handler := source.NewHandler(nil)

		// Create a resource collection with event stream sources
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
					ExternalID: "",
					Name:       "Test Source 2",
					Type:       "python",
					Enabled:    false,
				},
			},
		}
		collection.Set(source.ResourceType, resourceMap)

		st, err := handler.LoadStateFromResources(context.Background(), collection)

		assert.NoError(t, err)
		assert.NotNil(t, st)

		assert.Len(t, st.Resources, 1)
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
		}, st.Resources)
	})
}
