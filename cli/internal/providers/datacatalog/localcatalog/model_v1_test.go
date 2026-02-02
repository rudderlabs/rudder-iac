package localcatalog

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestConvertConfigKeysToSnakeCase(t *testing.T) {
	t.Parallel()

	t.Run("converts all camelCase keys", func(t *testing.T) {
		t.Parallel()

		config := map[string]interface{}{
			"minLength":        5,
			"maxLength":        50,
			"multipleOf":       3,
			"itemTypes":        []interface{}{"string", "integer"},
			"enum":             []interface{}{"value1", "value2"},
			"minimum":          0,
			"maximum":          100,
			"pattern":          "^[a-z]+$",
			"exclusiveMinimum": 1,
			"exclusiveMaximum": 99,
		}

		result := convertConfigKeysToSnakeCase(config)

		assert.Equal(t, 5, result["min_length"])
		assert.Equal(t, 50, result["max_length"])
		assert.Equal(t, 3, result["multiple_of"])
		assert.Equal(t, []interface{}{"string", "integer"}, result["item_types"])
		assert.Equal(t, []interface{}{"value1", "value2"}, result["enum"])
		assert.Equal(t, 0, result["minimum"])
		assert.Equal(t, 100, result["maximum"])
		assert.Equal(t, "^[a-z]+$", result["pattern"])
		assert.Equal(t, 1, result["exclusive_minimum"])
		assert.Equal(t, 99, result["exclusive_maximum"])

		// Verify camelCase keys don't exist
		assert.NotContains(t, result, "minLength")
		assert.NotContains(t, result, "maxLength")
		assert.NotContains(t, result, "multipleOf")
		assert.NotContains(t, result, "itemTypes")
		assert.NotContains(t, result, "exclusiveMinimum")
		assert.NotContains(t, result, "exclusiveMaximum")
	})

	t.Run("returns nil for nil config", func(t *testing.T) {
		t.Parallel()

		result := convertConfigKeysToSnakeCase(nil)
		assert.Nil(t, result)
	})

	t.Run("handles empty config", func(t *testing.T) {
		t.Parallel()

		config := map[string]interface{}{}
		result := convertConfigKeysToSnakeCase(config)
		assert.NotNil(t, result)
		assert.Empty(t, result)
	})

	t.Run("preserves already snake_case keys", func(t *testing.T) {
		t.Parallel()

		config := map[string]interface{}{
			"min_length": 10,
			"max_length": 20,
		}

		result := convertConfigKeysToSnakeCase(config)

		assert.Equal(t, 10, result["min_length"])
		assert.Equal(t, 20, result["max_length"])
	})
}

