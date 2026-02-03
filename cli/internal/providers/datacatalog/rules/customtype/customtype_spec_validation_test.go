package rules

import (
	"testing"

	"github.com/rudderlabs/rudder-iac/cli/internal/providers/datacatalog/localcatalog"
	"github.com/rudderlabs/rudder-iac/cli/internal/validation/rules"
	"github.com/stretchr/testify/assert"
)

// extractRefs extracts Reference fields from ValidationResults
func extractRefs(results []rules.ValidationResult) []string {
	refs := make([]string, len(results))
	for i, result := range results {
		refs[i] = result.Reference
	}
	return refs
}

// extractMsgs extracts Message fields from ValidationResults
func extractMsgs(results []rules.ValidationResult) []string {
	msgs := make([]string, len(results))
	for i, result := range results {
		msgs[i] = result.Message
	}
	return msgs
}

func TestNewCustomTypeSpecSyntaxValidRule_Metadata(t *testing.T) {
	t.Parallel()

	rule := NewCustomTypeSpecSyntaxValidRule()

	assert.Equal(t, "datacatalog/custom-types/spec-syntax-valid", rule.ID())
	assert.Equal(t, rules.Error, rule.Severity())
	assert.Equal(t, "custom type spec syntax must be valid", rule.Description())
	assert.Equal(t, []string{"custom-types"}, rule.AppliesTo())

	examples := rule.Examples()
	assert.NotEmpty(t, examples.Valid, "Rule should have valid examples")
	assert.NotEmpty(t, examples.Invalid, "Rule should have invalid examples")
}

func TestCustomTypeSpecSyntaxValidRule_ValidSpecs(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		spec localcatalog.CustomTypeSpec
	}{
		{
			name: "minimal valid spec with required fields only",
			spec: localcatalog.CustomTypeSpec{
				Types: []localcatalog.CustomType{
					{
						LocalID: "user_status",
						Name:    "User Status",
						Type:    "string",
					},
				},
			},
		},
		{
			name: "complete spec with all fields populated",
			spec: localcatalog.CustomTypeSpec{
				Types: []localcatalog.CustomType{
					{
						LocalID:     "address",
						Name:        "Address",
						Description: "Physical address structure",
						Type:        "object",
						Config:      map[string]any{"format": "json"},
					},
					{
						LocalID:     "coordinates",
						Name:        "Coordinates",
						Description: "Geographic coordinates",
						Type:        "object",
					},
				},
			},
		},
		{
			name: "custom type with properties references",
			spec: localcatalog.CustomTypeSpec{
				Types: []localcatalog.CustomType{
					{
						LocalID: "user_profile",
						Name:    "User Profile",
						Type:    "object",
						Properties: []localcatalog.CustomTypeProperty{
							{Ref: "#/properties/user-traits/name", Required: true},
							{Ref: "#/properties/user-traits/email", Required: true},
							{Ref: "#/properties/user-traits/age", Required: false},
						},
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results := validateCustomTypeSpec("custom-types", "rudder/v1", map[string]any{}, tt.spec)
			assert.Empty(t, results, "Valid spec should not produce validation errors")
		})
	}
}

func TestCustomTypeSpecSyntaxValidRule_InvalidSpecs(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		spec           localcatalog.CustomTypeSpec
		expectedErrors int
		expectedRefs   []string
		expectedMsgs   []string
	}{
		{
			name: "missing types array",
			spec: localcatalog.CustomTypeSpec{
				Types: nil,
			},
			expectedErrors: 1,
			expectedRefs:   []string{"/types"},
			expectedMsgs:   []string{"'types' is required"},
		},
		{
			name: "custom type missing id",
			spec: localcatalog.CustomTypeSpec{
				Types: []localcatalog.CustomType{
					{
						Name: "User Status",
						Type: "string",
					},
				},
			},
			expectedErrors: 1,
			expectedRefs:   []string{"/types/0/id"},
			expectedMsgs:   []string{"'id' is required"},
		},
		{
			name: "custom type missing name",
			spec: localcatalog.CustomTypeSpec{
				Types: []localcatalog.CustomType{
					{
						LocalID: "user_status",
						Type:    "string",
					},
				},
			},
			expectedErrors: 1,
			expectedRefs:   []string{"/types/0/name"},
			expectedMsgs:   []string{"'name' is required"},
		},
		{
			name: "custom type missing type",
			spec: localcatalog.CustomTypeSpec{
				Types: []localcatalog.CustomType{
					{
						LocalID: "user_status",
						Name:    "User Status",
					},
				},
			},
			expectedErrors: 1,
			expectedRefs:   []string{"/types/0/type"},
			expectedMsgs:   []string{"'type' is required"},
		},
		{
			name: "custom type missing all required fields",
			spec: localcatalog.CustomTypeSpec{
				Types: []localcatalog.CustomType{
					{
						Description: "Some description",
					},
				},
			},
			expectedErrors: 3,
			expectedRefs:   []string{"/types/0/id", "/types/0/name", "/types/0/type"},
			expectedMsgs:   []string{"'id' is required", "'name' is required", "'type' is required"},
		},
		{
			name: "custom type with invalid type value",
			spec: localcatalog.CustomTypeSpec{
				Types: []localcatalog.CustomType{
					{
						LocalID: "status",
						Name:    "Status",
						Type:    "invalid_type",
					},
				},
			},
			expectedErrors: 1,
			expectedRefs:   []string{"/types/0/type"},
			expectedMsgs:   []string{"'type' must be a valid primitive type (string, number, integer, boolean, null, array, or object)"},
		},
		{
			name: "custom type with property missing $ref",
			spec: localcatalog.CustomTypeSpec{
				Types: []localcatalog.CustomType{
					{
						LocalID: "user_profile",
						Name:    "User Profile",
						Type:    "object",
						Properties: []localcatalog.CustomTypeProperty{
							{Required: true},
						},
					},
				},
			},
			expectedErrors: 1,
			expectedRefs:   []string{"/types/0/properties/0/$ref"},
			expectedMsgs:   []string{"'$ref' is required"},
		},
		{
			name: "custom type with invalid property reference format",
			spec: localcatalog.CustomTypeSpec{
				Types: []localcatalog.CustomType{
					{
						LocalID: "user_profile",
						Name:    "User Profile",
						Type:    "object",
						Properties: []localcatalog.CustomTypeProperty{
							{Ref: "invalid-ref-format", Required: true},
						},
					},
				},
			},
			expectedErrors: 1,
			expectedRefs:   []string{"/types/0/properties/0/$ref"},
			expectedMsgs:   []string{"'$ref' is not a valid reference format"},
		},
		{
			name: "multiple custom types with errors at different indices",
			spec: localcatalog.CustomTypeSpec{
				Types: []localcatalog.CustomType{
					{LocalID: "valid_type", Name: "Valid Type", Type: "string"},
					{Name: "Missing ID", Type: "number"},
					{LocalID: "missing_name", Type: "boolean"},
					{LocalID: "missing_type", Name: "Missing Type"},
				},
			},
			expectedErrors: 3,
			expectedRefs:   []string{"/types/1/id", "/types/2/name", "/types/3/type"},
			expectedMsgs:   []string{"'id' is required", "'name' is required", "'type' is required"},
		},
		{
			name: "description too short",
			spec: localcatalog.CustomTypeSpec{
				Types: []localcatalog.CustomType{
					{
						LocalID:     "status",
						Name:        "Status",
						Type:        "string",
						Description: "ab",
					},
				},
			},
			expectedErrors: 1,
			expectedRefs:   []string{"/types/0/description"},
			expectedMsgs:   []string{"'description' length must be greater than or equal to 3"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results := validateCustomTypeSpec("custom-types", "rudder/v1", map[string]any{}, tt.spec)

			assert.Len(t, results, tt.expectedErrors, "Unexpected number of validation errors")

			if tt.expectedErrors > 0 {
				actualRefs := extractRefs(results)
				actualMsgs := extractMsgs(results)

				assert.ElementsMatch(t, tt.expectedRefs, actualRefs, "Validation error references don't match")
				assert.ElementsMatch(t, tt.expectedMsgs, actualMsgs, "Validation error messages don't match")
			}
		})
	}
}

