package customtype

import (
	"testing"

	"github.com/rudderlabs/rudder-iac/cli/internal/project/specs"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/datacatalog/localcatalog"
	"github.com/rudderlabs/rudder-iac/cli/internal/validation/rules"
	"github.com/stretchr/testify/assert"
)

func TestCustomTypeConfigValidRule_Metadata(t *testing.T) {
	t.Parallel()

	rule := NewCustomTypeConfigValidRule()

	assert.Equal(t, "datacatalog/custom-types/config-valid", rule.ID())
	assert.Equal(t, rules.Error, rule.Severity())
	assert.Equal(t, "custom type config must be valid for the given type", rule.Description())
	assert.Equal(t, []string{"custom-types"}, rule.AppliesTo())

	examples := rule.Examples()
	assert.NotEmpty(t, examples.Valid, "Rule should have valid examples")
	assert.NotEmpty(t, examples.Invalid, "Rule should have invalid examples")
}

func TestCustomTypeConfigValidRule_ObjectType(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		spec           localcatalog.CustomTypeSpec
		expectedErrors int
		expectedRefs   []string
		expectedMsgs   []string
	}{
		{
			name: "object type with no config is valid",
			spec: localcatalog.CustomTypeSpec{
				Types: []localcatalog.CustomType{
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
			spec: localcatalog.CustomTypeSpec{
				Types: []localcatalog.CustomType{
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
			expectedRefs:   []string{"/types/0/config"},
			expectedMsgs:   []string{"'config' is not allowed for custom type of type 'object'"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results := validateCustomTypeConfig(
				localcatalog.KindCustomTypes,
				specs.SpecVersionV0_1,
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

func TestCustomTypeConfigValidRule_StringType(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		spec           localcatalog.CustomTypeSpec
		expectedErrors int
		expectedRefs   []string
		expectedMsgs   []string
	}{
		{
			name: "valid string config with all fields",
			spec: localcatalog.CustomTypeSpec{
				Types: []localcatalog.CustomType{
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
			spec: localcatalog.CustomTypeSpec{
				Types: []localcatalog.CustomType{
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
			name: "enum with non-string values",
			spec: localcatalog.CustomTypeSpec{
				Types: []localcatalog.CustomType{
					{
						LocalID: "status",
						Name:    "Status",
						Type:    "string",
						Config: map[string]any{
							"enum": []any{"active", 123, "inactive"},
						},
					},
				},
			},
			expectedErrors: 1,
			expectedRefs:   []string{"/types/0/config/enum/1"},
			expectedMsgs:   []string{"'enum[1]' must be a string"},
		},
		{
			name: "min_length not number",
			spec: localcatalog.CustomTypeSpec{
				Types: []localcatalog.CustomType{
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
			expectedMsgs:   []string{"'min_length' must be a number"},
		},
		{
			name: "max_length not number",
			spec: localcatalog.CustomTypeSpec{
				Types: []localcatalog.CustomType{
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
			expectedMsgs:   []string{"'max_length' must be a number"},
		},
		{
			name: "pattern not string",
			spec: localcatalog.CustomTypeSpec{
				Types: []localcatalog.CustomType{
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
			spec: localcatalog.CustomTypeSpec{
				Types: []localcatalog.CustomType{
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
			spec: localcatalog.CustomTypeSpec{
				Types: []localcatalog.CustomType{
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
			name: "unknown config key",
			spec: localcatalog.CustomTypeSpec{
				Types: []localcatalog.CustomType{
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
			results := validateCustomTypeConfig(
				localcatalog.KindCustomTypes,
				specs.SpecVersionV0_1,
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

func TestCustomTypeConfigValidRule_NumberType(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		spec           localcatalog.CustomTypeSpec
		expectedErrors int
		expectedRefs   []string
		expectedMsgs   []string
	}{
		{
			name: "valid number config",
			spec: localcatalog.CustomTypeSpec{
				Types: []localcatalog.CustomType{
					{
						LocalID: "rating",
						Name:    "Rating",
						Type:    "number",
						Config: map[string]any{
							"enum":               []any{1.0, 2.5, 3.5, 4.0, 5.0},
							"minimum":            0.0,
							"maximum":            5.0,
							"exclusive_minimum":  0.0,
							"exclusive_maximum":  5.0,
							"multiple_of":        0.5,
						},
					},
				},
			},
			expectedErrors: 0,
		},
		{
			name: "enum not array",
			spec: localcatalog.CustomTypeSpec{
				Types: []localcatalog.CustomType{
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
			name: "enum with non-number values",
			spec: localcatalog.CustomTypeSpec{
				Types: []localcatalog.CustomType{
					{
						LocalID: "rating",
						Name:    "Rating",
						Type:    "number",
						Config: map[string]any{
							"enum": []any{1.0, "two", 3.0},
						},
					},
				},
			},
			expectedErrors: 1,
			expectedRefs:   []string{"/types/0/config/enum/1"},
			expectedMsgs:   []string{"'enum[1]' must be a number"},
		},
		{
			name: "minimum not number",
			spec: localcatalog.CustomTypeSpec{
				Types: []localcatalog.CustomType{
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
			name: "unknown config key",
			spec: localcatalog.CustomTypeSpec{
				Types: []localcatalog.CustomType{
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
			results := validateCustomTypeConfig(
				localcatalog.KindCustomTypes,
				specs.SpecVersionV0_1,
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

func TestCustomTypeConfigValidRule_IntegerType(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		spec           localcatalog.CustomTypeSpec
		expectedErrors int
		expectedRefs   []string
		expectedMsgs   []string
	}{
		{
			name: "valid integer config",
			spec: localcatalog.CustomTypeSpec{
				Types: []localcatalog.CustomType{
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
			spec: localcatalog.CustomTypeSpec{
				Types: []localcatalog.CustomType{
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
			name: "enum with float values",
			spec: localcatalog.CustomTypeSpec{
				Types: []localcatalog.CustomType{
					{
						LocalID: "age",
						Name:    "Age",
						Type:    "integer",
						Config: map[string]any{
							"enum": []any{18, 21.5, 30},
						},
					},
				},
			},
			expectedErrors: 1,
			expectedRefs:   []string{"/types/0/config/enum/1"},
			expectedMsgs:   []string{"'enum[1]' must be a integer"},
		},
		{
			name: "minimum not integer",
			spec: localcatalog.CustomTypeSpec{
				Types: []localcatalog.CustomType{
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
			expectedMsgs:   []string{"'minimum' must be a integer"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results := validateCustomTypeConfig(
				localcatalog.KindCustomTypes,
				specs.SpecVersionV0_1,
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

func TestCustomTypeConfigValidRule_ArrayType(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		spec           localcatalog.CustomTypeSpec
		expectedErrors int
		expectedRefs   []string
		expectedMsgs   []string
	}{
		{
			name: "valid array config with primitives",
			spec: localcatalog.CustomTypeSpec{
				Types: []localcatalog.CustomType{
					{
						LocalID: "tags",
						Name:    "Tags",
						Type:    "array",
						Config: map[string]any{
							"item_types":  []any{"string"},
							"minItems":    1,
							"maxItems":    10,
							"uniqueItems": true,
						},
					},
				},
			},
			expectedErrors: 0,
		},
		{
			name: "valid array config with custom type reference",
			spec: localcatalog.CustomTypeSpec{
				Types: []localcatalog.CustomType{
					{
						LocalID: "addresses",
						Name:    "Addresses",
						Type:    "array",
						Config: map[string]any{
							"item_types": []any{"#/custom-types/user-data/address"},
						},
					},
				},
			},
			expectedErrors: 0,
		},
		{
			name: "item_types not array",
			spec: localcatalog.CustomTypeSpec{
				Types: []localcatalog.CustomType{
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
			expectedMsgs:   []string{"'item_types' must be an array"},
		},
		{
			name: "item_types element not string",
			spec: localcatalog.CustomTypeSpec{
				Types: []localcatalog.CustomType{
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
			expectedRefs:   []string{"/types/0/config/item_types/0"},
			expectedMsgs:   []string{"'item_types[0]' must be a string value"},
		},
		{
			name: "custom type paired with other types",
			spec: localcatalog.CustomTypeSpec{
				Types: []localcatalog.CustomType{
					{
						LocalID: "mixed",
						Name:    "Mixed",
						Type:    "array",
						Config: map[string]any{
							"item_types": []any{"#/custom-types/user-data/address", "string"},
						},
					},
				},
			},
			expectedErrors: 1,
			expectedRefs:   []string{"/types/0/config/item_types/0"},
			expectedMsgs:   []string{"'item_types' containing custom type reference cannot be paired with other types"},
		},
		{
			name: "invalid primitive type",
			spec: localcatalog.CustomTypeSpec{
				Types: []localcatalog.CustomType{
					{
						LocalID: "tags",
						Name:    "Tags",
						Type:    "array",
						Config: map[string]any{
							"item_types": []any{"invalid-type"},
						},
					},
				},
			},
			expectedErrors: 1,
			expectedRefs:   []string{"/types/0/config/item_types/0"},
		},
		{
			name: "minItems not number",
			spec: localcatalog.CustomTypeSpec{
				Types: []localcatalog.CustomType{
					{
						LocalID: "tags",
						Name:    "Tags",
						Type:    "array",
						Config: map[string]any{
							"minItems": "one",
						},
					},
				},
			},
			expectedErrors: 1,
			expectedRefs:   []string{"/types/0/config/minItems"},
			expectedMsgs:   []string{"'minItems' must be a number"},
		},
		{
			name: "maxItems not number",
			spec: localcatalog.CustomTypeSpec{
				Types: []localcatalog.CustomType{
					{
						LocalID: "tags",
						Name:    "Tags",
						Type:    "array",
						Config: map[string]any{
							"maxItems": "ten",
						},
					},
				},
			},
			expectedErrors: 1,
			expectedRefs:   []string{"/types/0/config/maxItems"},
			expectedMsgs:   []string{"'maxItems' must be a number"},
		},
		{
			name: "uniqueItems not boolean",
			spec: localcatalog.CustomTypeSpec{
				Types: []localcatalog.CustomType{
					{
						LocalID: "tags",
						Name:    "Tags",
						Type:    "array",
						Config: map[string]any{
							"uniqueItems": "yes",
						},
					},
				},
			},
			expectedErrors: 1,
			expectedRefs:   []string{"/types/0/config/uniqueItems"},
			expectedMsgs:   []string{"'uniqueItems' must be a boolean"},
		},
		{
			name: "unknown config key",
			spec: localcatalog.CustomTypeSpec{
				Types: []localcatalog.CustomType{
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
			results := validateCustomTypeConfig(
				localcatalog.KindCustomTypes,
				specs.SpecVersionV0_1,
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

func TestCustomTypeConfigValidRule_BooleanNullTypes(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		spec           localcatalog.CustomTypeSpec
		expectedErrors int
	}{
		{
			name: "boolean type with config is valid",
			spec: localcatalog.CustomTypeSpec{
				Types: []localcatalog.CustomType{
					{
						LocalID: "flag",
						Name:    "Flag",
						Type:    "boolean",
						Config: map[string]any{
							"any_key": "any_value",
						},
					},
				},
			},
			expectedErrors: 0,
		},
		{
			name: "null type with config is valid",
			spec: localcatalog.CustomTypeSpec{
				Types: []localcatalog.CustomType{
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
			expectedErrors: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results := validateCustomTypeConfig(
				localcatalog.KindCustomTypes,
				specs.SpecVersionV0_1,
				map[string]any{},
				tt.spec,
			)

			assert.Len(t, results, tt.expectedErrors)
		})
	}
}

func TestCustomTypeConfigValidRule_MultipleTypes(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		spec           localcatalog.CustomTypeSpec
		expectedErrors int
		expectedRefs   []string
	}{
		{
			name: "mix of valid and invalid configs",
			spec: localcatalog.CustomTypeSpec{
				Types: []localcatalog.CustomType{
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
			expectedRefs:   []string{"/types/1/config", "/types/2/config/minimum"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results := validateCustomTypeConfig(
				localcatalog.KindCustomTypes,
				specs.SpecVersionV0_1,
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

func TestCustomTypeConfigValidRule_EdgeCases(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		spec           localcatalog.CustomTypeSpec
		expectedErrors int
	}{
		{
			name: "no config is valid",
			spec: localcatalog.CustomTypeSpec{
				Types: []localcatalog.CustomType{
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
			spec: localcatalog.CustomTypeSpec{
				Types: []localcatalog.CustomType{
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
			spec: localcatalog.CustomTypeSpec{
				Types: []localcatalog.CustomType{
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
			results := validateCustomTypeConfig(
				localcatalog.KindCustomTypes,
				specs.SpecVersionV0_1,
				map[string]any{},
				tt.spec,
			)

			assert.Len(t, results, tt.expectedErrors)
		})
	}
}