func TestPropertySpecV1_FromV0(t *testing.T) {
	t.Parallel()

	t.Run("property type conversions", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			name              string
			v0Type            string
			v0Config          map[string]interface{}
			expectedType      string
			expectedTypes     []string
			expectedItemType  string
			expectedItemTypes []string
			expectedConfig    map[string]interface{}
		}{
			{
				name:           "single primitive type",
				v0Type:         "string",
				v0Config:       nil,
				expectedType:   "string",
				expectedTypes:  nil,
				expectedConfig: nil,
			},
			{
				name:           "single type with camelCase config",
				v0Type:         "string",
				v0Config:       map[string]interface{}{"minLength": 5, "maxLength": 50},
				expectedType:   "string",
				expectedTypes:  nil,
				expectedConfig: map[string]interface{}{"min_length": 5, "max_length": 50},
			},
			{
				name:           "comma-separated types",
				v0Type:         "string,number",
				v0Config:       nil,
				expectedType:   "",
				expectedTypes:  []string{"string", "number"},
				expectedConfig: nil,
			},
			{
				name:           "comma-separated types with config",
				v0Type:         "string,number",
				v0Config:       map[string]interface{}{"multipleOf": 3, "pattern": "^[a-z]+$"},
				expectedType:   "",
				expectedTypes:  []string{"string", "number"},
				expectedConfig: map[string]interface{}{"multiple_of": 3, "pattern": "^[a-z]+$"},
			},
			{
				name:           "comma-separated types with spaces",
				v0Type:         "string, number, boolean",
				v0Config:       nil,
				expectedType:   "",
				expectedTypes:  []string{"string", "number", "boolean"},
				expectedConfig: nil,
			},
			{
				name:           "custom type reference",
				v0Type:         "#/custom-types/myGroup/MyType",
				v0Config:       nil,
				expectedType:   "#/custom-types/myGroup/MyType",
				expectedTypes:  nil,
				expectedConfig: nil,
			},
			{
				name:              "array type with single itemType config",
				v0Type:            "array",
				v0Config:          map[string]interface{}{"itemTypes": []interface{}{"string"}},
				expectedType:      "array",
				expectedTypes:     nil,
				expectedItemType:  "string",
				expectedItemTypes: nil,
				expectedConfig:    map[string]interface{}{},
			},
			{
				name:              "array type with itemTypes config",
				v0Type:            "array",
				v0Config:          map[string]interface{}{"itemTypes": []interface{}{"string", "number"}},
				expectedType:      "array",
				expectedTypes:     nil,
				expectedItemTypes: []string{"string", "number"},
				expectedConfig:    map[string]interface{}{},
			},
			{
				name:           "object type with nested config",
				v0Type:         "object",
				v0Config:       map[string]interface{}{"minLength": 1, "maxLength": 10},
				expectedType:   "object",
				expectedTypes:  nil,
				expectedConfig: map[string]interface{}{"min_length": 1, "max_length": 10},
			},
			{
				name:           "number type with complex config",
				v0Type:         "number",
				v0Config:       map[string]interface{}{"minimum": 0, "maximum": 100, "exclusiveMinimum": true, "exclusiveMaximum": false},
				expectedType:   "number",
				expectedTypes:  nil,
				expectedConfig: map[string]interface{}{"minimum": 0, "maximum": 100, "exclusive_minimum": true, "exclusive_maximum": false},
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()

				v0 := Property{
					LocalID: "test-prop",
					Name:    "Test Property",
					Type:    tt.v0Type,
					Config:  tt.v0Config,
				}

				var v1 PropertyV1
				v1.FromV0(v0)
				assert.Equal(t, tt.expectedType, v1.Type, "Type field mismatch")
				assert.Equal(t, tt.expectedTypes, v1.Types, "Types field mismatch")
				assert.Equal(t, tt.expectedItemType, v1.ItemType, "ItemType field mismatch")
				assert.Equal(t, tt.expectedItemTypes, v1.ItemTypes, "ItemTypes field mismatch")
				assert.Equal(t, tt.expectedConfig, v1.Config, "Config field mismatch")
				assert.Equal(t, v0.LocalID, v1.LocalID)
				assert.Equal(t, v0.Name, v1.Name)
			})
		}

	})

	t.Run("converts multiple properties with different configurations and preserves order", func(t *testing.T) {
		t.Parallel()

		v0Spec := PropertySpec{
			Properties: []Property{
				{
					LocalID:     "prop1",
					Name:        "Property 1",
					Description: "First property",
					Type:        "string",
					Config: map[string]interface{}{
						"minLength": 5,
					},
				},
				{
					LocalID:     "prop2",
					Name:        "Property 2",
					Description: "Second property",
					Type:        "integer",
					Config: map[string]interface{}{
						"minimum": 0,
						"maximum": 100,
					},
				},
				{
					LocalID: "prop3",
					Name:    "Property 3",
					Type:    "boolean",
					Config:  nil,
				},
				{
					LocalID: "prop4",
					Name:    "Property 4",
					Type:    "array",
				},
				{
					LocalID: "prop5",
					Name:    "Property 5",
					Type:    "#/custom-types/login_elements/email_type",
				},
			},
		}

		v1Spec := &PropertySpecV1{}
		err := v1Spec.FromV0(v0Spec)

		assert.NoError(t, err)
		assert.Len(t, v1Spec.Properties, 5)

		expected := []PropertyV1{
			{
				LocalID:     "prop1",
				Name:        "Property 1",
				Description: "First property",
				Type:        "string",
				Config: map[string]interface{}{
					"min_length": 5,
				},
			},
			{
				LocalID:     "prop2",
				Name:        "Property 2",
				Description: "Second property",
				Type:        "integer",
				Config: map[string]interface{}{
					"minimum": 0,
					"maximum": 100,
				},
			},
			{
				LocalID:     "prop3",
				Name:        "Property 3",
				Description: "",
				Type:        "boolean",
				Config:      nil,
			},
			{
				LocalID:     "prop4",
				Name:        "Property 4",
				Description: "",
				Type:        "array",
				Config:      nil,
			},
			{
				LocalID:     "prop5",
				Name:        "Property 5",
				Description: "",
				Type:        "#/custom-types/login_elements/email_type",
				Config:      nil,
			},
		}
		assert.Equal(t, expected, v1Spec.Properties)
	})
}

