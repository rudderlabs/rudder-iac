package validate

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	catalog "github.com/rudderlabs/rudder-iac/cli/internal/providers/datacatalog/localcatalog"
)

func TestPropertyArrayItemTypesValidation(t *testing.T) {
	validator := &RequiredKeysValidator{}

	testCases := []struct {
		name           string
		properties     map[catalog.EntityGroup][]catalog.Property
		expectedErrors int
		errorContains  string
	}{
		{
			name: "valid property with single custom type in itemTypes",
			properties: map[catalog.EntityGroup][]catalog.Property{
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
			properties: map[catalog.EntityGroup][]catalog.Property{
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
			properties: map[catalog.EntityGroup][]catalog.Property{
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
			properties: map[catalog.EntityGroup][]catalog.Property{
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
		properties     map[catalog.EntityGroup][]catalog.Property
		expectedErrors int
		errorContains  string
	}{
		{
			name: "valid property name without whitespace",
			properties: map[catalog.EntityGroup][]catalog.Property{
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
			properties: map[catalog.EntityGroup][]catalog.Property{
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
			properties: map[catalog.EntityGroup][]catalog.Property{
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
			properties: map[catalog.EntityGroup][]catalog.Property{
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
			properties: map[catalog.EntityGroup][]catalog.Property{
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
			properties: map[catalog.EntityGroup][]catalog.Property{
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
