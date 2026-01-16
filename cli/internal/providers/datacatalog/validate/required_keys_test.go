package validate

import (
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	catalog "github.com/rudderlabs/rudder-iac/cli/internal/providers/datacatalog/localcatalog"
)

func TestPropertyArrayItemTypesValidation(t *testing.T) {
	validator := &RequiredKeysValidator{}

	testCases := []struct {
		name           string
		properties     map[catalog.EntityGroup][]catalog.PropertyV1
		expectedErrors int
		errorContains  string
	}{
		{
			name: "valid property with single custom type in itemTypes",
			properties: map[catalog.EntityGroup][]catalog.PropertyV1{
				"test-group": {
					{
						LocalID:     "array-prop",
						Name:        "Array Property",
						Description: "Property with array type",
						Type:        "array",
						Config: map[string]interface{}{
							"itemTypes": []interface{}{"#/custom-types/test-group/TestType"},
						},
					},
				},
			},
			expectedErrors: 0,
		},
		{
			name: "invalid property with multiple types including custom type in itemTypes",
			properties: map[catalog.EntityGroup][]catalog.PropertyV1{
				"test-group": {
					{
						LocalID:     "array-prop",
						Name:        "Array Property",
						Description: "Property with array type",
						Type:        "array",
						Config: map[string]interface{}{
							"itemTypes": []interface{}{
								"#/custom-types/test-group/TestType",
								"string",
							},
						},
					},
				},
			},
			expectedErrors: 1,
			errorContains:  "cannot be paired with other types",
		},
		{
			name: "invalid property with non-array itemTypes",
			properties: map[catalog.EntityGroup][]catalog.PropertyV1{
				"test-group": {
					{
						LocalID:     "array-prop",
						Name:        "Array Property",
						Description: "Property with array type",
						Type:        "array",
						Config: map[string]interface{}{
							"itemTypes": "string", // Not an array
						},
					},
				},
			},
			expectedErrors: 1,
			errorContains:  "itemTypes must be an array",
		},
		{
			name: "invalid property with non-string item in itemTypes",
			properties: map[catalog.EntityGroup][]catalog.PropertyV1{
				"test-group": {
					{
						LocalID:     "array-prop",
						Name:        "Array Property",
						Description: "Property with array type",
						Type:        "array",
						Config: map[string]interface{}{
							"itemTypes": []interface{}{123}, // Not a string
						},
					},
				},
			},
			expectedErrors: 1,
			errorContains:  "must be string value",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create a minimal data catalog with just the properties
			dc := &catalog.DataCatalog{
				Properties: tc.properties,
			}

			// Validate
			errors := validator.Validate(dc)

			// Check results
			if tc.expectedErrors == 0 {
				assert.Empty(t, errors, "Expected no validation errors")
			} else {
				assert.Len(t, errors, tc.expectedErrors, "Expected specific number of validation errors")
				if tc.errorContains != "" {
					assert.Contains(t, errors[0].Error(), tc.errorContains, "Error message should contain expected text")
				}
			}
		})
	}
}

func Test_IsInteger(t *testing.T) {
	testCases := []struct {
		name     string
		input    any
		expected bool
	}{
		// Valid integer types
		{
			name:     "int type",
			input:    42,
			expected: true,
		},
		{
			name:     "int32 type",
			input:    int32(42),
			expected: true,
		},
		{
			name:     "int64 type",
			input:    int64(42),
			expected: true,
		},
		{
			name:     "negative int",
			input:    -42,
			expected: true,
		},
		{
			name:     "zero int",
			input:    0,
			expected: true,
		},
		{
			name:     "float64 representing positive integer",
			input:    float64(42),
			expected: true,
		},
		{
			name:     "float64 representing negative integer",
			input:    float64(-42),
			expected: true,
		},
		{
			name:     "float64 representing zero",
			input:    float64(0),
			expected: true,
		},
		{
			name:     "float64 with decimal part",
			input:    42.5,
			expected: false,
		},
		{
			name:     "string type",
			input:    "42",
			expected: false,
		},
		{
			name:     "boolean type",
			input:    true,
			expected: false,
		},
		{
			name:     "nil value",
			input:    nil,
			expected: false,
		},
		{
			name:     "slice type",
			input:    []int{1, 2, 3},
			expected: false,
		},
		{
			name:     "map type",
			input:    map[string]int{"key": 42},
			expected: false,
		},
		{
			name:     "float32 type",
			input:    float32(42.0),
			expected: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			result := isInteger(tc.input)
			assert.Equal(t, tc.expected, result, "isInteger(%v) should return %v", tc.input, tc.expected)
		})
	}
}

