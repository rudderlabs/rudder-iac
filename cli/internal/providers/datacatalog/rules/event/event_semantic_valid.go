package rules

import (
	"fmt"

	prules "github.com/rudderlabs/rudder-iac/cli/internal/provider/rules"
	"github.com/rudderlabs/rudder-iac/cli/internal/provider/rules/funcs"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/datacatalog/localcatalog"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/datacatalog/types"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
	"github.com/rudderlabs/rudder-iac/cli/internal/validation/rules"
)

var validateEventSemantic = func(_ string, _ string, _ map[string]any, spec localcatalog.EventSpec, graph *resources.Graph) []rules.ValidationResult {
	results := funcs.ValidateReferences(spec, graph)

	// (name, eventType) uniqueness across the entire resource graph
	results = append(results, validateEventNameUniqueness(spec, graph)...)

	return results
}

// validateEventNameUniqueness checks that each event's (name, eventType)
// combination is unique across the entire resource graph. For track events
// the name distinguishes them; for non-track events (screen, page, group,
// identify) the name is empty (enforced by syntactic validation) so only
// one of each non-track type can exist.
func validateEventNameUniqueness(spec localcatalog.EventSpec, graph *resources.Graph) []rules.ValidationResult {
	countMap := make(map[string]int)
	for _, resource := range graph.ResourcesByType(types.EventResourceType) {
		data := resource.Data()
		var (
			name, _      = data["name"].(string)
			eventType, _ = data["eventType"].(string)
		)
		key := name + "|" + eventType
		countMap[key]++
	}

	var results []rules.ValidationResult
	for i, event := range spec.Events {
		key := event.Name + "|" + event.Type
		if countMap[key] > 1 {
			results = append(results, rules.ValidationResult{
				Reference: fmt.Sprintf("/events/%d", i),
				Message:   fmt.Sprintf("duplicate name '%s' within kind 'events'", event.Name),
			})
		}
	}

	return results
}

func NewEventSemanticValidRule() rules.Rule {
	return prules.NewTypedRule(
		"datacatalog/events/semantic-valid",
		rules.Error,
		"event references must resolve to existing resources",
		rules.Examples{},
		prules.NewSemanticVariant(
			prules.LegacyVersionPatterns(localcatalog.KindEvents),
			validateEventSemantic,
		),
	)
}
