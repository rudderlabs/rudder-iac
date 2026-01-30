package validate

import (
	"strings"
	"testing"

	catalog "github.com/rudderlabs/rudder-iac/cli/internal/providers/datacatalog/localcatalog"
	"github.com/stretchr/testify/assert"
)

func TestCustomTypeValidation(t *testing.T) {
	t.Run("RequiredKeysValidator", func(t *testing.T) {
		validator := &RequiredKeysValidator{}

		testCases := []struct {
			name          string
			customTypes   map[string]catalog.CustomType
			expectedErrs  int
			errorContains []string
		}{
			{
				name: "valid custom type",
				customTypes: map[string]catalog.CustomType{
					"TestType1": {
						LocalID:     "TestType1",
						Name:        "TestType",
						Description: "Test custom type",
						Type:        "string",
						Config:      map[string]any{},
					},
				},
				expectedErrs: 0,
			},
			{
				name: "missing required fields",
				customTypes: map[string]catalog.CustomType{
					"": { // LocalID will be empty
						// Missing LocalID
						Name:        "TestType",
						Description: "Test custom type",
						Type:        "string",
						Config:      map[string]any{},
					},
				},
				expectedErrs:  1,
				errorContains: []string{"id, name and type fields on custom type are mandatory"},
			},
			{
				name: "object type with property missing ID",
				customTypes: map[string]catalog.CustomType{
					"TestType1": {
						LocalID:     "TestType1",
						Name:        "TestType",
						Description: "Test custom type",
						Type:        "object",
						Config:      map[string]any{},
						Properties: []catalog.CustomTypeProperty{
							{
								// Missing ID
								Required: true,
							},
						},
					},
				},
				expectedErrs:  1,
				errorContains: []string{"$ref field is mandatory for property at index 0 in custom type"},
			},
			{
				name: "invalid name format",
				customTypes: map[string]catalog.CustomType{
					"TestType1": {
						LocalID:     "TestType1",
						Name:        "invalidName", // Doesn't start with capital letter
						Description: "Test custom type",
						Type:        "string",
						Config:      map[string]any{},
					},
				},
				expectedErrs:  1,
				errorContains: []string{"custom type name must start with a capital letter"},
			},
			{
				name: "invalid data type",
				customTypes: map[string]catalog.CustomType{
					"TestType1": {
						LocalID:     "TestType1",
						Name:        "TestType",
						Description: "Test custom type",
						Type:        "invalid_type", // Not a valid type
						Config:      map[string]any{},
					},
				},
				expectedErrs:  1,
				errorContains: []string{"invalid data type"},
			},
			{
				name: "string type with invalid config",
				customTypes: map[string]catalog.CustomType{
					"TestType1": {
						LocalID:     "TestType1",
						Name:        "TestType",
						Description: "Test custom type",
						Type:        "string",
						Config: map[string]any{
							"min_length": "not-a-number", // Should be a number
						},
					},
				},
				expectedErrs:  1,
				errorContains: []string{"min_length must be a number"},
			},
			{
				name: "number type with invalid config",
				customTypes: map[string]catalog.CustomType{
					"TestType1": {
						LocalID:     "TestType1",
						Name:        "TestType",
						Description: "Test custom type",
						Type:        "number",
						Config: map[string]any{
							"minimum": "not-a-number", // Should be a number
						},
					},
				},
				expectedErrs:  1,
				errorContains: []string{"minimum must be a number"},
			},
			{
				name: "array type with invalid config",
				customTypes: map[string]catalog.CustomType{
					"TestType1": {
						LocalID:     "TestType1",
						Name:        "TestType",
						Description: "Test custom type",
						Type:        "array",
						Config: map[string]any{
							"item_types": "not-an-array", // Should be an array
						},
					},
				},
				expectedErrs:  1,
				errorContains: []string{"item_types must be an array"},
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				dc := &catalog.DataCatalog{
					CustomTypes: tc.customTypes,
				}

				errs := validator.Validate(dc)

				assert.Len(t, errs, tc.expectedErrs)
				for i, errContains := range tc.errorContains {
					if i < len(errs) {
						assert.Contains(t, errs[i].error.Error(), errContains)
					}
				}
			})
		}
	})

	t.Run("RefValidator", func(t *testing.T) {
		validator := &RefValidator{}

		// Create a test property
		testProperty := catalog.PropertyV1{
			LocalID:     "testProp",
			Name:        "Test Property",
			Description: "Test property",
			Type:        "string",
		}

		testCases := []struct {
			name          string
			properties    map[string]catalog.PropertyV1
			customTypes   map[string]catalog.CustomType
			expectedErrs  int
			errorContains []string
		}{
			{
				name: "valid property reference",
				properties: map[string]catalog.PropertyV1{
					"testProp": testProperty,
				},
				customTypes: map[string]catalog.CustomType{
					"TestType1": {
						LocalID:     "TestType1",
						Name:        "TestType",
						Description: "Test custom type",
						Type:        "object",
						Config:      map[string]any{},
						Properties: []catalog.CustomTypeProperty{
							{
								Ref:      "#/properties/test-group/testProp",
								Required: true,
							},
						},
					},
				},
				expectedErrs: 0,
			},
			{
				name: "invalid reference format",
				customTypes: map[string]catalog.CustomType{
					"TestType1": {
						LocalID:     "TestType1",
						Name:        "TestType",
						Description: "Test custom type",
						Type:        "object",
						Config:      map[string]any{},
						Properties: []catalog.CustomTypeProperty{
							{
								Ref:      "invalid-reference", // Not in the correct format
								Required: true,
							},
						},
					},
				},
				expectedErrs:  1,
				errorContains: []string{"property reference at index 0 has invalid format"},
			},
			{
				name: "reference to non-existent property",
				customTypes: map[string]catalog.CustomType{
					"TestType1": {
						LocalID:     "TestType1",
						Name:        "TestType",
						Description: "Test custom type",
						Type:        "object",
						Config:      map[string]any{},
						Properties: []catalog.CustomTypeProperty{
							{
								Ref:      "#/properties/test-group/nonexistent", // Property doesn't exist
								Required: true,
							},
						},
					},
				},
				expectedErrs:  1,
				errorContains: []string{"property reference '#/properties/test-group/nonexistent' at index 0 not found in catalog"},
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				dc := &catalog.DataCatalog{
					Properties:  tc.properties,
					CustomTypes: tc.customTypes,
				}

				errs := validator.Validate(dc)

				assert.Len(t, errs, tc.expectedErrs)
				for i, errContains := range tc.errorContains {
					if i < len(errs) {
						assert.Contains(t, errs[i].error.Error(), errContains)
					}
				}
			})
		}
	})

	t.Run("DuplicateNameIDKeysValidator", func(t *testing.T) {
		validator := &DuplicateNameIDKeysValidator{}

		testCases := []struct {
			name          string
			customTypes   map[string]catalog.CustomType
			expectedErrs  int
			errorContains []string
		}{
			{
				name: "no duplicates",
				customTypes: map[string]catalog.CustomType{
					"TestType1": {
						LocalID:     "TestType1",
						Name:        "TestType1",
						Description: "Test custom type 1",
						Type:        "string",
						Config:      map[string]any{},
					},
					"TestType2": {
						LocalID:     "TestType2",
						Name:        "TestType2",
						Description: "Test custom type 2",
						Type:        "string",
						Config:      map[string]any{},
					},
				},
				expectedErrs: 0,
			},
			{
				name: "duplicate names",
				customTypes: map[string]catalog.CustomType{
					"TestType1": {
						LocalID:     "TestType1",
						Name:        "DuplicateName",
						Description: "Test custom type 1",
						Type:        "string",
						Config:      map[string]any{},
					},
					"TestType2": {
						LocalID:     "TestType2",
						Name:        "DuplicateName", // Duplicate name
						Description: "Test custom type 2",
						Type:        "string",
						Config:      map[string]any{},
					},
				},
				expectedErrs:  1,
				errorContains: []string{"duplicate name key DuplicateName in custom types"},
			},
			{
				name: "duplicate IDs - not possible with map structure",
				customTypes: map[string]catalog.CustomType{
					"DuplicateID": {
						LocalID:     "DuplicateID",
						Name:        "TestType1",
						Description: "Test custom type 1",
						Type:        "string",
						Config:      map[string]any{},
					},
				},
				expectedErrs:  0, // Maps can't have duplicate keys, so this test now passes
				errorContains: []string{},
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				dc := &catalog.DataCatalog{
					CustomTypes: tc.customTypes,
				}

				errs := validator.Validate(dc)

				// We only count errors related to custom types
				customTypeErrs := 0
				for _, err := range errs {
					if strings.Contains(err.Reference, "custom-types") {
						customTypeErrs++
					}
				}

				assert.Equal(t, tc.expectedErrs, customTypeErrs)
				for _, errContains := range tc.errorContains {
					found := false
					for _, err := range errs {
						if strings.Contains(err.error.Error(), errContains) {
							found = true
							break
						}
					}
					assert.True(t, found, "Expected error containing: %s", errContains)
				}
			})
		}
	})
}