func Test_IsNumber(t *testing.T) {
	testCases := []struct {
		name     string
		input    any
		expected bool
	}{
		// Valid signed integer types
		{
			name:     "int type",
			input:    42,
			expected: true,
		},
		{
			name:     "int8 type",
			input:    int8(42),
			expected: true,
		},
		{
			name:     "int16 type",
			input:    int16(42),
			expected: true,
		},
		{
			name:     "int32 type",
			input:    int32(42),
			expected: true,
		},
		{
			name:     "int64 type",
			input:    int64(42),
			expected: true,
		},
		{
			name:     "negative int",
			input:    -42,
			expected: true,
		},
		{
			name:     "zero int",
			input:    0,
			expected: true,
		},
		// Valid unsigned integer types
		{
			name:     "uint type",
			input:    uint(42),
			expected: true,
		},
		{
			name:     "uint8 type",
			input:    uint8(42),
			expected: true,
		},
		{
			name:     "uint16 type",
			input:    uint16(42),
			expected: true,
		},
		{
			name:     "uint32 type",
			input:    uint32(42),
			expected: true,
		},
		{
			name:     "uint64 type",
			input:    uint64(42),
			expected: true,
		},
		// Valid float types
		{
			name:     "float32 type",
			input:    float32(42.5),
			expected: true,
		},
		{
			name:     "float64 type",
			input:    float64(42.5),
			expected: true,
		},
		{
			name:     "float64 representing integer",
			input:    float64(42),
			expected: true,
		},
		{
			name:     "negative float",
			input:    -42.5,
			expected: true,
		},
		{
			name:     "zero float",
			input:    float64(0),
			expected: true,
		},
		// Invalid non-numeric types
		{
			name:     "string type",
			input:    "42",
			expected: false,
		},
		{
			name:     "boolean type",
			input:    true,
			expected: false,
		},
		{
			name:     "nil value",
			input:    nil,
			expected: false,
		},
		{
			name:     "slice type",
			input:    []int{1, 2, 3},
			expected: false,
		},
		{
			name:     "map type",
			input:    map[string]int{"key": 42},
			expected: false,
		},
		{
			name:     "struct type",
			input:    struct{ Value int }{Value: 42},
			expected: false,
		},
		{
			name:     "pointer type",
			input:    &[]int{42}[0],
			expected: false,
		},
		{
			name:     "channel type",
			input:    make(chan int),
			expected: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			result := isNumber(tc.input)
			assert.Equal(t, tc.expected, result, "isNumber(%v) should return %v", tc.input, tc.expected)
		})
	}
}

