package customtype

import (
	prules "github.com/rudderlabs/rudder-iac/cli/internal/provider/rules"
	"github.com/rudderlabs/rudder-iac/cli/internal/provider/rules/funcs"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/datacatalog/localcatalog"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
	"github.com/rudderlabs/rudder-iac/cli/internal/validation/rules"
)

var validateCustomTypeSemantic = func(_ string, _ string, _ map[string]any, spec localcatalog.CustomTypeSpec, graph *resources.Graph) []rules.ValidationResult {
	return funcs.ValidateReferences(spec, graph)
}

func NewCustomTypeSemanticValidRule() rules.Rule {
	return prules.NewSemanticTypedRule(
		"datacatalog/custom-types/semantic-valid",
		rules.Error,
		"custom type references must resolve to existing resources",
		rules.Examples{},
		[]string{localcatalog.KindCustomTypes},
		validateCustomTypeSemantic,
	)
}
