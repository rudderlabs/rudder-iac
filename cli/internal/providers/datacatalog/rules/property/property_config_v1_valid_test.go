package property

import (
	"testing"

	"github.com/rudderlabs/rudder-iac/cli/internal/providers/datacatalog/localcatalog"
	"github.com/stretchr/testify/assert"
)

func TestPropertyConfigV1ValidRule_ObjectType(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		spec           localcatalog.PropertySpecV1
		expectedErrors int
		expectedRefs   []string
		expectedMsgs   []string
	}{
		{
			name: "object type with no config is valid",
			spec: localcatalog.PropertySpecV1{
				Properties: []localcatalog.PropertyV1{
					{
						LocalID: "address",
						Name:    "Address",
						Type:    "object",
					},
				},
			},
			expectedErrors: 0,
		},
		{
			name: "object type with config is invalid",
			spec: localcatalog.PropertySpecV1{
				Properties: []localcatalog.PropertyV1{
					{
						LocalID: "address",
						Name:    "Address",
						Type:    "object",
						Config: map[string]any{
							"properties": []string{"field1"},
						},
					},
				},
			},
			expectedErrors: 1,
			expectedRefs:   []string{"/properties/0/propConfig"},
			expectedMsgs:   []string{"config is not allowed for the specified type(s)"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results := validatePropertyConfigV1(
				localcatalog.KindProperties,
				"",
				map[string]any{},
				tt.spec,
			)

			assert.Len(t, results, tt.expectedErrors)

			if tt.expectedErrors > 0 {
				actualRefs := extractRefs(results)
				actualMsgs := extractMsgs(results)
				assert.ElementsMatch(t, tt.expectedRefs, actualRefs)
				assert.ElementsMatch(t, tt.expectedMsgs, actualMsgs)
			}
		})
	}
}

