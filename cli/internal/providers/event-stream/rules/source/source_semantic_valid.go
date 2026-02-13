package source

import (
	"fmt"

	prules "github.com/rudderlabs/rudder-iac/cli/internal/provider/rules"
	esSource "github.com/rudderlabs/rudder-iac/cli/internal/providers/event-stream/source"

	"github.com/rudderlabs/rudder-iac/cli/internal/providers/datacatalog/localcatalog"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/datacatalog/types"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
	"github.com/rudderlabs/rudder-iac/cli/internal/validation/rules"
)

var validateSourceSemantic = func(
	_ string,
	_ string,
	_ map[string]any,
	spec esSource.SourceSpec,
	graph *resources.Graph,
) []rules.ValidationResult {
	if spec.Governance == nil || spec.Governance.TrackingPlan == nil {
		return nil
	}

	ref := spec.Governance.TrackingPlan.Ref
	if ref == "" {
		return nil
	}

	matches := localcatalog.TrackingPlanRegex.FindStringSubmatch(ref)
	if len(matches) != 2 {
		return []rules.ValidationResult{{
			Reference: "/governance/validations/tracking_plan",
			Message:   fmt.Sprintf("invalid tracking plan reference format: %s", ref),
		}}
	}

	trackingPlanID := matches[1]
	urn := resources.URN(trackingPlanID, types.TrackingPlanResourceType)

	_, ok := graph.GetResource(urn)
	if !ok {
		return []rules.ValidationResult{{
			Reference: "/governance/validations/tracking_plan",
			Message:   fmt.Sprintf("tracking plan '%s' not found in the project", trackingPlanID),
		}}
	}

	return nil
}

func NewSourceSemanticValidRule() rules.Rule {
	return prules.NewSemanticTypedRule(
		"event-stream/source/semantic-valid",
		rules.Error,
		"event stream source references must resolve to existing resources",
		rules.Examples{},
		[]string{esSource.ResourceKind},
		validateSourceSemantic,
	)
}