func TestCustomTypeV1_FromV0(t *testing.T) {
	t.Parallel()

	t.Run("converts V0 custom type with variants to V1 format", func(t *testing.T) {
		t.Parallel()

		v0CustomType := CustomType{
			LocalID:     "user_profile_type",
			Name:        "User Profile Type",
			Description: "Custom type for user profile with variants",
			Type:        "object",
			Config: map[string]any{
				"minLength": 5,
				"maxLength": 100,
			},
			Properties: []CustomTypeProperty{
				{
					Ref:      "#property:profile_name",
					Required: true,
				},
				{
					Ref:      "#property:profile_type",
					Required: true,
				},
				{
					Ref:      "#property:premium_features",
					Required: false,
				},
			},
			Variants: Variants{
				{
					Type:          "discriminator",
					Discriminator: "#property:profile_type",
					Cases: []VariantCase{
						{
							DisplayName: "Premium User",
							Match:       []any{"premium", "vip"},
							Description: "applies when user is premium",
							Properties: []PropertyReference{
								{
									Ref:      "#property:premium_features",
									Required: true,
								},
							},
						},
						{
							DisplayName: "Basic User",
							Match:       []any{"basic", "free"},
							Description: "applies when user is basic",
							Properties: []PropertyReference{
								{
									Ref:      "#property:profile_name",
									Required: true,
								},
							},
						},
					},
					Default: []PropertyReference{
						{
							Ref:      "#property:profile_name",
							Required: true,
						},
					},
				},
			},
		}

		var v1CustomType CustomTypeV1
		err := v1CustomType.FromV0(v0CustomType)

		assert.NoError(t, err)
		// Expected V1 spec after conversion
		expectedV1CustomType := CustomTypeV1{
			LocalID:     "user_profile_type",
			Name:        "User Profile Type",
			Description: "Custom type for user profile with variants",
			Type:        "object",
			Config: map[string]any{
				"min_length": 5,
				"max_length": 100,
			},
			Properties: []CustomTypePropertyV1{
				{
					Property: "#property:profile_name",
					Required: true,
				},
				{
					Property: "#property:profile_type",
					Required: true,
				},
				{
					Property: "#property:premium_features",
					Required: false,
				},
			},
			Variants: VariantsV1{
				{
					Type:          "discriminator",
					Discriminator: "#property:profile_type",
					Cases: []VariantCaseV1{
						{
							DisplayName: "Premium User",
							Match:       []any{"premium", "vip"},
							Description: "applies when user is premium",
							Properties: []PropertyReferenceV1{
								{
									Property: "#property:premium_features",
									Required: true,
								},
							},
						},
						{
							DisplayName: "Basic User",
							Match:       []any{"basic", "free"},
							Description: "applies when user is basic",
							Properties: []PropertyReferenceV1{
								{
									Property: "#property:profile_name",
									Required: true,
								},
							},
						},
					},
					Default: DefaultPropertiesV1{
						Properties: []PropertyReferenceV1{
							{
								Property: "#property:profile_name",
								Required: true,
							},
						},
					},
				},
			},
		}
		assert.Equal(t, expectedV1CustomType, v1CustomType)
	})
}

