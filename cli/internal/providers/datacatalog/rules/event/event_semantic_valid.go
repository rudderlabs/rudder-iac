package rules

import (
	prules "github.com/rudderlabs/rudder-iac/cli/internal/provider/rules"
	"github.com/rudderlabs/rudder-iac/cli/internal/provider/rules/funcs"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/datacatalog/localcatalog"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
	"github.com/rudderlabs/rudder-iac/cli/internal/validation/rules"
)

var validateEventSemantic = func(_ string, _ string, _ map[string]any, spec localcatalog.EventSpec, graph *resources.Graph) []rules.ValidationResult {
	return funcs.ValidateReferences(spec, graph)
}

func NewEventSemanticValidRule() rules.Rule {
	return prules.NewSemanticTypedRule(
		"datacatalog/events/semantic-valid",
		rules.Error,
		"event references must resolve to existing resources",
		rules.Examples{},
		[]string{localcatalog.KindEvents},
		validateEventSemantic,
	)
}
