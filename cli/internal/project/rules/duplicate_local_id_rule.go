package rules

import (
	"fmt"

	"github.com/rudderlabs/rudder-iac/cli/internal/project/specs"
	"github.com/rudderlabs/rudder-iac/cli/internal/validation/rules"
)

// ParseSpecFunc extracts local IDs from a spec via the provider's ParseSpec.
type ParseSpecFunc func(path string, s *specs.Spec) (*specs.ParsedSpec, error)

type duplicateLocalIDRule struct {
	parseSpec ParseSpecFunc
}

func NewDuplicateLocalIDRule(parseSpec ParseSpecFunc) rules.Rule {
	return &duplicateLocalIDRule{parseSpec: parseSpec}
}

func (r *duplicateLocalIDRule) ID() string          { return "project/duplicate-local-id" }
func (r *duplicateLocalIDRule) Severity() rules.Severity { return rules.Error }
func (r *duplicateLocalIDRule) Description() string {
	return "local IDs must be unique within each kind"
}
func (r *duplicateLocalIDRule) AppliesTo() []string { return []string{"*"} }
func (r *duplicateLocalIDRule) Examples() rules.Examples {
	return rules.Examples{}
}

// Validate is a no-op — this rule operates as a ProjectRule via ValidateProject.
func (r *duplicateLocalIDRule) Validate(_ *rules.ValidationContext) []rules.ValidationResult {
	return nil
}

// ValidateProject checks for duplicate local IDs across all specs, grouped by kind.
// It calls ParseSpec on each spec to extract local IDs with their JSON Pointer paths.
func (r *duplicateLocalIDRule) ValidateProject(allSpecs map[string]*rules.ValidationContext) map[string][]rules.ValidationResult {
	type occurrence struct {
		filePath  string
		reference string
	}

	// kind -> id -> all occurrences
	idsByKind := make(map[string]map[string][]occurrence)

	for filePath, ctx := range allSpecs {
		parsed, err := r.parseSpec(filePath, &specs.Spec{
			Version:  ctx.Version,
			Kind:     ctx.Kind,
			Metadata: ctx.Metadata,
			Spec:     ctx.Spec,
		})
		if err != nil {
			// ParseSpec failure on a syntactically valid spec is unexpected.
			// Skip this spec — other validation rules will catch structural issues.
			continue
		}

		if idsByKind[ctx.Kind] == nil {
			idsByKind[ctx.Kind] = make(map[string][]occurrence)
		}

		for _, localID := range parsed.LocalIDs {
			idsByKind[ctx.Kind][localID.ID] = append(
				idsByKind[ctx.Kind][localID.ID],
				occurrence{filePath: filePath, reference: localID.Reference},
			)
		}
	}

	results := make(map[string][]rules.ValidationResult)

	for kind, ids := range idsByKind {
		for id, locations := range ids {
			if len(locations) <= 1 {
				continue
			}

			for _, loc := range locations {
				results[loc.filePath] = append(results[loc.filePath], rules.ValidationResult{
					Reference: loc.reference,
					Message: fmt.Sprintf("duplicate local id '%s' for kind '%s'",id, kind),
				})
			}
		}
	}

	return results
}