package table

import (
	"reflect"

	prules "github.com/rudderlabs/rudder-iac/cli/internal/provider/rules"
	"github.com/rudderlabs/rudder-iac/cli/internal/provider/rules/funcs"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/retl/table"
	"github.com/rudderlabs/rudder-iac/cli/internal/validation/rules"
)

var validateTableSpec = func(
	_ string,
	_ string,
	_ map[string]any,
	spec table.TableSpec,
) []rules.ValidationResult {
	validationErrors, err := rules.ValidateStruct(spec, "")
	if err != nil {
		return []rules.ValidationResult{{
			Message: err.Error(),
		}}
	}

	return funcs.ParseValidationErrors(validationErrors, reflect.TypeOf(spec))
}

// NewTableSpecSyntaxValidRule validates retl-source-table specs. The kind is
// v1-only, so it matches no legacy spec versions.
func NewTableSpecSyntaxValidRule() rules.Rule {
	return prules.NewTypedRule(
		"retl/table/spec-syntax-valid",
		rules.Error,
		"retl table source spec syntax must be valid (experimental kind)",
		rules.Examples{},
		prules.NewPatternValidator(
			prules.V1VersionPatterns(table.ResourceKind),
			validateTableSpec,
		),
	)
}
