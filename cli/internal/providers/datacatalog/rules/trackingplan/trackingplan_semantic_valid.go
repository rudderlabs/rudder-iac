package trackingplan

import (
	prules "github.com/rudderlabs/rudder-iac/cli/internal/provider/rules"
	"github.com/rudderlabs/rudder-iac/cli/internal/provider/rules/funcs"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/datacatalog/localcatalog"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
	"github.com/rudderlabs/rudder-iac/cli/internal/validation/rules"
)

var validateTrackingPlanSemantic = func(_ string, _ string, _ map[string]any, spec localcatalog.TrackingPlan, graph *resources.Graph) []rules.ValidationResult {
	return funcs.ValidateReferences(spec, graph)
}

func NewTrackingPlanSemanticValidRule() rules.Rule {
	return prules.NewSemanticTypedRule(
		"datacatalog/tracking-plans/semantic-valid",
		rules.Error,
		"tracking plan references must resolve to existing resources",
		rules.Examples{},
		[]string{localcatalog.KindTrackingPlans},
		validateTrackingPlanSemantic,
	)
}
