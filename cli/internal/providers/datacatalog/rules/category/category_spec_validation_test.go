package rules

import (
	"testing"

	"github.com/rudderlabs/rudder-iac/cli/internal/project/specs"
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

func TestNewCategorySpecSyntaxValidRule_Metadata(t *testing.T) {
	t.Parallel()

	rule := NewCategorySpecSyntaxValidRule()

	assert.Equal(t, "datacatalog/categories/spec-syntax-valid", rule.ID())
	assert.Equal(t, rules.Error, rule.Severity())
	assert.Equal(t, "category spec syntax must be valid", rule.Description())
	assert.Equal(t, []string{"categories"}, rule.AppliesTo())

	examples := rule.Examples()
	assert.NotEmpty(t, examples.Valid, "Rule should have valid examples")
	assert.NotEmpty(t, examples.Invalid, "Rule should have invalid examples")
}

func TestCategorySpecSyntaxValidRule_ValidSpecs(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		spec localcatalog.CategorySpec
	}{
		{
			name: "single category with all required fields",
			spec: localcatalog.CategorySpec{
				Categories: []localcatalog.Category{
					{
						LocalID: "user_actions",
						Name:    "User Actions",
					},
				},
			},
		},
		{
			name: "multiple categories",
			spec: localcatalog.CategorySpec{
				Categories: []localcatalog.Category{
					{LocalID: "user_actions", Name: "User Actions"},
					{LocalID: "system_events", Name: "System Events"},
					{LocalID: "analytics", Name: "Analytics"},
				},
			},
		},
		{
			name: "empty categories array is valid",
			spec: localcatalog.CategorySpec{
				Categories: []localcatalog.Category{},
			},
		},
		{
			name: "nil categories array is valid",
			spec: localcatalog.CategorySpec{
				Categories: nil,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results := validateCategorySpec(
				"categories",
				specs.SpecVersionV0_1,
				map[string]any{},
				tt.spec,
			)
			assert.Empty(t, results, "Valid spec should not produce validation errors")
		})
	}
}

func TestCategorySpecSyntaxValidRule_InvalidSpecs(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		spec           localcatalog.CategorySpec
		expectedErrors int
		expectedRefs   []string
		expectedMsgs   []string
	}{
		{
			name: "category missing id",
			spec: localcatalog.CategorySpec{
				Categories: []localcatalog.Category{
					{
						Name: "User Actions",
					},
				},
			},
			expectedErrors: 1,
			expectedRefs:   []string{"/categories/0/id"},
			expectedMsgs:   []string{"'id' is required"},
		},
		{
			name: "category missing name",
			spec: localcatalog.CategorySpec{
				Categories: []localcatalog.Category{
					{
						LocalID: "user_actions",
					},
				},
			},
			expectedErrors: 1,
			expectedRefs:   []string{"/categories/0/name"},
			expectedMsgs:   []string{"'name' is required"},
		},
		{
			name: "category missing both id and name",
			spec: localcatalog.CategorySpec{
				Categories: []localcatalog.Category{
					{},
				},
			},
			expectedErrors: 2,
			expectedRefs:   []string{"/categories/0/id", "/categories/0/name"},
			expectedMsgs:   []string{"'id' is required", "'name' is required"},
		},
		{
			name: "multiple categories with errors at different indices",
			spec: localcatalog.CategorySpec{
				Categories: []localcatalog.Category{
					{LocalID: "valid", Name: "Valid Category"},
					{Name: "Missing ID"},
					{LocalID: "missing_name"},
				},
			},
			expectedErrors: 2,
			expectedRefs:   []string{"/categories/1/id", "/categories/2/name"},
			expectedMsgs:   []string{"'id' is required", "'name' is required"},
		},
		{
			name: "large array with error at last index",
			spec: localcatalog.CategorySpec{
				Categories: []localcatalog.Category{
					{LocalID: "cat_1", Name: "Category 1"},
					{LocalID: "cat_2", Name: "Category 2"},
					{LocalID: "cat_3", Name: "Category 3"},
					{LocalID: "cat_4", Name: "Category 4"},
					{LocalID: "cat_5", Name: "Category 5"},
					{LocalID: "cat_6", Name: "Category 6"},
					{LocalID: "cat_7", Name: "Category 7"},
					{LocalID: "cat_8", Name: "Category 8"},
					{LocalID: "cat_9", Name: "Category 9"},
					{LocalID: "cat_10", Name: "Category 10"},
					{
						Name: "Missing ID at last index",
					},
				},
			},
			expectedErrors: 1,
			expectedRefs:   []string{"/categories/10/id"},
			expectedMsgs:   []string{"'id' is required"},
		},
		{
			name: "mixed errors across multiple categories",
			spec: localcatalog.CategorySpec{
				Categories: []localcatalog.Category{
					{LocalID: "cat_1", Name: "Valid"},
					{Name: "Missing ID"},
					{LocalID: "cat_3", Name: "Valid"},
					{LocalID: "missing_name"},
					{},
				},
			},
			expectedErrors: 4,
			expectedRefs:   []string{"/categories/1/id", "/categories/3/name", "/categories/4/id", "/categories/4/name"},
			expectedMsgs:   []string{"'id' is required", "'name' is required", "'id' is required", "'name' is required"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results := validateCategorySpec(
				"categories",
				specs.SpecVersionV0_1,
				map[string]any{},
				tt.spec,
			)

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
