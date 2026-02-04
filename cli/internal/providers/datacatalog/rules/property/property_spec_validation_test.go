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

func TestNewPropertySpecSyntaxValidRule_Metadata(t *testing.T) {
	t.Parallel()

	rule := NewPropertySpecSyntaxValidRule()

	assert.Equal(t, "datacatalog/properties/spec-syntax-valid", rule.ID())
	assert.Equal(t, rules.Error, rule.Severity())
	assert.Equal(t, "property spec syntax must be valid", rule.Description())
	assert.Equal(t, []string{"properties"}, rule.AppliesTo())

	examples := rule.Examples()
	assert.NotEmpty(t, examples.Valid, "Rule should have valid examples")
	assert.NotEmpty(t, examples.Invalid, "Rule should have invalid examples")
}

func TestPropertySpecSyntaxValidRule_ValidSpecs(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		spec localcatalog.PropertySpec
	}{
		{
			name: "minimal valid spec with required fields only",
			spec: localcatalog.PropertySpec{
				Properties: []localcatalog.Property{
					{
						LocalID: "user_id",
						Name:    "User ID",
					},
				},
			},
		},
		{
			name: "complete spec with all fields populated",
			spec: localcatalog.PropertySpec{
				Properties: []localcatalog.Property{
					{
						LocalID:     "user_id",
						Name:        "User ID",
						Description: "Unique identifier for the user",
						Type:        "string",
						Config:      map[string]any{"format": "uuid"},
					},
					{
						LocalID:     "email",
						Name:        "Email Address",
						Description: "User's email address",
						Type:        "string",
						Config:      map[string]any{"format": "email"},
					},
				},
			},
		},
		{
			name: "multiple valid properties",
			spec: localcatalog.PropertySpec{
				Properties: []localcatalog.Property{
					{LocalID: "user_id", Name: "User ID"},
					{LocalID: "email", Name: "Email"},
					{LocalID: "name", Name: "Full Name"},
					{LocalID: "age", Name: "Age"},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results := validatePropertySpec("properties", "rudder/v1", map[string]any{}, tt.spec)
			assert.Empty(t, results, "Valid spec should not produce validation errors")
		})
	}
}

func TestPropertySpecSyntaxValidRule_InvalidSpecs(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		spec           localcatalog.PropertySpec
		expectedErrors int
		expectedRefs   []string
		expectedMsgs   []string
	}{
		{
			name: "missing properties array",
			spec: localcatalog.PropertySpec{
				Properties: nil,
			},
			expectedErrors: 1,
			expectedRefs:   []string{"/properties"},
			expectedMsgs:   []string{"'properties' is required"},
		},
		{
			name: "property missing id",
			spec: localcatalog.PropertySpec{
				Properties: []localcatalog.Property{
					{
						Name: "User Name",
					},
				},
			},
			expectedErrors: 1,
			expectedRefs:   []string{"/properties/0/id"},
			expectedMsgs:   []string{"'id' is required"},
		},
		{
			name: "property missing name",
			spec: localcatalog.PropertySpec{
				Properties: []localcatalog.Property{
					{
						LocalID: "user_id",
					},
				},
			},
			expectedErrors: 1,
			expectedRefs:   []string{"/properties/0/name"},
			expectedMsgs:   []string{"'name' is required"},
		},
		{
			name: "property missing both id and name",
			spec: localcatalog.PropertySpec{
				Properties: []localcatalog.Property{
					{
						Description: "Some description",
						Type:        "string",
					},
				},
			},
			expectedErrors: 2,
			expectedRefs:   []string{"/properties/0/id", "/properties/0/name"},
			expectedMsgs:   []string{"'id' is required", "'name' is required"},
		},
		{
			name: "multiple properties with errors at different indices",
			spec: localcatalog.PropertySpec{
				Properties: []localcatalog.Property{
					{LocalID: "valid_id", Name: "Valid Name"},
					{Name: "Missing ID"},
					{LocalID: "missing_name"},
				},
			},
			expectedErrors: 2,
			expectedRefs:   []string{"/properties/1/id", "/properties/2/name"},
			expectedMsgs:   []string{"'id' is required", "'name' is required"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results := validatePropertySpec("properties", "rudder/v1", map[string]any{}, tt.spec)

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

func TestPropertySpecSyntaxValidRule_EdgeCases(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		spec           localcatalog.PropertySpec
		expectedErrors int
		expectedRefs   []string
		expectedMsgs   []string
	}{
		{
			name: "empty properties array is considered valid by go-validator",
			spec: localcatalog.PropertySpec{
				Properties: []localcatalog.Property{},
			},
			expectedErrors: 0,
			expectedRefs:   []string{},
			expectedMsgs:   []string{},
		},
		{
			name: "property with all fields empty",
			spec: localcatalog.PropertySpec{
				Properties: []localcatalog.Property{
					{},
				},
			},
			expectedErrors: 2,
			expectedRefs:   []string{"/properties/0/id", "/properties/0/name"},
			expectedMsgs:   []string{"'id' is required", "'name' is required"},
		},
		{
			name: "large array with error at last index",
			spec: localcatalog.PropertySpec{
				Properties: []localcatalog.Property{
					{LocalID: "prop_1", Name: "Property 1"},
					{LocalID: "prop_2", Name: "Property 2"},
					{LocalID: "prop_3", Name: "Property 3"},
					{LocalID: "prop_4", Name: "Property 4"},
					{LocalID: "prop_5", Name: "Property 5"},
					{LocalID: "prop_6", Name: "Property 6"},
					{LocalID: "prop_7", Name: "Property 7"},
					{LocalID: "prop_8", Name: "Property 8"},
					{LocalID: "prop_9", Name: "Property 9"},
					{LocalID: "prop_10", Name: "Property 10"},
					{
						Name: "Missing ID at last index",
					},
				},
			},
			expectedErrors: 1,
			expectedRefs:   []string{"/properties/10/id"},
			expectedMsgs:   []string{"'id' is required"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results := validatePropertySpec("properties", "rudder/v1", map[string]any{}, tt.spec)

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
