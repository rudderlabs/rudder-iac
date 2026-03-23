package sqlmodel

import (
	"fmt"

	prules "github.com/rudderlabs/rudder-iac/cli/internal/provider/rules"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/retl/sqlmodel"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
	"github.com/rudderlabs/rudder-iac/cli/internal/validation/rules"
)

var validateSQLModelSemantic = func(
	_ string,
	_ string,
	_ map[string]any,
	spec sqlmodel.SQLModelSpec,
	graph *resources.Graph,
) []rules.ValidationResult {
	return validateDisplayNameUniqueness(spec, graph)
}

func validateDisplayNameUniqueness(spec sqlmodel.SQLModelSpec, graph *resources.Graph) []rules.ValidationResult {
	countMap := make(map[string]int)
	for _, resource := range graph.ResourcesByType(sqlmodel.ResourceType) {
		data := resource.Data()
		displayName, _ := data[sqlmodel.DisplayNameKey].(string)
		countMap[displayName]++
	}

	if countMap[spec.DisplayName] > 1 {
		return []rules.ValidationResult{{
			Reference: "/display_name",
			Message:   fmt.Sprintf("duplicate display_name '%s' within kind '%s'", spec.DisplayName, sqlmodel.ResourceKind),
		}}
	}

	return nil
}

func NewSQLModelSemanticValidRule() rules.Rule {
	return prules.NewTypedRule(
		"retl/sqlmodel/semantic-valid",
		rules.Error,
		"retl sql model semantic constraints must be satisfied",
		rules.Examples{},
		prules.NewSemanticPatternValidator(
			prules.LegacyVersionPatterns(sqlmodel.ResourceKind),
			validateSQLModelSemantic,
		),
		prules.NewSemanticPatternValidator(
			prules.V1VersionPatterns(sqlmodel.ResourceKind),
			validateSQLModelSemantic,
		),
	)
}
