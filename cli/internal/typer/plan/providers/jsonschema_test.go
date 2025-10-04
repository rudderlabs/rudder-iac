package providers_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/rudderlabs/rudder-iac/api/client/catalog"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/datacatalog"
	"github.com/rudderlabs/rudder-iac/cli/internal/typer/plan"
	"github.com/rudderlabs/rudder-iac/cli/internal/typer/plan/providers"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockTrackingPlanStore implements catalog.TrackingPlanStore for testing
type mockTrackingPlanStore struct {
	*datacatalog.EmptyCatalog
	trackingPlanWithSchemas *catalog.TrackingPlanWithSchemas
	err                     error
}

func (m *mockTrackingPlanStore) GetTrackingPlanWithSchemas(ctx context.Context, id string) (*catalog.TrackingPlanWithSchemas, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.trackingPlanWithSchemas, nil
}

// Helper function to parse JSON rules into map[string]interface{}
func parseRulesJSON(rulesJSON string) map[string]interface{} {
	var rules map[string]interface{}
	if err := json.Unmarshal([]byte(rulesJSON), &rules); err != nil {
		panic("failed to parse rules JSON: " + err.Error())
	}
	return rules
}

func TestJSONSchemaPlanProvider_GetTrackingPlan(t *testing.T) {
	tests := []struct {
		name             string
		mockResponse     *catalog.TrackingPlanWithSchemas
		expectedPlan     *plan.TrackingPlan
		expectedError    bool
		expectedErrorMsg string
	}{
		{
			name: "successful parsing with all property types",
			mockResponse: &catalog.TrackingPlanWithSchemas{
				ID:           "tp_123",
				Name:         "Test Tracking Plan",
				CreationType: "manual",
				Version:      1,
				WorkspaceID:  "1ZFgGc4ZM7S0vJq3MPLHkMhJZxK",
				CreatedAt:    time.Now(),
				UpdatedAt:    time.Now(),
				Events: []catalog.TrackingPlanEventSchema{
					{
						ID:              "ev_3338E4WgCctF2oUyL5gnJnqntjl",
						Name:            "Test",
						Description:     "",
						EventType:       "track",
						CategoryId:      nil,
						WorkspaceId:     "1ZFgGc4ZM7S0vJq3MPLHkMhJZxK",
						CreatedBy:       "1ZFgEE5fDiU9nioWcvYFCuunWpf",
						UpdatedBy:       nil,
						CreatedAt:       time.Now(),
						UpdatedAt:       time.Now(),
						IdentitySection: "properties",
						Rules: struct {
							Schema     string                 `json:"$schema"`
							Type       string                 `json:"type"`
							Properties map[string]interface{} `json:"properties"`
						}{
							Schema: "https://json-schema.org/draft/2019-09/schema",
							Type:   "object",
							Properties: parseRulesJSON(`{
								"properties": {
									"type": "object",
									"properties": {
										"someObject": {
											"type": ["object"],
											"additionalProperties": true,
											"properties": {
												"someInteger": {
													"type": ["integer"]
												},
												"someNumber": {
													"type": ["number"]
												},
												"someString": {
													"type": ["string"]
												}
											},
											"required": []
										},
										"someInteger": {
											"type": ["integer"]
										},
										"someNumber": {
											"type": ["number"]
										},
										"someString": {
											"type": ["string"]
										},
										"someArrayOfStrings": {
											"type": ["array"],
											"items": {
												"type": ["string"]
											}
										},
										"someStringWithEnums": {
											"type": ["string"],
											"enum": ["one", "two", "three"]
										},
										"someStringOrBoolean": {
											"type": ["boolean", "string"],
											"pattern": "^\\d{4}-[01]\\d-[0-3]\\dT[0-2](?:\\d:[0-5]){2}\\d(?:\\.\\d+)?Z?$"
										}
									},
									"required": [],
									"unevaluatedProperties": false
								}
							}`),
						},
					},
				},
			},
			expectedPlan: &plan.TrackingPlan{
				Name: "Test Tracking Plan",
				Rules: []plan.EventRule{
					{
						Event: plan.Event{
							Name:        "Test",
							Description: "",
							EventType:   plan.EventTypeTrack,
						},
						Section: plan.IdentitySectionProperties,
						Schema: plan.ObjectSchema{
							Properties: map[string]plan.PropertySchema{
								"someObject": {
									Property: plan.Property{
										Name:  "someObject",
										Types: []plan.PropertyType{plan.PrimitiveTypeObject},
									},
									Required: false,
									Schema: &plan.ObjectSchema{
										Properties: map[string]plan.PropertySchema{
											"someInteger": {
												Property: plan.Property{
													Name:  "someInteger",
													Types: []plan.PropertyType{plan.PrimitiveTypeInteger},
												},
												Required: false,
											},
											"someNumber": {
												Property: plan.Property{
													Name:  "someNumber",
													Types: []plan.PropertyType{plan.PrimitiveTypeNumber},
												},
												Required: false,
											},
											"someString": {
												Property: plan.Property{
													Name:  "someString",
													Types: []plan.PropertyType{plan.PrimitiveTypeString},
												},
												Required: false,
											},
										},
									},
								},
								"someInteger": {
									Property: plan.Property{
										Name:  "someInteger",
										Types: []plan.PropertyType{plan.PrimitiveTypeInteger},
									},
									Required: false,
								},
								"someNumber": {
									Property: plan.Property{
										Name:  "someNumber",
										Types: []plan.PropertyType{plan.PrimitiveTypeNumber},
									},
									Required: false,
								},
								"someString": {
									Property: plan.Property{
										Name:  "someString",
										Types: []plan.PropertyType{plan.PrimitiveTypeString},
									},
									Required: false,
								},
								"someArrayOfStrings": {
									Property: plan.Property{
										Name:  "someArrayOfStrings",
										Types: []plan.PropertyType{plan.PrimitiveTypeArray},
									},
									Required: false,
								},
								"someStringWithEnums": {
									Property: plan.Property{
										Name:  "someStringWithEnums",
										Types: []plan.PropertyType{plan.PrimitiveTypeString},
										Config: &plan.PropertyConfig{
											Enum: []any{"one", "two", "three"},
										},
									},
									Required: false,
								},
								"someStringOrBoolean": {
									Property: plan.Property{
										Name:  "someStringOrBoolean",
										Types: []plan.PropertyType{plan.PrimitiveTypeBoolean},
									},
									Required: false,
								},
							},
						},
					},
				},
			},
			expectedError: false,
		},
		{
			name: "required properties",
			mockResponse: &catalog.TrackingPlanWithSchemas{
				ID:           "tp_456",
				Name:         "Required Props Plan",
				CreationType: "manual",
				Version:      1,
				WorkspaceID:  "ws_123",
				CreatedAt:    time.Now(),
				UpdatedAt:    time.Now(),
				Events: []catalog.TrackingPlanEventSchema{
					{
						ID:              "ev_456",
						Name:            "User Signup",
						Description:     "User registration event",
						EventType:       "track",
						IdentitySection: "properties",
						Rules: struct {
							Schema     string                 `json:"$schema"`
							Type       string                 `json:"type"`
							Properties map[string]interface{} `json:"properties"`
						}{
							Schema: "https://json-schema.org/draft/2019-09/schema",
							Type:   "object",
							Properties: parseRulesJSON(`{
								"properties": {
									"type": "object",
									"properties": {
										"email": {
											"type": ["string"]
										},
										"username": {
											"type": ["string"]
										}
									},
									"required": ["email", "username"]
								}
							}`),
						},
					},
				},
			},
			expectedPlan: &plan.TrackingPlan{
				Name: "Required Props Plan",
				Rules: []plan.EventRule{
					{
						Event: plan.Event{
							Name:        "User Signup",
							Description: "User registration event",
							EventType:   plan.EventTypeTrack,
						},
						Section: plan.IdentitySectionProperties,
						Schema: plan.ObjectSchema{
							Properties: map[string]plan.PropertySchema{
								"email": {
									Property: plan.Property{
										Name:  "email",
										Types: []plan.PropertyType{plan.PrimitiveTypeString},
									},
									Required: true,
								},
								"username": {
									Property: plan.Property{
										Name:  "username",
										Types: []plan.PropertyType{plan.PrimitiveTypeString},
									},
									Required: true,
								},
							},
						},
					},
				},
			},
			expectedError: false,
		},
		{
			name: "identify event with traits section",
			mockResponse: &catalog.TrackingPlanWithSchemas{
				ID:           "tp_789",
				Name:         "Identify Plan",
				CreationType: "manual",
				Version:      1,
				WorkspaceID:  "ws_123",
				CreatedAt:    time.Now(),
				UpdatedAt:    time.Now(),
				Events: []catalog.TrackingPlanEventSchema{
					{
						ID:              "ev_789",
						Name:            "User Identified",
						Description:     "User identify event",
						EventType:       "identify",
						IdentitySection: "traits",
						Rules: struct {
							Schema     string                 `json:"$schema"`
							Type       string                 `json:"type"`
							Properties map[string]interface{} `json:"properties"`
						}{
							Schema: "https://json-schema.org/draft/2019-09/schema",
							Type:   "object",
							Properties: parseRulesJSON(`{
								"traits": {
									"type": "object",
									"properties": {
										"userId": {
											"type": ["string"]
										}
									},
									"required": ["userId"]
								}
							}`),
						},
					},
				},
			},
			expectedPlan: &plan.TrackingPlan{
				Name: "Identify Plan",
				Rules: []plan.EventRule{
					{
						Event: plan.Event{
							Name:        "User Identified",
							Description: "User identify event",
							EventType:   plan.EventTypeIdentify,
						},
						Section: plan.IdentitySectionTraits,
						Schema: plan.ObjectSchema{
							Properties: map[string]plan.PropertySchema{
								"userId": {
									Property: plan.Property{
										Name:  "userId",
										Types: []plan.PropertyType{plan.PrimitiveTypeString},
									},
									Required: true,
								},
							},
						},
					},
				},
			},
			expectedError: false,
		},
		{
			name: "empty tracking plan",
			mockResponse: &catalog.TrackingPlanWithSchemas{
				ID:           "tp_empty",
				Name:         "Empty Plan",
				CreationType: "manual",
				Version:      1,
				WorkspaceID:  "ws_123",
				CreatedAt:    time.Now(),
				UpdatedAt:    time.Now(),
				Events:       []catalog.TrackingPlanEventSchema{},
			},
			expectedPlan: &plan.TrackingPlan{
				Name:  "Empty Plan",
				Rules: []plan.EventRule{},
			},
			expectedError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := &mockTrackingPlanStore{
				trackingPlanWithSchemas: tt.mockResponse,
				err:                     nil,
			}

			provider := providers.NewJSONSchemaPlanProvider("test-plan-id", mockClient)
			result, err := provider.GetTrackingPlan(context.Background())

			if tt.expectedError {
				require.Error(t, err)
				if tt.expectedErrorMsg != "" {
					assert.Contains(t, err.Error(), tt.expectedErrorMsg)
				}
			} else {
				require.NoError(t, err)
				require.NotNil(t, result)

				// Compare tracking plan name
				assert.Equal(t, tt.expectedPlan.Name, result.Name)

				// Compare number of rules
				assert.Len(t, result.Rules, len(tt.expectedPlan.Rules))

				// Compare each rule
				for i, expectedRule := range tt.expectedPlan.Rules {
					actualRule := result.Rules[i]

					// Compare event details
					assert.Equal(t, expectedRule.Event.Name, actualRule.Event.Name)
					assert.Equal(t, expectedRule.Event.Description, actualRule.Event.Description)
					assert.Equal(t, expectedRule.Event.EventType, actualRule.Event.EventType)

					// Compare section
					assert.Equal(t, expectedRule.Section, actualRule.Section)

					// Compare schema properties
					assert.Len(t, actualRule.Schema.Properties, len(expectedRule.Schema.Properties))

					for propName, expectedProp := range expectedRule.Schema.Properties {
						actualProp, exists := actualRule.Schema.Properties[propName]
						assert.True(t, exists, "Property %s should exist", propName)

						assert.Equal(t, expectedProp.Property.Name, actualProp.Property.Name)
						assert.Equal(t, expectedProp.Required, actualProp.Required)
						assert.Equal(t, len(expectedProp.Property.Types), len(actualProp.Property.Types))

						// Compare enum config if present
						if expectedProp.Property.Config != nil && expectedProp.Property.Config.Enum != nil {
							require.NotNil(t, actualProp.Property.Config)
							assert.ElementsMatch(t, expectedProp.Property.Config.Enum, actualProp.Property.Config.Enum)
						}

						// Compare nested schema if present
						if expectedProp.Schema != nil {
							require.NotNil(t, actualProp.Schema, "Nested schema should exist for %s", propName)
							assert.Len(t, actualProp.Schema.Properties, len(expectedProp.Schema.Properties))
						}
					}
				}
			}
		})
	}
}

