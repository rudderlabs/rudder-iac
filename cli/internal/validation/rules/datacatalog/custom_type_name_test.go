package datacatalog

import (
	"testing"

	"github.com/rudderlabs/rudder-iac/cli/internal/validation"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCustomTypeNameRule(t *testing.T) {
	rule := &CustomTypeNameRule{}

	t.Run("rule metadata", func(t *testing.T) {
		assert.Equal(t, "datacatalog/custom-types/name-format", rule.ID())
		assert.Equal(t, validation.SeverityError, rule.Severity())
		assert.Equal(t, []string{"custom-types"}, rule.AppliesTo())
	})

	t.Run("valid custom type names", func(t *testing.T) {
		testCases := []struct {
			name     string
			typeName string
		}{
			{"starts with capital, letters only", "EmailType"},
			{"with numbers", "Type123"},
			{"with underscores", "Email_Type"},
			{"with dashes", "Email-Type"},
			{"mixed characters", "My_Custom-Type123"},
			{"minimum length (3 chars)", "Abc"},
			{"maximum length (65 chars)", "ABCDEFGHIJKLMNOPQRSTUVWXYZ_abcdefghijklmnopqrstuvwxyz_123456789"},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				ctx := &validation.ValidationContext{
					Kind: "custom-types",
					Spec: map[string]any{
						"custom_types": []any{
							map[string]any{
								"id":   "test-id",
								"name": tc.typeName,
								"type": "string",
							},
						},
					},
				}
				errors := rule.Validate(ctx, nil)
				assert.Empty(t, errors, "expected no errors for name: %s", tc.typeName)
			})
		}
	})

	t.Run("invalid custom type names", func(t *testing.T) {
		testCases := []struct {
			name        string
			typeName    string
			description string
		}{
			{"starts with lowercase", "emailType", "must start with capital"},
			{"starts with number", "1Type", "must start with capital letter"},
			{"starts with underscore", "_MyType", "must start with capital letter"},
			{"starts with dash", "-MyType", "must start with capital letter"},
			{"contains spaces", "My Type", "no spaces allowed"},
			{"too short (2 chars)", "Ab", "minimum 3 characters"},
			{"single character", "A", "minimum 3 characters"},
			{"contains special chars", "My@Type", "no special characters"},
			{"contains dots", "My.Type", "no dots allowed"},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				ctx := &validation.ValidationContext{
					Kind: "custom-types",
					Spec: map[string]any{
						"custom_types": []any{
							map[string]any{
								"id":   "test-id",
								"name": tc.typeName,
								"type": "string",
							},
						},
					},
				}
				errors := rule.Validate(ctx, nil)
				require.Len(t, errors, 1, "expected 1 error for name: %s (%s)", tc.typeName, tc.description)
				assert.Contains(t, errors[0].Msg, "custom type name must start with a capital letter")
				assert.Equal(t, "name", errors[0].Fragment)
			})
		}
	})

	t.Run("multiple custom types with mixed validity", func(t *testing.T) {
		ctx := &validation.ValidationContext{
			Kind: "custom-types",
			Spec: map[string]any{
				"custom_types": []any{
					map[string]any{
						"id":   "valid-type",
						"name": "ValidType",
						"type": "string",
					},
					map[string]any{
						"id":   "invalid-type-1",
						"name": "invalidType", // starts with lowercase
						"type": "string",
					},
					map[string]any{
						"id":   "invalid-type-2",
						"name": "Ab", // too short
						"type": "string",
					},
				},
			},
		}
		errors := rule.Validate(ctx, nil)
		require.Len(t, errors, 2)
	})

	t.Run("empty name is skipped", func(t *testing.T) {
		ctx := &validation.ValidationContext{
			Kind: "custom-types",
			Spec: map[string]any{
				"custom_types": []any{
					map[string]any{
						"id":   "test-id",
						"name": "",
						"type": "string",
					},
				},
			},
		}
		errors := rule.Validate(ctx, nil)
		assert.Empty(t, errors, "empty name should be handled by required fields rule")
	})

	t.Run("missing name field is skipped", func(t *testing.T) {
		ctx := &validation.ValidationContext{
			Kind: "custom-types",
			Spec: map[string]any{
				"custom_types": []any{
					map[string]any{
						"id":   "test-id",
						"type": "string",
					},
				},
			},
		}
		errors := rule.Validate(ctx, nil)
		assert.Empty(t, errors, "missing name should be handled by required fields rule")
	})

	t.Run("invalid spec type returns empty", func(t *testing.T) {
		ctx := &validation.ValidationContext{
			Kind: "custom-types",
			Spec: "invalid",
		}
		errors := rule.Validate(ctx, nil)
		assert.Empty(t, errors)
	})

	t.Run("no custom_types field returns empty", func(t *testing.T) {
		ctx := &validation.ValidationContext{
			Kind: "custom-types",
			Spec: map[string]any{},
		}
		errors := rule.Validate(ctx, nil)
		assert.Empty(t, errors)
	})
}