func TestTrackingPlanV1_FromV0(t *testing.T) {
	t.Parallel()

	t.Run("converts V0 tracking plan with event rules to V1 format", func(t *testing.T) {
		t.Parallel()

		v0TrackingPlan := TrackingPlan{
			Name:        "Mobile App Events",
			LocalID:     "mobile_app_tracking",
			Description: "Tracking plan for mobile app events",
			Rules: []*TPRule{
				{
					Type:    "event_rule",
					LocalID: "user_signup_rule",
					Event: &TPRuleEvent{
						Ref:             "#event:user_signup",
						AllowUnplanned:  true,
						IdentitySection: "context",
					},
					Properties: []*TPRuleProperty{
						{
							Ref:      "#property:user_email",
							Required: true,
						},
						{
							Ref:      "#property:signup_source",
							Required: false,
						},
					},
					Variants: Variants{
						{
							Type:          "discriminator",
							Discriminator: "#property:platform",
							Cases: []VariantCase{
								{
									DisplayName: "iOS",
									Match:       []any{"ios"},
									Description: "iOS platform",
									Properties: []PropertyReference{
										{
											Ref:      "#property:device_id",
											Required: true,
										},
									},
								},
							},
						},
					},
				},
				{
					Type:    "event_rule",
					LocalID: "user_login_rule",
					Event: &TPRuleEvent{
						Ref:             "#event:user_login",
						AllowUnplanned:  false,
						IdentitySection: "traits",
					},
					Properties: []*TPRuleProperty{
						{
							Ref:      "#property:login_method",
							Required: true,
						},
					},
				},
			},
		}

		var v1TrackingPlan TrackingPlanV1
		err := v1TrackingPlan.FromV0(&v0TrackingPlan)

		assert.NoError(t, err)
		// Expected V1 spec after conversion
		expectedV1TrackingPlan := TrackingPlanV1{
			Name:        "Mobile App Events",
			LocalID:     "mobile_app_tracking",
			Description: "Tracking plan for mobile app events",
			Rules: []*TPRuleV1{
				{
					Type:                 "event_rule",
					LocalID:              "user_signup_rule",
					Event:                "#event:user_signup", // Converted from object to direct reference
					IdentitySection:      "context",            // Moved from rules.event.identity_section
					AdditionalProperties: true,                 // Converted from rules.event.allow_unplanned
					Properties: []*TPRuleProperty{
						{
							Ref:      "#property:user_email",
							Required: true,
						},
						{
							Ref:      "#property:signup_source",
							Required: false,
						},
					},
					Variants: Variants{
						{
							Type:          "discriminator",
							Discriminator: "#property:platform",
							Cases: []VariantCase{
								{
									DisplayName: "iOS",
									Match:       []any{"ios"},
									Description: "iOS platform",
									Properties: []PropertyReference{
										{
											Ref:      "#property:device_id",
											Required: true,
										},
									},
								},
							},
						},
					},
				},
				{
					Type:                 "event_rule",
					LocalID:              "user_login_rule",
					Event:                "#event:user_login", // Converted from object to direct reference
					IdentitySection:      "traits",            // Moved from rules.event.identity_section
					AdditionalProperties: false,               // allow_unplanned was false
					Properties: []*TPRuleProperty{
						{
							Ref:      "#property:login_method",
							Required: true,
						},
					},
				},
			},
			EventProps: nil,
		}
		// Compare public fields only (RulesV0 is internal)
		assert.Equal(t, expectedV1TrackingPlan, v1TrackingPlan)
	})

	t.Run("handles tracking plan with nil event", func(t *testing.T) {
		t.Parallel()

		v0TrackingPlan := TrackingPlan{
			Name:        "Test Plan",
			LocalID:     "test_plan",
			Description: "Test tracking plan",
			Rules: []*TPRule{
				{
					Type:    "event_rule",
					LocalID: "rule_without_event",
					Event:   nil, // Edge case: no event
				},
			},
		}

		var v1TrackingPlan TrackingPlanV1
		err := v1TrackingPlan.FromV0(&v0TrackingPlan)

		assert.NoError(t, err)

		expectedV1 := TrackingPlanV1{
			Name:        "Test Plan",
			LocalID:     "test_plan",
			Description: "Test tracking plan",
			Rules: []*TPRuleV1{
				{
					Type:                 "event_rule",
					LocalID:              "rule_without_event",
					Event:                "",
					IdentitySection:      "",
					AdditionalProperties: false,
					Properties:           nil,
					Includes:             nil,
					Variants:             nil,
				},
			},
			EventProps: nil,
		}
		assert.Equal(t, expectedV1, v1TrackingPlan)
	})
}
