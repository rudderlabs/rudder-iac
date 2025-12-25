package datacatalog

import (
	"testing"

	"github.com/rudderlabs/rudder-iac/cli/internal/validation"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCategoryNameRule(t *testing.T) {
	rule := &CategoryNameRule{}

	t.Run("rule metadata", func(t *testing.T) {
		assert.Equal(t, "datacatalog/categories/name-format", rule.ID())
		assert.Equal(t, validation.SeverityError, rule.Severity())
		assert.Equal(t, []string{"categories"}, rule.AppliesTo())
	})

	t.Run("valid category names", func(t *testing.T) {
		testCases := []struct {
			name         string
			categoryName string
		}{
			{"starts with uppercase", "User Actions"},
			{"starts with lowercase", "user_actions"},
			{"starts with underscore", "_internal_category"},
			{"with numbers", "Category123"},
			{"with commas", "Actions, Events"},
			{"with periods", "User.Actions"},
			{"with dashes", "User-Actions"},
			{"with spaces", "User Actions Category"},
			{"minimum length (3 chars)", "Abc"},
			{"maximum length (65 chars)", "This_is_a_very_long_category_name_that_is_exactly_65_characters_"},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				ctx := &validation.ValidationContext{
					Kind: "categories",
					Spec: map[string]any{
						"categories": []any{
							map[string]any{
								"id":   "test-id",
								"name": tc.categoryName,
							},
						},
					},
				}
				errors := rule.Validate(ctx, nil)
				assert.Empty(t, errors, "expected no errors for name: %s", tc.categoryName)
			})
		}
	})

	t.Run("invalid category names", func(t *testing.T) {
		testCases := []struct {
			name         string
			categoryName string
			description  string
			errContains  string
		}{
			{"starts with number", "1Category", "must start with letter/underscore", "must start with a letter or underscore"},
			{"starts with special char", "@Category", "must start with letter/underscore", "must start with a letter or underscore"},
			{"too short (2 chars)", "Ab", "minimum 3 characters", "must start with a letter or underscore"},
			{"single character", "A", "minimum 3 characters", "must start with a letter or underscore"},
			{"contains special chars", "My@Category", "no @ allowed", "must start with a letter or underscore"},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				ctx := &validation.ValidationContext{
					Kind: "categories",
					Spec: map[string]any{
						"categories": []any{
							map[string]any{
								"id":   "test-id",
								"name": tc.categoryName,
							},
						},
					},
				}
				errors := rule.Validate(ctx, nil)
				require.Len(t, errors, 1, "expected 1 error for name: %s (%s)", tc.categoryName, tc.description)
				assert.Contains(t, errors[0].Msg, tc.errContains)
				assert.Equal(t, "name", errors[0].Fragment)
			})
		}
	})

	t.Run("leading whitespace", func(t *testing.T) {
		ctx := &validation.ValidationContext{
			Kind: "categories",
			Spec: map[string]any{
				"categories": []any{
					map[string]any{
						"id":   "test-id",
						"name": "  User Actions",
					},
				},
			},
		}
		errors := rule.Validate(ctx, nil)
		require.Len(t, errors, 1)
		assert.Contains(t, errors[0].Msg, "cannot have leading or trailing whitespace")
	})

	t.Run("trailing whitespace", func(t *testing.T) {
		ctx := &validation.ValidationContext{
			Kind: "categories",
			Spec: map[string]any{
				"categories": []any{
					map[string]any{
						"id":   "test-id",
						"name": "User Actions  ",
					},
				},
			},
		}
		errors := rule.Validate(ctx, nil)
		require.Len(t, errors, 1)
		assert.Contains(t, errors[0].Msg, "cannot have leading or trailing whitespace")
	})

	t.Run("multiple categories with mixed validity", func(t *testing.T) {
		ctx := &validation.ValidationContext{
			Kind: "categories",
			Spec: map[string]any{
				"categories": []any{
					map[string]any{
						"id":   "valid-cat",
						"name": "Valid Category",
					},
					map[string]any{
						"id":   "invalid-cat-1",
						"name": "1InvalidCategory", // starts with number
					},
					map[string]any{
						"id":   "invalid-cat-2",
						"name": " Leading Space", // has leading whitespace
					},
				},
			},
		}
		errors := rule.Validate(ctx, nil)
		require.Len(t, errors, 2)
	})

	t.Run("empty name is skipped", func(t *testing.T) {
		ctx := &validation.ValidationContext{
			Kind: "categories",
			Spec: map[string]any{
				"categories": []any{
					map[string]any{
						"id":   "test-id",
						"name": "",
					},
				},
			},
		}
		errors := rule.Validate(ctx, nil)
		assert.Empty(t, errors, "empty name should be handled by required fields rule")
	})

	t.Run("missing name field is skipped", func(t *testing.T) {
		ctx := &validation.ValidationContext{
			Kind: "categories",
			Spec: map[string]any{
				"categories": []any{
					map[string]any{
						"id": "test-id",
					},
				},
			},
		}
		errors := rule.Validate(ctx, nil)
		assert.Empty(t, errors, "missing name should be handled by required fields rule")
	})

	t.Run("invalid spec type returns empty", func(t *testing.T) {
		ctx := &validation.ValidationContext{
			Kind: "categories",
			Spec: "invalid",
		}
		errors := rule.Validate(ctx, nil)
		assert.Empty(t, errors)
	})

	t.Run("no categories field returns empty", func(t *testing.T) {
		ctx := &validation.ValidationContext{
			Kind: "categories",
			Spec: map[string]any{},
		}
		errors := rule.Validate(ctx, nil)
		assert.Empty(t, errors)
	})
}