func TestPropertyConfigV1ValidRule_StringType(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		spec           localcatalog.PropertySpecV1
		expectedErrors int
		expectedRefs   []string
		expectedMsgs   []string
	}{
		{
			name: "valid string config with all fields",
			spec: localcatalog.PropertySpecV1{
				Properties: []localcatalog.PropertyV1{
					{
						LocalID: "email",
						Name:    "Email",
						Type:    "string",
						Config: map[string]any{
							"enum":       []any{"active@example.com", "inactive@example.com"},
							"min_length": 5,
							"max_length": 100,
							"pattern":    "^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\\.[a-zA-Z]{2,}$",
							"format":     "email",
						},
					},
				},
			},
			expectedErrors: 0,
		},
		{
			name: "enum not array",
			spec: localcatalog.PropertySpecV1{
				Properties: []localcatalog.PropertyV1{
					{
						LocalID: "status",
						Name:    "Status",
						Type:    "string",
						Config: map[string]any{
							"enum": "active",
						},
					},
				},
			},
			expectedErrors: 1,
			expectedRefs:   []string{"/properties/0/propConfig/enum"},
			expectedMsgs:   []string{"'enum' must be an array"},
		},
		{
			name: "min_length not integer",
			spec: localcatalog.PropertySpecV1{
				Properties: []localcatalog.PropertyV1{
					{
						LocalID: "username",
						Name:    "Username",
						Type:    "string",
						Config: map[string]any{
							"min_length": "five",
						},
					},
				},
			},
			expectedErrors: 1,
			expectedRefs:   []string{"/properties/0/propConfig/min_length"},
			expectedMsgs:   []string{"'min_length' must be an integer"},
		},
		{
			name: "max_length not integer",
			spec: localcatalog.PropertySpecV1{
				Properties: []localcatalog.PropertyV1{
					{
						LocalID: "description",
						Name:    "Description",
						Type:    "string",
						Config: map[string]any{
							"max_length": "hundred",
						},
					},
				},
			},
			expectedErrors: 1,
			expectedRefs:   []string{"/properties/0/propConfig/max_length"},
			expectedMsgs:   []string{"'max_length' must be an integer"},
		},
		{
			name: "pattern not string",
			spec: localcatalog.PropertySpecV1{
				Properties: []localcatalog.PropertyV1{
					{
						LocalID: "code",
						Name:    "Code",
						Type:    "string",
						Config: map[string]any{
							"pattern": 123,
						},
					},
				},
			},
			expectedErrors: 1,
			expectedRefs:   []string{"/properties/0/propConfig/pattern"},
			expectedMsgs:   []string{"'pattern' must be a string"},
		},
		{
			name: "format not string",
			spec: localcatalog.PropertySpecV1{
				Properties: []localcatalog.PropertyV1{
					{
						LocalID: "email_field",
						Name:    "EmailField",
						Type:    "string",
						Config: map[string]any{
							"format": 123,
						},
					},
				},
			},
			expectedErrors: 1,
			expectedRefs:   []string{"/properties/0/propConfig/format"},
			expectedMsgs:   []string{"'format' must be a string"},
		},
		{
			name: "format with invalid value",
			spec: localcatalog.PropertySpecV1{
				Properties: []localcatalog.PropertyV1{
					{
						LocalID: "email_field",
						Name:    "EmailField",
						Type:    "string",
						Config: map[string]any{
							"format": "invalid-format",
						},
					},
				},
			},
			expectedErrors: 1,
			expectedRefs:   []string{"/properties/0/propConfig/format"},
		},
		{
			name: "enum with duplicate values",
			spec: localcatalog.PropertySpecV1{
				Properties: []localcatalog.PropertyV1{
					{
						LocalID: "status",
						Name:    "Status",
						Type:    "string",
						Config: map[string]any{
							"enum": []any{"active", "inactive", "active"},
						},
					},
				},
			},
			expectedErrors: 1,
			expectedRefs:   []string{"/properties/0/propConfig/enum/2"},
			expectedMsgs:   []string{"'active' is a duplicate value"},
		},
		{
			name: "unknown config key",
			spec: localcatalog.PropertySpecV1{
				Properties: []localcatalog.PropertyV1{
					{
						LocalID: "status",
						Name:    "Status",
						Type:    "string",
						Config: map[string]any{
							"unknown_field": "value",
						},
					},
				},
			},
			expectedErrors: 1,
			expectedRefs:   []string{"/properties/0/propConfig/unknown_field"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results := validatePropertyConfigV1(
				localcatalog.KindProperties,
				"",
				map[string]any{},
				tt.spec,
			)

			assert.Len(t, results, tt.expectedErrors)

			if tt.expectedErrors > 0 {
				actualRefs := extractRefs(results)
				actualMsgs := extractMsgs(results)
				assert.ElementsMatch(t, tt.expectedRefs, actualRefs)
				if len(tt.expectedMsgs) > 0 {
					assert.ElementsMatch(t, tt.expectedMsgs, actualMsgs)
				}
			}
		})
	}
}

func TestPropertyConfigV1ValidRule_IntegerType(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		spec           localcatalog.PropertySpecV1
		expectedErrors int
		expectedRefs   []string
		expectedMsgs   []string
	}{
		{
			name: "valid integer config",
			spec: localcatalog.PropertySpecV1{
				Properties: []localcatalog.PropertyV1{
					{
						LocalID: "age",
						Name:    "Age",
						Type:    "integer",
						Config: map[string]any{
							"enum":        []any{18, 21, 30, 40},
							"minimum":     0,
							"maximum":     120,
							"multiple_of": 1,
						},
					},
				},
			},
			expectedErrors: 0,
		},
		{
			name: "valid integer config with float64 that are integers",
			spec: localcatalog.PropertySpecV1{
				Properties: []localcatalog.PropertyV1{
					{
						LocalID: "age",
						Name:    "Age",
						Type:    "integer",
						Config: map[string]any{
							"enum":    []any{float64(18), float64(21), float64(30)},
							"minimum": float64(0),
							"maximum": float64(120),
						},
					},
				},
			},
			expectedErrors: 0,
		},
		{
			name: "minimum not integer",
			spec: localcatalog.PropertySpecV1{
				Properties: []localcatalog.PropertyV1{
					{
						LocalID: "age",
						Name:    "Age",
						Type:    "integer",
						Config: map[string]any{
							"minimum": 0.5,
						},
					},
				},
			},
			expectedErrors: 1,
			expectedRefs:   []string{"/properties/0/propConfig/minimum"},
			expectedMsgs:   []string{"'minimum' must be an integer"},
		},
		{
			name: "enum with duplicate values",
			spec: localcatalog.PropertySpecV1{
				Properties: []localcatalog.PropertyV1{
					{
						LocalID: "count",
						Name:    "Count",
						Type:    "integer",
						Config: map[string]any{
							"enum": []any{1, 2, 1},
						},
					},
				},
			},
			expectedErrors: 1,
			expectedRefs:   []string{"/properties/0/propConfig/enum/2"},
			expectedMsgs:   []string{"'1' is a duplicate value"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results := validatePropertyConfigV1(
				localcatalog.KindProperties,
				"",
				map[string]any{},
				tt.spec,
			)

			assert.Len(t, results, tt.expectedErrors)

			if tt.expectedErrors > 0 {
				actualRefs := extractRefs(results)
				actualMsgs := extractMsgs(results)
				assert.ElementsMatch(t, tt.expectedRefs, actualRefs)
				if len(tt.expectedMsgs) > 0 {
					assert.ElementsMatch(t, tt.expectedMsgs, actualMsgs)
				}
			}
		})
	}
}

