package customtype

import (
	"testing"

	"github.com/rudderlabs/rudder-iac/cli/internal/providers/datacatalog/localcatalog"
	"github.com/stretchr/testify/assert"
)

func TestCustomTypeConfigV1ValidRule_ObjectType(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		spec           localcatalog.CustomTypeSpecV1
		expectedErrors int
		expectedRefs   []string
		expectedMsgs   []string
	}{
		{
			name: "object type with no config is valid",
			spec: localcatalog.CustomTypeSpecV1{
				Types: []localcatalog.CustomTypeV1{
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
			name: "object type with additional_properties in config is valid",
			spec: localcatalog.CustomTypeSpecV1{
				Types: []localcatalog.CustomTypeV1{
					{
						LocalID: "address",
						Name:    "Address",
						Type:    "object",
						Config: map[string]any{
							"additional_properties": true,
						},
					},
				},
			},
			expectedErrors: 0,
		},
		{
			name: "object type with additional_properties non-boolean is invalid",
			spec: localcatalog.CustomTypeSpecV1{
				Types: []localcatalog.CustomTypeV1{
					{
						LocalID: "address",
						Name:    "Address",
						Type:    "object",
						Config: map[string]any{
							"additional_properties": "true",
						},
					},
				},
			},
			expectedErrors: 1,
			expectedRefs:   []string{"/types/0/config/additional_properties"},
			expectedMsgs:   []string{"'additional_properties' must be a boolean"},
		},
		{
			name: "object type with unsupported field in config is invalid",
			spec: localcatalog.CustomTypeSpecV1{
				Types: []localcatalog.CustomTypeV1{
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
			expectedRefs:   []string{"/types/0/config/properties"},
			expectedMsgs:   []string{"'properties' is not applicable for type(s)"},
		},
		{
			name: "object type with additional_properties and unsupported field in config is invalid",
			spec: localcatalog.CustomTypeSpecV1{
				Types: []localcatalog.CustomTypeV1{
					{
						LocalID: "address",
						Name:    "Address",
						Type:    "object",
						Config: map[string]any{
							"additional_properties": false,
							"min_length":            5,
						},
					},
				},
			},
			expectedErrors: 1,
			expectedRefs:   []string{"/types/0/config/min_length"},
			expectedMsgs:   []string{"'min_length' is not applicable for type(s)"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results := validateCustomTypeConfigV1(
				localcatalog.KindCustomTypes,
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

func TestCustomTypeConfigV1ValidRule_StringType(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		spec           localcatalog.CustomTypeSpecV1
		expectedErrors int
		expectedRefs   []string
		expectedMsgs   []string
	}{
		{
			name: "valid string config with all fields",
			spec: localcatalog.CustomTypeSpecV1{
				Types: []localcatalog.CustomTypeV1{
					{
						LocalID: "status",
						Name:    "Status",
						Type:    "string",
						Config: map[string]any{
							"enum":       []any{"active", "inactive"},
							"min_length": 3,
							"max_length": 20,
							"pattern":    "^[a-z]+$",
							"format":     "email",
						},
					},
				},
			},
			expectedErrors: 0,
		},
		{
			name: "enum not array",
			spec: localcatalog.CustomTypeSpecV1{
				Types: []localcatalog.CustomTypeV1{
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
			expectedRefs:   []string{"/types/0/config/enum"},
			expectedMsgs:   []string{"'enum' must be an array"},
		},
		{
			name: "enum with mixed type values is valid",
			spec: localcatalog.CustomTypeSpecV1{
				Types: []localcatalog.CustomTypeV1{
					{
						LocalID: "status",
						Name:    "Status",
						Type:    "string",
						Config: map[string]any{
							"enum": []any{"active", 123, true, "inactive"},
						},
					},
				},
			},
			expectedErrors: 0,
		},
		{
			name: "min_length not integer",
			spec: localcatalog.CustomTypeSpecV1{
				Types: []localcatalog.CustomTypeV1{
					{
						LocalID: "status",
						Name:    "Status",
						Type:    "string",
						Config: map[string]any{
							"min_length": "three",
						},
					},
				},
			},
			expectedErrors: 1,
			expectedRefs:   []string{"/types/0/config/min_length"},
			expectedMsgs:   []string{"'min_length' must be an integer"},
		},
		{
			name: "max_length not integer",
			spec: localcatalog.CustomTypeSpecV1{
				Types: []localcatalog.CustomTypeV1{
					{
						LocalID: "status",
						Name:    "Status",
						Type:    "string",
						Config: map[string]any{
							"max_length": "ten",
						},
					},
				},
			},
			expectedErrors: 1,
			expectedRefs:   []string{"/types/0/config/max_length"},
			expectedMsgs:   []string{"'max_length' must be an integer"},
		},
		{
			name: "pattern not string",
			spec: localcatalog.CustomTypeSpecV1{
				Types: []localcatalog.CustomTypeV1{
					{
						LocalID: "status",
						Name:    "Status",
						Type:    "string",
						Config: map[string]any{
							"pattern": 123,
						},
					},
				},
			},
			expectedErrors: 1,
			expectedRefs:   []string{"/types/0/config/pattern"},
			expectedMsgs:   []string{"'pattern' must be a string"},
		},
		{
			name: "format not string",
			spec: localcatalog.CustomTypeSpecV1{
				Types: []localcatalog.CustomTypeV1{
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
			expectedRefs:   []string{"/types/0/config/format"},
			expectedMsgs:   []string{"'format' must be a string"},
		},
		{
			name: "format with invalid value",
			spec: localcatalog.CustomTypeSpecV1{
				Types: []localcatalog.CustomTypeV1{
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
			expectedRefs:   []string{"/types/0/config/format"},
		},
		{
			name: "enum with duplicate values",
			spec: localcatalog.CustomTypeSpecV1{
				Types: []localcatalog.CustomTypeV1{
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
			expectedRefs:   []string{"/types/0/config/enum/2"},
			expectedMsgs:   []string{"'active' is a duplicate value"},
		},
		{
			name: "enum with multiple duplicate values",
			spec: localcatalog.CustomTypeSpecV1{
				Types: []localcatalog.CustomTypeV1{
					{
						LocalID: "status",
						Name:    "Status",
						Type:    "string",
						Config: map[string]any{
							"enum": []any{"active", "inactive", "active", "pending", "inactive"},
						},
					},
				},
			},
			expectedErrors: 2,
			expectedRefs:   []string{"/types/0/config/enum/2", "/types/0/config/enum/4"},
			expectedMsgs:   []string{"'active' is a duplicate value", "'inactive' is a duplicate value"},
		},
		{
			name: "unknown config key",
			spec: localcatalog.CustomTypeSpecV1{
				Types: []localcatalog.CustomTypeV1{
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
			expectedRefs:   []string{"/types/0/config/unknown_field"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results := validateCustomTypeConfigV1(
				localcatalog.KindCustomTypes,
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

func TestCustomTypeConfigV1ValidRule_NumberType(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		spec           localcatalog.CustomTypeSpecV1
		expectedErrors int
		expectedRefs   []string
		expectedMsgs   []string
	}{
		{
			name: "valid number config",
			spec: localcatalog.CustomTypeSpecV1{
				Types: []localcatalog.CustomTypeV1{
					{
						LocalID: "rating",
						Name:    "Rating",
						Type:    "number",
						Config: map[string]any{
							"enum":              []any{1.0, 2.5, 3.5, 4.0, 5.0},
							"minimum":           0.0,
							"maximum":           5.0,
							"exclusive_minimum": 0.0,
							"exclusive_maximum": 5.0,
							"multiple_of":       0.5,
						},
					},
				},
			},
			expectedErrors: 0,
		},
		{
			name: "enum not array",
			spec: localcatalog.CustomTypeSpecV1{
				Types: []localcatalog.CustomTypeV1{
					{
						LocalID: "rating",
						Name:    "Rating",
						Type:    "number",
						Config: map[string]any{
							"enum": 5,
						},
					},
				},
			},
			expectedErrors: 1,
			expectedRefs:   []string{"/types/0/config/enum"},
			expectedMsgs:   []string{"'enum' must be an array"},
		},
		{
			name: "enum with mixed type values is valid",
			spec: localcatalog.CustomTypeSpecV1{
				Types: []localcatalog.CustomTypeV1{
					{
						LocalID: "rating",
						Name:    "Rating",
						Type:    "number",
						Config: map[string]any{
							"enum": []any{1.0, "two", true, 3.0},
						},
					},
				},
			},
			expectedErrors: 0,
		},
		{
			name: "minimum not number",
			spec: localcatalog.CustomTypeSpecV1{
				Types: []localcatalog.CustomTypeV1{
					{
						LocalID: "rating",
						Name:    "Rating",
						Type:    "number",
						Config: map[string]any{
							"minimum": "zero",
						},
					},
				},
			},
			expectedErrors: 1,
			expectedRefs:   []string{"/types/0/config/minimum"},
			expectedMsgs:   []string{"'minimum' must be a number"},
		},
		{
			name: "enum with duplicate values",
			spec: localcatalog.CustomTypeSpecV1{
				Types: []localcatalog.CustomTypeV1{
					{
						LocalID: "rating",
						Name:    "Rating",
						Type:    "number",
						Config: map[string]any{
							"enum": []any{1.0, 2.5, 1.0},
						},
					},
				},
			},
			expectedErrors: 1,
			expectedRefs:   []string{"/types/0/config/enum/2"},
			expectedMsgs:   []string{"'1' is a duplicate value"},
		},
		{
			name: "enum with multiple duplicate values",
			spec: localcatalog.CustomTypeSpecV1{
				Types: []localcatalog.CustomTypeV1{
					{
						LocalID: "rating",
						Name:    "Rating",
						Type:    "number",
						Config: map[string]any{
							"enum": []any{1.0, 2.5, 1.0, 3.5, 2.5},
						},
					},
				},
			},
			expectedErrors: 2,
			expectedRefs:   []string{"/types/0/config/enum/2", "/types/0/config/enum/4"},
			expectedMsgs:   []string{"'1' is a duplicate value", "'2.5' is a duplicate value"},
		},
		{
			name: "unknown config key",
			spec: localcatalog.CustomTypeSpecV1{
				Types: []localcatalog.CustomTypeV1{
					{
						LocalID: "rating",
						Name:    "Rating",
						Type:    "number",
						Config: map[string]any{
							"unknown_key": 123,
						},
					},
				},
			},
			expectedErrors: 1,
			expectedRefs:   []string{"/types/0/config/unknown_key"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results := validateCustomTypeConfigV1(
				localcatalog.KindCustomTypes,
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

func TestCustomTypeConfigV1ValidRule_IntegerType(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		spec           localcatalog.CustomTypeSpecV1
		expectedErrors int
		expectedRefs   []string
		expectedMsgs   []string
	}{
		{
			name: "valid integer config",
			spec: localcatalog.CustomTypeSpecV1{
				Types: []localcatalog.CustomTypeV1{
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
			spec: localcatalog.CustomTypeSpecV1{
				Types: []localcatalog.CustomTypeV1{
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
			name: "enum with mixed type values is valid",
			spec: localcatalog.CustomTypeSpecV1{
				Types: []localcatalog.CustomTypeV1{
					{
						LocalID: "age",
						Name:    "Age",
						Type:    "integer",
						Config: map[string]any{
							"enum": []any{18, 21.5, "thirty", true},
						},
					},
				},
			},
			expectedErrors: 0,
		},
		{
			name: "minimum not integer",
			spec: localcatalog.CustomTypeSpecV1{
				Types: []localcatalog.CustomTypeV1{
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
			expectedRefs:   []string{"/types/0/config/minimum"},
			expectedMsgs:   []string{"'minimum' must be an integer"},
		},
		{
			name: "enum with duplicate values",
			spec: localcatalog.CustomTypeSpecV1{
				Types: []localcatalog.CustomTypeV1{
					{
						LocalID: "age",
						Name:    "Age",
						Type:    "integer",
						Config: map[string]any{
							"enum": []any{18, 21, 18},
						},
					},
				},
			},
			expectedErrors: 1,
			expectedRefs:   []string{"/types/0/config/enum/2"},
			expectedMsgs:   []string{"'18' is a duplicate value"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results := validateCustomTypeConfigV1(
				localcatalog.KindCustomTypes,
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

func TestCustomTypeConfigV1ValidRule_ArrayType(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		spec           localcatalog.CustomTypeSpecV1
		expectedErrors int
		expectedRefs   []string
		expectedMsgs   []string
	}{
		{
			name: "valid array config with primitives",
			spec: localcatalog.CustomTypeSpecV1{
				Types: []localcatalog.CustomTypeV1{
					{
						LocalID:   "tags",
						Name:      "Tags",
						Type:      "array",
						ItemType:  "string",
						Config: map[string]any{
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
			name: "valid array config with multiple item types",
			spec: localcatalog.CustomTypeSpecV1{
				Types: []localcatalog.CustomTypeV1{
					{
						LocalID:   "mixed",
						Name:      "Mixed",
						Type:      "array",
						ItemTypes: []string{"string", "number"},
						Config: map[string]any{
							"min_items": 0,
							"max_items": 10,
						},
					},
				},
			},
			expectedErrors: 0,
		},
		{
			name: "item_types in config",
			spec: localcatalog.CustomTypeSpecV1{
				Types: []localcatalog.CustomTypeV1{
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
			expectedRefs:   []string{"/types/0/config/item_types"},
			expectedMsgs:   []string{"'item_types' is not applicable for type(s)"},
		},
		{
			name: "min_items not integer",
			spec: localcatalog.CustomTypeSpecV1{
				Types: []localcatalog.CustomTypeV1{
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
			expectedRefs:   []string{"/types/0/config/min_items"},
			expectedMsgs:   []string{"'min_items' must be an integer"},
		},
		{
			name: "max_items not integer",
			spec: localcatalog.CustomTypeSpecV1{
				Types: []localcatalog.CustomTypeV1{
					{
						LocalID: "tags",
						Name:    "Tags",
						Type:    "array",
						Config: map[string]any{
							"max_items": "ten",
						},
					},
				},
			},
			expectedErrors: 1,
			expectedRefs:   []string{"/types/0/config/max_items"},
			expectedMsgs:   []string{"'max_items' must be an integer"},
		},
		{
			name: "unique_items not boolean",
			spec: localcatalog.CustomTypeSpecV1{
				Types: []localcatalog.CustomTypeV1{
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
			expectedRefs:   []string{"/types/0/config/unique_items"},
			expectedMsgs:   []string{"'unique_items' must be a boolean"},
		},
		{
			name: "unknown config key",
			spec: localcatalog.CustomTypeSpecV1{
				Types: []localcatalog.CustomTypeV1{
					{
						LocalID: "tags",
						Name:    "Tags",
						Type:    "array",
						Config: map[string]any{
							"unknown_key": "value",
						},
					},
				},
			},
			expectedErrors: 1,
			expectedRefs:   []string{"/types/0/config/unknown_key"},
		},
	}

	for _, tt := range tests {
		
		t.Run(tt.name, func(t *testing.T) {
			results := validateCustomTypeConfigV1(
				localcatalog.KindCustomTypes,
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

func TestCustomTypeConfigV1ValidRule_BooleanType(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		spec           localcatalog.CustomTypeSpecV1
		expectedErrors int
		expectedRefs   []string
		expectedMsgs   []string
	}{
		{
			name: "valid boolean config with enum",
			spec: localcatalog.CustomTypeSpecV1{
				Types: []localcatalog.CustomTypeV1{
					{
						LocalID: "flag",
						Name:    "Flag",
						Type:    "boolean",
						Config: map[string]any{
							"enum": []any{true, false},
						},
					},
				},
			},
			expectedErrors: 0,
		},
		{
			name: "boolean type with non-enum config key",
			spec: localcatalog.CustomTypeSpecV1{
				Types: []localcatalog.CustomTypeV1{
					{
						LocalID: "flag",
						Name:    "Flag",
						Type:    "boolean",
						Config: map[string]any{
							"pattern": "value",
						},
					},
				},
			},
			expectedErrors: 1,
			expectedRefs:   []string{"/types/0/config/pattern"},
			expectedMsgs:   []string{"'pattern' is not applicable for type(s)"},
		},
		{
			name: "boolean enum with duplicate values",
			spec: localcatalog.CustomTypeSpecV1{
				Types: []localcatalog.CustomTypeV1{
					{
						LocalID: "flag",
						Name:    "Flag",
						Type:    "boolean",
						Config: map[string]any{
							"enum": []any{true, false, true},
						},
					},
				},
			},
			expectedErrors: 1,
			expectedRefs:   []string{"/types/0/config/enum/2"},
			expectedMsgs:   []string{"'true' is a duplicate value"},
		},
		{
			name: "boolean enum not array",
			spec: localcatalog.CustomTypeSpecV1{
				Types: []localcatalog.CustomTypeV1{
					{
						LocalID: "flag",
						Name:    "Flag",
						Type:    "boolean",
						Config: map[string]any{
							"enum": true,
						},
					},
				},
			},
			expectedErrors: 1,
			expectedRefs:   []string{"/types/0/config/enum"},
			expectedMsgs:   []string{"'enum' must be an array"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results := validateCustomTypeConfigV1(
				localcatalog.KindCustomTypes,
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

func TestCustomTypeConfigV1ValidRule_NullType(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		spec           localcatalog.CustomTypeSpecV1
		expectedErrors int
		expectedRefs   []string
		expectedMsgs   []string
	}{
		{
			name: "null type with no config is valid",
			spec: localcatalog.CustomTypeSpecV1{
				Types: []localcatalog.CustomTypeV1{
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
			spec: localcatalog.CustomTypeSpecV1{
				Types: []localcatalog.CustomTypeV1{
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
			expectedRefs:   []string{"/types/0/config"},
			expectedMsgs:   []string{"config is not allowed for the specified type(s)"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results := validateCustomTypeConfigV1(
				localcatalog.KindCustomTypes,
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

func TestCustomTypeConfigV1ValidRule_MultipleTypes(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		spec           localcatalog.CustomTypeSpecV1
		expectedErrors int
		expectedRefs   []string
	}{
		{
			name: "mix of valid and invalid configs",
			spec: localcatalog.CustomTypeSpecV1{
				Types: []localcatalog.CustomTypeV1{
					{
						LocalID: "status",
						Name:    "Status",
						Type:    "string",
						Config: map[string]any{
							"enum": []any{"active", "inactive"},
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
			expectedRefs:   []string{"/types/1/config/invalid", "/types/2/config/minimum"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results := validateCustomTypeConfigV1(
				localcatalog.KindCustomTypes,
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

func TestCustomTypeConfigV1ValidRule_EdgeCases(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		spec           localcatalog.CustomTypeSpecV1
		expectedErrors int
	}{
		{
			name: "no config is valid",
			spec: localcatalog.CustomTypeSpecV1{
				Types: []localcatalog.CustomTypeV1{
					{
						LocalID: "status",
						Name:    "Status",
						Type:    "string",
					},
				},
			},
			expectedErrors: 0,
		},
		{
			name: "empty config map is valid",
			spec: localcatalog.CustomTypeSpecV1{
				Types: []localcatalog.CustomTypeV1{
					{
						LocalID: "status",
						Name:    "Status",
						Type:    "string",
						Config:  map[string]any{},
					},
				},
			},
			expectedErrors: 0,
		},
		{
			name: "unknown type skips validation",
			spec: localcatalog.CustomTypeSpecV1{
				Types: []localcatalog.CustomTypeV1{
					{
						LocalID: "unknown",
						Name:    "Unknown",
						Type:    "unknown-type",
						Config: map[string]any{
							"anything": "goes",
						},
					},
				},
			},
			expectedErrors: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results := validateCustomTypeConfigV1(
				localcatalog.KindCustomTypes,
				"",
				map[string]any{},
				tt.spec,
			)

			assert.Len(t, results, tt.expectedErrors)
		})
	}
}
