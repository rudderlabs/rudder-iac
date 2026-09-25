package connection

import (
	"testing"

	prules "github.com/rudderlabs/rudder-iac/cli/internal/provider/rules"
	retlConnection "github.com/rudderlabs/rudder-iac/cli/internal/providers/retl/connection"
	"github.com/rudderlabs/rudder-iac/cli/internal/validation/rules"
	"github.com/stretchr/testify/assert"
)

func TestConnectionCronExpressionValidRule_Metadata(t *testing.T) {
	t.Parallel()

	rule := NewConnectionCronExpressionValidRule()

	assert.Equal(t, "retl/connection/cron-expression-valid", rule.ID())
	assert.Equal(t, rules.Warning, rule.Severity())
	assert.Equal(t, "retl connection cron expressions should be checkable against the supported cron grammar", rule.Description())
	assert.Equal(t, prules.V1VersionPatterns(retlConnection.ResourceKind), rule.AppliesTo())
}

func TestConnectionCronExpressionValid(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		schedule map[string]any
		expected []rules.ValidationResult
	}{
		{
			name:     "a schedule that is not cron carries no expression to check",
			schedule: map[string]any{"type": "basic", "every_minutes": 30},
		},
		{
			name:     "a missing expression is the spec-syntax rule's",
			schedule: map[string]any{"type": "cron"},
		},
		{
			name:     "an expression local analysis accepts",
			schedule: map[string]any{"type": "cron", "cron_expression": "0 */2 * * *"},
		},
		{
			name:     "a definitely invalid expression stays an error, not a warning",
			schedule: map[string]any{"type": "cron", "cron_expression": "70 * * * *"},
		},
		{
			name:     "a too-frequent expression stays an error, not a warning",
			schedule: map[string]any{"type": "cron", "cron_expression": "*/2 * * * *"},
		},
		{
			name:     "a Quartz extension stays an error, not a warning",
			schedule: map[string]any{"type": "cron", "cron_expression": "0 0 L * *"},
		},
		{
			name:     "the six-field seconds dialect stays an error, not a warning",
			schedule: map[string]any{"type": "cron", "cron_expression": "0 0 0 * * *"},
		},
		{
			name:     "a timezone directive the analysis evaluates in UTC only",
			schedule: map[string]any{"type": "cron", "cron_expression": "CRON_TZ=Asia/Kolkata 0 * * * *"},
			expected: []rules.ValidationResult{{
				Reference: "/connections/0/config/schedule/cron_expression",
				Message:   `'cron_expression' could not be checked locally: timezone directives such as "CRON_TZ=Asia/Kolkata" are not supported; expressions are evaluated in UTC; the backend stores cron expressions unparsed and accepts this one, so verify the schedule it produces yourself`,
			}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			raw := validRawSpec()
			rawConfig(raw)["schedule"] = tt.schedule

			assert.Equal(t, tt.expected, validateConnectionsCron("", "", nil, raw))
		})
	}
}

// TestConnectionCronExpressionValidRule_Validate drives the rule through its
// public entry point, so the constructor's wiring and the engine's "/spec"
// prefixing are covered rather than only the validator behind them.
func TestConnectionCronExpressionValidRule_Validate(t *testing.T) {
	t.Parallel()

	raw := validRawSpec()
	rawConfig(raw)["schedule"] = map[string]any{"type": "cron", "cron_expression": "CRON_TZ=Asia/Kolkata 0 * * * *"}

	assert.Equal(t, []rules.ValidationResult{{
		Reference: "/spec/connections/0/config/schedule/cron_expression",
		Message:   `'cron_expression' could not be checked locally: timezone directives such as "CRON_TZ=Asia/Kolkata" are not supported; expressions are evaluated in UTC; the backend stores cron expressions unparsed and accepts this one, so verify the schedule it produces yourself`,
	}}, NewConnectionCronExpressionValidRule().Validate(specContext(raw)))
}

// TestConnectionCronExpressionValidRule_LeavesShapeErrors covers an entry the
// spec-syntax rule already fails: a value of the wrong type is that rule's error
// to report, not a reason for this one to warn about the spec as a whole.
func TestConnectionCronExpressionValidRule_LeavesShapeErrors(t *testing.T) {
	t.Parallel()

	raw := validRawSpec()
	rawConfig(raw)["schedule"] = map[string]any{"type": "basic", "every_minutes": "thirty"}

	assert.Empty(t, NewConnectionCronExpressionValidRule().Validate(specContext(raw)))
}
