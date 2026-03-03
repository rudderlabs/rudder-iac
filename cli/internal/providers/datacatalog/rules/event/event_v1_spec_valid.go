package rules

import (
	"fmt"
	"slices"

	prules "github.com/rudderlabs/rudder-iac/cli/internal/provider/rules"
	"github.com/rudderlabs/rudder-iac/cli/internal/project/specs"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/datacatalog/localcatalog"
	drules "github.com/rudderlabs/rudder-iac/cli/internal/providers/datacatalog/rules"
	"github.com/rudderlabs/rudder-iac/cli/internal/provider/rules/funcs"
	"github.com/rudderlabs/rudder-iac/cli/internal/validation/rules"
)

var validateEventSpecV1 = func(Kind string, Version string, Metadata map[string]any, Spec localcatalog.EventSpecV1) []rules.ValidationResult {
	validationErrors, err := rules.ValidateStruct(
		Spec,
		"",
	)

	if err != nil {
		return []rules.ValidationResult{
			{
				Reference: "/events",
				Message:   err.Error(),
			},
		}
	}

	results := funcs.ParseValidationErrors(validationErrors, nil)

	// Cross-field: name validation depends on event_type.
	// Only run when event_type is valid to avoid confusing errors on top of the oneof failure.
	for i, event := range Spec.Events {
		if !slices.Contains(drules.ValidEventTypes, event.Type) {
			continue
		}

		if event.Type == "track" {
			if len(event.Name) < 1 || len(event.Name) > 64 {
				results = append(results, rules.ValidationResult{
					Reference: fmt.Sprintf("/events/%d/name", i),
					Message:   "name must be between 1 and 64 characters for track events",
				})
			}
			continue
		}

		if event.Name != "" {
			results = append(results, rules.ValidationResult{
				Reference: fmt.Sprintf("/events/%d/name", i),
				Message:   "name should be empty for non-track events",
			})
		}
	}

	return results
}

func newEventSpecV1SyntaxValidator() prules.PatternValidator {
	return prules.NewPatternValidator(
		[]rules.MatchPattern{rules.MatchKindVersion(localcatalog.KindEvents, specs.SpecVersionV1)},
		validateEventSpecV1,
	)
}
