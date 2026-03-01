package sqlmodel

import (
	"reflect"

	prules "github.com/rudderlabs/rudder-iac/cli/internal/provider/rules"
	"github.com/rudderlabs/rudder-iac/cli/internal/provider/rules/funcs"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/retl/sqlmodel"
	"github.com/rudderlabs/rudder-iac/cli/internal/validation/rules"
)

var validateSQLModelSpec = func(
	_ string,
	_ string,
	_ map[string]any,
	spec sqlmodel.SQLModelSpec,
) []rules.ValidationResult {
	validationErrors, err := rules.ValidateStruct(spec, "")
	if err != nil {
		return []rules.ValidationResult{{
			Message: err.Error(),
		}}
	}

	return funcs.ParseValidationErrors(validationErrors, reflect.TypeOf(spec))
}

func NewSQLModelSpecSyntaxValidRule() rules.Rule {
	return prules.NewTypedRule(
		"retl/sqlmodel/spec-syntax-valid",
		rules.Error,
		"retl sql model spec syntax must be valid",
		rules.Examples{},
		prules.NewVariant(
			prules.LegacyVersionPatterns(sqlmodel.ResourceKind),
			validateSQLModelSpec,
		),
	)
}
