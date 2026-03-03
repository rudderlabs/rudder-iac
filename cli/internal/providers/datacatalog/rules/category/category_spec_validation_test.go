package rules

import (
	"strings"
	"testing"

	prules "github.com/rudderlabs/rudder-iac/cli/internal/provider/rules"
	"github.com/rudderlabs/rudder-iac/cli/internal/project/specs"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/datacatalog/localcatalog"
	"github.com/rudderlabs/rudder-iac/cli/internal/validation/rules"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	// Trigger display_name pattern registration from parent rules package
	_ "github.com/rudderlabs/rudder-iac/cli/internal/providers/datacatalog/rules"
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

	expectedPatterns := append(
		prules.LegacyVersionPatterns("categories"),
		rules.MatchKindVersion("categories", specs.SpecVersionV1),
	)
	assert.Equal(t, expectedPatterns, rule.AppliesTo())

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
			name: "name with minimum length (3 chars)",
			spec: localcatalog.CategorySpec{
				Categories: []localcatalog.Category{
					{LocalID: "cat1", Name: "Abc"},
				},
			},
		},
		{
			name: "name with maximum length (65 chars)",
			spec: localcatalog.CategorySpec{
				Categories: []localcatalog.Category{
					{LocalID: "cat1", Name: "A" + strings.Repeat("b", 64)},
				},
			},
		},
		{
			name: "name starting with lowercase letter",
			spec: localcatalog.CategorySpec{
				Categories: []localcatalog.Category{
					{LocalID: "cat1", Name: "user actions"},
				},
			},
		},
		{
			name: "name starting with underscore",
			spec: localcatalog.CategorySpec{
				Categories: []localcatalog.Category{
					{LocalID: "cat1", Name: "_system_events"},
				},
			},
		},
		{
			name: "name with all allowed special chars",
			spec: localcatalog.CategorySpec{
				Categories: []localcatalog.Category{
					{LocalID: "cat1", Name: "User Actions, v2.0 - beta"},
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
			name: "name too short (2 chars)",
			spec: localcatalog.CategorySpec{
				Categories: []localcatalog.Category{
					{LocalID: "cat1", Name: "Ab"},
				},
			},
			expectedErrors: 1,
			expectedRefs:   []string{"/categories/0/name"},
			expectedMsgs:   []string{"'name' is not valid: must start with a letter or underscore, followed by 2-64 alphanumeric, space, comma, period, or hyphen characters"},
		},
		{
			name: "name too long (66 chars)",
			spec: localcatalog.CategorySpec{
				Categories: []localcatalog.Category{
					{LocalID: "cat1", Name: "A" + strings.Repeat("b", 65)},
				},
			},
			expectedErrors: 1,
			expectedRefs:   []string{"/categories/0/name"},
			expectedMsgs:   []string{"'name' is not valid: must start with a letter or underscore, followed by 2-64 alphanumeric, space, comma, period, or hyphen characters"},
		},
		{
			name: "name starting with digit",
			spec: localcatalog.CategorySpec{
				Categories: []localcatalog.Category{
					{LocalID: "cat1", Name: "1Invalid Name"},
				},
			},
			expectedErrors: 1,
			expectedRefs:   []string{"/categories/0/name"},
			expectedMsgs:   []string{"'name' is not valid: must start with a letter or underscore, followed by 2-64 alphanumeric, space, comma, period, or hyphen characters"},
		},
		{
			name: "name starting with special char",
			spec: localcatalog.CategorySpec{
				Categories: []localcatalog.Category{
					{LocalID: "cat1", Name: "#Invalid Name"},
				},
			},
			expectedErrors: 1,
			expectedRefs:   []string{"/categories/0/name"},
			expectedMsgs:   []string{"'name' is not valid: must start with a letter or underscore, followed by 2-64 alphanumeric, space, comma, period, or hyphen characters"},
		},
		{
			name: "name with disallowed chars",
			spec: localcatalog.CategorySpec{
				Categories: []localcatalog.Category{
					{LocalID: "cat1", Name: "Valid@Name"},
				},
			},
			expectedErrors: 1,
			expectedRefs:   []string{"/categories/0/name"},
			expectedMsgs:   []string{"'name' is not valid: must start with a letter or underscore, followed by 2-64 alphanumeric, space, comma, period, or hyphen characters"},
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

func TestCategorySpecV1SyntaxValidRule_ValidSpecs(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		spec localcatalog.CategorySpecV1
	}{
		{
			name: "single category with all required fields",
			spec: localcatalog.CategorySpecV1{
				Categories: []localcatalog.CategoryV1{
					{LocalID: "user_actions", Name: "User Actions"},
				},
			},
		},
		{
			name: "multiple categories",
			spec: localcatalog.CategorySpecV1{
				Categories: []localcatalog.CategoryV1{
					{LocalID: "user_actions", Name: "User Actions"},
					{LocalID: "system_events", Name: "System Events"},
					{LocalID: "analytics", Name: "Analytics"},
				},
			},
		},
		{
			name: "name with minimum length (3 chars)",
			spec: localcatalog.CategorySpecV1{
				Categories: []localcatalog.CategoryV1{
					{LocalID: "cat1", Name: "Abc"},
				},
			},
		},
		{
			name: "name with maximum length (65 chars)",
			spec: localcatalog.CategorySpecV1{
				Categories: []localcatalog.CategoryV1{
					{LocalID: "cat1", Name: "A" + strings.Repeat("b", 64)},
				},
			},
		},
		{
			name: "name starting with lowercase letter",
			spec: localcatalog.CategorySpecV1{
				Categories: []localcatalog.CategoryV1{
					{LocalID: "cat1", Name: "user actions"},
				},
			},
		},
		{
			name: "name starting with underscore",
			spec: localcatalog.CategorySpecV1{
				Categories: []localcatalog.CategoryV1{
					{LocalID: "cat1", Name: "_system_events"},
				},
			},
		},
		{
			name: "name with all allowed special chars",
			spec: localcatalog.CategorySpecV1{
				Categories: []localcatalog.CategoryV1{
					{LocalID: "cat1", Name: "User Actions, v2.0 - beta"},
				},
			},
		},
		{
			name: "empty categories array is valid",
			spec: localcatalog.CategorySpecV1{
				Categories: []localcatalog.CategoryV1{},
			},
		},
		{
			name: "nil categories array is valid",
			spec: localcatalog.CategorySpecV1{
				Categories: nil,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results := validateCategorySpecV1(
				"categories",
				specs.SpecVersionV1,
				map[string]any{},
				tt.spec,
			)
			assert.Empty(t, results, "Valid spec should not produce validation errors")
		})
	}
}

func TestCategorySpecV1SyntaxValidRule_InvalidSpecs(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		spec           localcatalog.CategorySpecV1
		expectedErrors int
		expectedRefs   []string
		expectedMsgs   []string
	}{
		{
			name: "category missing id",
			spec: localcatalog.CategorySpecV1{
				Categories: []localcatalog.CategoryV1{
					{Name: "User Actions"},
				},
			},
			expectedErrors: 1,
			expectedRefs:   []string{"/categories/0/id"},
			expectedMsgs:   []string{"'id' is required"},
		},
		{
			name: "category missing name",
			spec: localcatalog.CategorySpecV1{
				Categories: []localcatalog.CategoryV1{
					{LocalID: "user_actions"},
				},
			},
			expectedErrors: 1,
			expectedRefs:   []string{"/categories/0/name"},
			expectedMsgs:   []string{"'name' is required"},
		},
		{
			name: "category with whitespace-only id",
			spec: localcatalog.CategorySpecV1{
				Categories: []localcatalog.CategoryV1{
					{LocalID: "   ", Name: "User Actions"},
				},
			},
			expectedErrors: 1,
			expectedRefs:   []string{"/categories/0/id"},
			expectedMsgs:   []string{"'id' is required"},
		},
		{
			name: "category with whitespace-only name",
			spec: localcatalog.CategorySpecV1{
				Categories: []localcatalog.CategoryV1{
					{LocalID: "cat1", Name: "   "},
				},
			},
			expectedErrors: 1,
			expectedRefs:   []string{"/categories/0/name"},
			expectedMsgs:   []string{"'name' is required"},
		},
		{
			name: "name with leading whitespace",
			spec: localcatalog.CategorySpecV1{
				Categories: []localcatalog.CategoryV1{
					{LocalID: "cat1", Name: " User Actions"},
				},
			},
			expectedErrors: 1,
			expectedRefs:   []string{"/categories/0/name"},
			expectedMsgs:   []string{"'name' must not have leading or trailing whitespace"},
		},
		{
			name: "name with trailing whitespace",
			spec: localcatalog.CategorySpecV1{
				Categories: []localcatalog.CategoryV1{
					{LocalID: "cat1", Name: "User Actions "},
				},
			},
			expectedErrors: 1,
			expectedRefs:   []string{"/categories/0/name"},
			expectedMsgs:   []string{"'name' must not have leading or trailing whitespace"},
		},
		{
			name: "name with both leading and trailing whitespace",
			spec: localcatalog.CategorySpecV1{
				Categories: []localcatalog.CategoryV1{
					{LocalID: "cat1", Name: " User Actions "},
				},
			},
			expectedErrors: 1,
			expectedRefs:   []string{"/categories/0/name"},
			expectedMsgs:   []string{"'name' must not have leading or trailing whitespace"},
		},
		{
			name: "name too short (2 chars)",
			spec: localcatalog.CategorySpecV1{
				Categories: []localcatalog.CategoryV1{
					{LocalID: "cat1", Name: "Ab"},
				},
			},
			expectedErrors: 1,
			expectedRefs:   []string{"/categories/0/name"},
			expectedMsgs:   []string{"'name' is not valid: must start with a letter or underscore, followed by 2-64 alphanumeric, space, comma, period, or hyphen characters"},
		},
		{
			name: "name too long (66 chars)",
			spec: localcatalog.CategorySpecV1{
				Categories: []localcatalog.CategoryV1{
					{LocalID: "cat1", Name: "A" + strings.Repeat("b", 65)},
				},
			},
			expectedErrors: 1,
			expectedRefs:   []string{"/categories/0/name"},
			expectedMsgs:   []string{"'name' is not valid: must start with a letter or underscore, followed by 2-64 alphanumeric, space, comma, period, or hyphen characters"},
		},
		{
			name: "name starting with digit",
			spec: localcatalog.CategorySpecV1{
				Categories: []localcatalog.CategoryV1{
					{LocalID: "cat1", Name: "1Invalid Name"},
				},
			},
			expectedErrors: 1,
			expectedRefs:   []string{"/categories/0/name"},
			expectedMsgs:   []string{"'name' is not valid: must start with a letter or underscore, followed by 2-64 alphanumeric, space, comma, period, or hyphen characters"},
		},
		{
			name: "name starting with special char",
			spec: localcatalog.CategorySpecV1{
				Categories: []localcatalog.CategoryV1{
					{LocalID: "cat1", Name: "#Invalid Name"},
				},
			},
			expectedErrors: 1,
			expectedRefs:   []string{"/categories/0/name"},
			expectedMsgs:   []string{"'name' is not valid: must start with a letter or underscore, followed by 2-64 alphanumeric, space, comma, period, or hyphen characters"},
		},
		{
			name: "name with disallowed chars",
			spec: localcatalog.CategorySpecV1{
				Categories: []localcatalog.CategoryV1{
					{LocalID: "cat1", Name: "Valid@Name"},
				},
			},
			expectedErrors: 1,
			expectedRefs:   []string{"/categories/0/name"},
			expectedMsgs:   []string{"'name' is not valid: must start with a letter or underscore, followed by 2-64 alphanumeric, space, comma, period, or hyphen characters"},
		},
		{
			name: "category missing both id and name",
			spec: localcatalog.CategorySpecV1{
				Categories: []localcatalog.CategoryV1{
					{},
				},
			},
			expectedErrors: 2,
			expectedRefs:   []string{"/categories/0/id", "/categories/0/name"},
			expectedMsgs:   []string{"'id' is required", "'name' is required"},
		},
		{
			name: "multiple categories with errors at different indices",
			spec: localcatalog.CategorySpecV1{
				Categories: []localcatalog.CategoryV1{
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
			name: "mixed errors across multiple categories",
			spec: localcatalog.CategorySpecV1{
				Categories: []localcatalog.CategoryV1{
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
		{
			name: "whitespace error takes precedence over pattern error",
			spec: localcatalog.CategorySpecV1{
				Categories: []localcatalog.CategoryV1{
					{LocalID: "cat1", Name: " Ab"},
				},
			},
			expectedErrors: 1,
			expectedRefs:   []string{"/categories/0/name"},
			expectedMsgs:   []string{"'name' must not have leading or trailing whitespace"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results := validateCategorySpecV1(
				"categories",
				specs.SpecVersionV1,
				map[string]any{},
				tt.spec,
			)

			require.Len(t, results, tt.expectedErrors, "Unexpected number of validation errors")

			actualRefs := extractRefs(results)
			actualMsgs := extractMsgs(results)

			assert.ElementsMatch(t, tt.expectedRefs, actualRefs, "Validation error references don't match")
			assert.ElementsMatch(t, tt.expectedMsgs, actualMsgs, "Validation error messages don't match")
		})
	}
}
