package rules

import (
	"fmt"
	"slices"

	prules "github.com/rudderlabs/rudder-iac/cli/internal/provider/rules"
	"github.com/rudderlabs/rudder-iac/cli/internal/provider/rules/funcs"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/datacatalog/localcatalog"
	drules "github.com/rudderlabs/rudder-iac/cli/internal/providers/datacatalog/rules"
	"github.com/rudderlabs/rudder-iac/cli/internal/validation/rules"
)

var examples = rules.Examples{
	Valid: []string{
		`events:
  - id: page_viewed
    event_type: track
	name: Page Viewed
    description: User viewed a page
  - id: product_clicked
    name: Product Clicked
    event_type: track`,
	},
	Invalid: []string{
		`events:
  - name: Missing ID
    event_type: track`,
		`events:
  - id: missing_type
    # Missing required event_type field`,
	},
}

// Main validation function for event spec
// which delegates the validation to the go-validator through struct tags.
var validateEventSpec = func(Kind string, Version string, Metadata map[string]any, Spec localcatalog.EventSpec) []rules.ValidationResult {
	validationErrors, err := rules.ValidateStruct(
		Spec,
		"",
		getValidateFuncs(Version)..., // attach custom validate funcs specially for event specs
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

func NewEventSpecSyntaxValidRule() rules.Rule {
	return prules.NewTypedRule(
		"datacatalog/events/spec-syntax-valid",
		rules.Error,
		"event spec syntax must be valid",
		examples,
		[]string{"events"},
		validateEventSpec,
	)
}

// getValidateFuncs returns custom validate functions which the event spec employs
// by adding requisite tags to fields in the spec struct
func getValidateFuncs(_ string) []rules.CustomValidateFunc {
	return nil
}
