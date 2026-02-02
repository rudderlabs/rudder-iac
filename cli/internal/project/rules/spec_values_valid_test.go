package rules

import (
	"testing"

	"github.com/rudderlabs/rudder-iac/cli/internal/project/specs"
	"github.com/rudderlabs/rudder-iac/cli/internal/validation/rules"
	"github.com/stretchr/testify/assert"
)

func TestSpecSemanticValidRule_Validate(t *testing.T) {
	t.Parallel()

	// Inject test values
	validKinds := []string{"properties", "events", "tp", "custom-types", "categories"}
	validVersions := []string{
		specs.SpecVersionV0_1,
		specs.SpecVersionV0_1Variant,
		specs.SpecVersionV1,
	}

	tests := []struct {
		name           string
		ctx            *rules.ValidationContext
		expectedErrors int
		expectedRefs   []string
	}{
		{
			name: "valid kind and version",
			ctx: &rules.ValidationContext{
				Kind:    "properties",
				Version: specs.SpecVersionV1,
			},
			expectedErrors: 0,
		},
		{
			name: "valid kind with legacy version v0.1",
			ctx: &rules.ValidationContext{
				Kind:    "events",
				Version: specs.SpecVersionV0_1,
			},
			expectedErrors: 0,
		},
		{
			name: "valid kind with legacy version variant",
			ctx: &rules.ValidationContext{
				Kind:    "tp",
				Version: specs.SpecVersionV0_1Variant,
			},
			expectedErrors: 0,
		},
		{
			name: "invalid version",
			ctx: &rules.ValidationContext{
				Kind:    "properties",
				Version: "rudder/v2",
			},
			expectedErrors: 1,
			expectedRefs:   []string{"/version"},
		},
		{
			name: "invalid kind",
			ctx: &rules.ValidationContext{
				Kind:    "unsupported-kind",
				Version: specs.SpecVersionV1,
			},
			expectedErrors: 1,
			expectedRefs:   []string{"/kind"},
		},
		{
			name: "both invalid kind and version",
			ctx: &rules.ValidationContext{
				Kind:    "invalid-kind",
				Version: "invalid-version",
			},
			expectedErrors: 2,
			expectedRefs:   []string{"/kind", "/version"},
		},
		{
			name: "wrong version format",
			ctx: &rules.ValidationContext{
				Kind:    "properties",
				Version: "v1",
			},
			expectedErrors: 1,
			expectedRefs:   []string{"/version"},
		},
		{
			name: "empty version is skipped (handled by spec_syntax_valid)",
			ctx: &rules.ValidationContext{
				Kind:    "properties",
				Version: "",
			},
			expectedErrors: 0,
		},
		{
			name: "special characters in kind",
			ctx: &rules.ValidationContext{
				Kind:    "prop@rties",
				Version: specs.SpecVersionV1,
			},
			expectedErrors: 1,
			expectedRefs:   []string{"/kind"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rule := NewSpecSemanticValidRule(validKinds, validVersions)
			results := rule.Validate(tt.ctx)

			assert.Len(t, results, tt.expectedErrors, "unexpected number of validation errors")

			if tt.expectedErrors > 0 {
				actualRefs := make([]string, len(results))
				for i, r := range results {
					actualRefs[i] = r.Reference
				}
				assert.ElementsMatch(t, tt.expectedRefs, actualRefs, "references don't match")
				for _, r := range results {
					assert.NotEmpty(t, r.Message, "error message should not be empty")
				}
			}
		})
	}
}

func TestSpecSemanticValidRule_Metadata(t *testing.T) {
	t.Parallel()

	rule := NewSpecSemanticValidRule([]string{"properties"}, []string{"rudder/v1"})

	assert.Equal(t, "project/spec-values-valid", rule.ID())
	assert.Equal(t, rules.Error, rule.Severity())
	assert.Equal(t, "spec kind and version must be valid and supported", rule.Description())
	assert.Equal(t, []string{"*"}, rule.AppliesTo())

	examples := rule.Examples()
	assert.NotEmpty(t, examples.Valid)
	assert.NotEmpty(t, examples.Invalid)
}
