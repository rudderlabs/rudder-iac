package rules

import (
	"fmt"

	"github.com/rudderlabs/rudder-iac/cli/internal/project/specs"
	"github.com/rudderlabs/rudder-iac/cli/internal/validation/rules"
)

// ParseSpecFunc extracts URNs from a spec via the provider's ParseSpec.
type ParseSpecFunc func(path string, s *specs.Spec) (*specs.ParsedSpec, error)

type duplicateURNRule struct {
	parseSpec ParseSpecFunc
}

func NewDuplicateURNRule(parseSpec ParseSpecFunc) rules.Rule {
	return &duplicateURNRule{parseSpec: parseSpec}
}

func (r *duplicateURNRule) ID() string               { return "project/duplicate-urn" }
func (r *duplicateURNRule) Severity() rules.Severity { return rules.Error }
func (r *duplicateURNRule) Description() string {
	return "URNs must be unique across the project"
}
func (r *duplicateURNRule) AppliesTo() []string { return []string{"*"} }
func (r *duplicateURNRule) Examples() rules.Examples {
	return rules.Examples{}
}

// Validate is a no-op — this rule operates as a ProjectRule via ValidateProject.
func (r *duplicateURNRule) Validate(_ *rules.ValidationContext) []rules.ValidationResult {
	return nil
}

// ValidateProject checks for duplicate URNs across all specs.
// URNs encode the resource type, so duplicates are only flagged when both
// the type and local ID match. This correctly allows the same local ID
// across different resource types.
func (r *duplicateURNRule) ValidateProject(allSpecs map[string]*rules.ValidationContext) map[string][]rules.ValidationResult {
	type occurrence struct {
		filePath string
		jsonPath string
	}

	// urn -> all occurrences
	occurrences := make(map[string][]occurrence)

	for filePath, ctx := range allSpecs {
		parsed, err := r.parseSpec(filePath, &specs.Spec{
			Version:  ctx.Version,
			Kind:     ctx.Kind,
			Metadata: ctx.Metadata,
			Spec:     ctx.Spec,
		})
		if err != nil {
			// ParseSpec failure on a syntactically valid spec is unexpected.
			// Skip this spec — other validation rules
			// will catch structural issues.
			continue
		}

		for _, urnEntry := range parsed.URNs {
			occurrences[urnEntry.URN] = append(
				occurrences[urnEntry.URN],
				occurrence{
					filePath: filePath,
					jsonPath: urnEntry.JSONPointerPath,
				},
			)
		}
	}

	results := make(map[string][]rules.ValidationResult)

	for urn, locations := range occurrences {
		if len(locations) <= 1 {
			// No duplicates found for this URN
			continue
		}

		for _, loc := range locations {
			results[loc.filePath] = append(results[loc.filePath], rules.ValidationResult{
				Reference: loc.jsonPath,
				Message: fmt.Sprintf(
					"duplicate URN '%s'",
					urn,
				),
			})
		}
	}

	return results
}