func TestPropertyNameWhitespaceValidation(t *testing.T) {
	validator := &RequiredKeysValidator{}

	testCases := []struct {
		name           string
		properties     map[catalog.EntityGroup][]catalog.PropertyV1
		expectedErrors int
		errorContains  string
	}{
		{
			name: "property with empty type",
			properties: map[catalog.EntityGroup][]catalog.PropertyV1{
				"test-group": {
					{
						LocalID: "prop-without-type",
						Name:    "Property Without Type",
					},
				},
			},
			expectedErrors: 0,
		},
		{
			name: "valid property name without whitespace",
			properties: map[catalog.EntityGroup][]catalog.PropertyV1{
				"test-group": {
					{
						LocalID:     "valid-prop",
						Name:        "Valid Property Name",
						Description: "A valid property name",
						Type:        "string",
					},
				},
			},
			expectedErrors: 0,
		},
		{
			name: "property name with leading whitespace",
			properties: map[catalog.EntityGroup][]catalog.PropertyV1{
				"test-group": {
					{
						LocalID:     "leading-space-prop",
						Name:        " Property With Leading Space",
						Description: "Property with leading whitespace",
						Type:        "string",
					},
				},
			},
			expectedErrors: 1,
			errorContains:  "property name cannot have leading or trailing whitespace characters",
		},
		{
			name: "property name with trailing whitespace",
			properties: map[catalog.EntityGroup][]catalog.PropertyV1{
				"test-group": {
					{
						LocalID:     "trailing-space-prop",
						Name:        "Property With Trailing Space ",
						Description: "Property with trailing whitespace",
						Type:        "string",
					},
				},
			},
			expectedErrors: 1,
			errorContains:  "property name cannot have leading or trailing whitespace characters",
		},
		{
			name: "property name with both leading and trailing whitespace",
			properties: map[catalog.EntityGroup][]catalog.PropertyV1{
				"test-group": {
					{
						LocalID:     "both-space-prop",
						Name:        " Property With Both Spaces ",
						Description: "Property with both leading and trailing whitespace",
						Type:        "string",
					},
				},
			},
			expectedErrors: 1,
			errorContains:  "property name cannot have leading or trailing whitespace characters",
		},
		{
			name: "property name with internal spaces (should be valid)",
			properties: map[catalog.EntityGroup][]catalog.PropertyV1{
				"test-group": {
					{
						LocalID:     "internal-space-prop",
						Name:        "Property With Internal Spaces",
						Description: "Property with internal spaces which should be allowed",
						Type:        "string",
					},
				},
			},
			expectedErrors: 0,
		},
		{
			name: "empty property name should trigger mandatory field error, not whitespace error",
			properties: map[catalog.EntityGroup][]catalog.PropertyV1{
				"test-group": {
					{
						LocalID:     "empty-name-prop",
						Name:        "",
						Description: "Property with empty name",
						Type:        "string",
					},
				},
			},
			expectedErrors: 1,
			errorContains:  "id, name and type fields on property are mandatory",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create a minimal data catalog with just the properties
			dc := &catalog.DataCatalog{
				Properties: tc.properties,
			}

			// Validate
			errors := validator.Validate(dc)

			// Check results
			if tc.expectedErrors == 0 {
				assert.Empty(t, errors, "Expected no validation errors")
			} else {
				assert.Len(t, errors, tc.expectedErrors, "Expected %d validation errors, got %d", tc.expectedErrors, len(errors))
				if tc.errorContains != "" {
					found := false
					for _, err := range errors {
						if strings.Contains(err.Error(), tc.errorContains) {
							found = true
							break
						}
					}
					assert.True(t, found, "Expected to find error containing '%s' in errors: %v", tc.errorContains, errors)
				}
			}
		})
	}
}

