package connection

import (
	"fmt"
	"slices"

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
//
// It reads the raw map rather than the spec type: a wrong-typed value anywhere
// in the spec would fail that decode, and the spec-syntax rule already reports
// it against the field.
var validateConnectionsCron = func(
	_ string,
	_ string,
	_ map[string]any,
	spec map[string]any,
) []rules.ValidationResult {
	var results []rules.ValidationResult

	connections, _ := spec[retlConnection.ConnectionsKey].([]any)
	for index, entry := range connections {
		c, _ := entry.(map[string]any)
		config, _ := c["config"].(map[string]any)
		schedule, _ := config["schedule"].(map[string]any)
		scheduleType, _ := schedule["type"].(string)
		expression, _ := schedule["cron_expression"].(string)

		results = append(results, cronResults(
			index, scheduleType, expression,
			[]CronStatus{CronUnsupportedDialect, CronInconclusive},
			"'cron_expression' could not be checked locally: %s; the backend stores cron expressions unparsed and accepts this one, so verify the schedule it produces yourself",
		)...)
	}
	return results
}

// cronResults reports a cron schedule's expression when its verdict is one of
// the statuses the calling rule owns, worded by format around the verdict's
// reason. Other schedule types and a missing expression are the struct tags'
// to report.
func cronResults(index int, scheduleType, expression string, owned []CronStatus, format string) []rules.ValidationResult {
	if scheduleType != "cron" || expression == "" {
		return nil
	}

	check := CheckCron(expression)
	if !slices.Contains(owned, check.Status) {
		return nil
	}

	return []rules.ValidationResult{result(
		scheduleRef(index)+"/cron_expression",
		fmt.Sprintf(format, check.Reason),
	)}
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
