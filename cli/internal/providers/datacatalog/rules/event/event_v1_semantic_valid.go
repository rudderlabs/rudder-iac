package rules

import (
	"fmt"

	prules "github.com/rudderlabs/rudder-iac/cli/internal/provider/rules"
	"github.com/rudderlabs/rudder-iac/cli/internal/provider/rules/funcs"
	"github.com/rudderlabs/rudder-iac/cli/internal/project/specs"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/datacatalog/localcatalog"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/datacatalog/types"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
	"github.com/rudderlabs/rudder-iac/cli/internal/validation/rules"
)

var validateEventSemanticV1 = func(_ string, _ string, _ map[string]any, spec localcatalog.EventSpecV1, graph *resources.Graph) []rules.ValidationResult {
	results := funcs.ValidateReferences(spec, graph)

	results = append(results, validateEventNameUniquenessV1(spec, graph)...)

	return results
}

// validateEventNameUniquenessV1 checks that each event's (name, eventType)
// combination is unique across the entire resource graph.
func validateEventNameUniquenessV1(spec localcatalog.EventSpecV1, graph *resources.Graph) []rules.ValidationResult {
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

func newEventSemanticV1Validator() prules.PatternValidator {
	return prules.NewSemanticPatternValidator(
		[]rules.MatchPattern{rules.MatchKindVersion(localcatalog.KindEvents, specs.SpecVersionV1)},
		validateEventSemanticV1,
	)
}
