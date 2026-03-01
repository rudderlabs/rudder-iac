package source

import (
	prules "github.com/rudderlabs/rudder-iac/cli/internal/provider/rules"
	"github.com/rudderlabs/rudder-iac/cli/internal/provider/rules/funcs"
	esSource "github.com/rudderlabs/rudder-iac/cli/internal/providers/event-stream/source"
	"github.com/rudderlabs/rudder-iac/cli/internal/validation/rules"
)

var validateSourceSpec = func(
	_ string,
	_ string,
	_ map[string]any,
	spec esSource.SourceSpec,
) []rules.ValidationResult {
	validationErrors, err := rules.ValidateStruct(spec, "")
	if err != nil {
		return []rules.ValidationResult{{
			Message: err.Error(),
		}}
	}

	return funcs.ParseValidationErrors(validationErrors, nil)
}

func NewSourceSpecSyntaxValidRule() rules.Rule {
	return prules.NewTypedRule(
		"event-stream/source/spec-syntax-valid",
		rules.Error,
		"event stream source spec syntax must be valid",
		rules.Examples{},
		prules.NewVariant(
			prules.LegacyVersionPatterns(esSource.ResourceKind),
			validateSourceSpec,
		),
	)
}