func TestPropertyConfigV1ValidRule_ArrayType(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		spec           localcatalog.PropertySpecV1
		expectedErrors int
		expectedRefs   []string
		expectedMsgs   []string
	}{
		{
			name: "valid array config with primitives",
			spec: localcatalog.PropertySpecV1{
				Properties: []localcatalog.PropertyV1{
					{
						LocalID: "tags",
						Name:    "Tags",
						Type:    "array",
						Config: map[string]any{
							"item_types":   []any{"string"},
							"min_items":    1,
							"max_items":    10,
							"unique_items": true,
						},
					},
				},
			},
			expectedErrors: 0,
		},
		{
			name: "item_types not array",
			spec: localcatalog.PropertySpecV1{
				Properties: []localcatalog.PropertyV1{
					{
						LocalID: "tags",
						Name:    "Tags",
						Type:    "array",
						Config: map[string]any{
							"item_types": "string",
						},
					},
				},
			},
			expectedErrors: 1,
			expectedRefs:   []string{"/properties/0/propConfig/item_types"},
			expectedMsgs:   []string{"'item_types' must be an array"},
		},
		{
			name: "item_types element not string",
			spec: localcatalog.PropertySpecV1{
				Properties: []localcatalog.PropertyV1{
					{
						LocalID: "tags",
						Name:    "Tags",
						Type:    "array",
						Config: map[string]any{
							"item_types": []any{123},
						},
					},
				},
			},
			expectedErrors: 1,
			expectedRefs:   []string{"/properties/0/propConfig/item_types/0"},
			expectedMsgs:   []string{"'123' must be a string value"},
		},
		{
			name: "min_items not integer",
			spec: localcatalog.PropertySpecV1{
				Properties: []localcatalog.PropertyV1{
					{
						LocalID: "tags",
						Name:    "Tags",
						Type:    "array",
						Config: map[string]any{
							"min_items": "one",
						},
					},
				},
			},
			expectedErrors: 1,
			expectedRefs:   []string{"/properties/0/propConfig/min_items"},
			expectedMsgs:   []string{"'min_items' must be an integer"},
		},
		{
			name: "unique_items not boolean",
			spec: localcatalog.PropertySpecV1{
				Properties: []localcatalog.PropertyV1{
					{
						LocalID: "tags",
						Name:    "Tags",
						Type:    "array",
						Config: map[string]any{
							"unique_items": "yes",
						},
					},
				},
			},
			expectedErrors: 1,
			expectedRefs:   []string{"/properties/0/propConfig/unique_items"},
			expectedMsgs:   []string{"'unique_items' must be a boolean"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results := validatePropertyConfigV1(
				localcatalog.KindProperties,
				"",
				map[string]any{},
				tt.spec,
			)

			assert.Len(t, results, tt.expectedErrors)

			if tt.expectedErrors > 0 {
				actualRefs := extractRefs(results)
				actualMsgs := extractMsgs(results)
				assert.ElementsMatch(t, tt.expectedRefs, actualRefs)
				if len(tt.expectedMsgs) > 0 {
					assert.ElementsMatch(t, tt.expectedMsgs, actualMsgs)
				}
			}
		})
	}
}