func TestJSONSchemaPlanProvider_ErrorCases(t *testing.T) {
	mockInvalidRespone := func(eventType, identitySection, rulesJSON string) *catalog.TrackingPlanWithSchemas {
		return &catalog.TrackingPlanWithSchemas{
			ID:   "tp_123",
			Name: "Test Plan",
			Events: []catalog.TrackingPlanEventSchema{
				{
					ID:              "ev_123",
					Name:            "Test Event",
					EventType:       eventType,
					IdentitySection: identitySection,
					Rules: struct {
						Schema     string                 `json:"$schema"`
						Type       string                 `json:"type"`
						Properties map[string]interface{} `json:"properties"`
					}{
						Schema:     "https://json-schema.org/draft/2019-09/schema",
						Type:       "object",
						Properties: parseRulesJSON(rulesJSON),
					},
				},
			},
		}
	}

	tests := []struct {
		name             string
		mockResponse     *catalog.TrackingPlanWithSchemas
		expectedErrorMsg string
	}{
		{
			name: "invalid event type",
			mockResponse: mockInvalidRespone("invalid_type", "properties", `{
				"properties": {
					"type": "object",
					"properties": {},
					"required": []
				}
			}`),
			expectedErrorMsg: "invalid event type",
		},
		{
			name:             "missing identity section in properties",
			mockResponse:     mockInvalidRespone("track", "properties", `{}`),
			expectedErrorMsg: "identity section 'properties' not found in properties",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := &mockTrackingPlanStore{
				trackingPlanWithSchemas: tt.mockResponse,
				err:                     nil,
			}

			provider := providers.NewJSONSchemaPlanProvider("test-plan-id", mockClient)
			_, err := provider.GetTrackingPlan(context.Background())

			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.expectedErrorMsg)
		})
	}
}
