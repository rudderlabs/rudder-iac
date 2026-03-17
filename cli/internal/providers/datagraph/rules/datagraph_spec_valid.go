package rules

import (
	"fmt"
	"reflect"
	"regexp"

	prules "github.com/rudderlabs/rudder-iac/cli/internal/provider/rules"
	"github.com/rudderlabs/rudder-iac/cli/internal/provider/rules/funcs"
	dgModel "github.com/rudderlabs/rudder-iac/cli/internal/providers/datagraph/model"
	"github.com/rudderlabs/rudder-iac/cli/internal/validation/rules"
)

var tableRefPattern = regexp.MustCompile(`^[^.]+\.[^.]+\.[^.]+$`)

var examples = rules.Examples{
	Valid: []string{
		`id: my-data-graph
account_id: wh-account-123
models:
  - id: user
    display_name: User
    type: entity
    table: db.schema.users
    primary_id: user_id`,
	},
	Invalid: []string{
		`id: my-data-graph
# Missing required account_id`,
		`id: my-data-graph
account_id: wh-123
models:
  - id: user
    display_name: User
    type: invalid
    table: db.schema.users`,
	},
}

var validateDataGraphSpec = func(_ string, _ string, _ map[string]any, spec dgModel.DataGraphSpec) []rules.ValidationResult {
	validationErrors, err := rules.ValidateStruct(spec, "")
	if err != nil {
		return []rules.ValidationResult{{
			Reference: "",
			Message:   err.Error(),
		}}
	}

	results := funcs.ParseValidationErrors(validationErrors, reflect.TypeOf(dgModel.DataGraphSpec{}))

	// Custom validation not expressible via struct tags
	for i, model := range spec.Models {
		// Table format: must be 3-part reference (catalog.schema.table)
		if model.Table != "" && !tableRefPattern.MatchString(model.Table) {
			results = append(results, rules.ValidationResult{
				Reference: fmt.Sprintf("/models/%d/table", i),
				Message:   "'table' must be a 3-part reference in the format catalog.schema.table",
			})
		}

		// Conditional required fields based on model type
		switch model.Type {
		case "entity":
			if model.PrimaryID == "" {
				results = append(results, rules.ValidationResult{
					Reference: fmt.Sprintf("/models/%d/primary_id", i),
					Message:   "'primary_id' is required for entity models",
				})
			}
		case "event":
			if model.Timestamp == "" {
				results = append(results, rules.ValidationResult{
					Reference: fmt.Sprintf("/models/%d/timestamp", i),
					Message:   "'timestamp' is required for event models",
				})
			}
		}
	}

	return results
}

func NewDataGraphSpecSyntaxValidRule() rules.Rule {
	return prules.NewTypedRule(
		"datagraph/data-graph/spec-syntax-valid",
		rules.Error,
		"data graph spec syntax must be valid",
		examples,
		prules.NewPatternValidator(
			prules.LegacyVersionPatterns("data-graph"),
			validateDataGraphSpec,
		),
		prules.NewPatternValidator(
			prules.V1VersionPatterns("data-graph"),
			validateDataGraphSpec,
		),
	)
}