func TestCategoryValidation(t *testing.T) {
	validator := &RequiredKeysValidator{}

	testCases := []struct {
		name           string
		categories     map[catalog.EntityGroup][]catalog.Category
		expectedErrors int
		errorContains  string
	}{
		{
			name: "valid category",
			categories: map[catalog.EntityGroup][]catalog.Category{
				"test-group": {
					{
						LocalID: "valid-category",
						Name:    "Valid Category",
					},
				},
			},
			expectedErrors: 0,
		},
		{
			name: "category with missing fields",
			categories: map[catalog.EntityGroup][]catalog.Category{
				"test-group": {
					{
						LocalID: "",
						Name:    "",
					},
				},
			},
			expectedErrors: 1,
			errorContains:  "id and name fields on category are mandatory",
		},
		{
			name: "category with missing LocalID",
			categories: map[catalog.EntityGroup][]catalog.Category{
				"test-group": {
					{
						LocalID: "",
						Name:    "Valid Name",
					},
				},
			},
			expectedErrors: 1,
			errorContains:  "id and name fields on category are mandatory",
		},
		{
			name: "category with missing Name",
			categories: map[catalog.EntityGroup][]catalog.Category{
				"test-group": {
					{
						LocalID: "valid-id",
						Name:    "",
					},
				},
			},
			expectedErrors: 1,
			errorContains:  "id and name fields on category are mandatory",
		},
		{
			name: "category with leading whitespace in name",
			categories: map[catalog.EntityGroup][]catalog.Category{
				"test-group": {
					{
						LocalID: "leading-space",
						Name:    " Category With Leading Space",
					},
				},
			},
			expectedErrors: 1,
			errorContains:  "category name cannot have leading or trailing whitespace characters",
		},
		{
			name: "category with trailing whitespace in name",
			categories: map[catalog.EntityGroup][]catalog.Category{
				"test-group": {
					{
						LocalID: "trailing-space",
						Name:    "Category With Trailing Space ",
					},
				},
			},
			expectedErrors: 1,
			errorContains:  "category name cannot have leading or trailing whitespace characters",
		},
		{
			name: "category with invalid name format",
			categories: map[catalog.EntityGroup][]catalog.Category{
				"test-group": {
					{
						LocalID: "invalid-format",
						Name:    "!@#Invalid",
					},
				},
			},
			expectedErrors: 1,
			errorContains:  "category name must start with a letter (upper/lower case) or underscore",
		},
		{
			name: "category with valid name formats",
			categories: map[catalog.EntityGroup][]catalog.Category{
				"test-group": {
					{
						LocalID: "uppercase-start",
						Name:    "Uppercase Start",
					},
					{
						LocalID: "lowercase-start",
						Name:    "lowercase start",
					},
					{
						LocalID: "underscore-start",
						Name:    "_underscore start",
					},
					{
						LocalID: "with-numbers",
						Name:    "Category123",
					},
					{
						LocalID: "with-special-chars",
						Name:    "Category-Name.With,Special",
					},
				},
			},
			expectedErrors: 0,
		},
		{
			name: "category name too short",
			categories: map[catalog.EntityGroup][]catalog.Category{
				"test-group": {
					{
						LocalID: "too-short",
						Name:    "A",
					},
				},
			},
			expectedErrors: 1,
			errorContains:  "category name must start with a letter (upper/lower case) or underscore",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create a minimal data catalog with just the categories
			dc := &catalog.DataCatalog{
				Categories: tc.categories,
			}

			// Validate
			errors := validator.Validate(dc)

			// Check results
			if tc.expectedErrors == 0 {
				assert.Empty(t, errors, "Expected no validation errors")
			} else {
				assert.Len(t, errors, tc.expectedErrors, "Expected %d validation errors, got %d", tc.expectedErrors, len(errors))
				if tc.errorContains != "" {
					found := false
					for _, err := range errors {
						if strings.Contains(err.Error(), tc.errorContains) {
							found = true
							break
						}
					}
					assert.True(t, found, "Expected to find error containing '%s' in errors: %v", tc.errorContains, errors)
				}
			}
		})
	}
}

