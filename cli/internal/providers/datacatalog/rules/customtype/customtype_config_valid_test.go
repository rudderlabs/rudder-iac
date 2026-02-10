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
			expectedMsgs:   []string{"config is not allowed for the specified type(s)"},
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
							"enum":      []any{"active", "inactive"},
							"minLength": 3,
							"maxLength": 20,
							"pattern":   "^[a-z]+$",
							"format":    "email",
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
			name: "enum with mixed type values is valid",
			spec: localcatalog.CustomTypeSpec{
				Types: []localcatalog.CustomType{
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
			name: "minLength not integer",
			spec: localcatalog.CustomTypeSpec{
				Types: []localcatalog.CustomType{
					{
						LocalID: "status",
						Name:    "Status",
						Type:    "string",
						Config: map[string]any{
							"minLength": "three",
						},
					},
				},
			},
			expectedErrors: 1,
			expectedRefs:   []string{"/types/0/config/minLength"},
			expectedMsgs:   []string{"'minLength' must be an integer"},
		},
		{
			name: "maxLength not integer",
			spec: localcatalog.CustomTypeSpec{
				Types: []localcatalog.CustomType{
					{
						LocalID: "status",
						Name:    "Status",
						Type:    "string",
						Config: map[string]any{
							"maxLength": "ten",
						},
					},
				},
			},
			expectedErrors: 1,
			expectedRefs:   []string{"/types/0/config/maxLength"},
			expectedMsgs:   []string{"'maxLength' must be an integer"},
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
			name: "enum with duplicate values",
			spec: localcatalog.CustomTypeSpec{
				Types: []localcatalog.CustomType{
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
			spec: localcatalog.CustomTypeSpec{
				Types: []localcatalog.CustomType{
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
							"enum":             []any{1.0, 2.5, 3.5, 4.0, 5.0},
							"minimum":          0.0,
							"maximum":          5.0,
							"exclusiveMinimum": 0.0,
							"exclusiveMaximum": 5.0,
							"multipleOf":       0.5,
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
			name: "enum with mixed type values is valid",
			spec: localcatalog.CustomTypeSpec{
				Types: []localcatalog.CustomType{
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
			name: "enum with duplicate values",
			spec: localcatalog.CustomTypeSpec{
				Types: []localcatalog.CustomType{
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
			spec: localcatalog.CustomTypeSpec{
				Types: []localcatalog.CustomType{
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
							"enum":       []any{18, 21, 30, 40},
							"minimum":    0,
							"maximum":    120,
							"multipleOf": 1,
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
			name: "enum with mixed type values is valid",
			spec: localcatalog.CustomTypeSpec{
				Types: []localcatalog.CustomType{
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
			expectedMsgs:   []string{"'minimum' must be an integer"},
		},
		{
			name: "enum with duplicate values",
			spec: localcatalog.CustomTypeSpec{
				Types: []localcatalog.CustomType{
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
							"itemTypes":   []any{"string"},
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
							"itemTypes": []any{"#/custom-types/user-data/address"},
						},
					},
				},
			},
			expectedErrors: 0,
		},
		{
			name: "itemTypes not array",
			spec: localcatalog.CustomTypeSpec{
				Types: []localcatalog.CustomType{
					{
						LocalID: "tags",
						Name:    "Tags",
						Type:    "array",
						Config: map[string]any{
							"itemTypes": "string",
						},
					},
				},
			},
			expectedErrors: 1,
			expectedRefs:   []string{"/types/0/config/itemTypes"},
			expectedMsgs:   []string{"'itemTypes' must be an array"},
		},
		{
			name: "itemTypes element not string",
			spec: localcatalog.CustomTypeSpec{
				Types: []localcatalog.CustomType{
					{
						LocalID: "tags",
						Name:    "Tags",
						Type:    "array",
						Config: map[string]any{
							"itemTypes": []any{123},
						},
					},
				},
			},
			expectedErrors: 1,
			expectedRefs:   []string{"/types/0/config/itemTypes/0"},
			expectedMsgs:   []string{"'123' must be a string value"},
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
							"itemTypes": []any{"#/custom-types/user-data/address", "string"},
						},
					},
				},
			},
			expectedErrors: 1,
			expectedRefs:   []string{"/types/0/config/itemTypes/0"},
			expectedMsgs:   []string{"'#/custom-types/user-data/address' custom type reference cannot be paired with other types"},
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
							"itemTypes": []any{"invalid-type"},
						},
					},
				},
			},
			expectedErrors: 1,
			expectedRefs:   []string{"/types/0/config/itemTypes/0"},
		},
		{
			name: "minItems not integer",
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
			expectedMsgs:   []string{"'minItems' must be an integer"},
		},
		{
			name: "maxItems not integer",
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
			expectedMsgs:   []string{"'maxItems' must be an integer"},
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

func TestCustomTypeConfigValidRule_BooleanType(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		spec           localcatalog.CustomTypeSpec
		expectedErrors int
		expectedRefs   []string
		expectedMsgs   []string
	}{
		{
			name: "valid boolean config with enum",
			spec: localcatalog.CustomTypeSpec{
				Types: []localcatalog.CustomType{
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
			spec: localcatalog.CustomTypeSpec{
				Types: []localcatalog.CustomType{
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
			spec: localcatalog.CustomTypeSpec{
				Types: []localcatalog.CustomType{
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
			spec: localcatalog.CustomTypeSpec{
				Types: []localcatalog.CustomType{
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

func TestCustomTypeConfigValidRule_NullType(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		spec           localcatalog.CustomTypeSpec
		expectedErrors int
		expectedRefs   []string
		expectedMsgs   []string
	}{
		{
			name: "null type with no config is valid",
			spec: localcatalog.CustomTypeSpec{
				Types: []localcatalog.CustomType{
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
			expectedErrors: 1,
			expectedRefs:   []string{"/types/0/config"},
			expectedMsgs:   []string{"config is not allowed for the specified type(s)"},
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
