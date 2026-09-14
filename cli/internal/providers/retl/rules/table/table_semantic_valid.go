package table

import (
	prules "github.com/rudderlabs/rudder-iac/cli/internal/provider/rules"
	sqlmodelRules "github.com/rudderlabs/rudder-iac/cli/internal/providers/retl/rules/sqlmodel"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/retl/table"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
	"github.com/rudderlabs/rudder-iac/cli/internal/validation/rules"
)

// accountSpec is the part of a table source spec this rule reads. The kind's
// own TableSpec is decoded with mapstructure and carries no json tags for the
// typed rule's JSON round-trip.
type accountSpec struct {
	Account          string `json:"account"`
	SourceDefinition string `json:"source_definition"`
}

var validateTableSemantic = func(
	_ string,
	_ string,
	_ map[string]any,
	spec accountSpec,
	graph *resources.Graph,
) []rules.ValidationResult {
	return sqlmodelRules.ValidateAccountReference(spec.Account, spec.SourceDefinition, graph)
}

// NewTableSemanticValidRule checks a table source's account reference against
// the project. The rest of the kind's validation runs in its handler's LoadSpec.
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
