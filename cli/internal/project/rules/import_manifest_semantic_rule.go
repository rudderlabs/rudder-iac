package rules

import (
	"fmt"

	"github.com/rudderlabs/rudder-iac/cli/internal/project/specs"
	"github.com/rudderlabs/rudder-iac/cli/internal/validation/rules"
)

type importManifestSemanticRule struct{}

func NewImportManifestSemanticRule() rules.Rule {
	return &importManifestSemanticRule{}
}

func (r *importManifestSemanticRule) ID() string               { return "project/import-manifest-semantic" }
func (r *importManifestSemanticRule) Severity() rules.Severity { return rules.Error }
func (r *importManifestSemanticRule) Description() string {
	return "manifest URNs must reference existing resources"
}
func (r *importManifestSemanticRule) AppliesTo() []rules.MatchPattern {
	return []rules.MatchPattern{rules.MatchKind(specs.KindImportManifest)}
}
func (r *importManifestSemanticRule) Examples() rules.Examples { return rules.Examples{} }

func (r *importManifestSemanticRule) Validate(ctx *rules.ValidationContext) []rules.ValidationResult {
	if !ctx.HasGraph() {
		return nil
	}

	var results []rules.ValidationResult

	// extractManifestURNs is defined in import_manifest_project_rule.go (same package)
	for _, entry := range extractManifestURNs(ctx.Spec) {
		if _, exists := ctx.Graph.GetResource(entry.URN); !exists {
			results = append(results, rules.ValidationResult{
				RuleID:    r.ID(),
				Severity:  r.Severity(),
				Message:   fmt.Sprintf("manifest URN '%s' does not match any resource in specs", entry.URN),
				FilePath:  ctx.FilePath,
				FileName:  ctx.FileName,
				Reference: entry.Reference,
			})
		}
	}

	return results
}
