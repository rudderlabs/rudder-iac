package connection

import (
	"testing"

	prules "github.com/rudderlabs/rudder-iac/cli/internal/provider/rules"
	retlConnection "github.com/rudderlabs/rudder-iac/cli/internal/providers/retl/connection"
	"github.com/rudderlabs/rudder-iac/cli/internal/validation/rules"
	"github.com/samber/lo"
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
		schedule retlConnection.ScheduleSpec
		expected []rules.ValidationResult
	}{
		{
			name:     "a schedule that is not cron carries no expression to check",
			schedule: retlConnection.ScheduleSpec{Type: "basic", EveryMinutes: lo.ToPtr(30)},
		},
		{
			name:     "a missing expression is the spec-syntax rule's",
			schedule: retlConnection.ScheduleSpec{Type: "cron"},
		},
		{
			name:     "an expression local analysis accepts",
			schedule: retlConnection.ScheduleSpec{Type: "cron", CronExpression: "0 */2 * * *"},
		},
		{
			name:     "a definitely invalid expression stays an error, not a warning",
			schedule: retlConnection.ScheduleSpec{Type: "cron", CronExpression: "70 * * * *"},
		},
		{
			name:     "a too-frequent expression stays an error, not a warning",
			schedule: retlConnection.ScheduleSpec{Type: "cron", CronExpression: "*/2 * * * *"},
		},
		{
			name:     "a Quartz extension the analysis cannot reason about",
			schedule: retlConnection.ScheduleSpec{Type: "cron", CronExpression: "0 0 L * *"},
			expected: []rules.ValidationResult{{
				Reference: "/connections/0/config/schedule/cron_expression",
				Message:   `'cron_expression' could not be checked locally: day-of-month field "L": the "L" (last day) extension is not supported; the backend stores cron expressions unparsed and accepts this one, so verify the schedule it produces yourself`,
			}},
		},
		{
			name:     "the six-field seconds dialect",
			schedule: retlConnection.ScheduleSpec{Type: "cron", CronExpression: "0 0 0 * * *"},
			expected: []rules.ValidationResult{{
				Reference: "/connections/0/config/schedule/cron_expression",
				Message:   "'cron_expression' could not be checked locally: six-field expressions (leading seconds field) are not supported; use the five-field minute hour day-of-month month day-of-week form; the backend stores cron expressions unparsed and accepts this one, so verify the schedule it produces yourself",
			}},
		},
		{
			name:     "a timezone directive the analysis evaluates in UTC only",
			schedule: retlConnection.ScheduleSpec{Type: "cron", CronExpression: "CRON_TZ=Asia/Kolkata 0 * * * *"},
			expected: []rules.ValidationResult{{
				Reference: "/connections/0/config/schedule/cron_expression",
				Message:   `'cron_expression' could not be checked locally: timezone directives such as "CRON_TZ=Asia/Kolkata" are not supported; expressions are evaluated in UTC; the backend stores cron expressions unparsed and accepts this one, so verify the schedule it produces yourself`,
			}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			c := validConnection()
			c.Config.Schedule = tt.schedule
			spec := retlConnection.ConnectionsSpec{Connections: []retlConnection.ConnectionSpec{c}}

			assert.Equal(t, tt.expected, validateConnectionsCron("", "", nil, spec))
		})
	}
}

// TestConnectionCronExpressionValidRule_Validate drives the rule through its
// public entry point, so the constructor's wiring and the engine's "/spec"
// prefixing are covered rather than only the validator behind them.
func TestConnectionCronExpressionValidRule_Validate(t *testing.T) {
	t.Parallel()

	raw := validRawSpec()
	rawConfig(raw)["schedule"] = map[string]any{"type": "cron", "cron_expression": "0 0 L * *"}

	assert.Equal(t, []rules.ValidationResult{{
		Reference: "/spec/connections/0/config/schedule/cron_expression",
		Message:   `'cron_expression' could not be checked locally: day-of-month field "L": the "L" (last day) extension is not supported; the backend stores cron expressions unparsed and accepts this one, so verify the schedule it produces yourself`,
	}}, NewConnectionCronExpressionValidRule().Validate(specContext(raw)))
}
