package table

import (
	"fmt"

	prules "github.com/rudderlabs/rudder-iac/cli/internal/provider/rules"
	sqlmodelRules "github.com/rudderlabs/rudder-iac/cli/internal/providers/retl/rules/sqlmodel"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/retl/sourcekeys"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/retl/sqlmodel"
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
	results := validateDisplayNameUniqueness(spec, graph)
	return append(results, sqlmodelRules.ValidateAccountReference(spec.Account, spec.SourceDefinition, graph)...)
}

// validateDisplayNameUniqueness checks the table source's display_name against
// every other RETL source, SQL models included: the control plane names both
// kinds from one namespace. retl/sqlmodel/semantic-valid checks SQL models
// only against each other, so a table/SQL model clash is reported once, here.
func validateDisplayNameUniqueness(spec table.TableSpec, graph *resources.Graph) []rules.ValidationResult {
	names := prules.NamesByKey(graph, sourcekeys.DisplayNameKey, sqlmodel.ResourceType, table.ResourceType)
	clash := prules.NameClashMessage(sourcekeys.DisplayNameKey, spec.DisplayName, names)
	if clash == "" {
		return nil
	}
	// "across RETL sources" rather than naming both kinds: a table-to-table
	// clash in a project with no SQL model would otherwise name a kind the
	// user has never used.
	return []rules.ValidationResult{{
		Reference: "/" + sourcekeys.DisplayNameKey,
		Message:   fmt.Sprintf("%s across RETL sources", clash),
	}}
}

// NewTableSemanticValidRule checks a table source's display_name and account
// reference against the project. The spec's own field rules live in
// retl/table/spec-syntax-valid.
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