func TestPropertyConfigV1ValidRule_NullType(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		spec           localcatalog.PropertySpecV1
		expectedErrors int
		expectedRefs   []string
		expectedMsgs   []string
	}{
		{
			name: "null type with no config is valid",
			spec: localcatalog.PropertySpecV1{
				Properties: []localcatalog.PropertyV1{
					{
						LocalID: "empty",
						Name:    "Empty",
						Type:    "null",
					},
				},
			},
			expectedErrors: 0,
		},
		{
			name: "null type with config is invalid",
			spec: localcatalog.PropertySpecV1{
				Properties: []localcatalog.PropertyV1{
					{
						LocalID: "empty",
						Name:    "Empty",
						Type:    "null",
						Config: map[string]any{
							"any_key": "any_value",
						},
					},
				},
			},
			expectedErrors: 1,
			expectedRefs:   []string{"/properties/0/propConfig"},
			expectedMsgs:   []string{"config is not allowed for the specified type(s)"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results := validatePropertyConfigV1(
				localcatalog.KindProperties,
				"",
				map[string]any{},
				tt.spec,
			)

			assert.Len(t, results, tt.expectedErrors)

			if tt.expectedErrors > 0 {
				actualRefs := extractRefs(results)
				actualMsgs := extractMsgs(results)
				assert.ElementsMatch(t, tt.expectedRefs, actualRefs)
				assert.ElementsMatch(t, tt.expectedMsgs, actualMsgs)
			}
		})
	}
}

func TestPropertyConfigV1ValidRule_MultipleProperties(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		spec           localcatalog.PropertySpecV1
		expectedErrors int
		expectedRefs   []string
	}{
		{
			name: "mix of valid and invalid configs",
			spec: localcatalog.PropertySpecV1{
				Properties: []localcatalog.PropertyV1{
					{
						LocalID: "email",
						Name:    "Email",
						Type:    "string",
						Config: map[string]any{
							"format": "email",
						},
					},
					{
						LocalID: "address",
						Name:    "Address",
						Type:    "object",
						Config: map[string]any{
							"invalid": "config",
						},
					},
					{
						LocalID: "age",
						Name:    "Age",
						Type:    "integer",
						Config: map[string]any{
							"minimum": 0.5, // Invalid - should be integer
						},
					},
				},
			},
			expectedErrors: 2,
			expectedRefs:   []string{"/properties/1/propConfig", "/properties/2/propConfig/minimum"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results := validatePropertyConfigV1(
				localcatalog.KindProperties,
				"",
				map[string]any{},
				tt.spec,
			)

			assert.Len(t, results, tt.expectedErrors)

			if tt.expectedErrors > 0 {
				actualRefs := extractRefs(results)
				assert.ElementsMatch(t, tt.expectedRefs, actualRefs)
			}
		})
	}
}

func TestPropertyConfigV1ValidRule_EdgeCases(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		spec           localcatalog.PropertySpecV1
		expectedErrors int
	}{
		{
			name: "no config is valid",
			spec: localcatalog.PropertySpecV1{
				Properties: []localcatalog.PropertyV1{
					{
						LocalID: "username",
						Name:    "Username",
						Type:    "string",
					},
				},
			},
			expectedErrors: 0,
		},
		{
			name: "empty config map is valid",
			spec: localcatalog.PropertySpecV1{
				Properties: []localcatalog.PropertyV1{
					{
						LocalID: "username",
						Name:    "Username",
						Type:    "string",
						Config:  map[string]any{},
					},
				},
			},
			expectedErrors: 0,
		},
		{
			name: "empty type defaults to all primitive types",
			spec: localcatalog.PropertySpecV1{
				Properties: []localcatalog.PropertyV1{
					{
						LocalID: "flexible",
						Name:    "Flexible",
						Type:    "",
						Config: map[string]any{
							"enum": []any{"a", "b"},
						},
					},
				},
			},
			expectedErrors: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results := validatePropertyConfigV1(
				localcatalog.KindProperties,
				"",
				map[string]any{},
				tt.spec,
			)

			assert.Len(t, results, tt.expectedErrors)
		})
	}
}
