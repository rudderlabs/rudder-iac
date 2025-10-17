package source_test

import (
	"context"
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	sourceClient "github.com/rudderlabs/rudder-iac/api/client/event-stream/source"

	"github.com/rudderlabs/rudder-iac/cli/internal/project/specs"
	dcstate "github.com/rudderlabs/rudder-iac/cli/internal/providers/datacatalog/state"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/event-stream/source"
	"github.com/rudderlabs/rudder-iac/cli/internal/syncer/resources"
	"github.com/rudderlabs/rudder-iac/cli/internal/syncer/state"
)

// Helper function to convert boolean to pointer
func boolPtr(b bool) *bool {
	return &b
}

func TestEventStreamSourceHandler(t *testing.T) {
	t.Run("LoadSpec", func(t *testing.T) {
		testCases := []struct {
			name         string
			spec         *specs.Spec
			loadTwice    bool
			errorMessage string
		}{
			{
				name: "error with invalid data type",
				spec: &specs.Spec{
					Version: "rudder/v0.1",
					Kind:    "event-stream-source",
					Spec: map[string]interface{}{
						"id":      123,
						"name":    "Test Source",
						"type":    "javascript",
						"enabled": true,
					},
				},
				errorMessage: "'id' expected type 'string'",
			},
			{
				name:      "error with duplicate id",
				loadTwice: true,
				spec: &specs.Spec{
					Version: "rudder/v0.1",
					Kind:    "event-stream-source",
					Spec: map[string]interface{}{
						"id":      "test-source",
						"name":    "Test Source",
						"type":    "javascript",
						"enabled": true,
					},
				},
				errorMessage: "event stream source with id test-source already exists",
			},
			{
				name: "with invalid tracking plan ref format",
				spec: &specs.Spec{
					Version: "rudder/v0.1",
					Kind:    "event-stream-source",
					Spec: map[string]interface{}{
						"id":      "test-source",
						"name":    "Test Source",
						"type":    "javascript",
						"enabled": true,
						"governance": map[string]interface{}{
							"validations": map[string]interface{}{
								"tracking_plan": "invalid-ref",
								"config": map[string]interface{}{
									"track": map[string]interface{}{
										"propagate_violations": true,
									},
								},
							},
						},
					},
				},
				errorMessage: "invalid ref format: invalid-ref",
			},
			{
				name: "with invalid tracking plan entity type",
				spec: &specs.Spec{
					Version: "rudder/v0.1",
					Kind:    "event-stream-source",
					Spec: map[string]interface{}{
						"id":      "test-source",
						"name":    "Test Source",
						"type":    "javascript",
						"enabled": true,
						"governance": map[string]interface{}{
							"validations": map[string]interface{}{
								"tracking_plan": "#/invalid-entity/group/tp-123",
								"config": map[string]interface{}{
									"track": map[string]interface{}{
										"propagate_violations": true,
									},
								},
							},
						},
					},
				},
				errorMessage: "invalid entity type: invalid-entity",
			},
			{
				name: "with tracking plan config missing",
				spec: &specs.Spec{
					Version: "rudder/v0.1",
					Kind:    "event-stream-source",
					Spec: map[string]interface{}{
						"id":      "test-source",
						"name":    "Test Source",
						"type":    "javascript",
						"enabled": true,
						"governance": map[string]interface{}{
							"validations": map[string]interface{}{
								"tracking_plan": "#/tracking-plans/group/tp-123",
							},
						},
					},
				},
				errorMessage: "governance.validations.config is required",
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				enableStatelessCLI(t)
				mockClient := source.NewMockSourceClient()
				handler := source.NewHandler(mockClient)
				if tc.loadTwice {
					err := handler.LoadSpec("", tc.spec)
					require.NoError(t, err)
				}
				err := handler.LoadSpec("", tc.spec)
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tc.errorMessage)
			})
		}
	})

	t.Run("Validate", func(t *testing.T) {
		testCases := []struct {
			name                   string
			externalGraphResources []*resources.Resource
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
					{
						Version: "rudder/v0.1",
						Kind:    "event-stream-source",
						Spec: map[string]interface{}{
							"id":      "test-source-2",
							"name":    "Test Source 2",
							"type":    "python",
							"enabled": false,
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
							"name":    "Test Source",
							"type":    "javascript",
							"enabled": true,
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
							"id":      "test-source",
							"type":    "javascript",
							"enabled": true,
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
							"id":      "test-source",
							"name":    "Test Source",
							"type":    "InvalidSDK",
							"enabled": true,
						},
					},
				},
				expectedError: true,
				errorMessage:  "type 'InvalidSDK' is invalid",
			},
			{
				name: "Validates existence of tracking plan resource reference",
				externalGraphResources: []*resources.Resource{
					resources.NewResource("tp-123", dcstate.TrackingPlanResourceType, resources.ResourceData{}, nil),
				},
				specs: []*specs.Spec{
					{
						Version: "rudder/v0.1",
						Kind:    "event-stream-source",
						Spec: map[string]interface{}{
							"id":   "test-source-2",
							"name": "Test Source 2",
							"type": "ios",
							"governance": map[string]interface{}{
								"validations": map[string]interface{}{
									"tracking_plan": "#/tracking-plans/some-name/tp-123",
									"config":        map[string]interface{}{},
								},
							},
						},
					},
				},
				expectedError: false,
			},
			{
				name: "Invalid tracking plan reference",
				specs: []*specs.Spec{
					{
						Version: "rudder/v0.1",
						Kind:    "event-stream-source",
						Spec: map[string]interface{}{
							"id":   "test-source-3",
							"name": "Test Source 3",
							"type": "ios",
							"governance": map[string]interface{}{
								"validations": map[string]interface{}{
									"tracking_plan": "#/tracking-plans/some-name/non-existent-tp",
									"config":        map[string]interface{}{},
								},
							},
						},
					},
				},
				expectedError: true,
				errorMessage:  "validating event stream source spec: tracking plan with URN 'tracking-plan:non-existent-tp' not found in the project",
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				enableStatelessCLI(t)
				mockClient := source.NewMockSourceClient()
				handler := source.NewHandler(mockClient)

				for _, spec := range tc.specs {
					err := handler.LoadSpec("", spec)
					require.NoError(t, err)
				}

				// Add external resources to the graph
				graph := resources.NewGraph()
				for _, res := range tc.externalGraphResources {
					graph.AddResource(res)
				}
				err := handler.Validate(graph)

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
		enableStatelessCLI(t)
		mockClient := source.NewMockSourceClient()
		handler := source.NewHandler(mockClient)

		specs := []*specs.Spec{
			{
				Version: "rudder/v0.1",
				Kind:    "event-stream-source",
				Spec: map[string]interface{}{
					"id":   "test-source-1",
					"name": "Test Source 1",
					"type": "javascript",
				},
			},
			{
				Version: "rudder/v0.1",
				Kind:    "event-stream-source",
				Spec: map[string]interface{}{
					"id":      "test-source-2",
					"name":    "Test Source 2",
					"type":    "python",
					"enabled": false,
					"governance": map[string]interface{}{
						"validations": map[string]interface{}{
							"tracking_plan": "#/tracking-plans/group/tp-123",
							"config": map[string]interface{}{
								"track": map[string]interface{}{
									"propagate_violations":      true,
									"drop_unplanned_events":     false,
									"drop_unplanned_properties": false,
									"drop_other_violations":     true,
								},
								"identify": map[string]interface{}{
									"propagate_violations":  false,
									"drop_other_violations": false,
								},
								"group": map[string]interface{}{
									"drop_unplanned_properties": true,
								},
							},
						},
					},
				},
			},
		}

		for _, spec := range specs {
			err := handler.LoadSpec("", spec)
			require.NoError(t, err)
		}

		res, err := handler.GetResources()
		assert.NoError(t, err)
		assert.Len(t, res, 2)

		// Create a map for order-agnostic assertion
		resourceMap := make(map[string]*resources.Resource)
		for _, r := range res {
			resourceMap[r.ID()] = r
		}

		// Assert test-source-1
		source1, exists := resourceMap["test-source-1"]
		require.True(t, exists, "test-source-1 should exist in resources")
		assert.Equal(t, resources.ResourceData{
			"name":    "Test Source 1",
			"enabled": true,
			"type":    "javascript",
		}, source1.Data())

		// Assert test-source-2
		source2, exists := resourceMap["test-source-2"]
		require.True(t, exists, "test-source-2 should exist in resources")
		assert.Equal(t, resources.ResourceData{
			"name":    "Test Source 2",
			"enabled": false,
			"type":    "python",
			"tracking_plan": &resources.PropertyRef{
				URN:      resources.URN("tp-123", dcstate.TrackingPlanResourceType),
				Property: "id",
			},
			"tracking_plan_config": map[string]interface{}{
				"track": map[string]interface{}{
					"propagate_violations":      true,
					"drop_unplanned_events":     false,
					"drop_unplanned_properties": false,
					"drop_other_violations":     true,
				},
				"identify": map[string]interface{}{
					"propagate_violations":  false,
					"drop_other_violations": false,
				},
				"group": map[string]interface{}{
					"drop_unplanned_properties": true,
				},
			},
		}, source2.Data())
	})

	t.Run("Create", func(t *testing.T) {
		testCases := []struct {
			name                 string
			data                 resources.ResourceData
			expectedLinkTPCalled bool
		}{
			{
				name: "without tracking plan",
				data: resources.ResourceData{
					"name":    "Test Source",
					"enabled": true,
					"type":    "javascript",
				},
				expectedLinkTPCalled: false,
			},
			{
				name: "with tracking plan",
				data: resources.ResourceData{
					"name":          "Test Source",
					"enabled":       true,
					"type":          "javascript",
					"tracking_plan": "tp-123",
					"tracking_plan_config": map[string]interface{}{
						"track": &source.TrackConfigResource{
							EventConfigResource: &source.EventConfigResource{
								PropagateViolations: boolPtr(true),
								DropOtherViolations: boolPtr(false),
							},
							DropUnplannedEvents: boolPtr(true),
						},
						"page": &source.EventConfigResource{
							PropagateViolations: boolPtr(false),
							DropOtherViolations: boolPtr(true),
						},
					},
				},
				expectedLinkTPCalled: true,
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				mockClient := source.NewMockSourceClient()
				handler := source.NewHandler(mockClient)

				result, err := handler.Create(context.Background(), "test-source", tc.data)

				assert.NoError(t, err)
				assert.True(t, mockClient.CreateCalled())
				assert.Equal(t, tc.expectedLinkTPCalled, mockClient.LinkTPCalled())
				if tc.expectedLinkTPCalled {
					assert.Equal(t, &resources.ResourceData{
						"id":               "",
						"tracking_plan_id": "tp-123",
					}, result)
				} else {
					assert.Equal(t, &resources.ResourceData{
						"id": "",
					}, result)
				}
			})
		}
	})

	t.Run("Update", func(t *testing.T) {
		tpConfig := map[string]interface{}{
			"track": &source.TrackConfigResource{
				EventConfigResource: &source.EventConfigResource{
					PropagateViolations: boolPtr(true),
					DropOtherViolations: boolPtr(false),
				},
			},
		}
		testCases := []struct {
			name                   string
			data                   resources.ResourceData
			state                  resources.ResourceData
			expectedUpdateCalled   bool
			expectedLinkTPCalled   bool
			expectedUnlinkTPCalled bool
			expectedUpdateTPCalled bool
			expectedError          bool
			errorMessage           string
			expectedResult         *resources.ResourceData
		}{
			{
				name: "Source definition cannot be changed",
				data: resources.ResourceData{
					"type": "python",
				},
				state: resources.ResourceData{
					"id":      "remote123",
					"name":    "Original Source",
					"enabled": true,
					"type":    "javascript",
				},
				expectedUpdateCalled:   false,
				expectedLinkTPCalled:   false,
				expectedUnlinkTPCalled: false,
				expectedUpdateTPCalled: false,
				expectedError:          true,
				errorMessage:           "type cannot be changed",
			},
			{
				name: "without tracking plan",
				data: resources.ResourceData{
					"name":    "Updated Source",
					"enabled": false,
					"type":    "javascript",
				},
				state: resources.ResourceData{
					"id":      "remote123",
					"name":    "Original Source",
					"enabled": true,
					"type":    "javascript",
				},
				expectedUpdateCalled:   true,
				expectedLinkTPCalled:   false,
				expectedUnlinkTPCalled: false,
				expectedUpdateTPCalled: false,
				expectedError:          false,
				expectedResult: &resources.ResourceData{
					"id": "remote123",
				},
			},
			{
				name: "no tracking plan changes",
				data: resources.ResourceData{
					"name":                 "Updated Source",
					"enabled":              false,
					"type":                 "javascript",
					"tracking_plan":        "tp-123",
					"tracking_plan_config": tpConfig,
				},
				state: resources.ResourceData{
					"id":                   "remote123",
					"name":                 "Original Source",
					"enabled":              true,
					"type":                 "javascript",
					"tracking_plan_id":     "tp-123",
					"tracking_plan_config": tpConfig,
				},
				expectedUpdateCalled:   true,
				expectedLinkTPCalled:   false,
				expectedUnlinkTPCalled: false,
				expectedUpdateTPCalled: false,
				expectedError:          false,
				expectedResult: &resources.ResourceData{
					"id":               "remote123",
					"tracking_plan_id": "tp-123",
				},
			},
			{
				name: "same tracking plan with different config",
				data: resources.ResourceData{
					"name":          "Original Source",
					"enabled":       true,
					"type":          "javascript",
					"tracking_plan": "tp-123",
					"tracking_plan_config": map[string]interface{}{
						"track": &source.TrackConfigResource{
							EventConfigResource: &source.EventConfigResource{
								PropagateViolations: boolPtr(false),
								DropOtherViolations: boolPtr(false),
							},
						},
					},
				},
				state: resources.ResourceData{
					"id":                   "remote123",
					"name":                 "Original Source",
					"enabled":              true,
					"type":                 "javascript",
					"tracking_plan_id":     "tp-123",
					"tracking_plan_config": tpConfig,
				},
				expectedUpdateCalled:   false,
				expectedLinkTPCalled:   false,
				expectedUnlinkTPCalled: false,
				expectedUpdateTPCalled: true,
				expectedError:          false,
				expectedResult: &resources.ResourceData{
					"id":               "remote123",
					"tracking_plan_id": "tp-123",
				},
			},
			{
				name: "change tracking plan",
				data: resources.ResourceData{
					"name":                 "Original Source",
					"enabled":              true,
					"type":                 "javascript",
					"tracking_plan":        "tp-456",
					"tracking_plan_config": tpConfig,
				},
				state: resources.ResourceData{
					"id":                   "remote123",
					"name":                 "Original Source",
					"enabled":              true,
					"type":                 "javascript",
					"tracking_plan_id":     "tp-123",
					"tracking_plan_config": tpConfig,
				},
				expectedUpdateCalled:   false,
				expectedLinkTPCalled:   true,
				expectedUnlinkTPCalled: true,
				expectedUpdateTPCalled: false,
				expectedError:          false,
				expectedResult: &resources.ResourceData{
					"id":               "remote123",
					"tracking_plan_id": "tp-456",
				},
			},
			{
				name: "link tracking plan",
				data: resources.ResourceData{
					"name":                 "Updated Source",
					"enabled":              false,
					"type":                 "javascript",
					"tracking_plan":        "tp-123",
					"tracking_plan_config": tpConfig,
				},
				state: resources.ResourceData{
					"id":      "remote123",
					"name":    "Original Source",
					"enabled": true,
					"type":    "javascript",
				},
				expectedUpdateCalled:   true,
				expectedLinkTPCalled:   true,
				expectedUnlinkTPCalled: false,
				expectedUpdateTPCalled: false,
				expectedError:          false,
				expectedResult: &resources.ResourceData{
					"id":               "remote123",
					"tracking_plan_id": "tp-123",
				},
			},
			{
				name: "unlink tracking plan",
				data: resources.ResourceData{
					"name":    "Updated Source",
					"enabled": false,
					"type":    "javascript",
				},
				state: resources.ResourceData{
					"id":                   "remote123",
					"name":                 "Original Source",
					"enabled":              true,
					"type":                 "javascript",
					"tracking_plan_id":     "tp-123",
					"tracking_plan_config": tpConfig,
				},
				expectedUpdateCalled:   true,
				expectedLinkTPCalled:   false,
				expectedUnlinkTPCalled: true,
				expectedUpdateTPCalled: false,
				expectedError:          false,
				expectedResult: &resources.ResourceData{
					"id": "remote123",
				},
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				mockClient := source.NewMockSourceClient()
				handler := source.NewHandler(mockClient)

				result, err := handler.Update(context.Background(), "test-source", tc.data, tc.state)

				if tc.expectedError {
					assert.Error(t, err)
					assert.Contains(t, err.Error(), tc.errorMessage)
				} else {
					assert.NoError(t, err)
					assert.Equal(t, tc.expectedResult, result)
				}
				assert.Equal(t, tc.expectedUpdateCalled, mockClient.UpdateCalled())
				assert.Equal(t, tc.expectedLinkTPCalled, mockClient.LinkTPCalled())
				assert.Equal(t, tc.expectedUnlinkTPCalled, mockClient.UnlinkTPCalled())
				assert.Equal(t, tc.expectedUpdateTPCalled, mockClient.UpdateTPConnectionCalled())
			})
		}
	})

	t.Run("Delete", func(t *testing.T) {
		testCases := []struct {
			name                   string
			state                  resources.ResourceData
			expectedUnlinkTPCalled bool
		}{
			{
				name: "without tracking plan",
				state: resources.ResourceData{
					"id": "remote123",
				},
				expectedUnlinkTPCalled: false,
			},
			{
				name: "with tracking plan",
				state: resources.ResourceData{
					"id":               "remote123",
					"tracking_plan_id": "tp-123",
				},
				expectedUnlinkTPCalled: true,
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				mockClient := source.NewMockSourceClient()
				handler := source.NewHandler(mockClient)

				err := handler.Delete(context.Background(), "test-source", tc.state)

				assert.NoError(t, err)
				assert.True(t, mockClient.DeleteCalled())
				assert.Equal(t, tc.expectedUnlinkTPCalled, mockClient.UnlinkTPCalled())
			})
		}
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
					TrackingPlan: &sourceClient.TrackingPlan{
						ID: "remote-tp-123",
						Config: &sourceClient.TrackingPlanConfig{
							Track: &sourceClient.TrackConfig{
								DropUnplannedEvents: boolPtr(true),
							},
							Identify: &sourceClient.EventTypeConfig{
								PropagateViolations:     boolPtr(false),
								DropUnplannedProperties: boolPtr(true),
								DropOtherViolations:     boolPtr(false),
							},
							Group: &sourceClient.EventTypeConfig{
								PropagateViolations:     boolPtr(true),
								DropUnplannedProperties: boolPtr(false),
								DropOtherViolations:     boolPtr(false),
							},
							Page: &sourceClient.EventTypeConfig{
								PropagateViolations:     boolPtr(false),
								DropUnplannedProperties: boolPtr(false),
								DropOtherViolations:     boolPtr(true),
							},
							Screen: &sourceClient.EventTypeConfig{
								PropagateViolations:     boolPtr(true),
								DropUnplannedProperties: boolPtr(false),
								DropOtherViolations:     boolPtr(false),
							},
						},
					},
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
					Type:       "Go",
					Enabled:    true,
				},
			}, nil
		})
		handler := source.NewHandler(mockClient)

		st, err := handler.LoadState(context.Background())

		assert.NoError(t, err)
		assert.True(t, mockClient.GetSourcesCalled())

		// Assert that we have exactly 2 resources (external-456 and external-123, external-789 is skipped due to empty ExternalID)
		assert.Len(t, st.Resources, 2)

		// Assert external-123 resource
		resource123, exists := st.Resources["event-stream-source:external-123"]
		require.True(t, exists, "event-stream-source:external-123 should exist in resources")
		assert.Equal(t, &state.ResourceState{
			ID:   "external-123",
			Type: "event-stream-source",
			Input: resources.ResourceData{
				"name":    "Test Source 1",
				"enabled": true,
				"type":    "javascript",
				"tracking_plan": &resources.PropertyRef{
					URN:      "",
					Property: "id",
				},
				"tracking_plan_config": map[string]interface{}{
					"track": map[string]interface{}{
						"drop_unplanned_events": true,
					},
					"identify": map[string]interface{}{
						"propagate_violations":      false,
						"drop_unplanned_properties": true,
						"drop_other_violations":     false,
					},
					"group": map[string]interface{}{
						"propagate_violations":      true,
						"drop_unplanned_properties": false,
						"drop_other_violations":     false,
					},
					"page": map[string]interface{}{
						"propagate_violations":      false,
						"drop_unplanned_properties": false,
						"drop_other_violations":     true,
					},
					"screen": map[string]interface{}{
						"propagate_violations":      true,
						"drop_unplanned_properties": false,
						"drop_other_violations":     false,
					},
				},
			},
			Output: resources.ResourceData{
				"id":               "remote123",
				"tracking_plan_id": "remote-tp-123",
			},
		}, resource123)

		// Assert external-456 resource
		resource456, exists := st.Resources["event-stream-source:external-456"]
		require.True(t, exists, "event-stream-source:external-456 should exist in resources")
		assert.Equal(t, &state.ResourceState{
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
		}, resource456)
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

		// Assert remote123 resource
		resource123, exists := esResources["remote123"]
		require.True(t, exists, "remote123 should exist in resources")
		assert.Equal(t, &resources.RemoteResource{
			ID:         "remote123",
			ExternalID: "external-123",
			Data: sourceClient.EventStreamSource{
				ID:         "remote123",
				ExternalID: "external-123",
				Name:       "Test Source 1",
				Type:       "javascript",
				Enabled:    true,
			},
		}, resource123)

		// Assert remote456 resource
		resource456, exists := esResources["remote456"]
		require.True(t, exists, "remote456 should exist in resources")
		assert.Equal(t, &resources.RemoteResource{
			ID:         "remote456",
			ExternalID: "external-456",
			Data: sourceClient.EventStreamSource{
				ID:         "remote456",
				ExternalID: "external-456",
				Name:       "Test Source 2",
				Type:       "python",
				Enabled:    false,
			},
		}, resource456)
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
			"remote789": {
				ID:         "remote789",
				ExternalID: "external-789",
				Data: sourceClient.EventStreamSource{
					ID:         "remote789",
					ExternalID: "external-789",
					Name:       "Test Source 3",
					Type:       "javascript",
					Enabled:    true,
					TrackingPlan: &sourceClient.TrackingPlan{
						ID: "remote-tp-789",
						Config: &sourceClient.TrackingPlanConfig{
							Track: &sourceClient.TrackConfig{
								DropUnplannedEvents: boolPtr(true),
							},
						},
					},
				},
			},
		}
		collection.Set(source.ResourceType, resourceMap)

		// Add tracking plan resources to the collection so they can be resolved
		trackingPlanResourceMap := map[string]*resources.RemoteResource{
			"remote-tp-789": {
				ID:         "remote-tp-789",
				ExternalID: "external-tp-789",
			},
		}
		collection.Set(dcstate.TrackingPlanResourceType, trackingPlanResourceMap)

		st, err := handler.LoadStateFromResources(context.Background(), collection)

		assert.NoError(t, err)
		assert.NotNil(t, st)

		assert.Len(t, st.Resources, 2)

		// Assert external-123 resource
		resource123, exists := st.Resources["event-stream-source:external-123"]
		require.True(t, exists, "event-stream-source:external-123 should exist in resources")
		assert.Equal(t, &state.ResourceState{
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
		}, resource123)

		// Assert external-789 resource
		resource789, exists := st.Resources["event-stream-source:external-789"]
		require.True(t, exists, "event-stream-source:external-789 should exist in resources")
		assert.Equal(t, &state.ResourceState{
			ID:   "external-789",
			Type: "event-stream-source",
			Input: resources.ResourceData{
				"name":    "Test Source 3",
				"enabled": true,
				"type":    "javascript",
				"tracking_plan": &resources.PropertyRef{
					URN:      resources.URN("external-tp-789", dcstate.TrackingPlanResourceType),
					Property: "id",
				},
				"tracking_plan_config": map[string]interface{}{
					"track": map[string]interface{}{
						"drop_unplanned_events": true,
					},
				},
			},
			Output: resources.ResourceData{
				"id":               "remote789",
				"tracking_plan_id": "remote-tp-789",
			},
		}, resource789)
	})
}

func enableStatelessCLI(t *testing.T) {
	viper.Set("experimental", true)
	viper.Set("flags.statelessCLI", true)

	t.Cleanup(func() {
		viper.Set("experimental", false)
		viper.Set("flags.statelessCLI", false)
	})
}
