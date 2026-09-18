package table

import (
	prules "github.com/rudderlabs/rudder-iac/cli/internal/provider/rules"
	sqlmodelRules "github.com/rudderlabs/rudder-iac/cli/internal/providers/retl/rules/sqlmodel"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/retl/table"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
	"github.com/rudderlabs/rudder-iac/cli/internal/validation/rules"
)

var validateTableSemantic = func(
	_ string,
	_ string,
	_ map[string]any,
	spec table.TableSpec,
	graph *resources.Graph,
) []rules.ValidationResult {
	return sqlmodelRules.ValidateAccountReference(spec.Account, spec.SourceDefinition, graph)
}

// NewTableSemanticValidRule checks a table source's account reference against
// the project. The spec's own field rules live in retl/table/spec-syntax-valid.
func NewTableSemanticValidRule() rules.Rule {
	return prules.NewTypedRule(
		"retl/table/semantic-valid",
		rules.Error,
		"retl table source semantic constraints must be satisfied",
		rules.Examples{},
		prules.NewSemanticPatternValidator(
			prules.V1VersionPatterns(table.ResourceKind),
			validateTableSemantic,
		),
	)
}
