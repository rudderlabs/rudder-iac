package connection

import (
	"fmt"

	prules "github.com/rudderlabs/rudder-iac/cli/internal/provider/rules"
	retlConnection "github.com/rudderlabs/rudder-iac/cli/internal/providers/retl/connection"
	"github.com/rudderlabs/rudder-iac/cli/internal/validation/rules"
)

// validateConnectionsCron reports the cron expressions local analysis could not
// reach a verdict on: a dialect this CLI does not parse, or an analysis that
// stopped short. The backend stores cron expressions verbatim and never parses
// them, so it accepts these and the apply must not be blocked — but the author
// is the only one left who can confirm the schedule. Definitely invalid and
// too-frequent expressions belong to the spec-syntax rule and are errors there.
var validateConnectionsCron = func(
	_ string,
	_ string,
	_ map[string]any,
	spec retlConnection.ConnectionsSpec,
) []rules.ValidationResult {
	var results []rules.ValidationResult
	for index, c := range spec.Connections {
		schedule := c.Config.Schedule
		if schedule.Type != "cron" || schedule.CronExpression == "" {
			continue
		}

		check := CheckCron(schedule.CronExpression)
		if check.Status != CronUnsupportedDialect && check.Status != CronInconclusive {
			continue
		}

		results = append(results, result(
			scheduleRef(index)+"/cron_expression",
			fmt.Sprintf(
				"'cron_expression' could not be checked locally: %s; the backend stores cron expressions unparsed and accepts this one, so verify the schedule it produces yourself",
				check.Reason,
			),
		))
	}
	return results
}

func NewConnectionCronExpressionValidRule() rules.Rule {
	return prules.NewTypedRule(
		"retl/connection/cron-expression-valid",
		rules.Warning,
		"retl connection cron expressions should be checkable against the supported cron grammar",
		rules.Examples{},
		prules.NewPatternValidator(
			prules.V1VersionPatterns(retlConnection.ResourceKind),
			validateConnectionsCron,
		),
	)
}