func TestVariantsValidation(t *testing.T) {
	validator := &RequiredKeysValidator{}

	t.Run("variants validation success", func(t *testing.T) {
		testCases := []struct {
			name          string
			trackingPlans map[catalog.EntityGroup]*catalog.TrackingPlan
			customTypes   map[catalog.EntityGroup][]catalog.CustomType
		}{
			{
				name: "valid variants in tracking plan",
				trackingPlans: map[catalog.EntityGroup]*catalog.TrackingPlan{
					"test-group": {
						LocalID: "test-tp",
						Name:    "Test Tracking Plan",
						Rules: []*catalog.TPRule{
							{
								LocalID: "test-rule",
								Type:    "event_rule",
								Event: &catalog.TPRuleEvent{
									Ref: "#/events/test-group/test-event",
								},
								Variants: catalog.Variants{
									{
										Type:          "discriminator",
										Discriminator: "page_name",
										Cases: []catalog.VariantCase{
											{
												DisplayName: "Search Page",
												Match:       []any{"search", "search_bar"},
												Properties: []catalog.PropertyReference{
													{Ref: "#/properties/test-group/search_term", Required: true},
												},
											},
										},
										Default: []catalog.PropertyReference{
											{Ref: "#/properties/test-group/page_url", Required: true},
										},
									},
								},
							},
						},
					},
				},
			},
			{
				name: "valid variants in custom type",
				customTypes: map[catalog.EntityGroup][]catalog.CustomType{
					"test-group": {
						{
							LocalID:     "TestType",
							Name:        "TestType",
							Description: "Test custom type with variants",
							Type:        "object",
							Properties: []catalog.CustomTypeProperty{
								{Ref: "#/properties/test-group/profile_type", Required: true},
							},
							Variants: catalog.Variants{
								{
									Type:          "discriminator",
									Discriminator: "profile_type",
									Cases: []catalog.VariantCase{
										{
											DisplayName: "Premium User",
											Match:       []any{"premium", "vip"},
											Properties: []catalog.PropertyReference{
												{Ref: "#/properties/test-group/subscription_tier", Required: true},
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
				name: "valid variants with mixed match value types",
				trackingPlans: map[catalog.EntityGroup]*catalog.TrackingPlan{
					"test-group": {
						LocalID: "test-tp",
						Name:    "Test Tracking Plan",
						Rules: []*catalog.TPRule{
							{
								LocalID: "test-rule",
								Type:    "event_rule",
								Event: &catalog.TPRuleEvent{
									Ref: "#/events/test-group/test-event",
								},
								Variants: catalog.Variants{
									{
										Type:          "discriminator",
										Discriminator: "user_id",
										Cases: []catalog.VariantCase{
											{
												DisplayName: "Admin User",
												Match:       []any{123, 123.0, true, "admin"},
												Properties: []catalog.PropertyReference{
													{Ref: "#/properties/test-group/admin_level", Required: true},
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

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				dc := &catalog.DataCatalog{
					TrackingPlans: tc.trackingPlans,
					CustomTypes:   tc.customTypes,
				}
				errors := validator.Validate(dc)
				assert.Empty(t, errors, "Expected no validation errors")
			})
		}
	})

	t.Run("variants validation failures", func(t *testing.T) {
		testCases := []struct {
			name           string
			trackingPlans  map[catalog.EntityGroup]*catalog.TrackingPlan
			customTypes    map[catalog.EntityGroup][]catalog.CustomType
			expectedErrors int
			errorContains  []ValidationError
		}{
			{
				name: "structural validation failures",
				trackingPlans: map[catalog.EntityGroup]*catalog.TrackingPlan{
					"test-group": {
						LocalID: "test-tp",
						Name:    "Test Tracking Plan",
						Rules: []*catalog.TPRule{
							{
								LocalID: "test-rule",
								Type:    "event_rule",
								Event: &catalog.TPRuleEvent{
									Ref: "#/events/test-group/test-event",
								},
								Variants: catalog.Variants{
									{
										Type:          "other_type", // Invalid type
										Discriminator: "",           // Missing discriminator
										Cases: []catalog.VariantCase{
											{
												DisplayName: "",                            // Missing display name
												Match:       []any{},                       // Empty match array
												Properties:  []catalog.PropertyReference{}, // Empty properties array
											},
										},
									},
									{
										Type:          "discriminator", // Second variant (should fail length check)
										Discriminator: "page_name",
										Cases: []catalog.VariantCase{
											{
												DisplayName: "Search Page",
												Match:       []any{"search"},
												Properties: []catalog.PropertyReference{
													{Required: true}, // Missing Ref
												},
											},
										},
									},
								},
							},
						},
					},
				},
				expectedErrors: 7,
				errorContains: []ValidationError{
					{
						error:     fmt.Errorf("type field is mandatory for variant and must be 'discriminator'"),
						Reference: "#/tp/test-group/test-tp/rules/test-rule/variants[0]",
					},
					{
						error:     fmt.Errorf("discriminator field is mandatory for variant"),
						Reference: "#/tp/test-group/test-tp/rules/test-rule/variants[0]",
					},
					{
						error:     fmt.Errorf("display_name field is mandatory for variant case"),
						Reference: "#/tp/test-group/test-tp/rules/test-rule/variants[0]/cases[0]",
					},
					{
						error:     fmt.Errorf("match array must have at least one element"),
						Reference: "#/tp/test-group/test-tp/rules/test-rule/variants[0]/cases[0]",
					},
					{
						error:     fmt.Errorf("properties array must have at least one element"),
						Reference: "#/tp/test-group/test-tp/rules/test-rule/variants[0]/cases[0]",
					},
					{
						error:     fmt.Errorf("variants array cannot have more than 1 variant (current length: 2)"),
						Reference: "#/tp/test-group/test-tp/rules/test-rule",
					},
					{
						error:     fmt.Errorf("$ref field is mandatory for property reference"),
						Reference: "#/tp/test-group/test-tp/rules/test-rule/variants[1]/cases[0]/properties[0]",
					},
				},
			},
			{
				name: "custom type validation failures",
				customTypes: map[catalog.EntityGroup][]catalog.CustomType{
					"test-group": {
						{
							LocalID:     "TestType",
							Name:        "TestType",
							Description: "Test custom type with variants",
							Type:        "string",
							Variants: catalog.Variants{
								{
									Type:          "discriminator",
									Discriminator: "profile_type",
									Cases: []catalog.VariantCase{
										{
											DisplayName: "Premium User",
											Match:       []any{"premium"},
											Properties: []catalog.PropertyReference{
												{Ref: "#/properties/test-group/subscription_tier", Required: true},
											},
										},
									},
								},
							},
						},
					},
				},
				expectedErrors: 1,
				errorContains: []ValidationError{
					{
						error:     fmt.Errorf("variants are only allowed for custom type of type object"),
						Reference: "#/custom-types/test-group/TestType",
					},
				},
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {

				dc := &catalog.DataCatalog{
					TrackingPlans: tc.trackingPlans,
					CustomTypes:   tc.customTypes,
				}
				errors := validator.Validate(dc)
				assert.Len(t, errors, tc.expectedErrors, "Expected %d validation errors, got %d", tc.expectedErrors, len(errors))

				if len(tc.errorContains) > 0 {
					// Check that we have the expected number of errors
					assert.Len(t, errors, tc.expectedErrors, "Expected %d validation errors, got %d", tc.expectedErrors, len(errors))

					for _, actual := range errors {
						found := false

						for _, expected := range tc.errorContains {
							if actual.Error() == expected.Error() && actual.Reference == expected.Reference {
								found = true
								break
							}
						}

						if !found {
							assert.Failf(t, "variants_validation_failures", "Expected to find error: %v with reference: %s in expected", actual, actual.Reference)
						}

					}
				}
			})
		}
	})
}

func TestRequiredKeysValidator_NestedPropertiesValidation(t *testing.T) {
	t.Parallel()

	falseVal := false
	// Setup test data catalog
	dc := &catalog.DataCatalog{
		Properties: map[catalog.EntityGroup][]catalog.PropertyV1{
			"test_props": {
				{
					LocalID:     "user_id",
					Name:        "User ID",
					Type:        "string",
					Description: "User identifier",
				},
				{
					LocalID:     "user_profile",
					Name:        "User Profile",
					Type:        "object",
					Description: "User profile object",
				},
				{
					LocalID:     "profile_name",
					Name:        "Profile Name",
					Type:        "string",
					Description: "User's display name",
				},
				{
					LocalID:     "profile_settings",
					Name:        "Profile Settings",
					Type:        "object",
					Description: "User profile settings",
				},
				{
					LocalID:     "theme_preference",
					Name:        "Theme Preference",
					Type:        "string",
					Description: "User's theme preference",
				},
				{
					LocalID:     "button_signin",
					Name:        "Button Sign In",
					Type:        "string", // Wrong type for nested properties
					Description: "Sign in button",
				},
				{
					LocalID:     "deeply_nested_property",
					Name:        "Deeply Nested Property",
					Type:        "string",
					Description: "Deeply nested property",
				},
			},
		},
		TrackingPlans: map[catalog.EntityGroup]*catalog.TrackingPlan{
			"test_tp": {
				LocalID: "test_plan",
				Name:    "Test Tracking Plan",
				Rules: []*catalog.TPRule{
					{
						LocalID: "valid_non_nested_rule",
						Type:    "event_rule",
						Event: &catalog.TPRuleEvent{
							Ref: "#/events/test/signup",
						},
						Properties: []*catalog.TPRuleProperty{
							{
								Ref:      "#/properties/test_props/user_profile",
								Required: true,
								AdditionalProperties: &falseVal,
							},
						},
					},
					{
						LocalID: "valid_nested_rule",
						Type:    "event_rule",
						Event: &catalog.TPRuleEvent{
							Ref: "#/events/test/signup",
						},
						Properties: []*catalog.TPRuleProperty{
							{
								Ref:      "#/properties/test_props/user_profile",
								Required: true,
								Properties: []*catalog.TPRuleProperty{
									{
										Ref:      "#/properties/test_props/profile_name",
										Required: true,
									},
								},
							},
						},
					},
					{
						LocalID: "invalid_object_type_rule",
						Type:    "event_rule",
						Event: &catalog.TPRuleEvent{
							Ref: "#/events/test/signup",
						},
						Properties: []*catalog.TPRuleProperty{
							{
								Ref:      "#/properties/test_props/button_signin", // string type with nested properties
								Required: true,
								Properties: []*catalog.TPRuleProperty{
									{
										Ref:      "#/properties/test_props/user_id",
										Required: true,
									},
								},
							},
						},
					},
					{
						LocalID: "exceed_depth_rule",
						Type:    "event_rule",
						Event: &catalog.TPRuleEvent{
							Ref: "#/events/test/signup",
						},
						Properties: []*catalog.TPRuleProperty{
							{
								Ref:      "#/properties/test_props/user_profile",
								Required: true,
								Properties: []*catalog.TPRuleProperty{
									{
										Ref:      "#/properties/test_props/profile_settings",
										Required: true,
										Properties: []*catalog.TPRuleProperty{
											{
												Ref:      "#/properties/test_props/user_profile",
												Required: true,
												Properties: []*catalog.TPRuleProperty{
													{
														Ref:      "#/properties/test_props/theme_preference", // 4th level - should exceed limit
														Required: true,
														Properties: []*catalog.TPRuleProperty{
															{
																Ref:      "#/properties/test_props/deeply_nested_property",
																Required: true,
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
					},
				},
			},
		},
		Events:      make(map[catalog.EntityGroup][]catalog.Event),
		CustomTypes: make(map[catalog.EntityGroup][]catalog.CustomType),
		Categories:  make(map[catalog.EntityGroup][]catalog.Category),
	}

	validator := &RequiredKeysValidator{}
	errors := validator.Validate(dc)

	// Test 1: Valid nested properties should not generate errors for nested structure
	validRuleErrors := filterErrorsByReference(errors, "#/tp/test_tp/test_plan/rules/valid_nested_rule")
	assert.Empty(t, validRuleErrors, "Valid nested properties should not generate validation errors")

	// Test 2: Object-type constraint violation should generate error
	objectTypeErrors := filterErrorsByReference(errors, "#/tp/test_tp/test_plan/rules/invalid_object_type_rule")
	assert.Len(t, objectTypeErrors, 1, "Should have one error for object-type constraint violation")
	assert.Contains(t, objectTypeErrors[0].Error(), "nested properties are not allowed for property")

	// Test 3: Nesting depth exceeded should generate error
	depthErrors := filterErrorsByReference(errors, "#/tp/test_tp/test_plan/rules/exceed_depth_rule")
	depthFound := false
	for _, err := range depthErrors {
		if strings.Contains(err.Error(), "maximum property nesting depth of 3 levels exceeded in event_rule") {
			depthFound = true
			break
		}
	}
	assert.True(t, depthFound, "Should have error for exceeding maximum nesting depth")

	// Test 4: Additional properties not allowed for non-nested properties
	additionalPropertiesErrors := filterErrorsByReference(errors, "#/tp/test_tp/test_plan/rules/valid_non_nested_rule")
	assert.Len(t, additionalPropertiesErrors, 1, "Should have one error for additional properties not allowed for non-nested properties")
	assert.Contains(t, additionalPropertiesErrors[0].Error(), "setting additional_properties is only allowed for nested properties")
}

// Helper function to filter errors by reference pattern
func filterErrorsByReference(errors []ValidationError, referencePattern string) []ValidationError {
	var filtered []ValidationError
	for _, err := range errors {
		if strings.Contains(err.Reference, referencePattern) {
			filtered = append(filtered, err)
		}
	}
	return filtered
}

func TestNestedPropertiesAllowed(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name          string
		propertyType  string
		config        map[string]any
		expectedAllow bool
		expectError   bool
		errorContains string
	}{
		{
			name:          "object type only - should allow",
			propertyType:  "object",
			config:        map[string]any{},
			expectedAllow: true,
			expectError:   false,
		},
		{
			name:          "object type with other types - should allow",
			propertyType:  "string, object, null",
			config:        map[string]any{},
			expectedAllow: true,
			expectError:   false,
		},
		{
			name:          "array type without itemTypes config - should allow",
			propertyType:  "array, string, number",
			config:        map[string]any{"maxItems": 10, "minItems": 1},
			expectedAllow: true,
			expectError:   false,
		},
		{
			name:         "array with itemTypes containing object - should allow",
			propertyType: "array",
			config: map[string]any{
				"itemTypes": []any{"string", "object", "null"},
			},
			expectedAllow: true,
			expectError:   false,
		},
		{
			name:         "array with itemTypes containing multiple non-object types - should not allow",
			propertyType: "array",
			config: map[string]any{
				"itemTypes": []any{"string", "number", "boolean"},
			},
			expectedAllow: false,
			expectError:   false,
		},
		{
			name:          "string type - should not allow",
			propertyType:  "string,number,boolean",
			config:        map[string]any{},
			expectedAllow: false,
			expectError:   false,
		},
		{
			name:          "object and array type - should not allow",
			propertyType:  "object, array",
			config:        map[string]any{},
			expectedAllow: false,
			expectError:   false,
		},
		{
			name:         "array with itemTypes as string instead of array - should error",
			propertyType: "array",
			config: map[string]any{
				"itemTypes": "string",
			},
			expectedAllow: false,
			expectError:   true,
			errorContains: "itemTypes must be an array",
		},
		{
			name:         "array with itemTypes containing mixed types - should error",
			propertyType: "array",
			config: map[string]any{
				"itemTypes": []any{"object", "string", 456},
			},
			expectedAllow: false,
			expectError:   true,
			errorContains: "itemTypes at index 2 must be a string value",
		},
		{
			name:         "array with itemTypes containing nil - should error",
			propertyType: "array",
			config: map[string]any{
				"itemTypes": []any{nil},
			},
			expectedAllow: false,
			expectError:   true,
			errorContains: "itemTypes at index 0 must be a string value",
		},
		{
			name:          "empty property type - should not allow",
			propertyType:  "",
			config:        map[string]any{},
			expectedAllow: false,
			expectError:   false,
		},
		{
			name:          "nil config with object type - should allow",
			propertyType:  "object",
			config:        nil,
			expectedAllow: true,
			expectError:   false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			allowed, err := nestedPropertiesAllowed(tc.propertyType, tc.config)

			// Check error expectations
			if tc.expectError {
				assert.Error(t, err, "Expected an error but got none")
				if tc.errorContains != "" {
					assert.Contains(t, err.Error(), tc.errorContains, "Error message should contain expected text")
				}
			} else {
				assert.NoError(t, err, "Expected no error but got: %v", err)
			}

			// Check allowed expectation
			assert.Equal(t, tc.expectedAllow, allowed, "nestedPropertiesAllowed returned unexpected result")
		})
	}
}
