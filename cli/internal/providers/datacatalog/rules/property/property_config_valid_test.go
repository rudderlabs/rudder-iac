package property

import (
	"testing"

	"github.com/rudderlabs/rudder-iac/cli/internal/project/specs"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/datacatalog/localcatalog"
	catalogRules "github.com/rudderlabs/rudder-iac/cli/internal/providers/datacatalog/rules"
	"github.com/rudderlabs/rudder-iac/cli/internal/validation/rules"
	"github.com/stretchr/testify/assert"
)

func TestPropertyConfigValidRule_Metadata(t *testing.T) {
	t.Parallel()

	rule := NewPropertyConfigValidRule()

	assert.Equal(t, "datacatalog/properties/config-valid", rule.ID())
	assert.Equal(t, rules.Error, rule.Severity())
	assert.Equal(t, "property config must be valid for the given type", rule.Description())
	assert.Equal(t, []rules.MatchPattern{rules.MatchKind("properties")}, rule.AppliesTo())

	examples := rule.Examples()
	assert.NotEmpty(t, examples.Valid, "Rule should have valid examples")
	assert.NotEmpty(t, examples.Invalid, "Rule should have invalid examples")
}

func TestPropertyConfigValidRule_ObjectType(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		spec           localcatalog.PropertySpec
		expectedErrors int
		expectedRefs   []string
		expectedMsgs   []string
	}{
		{
			name: "object type with no config is valid",
			spec: localcatalog.PropertySpec{
				Properties: []localcatalog.Property{
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
			spec: localcatalog.PropertySpec{
				Properties: []localcatalog.Property{
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
			results := validatePropertyConfig(
				localcatalog.KindProperties,
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

func TestPropertyConfigValidRule_StringType(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		spec           localcatalog.PropertySpec
		expectedErrors int
		expectedRefs   []string
		expectedMsgs   []string
	}{
		{
			name: "valid string config with all fields",
			spec: localcatalog.PropertySpec{
				Properties: []localcatalog.Property{
					{
						LocalID: "email",
						Name:    "Email",
						Type:    "string",
						Config: map[string]any{
							"enum":      []any{"active@example.com", "inactive@example.com"},
							"minLength": 5,
							"maxLength": 100,
							"pattern":   "^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\\.[a-zA-Z]{2,}$",
							"format":    "email",
						},
					},
				},
			},
			expectedErrors: 0,
		},
		{
			name: "enum not array",
			spec: localcatalog.PropertySpec{
				Properties: []localcatalog.Property{
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
			name: "minLength not integer",
			spec: localcatalog.PropertySpec{
				Properties: []localcatalog.Property{
					{
						LocalID: "username",
						Name:    "Username",
						Type:    "string",
						Config: map[string]any{
							"minLength": "five",
						},
					},
				},
			},
			expectedErrors: 1,
			expectedRefs:   []string{"/properties/0/propConfig/minLength"},
			expectedMsgs:   []string{"'minLength' must be an integer"},
		},
		{
			name: "maxLength not integer",
			spec: localcatalog.PropertySpec{
				Properties: []localcatalog.Property{
					{
						LocalID: "description",
						Name:    "Description",
						Type:    "string",
						Config: map[string]any{
							"maxLength": "hundred",
						},
					},
				},
			},
			expectedErrors: 1,
			expectedRefs:   []string{"/properties/0/propConfig/maxLength"},
			expectedMsgs:   []string{"'maxLength' must be an integer"},
		},
		{
			name: "pattern not string",
			spec: localcatalog.PropertySpec{
				Properties: []localcatalog.Property{
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
			spec: localcatalog.PropertySpec{
				Properties: []localcatalog.Property{
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
			spec: localcatalog.PropertySpec{
				Properties: []localcatalog.Property{
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
			spec: localcatalog.PropertySpec{
				Properties: []localcatalog.Property{
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
			spec: localcatalog.PropertySpec{
				Properties: []localcatalog.Property{
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
			results := validatePropertyConfig(
				localcatalog.KindProperties,
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

func TestPropertyConfigValidRule_IntegerType(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		spec           localcatalog.PropertySpec
		expectedErrors int
		expectedRefs   []string
		expectedMsgs   []string
	}{
		{
			name: "valid integer config",
			spec: localcatalog.PropertySpec{
				Properties: []localcatalog.Property{
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
			spec: localcatalog.PropertySpec{
				Properties: []localcatalog.Property{
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
			spec: localcatalog.PropertySpec{
				Properties: []localcatalog.Property{
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
			spec: localcatalog.PropertySpec{
				Properties: []localcatalog.Property{
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
			results := validatePropertyConfig(
				localcatalog.KindProperties,
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

func TestPropertyConfigValidRule_ArrayType(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		spec           localcatalog.PropertySpec
		expectedErrors int
		expectedRefs   []string
		expectedMsgs   []string
	}{
		{
			name: "valid array config with primitives",
			spec: localcatalog.PropertySpec{
				Properties: []localcatalog.Property{
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
			name: "itemTypes not array",
			spec: localcatalog.PropertySpec{
				Properties: []localcatalog.Property{
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
			expectedRefs:   []string{"/properties/0/propConfig/itemTypes"},
			expectedMsgs:   []string{"'itemTypes' must be an array"},
		},
		{
			name: "itemTypes element not string",
			spec: localcatalog.PropertySpec{
				Properties: []localcatalog.Property{
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
			expectedRefs:   []string{"/properties/0/propConfig/itemTypes/0"},
			expectedMsgs:   []string{"'123' must be a string value"},
		},
		{
			name: "minItems not integer",
			spec: localcatalog.PropertySpec{
				Properties: []localcatalog.Property{
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
			expectedRefs:   []string{"/properties/0/propConfig/minItems"},
			expectedMsgs:   []string{"'minItems' must be an integer"},
		},
		{
			name: "uniqueItems not boolean",
			spec: localcatalog.PropertySpec{
				Properties: []localcatalog.Property{
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
			expectedRefs:   []string{"/properties/0/propConfig/uniqueItems"},
			expectedMsgs:   []string{"'uniqueItems' must be a boolean"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results := validatePropertyConfig(
				localcatalog.KindProperties,
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

func TestPropertyConfigValidRule_NullType(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		spec           localcatalog.PropertySpec
		expectedErrors int
		expectedRefs   []string
		expectedMsgs   []string
	}{
		{
			name: "null type with no config is valid",
			spec: localcatalog.PropertySpec{
				Properties: []localcatalog.Property{
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
			spec: localcatalog.PropertySpec{
				Properties: []localcatalog.Property{
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
			results := validatePropertyConfig(
				localcatalog.KindProperties,
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

func TestPropertyConfigValidRule_MultipleProperties(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		spec           localcatalog.PropertySpec
		expectedErrors int
		expectedRefs   []string
	}{
		{
			name: "mix of valid and invalid configs",
			spec: localcatalog.PropertySpec{
				Properties: []localcatalog.Property{
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
			results := validatePropertyConfig(
				localcatalog.KindProperties,
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

func TestPropertyConfigValidRule_EdgeCases(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		spec           localcatalog.PropertySpec
		expectedErrors int
	}{
		{
			name: "no config is valid",
			spec: localcatalog.PropertySpec{
				Properties: []localcatalog.Property{
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
			spec: localcatalog.PropertySpec{
				Properties: []localcatalog.Property{
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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results := validatePropertyConfig(
				localcatalog.KindProperties,
				specs.SpecVersionV0_1,
				map[string]any{},
				tt.spec,
			)

			assert.Len(t, results, tt.expectedErrors)
		})
	}
}

func TestParsePropertyType(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		typeStr  string
		expected []string
	}{
		{"single type", "string", []string{"string"}},
		{"multi-type", "string,null", []string{"string", "null"}},
		{"empty string", "", catalogRules.ValidPrimitiveTypes},
		{"custom type", "Address", []string{"Address"}},
		{"multi with spaces", "string, null, integer", []string{"string", "null", "integer"}},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := parsePropertyType(tc.typeStr)
			assert.Equal(t, tc.expected, result)
		})
	}
}
