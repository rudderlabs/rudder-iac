package rules

import (
	"testing"

	"github.com/rudderlabs/rudder-iac/cli/internal/project/specs"
	"github.com/rudderlabs/rudder-iac/cli/internal/validation/rules"
	"github.com/stretchr/testify/assert"
)

func TestSpecSyntaxValidRule_Validate(t *testing.T) {
	t.Parallel()

	appliesToVersions := []string{
		specs.SpecVersionV0_1,
		specs.SpecVersionV0_1Variant,
	}

	tests := []struct {
		name           string
		ctx            *rules.ValidationContext
		expectedErrors int
		expectedRefs   []string
		expectedMsgs   []string
	}{
		{
			name: "all required fields present",
			ctx: &rules.ValidationContext{
				Kind:     "properties",
				Version:  "rudder/v1",
				Metadata: map[string]any{"name": "test"},
				Spec:     map[string]any{"properties": []any{}},
			},
			expectedErrors: 0,
			expectedRefs:   []string{},
			expectedMsgs:   []string{},
		},
		{
			name: "all required fields missing",
			ctx: &rules.ValidationContext{
				Kind:     "",
				Version:  "",
				Metadata: nil,
				Spec:     nil,
			},
			expectedErrors: 4,
			expectedRefs:   []string{"/kind", "/version", "/metadata", "/spec"},
			expectedMsgs:   []string{"'kind' is required", "'version' is required", "'metadata' is required", "'spec' is required"},
		},
		{
			name: "empty metadata map fails validation",
			ctx: &rules.ValidationContext{
				Kind:     "properties",
				Version:  "rudder/v1",
				Metadata: map[string]any{},
				Spec:     map[string]any{"properties": []any{}},
			},
			expectedErrors: 1,
			expectedRefs:   []string{"/metadata"},
			expectedMsgs:   []string{"'metadata' is required"},
		},
		{
			name: "empty spec map fails validation",
			ctx: &rules.ValidationContext{
				Kind:     "properties",
				Version:  "rudder/v1",
				Metadata: map[string]any{"name": "test"},
				Spec:     map[string]any{},
			},
			expectedErrors: 1,
			expectedRefs:   []string{"/spec"},
			expectedMsgs:   []string{"'spec' is required"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rule := NewSpecSyntaxValidRule(appliesToVersions)
			results := rule.Validate(tt.ctx)

			assert.Len(t, results, tt.expectedErrors, "unexpected number of validation errors")

			if tt.expectedErrors > 0 {
				actualRefs := make([]string, len(results))
				actualMsgs := make([]string, len(results))
				for i, r := range results {
					actualRefs[i] = r.Reference
					actualMsgs[i] = r.Message
				}

				assert.ElementsMatch(t, tt.expectedRefs, actualRefs, "references don't match")
				assert.ElementsMatch(t, tt.expectedMsgs, actualMsgs, "messages don't match")
			}
		})
	}
}

func TestSpecSyntaxValidRule_Metadata(t *testing.T) {
	t.Parallel()

	appliesToVersions := []string{
		specs.SpecVersionV0_1,
		specs.SpecVersionV0_1Variant,
	}
	rule := NewSpecSyntaxValidRule(appliesToVersions)

	assert.Equal(t, "project/spec-syntax-valid", rule.ID())
	assert.Equal(t, rules.Error, rule.Severity())
	assert.Equal(t, "spec syntax must be valid", rule.Description())
	assert.Equal(t, []string{"*"}, rule.AppliesToKinds())
	assert.Equal(t, appliesToVersions, rule.AppliesToVersions())

	examples := rule.Examples()
	assert.NotEmpty(t, examples.Valid)
	assert.NotEmpty(t, examples.Invalid)
}
