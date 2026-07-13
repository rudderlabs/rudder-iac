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
var validateEventSpec = func(_ string, _ string, _ map[string]any, spec localcatalog.EventSpec) []rules.ValidationResult {
	validationErrors, err := rules.ValidateStruct(spec, "")

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
	for i, event := range spec.Events {
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

// V1 validation function for event spec, targeting EventSpecV1 / EventV1 structs.
var validateEventSpecV1 = func(_ string, _ string, _ map[string]any, spec localcatalog.EventSpecV1) []rules.ValidationResult {
	validationErrors, err := rules.ValidateStruct(spec, "")
	if err != nil {
		return []rules.ValidationResult{
			{
				Reference: "/events",
				Message:   err.Error(),
			},
		}
	}

	results := funcs.ParseValidationErrors(validationErrors, nil)

	for i, event := range spec.Events {
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
		prules.NewPatternValidator(
			prules.LegacyVersionPatterns(localcatalog.KindEvents),
			validateEventSpec,
		),
		prules.NewPatternValidator(
			prules.V1VersionPatterns(localcatalog.KindEvents),
			validateEventSpecV1,
		),
	)
}
