package providers_test

import (
	"context"
	"encoding/json"
	"testing"

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

// helper function to parse JSON rules into map[string]interface{}
func parseRulesJSON(rulesJSON string) map[string]any {
	if rulesJSON == "" {
		return nil
	}

	var rules map[string]interface{}
	if err := json.Unmarshal([]byte(rulesJSON), &rules); err != nil {
		panic("failed to parse rules JSON: " + err.Error())
	}
	return rules
}

func constructTrackingPlanEventSchema(name, eventType, identitySection string, rules map[string]interface{}, defs map[string]interface{}) catalog.TrackingPlanEventSchema {
	return catalog.TrackingPlanEventSchema{
		Name:            name,
		EventType:       eventType,
		IdentitySection: identitySection,
		Rules: struct {
			Schema     string                 `json:"$schema"`
			Type       string                 `json:"type"`
			Properties map[string]interface{} `json:"properties"`
			Defs       map[string]interface{} `json:"$defs,omitempty"`
		}{
			Schema:     "https://json-schema.org/draft/2019-09/schema",
			Type:       "object",
			Properties: rules,
			Defs:       defs,
		},
	}
}

func constructTrackingPlanWithSchemas(id, name string, events []catalog.TrackingPlanEventSchema) *catalog.TrackingPlanWithSchemas {
	return &catalog.TrackingPlanWithSchemas{
		ID:           id,
		Name:         name,
		CreationType: "manual",
		Events:       events,
	}
}

func constructPropertySchema(name string, types []plan.PropertyType, required bool, config *plan.PropertyConfig, schema *plan.ObjectSchema, itemTypes []plan.PropertyType) plan.PropertySchema {
	return plan.PropertySchema{
		Property: plan.Property{
			Name:      name,
			Types:     types,
			Config:    config,
			ItemTypes: itemTypes,
		},
		Required: required,
		Schema:   schema,
	}
}

func constructCustomType(name string, baseType plan.PrimitiveType, itemType plan.PropertyType, config *plan.PropertyConfig, schema *plan.ObjectSchema) *plan.CustomType {
	return &plan.CustomType{
		Name:     name,
		Type:     baseType,
		ItemType: itemType,
		Config:   config,
		Schema:   schema,
	}
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
			mockResponse: constructTrackingPlanWithSchemas("tp_123", "Test Tracking Plan", []catalog.TrackingPlanEventSchema{
				constructTrackingPlanEventSchema("Test", "track", "properties", parseRulesJSON(`{
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
								"required": ["someInteger", "someString"]
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
								"type": ["boolean", "string"]
							},
							"someCustomString": {
								"$ref": "#/$defs/CustomString"
							},
							"someCustomObject": {
								"$ref": "#/$defs/CustomObject"
			        },
							"someCustomStringArray": {
								"$ref": "#/$defs/CustomStringArray"
							},
							"someCustomObjectArray": {
								"$ref": "#/$defs/CustomObjectArray"
							},
							"arrayWithMultipleTypes": {
								"type": ["array"],
								"items": {
									"type": ["string", "integer"]
								}
							},
							"propertyWithoutType": {
							},
							"multipleTypesProperty": {
								"type": ["string", "integer", "boolean"]
							}
						},
						"required": [],
						"unevaluatedProperties": false
					}
				}`), parseRulesJSON(`{
					"CustomString": {
						"type": "string",
						"enum": ["one", "two", "three"]
					},
					"CustomObject": {
						"type": ["object"],
						"properties": {
							"id": {
								"$ref": "#/$defs/CustomString"
							},
							"count": {
								"type": ["integer"]
							}
						},
						"required": ["id"]
					},
					"CustomStringArray": {
						"type": ["array"],
						"items": {
							"type": ["string"]
						}
					},
					"CustomObjectArray": {
						"type": ["array"],
						"items": {
							"$ref": "#/$defs/CustomObject"
						}
					}
				}`)),
				constructTrackingPlanEventSchema("", "identify", "traits", parseRulesJSON(`{
					"traits": {
						"type": "object",
						"properties": {
							"someString": {
								"type": ["string"]
							},
							"someInteger": {
								"type": ["integer"]
							}
						},
						"required": ["someString", "someInteger"],
						"unevaluatedProperties": false
					}
				}`), nil),
				constructTrackingPlanEventSchema("", "screen", "traits", parseRulesJSON(`{
					"traits": {
						"type": "object",
						"properties": {
							"someString": {
								"type": ["string"]
							}
						},
						"unevaluatedProperties": false
					}
				}`), nil),
				constructTrackingPlanEventSchema("", "page", "traits", parseRulesJSON(`{
					"traits": {
						"type": "object",
						"properties": {
							"someString": {
								"type": ["string"]
							}
						},
						"unevaluatedProperties": false
					}
				}`), nil),
				constructTrackingPlanEventSchema("", "group", "traits", parseRulesJSON(`{
					"traits": {
						"type": "object",
						"properties": {
							"someString": {
								"type": ["string"]
							}
						},
						"unevaluatedProperties": false
					}
				}`), nil),
			}),
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
								"someObject": constructPropertySchema("someObject", []plan.PropertyType{plan.PrimitiveTypeObject}, false, nil, &plan.ObjectSchema{
									Properties: map[string]plan.PropertySchema{
										"someInteger": constructPropertySchema("someInteger", []plan.PropertyType{plan.PrimitiveTypeInteger}, true, nil, nil, nil),
										"someString":  constructPropertySchema("someString", []plan.PropertyType{plan.PrimitiveTypeString}, true, nil, nil, nil),
										"someNumber":  constructPropertySchema("someNumber", []plan.PropertyType{plan.PrimitiveTypeNumber}, false, nil, nil, nil),
									},
								}, nil),
								"someInteger":        constructPropertySchema("someInteger", []plan.PropertyType{plan.PrimitiveTypeInteger}, false, nil, nil, nil),
								"someNumber":         constructPropertySchema("someNumber", []plan.PropertyType{plan.PrimitiveTypeNumber}, false, nil, nil, nil),
								"someString":         constructPropertySchema("someString", []plan.PropertyType{plan.PrimitiveTypeString}, false, nil, nil, nil),
								"someArrayOfStrings": constructPropertySchema("someArrayOfStrings", []plan.PropertyType{plan.PrimitiveTypeArray}, false, nil, nil, []plan.PropertyType{plan.PrimitiveTypeString}),
								"someStringWithEnums": constructPropertySchema("someStringWithEnums", []plan.PropertyType{plan.PrimitiveTypeString}, false, &plan.PropertyConfig{
									Enum: []any{"one", "two", "three"},
								}, nil, nil),
								"someStringOrBoolean": constructPropertySchema("someStringOrBoolean", []plan.PropertyType{plan.PrimitiveTypeBoolean, plan.PrimitiveTypeString}, false, nil, nil, nil),
								"someCustomString": constructPropertySchema("someCustomString", []plan.PropertyType{
									constructCustomType("CustomString", plan.PrimitiveTypeString, nil, &plan.PropertyConfig{
										Enum: []any{"one", "two", "three"},
									}, nil),
								}, false, nil, nil, nil),
								"someCustomObject": constructPropertySchema("someCustomObject", []plan.PropertyType{
									constructCustomType("CustomObject", plan.PrimitiveTypeObject, nil, nil, &plan.ObjectSchema{
										Properties: map[string]plan.PropertySchema{
											"id": constructPropertySchema("id", []plan.PropertyType{constructCustomType("CustomString", plan.PrimitiveTypeString, nil, &plan.PropertyConfig{
												Enum: []any{"one", "two", "three"},
											}, nil)}, true, nil, nil, nil),
											"count": constructPropertySchema("count", []plan.PropertyType{plan.PrimitiveTypeInteger}, false, nil, nil, nil),
										},
									}),
								}, false, nil, nil, nil),
								"someCustomStringArray": constructPropertySchema("someCustomStringArray", []plan.PropertyType{
									constructCustomType("CustomStringArray", plan.PrimitiveTypeArray, plan.PrimitiveTypeString, nil, nil),
								}, false, nil, nil, nil),
								"someCustomObjectArray": constructPropertySchema("someCustomObjectArray", []plan.PropertyType{
									constructCustomType("CustomObjectArray", plan.PrimitiveTypeArray, constructCustomType("CustomObject", plan.PrimitiveTypeObject, nil, nil, &plan.ObjectSchema{
										Properties: map[string]plan.PropertySchema{
											"id": constructPropertySchema("id", []plan.PropertyType{constructCustomType("CustomString", plan.PrimitiveTypeString, nil, &plan.PropertyConfig{
												Enum: []any{"one", "two", "three"},
											}, nil)}, true, nil, nil, nil),
											"count": constructPropertySchema("count", []plan.PropertyType{plan.PrimitiveTypeInteger}, false, nil, nil, nil),
										},
									}), nil, nil),
								}, false, nil, nil, nil),
								"arrayWithMultipleTypes": constructPropertySchema("arrayWithMultipleTypes", []plan.PropertyType{plan.PrimitiveTypeArray}, false, nil, nil, []plan.PropertyType{plan.PrimitiveTypeString, plan.PrimitiveTypeInteger}),
								"propertyWithoutType":    constructPropertySchema("propertyWithoutType", []plan.PropertyType{plan.PrimitiveTypeAny}, false, nil, nil, nil),
								"multipleTypesProperty":  constructPropertySchema("multipleTypesProperty", []plan.PropertyType{plan.PrimitiveTypeString, plan.PrimitiveTypeInteger, plan.PrimitiveTypeBoolean}, false, nil, nil, nil),
							},
						},
					},
					{
						Event: plan.Event{
							EventType: plan.EventTypeIdentify,
						},
						Section: plan.IdentitySectionTraits,
						Schema: plan.ObjectSchema{
							Properties: map[string]plan.PropertySchema{
								"someString":  constructPropertySchema("someString", []plan.PropertyType{plan.PrimitiveTypeString}, true, nil, nil, nil),
								"someInteger": constructPropertySchema("someInteger", []plan.PropertyType{plan.PrimitiveTypeInteger}, true, nil, nil, nil),
							},
						},
					},
					{
						Event: plan.Event{
							EventType: plan.EventTypeScreen,
						},
						Section: plan.IdentitySectionTraits,
						Schema: plan.ObjectSchema{
							Properties: map[string]plan.PropertySchema{
								"someString": constructPropertySchema("someString", []plan.PropertyType{plan.PrimitiveTypeString}, false, nil, nil, nil),
							},
						},
					},
					{
						Event: plan.Event{
							EventType: plan.EventTypePage,
						},
						Section: plan.IdentitySectionTraits,
						Schema: plan.ObjectSchema{
							Properties: map[string]plan.PropertySchema{
								"someString": constructPropertySchema("someString", []plan.PropertyType{plan.PrimitiveTypeString}, false, nil, nil, nil),
							},
						},
					},
					{
						Event: plan.Event{
							EventType: plan.EventTypeGroup,
						},
						Section: plan.IdentitySectionTraits,
						Schema: plan.ObjectSchema{
							Properties: map[string]plan.PropertySchema{
								"someString": constructPropertySchema("someString", []plan.PropertyType{plan.PrimitiveTypeString}, false, nil, nil, nil),
							},
						},
					},
				},
			},
			expectedError: false,
		},
		{
			name: "empty object properties",
			mockResponse: constructTrackingPlanWithSchemas("tp_empty", "Empty Properties Plan", []catalog.TrackingPlanEventSchema{
				constructTrackingPlanEventSchema("EmptyEvent", "track", "properties", parseRulesJSON(`{
					"properties": {
						"type": "object",
						"properties": {}
					}
				}`), nil),
			}),
			expectedPlan: &plan.TrackingPlan{
				Name: "Empty Properties Plan",
				Rules: []plan.EventRule{
					{
						Event: plan.Event{
							Name:        "EmptyEvent",
							Description: "",
							EventType:   plan.EventTypeTrack,
						},
						Section: plan.IdentitySectionProperties,
						Schema: plan.ObjectSchema{
							Properties: map[string]plan.PropertySchema{},
						},
					},
				},
			},
			expectedError: false,
		},
		{
			name: "object without properties field",
			mockResponse: constructTrackingPlanWithSchemas("tp_no_props", "No Properties Plan", []catalog.TrackingPlanEventSchema{
				constructTrackingPlanEventSchema("NoPropsEvent", "track", "properties", parseRulesJSON(`{
					"properties": {
						"type": "object"
					}
				}`), nil),
			}),
			expectedPlan: &plan.TrackingPlan{
				Name: "No Properties Plan",
				Rules: []plan.EventRule{
					{
						Event: plan.Event{
							Name:        "NoPropsEvent",
							Description: "",
							EventType:   plan.EventTypeTrack,
						},
						Section: plan.IdentitySectionProperties,
						Schema: plan.ObjectSchema{
							Properties: map[string]plan.PropertySchema{},
						},
					},
				},
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

				// Compare metadata if expected
				if tt.expectedPlan.Metadata.TrackingPlanID != "" {
					assert.Equal(t, tt.expectedPlan.Metadata.TrackingPlanID, result.Metadata.TrackingPlanID)
					assert.Equal(t, tt.expectedPlan.Metadata.TrackingPlanVersion, result.Metadata.TrackingPlanVersion)
				}

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
					assert.Len(t, actualRule.Schema.Properties, len(expectedRule.Schema.Properties), "Properties length should match for event %s", expectedRule.Event.Name)

					for propName, expectedProp := range expectedRule.Schema.Properties {
						actualProp, exists := actualRule.Schema.Properties[propName]
						assert.True(t, exists, "Property %s should exist", propName)

						assert.Equal(t, expectedProp.Property.Name, actualProp.Property.Name, "Property names should match for property %s", propName)
						assert.Equal(t, expectedProp.Required, actualProp.Required, "Required should match for property %s", propName)

						// Compare all types in the Types slice
						assert.Equal(t, len(expectedProp.Property.Types), len(actualProp.Property.Types), "Types length should match for property %s", propName)
						for j, expectedType := range expectedProp.Property.Types {
							assert.Equal(t, expectedType, actualProp.Property.Types[j], "Type at index %d should match for property %s", j, propName)
						}

						// Compare all types in the ItemTypes slice
						assert.Equal(t, len(expectedProp.Property.ItemTypes), len(actualProp.Property.ItemTypes), "ItemTypes length should match for property %s", propName)
						for j, expectedItemType := range expectedProp.Property.ItemTypes {
							assert.Equal(t, expectedItemType, actualProp.Property.ItemTypes[j], "ItemType at index %d should match for property %s", j, propName)
						}

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
						Defs       map[string]interface{} `json:"$defs,omitempty"`
					}{
						Schema:     "https://json-schema.org/draft/2019-09/schema",
						Type:       "object",
						Properties: parseRulesJSON(rulesJSON),
						Defs:       nil,
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

func TestJSONSchemaPlanProvider_Variants(t *testing.T) {
	tests := []struct {
		name         string
		mockResponse *catalog.TrackingPlanWithSchemas
		expectedPlan *plan.TrackingPlan
	}{
		{
			name: "custom type with variants",
			mockResponse: constructTrackingPlanWithSchemas("tp_variants", "Variants Test Plan", []catalog.TrackingPlanEventSchema{
				constructTrackingPlanEventSchema("PageView", "track", "properties", parseRulesJSON(`{
					"properties": {
						"type": "object",
						"properties": {
							"pageContext": {
								"$ref": "#/$defs/PageContext"
							}
						},
						"required": []
					}
				}`), parseRulesJSON(`{
					"PageContext": {
						"type": "object",
						"properties": {
							"pageType": {
								"type": "string"
							}
						},
						"required": ["pageType"],
						"allOf": [
							{
								"type": "object",
								"allOf": [
									{
										"if": {
											"properties": {
												"pageType": {
													"enum": ["search"]
												}
											},
											"required": ["pageType"]
										},
										"then": {
											"properties": {
												"query": {
													"type": "string"
												}
											},
											"required": ["query"]
										}
									},
									{
										"if": {
											"properties": {
												"pageType": {
													"enum": ["product"]
												}
											},
											"required": ["pageType"]
										},
										"then": {
											"properties": {
												"productId": {
													"type": "string"
												}
											},
											"required": ["productId"]
										}
									},
									{
										"if": {
											"properties": {
												"pageType": {
													"not": {
														"enum": ["search", "product"]
													}
												}
											},
											"required": ["pageType"]
										},
										"then": {
											"properties": {
												"pageData": {
													"type": "object"
												}
											},
											"required": []
										}
									}
								]
							}
						]
					}
				}`)),
			}),
			expectedPlan: &plan.TrackingPlan{
				Name: "Variants Test Plan",
				Rules: []plan.EventRule{
					{
						Event: plan.Event{
							Name:        "PageView",
							Description: "",
							EventType:   plan.EventTypeTrack,
						},
						Section: plan.IdentitySectionProperties,
						Schema: plan.ObjectSchema{
							Properties: map[string]plan.PropertySchema{
								"pageContext": constructPropertySchema("pageContext", []plan.PropertyType{
									constructCustomType("PageContext", plan.PrimitiveTypeObject, nil, nil, &plan.ObjectSchema{
										Properties: map[string]plan.PropertySchema{
											"pageType": constructPropertySchema("pageType", []plan.PropertyType{plan.PrimitiveTypeString}, true, nil, nil, nil),
										},
									}),
								}, false, nil, nil, nil),
							},
						},
					},
				},
			},
		},
		{
			name: "event rule with variants",
			mockResponse: constructTrackingPlanWithSchemas("tp_event_variants", "Event Variants Plan", []catalog.TrackingPlanEventSchema{
				constructTrackingPlanEventSchema("VariantEvent", "track", "properties", parseRulesJSON(`{
					"properties": {
						"type": "object",
						"properties": {
							"eventType": {
								"type": "string"
							},
							"baseProperty": {
								"type": "string"
							}
						},
						"required": ["eventType"],
						"allOf": [
							{
								"type": "object",
								"allOf": [
									{
										"if": {
											"properties": {
												"eventType": {
													"enum": ["click"]
												}
											},
											"required": ["eventType"]
										},
										"then": {
											"properties": {
												"elementId": {
													"type": "string"
												}
											},
											"required": ["elementId"]
										}
									},
									{
										"if": {
											"properties": {
												"eventType": {
													"enum": ["scroll"]
												}
											},
											"required": ["eventType"]
										},
										"then": {
											"properties": {
												"scrollDepth": {
													"type": "number"
												}
											},
											"required": ["scrollDepth"]
										}
									}
								]
							}
						]
					}
				}`), nil),
			}),
			expectedPlan: &plan.TrackingPlan{
				Name: "Event Variants Plan",
				Rules: []plan.EventRule{
					{
						Event: plan.Event{
							Name:        "VariantEvent",
							Description: "",
							EventType:   plan.EventTypeTrack,
						},
						Section: plan.IdentitySectionProperties,
						Schema: plan.ObjectSchema{
							Properties: map[string]plan.PropertySchema{
								"eventType":    constructPropertySchema("eventType", []plan.PropertyType{plan.PrimitiveTypeString}, true, nil, nil, nil),
								"baseProperty": constructPropertySchema("baseProperty", []plan.PropertyType{plan.PrimitiveTypeString}, false, nil, nil, nil),
							},
						},
						Variants: []plan.Variant{
							{
								Type:          "discriminator",
								Discriminator: "eventType",
								Cases: []plan.VariantCase{
									{
										Match: []any{"click"},
										Schema: plan.ObjectSchema{
											Properties: map[string]plan.PropertySchema{
												"elementId": constructPropertySchema("elementId", []plan.PropertyType{plan.PrimitiveTypeString}, true, nil, nil, nil),
											},
										},
									},
									{
										Match: []any{"scroll"},
										Schema: plan.ObjectSchema{
											Properties: map[string]plan.PropertySchema{
												"scrollDepth": constructPropertySchema("scrollDepth", []plan.PropertyType{plan.PrimitiveTypeNumber}, true, nil, nil, nil),
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
		{
			name: "variant with multiple match values",
			mockResponse: constructTrackingPlanWithSchemas("tp_multi_match", "Multi Match Plan", []catalog.TrackingPlanEventSchema{
				constructTrackingPlanEventSchema("MultiMatch", "track", "properties", parseRulesJSON(`{
					"properties": {
						"type": "object",
						"properties": {
							"status": {
								"type": "string"
							}
						},
						"required": ["status"],
						"allOf": [
							{
								"type": "object",
								"allOf": [
									{
										"if": {
											"properties": {
												"status": {
													"enum": ["active", "enabled", "running"]
												}
											},
											"required": ["status"]
										},
										"then": {
											"properties": {
												"uptime": {
													"type": "number"
												}
											},
											"required": ["uptime"]
										}
									}
								]
							}
						]
					}
				}`), nil),
			}),
			expectedPlan: &plan.TrackingPlan{
				Name: "Multi Match Plan",
				Rules: []plan.EventRule{
					{
						Event: plan.Event{
							Name:        "MultiMatch",
							Description: "",
							EventType:   plan.EventTypeTrack,
						},
						Section: plan.IdentitySectionProperties,
						Schema: plan.ObjectSchema{
							Properties: map[string]plan.PropertySchema{
								"status": constructPropertySchema("status", []plan.PropertyType{plan.PrimitiveTypeString}, true, nil, nil, nil),
							},
						},
						Variants: []plan.Variant{
							{
								Type:          "discriminator",
								Discriminator: "status",
								Cases: []plan.VariantCase{
									{
										Match: []any{"active", "enabled", "running"},
										Schema: plan.ObjectSchema{
											Properties: map[string]plan.PropertySchema{
												"uptime": constructPropertySchema("uptime", []plan.PropertyType{plan.PrimitiveTypeNumber}, true, nil, nil, nil),
											},
										},
									},
								},
							},
						},
					},
				},
			},
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
				assert.Equal(t, expectedRule.Event.EventType, actualRule.Event.EventType)

				// Compare section
				assert.Equal(t, expectedRule.Section, actualRule.Section)

				// Compare schema properties
				assert.Len(t, actualRule.Schema.Properties, len(expectedRule.Schema.Properties))

				// Compare variants if present
				if len(expectedRule.Variants) > 0 {
					assert.Len(t, actualRule.Variants, len(expectedRule.Variants), "Variants length should match")

					for vi, expectedVariant := range expectedRule.Variants {
						actualVariant := actualRule.Variants[vi]

						assert.Equal(t, expectedVariant.Type, actualVariant.Type, "Variant type should match")
						assert.Equal(t, expectedVariant.Discriminator, actualVariant.Discriminator, "Discriminator should match")
						assert.Len(t, actualVariant.Cases, len(expectedVariant.Cases), "Cases length should match")

						for ci, expectedCase := range expectedVariant.Cases {
							actualCase := actualVariant.Cases[ci]

							assert.ElementsMatch(t, expectedCase.Match, actualCase.Match, "Match values should match")
							assert.Len(t, actualCase.Schema.Properties, len(expectedCase.Schema.Properties), "Case schema properties should match")
						}

						// Check default schema if present
						if expectedVariant.DefaultSchema != nil {
							require.NotNil(t, actualVariant.DefaultSchema, "Default schema should be present")
							assert.Len(t, actualVariant.DefaultSchema.Properties, len(expectedVariant.DefaultSchema.Properties))
						}
					}
				}
			}
		})
	}
}