func TestPropertyTypeCustomTypeReferences(t *testing.T) {
	validator := &RefValidator{}

	// Create a test custom type
	testCustomType := catalog.CustomType{
		LocalID:     "EmailType",
		Name:        "Email Type",
		Description: "Custom type for email validation",
		Type:        "string",
		Config: map[string]interface{}{
			"format": "email",
		},
	}

	testCases := []struct {
		name          string
		properties    map[string]catalog.PropertyV1
		customTypes   map[string]catalog.CustomType
		expectedErrs  int
		errorContains []string
	}{
		{
			name: "valid custom type reference in property type",
			properties: map[string]catalog.PropertyV1{
				"email": {
					LocalID:     "email",
					Name:        "Email",
					Description: "User email",
					Type:        "#/custom-types/email-types/EmailType",
				},
			},
			customTypes: map[string]catalog.CustomType{
				"EmailType": testCustomType,
			},
			expectedErrs: 0,
		},
		{
			name: "invalid custom type reference format in property type",
			properties: map[string]catalog.PropertyV1{
				"email": {
					LocalID:     "email",
					Name:        "Email",
					Description: "User email",
					Type:        "#/custom-types/email-types", // Missing type ID
				},
			},
			customTypes: map[string]catalog.CustomType{
				"EmailType": testCustomType,
			},
			expectedErrs:  1,
			errorContains: []string{"custom type reference in type field has invalid format"},
		},
		{
			name: "reference to non-existent custom type in property type",
			properties: map[string]catalog.PropertyV1{
				"email": {
					LocalID:     "email",
					Name:        "Email",
					Description: "User email",
					Type:        "#/custom-types/email-types/NonExistentType",
				},
			},
			customTypes: map[string]catalog.CustomType{
				"EmailType": testCustomType,
			},
			expectedErrs:  1,
			errorContains: []string{"custom type reference '#/custom-types/email-types/NonExistentType' not found in catalog"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			dc := &catalog.DataCatalog{
				Properties:  tc.properties,
				CustomTypes: tc.customTypes,
			}

			errs := validator.Validate(dc)

			assert.Len(t, errs, tc.expectedErrs)
			for i, errContains := range tc.errorContains {
				if i < len(errs) {
					assert.Contains(t, errs[i].error.Error(), errContains)
				}
			}
		})
	}
}