func TestCustomTypeSpecSyntaxValidRule_EdgeCases(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		spec           localcatalog.CustomTypeSpec
		expectedErrors int
		expectedRefs   []string
		expectedMsgs   []string
	}{
		{
			name: "empty types array is considered valid by go-validator",
			spec: localcatalog.CustomTypeSpec{
				Types: []localcatalog.CustomType{},
			},
			expectedErrors: 0,
			expectedRefs:   []string{},
			expectedMsgs:   []string{},
		},
		{
			name: "custom type with all fields empty",
			spec: localcatalog.CustomTypeSpec{
				Types: []localcatalog.CustomType{
					{},
				},
			},
			expectedErrors: 3,
			expectedRefs:   []string{"/types/0/id", "/types/0/name", "/types/0/type"},
			expectedMsgs:   []string{"'id' is required", "'name' is required", "'type' is required"},
		},
		{
			name: "custom type with empty properties array",
			spec: localcatalog.CustomTypeSpec{
				Types: []localcatalog.CustomType{
					{
						LocalID:    "user_profile",
						Name:       "User Profile",
						Type:       "object",
						Properties: []localcatalog.CustomTypeProperty{},
					},
				},
			},
			expectedErrors: 0,
			expectedRefs:   []string{},
			expectedMsgs:   []string{},
		},
		{
			name: "description at exact minimum length (3 chars)",
			spec: localcatalog.CustomTypeSpec{
				Types: []localcatalog.CustomType{
					{
						LocalID:     "status",
						Name:        "Status",
						Type:        "string",
						Description: "abc",
					},
				},
			},
			expectedErrors: 0,
			expectedRefs:   []string{},
			expectedMsgs:   []string{},
		},
		{
			name: "multiple properties with one invalid reference",
			spec: localcatalog.CustomTypeSpec{
				Types: []localcatalog.CustomType{
					{
						LocalID: "user_profile",
						Name:    "User Profile",
						Type:    "object",
						Properties: []localcatalog.CustomTypeProperty{
							{Ref: "#/properties/user-traits/name", Required: true},
							{Ref: "invalid-ref", Required: true},
							{Ref: "#/properties/user-traits/age", Required: false},
						},
					},
				},
			},
			expectedErrors: 1,
			expectedRefs:   []string{"/types/0/properties/1/$ref"},
			expectedMsgs:   []string{"'$ref' is not a valid reference format"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results := validateCustomTypeSpec("custom-types", "rudder/v1", map[string]any{}, tt.spec)

			assert.Len(t, results, tt.expectedErrors, "Unexpected number of validation errors")

			if tt.expectedErrors > 0 {
				actualRefs := extractRefs(results)
				actualMsgs := extractMsgs(results)

				assert.ElementsMatch(t, tt.expectedRefs, actualRefs, "Validation error references don't match")
				assert.ElementsMatch(t, tt.expectedMsgs, actualMsgs, "Validation error messages don't match")
			}
		})
	}
}
